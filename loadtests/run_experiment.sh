#!/usr/bin/env bash
# run_experiment.sh <env> <seria> <iface> <op|tag> <cfg> <rep> <rate>
# Dodatkowe parametry scenariusza przekazuj zmiennymi środowiskowymi (np. ATTACK_RATE, SIZE, VARIANT, LEVEL).
# MONITORING=1 włącza podgląd na żywo (Prometheus + cAdvisor + Grafana na http://localhost:3000); domyślnie wyłączony,
# bo obciąża tę samą maszynę. Stos zostaje uruchomiony po przebiegu (zatrzymanie: make monitoring-down).
# Zmienna TAG zastępuje <op> w nazwie plików. Przykład:
#   loadtests/run_experiment.sh local seria_a rest O1 C0 1 200
set -euo pipefail
ENVNAME=$1; SERIES=$2; IFACE=$3; OP=$4; CFG=$5; REP=$6; RATE=$7
TAG=${TAG:-$OP}
WARMUP=${WARMUP:-120s}; DURATION=${DURATION:-600s}; COOLDOWN=${COOLDOWN:-120s}

ROOT=$(cd "$(dirname "$0")/.." && pwd)
cd "$ROOT"

case "$CFG" in
  C0) FL=(false false false false false) ;;
  C1) FL=(true  false false false false) ;;
  C2) FL=(true  true  false false false) ;;
  C3) FL=(true  true  true  false false) ;;
  C4) FL=(true  true  true  true  false) ;;
  C5) FL=(true  true  true  true  true)  ;;
  *) echo "nieznana konfiguracja $CFG" >&2; exit 2 ;;
esac
export TLS_ENABLED=${FL[0]} AUTH_ENABLED=${FL[1]} VALIDATION_ENABLED=${FL[2]} RATELIMIT_ENABLED=${FL[3]} COMPLEXITY_ENABLED=${FL[4]}

case "$IFACE" in rest) PORT=8443;; graphql) PORT=8444;; grpc) PORT=8445;; *) echo "zły interfejs" >&2; exit 2;; esac
CTR="infra-app-${IFACE}-1"
DC="docker compose -f infra/docker-compose.yml"
OUTDIR="results/${ENVNAME}/${SERIES}"; mkdir -p "$OUTDIR"
OUT="${OUTDIR}/${IFACE}_${TAG}_${CFG}_${REP}"

[ -f infra/certs/server.crt ] || make certs >/dev/null
[ -f loadtests/tokens.json ] || make tokens >/dev/null
mkdir -p bin && (cd app && go build -o ../bin/seed ./cmd/seed)

MONITORING=${MONITORING:-0}
if [ "$MONITORING" = "1" ]; then
  $DC --profile monitoring up -d >/dev/null
elif [ -n "$($DC --profile monitoring ps -q 2>/dev/null)" ]; then
  echo "UWAGA: stos monitoringu działa, a MONITORING=0 — pomiar będzie zakłócony (make monitoring-down)" >&2
fi

# 1-4: czysty stan, start, seed
$DC --profile "$IFACE" down >/dev/null 2>&1 || true
$DC --profile "$IFACE" up -d "app-${IFACE}" >/dev/null
DB_URL=postgres://postgres:test@localhost:5432/shop ./bin/seed > "${OUT}_seed.txt"
loadtests/wait_ready.sh "$IFACE"

# 5-8: rozgrzewka/pomiar/wystygnięcie realizuje k6 (fazy zdefiniowane w skrypcie serii)
loadtests/export_resources.sh start "$CTR" "${OUT}_resources.csv"
START=$(date -Iseconds)
# parametry scenariuszy przekazywane jawnie (nie polegamy na dziedziczeniu środowiska)
EXTRA=()
for v in MODE RAMP MAXRATE LEGIT_RATE ATTACK_RATE LEGIT_ONLY ATTACK_DUR RECOVER BG_RATE BIG_RATE SIZE VARIANT LEVEL; do
  [ -n "${!v:-}" ] && EXTRA+=(-e "$v=${!v}")
done
set +e
( cd loadtests && k6 run "${EXTRA[@]}" \
    -e IFACE="$IFACE" -e OP="$OP" -e RATE="$RATE" -e HOST="localhost:${PORT}" \
    -e TLS="$TLS_ENABLED" -e AUTH="$AUTH_ENABLED" \
    -e WARMUP="$WARMUP" -e DURATION="$DURATION" -e COOLDOWN="$COOLDOWN" \
    --summary-export "../${OUT}.json" --out "csv=../${OUT}_points.csv.gz" "${SERIES_JS:-$SERIES}.js" > "../${OUT}_k6.log" 2>&1 )
K6RC=$?
set -e
END=$(date -Iseconds)
loadtests/export_resources.sh stop "$CTR" "${OUT}_resources.csv"

# 9-10: metadane i flagi zdarzeń (OOM, restart)
OOM=$($DC --profile "$IFACE" ps -q "app-${IFACE}" | xargs -r docker inspect -f '{{.State.OOMKilled}}' 2>/dev/null || echo unknown)
RESTARTS=$($DC --profile "$IFACE" ps -q "app-${IFACE}" | xargs -r docker inspect -f '{{.RestartCount}}' 2>/dev/null || echo unknown)
cat > "${OUT}_meta.json" <<JSON
{"env":"${ENVNAME}","series":"${SERIES}","iface":"${IFACE}","op":"${OP}","tag":"${TAG}","config":"${CFG}","rep":${REP},"rate":${RATE},
 "flags":{"TLS":${TLS_ENABLED},"AUTH":${AUTH_ENABLED},"VALIDATION":${VALIDATION_ENABLED},"RATELIMIT":${RATELIMIT_ENABLED},"COMPLEXITY":${COMPLEXITY_ENABLED}},
 "params":{"WARMUP":"${WARMUP}","DURATION":"${DURATION}","COOLDOWN":"${COOLDOWN}","ATTACK_RATE":"${ATTACK_RATE:-}","LEGIT_RATE":"${LEGIT_RATE:-}","SIZE":"${SIZE:-}","VARIANT":"${VARIANT:-}","LEVEL":"${LEVEL:-}","LEGIT_ONLY":"${LEGIT_ONLY:-}","ATTACK_DUR":"${ATTACK_DUR:-}","RECOVER":"${RECOVER:-}","BG_RATE":"${BG_RATE:-}","BIG_RATE":"${BIG_RATE:-}"},
 "commit":"$(git rev-parse HEAD 2>/dev/null || echo none)","dirty":"$(git status --porcelain 2>/dev/null | wc -l)",
 "monitoring":${MONITORING},"host":"$(hostname)","start":"${START}","end":"${END}","k6_exit":${K6RC},"oom_killed":"${OOM}","restart_count":"${RESTARTS}"}
JSON

$DC --profile "$IFACE" down >/dev/null 2>&1 || true
[ "$K6RC" -eq 0 ] || { echo "k6 zakończył się kodem $K6RC (przebieg: ${OUT})" >&2; exit "$K6RC"; }
echo "OK ${OUT}"
