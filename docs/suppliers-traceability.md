# Suppliers Traceability Document

## Context
P1 **suppliers** is the workshop vendor master. Received invoices **MAY** reference a supplier; they **MUST** be creatable without one.

**Primary specification:** [openspec/specs/suppliers/spec.md](openspec/specs/suppliers/spec.md)

**Reference archive:** [openspec/changes/archive/2026-04-20-p1-invoices-billing-suppliers/](openspec/changes/archive/2026-04-20-p1-invoices-billing-suppliers/)

Related: [docs/invoices-traceability.md](./invoices-traceability.md), [docs/billing-traceability.md](./billing-traceability.md)

## HTTP / UI
- HTTP: `/api/v1/suppliers` (staff trio via `RequireWorkshopStaff`)
- UI: `/accounting/suppliers` (layout gate `isWorkshopStaff`)

## Traceability Matrix

| Requirement | Spec rule | Audited implementation | Status |
| :--- | :--- | :--- | :--- |
| Staff CRUD | Alta, list, detalle, update, baixa/soft-delete | `SupplierHandler` + `SupplierService` + postgres repo; `deleted_at` soft-delete | Compliant |
| Client denied | Client MUST NOT access supplier CRUD | Middleware 403 + service `requireEmployee`; `TestP1Accounting_ClientGETSuppliers_403`; accounting layout | Compliant |
| Min fields | Commercial id, contact, optional tax id, optional notes | `name` required; `contactEmail`/`contactPhone`; `taxId`; `notes` | Compliant |
| Optional invoice link | Received invoice without supplier MUST save | `ReceivedInvoice.SupplierID *uuid`; UI field optional; POST without `supplierId` 201 | Compliant |

## Residuals
- App delete is **soft-delete**, so SQL `ON DELETE SET NULL` does not run; received invoices may keep `supplier_id` pointing at a hidden row.
- Contact email/phone may be empty.
- List is paginated; no dedicated name-search API.
- Client 403 is asserted on GET list; other methods rely on the same group middleware.
