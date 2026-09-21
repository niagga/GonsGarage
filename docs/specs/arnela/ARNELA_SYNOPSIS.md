# Arnela — resumen para GonsGarage

**Fuente canónica (local):** `D:\Repos\Arnela`  
**Fecha de lectura / revisión:** 2026-04-16 · **Revisión GonsGarage:** 2026-04-20 (`arnela-parity`)  
**Nota:** Este archivo es un resumen operativo; el detalle vive en el repo Arnela (especialmente `arnela-rules/` y `docs/`).

## Qué es Arnela

CRM/CMS para **gabinete profesional** (caso Arnela Gabinete, Vigo): sustituye Excel y herramientas dispersas por una plataforma unificada.

**Tres superficies:**

1. Landing (réplica web + login modal)  
2. Área cliente (auto-gestión de citas)  
3. Backoffice (CRM/CMS: clientes, empleados, citas, tareas, informes)

Idioma principal: **español**.

## Stack (README y `arnela-rules/tech-stack.md`)

| Capa | Arnela |
|------|--------|
| Backend | Go **1.27** (baseline actual en GonsGarage), **Gin**, Clean Architecture / **monolito modular** |
| Acceso BD | **sqlx** + migraciones **golang-migrate** (15 SQL en `migrations/`) |
| Frontend | **Next.js 16**, TypeScript, **Zustand**, **Tailwind v4**, **Shadcn/Radix** |
| BD | PostgreSQL **16** |
| Cache / cola | **Redis 8** (cache-aside, worker pool async) |
| Auth | JWT, roles `admin` / `employee` / `client`, **rate limiting** |
| Docs API | Swagger/OpenAPI (swaggo) |
| Infra | **Docker Compose** en raíz, Nginx en prod, **GitHub Actions** CI |
| Frontend pkg manager | **pnpm** (Node **24+**) |

## Estructura backend (reglas)

- Entrada: `backend/cmd/api/main.go`  
- `internal/domain`, `repository` (+ `postgres/`, `mocks/`), `service`, `handler`, `middleware`  
- `pkg/`: database, cache, email, errors, gcal, jwt, logger, pdf, queue, etc.  
- Endpoints de ejemplo documentados: `POST/GET /api/v1/auth/*`, **GET `/api/v1/auth/me`**, CRUD `/api/v1/users`, `/api/v1/clients` (ver `arnela-rules/api-endpoints.md`).  
- Health adicional: **readiness** en `http://localhost:8080/readiness` (README Arnela). **En GonsGarage** el contrato equivalente de “listo para tráfico” es **`GET /ready`** (mismo proceso que `/health` con chequeos adicionales en código); detrás de nginx en prod: `http://<host>:8102/ready`.

## Estructura frontend (reglas)

- App Router con grupos `(auth)`, `(client)`, `(backoffice)`  
- `components/ui` (Shadcn), `common`, `client`, `backoffice`  
- `stores/`, `hooks/`, `lib/` (cliente API), `types/`

## Convenciones alineables con GonsGarage

- JSON **camelCase** en API (`json:"firstName"` etc.) — igual que `Agent.md` de GonsGarage.  
- Naming Go/TS como en `arnela-rules/conventions.md` (PascalCase exportado, camelCase interno).

## Documentación y proceso

- Índice maestro: `Arnela/docs/DOCUMENTATION_INDEX.md`  
- Contribución: `CONTRIBUTING.md` (Docker Compose raíz, `.env.example` backend/frontend, branches `feat/`, `fix/`, Conventional Commits)  
- Reglas compactas para agentes: `Arnela/arnela-rules/*.md`  
- Skill registry: `Arnela/.atl/skill-registry.md`

## Diferencias importantes respecto a GonsGarage

| Aspecto | Arnela | GonsGarage |
|---------|--------|------------|
| Dominio | Clientes (CRM), citas, facturación, tareas, integraciones (GCal, email, PDF) | Taller: coches, citas, empleados, reparaciones (API repairs aún no expuesta) |
| Persistencia | sqlx + migraciones SQL versionadas | GORM + AutoMigrate en `cmd/api` (+ scripts SQL sueltos) |
| Compose | `docker-compose.yml` en **raíz** (PG + Redis) | **Actualizado:** `docker-compose.yml` raíz (PG+Redis) alineado con defaults del backend |
| CI | GitHub Actions | **`.github/workflows/ci.yml`** (Go + frontend) y `deploy.yml` (manual) |
| Frontend | pnpm, Tailwind v4, Shadcn | **pnpm**, Next 16; Tailwind v4 / Shadcn no replicados; estructura `app/` por features (auth, cars, accounting, etc.) |

## Archivos clave a abrir en Arnela

| Ruta (desde raíz Arnela) | Contenido |
|--------------------------|-----------|
| `README.md` | Stack, quick start, estructura |
| `arnela-rules/core-identity.md` | Objetivo y alcance |
| `arnela-rules/tech-stack.md` | Tabla de tecnologías |
| `arnela-rules/architecture.md` | Capas backend/frontend |
| `arnela-rules/backend-structure.md` | Árbol de carpetas Go |
| `arnela-rules/frontend-structure.md` | Árbol Next.js |
| `arnela-rules/domain-models.md` | User, Client, roles |
| `arnela-rules/api-endpoints.md` | Tabla de rutas |
| `arnela-rules/conventions.md` | Naming y JSON |
| `docs/DOCUMENTATION_INDEX.md` | Índice de toda la doc |
| `CONTRIBUTING.md` | Setup y workflow |
| `docker-compose.yml` | Servicios locales |
