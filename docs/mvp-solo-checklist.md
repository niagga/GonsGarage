# Checklist MVP — equipo de una persona

**Contexto**

- **Equipo:** una sola persona (vos); las decisiones de alcance las cerrás vos en la **Fase 1** y podés validar el resto con el asistente tarea por tarea.
- **Entorno:** no hay URL “staging” separada de “producción”. Tu **servidor de pruebas** es hoy el único entorno remoto: tratálo como **staging = prod de pruebas** (mismas reglas de secretos y CORS que usarías en prod real).

**Plan maestro (archivado):** [mvp-funcionando-plan proposal](../openspec/changes/archive/2026-04-20-mvp-funcionando-plan/proposal.md) · Fases técnicas históricas: [mvp-minimum-phases.md](./mvp-minimum-phases.md).

**Post‑MVP v1:** [mvp-next-steps.md](./mvp-next-steps.md) (prioridades P0/P1/P2).

**P1 — Facturas recibidas, billing emitido, suppliers (2026-04-20):** specs principais [`openspec/specs/invoices/`](../openspec/specs/invoices/spec.md), [`billing/`](../openspec/specs/billing/spec.md), [`suppliers/`](../openspec/specs/suppliers/spec.md); contexto aplazamento MVP + ponte P1 en [`p1-accounting-defer`](../openspec/specs/p1-accounting-defer/spec.md). Change archivado: [`openspec/changes/archive/2026-04-20-p1-invoices-billing-suppliers/`](../openspec/changes/archive/2026-04-20-p1-invoices-billing-suppliers/). **API** (`/api/v1/…`): `received-invoices` (spec invoices = recibidas, staff); `billing-documents` (payroll/irs/other, staff); `suppliers` (staff); facturas emitidas a clientes: `GET /invoices/me`, `GET` + `PATCH /invoices/:id` (cliente); staff `POST|GET|DELETE /invoices`. **`BillingDocumentKind`:** `payroll` \| `irs` \| `other` — `backend/internal/domain/billing_document.go`. Facturas a cliente = `domain.Invoice`, no un kind de billing. **UI Next.js:** taller `/accounting/*`; cliente `/my-invoices`. Trazabilidad: [`billing-traceability.md`](./billing-traceability.md).

**Verificación por rol (MVP):** matriz y escenarios en [openspec/specs/mvp-role-access/spec.md](../openspec/specs/mvp-role-access/spec.md). Seeds: `go run ./cmd/seed-test-client` (cliente demo) y `go run ./cmd/seed-mvp-users` (admin / manager / employee); credenciales vía variables de entorno (`SEED_*`, ver comentarios en cada `main.go`). **Solo desarrollo** — no ejecutar seeds contra producción.

---

## Cómo avanzar con el asistente (aprobado)

1. Seguí el orden **1 → 2 → 3** (podés adelantar partes de **4** si ya tenés servidor listo).
2. Para cada ítem, en el chat: **“Fase X.Y — aprobado”** o **“Fase X.Y — no, porque …”** (el asistente puede ejecutar o ajustar el checklist en el repo).
3. Marcá vos las casillas `- [x]` en este archivo al cerrar cada ítem, o pedí al asistente que lo haga tras tu “aprobado”.

**IDs de tarea** (para referencia en chat): `1.1`, `1.2`, … como en la tabla abajo.

---

## Fase 1 — Congelar alcance MVP v1

