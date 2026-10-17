#!/usr/bin/env bash
# check_complete.sh <env> <seria> — liczy wyniki (_meta.json) i porównuje z oczekiwaną macierzą (tryb 2 miesięcy).
set -euo pipefail
cd "$(dirname "$0")/.."
ENVNAME=${1:?env}; SERIES=${2:?seria}
case "$SERIES" in
  seria_a) EXPECT=$((12 + 108)) ;;  # A1 (12) + A2 (3 iface x 2 op x 6 cfg x 3 rep)
  seria_b) EXPECT=81 ;;
  seria_c) EXPECT=72 ;;
  seria_d) EXPECT=54 ;;
  *) echo "nieznana seria" >&2; exit 2 ;;
esac
HAVE=$(ls results/${ENVNAME}/${SERIES}/*_meta.json 2>/dev/null | wc -l)
BAD=$(grep -L '"k6_exit":0' results/${ENVNAME}/${SERIES}/*_meta.json 2>/dev/null | wc -l || true)
OOM=$(grep -l '"oom_killed":"true"' results/${ENVNAME}/${SERIES}/*_meta.json 2>/dev/null | wc -l || true)
echo "${ENVNAME}/${SERIES}: ${HAVE}/${EXPECT} przebiegów, błędnych k6: ${BAD}, OOM: ${OOM}"
[ "$HAVE" -eq "$EXPECT" ] && [ "$BAD" -eq 0 ]
