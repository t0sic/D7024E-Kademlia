# ---- Build stage ----
FROM golang:1.25 AS build
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /node ./cmd/node

# ---- Runtime stage ----
FROM debian:stable-slim

# Install debugging/network tools
RUN apt-get update && apt-get install -y --no-install-recommends \
        netcat-openbsd \
        iputils-ping \
        dnsutils \
        curl \
        socat \
    && rm -rf /var/lib/apt/lists/*

COPY --from=build /node /node
ENTRYPOINT ["/node"]
    