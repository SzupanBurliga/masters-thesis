#!/usr/bin/env bash
# check_complete.sh <env> <seria> — liczy wyniki (_meta.json) i porównuje z oczekiwaną macierzą (tryb 2 miesięcy).
set -euo pipefail
cd "$(dirname "$0")/.."
ENVNAME=${1:?env}; SERIES=${2:?seria}
REPS=${REPS:-3}; REPS_C=${REPS_C:-2}; REPS_D=${REPS_D:-2}   # jak w run_matrix.sh
case "$SERIES" in
  seria_a) EXPECT=$((18 + 3*2*5*REPS)) ;;   # A1 (3 iface x 2 op x 3 rep, run_a1_calibration.sh) + A2 (x 5 cfg: C0,C2-C5)
  seria_b) EXPECT=$((3*3*3*REPS)) ;;        # 3 iface x 3 warianty x 3 cfg
  seria_c) EXPECT=$((3*4*3*REPS_C)) ;;      # 3 iface x 4 rozmiary x 3 cfg
  seria_d) EXPECT=$((18*3*REPS_D)) ;;       # 18 (wariant, poziom): graphql 9, rest 6, grpc 3; x 3 cfg
  *) echo "nieznana seria" >&2; exit 2 ;;
esac
HAVE=$(ls results/${ENVNAME}/${SERIES}/*_meta.json 2>/dev/null | wc -l)
BAD=$(grep -L '"k6_exit":0' results/${ENVNAME}/${SERIES}/*_meta.json 2>/dev/null | wc -l || true)
OOM=$(grep -l '"oom_killed":"true"' results/${ENVNAME}/${SERIES}/*_meta.json 2>/dev/null | wc -l || true)
echo "${ENVNAME}/${SERIES}: ${HAVE}/${EXPECT} przebiegów, błędnych k6: ${BAD}, OOM: ${OOM}"
[ "$HAVE" -eq "$EXPECT" ] && [ "$BAD" -eq 0 ]
