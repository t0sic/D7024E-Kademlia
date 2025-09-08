#!/bin/sh
set -eu

apk add --no-cache netcat-openbsd >/dev/null

send_udp() {
  # $1 host, $2 port, $3 payload, $4 timeout seconds
  echo -n "$3" | nc -u -w "$4" "$1" "$2"
}

is_pong() {
  # accept "PONG" or "PONG <anything>"
  resp_trim=$(printf %s "$1" | tr -d '\r\n')
  case "$resp_trim" in
    PONG|PONG\ *) return 0 ;;
    *)            return 1 ;;
  esac
}

sleep 1

RESP="$(send_udp node 6882 PING 1 || true)"
if ! is_pong "$RESP"; then
  echo "Test failed: node did not respond with PONG (got '$(printf %s "$RESP")')"
  exit 1
fi

RESP2="$(send_udp bootstrap 6881 PING 1 || true)"
if ! is_pong "$RESP2"; then
  echo "Test failed: bootstrap did not respond with PONG (got '$(printf %s "$RESP2")')"
  exit 1
fi

echo "OK: both nodes responded with PONG"