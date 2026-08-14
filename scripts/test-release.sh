#!/usr/bin/env bash
# Test script for release workflow: vet, test, and dry-build both architectures.
# Exits non-zero on any failure so CI/local hooks can fail fast.

set -euo pipefail

cd "$(dirname "$0")/.."

: "${VERSION:=test}"

export CGO_ENABLED=0

echo "=== go vet ./... ==="
go vet ./...

echo "=== go test ./... ==="
go test ./...

echo "=== build amd64 ==="
GOARCH=amd64 go build \
  -o "dist/superdaemon_linux_amd64" \
  -v -trimpath \
  -ldflags="-s -w -X superdaemon/system.Version=${VERSION}" \
  superdaemon

echo "=== build arm64 ==="
GOARCH=arm64 go build \
  -o "dist/superdaemon_linux_arm64" \
  -v -trimpath \
  -ldflags="-s -w -X superdaemon/system.Version=${VERSION}" \
  superdaemon

echo "=== checksums ==="
cd dist
sha256sum superdaemon_linux_amd64 superdaemon_linux_arm64 > checksums.txt
cat checksums.txt

echo "=== release dry-build OK ==="