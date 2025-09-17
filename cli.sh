#!/usr/bin/env bash
set -euo pipefail

# Avoid MSYS path conversion when running under Git Bash on Windows
if [ -n "${MSYSTEM-}" ] || [ -n "${MSYS-}" ]; then
  export MSYS_NO_PATHCONV=1
fi
# run-in.sh — run a command inside a docker container or compose service
# Examples:
#   ./run-in.sh -c kad-bootstrap -- /app/kad --help
#   ./run-in.sh -s node --index 2 -- /app/kad hello --name "King Mikolaj"
#   ./run-in.sh kad --service bootstrap -- --addr=:6882   # convenience: runs /app/kad <args>
#
# Notes:
# - If you pass --compose, it will prefer `docker compose`, else fall back to `docker-compose`.
# - If DOCKER requires root on your machine, this script auto-uses sudo.

usage() {
  cat <<USAGE
Usage:
  $0 -c <container-name> -- <command...>
  $0 -s <compose-service> [--index N] [--compose] -- <command...>
  $0 kad -c <container-name> -- <kad-args...>
  $0 kad -s <compose-service> [--index N] [--compose] -- <kad-args...>

Options:
  -c, --container  Exact container name (e.g. kad-bootstrap)
  -s, --service    Compose service name (e.g. bootstrap, node)
      --index N    Nth replica of a compose service (default 1)
      --compose    Force use of docker compose (instead of docker-compose)
  -h, --help       Show this help

The first form runs any command you provide inside the container.
The 'kad' form is a convenience to run '/app/kad <args>' inside the container.
USAGE
}

# --- detect docker / sudo ---
DOCKER_BIN="docker"
if ! command -v docker >/dev/null 2>&1; then
  echo "docker not found in PATH" >&2; exit 127
fi
if ! $DOCKER_BIN version >/dev/null 2>&1; then
  DOCKER_BIN="sudo docker"
fi

# --- detect compose ---
compose_cmd() {
  if $DOCKER_BIN compose version >/dev/null 2>&1; then
    $DOCKER_BIN compose "$@"
  else
    if command -v docker-compose >/dev/null 2>&1; then
      docker-compose "$@"
    else
      echo "docker compose / docker-compose not available" >&2; exit 1
    fi
  fi
}

# --- args ---
CONTAINER=""
SERVICE=""
INDEX=1
FORCE_COMPOSE=false
MODE="exec"  # or "kad"

# parse flags until --
while [[ $# -gt 0 ]]; do
  case "$1" in
    -c|--container) CONTAINER="${2:-}"; shift 2 ;;
    -s|--service)   SERVICE="${2:-}"; shift 2 ;;
    --index)        INDEX="${2:-}"; shift 2 ;;
    --compose)      FORCE_COMPOSE=true; shift ;;
    kad)            MODE="kad"; shift ;;
    -h|--help)      usage; exit 0 ;;
    --)             shift; break ;;
    *)              # if user placed subcommand without flags in kad mode, treat remainder as cmd
                    break ;;
  esac
done

# remaining are the command to run inside the container
if [[ "$MODE" == "kad" ]]; then
  # Convenience: always run /app/kad with remaining args
  if [[ $# -eq 0 ]]; then
    echo "No kad args provided. Example: $0 kad -c myct -- hello --name King" >&2
    exit 2
  fi
  IN_CMD=(/app/kad "$@")
else
  if [[ $# -eq 0 ]]; then
    echo "No command provided to run inside container. Use -- and then your command." >&2
    usage; exit 2
  fi
  IN_CMD=("$@")
fi

# --- resolve container id/name ---
resolve_container_by_service() {
  local svc="$1" idx="$2"
  # fetch container ids for service, pick Nth (default 1)
  # works for both docker compose v2 and docker-compose
  mapfile -t ids < <(compose_cmd ps -q "$svc")
  if [[ ${#ids[@]} -lt "$idx" || "$idx" -le 0 ]]; then
    echo "No container for service '$svc' at index $idx" >&2; exit 1
  fi
  echo "${ids[$((idx-1))]}"
}

if [[ -n "$SERVICE" ]]; then
  CID="$(resolve_container_by_service "$SERVICE" "$INDEX")"
elif [[ -n "$CONTAINER" ]]; then
  CID="$CONTAINER"
else
  echo "Specify either --container or --service" >&2
  usage; exit 2
fi

# --- sanity: ensure container exists/running (exec requires running) ---
if ! $DOCKER_BIN ps --format '{{.Names}}' | grep -Fxq "$CID"; then
  # If user passed a container ID from compose_cmd ps -q, names won't match.
  # In that case, try 'docker inspect' to confirm it exists.
  if ! $DOCKER_BIN inspect "$CID" >/dev/null 2>&1; then
    echo "Container '$CID' not found or not running." >&2
    exit 1
  fi
fi

# --- exec inside container ---
# Use -it when interactive TTY; otherwise non-interactive
if [[ -t 0 && -t 1 ]]; then
  $DOCKER_BIN exec -it "$CID" "${IN_CMD[@]}"
else
  $DOCKER_BIN exec "$CID" "${IN_CMD[@]}"
fi
