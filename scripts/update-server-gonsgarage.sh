#!/usr/bin/env bash
#
# Compatibilidad: apunta al flujo canónico scripts/deploy-prod.sh
# Preferí: bash scripts/deploy-prod.sh deploy
#
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
exec bash "$ROOT/deploy-prod.sh" deploy "$@"
