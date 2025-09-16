FROM golang:1.25 AS build
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/kad ./cmd/kad

FROM debian:stable-slim
WORKDIR /app
RUN apt-get update && apt-get install -y --no-install-recommends \
          netcat-openbsd iputils-ping dnsutils curl socat \
        && rm -rf /var/lib/apt/lists/*

COPY --from=build /out/kad /app/kad
COPY docker/entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh /app/kad

EXPOSE 6882/udp
ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
CMD ["run"]
