#!/usr/bin/env bash
# Build everything (including the examples) and run all tests with coverage.
set -euo pipefail
cd "$(dirname "$0")"

log=$(mktemp "${TMPDIR:-/tmp}/gotest.XXXXXX")

# the examples are programs without tests, make sure they still compile
go build ./...
go vet ./...

# a failing test must fail the script, not only the formatter at the end of the pipe
if command -v gotestfmt >/dev/null; then
  fmt=gotestfmt
else
  fmt=cat
fi
go test -race -json -coverprofile=coverage.txt -v $(go list ./... | grep -v /examples/) 2>&1 | tee "$log" | $fmt
echo "test log: $log"

go tool cover -func=coverage.txt
