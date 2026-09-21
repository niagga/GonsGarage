# Análisis de la aplicación GonsGarage

**Última revisión:** 2026-06-01 — alineado con `backend/cmd/api/main.go` y `frontend/src/app/`.

## Propósito

Sistema de gestión para **taller mecánico** (una instancia = un taller): usuarios con roles, vehículos, citas, reparaciones, **órdenes de taller** (*service jobs*), **inventario de repuestos** y **contabilidad staff** (proveedores, facturas recibidas, documentos billing, facturas emitidas a clientes).

## Stack real (código actual)

| Capa | Tecnología |
|------|------------|
| Backend | Go 1.27.1, Gin, GORM (PostgreSQL), JWT, Redis opcional (cache con fallback si no hay conexión) |
| Frontend | Next.js 16.2.x (App Router), React 19.1, TypeScript 5, Zustand, Tailwind CSS 4 |
| Datos | PostgreSQL; migraciones vía `AutoMigrate` en `backend/cmd/api/main.go` y scripts SQL en `backend/scripts/` |
| Tests | Backend: `go test`; frontend: **Vitest** (por defecto), Jest legacy para algunos suites |

## Punto de entrada del backend

- **API:** `backend/cmd/api/main.go`
- **Seeds (solo desarrollo):** `backend/cmd/seed-test-client` (cliente demo), `backend/cmd/seed-mvp-users` (admin / manager / employee)
- **Swagger:** `GET /swagger/*any` (swaggo)
- **Salud:** `GET /health`, `GET /ready`
- **Prefijo API:** `/api/v1`
- **CORS:** con `GIN_MODE=release`, orígenes = localhost:3000/3001 más **`CORS_ORIGINS`** (coma-separado). Ver `corsMiddleware` en `main.go`. Despliegue Docker/LAN: [`deploy/README.md`](./deploy/README.md).

## Rutas API expuestas (resumen)

Fuente: `setupRoutes` en `backend/cmd/api/main.go`. JSON **camelCase** en respuestas.

### Públicas

- `POST /api/v1/auth/register`
- `POST /api/v1/auth/login`

### Protegidas (JWT)

| Dominio | Rutas | Notas |
|---------|--------|--------|
| Sesión | `GET /auth/me` | Perfil actual |
| Admin users | `POST /admin/users` | Solo admin/manager; no crea rol `admin` por este flujo |
| Empleados | `POST\|GET\|GET/:id\|PUT\|DELETE /employees/...` | Solo admin/manager |
| Repuestos | `POST\|GET\|GET/:id\|PATCH\|DELETE /parts/...` | Inventario (staff) |
| Coches | `POST\|GET\|GET/:id\|PUT\|DELETE /cars/...` | Listado por cliente: `GET /cars?ownerId=&limit=&offset=` |
| Citas | `POST\|GET\|GET/:id\|PUT\|DELETE /appointments/...` | Estado vía `PUT /appointments/:id` con `{ status, … }` |
| Reparaciones | `GET /repairs/car/:carId`, `POST\|GET\|PUT\|DELETE /repairs/...` | Escritura staff; cliente solo lectura por su coche |
| Taller (*service jobs*) | `POST\|GET /service-jobs`, `GET /service-jobs/car/:carId`, `GET\|PUT /service-jobs/:id/...` | Recepción `PUT …/reception`, entrega `PUT …/handover`; stub OBD `GET …/:id/obd` |
| Proveedores | CRUD `/suppliers/...` | Contabilidad P1 |
| Facturas recibidas | CRUD `/received-invoices/...` | Contabilidad P1 |
| Documentos billing | CRUD `/billing-documents/...` | Tipos: `payroll`, `irs`, `other` (facturas emitidas a clientes: agregado `Invoice` / `/invoices`) |
| Facturas emitidas | `GET /invoices/me`, `GET\|PATCH /invoices/:id` (cliente); staff `POST\|GET\|DELETE /invoices` | Cliente ve las suyas; staff emite y lista |

**Regresión por rol:** `backend/internal/handler/mvp_role_access_test.go`; spec [`openspec/specs/mvp-role-access/spec.md`](../openspec/specs/mvp-role-access/spec.md).

