# Received Invoices Traceability Document

## Context
P1 **invoices** in the catalogue are documents **received** by the workshop (accounts payable / purchases). They are not client-emitted invoices and not payroll/IRS billing.

**Primary specification:** [openspec/specs/invoices/spec.md](openspec/specs/invoices/spec.md)

**Reference archive:** [openspec/changes/archive/2026-04-20-p1-invoices-billing-suppliers/](openspec/changes/archive/2026-04-20-p1-invoices-billing-suppliers/)

Related: [docs/billing-traceability.md](./billing-traceability.md)

## HTTP name map

| Catalogue name | Meaning | HTTP | UI |
| :--- | :--- | :--- | :--- |
| `invoices` | Received (this module) | `/api/v1/received-invoices` | `/accounting/received-invoices` |
| (emitted client invoice) | Factura a cliente | `/api/v1/invoices` | `/accounting/issued-invoices`, `/my-invoices` |
| `billing` | Payroll / IRS / other | `/api/v1/billing-documents` | `/accounting/billing-documents` |

Routes are **not** renamed.

## Glossary
- **Received invoice:** Purchase/supplier document the workshop receives. Table `received_invoices`, domain `ReceivedInvoice`.
- **Staff trio:** `admin` = `manager` = `employee` (`RequireWorkshopStaff`).
- **Client:** MUST NOT create or list workshop received invoices.

## Traceability Matrix

| Requirement | Spec rule | Audited implementation | Status |
| :--- | :--- | :--- | :--- |
| Received-only | MUST NOT model emitted client invoices, payroll, or tax output here | Separate aggregate/table/HTTP/UI from `Invoice` and `BillingDocument` | Compliant |
| Staff alta | Staff persist min fields (optional supplier, amount, date, category) | `ReceivedInvoice.Validate()`; CRUD on `/api/v1/received-invoices` | Compliant |
| Client denied | Client MUST get 403 (or equivalent) on create/list | `RequireWorkshopStaff` + `requireEmployee`; `TestP1Accounting_ClientGETReceivedInvoices_403` | Compliant |
| CRUD + delete | CRUD with defined delete policy; no broken required refs | Soft-delete `deleted_at`; list/get/update skip deleted; `supplier_id` `ON DELETE SET NULL`. Repeat DELETE → 404 (safe, not idempotent 204) | Compliant |

## Residuals
- Repeat DELETE returns 404, not 204.
- Stale comment on `BillingDocumentRepository` (“client invoice, etc.”) in `backend/internal/core/ports/repositories.go`.

Staff gate for `/accounting/*` lives in `frontend/src/app/accounting/layout.tsx` (`isWorkshopStaff`; client → `/dashboard`). Locked by `frontend/src/app/accounting/layout.test.tsx`.
