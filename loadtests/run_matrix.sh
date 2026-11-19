#!/usr/bin/env bash
# run_matrix.sh <env> <seria_a1|seria_a2|seria_b|seria_c|seria_d>
# Iteruje po macierzy serii (tryb 2 miesięcy) i POMIJA przebiegi, których plik wyniku już istnieje.
# Poziomy RPS bierze z loadtests/rates.env (plik wypełniany po wyznaczeniu nasycenia: SAT_<IFACE>_<OP>=rps).
set -uo pipefail
cd "$(dirname "$0")/.."
ENVNAME=${1:?env}; WHICH=${2:?seria}
REPS=${REPS:-3}      # A2 i B
REPS_C=${REPS_C:-2}  # seria C (kolejność cięć p. 3: 3 -> 2)
REPS_D=${REPS_D:-2}  # seria D
export PG_CPUS=${PG_CPUS:-3}  # kalibracja w rates.env dotyczy Postgresa z 3 CPU
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
  seria_a1) # A1 wykonuje loadtests/run_a1_calibration.sh (rampy 8000/4000 RPS, 3 powtórzenia, CPU/RAM Postgresa)
    echo "A1 uruchamiaj: loadtests/run_a1_calibration.sh ${ENVNAME}" >&2; exit 2 ;;
  seria_a2) # obciążenie stałe, 70% nasycenia C0
    for i in "${IFACES[@]}"; do for op in O1 O3; do
      rate=$(( $(sat "$i" "$op") * 70 / 100 ))
      for cfg in C0 C2 C3 C4 C5; do for r in $(seq 1 "$REPS"); do  # C1 pominięte: TLS+JWT razem w C2
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
      for cfg in C0 C3 C5; do for r in $(seq 1 "$REPS_C"); do
        SIZE=$size BG_RATE=$bg TAG="s${size}" run seria_c "$i" O1 "$cfg" "$r" "$bg"
      done; done
    done; done ;;
  seria_d) # złożone ładunki: warianty wg interfejsu, 3 poziomy, C0/C3/C5
    # tablica: najwyższy poziom 2000 (REST JSON max ~64 030 B < 64 KiB), żeby w C5 odrzucał limit złożoności (długość
    # tablicy), a nie limit rozmiaru z C3; przy 10 000 (317 808 B) oba dawały to samo (pilotaż 19.11, P4)
    declare -A VARS=( [graphql]="depth:3,6,10 alias:5,10,50 limit:10,25,50" [rest]="json:4,8,64 array:100,1000,2000" [grpc]="array:100,1000,2000" )
    for i in "${IFACES[@]}"; do for spec in ${VARS[$i]}; do
      variant=${spec%%:*}; levels=${spec#*:}
      for lvl in ${levels//,/ }; do
        bg=$(( $(sat "$i" O1) * 30 / 100 ))
        v=$variant; [ "$v" = json ] && v=depth
        for cfg in C0 C3 C5; do for r in $(seq 1 "$REPS_D"); do
          VARIANT=$v LEVEL=$lvl BG_RATE=$bg TAG="${variant}${lvl}" run seria_d "$i" O1 "$cfg" "$r" "$bg"
        done; done
      done
    done; done ;;
  *) echo "nieznana seria $WHICH" >&2; exit 2 ;;
esac
