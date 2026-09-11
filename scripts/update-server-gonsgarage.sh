#!/usr/bin/env bash
#
# GonsGarage — actualiza el código en ESTE clone y reconstruye contenedores de producción.
# (Nombre explícito para servidores con varias apps en /DATA/AppData/…)
#
# IMPORTANTE: en producción se usa SOLO docker-compose.prod.yml. NO mezclar con
# docker-compose.yml: en el repo base, go-api tiene profile "app" y al fusionar
# Compose deja frontend dependiendo de un servicio go-api "inexistente".
#
# Uso típico vía SSH:
#   ssh -i "$HOME/.ssh/tu_clave" user@host 'bash -s' < scripts/update-server-gonsgarage.sh
#
# Requisitos: git, Docker Compose v2, .env.prod en GONSGARAGE_DIR.
#
# Postgres en contenedor Arnela (`arnela-postgres` en DATABASE_URL):
#   export COMPOSE_OVERRIDE=docker-compose.prod.arnela-network.yml
#   …y ejecutá este script (el export aplica al proceso actual).
# (y editar `name:` en ese YAML si la red externa no es `arnela_arnela-network`).

set -euo pipefail

# --- configuración (override con export VAR=... antes de ejecutar) ---
: "${GONSGARAGE_DIR:=/DATA/AppData/gonsgarage}"
: "${GIT_REF:=main}"
: "${COMPOSE_FILE:=docker-compose.prod.yml}"
# Segundo `-f` opcional (red Arnela u otros overrides).
: "${COMPOSE_OVERRIDE:=}"
: "${ENV_FILE:=.env.prod}"

cd "$GONSGARAGE_DIR"

compose_args=( -f "$COMPOSE_FILE" )
if [[ -n "$COMPOSE_OVERRIDE" ]]; then
  if [[ ! -f "$COMPOSE_OVERRIDE" ]]; then
    echo "Error: COMPOSE_OVERRIDE=$COMPOSE_OVERRIDE no existe en $(pwd)." >&2
    exit 1
  fi
  compose_args+=( -f "$COMPOSE_OVERRIDE" )
fi

echo "==> Directorio: $(pwd)"
echo "==> Rama/ref: $GIT_REF"
echo "==> Compose: ${compose_args[*]}"

if [[ ! -f "$COMPOSE_FILE" ]]; then
  echo "Error: no se encuentra $COMPOSE_FILE en $GONSGARAGE_DIR" >&2
  exit 1
fi

if [[ ! -f "$ENV_FILE" ]]; then
  echo "Error: falta $ENV_FILE (variables de producción)." >&2
  exit 1
fi

if grep -qE '@arnela-postgres[:/]|//arnela-postgres' "$ENV_FILE" 2>/dev/null && [[ -z "${COMPOSE_OVERRIDE:-}" ]]; then
  arnela_override="docker-compose.prod.arnela-network.yml"
  if [[ -f "$arnela_override" ]]; then
    echo "Error: $ENV_FILE referencia arnela-postgres pero COMPOSE_OVERRIDE está vacío." >&2
    echo "      Sin -f $arnela_override el API no resuelve ese DNS (red distinta) y suele reiniciar en bucle (502)." >&2
    echo "      Ej.: export COMPOSE_OVERRIDE=$arnela_override" >&2
    echo "      Luego volvé a ejecutar este script." >&2
    exit 1
  fi
  echo "WARN: $ENV_FILE usa host arnela-postgres pero COMPOSE_OVERRIDE está vacío." >&2
  echo "      Sin $arnela_override el API no resuelve ese DNS (red distinta)." >&2
  echo "      Ej.: export COMPOSE_OVERRIDE=$arnela_override" >&2
fi

echo "==> git fetch"
git fetch --all --prune

echo "==> git checkout $GIT_REF && pull"
git checkout "$GIT_REF"
git pull --ff-only origin "$GIT_REF"

echo "==> docker compose up -d --build"
docker compose "${compose_args[@]}" --env-file "$ENV_FILE" up -d --build

echo "==> Estado de contenedores"
# Sin --env-file, Compose vuelve a interpolar el YAML y falla (p. ej. NEXT_PUBLIC_API_URL).
docker compose "${compose_args[@]}" --env-file "$ENV_FILE" ps

echo "==> Health vía nginx (puerto 8102; el API no está publicado en el host)"
health_code=$(curl -sS -o /dev/null -w "%{http_code}" http://127.0.0.1:8102/health || echo "000")
echo "GET /health -> ${health_code}"
curl -sS -o /dev/null -w "GET /ready -> %{http_code}\n" http://127.0.0.1:8102/ready || true

if [[ "${health_code}" != "200" ]]; then
  echo "==> El API no responde OK (502/reinicios suelen ser DATABASE_URL, Postgres inalcanzable o JWT/CORS)."
  echo "==> Estado del contenedor API:"
  docker inspect -f '{{.State.Status}} exit={{.State.ExitCode}} err={{.State.Error}}' gonsgarage-api 2>/dev/null || true
  echo "==> Últimas líneas de log (gonsgarage-api):"
  docker logs gonsgarage-api --tail 80 2>&1 || true
fi

echo "==> Listo (GonsGarage)."
