#!/usr/bin/env bash
set -euo pipefail

# Usage: ./run.sh [N]
# Starts N node replicas (default 5), builds image, and tails node logs.

N="${1:-5}"

# Basic validation: N must be a positive integer
if ! [[ "$N" =~ ^[1-9][0-9]*$ ]]; then
  echo "Usage: $0 [positive-integer]"
  exit 1
fi

# Support both "docker compose" and legacy "docker-compose"
compose() {
  if command -v docker &>/dev/null && docker compose version &>/dev/null; then
    docker compose "$@"
  else
    docker-compose "$@"
  fi
}

echo "Building image..."
compose build

echo "Starting bootstrap + $N node(s)..."
compose up -d --scale node="$N"

echo "Following logs for 'node' (Ctrl+C to stop viewing, containers stay up)..."
compose logs -f node
