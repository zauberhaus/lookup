#!/bin/sh
cd "$(dirname "$0")/.."
go run github.com/elastic/go-licenser@latest -copyright -license ASL2-Short -licensor "Zauberhaus"
