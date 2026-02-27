#!/usr/bin/env bash
set -euo pipefail

export GOCACHE="${GOCACHE:-/tmp/go-cache}"

printf '\n[1/3] gofmt check...\n'
UNFORMATTED="$(gofmt -l .)"
if [[ -n "$UNFORMATTED" ]]; then
  echo "Please run gofmt on:" >&2
  echo "$UNFORMATTED" >&2
  exit 1
fi

printf '\n[2/3] go vet...\n'
go vet ./...

printf '\n[3/3] go test...\n'
go test ./...

echo "\nAll checks passed."
