# Fases mínimas para un primer MVP (GonsGarage)

Objetivo del MVP: **un taller puede registrar usuarios, clientes con coches, pedir citas y el personal puede ver/gestionar lo básico** con despliegue reproducible y calidad mínima aceptable.

**Plan consolidado (archivado):** [mvp-funcionando-plan proposal](../openspec/changes/archive/2026-04-20-mvp-funcionando-plan/proposal.md). **Siguientes pasos:** [mvp-next-steps.md](./mvp-next-steps.md).

**Brecha MVP vs Entra 1.1 + roadmap P0–P3 (tareas SDD):** [proposal archivado](../openspec/changes/archive/2026-04-19-mvp-gap-roadmap-2026/proposal.md) · [`tasks.md` archivado](../openspec/changes/archive/2026-04-19-mvp-gap-roadmap-2026/tasks.md).

**Checklist operativo (equipo de una persona, servidor de pruebas = staging):** [mvp-solo-checklist.md](./mvp-solo-checklist.md).

## Fase A — Base técnica (cerrada)

- [x] Compose local (Postgres + Redis) y variables documentadas (`docker-compose.yml`, `backend/.env.example`, `frontend/.env.local.example`, `docs/development-guide.md`).
- [x] CI en GitHub (backend + frontend) en `.github/workflows/ci.yml`.
- [x] Política **TDD** en `docs/testing-tdd.md` y tests base: backend `auth`, `car`, `appointment` (`*_service_test.go`); frontend stores/API (`__tests__/auth.*`, `car.store`, `appointment.*`).

**Criterio de salida:** `main` con CI verde; README y `docs/development-guide.md` reflejan **pnpm** y **`go run ./cmd/api`**.

## Fase B — Identidad y datos coherentes (cerrada)

- Roles claros (`client` / `employee` / `admin` / `manager`) y comprobaciones en API alineadas con la UI.
- Flujo **register + login + JWT** estable end-to-end.
- **Coches y citas** CRUD coherentes con el contrato JSON camelCase; **OpenAPI generado con swag** (`backend/docs/`, UI `/swagger/index.html`).

**Avances en código (iteración actual):**

- [x] `GET /api/v1/auth/me` (JWT) — perfil del usuario autenticado; JSON **camelCase** en registro y en `me`.
- [x] Registro público: rol por defecto **`client`** si no se envía `role`; conflicto con `409` usando `ErrUserAlreadyExists`.
- [x] Rutas **`/employees/*`** restringidas a **admin** y **manager** (`RequireStaffManagers`).
- [x] **Coches:** cliente solo su flota; personal (`employee` / `manager` / `admin`) puede **listar inventario** (`GET /cars?limit=&offset=&ownerId=`) y **crear coche para un cliente** (`ownerID` en el body si no es `client`). Lectura de cualquier coche para staff en servicio de dominio.
- [x] **Citas:** listado acotado a **customer_id** del cliente; staff ve/filtra con query params; creación valida que el **coche pertenezca al cliente** de la cita; actualización conserva `customer_id` y admite `scheduledAt` opcional (RFC3339).
- [x] **Frontend:** `UserRole.manager`, `canManageUsers` alineado con backend; **login** y **checkAuthStatus** validan sesión con **`/auth/me`**.

**Criterio de salida:** demo en local: cliente crea coche y cita; empleado/admin lista y actualiza estados básicos.

## Fase C — Reparaciones (cerrada)

- [x] **API:** `GET /api/v1/repairs/car/:carId` (lista por coche; permisos en servicio).
- [x] **UI cliente:** detalle de coche y dashboard con historial (lectura).
- [x] **Staff CRUD repairs (MVP+):** `POST|GET|PUT|DELETE /api/v1/repairs…` + panel en `/cars/[id]` — checklist [mvp-solo-checklist.md](./mvp-solo-checklist.md) ítem **6.1** (2026-04-19).

**Criterio de salida:** cumplido (lectura cliente + escritura staff).

## Fase D — MVP en producción (parcialmente cerrada en LAN)

- [x] **Secrets** documentados fuera del repo (`.env.prod.example`, `deploy/README.md`; checklist fases 4–5).
- [x] **Stack Docker prod/LAN** (`docker-compose.prod.yml`, nginx :8102, smoke documentado).
- [x] **Deploy workflow** — [`.github/workflows/deploy.yml`](../.github/workflows/deploy.yml) con política **solo manual**; runbook en `deploy/README.md` (no CD automático al servidor).
- [ ] **HTTPS** en entorno expuesto (doc LAN vs TLS en `deploy/README.md`; opcional si solo red interna).
- [ ] **Automatizar CI → servidor** (opcional; ver checklist Fase **6.2** — explícitamente no priorizado).

**Criterio de salida mínimo:** entorno de pruebas accesible con backup y rollback documentados (**cumplido** en LAN según checklist). HTTPS/CD automático = mejora posterior.

## Post-MVP entregado (referencia, no bloquea Fase D)

Funcionalidad ya en `main` además del MVP v1 original; detalle en [application-analysis.md](./application-analysis.md) y specs `openspec/specs/`:

- [x] Contabilidad P1 — API + UI `/accounting/*`, cliente `/my-invoices` (change archivado `2026-04-20-p1-invoices-billing-suppliers`).
- [x] Taller — *service jobs* (`/workshop/*`, API `/service-jobs`).
- [x] Inventario repuestos (`/admin/parts/*`, API `/parts`).
- [x] Aprovisionamiento usuarios staff (`/admin/users`, `POST /admin/users`).
- [x] Frontend Next.js 16 / React 19 / Tailwind v4 (change archivado `2026-04-27-nextjs-16-react19-migration`).

**Siguientes pasos operativos:** [mvp-next-steps.md](./mvp-next-steps.md) (P0/P1/P2).

## Orden recomendado

`A → B → (C o decisión explícita) → D`. No hace falta “Arnela-paridad” (Tailwind v4, colas pesadas, etc.) para cerrar el MVP; eso entra como deuda o fase posterior.
