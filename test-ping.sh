#!/bin/sh
set -eu

# install netcat-openbsd (alpine: use apk)
apk add --no-cache netcat-openbsd >/dev/null

send_udp() {
  # $1 host, $2 port, $3 payload, $4 timeout seconds
  # echo -n to avoid newline if protocol is strict
  echo -n "$3" | nc -u -w "$4" "$1" "$2"
}

# give services a brief moment (or add a healthcheck to be fancy)
sleep 1

# 1) Ping the node inside the docker network (NOT localhost!)
RESP="$(send_udp node 6882 PING 1 || true)"
if [ "$RESP" != "PONG" ]; then
  echo "Test failed: node did not respond with PONG (got '$(printf %s "$RESP")')"
  exit 1
fi

# 2) (optional) also ping bootstrap
RESP2="$(send_udp bootstrap 6881 PING 1 || true)"
if [ "$RESP2" != "PONG" ]; then
  echo "Test failed: bootstrap did not respond with PONG (got '$(printf %s "$RESP2")')"
  exit 1
fi

echo "OK: both nodes responded with PONG"
