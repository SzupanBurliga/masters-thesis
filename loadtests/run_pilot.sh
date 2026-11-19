#!/usr/bin/env bash
# run_pilot.sh [env] — pilotaż serii A2: O1, rest/graphql/grpc x C0/C2/C4/C5, 1 powtórzenie, krótkie czasy.
# Kolejność przebiegów losowana ze stałym ziarnem (SEED) i zapisywana w results/<env>/order.txt,
# żeby konfiguracje nie trafiały systematycznie na rozgrzany host. Przed każdym przebiegiem zapis stanu hosta
# (loadavg, % bezczynności CPU, "cpu MHz" z /proc/cpuinfo — w WSL2 wartość raczej stała — oraz czas stałego
# testu obliczeniowego jako miara efektywnego taktowania) do results/<env>/host_pre.csv.
# Wznawialny: pomija przebiegi z istniejącym wynikiem. RATE = 70% SAT_* z loadtests/rates.env.
set -uo pipefail
cd "$(dirname "$0")/.."
ENVNAME=${1:-pilot}
SEED=${SEED:-20261119}
PAUSE=${PAUSE:-60}
export WARMUP=${WARMUP:-30s} DURATION=${DURATION:-120s} COOLDOWN=${COOLDOWN:-30s} MODE=const
export PG_CPUS=${PG_CPUS:-3}
source loadtests/rates.env
DIR=results/${ENVNAME}; mkdir -p "$DIR/seria_a"
ORDER=$DIR/order.txt; HOST=$DIR/host_pre.csv

if [ ! -s "$ORDER" ]; then
  python3 -I -c "
import random
runs=[(i,c) for i in ('rest','graphql','grpc') for c in ('C0','C2','C4','C5')]
random.Random($SEED).shuffle(runs)
print('# seed=$SEED; kolejność: nr iface cfg')
for n,(i,c) in enumerate(runs,1): print(n,i,c)" > "$ORDER"
fi
[ -s "$HOST" ] || echo "ts,nr,iface,cfg,load1,load5,cpu_idle_pct,cpuinfo_mhz,bench_ms_1,bench_ms_2,bench_ms_3" > "$HOST"

bench() { python3 -I -c "
import time
t=time.perf_counter(); s=0
for i in range(3_000_000): s+=i*i
print(round((time.perf_counter()-t)*1000,1))"; }

grep -v '^#' "$ORDER" | while read -r nr iface cfg; do
  f="$DIR/seria_a/${iface}_O1_${cfg}_1.json"
  if [ -s "$f" ]; then echo "pomijam $f"; continue; fi
  v="SAT_${iface^^}_O1"; rate=$(( ${!v} * 70 / 100 ))
  read -r l1 l5 _ < /proc/loadavg
  idle=$(vmstat 1 2 | tail -1 | awk '{print $15}')
  mhz=$(awk -F: '/cpu MHz/{s+=$2;n++} END{printf "%.1f", s/n}' /proc/cpuinfo)
  echo "$(date -Iseconds),$nr,$iface,$cfg,$l1,$l5,$idle,$mhz,$(bench),$(bench),$(bench)" >> "$HOST"
  echo "[$nr/12] $(date +%T) $iface $cfg rate=$rate"
  loadtests/run_experiment.sh "$ENVNAME" seria_a "$iface" O1 "$cfg" 1 "$rate" || echo "BŁĄD: $f" >> "$DIR/failed.txt"
  sleep "$PAUSE"
done
echo KONIEC
