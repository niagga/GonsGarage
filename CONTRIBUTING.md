# Contribuir a GonsGarage

## Herramientas

- **Go**: versión indicada en `backend/go.mod`.
- **Node.js**: 22+ recomendado.
- **pnpm**: 9.x (activar con `corepack enable` si usas `packageManager` del `frontend/package.json`).
- **Docker**: para Postgres + Redis locales (`docker-compose.yml` en la raíz).

## Instalación rápida

```powershell
# Raíz del repo
docker compose up -d

cd backend
copy .env.example .env   # o cp en bash
go run ./cmd/api

cd ../frontend
copy .env.local.example .env.local
pnpm install
pnpm dev
```

## TDD obligatorio

Antes de implementar una funcionalidad nueva o corregir un bug con impacto en comportamiento:

1. Escribir o actualizar un **test automatizado** que describa el comportamiento deseado.
2. Verificar que falla por la razón correcta.
3. Implementar el cambio mínimo para que pase.
4. Refactorizar manteniendo los tests verdes.

Detalle y convenciones: [docs/testing-tdd.md](docs/testing-tdd.md).

## Comprobaciones antes de abrir PR

```powershell
cd backend && go vet ./... && go test ./... -count=1
cd ../frontend && pnpm lint && pnpm typecheck && pnpm test
```

## Tasks VS Code / Cursor (verify tests + commit + push)

Hay tareas en [`.vscode/tasks.json`](.vscode/tasks.json):

| Task | Qué hace |
|------|-----------|
| **dev: verify (tests only)** | Solo tests: `go test` (backend, como CI) y `pnpm test` (frontend). No git. |
| **dev: verify + commit + push** | Tras los tests, pide **asunto** y **explicación**; genera un mensaje con análisis (`git status`, `diff --cached`, `diff` no staged) + bloque de verificación; hace `git commit -F` con lo **ya stageado** y `git push`. |
| **dev: verify + commit + push (Unix)** | Igual, usando `scripts/dev-verify-and-commit.sh`. |

**Antes** de `dev: verify + commit + push`: `git add` solo los ficheros que quieras en el commit. Si no hay nada en stage, el script falla (salvo `GONS_ALLOW_EMPTY_COMMIT=1` para un commit vacío de marcador).

Variables opcionales: `GONS_SKIP_PUSH=1` (no push), `GONS_GH_PR=1` + `gh` (comentario en PR), `GONS_ALLOW_EMPTY_COMMIT=1` (commit vacío si no hay stage).

Scripts: `scripts/dev-verify-and-commit.ps1` (Windows) y `scripts/dev-verify-and-commit.sh` (POSIX).

## Commits

Se recomienda [Conventional Commits](https://www.conventionalcommits.org/) (`feat:`, `fix:`, `test:`, `docs:`, etc.), alineado con el flujo descrito en [Agent.md](Agent.md).
