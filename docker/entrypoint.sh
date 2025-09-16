#!/usr/bin/env sh
set -eu

# If the first arg is a flag (e.g., --help or --addr=...), inject default subcommand "run".
if [ $# -gt 0 ] && [ "${1#-}" != "$1" ]; then
  exec /app/kad run "$@"
fi

# Otherwise, first arg is the subcommand (or empty -> default to run)
SUBCMD="${1:-run}"
shift || true

# Build argv: /app/kad <subcmd> [env-derived flags...]
set -- /app/kad "$SUBCMD"

[ -n "${KAD_ADDR:-}" ]      && set -- "$@" --addr      "$KAD_ADDR"
[ -n "${KAD_ID:-}" ]        && set -- "$@" --id        "$KAD_ID"
[ -n "${KAD_ID_SEED:-}" ]   && set -- "$@" --id-seed   "$KAD_ID_SEED"
case "${KAD_BOOTSTRAP:-}" in
  1|true|TRUE|yes|YES) set -- "$@" --bootstrap ;;
esac
[ -n "${KAD_PEERS:-}" ]     && set -- "$@" --peers     "$KAD_PEERS"

exec "$@"
