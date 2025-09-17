#!/usr/bin/env bash
set -euo pipefail

# Avoid MSYS path conversion under Git Bash on Windows
if [ -n "${MSYSTEM-}" ] || [ -n "${MSYS-}" ]; then
  export MSYS_NO_PATHCONV=1
fi

usage() {
  cat <<'USAGE'
Usage:
  cli.sh -c <container> -- <command...>
  cli.sh -s <service> [--index N] -- <command...>
  cli.sh kad -c <container> -- <kad-args...>
  cli.sh kad -s <service> [--index N] -- <kad-args...>

Convenience REPL modes:
  cli.sh repl [ -s <service> ] [ -- <kad-run-args-or-flags...> ]
    -> Starts a one-off interactive container (docker compose run) and runs
       your entrypoint (which invokes /app/kad). DO NOT prefix /app/kad here.
       Defaults to flags only: --addr=:0 --peers=bootstrap:6882
       (entrypoint will inject the 'run' subcommand automatically)

  cli.sh exec-repl ( -c <container> | -s <service> [--index N] ) [ -- <kad-run-args...> ]
    -> Runs a REPL *inside an existing running container* using docker exec -it.
       Here we call /app/kad directly. Default: run --addr=:0 --peers=bootstrap:6882

Notes:
- Use --addr=:0 for the REPL to pick a free UDP port (avoids conflict with the daemon on :6882).
- Headless cluster from ./run.sh stays up; REPL is separate/ephemeral.
USAGE
}

# ---- docker helpers ----
DOCKER_BIN="docker"
if ! command -v docker >/dev/null 2>&1; then
  echo "docker not found in PATH" >&2; exit 127
fi
if ! $DOCKER_BIN version >/dev/null 2>&1; then
  DOCKER_BIN="sudo docker"
fi

compose_cmd() {
  if $DOCKER_BIN compose version >/dev/null 2>&1; then
    $DOCKER_BIN compose "$@"
  elif command -v docker-compose >/dev/null 2>&1; then
    docker-compose "$@"
  else
    echo "docker compose / docker-compose not available" >&2; exit 1
  fi
}

resolve_container_by_service() {
  local svc="$1" idx="$2"
  mapfile -t ids < <(compose_cmd ps -q "$svc")
  if [[ ${#ids[@]} -lt "$idx" || "$idx" -le 0 ]]; then
    echo "No container for service '$svc' at index $idx" >&2; exit 1
  fi
  echo "${ids[$((idx-1))]}"
}

# ---- arg parsing ----
CONTAINER=""
SERVICE=""
INDEX=1
MODE="exec"   # default passthrough; others: "kad", "repl", "exec-repl"

while [[ $# -gt 0 ]]; do
  case "$1" in
    -c|--container) CONTAINER="${2:-}"; shift 2 ;;
    -s|--service)   SERVICE="${2:-}"; shift 2 ;;
    --index)        INDEX="${2:-}"; shift 2 ;;
    kad)            MODE="kad"; shift ;;
    repl)           MODE="repl"; shift ;;
    exec-repl)      MODE="exec-repl"; shift ;;
    -h|--help)      usage; exit 0 ;;
    --)             shift; break ;;
    *)              break ;;
  esac
done
REMAINING_ARGS=("$@")

# ---- repl: one-off interactive container ----
if [[ "$MODE" == "repl" ]]; then
  svc="${SERVICE:-node}"
  # Default to FLAGS ONLY so entrypoint injects 'run'
  if [[ ${#REMAINING_ARGS[@]} -eq 0 ]]; then
    REMAINING_ARGS=(--addr=:0 --peers=bootstrap:6882)
  fi
  # IMPORTANT: do NOT pass /app/kad here; entrypoint will run it.
  exec $DOCKER_BIN compose run --rm --service-ports \
       "$svc" -- "${REMAINING_ARGS[@]}"
fi

# ---- exec-repl: REPL inside existing container ----
if [[ "$MODE" == "exec-repl" ]]; then
  if [[ -z "$CONTAINER" && -z "$SERVICE" ]]; then
    echo "exec-repl requires --container or --service [--index N]" >&2; exit 2
  fi
  if [[ -n "$SERVICE" ]]; then
    CONTAINER="$(resolve_container_by_service "$SERVICE" "$INDEX")"
  fi
  if ! $DOCKER_BIN ps --format '{{.Names}}' | grep -Fxq "$CONTAINER"; then
    if ! $DOCKER_BIN inspect "$CONTAINER" >/dev/null 2>&1; then
      echo "Container '$CONTAINER' not found or not running." >&2; exit 1
    fi
  fi
  # Here we call /app/kad directly (entrypoint not used on docker exec)
  if [[ ${#REMAINING_ARGS[@]} -eq 0 ]]; then
    REMAINING_ARGS=(run --addr=:0 --peers=bootstrap:6882)
  fi
  if [[ -t 0 && -t 1 ]]; then
    exec $DOCKER_BIN exec -it "$CONTAINER" /app/kad "${REMAINING_ARGS[@]}"
  else
    exec $DOCKER_BIN exec "$CONTAINER" /app/kad "${REMAINING_ARGS[@]}"
  fi
fi

# ---- kad passthrough (/app/kad inside container) ----
if [[ "$MODE" == "kad" ]]; then
  if [[ -z "$CONTAINER" && -z "$SERVICE" ]]; then
    echo "kad mode requires --container or --service [--index N]" >&2; exit 2
  fi
  if [[ -n "$SERVICE" ]]; then
    CONTAINER="$(resolve_container_by_service "$SERVICE" "$INDEX")"
  fi
  if [[ ${#REMAINING_ARGS[@]} -eq 0 ]]; then
    echo "No kad args provided. Example: cli.sh kad -s node -- --help" >&2
    exit 2
  fi
  if [[ -t 0 && -t 1 ]]; then
    exec $DOCKER_BIN exec -it "$CONTAINER" /app/kad "${REMAINING_ARGS[@]}"
  else
    exec $DOCKER_BIN exec "$CONTAINER" /app/kad "${REMAINING_ARGS[@]}"
  fi
fi

# ---- default: arbitrary exec inside container ----
if [[ -z "$CONTAINER" && -z "$SERVICE" ]]; then
  echo "Specify either --container or --service" >&2
  usage; exit 2
fi
if [[ -n "$SERVICE" ]]; then
  CONTAINER="$(resolve_container_by_service "$SERVICE" "$INDEX")"
fi
if [[ ${#REMAINING_ARGS[@]} -eq 0 ]]; then
  echo "No command provided. Use -- and then your command." >&2
  usage; exit 2
fi

if [[ -t 0 && -t 1 ]]; then
  exec $DOCKER_BIN exec -it "$CONTAINER" "${REMAINING_ARGS[@]}"
else
  exec $DOCKER_BIN exec "$CONTAINER" "${REMAINING_ARGS[@]}"
fi
