#!/usr/bin/env bash
# Build everything (including any examples) and run all tests with coverage.
set -euo pipefail
cd "$(dirname "$0")"

log=$(mktemp "${TMPDIR:-/tmp}/gotest.XXXXXX")

# examples are programs without tests and mocks are generated: make sure
# everything still compiles, but do not test or measure them
go build ./...
go vet ./...

# a failing test must fail the script, not only the formatter at the end of the pipe
if command -v gotestfmt >/dev/null; then
  fmt=gotestfmt
else
  fmt=cat
fi
# shellcheck disable=SC2046 # the package list must expand into arguments
go test -race -json -coverprofile=coverage.txt -covermode atomic -v \
  $(go list ./... | grep -Ev '/(examples|mock)(/|$)' || true) 2>&1 | tee "$log" | $fmt
echo "test log: $log"

# generated mock files do not count towards the coverage
grep -v 'mock\.go:' coverage.txt > coverage.tmp || true
mv coverage.tmp coverage.txt

go tool cover -func=coverage.txt
