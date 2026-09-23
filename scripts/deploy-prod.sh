#!/usr/bin/env bash
#
# GonsGarage — deploy / rollback en el servidor (git pull + docker compose).
#
# No copia código desde tu PC: usa el clone en el servidor y lo que ya está en Git.
#
# Uso (en el servidor, p. ej. /DATA/AppData/gonsgarage):
#   export COMPOSE_OVERRIDE=docker-compose.prod.arnela-network.yml   # si DATABASE_URL usa arnela-postgres
#   bash scripts/deploy-prod.sh deploy
#   bash scripts/deploy-prod.sh rollback
#   bash scripts/deploy-prod.sh status
#
# Variables opcionales:
#   GONSGARAGE_DIR   (default: /DATA/AppData/gonsgarage)
#   GIT_REF          (default: main)
#   COMPOSE_FILE     (default: docker-compose.prod.yml)
#   COMPOSE_OVERRIDE (default: vacío; obligatorio con arnela-postgres)
#   ENV_FILE         (default: .env.prod)
#   HEALTH_URL       (default: http://127.0.0.1:8102/health)
#   READY_URL        (default: http://127.0.0.1:8102/ready)
#   HEALTH_RETRIES   (default: 30)
#   HEALTH_SLEEP_SEC (default: 2)
#
set -euo pipefail

: "${GONSGARAGE_DIR:=/DATA/AppData/gonsgarage}"
: "${GIT_REF:=main}"
: "${COMPOSE_FILE:=docker-compose.prod.yml}"
: "${COMPOSE_OVERRIDE:=}"
: "${ENV_FILE:=.env.prod}"
: "${HEALTH_URL:=http://127.0.0.1:8102/health}"
: "${READY_URL:=http://127.0.0.1:8102/ready}"
: "${HEALTH_RETRIES:=30}"
: "${HEALTH_SLEEP_SEC:=2}"

LAST_GOOD_FILE=".deploy-last-good"
PRE_DEPLOY_FILE=".deploy-pre-deploy"

usage() {
  cat <<'EOF'
Uso: bash scripts/deploy-prod.sh <comando>

Comandos:
  deploy     git fetch/pull de GIT_REF + docker compose up -d --build + health/ready
  rollback   checkout del último SHA marcado como estable (.deploy-last-good) + rebuild + health
  status     muestra HEAD, last-good, contenedores y curls de health/ready

Ejemplos:
  export COMPOSE_OVERRIDE=docker-compose.prod.arnela-network.yml
  bash scripts/deploy-prod.sh deploy
  bash scripts/deploy-prod.sh rollback
EOF
}

die() {
  echo "Error: $*" >&2
  exit 1
}

require_files() {
  [[ -f "$COMPOSE_FILE" ]] || die "no se encuentra $COMPOSE_FILE en $(pwd)"
  [[ -f "$ENV_FILE" ]] || die "falta $ENV_FILE (variables de producción; no versionar)"
}

compose_args=()

setup_compose_args() {
  compose_args=( -f "$COMPOSE_FILE" )
  if [[ -n "$COMPOSE_OVERRIDE" ]]; then
    [[ -f "$COMPOSE_OVERRIDE" ]] || die "COMPOSE_OVERRIDE=$COMPOSE_OVERRIDE no existe en $(pwd)"
    compose_args+=( -f "$COMPOSE_OVERRIDE" )
  fi

  if grep -qE '@arnela-postgres[:/]|//arnela-postgres' "$ENV_FILE" 2>/dev/null && [[ -z "${COMPOSE_OVERRIDE:-}" ]]; then
    local arnela_override="docker-compose.prod.arnela-network.yml"
    if [[ -f "$arnela_override" ]]; then
      die "$ENV_FILE referencia arnela-postgres pero COMPOSE_OVERRIDE está vacío.
      Sin -f $arnela_override el API no resuelve ese DNS y suele reiniciar (502).
      Ej.: export COMPOSE_OVERRIDE=$arnela_override"
    fi
    echo "WARN: $ENV_FILE usa arnela-postgres sin COMPOSE_OVERRIDE." >&2
  fi
}

compose() {
  docker compose "${compose_args[@]}" --env-file "$ENV_FILE" "$@"
}

http_code() {
  local url="$1"
  curl -sS -o /dev/null -w "%{http_code}" "$url" 2>/dev/null || echo "000"
}

verify_health() {
  local health_code="000"
  local ready_code="000"
  local i

  echo "==> Verificando health/ready (hasta ${HEALTH_RETRIES} intentos)..."
  for ((i = 1; i <= HEALTH_RETRIES; i++)); do
    health_code="$(http_code "$HEALTH_URL")"
    ready_code="$(http_code "$READY_URL")"
    echo "  intento ${i}/${HEALTH_RETRIES}: health=${health_code} ready=${ready_code}"
    if [[ "$health_code" == "200" && "$ready_code" == "200" ]]; then
      echo "==> Health OK"
      curl -sS "$HEALTH_URL" || true
      echo
      curl -sS "$READY_URL" || true
      echo
      return 0
    fi
    sleep "$HEALTH_SLEEP_SEC"
  done

  echo "==> Health/ready NO OK (health=${health_code} ready=${ready_code})" >&2
  echo "==> Contenedores:" >&2
  compose ps >&2 || true
  echo "==> Logs gonsgarage-api (tail 80):" >&2
  docker logs gonsgarage-api --tail 80 2>&1 || true
  return 1
}

