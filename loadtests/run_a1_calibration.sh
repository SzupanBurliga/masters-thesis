#!/usr/bin/env bash
# run_a1_calibration.sh [env] — powtórka kalibracji A1 (C0) z wyższymi sufitami rampy i większym limitem CPU bazy.
# Rampy mają to samo nachylenie (~667 RPS/min): O1 do 8000 RPS/12 min, O3 do 4000 RPS/6 min; 3 powtórzenia.
# Dodatkowo: CPU/RAM Postgresa (*_pg_resources.csv) i CPU całej maszyny (*_hostcpu.txt, vmstat co 5 s).
# Limit CPU Postgresa: PG_CPUS (domyślnie 3). Wznawialny: pomija przebiegi z istniejącym wynikiem.
set -uo pipefail
cd "$(dirname "$0")/.."
ENVNAME=${1:-local_db3cpu}
export PG_CPUS=${PG_CPUS:-3}
PAUSE=${PAUSE:-60}
DC="docker compose -f infra/docker-compose.yml"
OUTDIR=results/${ENVNAME}/seria_a; mkdir -p "$OUTDIR"
echo "PG_CPUS=${PG_CPUS} app_cpus=2 data=$(date -Iseconds) commit=$(git rev-parse --short HEAD)" > "results/${ENVNAME}/config.txt"

$DC up -d --force-recreate postgres >/dev/null
for _ in $(seq 30); do [ "$($DC ps -q postgres | xargs docker inspect -f '{{.State.Health.Status}}')" = healthy ] && break; sleep 2; done

for rep in 1 2 3; do for op in O1 O3; do for iface in rest graphql grpc; do
  case $op in O1) MAXRATE=8000; RAMP=12m;; O3) MAXRATE=4000; RAMP=6m;; esac
  out="${OUTDIR}/${iface}_${op}_ramp_C0_${rep}"
  [ -s "$out.json" ] && { echo "pomijam $out"; continue; }
  vmstat -t 5 > "${out}_hostcpu.txt" & VM=$!
  loadtests/export_resources.sh start infra-postgres-1 "${out}_pg_resources.csv"
  MODE=ramp RAMP=$RAMP MAXRATE=$MAXRATE WARMUP=0s DURATION=1s COOLDOWN=0s TAG="${op}_ramp" \
    loadtests/run_experiment.sh "$ENVNAME" seria_a "$iface" "$op" C0 "$rep" 100 || echo "BŁĄD: $out" >> "results/${ENVNAME}/failed.txt"
  kill $VM 2>/dev/null
  loadtests/export_resources.sh stop infra-postgres-1 "${out}_pg_resources.csv"
  sleep "$PAUSE"
done; done; done
echo KONIEC
