#!/bin/bash
set -euo pipefail
cd "$(dirname "$0")/.."

unformatted=$(gofmt -l .)
if [[ -n $unformatted ]]; then
  printf 'Format these Go files first:\n%s\n' "$unformatted" >&2
  exit 1
fi
bash -n install.sh
bash -n book_review.sh
bash -n scripts/check.sh
bash -n scripts/package.sh
go vet ./...
go test -race -timeout 90s ./...
mkdir -p bin
go build -o bin/mino ./cmd/mino
