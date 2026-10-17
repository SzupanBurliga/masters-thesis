#!/usr/bin/env bash
# wait_ready.sh <iface> [timeout_s]  — czeka aż port interfejsu przyjmie połączenia TCP.
set -euo pipefail
IFACE=$1; TO=${2:-60}
case "$IFACE" in rest) P=8443;; graphql) P=8444;; grpc) P=8445;; *) echo "zły interfejs"; exit 2;; esac
for ((i = 0; i < TO * 2; i++)); do
  if (exec 3<>/dev/tcp/127.0.0.1/$P) 2>/dev/null; then sleep 1; exit 0; fi
  sleep 0.5
done
echo "timeout: $IFACE nie wstał" >&2; exit 1
