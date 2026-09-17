# Billing (emitido) — especificación P1

> **Promoción:** incorporado ao catálogo principal desde `openspec/changes/archive/2026-04-20-p1-invoices-billing-suppliers/` (2026-04-20).

## Purpose

**Billing** en P1 designa documentos **emitidos polo taller** que **non** son facturas a clientes: **nóminas de soldos**, documentación **IRS** e outras saídas (`payroll` | `irs` | `other` en `billing_documents`). As **facturas emitidas a clientes** son o agregado `Invoice` (`/api/v1/invoices`, UI `/accounting/issued-invoices` e `/my-invoices`). É distinto de **invoices** no spec (só documentos **recibidos**; HTTP `/api/v1/received-invoices`). **The system MUST NOT** mesturar facturas recibidas en billing.

## Requirements

### Requirement: Alcance e non solapamento

**The system SHALL** modelar “billing” (`billing_documents`) só como rexistros **emitidos** polo taller de tipo payroll/IRS/other, staff-only. **The system MUST NOT** mesturar neste dominio as facturas de compra recibidas de provedores (iso é **invoices** no spec, HTTP `/api/v1/received-invoices`). **The system MUST NOT** usar `billing_documents` para facturas a clientes; iso é `domain.Invoice`.

#### Scenario: Rexistro emitido staff-only

- GIVEN un usuario staff
- WHEN crea un rexistro de billing (tipo “nómina” / “IRS” / “other” segundo catálogo P1)
- THEN queda almacenado con tipo/categoría distinguible e trazable, sen mesturar facturas recibidas nin facturas a clientes

#### Scenario: Cliente ve o seu

- GIVEN un cliente autenticado
- WHEN lista ou consulta as facturas emitidas a el
- THEN **SHALL** velas só vía `/api/v1/invoices/me` (agregado `Invoice`), **MUST NOT** acceder a `billing_documents` (payroll/IRS/other son staff-only)

### Requirement: CRUD mínimo

**The system SHALL** expor CRUD para entidades de billing acotadas en deseño, con matriz rol × operación coherente con `mvp-role-access`.

#### Scenario: Operación prohibida

- GIVEN un rol sen permiso para o subtipo (p. ex. cliente editando nómina)
- WHEN intenta mutación
- THEN **MUST** responder denegación acorde ao stack API existente
