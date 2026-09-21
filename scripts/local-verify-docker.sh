#!/usr/bin/env bash
# Run local verification with Dockerized toolchains (Option B).
# Usage:
#   ./scripts/local-verify-docker.sh
#   FRONTEND_ONLY=1 ./scripts/local-verify-docker.sh
#   BACKEND_ONLY=1 ./scripts/local-verify-docker.sh

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

DOCKER_BIN="${DOCKER_BIN:-}"

if [[ -z "$DOCKER_BIN" ]]; then
  if command -v docker >/dev/null 2>&1 && docker info >/dev/null 2>&1; then
    DOCKER_BIN="docker"
  elif command -v docker.exe >/dev/null 2>&1 && docker.exe info >/dev/null 2>&1; then
    DOCKER_BIN="docker.exe"
  fi
fi

if [[ -z "$DOCKER_BIN" ]]; then
  echo "Error: Docker is required for Option B local verification (docker or docker.exe)." >&2
  exit 1
fi

run_backend() {
  echo "== Backend verification (Go 1.27) =="
  "$DOCKER_BIN" run --rm \
    -v "${ROOT}/backend:/src" \
    -w /src \
    golang:1.27 \
    sh -lc 'CGO_ENABLED=1 /usr/local/go/bin/go mod download && CGO_ENABLED=1 /usr/local/go/bin/go vet ./... && CGO_ENABLED=1 /usr/local/go/bin/go test ./... -count=1 -race -timeout=2m && CGO_ENABLED=1 /usr/local/go/bin/go build -o /tmp/gonsgarage-api ./cmd/api'
}

run_frontend() {
  echo "== Frontend verification (Node 24 + pnpm 12) =="
  "$DOCKER_BIN" run --rm \
    -v "${ROOT}/frontend:/work" \
    -w /work \
    node:24-alpine \
    sh -lc 'corepack enable && corepack prepare pnpm@12.5.1 --activate && (pnpm install --frozen-lockfile || (pnpm approve-builds --all && pnpm install --frozen-lockfile)) && pnpm lint && pnpm typecheck && pnpm test && pnpm build'
}

if [[ "${FRONTEND_ONLY:-0}" == "1" && "${BACKEND_ONLY:-0}" == "1" ]]; then
  echo "Error: FRONTEND_ONLY and BACKEND_ONLY are mutually exclusive." >&2
  exit 1
fi

if [[ "${FRONTEND_ONLY:-0}" == "1" ]]; then
  run_frontend
  exit 0
fi

if [[ "${BACKEND_ONLY:-0}" == "1" ]]; then
  run_backend
  exit 0
fi

run_backend
run_frontend

echo "== Docker local verification complete =="
