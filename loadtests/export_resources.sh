#!/usr/bin/env bash
# export_resources.sh start|stop <kontener> <plik.csv>
# start: w tle próbkuje `docker stats` (czas, cpu%, pamięć) ~co 1 s do pliku CSV; stop: kończy próbkowanie.
set -euo pipefail
ACTION=$1; CTR=$2; OUT=$3
case "$ACTION" in
  start)
    echo "ts,cpu_perc,mem_used,mem_limit" > "$OUT"
    ( while true; do
        line=$(docker stats "$CTR" --no-stream --format '{{.CPUPerc}},{{.MemUsage}}' 2>/dev/null || true)
        [ -n "$line" ] && echo "$(date +%s.%N),${line// \/ /,}" >> "$OUT"
      done ) &
    echo $! > "$OUT.pid" ;;
  stop)
    if [ -f "$OUT.pid" ]; then
      kill "$(cat "$OUT.pid")" 2>/dev/null || true
      rm -f "$OUT.pid"
    fi ;;
esac
