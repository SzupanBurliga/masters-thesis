#!/usr/bin/env bash
# Weryfikacja macierzy konfiguracji C0-C5 (praktyczna.md 3.6) dla jednego lub wszystkich interfejsów.
# Użycie: infra/config_check.sh [rest|graphql|grpc]   (domyślnie wszystkie)
set -uo pipefail
cd "$(dirname "$0")/.."
IFACES=${1:-"rest graphql grpc"}
declare -A PORT=([rest]=8443 [graphql]=8444 [grpc]=8445)

# kolumny: plain_rejected no_token_401 big_payload_rejected ratelimit_429 deep_rejected
declare -A EXPECT=(
  [C0]="false false false false false"
  [C1]="true  false false false false"
  [C2]="true  true  false false false"
  [C3]="true  true  true  false false"
  [C4]="true  true  true  true  false"
  [C5]="true  true  true  true  true"
)
flags() { case $1 in
  C0) echo "false false false false false";; C1) echo "true false false false false";;
  C2) echo "true true false false false";;  C3) echo "true true true false false";;
  C4) echo "true true true true false";;    C5) echo "true true true true true";;
esac; }

[ -f loadtests/tokens.json ] || make tokens >/dev/null
[ -f infra/certs/server.crt ] || make certs >/dev/null
TOKEN=$(jq -r '.legit[0]' loadtests/tokens.json)
DC="docker compose -f infra/docker-compose.yml"
fail=0
(cd app && go build -o ../bin/configcheck ./cmd/configcheck)

for iface in $IFACES; do
  for cfg in C0 C1 C2 C3 C4 C5; do
    read -r T A V R X <<<"$(flags $cfg)"
    $DC --profile "$iface" down >/dev/null 2>&1
    TLS_ENABLED=$T AUTH_ENABLED=$A VALIDATION_ENABLED=$V RATELIMIT_ENABLED=$R COMPLEXITY_ENABLED=$X \
      $DC --profile "$iface" up -d "app-$iface" >/dev/null 2>&1
    sleep 4
    # C3 w tej macierzy obejmuje walidację; C5 złożoność. Dla C1 token nie jest wymagany, ale go podajemy.
    out=$(./bin/configcheck -iface "$iface" -addr "localhost:${PORT[$iface]}" -tls="$T" -token "$TOKEN" 2>&1)
    get() { echo "$out" | grep "^$1=" | cut -d= -f2; }
    got="$(get plain_rejected) $(get no_token_401) $(get big_payload_rejected) $(get ratelimit_429) $(get deep_rejected)"
    want=$(echo ${EXPECT[$cfg]})
    if [ "$(get baseline_ok)" != "true" ]; then st="BASELINE-FAIL"; fail=1
    elif [ "$got" == "$want" ]; then st="OK"
    else st="MISMATCH"; fail=1; fi
    printf "%-8s %s  got=[%s] want=[%s]  %s\n" "$iface" "$cfg" "$got" "$want" "$st"
  done
  $DC --profile "$iface" down >/dev/null 2>&1
done
exit $fail
