#!/bin/sh
set -eu

NODE_SERVICE="${NODE_SERVICE:-node}"
NODE_PORT="${NODE_PORT:-6882}"
BOOTSTRAP_HOST="${BOOTSTRAP_HOST:-bootstrap}"
BOOTSTRAP_PORT="${BOOTSTRAP_PORT:-6881}"
MAX_INDEX="${MAX_INDEX:-200}"
TIMEOUT="${TIMEOUT:-1}"

apk add --no-cache netcat-openbsd bind-tools >/dev/null 2>&1 || true

send_udp() { echo -n "$3" | nc -u -w "$4" "$1" "$2" || true; }

is_pong() {
  resp_trim=$(printf %s "$1" | tr -d '\r\n')
  case "$resp_trim" in PONG|PONG\ *) return 0 ;; *) return 1 ;; esac
}

resolve_all_ipv4() {
  _host="$1"
  if command -v getent >/dev/null 2>&1; then
    getent ahostsv4 "$_host" 2>/dev/null | awk '{print $1}' | sort -u; return 0
  fi
  if command -v drill >/dev/null 2>&1; then
    drill A "$_host" 2>/dev/null | awk '/ A /{print $5}' | sort -u; return 0
  fi
  if command -v nslookup >/dev/null 2>&1; then
    nslookup "$_host" 2>/dev/null | awk '/Address: /{print $2}' | tail -n +2 | sort -u; return 0
  fi
  return 1
}

discover_node_ips() {
  ips=$(resolve_all_ipv4 "$NODE_SERVICE" || true)
  if [ -n "${ips:-}" ]; then printf '%s\n' "$ips" | sort -u; return 0; fi
  found=0
  for i in $(seq 1 "$MAX_INDEX"); do
    ip=$(resolve_all_ipv4 "${NODE_SERVICE}-${i}" || true)
    [ -n "${ip:-}" ] && { echo "$ip"; found=1; }
  done
  [ "$found" -eq 1 ]
}

# Give the cluster a moment; add healthchecks later if needed
sleep 1

NODE_IPS="$(discover_node_ips || true)"
if [ -z "${NODE_IPS:-}" ]; then
  echo "ERROR: no nodes discovered (${NODE_SERVICE} or ${NODE_SERVICE}-1..${MAX_INDEX})."; exit 2
fi

echo "Discovered nodes on ${NODE_SERVICE}:${NODE_PORT}:"
echo "$NODE_IPS" | sed 's/^/  - /'

# Optional: check bootstrap
RESP_BOOT="$(send_udp "$BOOTSTRAP_HOST" "$BOOTSTRAP_PORT" PING "$TIMEOUT")"
if ! is_pong "$RESP_BOOT"; then
  echo "Bootstrap FAIL: ${BOOTSTRAP_HOST}:${BOOTSTRAP_PORT} → '$(printf %s "$RESP_BOOT")'"; exit 1
fi
echo "Bootstrap OK"

fails=0; passes=0
echo; echo "Pinging nodes:"
for ip in $NODE_IPS; do
  resp="$(send_udp "$ip" "$NODE_PORT" PING "$TIMEOUT")"
  if is_pong "$resp"; then
    echo "  PASS  $ip:$NODE_PORT  ($resp)"; passes=$((passes+1))
  else
    echo "  FAIL  $ip:$NODE_PORT  (got '$resp')"; fails=$((fails+1))
  fi
done

echo; echo "Summary: $passes passed, $fails failed"
[ "$fails" -eq 0 ] || exit 1
echo "OK: all nodes responded with PONG"
