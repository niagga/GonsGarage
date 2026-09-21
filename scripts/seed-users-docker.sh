#!/usr/bin/env bash
# Ejecuta los seeds de usuarios (MVP + cliente demo) usando el mismo acceso a red
# que el contenedor del API (útil cuando DATABASE_URL usa hostnames Docker, p. ej. arnela-postgres).
#
# Requisitos: Docker; contenedor API en marcha; clon del repo con backend/ en disco.
# Variables: lee .env.prod por defecto (mismo que update-server-gonsgarage.sh).
#
# Uso (en el servidor, raíz del repo):
#   ./scripts/seed-users-docker.sh
#   ENV_FILE=/ruta/.env.prod API_CONTAINER=gonsgarage-api ./scripts/seed-users-docker.sh
#
# Opcional: exportá SEED_* antes de llamar al script para no usar las contraseñas por defecto del código.
#
# Nota: en docs/mvp-solo-checklist.md los seeds están pensados para desarrollo; si corrés esto
# contra una base “real”, hacelo a sabiendas y con credenciales fuertes vía SEED_*.

set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

: "${ENV_FILE:=.env.prod}"
: "${API_CONTAINER:=gonsgarage-api}"

if [[ ! -f "$ENV_FILE" ]]; then
  echo "Error: no existe $ENV_FILE (definí ENV_FILE=… o copiá desde .env.prod.example)." >&2
  exit 1
fi

set -a
# shellcheck disable=SC1090
source "$ENV_FILE"
set +a

if [[ -z "${DATABASE_URL:-}" ]]; then
  echo "Error: DATABASE_URL vacío tras cargar $ENV_FILE." >&2
  exit 1
fi

if ! docker info >/dev/null 2>&1; then
  echo "Error: Docker no disponible o sin permisos." >&2
  exit 1
fi

if ! docker ps --format '{{.Names}}' | grep -qx "$API_CONTAINER"; then
  echo "Error: el contenedor '$API_CONTAINER' no está en ejecución (levantá el stack antes)." >&2
  exit 1
fi

echo "==> Seeds contra DATABASE_URL (host visto desde $API_CONTAINER)"
docker run --rm \
  --network "container:${API_CONTAINER}" \
  -e DATABASE_URL \
  -e SEED_ADMIN_EMAIL -e SEED_ADMIN_PASSWORD \
  -e SEED_MANAGER_EMAIL -e SEED_MANAGER_PASSWORD \
  -e SEED_EMPLOYEE_EMAIL -e SEED_EMPLOYEE_PASSWORD \
  -e SEED_CLIENT_EMAIL -e SEED_CLIENT_PASSWORD \
  -v "${ROOT}/backend:/src" \
  -w /src \
  golang:1.27-alpine \
  sh -c 'apk add --no-cache git >/dev/null && /usr/local/go/bin/go mod download && /usr/local/go/bin/go run ./cmd/seed-mvp-users && /usr/local/go/bin/go run ./cmd/seed-test-client'

echo "==> Seeds terminados."