## Frontend (App Router)

Rutas bajo `frontend/src/app/` (principales):

| Área | Rutas |
|------|--------|
| Auth / home | `/`, `/auth/login`, `/auth/register`, `/dashboard`, `/client` |
| Flota y citas | `/cars`, `/cars/[id]`, `/appointments`, `/appointments/new` |
| Personal | `/employees`, `/admin/users` |
| Taller | `/workshop`, `/workshop/recepcion`, `/workshop/[id]` |
| Inventario | `/admin/parts`, `/admin/parts/new`, `/admin/parts/[id]` |
| Contabilidad (staff) | `/accounting`, `/accounting/suppliers`, `/accounting/received-invoices`, `/accounting/billing-documents`, `/accounting/issued-invoices` (+ subrutas `new`, `[id]`) |
| Cliente | `/my-invoices`, `/my-invoices/[id]` |

**Navegación:** cada `Link` / `router.push` a página debe existir como `page.tsx` bajo `src/app/` (regla Cursor: `.cursor/rules/nextjs-app-router-navigation.mdc`). Edición de citas: modal en lista, no rutas `/appointments/[id]/edit` inventadas.

**Cliente HTTP:** `frontend/src/lib/api-client.ts`; dominios en `lib/api/*` y `lib/services/*`; legacy `lib/api.ts` solo para rutas aún usadas por dashboard/coches.

## Seeds de demo (solo desarrollo)

| Comando | Roles / usuario |
|---------|------------------|
| `go run ./cmd/seed-test-client` | Cliente: `cliente.demo@gonsgarage.local` / `ClienteDemo123` (idempotente) |
| `go run ./cmd/seed-mvp-users` | Admin, manager, employee (`*.gonsgarage.local`; ver env `SEED_*` en `main.go`) |

No ejecutar seeds contra producción. Ver también [`mvp-solo-checklist.md`](./mvp-solo-checklist.md).

## Infraestructura y documentación

1. **Docker Compose** en raíz (`docker-compose.yml`): PostgreSQL + Redis (dev).
2. **Producción/LAN:** `docker-compose.prod.yml` + [`deploy/README.md`](../deploy/README.md) (nginx **8102**, Postgres en host, rollback, CORS, backup).
3. **CI:** [`.github/workflows/ci.yml`](../.github/workflows/ci.yml) — backend vet/test `-race`; frontend lint/typecheck/test/build.
4. **Deploy:** [`.github/workflows/deploy.yml`](../.github/workflows/deploy.yml) — **solo manual** (`workflow_dispatch`); despliegue real documentado en `deploy/`.

Guía operativa: [development-guide.md](./development-guide.md). Post-MVP: [mvp-next-steps.md](./mvp-next-steps.md). Planificación: [roadmap.md](./roadmap.md).

## Estructura de código (alto nivel)

```text
backend/
  cmd/api/main.go              # Arranque, AutoMigrate, rutas Gin
  cmd/seed-test-client/        # Seed cliente demo
  cmd/seed-mvp-users/          # Seed admin / manager / employee
  internal/domain              # Entidades
  internal/service/            # Casos de uso (auth, car, appointment, repair, servicejob, part, …)
  internal/handler/            # HTTP Gin
  internal/middleware/         # JWT, roles, CORS
  internal/repository/postgres|redis|mock
  tests/integration/           # Tests HTTP integración

frontend/
  src/app/                     # App Router (páginas)
  src/components/              # UI compartida
  src/stores/                  # Zustand
  src/lib/api/                 # Cliente HTTP por dominio
```

## OpenSpec (SDD)

Especificaciones de dominio en `openspec/specs/` (p. ej. `mvp-role-access`, `workshop-repair-execution`, `parts-inventory`, `invoices`, `billing`, `suppliers`). Changes activos y archivados en `openspec/changes/`.

## Referencias cruzadas

- Convenciones de código: [../Agent.md](../Agent.md)
- Cliente API frontend: [../frontend/docs/api-client.md](../frontend/docs/api-client.md)
- Checklist MVP: [mvp-solo-checklist.md](./mvp-solo-checklist.md)
