#!/usr/bin/env bash
# run_a1_calibration.sh [env] — kalibracja A1 (domyślnie C0) z wyższymi sufitami rampy i większym limitem CPU bazy.
# CFG=C2 itd. wybiera konfigurację (np. kalibracja z TLS); IFACES / OPS / REPS zawężają macierz (np. REPS=1 OPS=O1).
# Rampy mają to samo nachylenie (~667 RPS/min): O1 do 8000 RPS/12 min, O3 do 4000 RPS/6 min; 3 powtórzenia.
# Dodatkowo: CPU/RAM Postgresa (*_pg_resources.csv); CPU całej maszyny (*_hostcpu.txt, vmstat co 5 s) zapisuje run_experiment.sh.
# Limit CPU Postgresa: PG_CPUS (domyślnie 3). Wznawialny: pomija przebiegi zakończone poprawnie (_meta.json z k6_exit 0;
# sam .json nie wystarcza, bo k6 zapisuje go także po przerwaniu).
set -uo pipefail
cd "$(dirname "$0")/.."
ENVNAME=${1:-local_db3cpu}
export PG_CPUS=${PG_CPUS:-3}
PAUSE=${PAUSE:-60}
CFG=${CFG:-C0}
IFACES=${IFACES:-"rest graphql grpc"}; OPS=${OPS:-"O1 O3"}; REPS=${REPS:-"1 2 3"}
DC="docker compose -f infra/docker-compose.yml"
OUTDIR=results/${ENVNAME}/seria_a; mkdir -p "$OUTDIR"
# config.txt opisuje kalibrację C0 (wersjonowany); inne konfiguracje dopisują się do config_<CFG>.txt
CONF=results/${ENVNAME}/config.txt; [ "$CFG" = C0 ] || CONF=results/${ENVNAME}/config_${CFG}.txt
echo "CFG=${CFG} PG_CPUS=${PG_CPUS} app_cpus=2 data=$(date -Iseconds) commit=$(git rev-parse --short HEAD) ifaces=${IFACES// /,} ops=${OPS// /,} reps=${REPS// /,}" >> "$CONF"

$DC up -d --force-recreate postgres >/dev/null
for _ in $(seq 30); do [ "$($DC ps -q postgres | xargs docker inspect -f '{{.State.Health.Status}}')" = healthy ] && break; sleep 2; done

for rep in $REPS; do for op in $OPS; do for iface in $IFACES; do
  case $op in O1) MAXRATE=8000; RAMP=12m;; O3) MAXRATE=4000; RAMP=6m;; esac
  out="${OUTDIR}/${iface}_${op}_ramp_${CFG}_${rep}"
  grep -qs '"k6_exit":0' "${out}_meta.json" && { echo "pomijam $out"; continue; }
  loadtests/export_resources.sh start infra-postgres-1 "${out}_pg_resources.csv"
  MODE=ramp RAMP=$RAMP MAXRATE=$MAXRATE WARMUP=0s DURATION=1s COOLDOWN=0s TAG="${op}_ramp" \
    loadtests/run_experiment.sh "$ENVNAME" seria_a "$iface" "$op" "$CFG" "$rep" 100 || echo "BŁĄD: $out" >> "results/${ENVNAME}/failed.txt"
  loadtests/export_resources.sh stop infra-postgres-1 "${out}_pg_resources.csv"
  sleep "$PAUSE"
done; done; done
echo KONIEC
