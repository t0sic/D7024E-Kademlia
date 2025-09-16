#!/usr/bin/env bash
set -euo pipefail

ADDR="${ADDR:-:6882}"

echo "[kad] building..."
go mod tidy
go build -o kad ./cmd/kad

echo "[kad] starting on ${ADDR} (auto-restart enabled)"
while true; do
  ./kad run --addr "${ADDR}"
  echo "[kad] exited; restarting in 2s..."
  sleep 2
done
