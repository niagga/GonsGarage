# Plantilla tecnológica (estilo Arnela) — input para nuevo proyecto

**Referencia:** [Arnela en GitHub](https://github.com/gaston-garcia-cegid/Arnela.git) — `README.md`, `arnela-rules/*`, `CONTRIBUTING.md`, `Agent.md`, `frontend/package.json`, `backend/go.mod`, `.github/workflows/ci.yml`.

**Rellena los campos `{{...}}` antes de usar este archivo como prompt.**

| Campo | Valor |
|--------|--------|
| **Nombre del proyecto** | `{{PROJECT_NAME}}` |
| **Dominio de negocio** | `{{BUSINESS_DOMAIN}}` |
| **Idioma UI/docs principal** | `{{LOCALE}}` (ej. `es`) |

---

## 1. Arquitectura

| Capa | Decisión |
|------|----------|
| **Backend** | **Monolito modular** en Go: **Clean Architecture** — transporte (`handler/`, `cmd/`, `middleware/`), reglas (`service/`), entidades (`domain/`), persistencia (`repository/` + Postgres), utilidades compartidas (`pkg/`). |
| **Frontend** | **Next.js (App Router)** + TypeScript: aplicación **monolítica** (rutas por segmentos: público, cliente, staff). Estado global con **Zustand**. |
| **Datos** | **PostgreSQL** como fuente de verdad; **Redis** para caché, rate limiting y/o **cola de tareas async** (emails, notificaciones, jobs). |
| **API** | REST JSON bajo prefijo estable (ej. `/api/v1`); contrato documentado con **OpenAPI/Swagger** generado desde el backend. |
| **Despliegue dev** | **Docker Compose** (Postgres + Redis mínimo); opcional `docker-compose.prod.yml` + reverse proxy (ej. Nginx). |
| **CI** | GitHub Actions: backend `vet` → `build` → `test -race`; frontend `lint` → `tsc --noEmit` → `vitest` → `build`. |

---

## 2. Stack tecnológico

| Componente | Elección |
|--------------|----------|
| **Runtime backend** | Go **1.27** (`go.mod` como referencia de versión). |
| **HTTP** | **Gin** (`github.com/gin-gonic/gin`). |
| **BD** | **PostgreSQL 16**; acceso con **sqlx** (`github.com/jmoiron/sqlx`); driver `lib/pq`. |
| **Migraciones** | **golang-migrate** (`github.com/golang-migrate/migrate/v4`), SQL en `migrations/`. |
| **Caché / cola** | **Redis** (`github.com/go-redis/redis/v8`); tests con **miniredis** donde aplique. |
| **Node** | **24+**; gestor **pnpm 12+**; CI con `--frozen-lockfile`. |
| **Frontend** | **Next.js 16**, **React 19**, **TypeScript 5.9+**. |
| **CSS** | **Tailwind CSS v4** (`tailwindcss`, `@tailwindcss/postcss`). |
| **Logging** | **Zerolog** (`github.com/rs/zerolog`) estructurado. |
| **Config** | Variables de entorno (`joho/godotenv` en dev); sin secretos en repo. |
| **Probes** | `GET /health` y `GET /readiness`. |

---

## 3. Librerías y herramientas

### Backend (Go)

| Área | Librerías |
|------|-----------|
| **Auth / sesión** | JWT `github.com/golang-jwt/jwt/v5`; passwords `golang.org/x/crypto`. |
| **HTTP / CORS** | Gin + `github.com/gin-contrib/cors`. |
| **Validación** | Cadena Gin / `go-playground/validator`. |
| **IDs** | `github.com/google/uuid`. |
| **OpenAPI** | `github.com/swaggo/swag`, `github.com/swaggo/gin-swagger`, `github.com/swaggo/files`. |
| **Tests** | `github.com/stretchr/testify` (assert + mock); tests **table-driven** en **services**. |

### Frontend (TypeScript)

| Área | Librerías |
|------|-----------|
| **UI / accesibilidad** | **Radix UI** + **Shadcn** (`components/ui`). |
| **Utilidades CSS** | `class-variance-authority`, `clsx`, `tailwind-merge`, `tailwindcss-animate`. |
| **Iconos** | `lucide-react`. |
| **Tema** | `next-themes`. |
| **Formularios** | `react-hook-form` + `zod` + `@hookform/resolvers`. |
| **Fechas** | `date-fns` + `react-day-picker`. |
| **Toasts** | `sonner`. |
| **Export** (opcional) | `xlsx`. |
| **Errores cliente** (opcional) | `@sentry/browser`. |
| **Lint / types** | ESLint 9 + `eslint-config-next` + `typescript-eslint`. |
| **Tests** | **Vitest** + **Testing Library** + **jsdom**. |

### Infra / calidad

| Área | Herramienta |
|------|-------------|
| **CI** | GitHub Actions (jobs `backend` / `frontend`, `working-directory` por carpeta). |
| **Contenedores** | Docker + Compose; `.env.example` en backend y frontend. |

---

## 4. Rules (convenciones y gobernanza)

1. **Ramas:** `main` protegida; features `feat/...`, fixes `fix/...`.
2. **Commits:** Conventional Commits (`feat:`, `fix:`, `chore:`, `docs:`, `test:`, `refactor:`).
3. **PRs:** desde `main`, descripción clara, **CI verde**; API reflejada en Swagger.
4. **API JSON:** claves **camelCase**; en Go, tags `json:"camelCase"` alineados con TypeScript.
5. **Naming:** Go — `PascalCase` exportado, `camelCase` interno; TS — `PascalCase` componentes/tipos, `camelCase` props/funciones.
6. **Testing:** TDD preferente en **servicios** backend; mocks de repos con `testify/mock`; frontend: hooks, cliente API, validadores, componentes críticos.
7. **Estructura:** `internal/domain|service|repository|handler|middleware` y `frontend/src/{app,components,hooks,lib,stores,types}`.
8. **Validación de dominio:** reglas por mercado — definir en `{{VALIDATION_RULES}}` (ej. fiscal/teléfono/documento por país).
9. **Docs / agentes:** `docs/DOCUMENTATION_INDEX.md`; reglas compactas en carpeta tipo `{{project}}-rules/` (equivalente a `arnela-rules/`).
10. **Seguridad:** rate limiting en auth pública; JWT `Authorization: Bearer`; roles explícitos según producto.

---

## 5. Alcance para el primer prompt (rellenar)

- **MVP (qué entra sí o sí):** `{{MVP_SCOPE}}`
- **Integraciones obligatorias:** `{{INTEGRATIONS}}` (pagos, delivery, fiscal, calendario, etc.)
- **Multi-local / single-tenant:** `{{TENANCY}}`
- **Ruta o URL del repo (si ya existe):** `{{REPO}}`

---

## 6. Instrucción corta para el asistente

> Usa este documento como especificación base del repositorio. Genera (o adapta) la estructura `backend/` + `frontend/`, `docker-compose.yml`, CI, `README`, y las carpetas de reglas alineadas con las secciones 1–4. Respeta `{{LOCALE}}` y el dominio `{{BUSINESS_DOMAIN}}`. No añadas stack no listado salvo que lo propongas explícitamente con motivo.
