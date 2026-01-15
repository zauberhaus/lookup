#!/bin/sh
go test -race -json -coverprofile=coverage.txt -v $(go list ./... | grep -v /examples/) 2>&1 | tee /tmp/gotest.log | gotestfmt
