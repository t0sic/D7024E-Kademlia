#!/usr/bin/env bash
set -euo pipefail

# Run tests in the tests folder, measuring coverage across internal code
go test ./tests -coverpkg=./internal/... -cover

# Optional: generate coverage report

# go test ./tests -coverpkg=./D7024E-Kademlia/... -coverprofile=coverage.out

# Print summary
# go tool cover -func=coverage.out | grep total:

# Optional: open HTML report
# go tool cover -html=coverage.out -o coverage.html
