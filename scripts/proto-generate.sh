#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

echo "==> Building proto tools image..."
docker build -f "$ROOT/Dockerfile.dev" -t fleet-reservation-dev "$ROOT"

echo "==> Generating protobuf..."
docker run --rm \
  -v "$ROOT":/reservation \
  -v "$HOME/.cache/buf":/root/.cache/buf \
  -w /reservation/protobuf \
  fleet-reservation-dev \
  sh -c 'buf generate'

echo "==> Done. Generated files: gen/reservation/api/v1/"