mark_last_good() {
  local sha
  sha="$(git rev-parse HEAD)"
  {
    echo "sha=${sha}"
    echo "ref=$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo detached)"
    echo "recorded_at=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
    echo "subject=$(git log -1 --pretty=%s | tr '\n' ' ')"
  } >"$LAST_GOOD_FILE"
  echo "==> Marcado estable: $sha -> $LAST_GOOD_FILE"
}

read_last_good_sha() {
  [[ -f "$LAST_GOOD_FILE" ]] || die "no hay $LAST_GOOD_FILE; no se puede rollback automático.
      Tras un deploy exitoso el script lo crea. Alternativa: git checkout <sha> && bash scripts/deploy-prod.sh deploy (con GIT_REF=<sha> no aplica pull de main; usá rollback solo con last-good)."
  # shellcheck disable=SC2002
  sed -n 's/^sha=//p' "$LAST_GOOD_FILE" | head -1
}

cmd_status() {
  echo "==> Directorio: $(pwd)"
  echo "==> HEAD: $(git rev-parse --short HEAD 2>/dev/null || echo n/a) $(git log -1 --pretty=%s 2>/dev/null || true)"
  if [[ -f "$LAST_GOOD_FILE" ]]; then
    echo "==> Last-good:"
    cat "$LAST_GOOD_FILE"
  else
    echo "==> Last-good: (ninguno)"
  fi
  compose ps || true
  echo "==> curl health: $(http_code "$HEALTH_URL")"
  curl -sS "$HEALTH_URL" 2>/dev/null || true
  echo
  echo "==> curl ready: $(http_code "$READY_URL")"
  curl -sS "$READY_URL" 2>/dev/null || true
  echo
}

cmd_deploy() {
  echo "==> Directorio: $(pwd)"
  echo "==> Rama/ref: $GIT_REF"
  echo "==> Compose: ${compose_args[*]}"

  if git rev-parse HEAD >/dev/null 2>&1; then
    git rev-parse HEAD >"$PRE_DEPLOY_FILE"
    echo "==> Pre-deploy SHA: $(cat "$PRE_DEPLOY_FILE")"
  fi

  echo "==> git fetch"
  git fetch --all --prune

  echo "==> git checkout $GIT_REF && pull --ff-only"
  git checkout "$GIT_REF"
  git pull --ff-only origin "$GIT_REF"

  local new_sha
  new_sha="$(git rev-parse HEAD)"
  echo "==> Deploy SHA: $new_sha ($(git log -1 --pretty=%s))"

  echo "==> docker compose up -d --build"
  compose up -d --build

  echo "==> Estado de contenedores"
  compose ps

  if verify_health; then
    mark_last_good
    rm -f "$PRE_DEPLOY_FILE"
    echo "==> Deploy listo."
    return 0
  fi

  echo "==> Deploy falló health/ready. Last-good NO se actualizó." >&2
  if [[ -f "$LAST_GOOD_FILE" ]]; then
    echo "      Para volver al último estable: bash scripts/deploy-prod.sh rollback" >&2
  fi
  exit 1
}

cmd_rollback() {
  local target
  target="$(read_last_good_sha)"
  [[ -n "$target" ]] || die "SHA vacío en $LAST_GOOD_FILE"

  local current
  current="$(git rev-parse HEAD)"
  echo "==> Rollback: $current -> $target"
  if [[ "$current" == "$target" ]]; then
    echo "==> Ya estás en el SHA last-good; solo rebuild + health."
  else
    echo "==> git fetch + checkout $target"
    git fetch --all --prune
    git checkout --detach "$target"
  fi

  echo "==> docker compose up -d --build"
  compose up -d --build
  compose ps

  if verify_health; then
    echo "==> Rollback listo (sigue siendo el last-good registrado)."
    return 0
  fi

  echo "==> Rollback NO pasó health/ready. Revisá logs / DATABASE_URL." >&2
  exit 1
}

main() {
  local cmd="${1:-}"
  if [[ -z "$cmd" ]]; then
    usage
    exit 1
  fi
  shift || true

  cd "$GONSGARAGE_DIR"
  require_files
  setup_compose_args

  case "$cmd" in
    deploy) cmd_deploy "$@" ;;
    rollback) cmd_rollback "$@" ;;
    status) cmd_status "$@" ;;
    -h|--help|help) usage ;;
    *)
      usage >&2
      die "comando desconocido: $cmd"
      ;;
  esac
}

main "$@"
