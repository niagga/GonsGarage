# Reporte de situación del MVP (GonsGarage)

**Fecha de referencia:** 27 de noviembre de 2026.
**Alcance:** estado real del MVP v1 contra el código en `main` (`98d7f96`).
**Fuentes de verdad:** [`mvp-solo-checklist.md`](./mvp-solo-checklist.md), [`mvp-minimum-phases.md`](./mvp-minimum-phases.md), [`application-analysis.md`](./application-analysis.md) y las specs en [`openspec/specs/`](../openspec/specs/).

> Este documento reemplaza la versión previa (16-abr-2026, en portugués), que declaraba la Fase C como parcial y la Fase D como pendiente. Ambas afirmaciones quedaron desactualizadas: el CRUD de reparaciones para staff se entregó el 2026-04-19 (checklist ítem 6.1) y la Fase D está cerrada en LAN (checklist Fase 4, 4.1–4.6).

## Dónde estamos

| Fase (ver `mvp-minimum-phases.md`) | Estado | Evidencia |
|------------------------------------|--------|-----------|
| **A — Base técnica** | **Cerrada** | `docker-compose.yml`, `.github/workflows/ci.yml`, `docs/testing-tdd.md` |
| **B — Identidad y datos** | **Cerrada** | JWT + roles, `/auth/me`, coches y citas con permisos, OpenAPI con swag |
| **C — Reparaciones** | **Cerrada** | `GET/POST/PATCH/DELETE /api/v1/repairs…` (`backend/cmd/api/main.go:622-629`) + UI staff en `frontend/src/app/(staff)/repairs/{new,[id],[id]/edit}/page.tsx`; lectura cliente en el detalle del coche |
| **D — Producción** | **Cerrada en LAN** | Stack Docker prod + nginx `:8102`, `deploy/README.md` con smoke, backup, rollback y política HTTPS; deploy **solo manual** (`.github/workflows/deploy.yml`) |

**Checklist operativo:** las 6 fases del [checklist de una persona](./mvp-solo-checklist.md) figuran **hechas** (1 a 6, incluidos 4.1–4.6 y 6.1–6.2).

## Dos ítems abiertos (no bloquean el MVP)

| Ítem | Estado | Nota |
|------|--------|------|
| **HTTPS/TLS** en entorno expuesto | Documentado como excepción para LAN | La LAN actual usa HTTP; criterio de corte en `deploy/README.md` § *HTTPS vs HTTP en LAN*. Pasa a obligatorio al exponer el servidor fuera de la red interna. |
| **CI → servidor automático** | Explícitamente **no priorizado** | El deploy es manual por decisión (checklist 6.2). No es un gap: es alcance descartado. |

## Post-MVP entregado

Funcionalidad ya en `main`, fuera del MVP v1 original y no bloqueante:

| Área | Entrega | Referencia |
|------|---------|------------|
| Contabilidad P1 | `received-invoices`, `billing-documents`, `suppliers`, facturas a cliente (`/api/v1/…`); UI taller `/accounting/*` y cliente `/my-invoices` | `openspec/specs/{invoices,billing,suppliers,p1-accounting-defer}/` |
| Taller | *service jobs* (`/workshop/*`, API `/service-jobs`) | `openspec/specs/workshop-repair-execution/` |
| Inventario de repuestos | `/admin/parts/*`, API `/parts` | `openspec/specs/parts-inventory/` |
| Usuarios staff | Aprovisionamiento + UI de gestión (`/admin/users`, `POST /admin/users`) | `openspec/specs/{staff-user-provisioning,staff-user-management-ui}/` |
| Sistema de UI | Componentes + brand shell | `openspec/specs/{ui-component-system,ui-brand-shell}/` |
| Frontend | Next.js 16.2 / React 19 / Tailwind v4 | `docs/adr/0003-nextjs16-react19-migration-main.md` |
| Facturación fiscal | Integración Cloudware con *dark launch* (WU1–WU11), flags apagadas y producción *fail-closed* | `openspec/changes/cloudware-fiscal-integration-foundation/` |

## Área del cliente

**Cubierto:** registro/login, coches (CRUD), citas (lista + modal), panel con resumen, detalle de coche con historial de servicios, facturas propias en `/my-invoices`, tema oscuro e i18n de UI en **es** (única lengua, por decisión de alcance).

**Gaps conocidos para "MVP cliente cerrado":** notificaciones por e-mail y recuperación de contraseña. Ninguno de los dos está en el alcance congelado de MVP v1 ([checklist, Fase 1](./mvp-solo-checklist.md#decisiones-cerradas-mvp-v1)).

## Deuda abierta (no bloqueante, ordenada por impacto)

1. **Cobertura de specs:** `repairs`, `cars` y `appointments` son el núcleo del negocio y **no tienen spec** en `openspec/specs/`; solo hay cobertura de RBAC en `openspec/specs/mvp-role-access/spec.md`.
2. **Vínculo factura ↔ reparación:** la migración `backend/migrations/012_invoice_repair_link.up.sql` existe sin spec que la rija.
3. **Specs fiscales no promovidas:** viven dentro del change (`cloudware-fiscal-integration-foundation/specs/fiscal-*`), sin contraparte en `openspec/specs/`.
4. **Cierre fiscal en vivo (WU12):** bloqueado por credenciales Cloudware reales; 50/58 tareas, `verify-report.md` con `verdict: fail` por gate de archivo, no por defecto de implementación.

## Próximo paso recomendado

Mantener el MVP como **cerrado** y no reabrirlo por la deuda de specs: esa deuda es documental y se salda con un change de consolidación retroactiva, antes de sumar features nuevas. La decisión pendiente y única es qué hacer con la integración fiscal en vivo (congelarla formalmente como *deferred* o conseguir credenciales y correr WU12).