| ID | Tarea | Criterio de hecho |
|----|--------|-------------------|
| 1.1 | Escribir la lista **«Entra»** en MVP v1 | **Hecho** 2026-04-17 — bloque [Entra](#decisiones-cerradas-mvp-v1) abajo |
| 1.2 | Escribir **3 bullets** “fuera de MVP v1” (ej. pagos, i18n, multi-tenant) | **Hecho** 2026-04-17 — tres ítems bajo «Fuera» + nota de propuesta |
| 1.3 | Cerrar explícitamente **repairs staff** (API/UI escritura) vs solo lectura | **Hecho** 2026-04-17 — línea «Repairs staff» abajo (**incluido**) |

### Decisiones cerradas MVP v1

**Fecha cierre Fase 1 (alcance):** 2026-04-17 — *1.1, 1.2 y 1.3 cerradas en checklist.*  
**Reconfirmación 1.1 (chat):** lista Entra con desglose de **invoices** (clientes, proveedores, recibos de sueldos) y **billing** solo clientes.  
**Propuesta 1.2 (chat) — «Fuera»:** tres límites claros compatibles con **CRUD billing** (registro contable / facturación hacia clientes) sin convertir el MVP en un PSP.

- Entra (MVP v1):
  - **CRUD users** — registro / login JWT y perfil (`/auth/me`).
  - **CRUD cars**
  - **CRUD appointments** (citas)
  - **CRUD repairs** (reparaciones)
  - **CRUD invoices** — **diferido post‑MVP v1** (2026-04-17): sin rutas HTTP ni UI en esta entrega; dominio parcial en `internal/service/invoice`; decisión registrada en [`openspec/specs/p1-accounting-defer/spec.md`](../openspec/specs/p1-accounting-defer/spec.md) (change archivado: `openspec/changes/archive/2026-04-19-mvp-gap-roadmap-2026/`).
  - **CRUD billing** — **diferido post‑MVP v1** (2026-04-17), mismo criterio que invoices.
- Fuera (MVP v1):
  - **Cobro electrónico con terceros** — pasarelas (Stripe, Adyen, …), TPV con adquirente, **MB Way / transferencia como canal conectado** (API, webhooks, firma, devoluciones), tokenización de tarjeta y **conciliación bancaria automática**. Sí podés modelar *medio* e *importe* como **datos** en facturas/billing (efectivo, tarjeta, MB Way, transferencia como etiqueta o importe registrado); **no** integración ni cobro online end‑to‑end.
  - **i18n** — una sola lengua de interfaz y de documentación operativa en MVP v1.
  - **Multi‑tenant / varios talleres** — una instancia = un taller (una marca, un conjunto de datos); sin subdominios por cliente corporativo ni aislamiento duro entre empresas.
- Repairs staff (POST/PATCH/DELETE + UI): **incluido** en MVP v1 — coherente con «CRUD repairs» en Entra (1.1 aprobado).

---

## Fase 2 — Coherencia contrato + docs

| ID | Tarea | Criterio de hecho |
|----|--------|-------------------|
| 2.1 | Revisar que **Swagger generado** (`backend/docs/`) refleje rutas reales en `cmd/api` (grupos `/api/v1/...`) | **Hecho** 2026-04-18 — faltaba `GET /api/v1/repairs/car/{carId}` en Swagger; añadidas anotaciones swag en `repair_handler_gin.go` y regenerado `swagger.json` / `swagger.yaml` / `docs.go`. Resto de paths alineados con `setupRoutes`. |
| 2.2 | **Frontend:** buscar llamadas a endpoints que no existan o estén mal prefijados | **Hecho** 2026-04-18 — citas: `PATCH …/cancel|confirm|complete` → `PUT /appointments/{id}` con `{ status }`. Coches: `GET /cars/owner/:id` → `GET /cars?ownerId&limit&offset`. Repairs: `getCarWithRepairs` usa `/repairs/car/{id}`; `api.ts` `getRepairs` exige `carId`; `getProfile`→`/auth/me`; logout sin `POST /auth/logout`. Constantes `API_ENDPOINTS` alineadas. Tests Jest `auth.service` actualizados. |
| 2.3 | Mantener [application-analysis.md](./application-analysis.md) alineado cuando agregues rutas | **Hecho** 2026-04-18 — citas `PUT`+`status`, coches `ownerId` query, nota cliente HTTP. |

---

## Fase 3 — Demo local reproducible (una máquina limpia)

| ID | Tarea | Criterio de hecho |
|----|--------|-------------------|
| 3.1 | Seguir solo [README.md](../README.md) + [development-guide.md](./development-guide.md): compose → API → `pnpm dev` | **Hecho** 2026-04-18 — nueva sección «Demo local (secuencia mínima, checklist 3.1)» en `development-guide.md`; README: `NEXT_PUBLIC_API_URL` sin `/api/v1`, nota Windows `cp`/`copy`, enlace a guía; seed unificado a `go run ./cmd/api`; aclaración Redis+compose y rutas `api-client` vs `lib/api.ts`. `go build ./cmd/api` verificado. |
| 3.2 | Flujo mínimo manual: **login** → **coche** → **cita** → **ver repairs** en detalle coche (si aplica a tu MVP v1) | **Hecho** 2026-04-18 — flujo documentado en `development-guide.md` (subsection checklist 3.2); API verificada: login + `GET /cars` + `GET /appointments` OK; `GET /repairs/car/{id}` fallaba en BD legada sin `technician_id` → `ensureRepairsTechnicianIDColumn` en `cmd/api/main.go` al arrancar. Tras pull, reiniciar API para aplicar. |
| 3.3 | Seed cliente demo (`go run ./cmd/seed-test-client`) documentado si lo usás en demos | **Hecho** 2026-04-18 — `go run ./cmd/seed-test-client` probado desde `backend/` (usuario demo ya existía → log idempotente, exit 0). `development-guide.md`: subsección 3.3 (consulta email antes de crear). `README.md`: tabla demo users (client + nota admin por registro). |

---

## Fase 4 — Servidor de pruebas (= tu staging / prod de pruebas)

| ID | Tarea | Criterio de hecho |
|----|--------|-------------------|
| 4.1 | Definir **URL base** del front y del API (aunque sea IP + puerto) y anotarlas abajo | **Hecho** (repo) — bloque [Entorno remoto](#entorno-remoto-servidor-de-pruebas); mismo origen `http://192.168.1.100:8102` vía nginx. |
| 4.2 | **Secretos:** `JWT_SECRET` fuerte; `DATABASE_URL` / Redis solo en el servidor (no en git) | **Hecho** (repo) — [`.env.prod.example`](../.env.prod.example), `.gitignore` → `.env.prod`; sección **Secretos** en [`deploy/README.md`](./deploy/README.md). Rellenar secretos reales solo en el servidor. |
| 4.3 | **Backend:** build (`go build -o … ./cmd/api`) o imagen Docker; proceso bajo systemd/supervisor o compose en el servidor | **Hecho** (LAN, 2026-04-17) — imagen Docker + compose prod; smoke `curl …/health` y `/ready` OK vía nginx; criterios en [`deploy/README.md`](./deploy/README.md) §**Verificación**. |
| 4.4 | **Frontend:** `pnpm build` con `NEXT_PUBLIC_API_URL` apuntando al API del servidor; servir con lo que elijas (nginx, Node, etc.) | **Hecho** (LAN, 2026-04-17) — build en imagen + nginx; login y navegación (coche, cita, repairs en lectura donde aplique) contra `http://192.168.1.100:8102`; mismo doc §**Verificación**. |
| 4.5 | **HTTPS** si el servidor es expuesto (certificado Let’s Encrypt o TLS detrás de proxy) | **Hecho** (doc) — excepción **HTTP en LAN** y cuándo pasar a TLS: [`deploy/README.md`](./deploy/README.md) §**HTTPS vs HTTP en LAN**. |
| 4.6 | **Rollback:** una página o sección “cómo volver atrás” (versión anterior binario + migración si aplica) | **Hecho** (doc) — §**Rollback** en [`deploy/README.md`](./deploy/README.md). |

### Entorno remoto (servidor de pruebas)

- URL API (misma entrada nginx, ej.): `http://192.168.1.100:8102` — rutas bajo `/api/v1/...`, `/health`, `/swagger/`
- URL frontend: `http://192.168.1.100:8102` — mismo origen; `NEXT_PUBLIC_API_URL` = esa base (sin `/api/v1`)
- Notas (proveedor, SSH, etc.): plantilla Docker en [`deploy/README.md`](./deploy/README.md), [`docker-compose.prod.yml`](../docker-compose.prod.yml), Postgres en host (`DATABASE_URL` → `host.docker.internal` en el compose prod)

---

## Fase 5 — Endurecimiento antes de dar por cerrado el MVP remoto

| ID | Tarea | Criterio de hecho |
|----|--------|-------------------|
| 5.1 | **CORS:** origen permitido = URL real del front de pruebas (no `*` en `release` si podés evitarlo) | **Hecho** (doc) — `CORS_ORIGINS` en [`.env.prod.example`](../.env.prod.example) y criterio en [`deploy/README.md`](./deploy/README.md) §**CORS, `GIN_MODE=release` y `JWT_SECRET`**. |
| 5.2 | **`GIN_MODE=release`** en el servidor de pruebas si simulás prod | **Hecho** (doc) — `GIN_MODE=release` en `.env.prod.example` y checklist servidor en `deploy/README.md` § anterior. |
| 5.3 | **Backup BD:** comando o política mínima (pg_dump semanal, etc.) | **Hecho** (doc) — §**Backup de BD** en [`deploy/README.md`](./deploy/README.md) (`pg_dump` + política mínima). |

---

## Fase 6 — MVP+ (opcional; no bloquea Fase 4–5)

| ID | Tarea | Criterio de hecho |
|----|--------|-------------------|
| 6.1 | API + UI staff para **crear/editar** repairs | **Hecho** (2026-04-19) — Gin `POST|GET|PUT|DELETE /api/v1/repairs…` + panel staff en [`frontend/src/app/cars/[id]/page.tsx`](../frontend/src/app/cars/[id]/page.tsx). |
| 6.2 | Automatizar **deploy** (GitHub Actions → tu servidor) | **Hecho** (doc, 2026-04-19) — política **solo manual** en [`.github/workflows/deploy.yml`](../.github/workflows/deploy.yml); despliegue real vía `deploy/README.md`. **CI→servidor automático:** opcional / **no priorizado** (no bloquea cerrar MVP). |

---

## Estado rápido (rellená al ir cerrando)

| Fase | Estado |
|------|--------|
| 1 Congelar alcance | **hecha** (1.1–1.3 cerradas 2026-04-17) |
| 2 Contrato + docs | **hecha** (2.1–2.3, 2026-04-18) |
| 3 Demo local | **hecha** (3.1–3.3, 2026-04-18) |
| 4 Servidor pruebas | **Hecha** (4.1–4.6; smoke LAN 2026-04-17 en `192.168.1.100:8102`) |
| 5 Endurecimiento | **Hecha** (5.1–5.3 documentadas en `deploy/README.md`, 2026-04-17) |
| 6 MVP+ | **Hecha** (6.1 staff repairs + 6.2 política deploy doc; CI→servidor explícitamente opcional / no priorizado) |
