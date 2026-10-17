#!/usr/bin/env bash
# run_matrix.sh <env> <seria_a1|seria_a2|seria_b|seria_c|seria_d>
# Iteruje po macierzy serii (tryb 2 miesięcy) i POMIJA przebiegi, których plik wyniku już istnieje.
# Poziomy RPS bierze z loadtests/rates.env (plik wypełniany po wyznaczeniu nasycenia: SAT_<IFACE>_<OP>=rps).
set -uo pipefail
cd "$(dirname "$0")/.."
ENVNAME=${1:?env}; WHICH=${2:?seria}
REPS=${REPS:-3}
[ -f loadtests/rates.env ] && source loadtests/rates.env
IFACES=(rest graphql grpc)
sat() { local v="SAT_${1^^}_${2}"; echo "${!v:-200}"; }   # domyślnie 200 RPS do czasu kalibracji
run() { # series iface op cfg rep rate  (zmienne TAG itd. w środowisku)
  local series=$1 iface=$2 op=$3 cfg=$4 rep=$5 rate=$6 tag=${TAG:-$3}
  local f="results/${ENVNAME}/${series}/${iface}_${tag}_${cfg}_${rep}.json"
  if [ -s "$f" ]; then echo "pomijam $f"; return; fi
  loadtests/run_experiment.sh "$ENVNAME" "$series" "$iface" "$op" "$cfg" "$rep" "$rate" || echo "BŁĄD: $f" >> results/${ENVNAME}/failed.txt
  echo "pauza ${PAUSE:-600}s (stabilizacja temperatury)"; sleep "${PAUSE:-600}"
}

case "$WHICH" in
  seria_a1) # nasycenie, tylko C0, 2 powtórzenia
    for i in "${IFACES[@]}"; do for op in O1 O3; do for r in 1 2; do
      MODE=ramp RAMP=${RAMP:-3m} WARMUP=0s DURATION=1s COOLDOWN=0s TAG="${op}_ramp" run seria_a "$i" "$op" C0 "$r" 100
    done; done; done ;;
  seria_a2) # obciążenie stałe, 70% nasycenia C0
    for i in "${IFACES[@]}"; do for op in O1 O3; do
      rate=$(( $(sat "$i" "$op") * 70 / 100 ))
      for cfg in C0 C1 C2 C3 C4 C5; do for r in $(seq 1 "$REPS"); do
        MODE=const run seria_a "$i" "$op" "$cfg" "$r" "$rate"
      done; done
    done; done ;;
  seria_b) # atak wolumetryczny: O3 5x, O3 10x, O1 10x; C0, C4, C5
    for i in "${IFACES[@]}"; do
      for v in "O3 5" "O3 10" "O1 10"; do read -r op mult <<<"$v"
        s=$(sat "$i" "$op"); legit=$(( s * 30 / 100 )); atk=$(( s * mult ))
        for cfg in C0 C4 C5; do for r in $(seq 1 "$REPS"); do
          LEGIT_RATE=$legit ATTACK_RATE=$atk TAG="${op}_x${mult}" run seria_b "$i" "$op" "$cfg" "$r" "$legit"
        done; done
      done
    done ;;
  seria_c) # nadmiarowe ładunki: 4 rozmiary, C0/C3/C5
    for i in "${IFACES[@]}"; do for size in 10240 102400 1048576 5242880; do
      bg=$(( $(sat "$i" O1) * 30 / 100 ))
      for cfg in C0 C3 C5; do for r in $(seq 1 "$REPS"); do
        SIZE=$size BG_RATE=$bg TAG="s${size}" run seria_c "$i" O1 "$cfg" "$r" "$bg"
      done; done
    done; done ;;
  seria_d) # złożone ładunki: warianty wg interfejsu, 3 poziomy, C0/C3/C5
    declare -A VARS=( [graphql]="depth:3,6,10 alias:5,10,50 limit:10,25,50" [rest]="json:4,8,64 array:100,1000,10000" [grpc]="array:100,1000,10000" )
    for i in "${IFACES[@]}"; do for spec in ${VARS[$i]}; do
      variant=${spec%%:*}; levels=${spec#*:}
      for lvl in ${levels//,/ }; do
        bg=$(( $(sat "$i" O1) * 30 / 100 ))
        v=$variant; [ "$v" = json ] && v=depth
        for cfg in C0 C3 C5; do for r in $(seq 1 "$REPS"); do
          VARIANT=$v LEVEL=$lvl BG_RATE=$bg TAG="${variant}${lvl}" run seria_d "$i" O1 "$cfg" "$r" "$bg"
        done; done
      done
    done; done ;;
  *) echo "nieznana seria $WHICH" >&2; exit 2 ;;
esac
