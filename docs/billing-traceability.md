# Billing Traceability Document

## Context
This document traces P1 **billing** (workshop-emitted documents) against the catalogue spec, HTTP surfaces, and the dual-model remediation: client-emitted invoices live only on `domain.Invoice`; `billing_documents` is payroll/IRS/other.

**Primary specification:** [openspec/specs/billing/spec.md](../openspec/specs/billing/spec.md)

**Role matrix:** [openspec/specs/mvp-role-access/spec.md](../openspec/specs/mvp-role-access/spec.md)

**Reference archives (history):**
- [openspec/changes/archive/2026-04-20-p1-invoices-billing-suppliers/](../openspec/changes/archive/2026-04-20-p1-invoices-billing-suppliers/) — P1 HTTP/UI
- [openspec/specs/p1-accounting-defer/spec.md](../openspec/specs/p1-accounting-defer/spec.md) — MVP v1 deferral of accounting HTTP (history, not current P1)

**Working ODD task:** [odd/tasks/billing.md](../odd/tasks/billing.md)

## Glossary

| Term | Meaning |
| :--- | :--- |
| **Issued / client-emitted invoice** | Invoice the workshop issues to a customer. Source of truth: `domain.Invoice`. |
| **Received invoice** | Purchase / AP document the workshop receives (suppliers). Spec name `invoices`. HTTP `/api/v1/received-invoices`. |
| **Billing document** | Staff-only emitted record that is **not** a client invoice: `payroll`, `irs`, or `other`. Table `billing_documents`. |
| **Staff trio** | `admin` = `manager` = `employee` on billing, received invoices, and issued-invoice staff CRUD (`RequireWorkshopStaff` / `User.IsEmployee`). |
| **Own-only (client)** | A `client` may list/read (and notes-only patch) only `domain.Invoice` rows whose `customerId` is their user id, via `/api/v1/invoices/me` and `GET/PATCH /api/v1/invoices/:id`. |

HTTP routes are **not** renamed. Spec vocabulary and URL prefixes differ:

| Spec / product name | HTTP | UI | Aggregate |
| :--- | :--- | :--- | :--- |
| Invoices (received) | `/api/v1/received-invoices` | `/accounting/received-invoices` | `ReceivedInvoice` |
| Issued / client-emitted invoices | `/api/v1/invoices` (`GET /me` for client) | `/accounting/issued-invoices`, `/my-invoices` | `Invoice` |
| Billing documents (payroll / IRS / other) | `/api/v1/billing-documents` | `/accounting/billing-documents` | `BillingDocument` |

## Requirement → implementation matrix

| Requirement | Spec rule | Implementation | Status |
| :--- | :--- | :--- | :--- |
| Emitted vs received | Billing MUST NOT mix received (purchase) invoices | Received invoices are a separate group in `backend/cmd/api/main.go` (`/received-invoices`). Billing handler/service never persist `ReceivedInvoice`. | Compliant |
| Client-emitted source of truth | Client invoices are `Invoice`, not `billing_documents.kind=client_invoice` | `BillingDocumentKind` is `payroll` \| `irs` \| `other` (`backend/internal/domain/billing_document.go`). UI kinds in billing-documents pages omit `client_invoice`. Staff emit client invoices via `/api/v1/invoices`. | Compliant (create/update); see residual risk for legacy rows |
| Client “sees own” | Client sees own issued invoices, not payroll/IRS | `GET /api/v1/invoices/me` (`ListMyInvoices`); `GetInvoice` denies other customers. `/billing-documents` uses `RequireWorkshopStaff` — clients MUST NOT. | Compliant |
| Staff CRUD billing | Staff trio MUST; client MUST NOT | `RequireWorkshopStaff` on `/api/v1/billing-documents`; service denies non-employee. UI `/accounting/billing-documents`. | Compliant (RBAC unchanged) |
| Staff CRUD received | Staff trio MUST; client MUST NOT | `RequireWorkshopStaff` on `/api/v1/received-invoices`. | Compliant (RBAC unchanged) |
| Issued invoices roles | Client own-only; staff trio CRUD | Staff `POST/GET/DELETE /invoices` behind `RequireWorkshopStaff`. Client `GET /me`; `GET/PATCH /:id` with ownership in `InvoiceService`. | Compliant (RBAC unchanged) |

## Residual risk

- **Legacy `client_invoice` rows:** There is no CHECK constraint on `billing_documents.kind` and no migration to rewrite or delete historical rows. Existing `kind=client_invoice` rows **MAY** still load on staff list/get. They are **not** creatable or updatable to that kind (`IsValid` / `Validate` reject `client_invoice`). List UI may show the raw kind string for unknown values.
- **Name inversion remains by design:** spec `invoices` ≠ HTTP `/invoices`. Do not rename routes; use the map above.
- **No DB backfill** of legacy billing `client_invoice` into `invoices` is in scope for this remediation.
