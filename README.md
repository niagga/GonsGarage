# GonsGarage

Auto repair shop management system: **Go** API (Gin, GORM, PostgreSQL, Redis) and **Next.js** web app (App Router, React, Zustand).

[![CI](https://img.shields.io/badge/CI-GitHub_Actions-blue)](.github/workflows/ci.yml)
[![Go](https://img.shields.io/badge/Go-1.27-blue)](backend/go.mod)
[![Next.js](https://img.shields.io/badge/Next.js-16.2-black)](frontend/package.json)
[![React](https://img.shields.io/badge/React-19-61dafb)](frontend/package.json)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)

## Documentation

Technical deep-dives (architecture, env vars, TDD, roadmap) live under **[`docs/`](docs/)** — start at [`docs/README.md`](docs/README.md).

| Topic | Link |
|--------|------|
| Development setup | [`docs/development-guide.md`](docs/development-guide.md) |
| Docker / LAN deploy (plantilla Arnela) | [`deploy/README.md`](deploy/README.md) |
| TDD / testing | [`docs/testing-tdd.md`](docs/testing-tdd.md) |
| Contributing | [`CONTRIBUTING.md`](CONTRIBUTING.md) |
| Agent / code standards | [`Agent.md`](Agent.md) |
| P1 accounting (recibidas, billing, suppliers, facturas cliente) | Specs: [`openspec/specs/invoices/`](openspec/specs/invoices/spec.md), [`billing/`](openspec/specs/billing/spec.md), [`suppliers/`](openspec/specs/suppliers/spec.md); change archivado [`openspec/changes/archive/2026-04-20-p1-invoices-billing-suppliers/`](openspec/changes/archive/2026-04-20-p1-invoices-billing-suppliers/) — API `/api/v1/`; UI staff `/accounting`, cliente `/my-invoices` |

## Verified stack (source of truth)

Versions below are taken from the repo manifests as of the last README refresh. Upgrade the files first, then update this table.

| Layer | Technology | Where it is defined |
|--------|------------|---------------------|
| Backend runtime | **Go 1.27** (`go 1.27.1` directive) | [`backend/go.mod`](backend/go.mod) |
| HTTP | Gin, JWT, GORM, sqlx, Redis client | `backend/go.mod` |
| Database (local) | **PostgreSQL 16** (`postgres:16-alpine`) | [`docker-compose.yml`](docker-compose.yml) |
| Cache (local) | **Redis 8** (`redis:8-alpine`) | [`docker-compose.yml`](docker-compose.yml) |
| Frontend | **Next.js 16.2.4**, **React 19.1.0**, TypeScript **^5**, Tailwind **4** | [`frontend/package.json`](frontend/package.json) |
| Package manager | **pnpm 12.5.1** (`packageManager` field) | [`frontend/package.json`](frontend/package.json) |
| Unit / component tests (default) | **Vitest** + Testing Library | `frontend/package.json` → `pnpm test` |
| Lint / types | ESLint 9, `eslint-config-next` aligned with Next | `frontend/package.json` |

CI runs **Node 24**, **pnpm 12.5.1**, `pnpm lint` → `pnpm typecheck` → `pnpm test` → `pnpm build` for the frontend, and **Go from `go.mod`** for `vet` / `test -race` / `build` on the backend (see [`.github/workflows/ci.yml`](.github/workflows/ci.yml)).

## Prerequisites

- **Node.js** 24+ and **pnpm** 12+ ([`corepack enable`](https://nodejs.org/api/corepack.html) recommended)
- **Go** 1.27+ (match `backend/go.mod`)
- **Docker** with Compose v2 (`docker compose`) for local PostgreSQL and Redis
- **Git**

## Quick start

### 1. Clone

```bash
git clone https://github.com/gaston-garcia-cegid/GonsGarage.git
cd GonsGarage
```

### 2. Environment files

```bash
cp backend/.env.example backend/.env
cp frontend/.env.local.example frontend/.env.local
```

On **Windows** (CMD/PowerShell), use `copy .env.example .env` in `backend/` and `copy .env.local.example .env.local` in `frontend/`, or `Copy-Item` as in [`docs/development-guide.md`](docs/development-guide.md).

### 3. Databases (recommended)

From the **repository root**:

```bash
docker compose up -d
```

Services: **postgres** (port `5432`) and **redis** (`6379`). Defaults match `backend/.env.example` and `backend/cmd/api/main.go` when `DATABASE_URL` / `REDIS_URL` are unset.

### 4. Backend API

```bash
cd backend
go mod download
go run ./cmd/api
```

- API: <http://localhost:8080> (or `SERVER_PORT`)
- Swagger UI: <http://localhost:8080/swagger/index.html>
- Health: <http://localhost:8080/health> · Readiness: <http://localhost:8080/ready>

Regenerate Swagger after changing `// @Summary` / `// @Router` annotations:

```bash
cd backend
go run github.com/swaggo/swag/cmd/swag@v1.8.12 init -g main.go -o docs -d ./cmd/api,./internal/handler,./internal/core/ports --parseInternal
```

### 5. Frontend

```bash
cd frontend
pnpm install
pnpm dev
```

App: <http://localhost:3000>. Set `NEXT_PUBLIC_API_URL` in `frontend/.env.local` if the API is not on `http://localhost:8080` (value is the **host only**, no `/api/v1`; the HTTP client adds `/api/v1` automatically).

**Suggested first-time order:** clone → env (step 2) → `docker compose up -d` → `go run ./cmd/api` in `backend/` → `pnpm dev` in `frontend/`. Step-by-step (PowerShell, optional seed): [`docs/development-guide.md`](docs/development-guide.md) — section **Demo local (secuencia mínima, checklist 3.1)** at the top.

## Testing (mirrors CI)

**Backend**

```bash
cd backend
go vet ./...
go test ./... -count=1 -race
```

**Frontend**

```bash
cd frontend
pnpm lint
pnpm typecheck
pnpm test
pnpm build
```

The default **`pnpm test`** script runs **Vitest**. Jest remains available for legacy scripts (`pnpm test:jest`) but is not the primary runner documented in CI.

### Option B — local tests via Docker (no host toolchain)

If your host doesn't have Go/Node/pnpm installed, run checks in containers from the repository root:

```bash
# Backend (Go 1.27)
docker run --rm -v "$PWD/backend":/src -w /src golang:1.27 \
  sh -lc 'CGO_ENABLED=1 /usr/local/go/bin/go mod download && CGO_ENABLED=1 /usr/local/go/bin/go test ./... -count=1 -race -timeout=2m'

# Frontend (Node 24 + pnpm 12)
docker run --rm -v "$PWD/frontend":/work -w /work node:24-alpine \
  sh -lc 'corepack enable && corepack prepare pnpm@12.5.1 --activate && (pnpm install || (pnpm approve-builds --all && pnpm install)) && pnpm lint && pnpm typecheck && pnpm test && pnpm build'
```

## Demo users

**Solo desarrollo** — con PostgreSQL en marcha y tablas migradas (arrancá la API al menos una vez). Los seeds son **idempotentes** (segunda ejecución sin duplicar).

```bash
cd backend
go run ./cmd/seed-mvp-users      # admin, manager, employee
go run ./cmd/seed-test-client    # cliente demo
```

| Role | Email (default) | Password (default) | Seed command |
|------|-----------------|-------------------|--------------|
| Admin | `admin.demo@gonsgarage.local` | `AdminDemo123` | `seed-mvp-users` |
| Manager | `manager.demo@gonsgarage.local` | `ManagerDemo123` | `seed-mvp-users` |
| Employee | `employee.demo@gonsgarage.local` | `EmployeeDemo123` | `seed-mvp-users` |
| Client | `cliente.demo@gonsgarage.local` | `ClienteDemo123` | `seed-test-client` |

Override emails/passwords with `SEED_ADMIN_*`, `SEED_MANAGER_*`, `SEED_EMPLOYEE_*`, `SEED_CLIENT_*`, and `DATABASE_URL` (see comments in each `cmd/seed-*/main.go`).

The login UI may still show `admin@gonsgarage.com` / `admin123` as a hint — use the seeded admin above or **Register** if you prefer that account.

## Project layout

```text
backend/          Go API (cmd/api, internal/, docs/swagger)
frontend/         Next.js App Router (src/app, components, stores)
docs/             Technical documentation index
openspec/         Spec-driven development (OpenSpec)
docker-compose.yml   Local Postgres + Redis
```

## Contributing

See [`CONTRIBUTING.md`](CONTRIBUTING.md) and [`docs/testing-tdd.md`](docs/testing-tdd.md) for branch workflow and TDD expectations.

## License

[MIT](LICENSE).
