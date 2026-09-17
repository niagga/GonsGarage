# Task: Received Invoices Audit & Traceability

## Context
P1 **invoices** = documents **received** by the workshop (`/api/v1/received-invoices`). Distinct from emitted client invoices (`domain.Invoice` / `/api/v1/invoices`) and from `billing_documents` (payroll/irs/other). Spec: `openspec/specs/invoices/spec.md`.

## Scope & Constraints
- **Goal:** Verify received-only domain, staff CRUD, client denial, and delete policy.
- **Constraint:** Do not mix emitted client invoices or payroll/IRS into this domain.
- HTTP names stay as-is (documented map).

## Action Plan
1. [x] Audit domain separation vs emitted `/invoices` and `billing_documents`.
2. [x] Audit staff CRUD + client 403.
3. [x] Audit delete policy (physical vs soft-delete, no broken required refs).
4. [x] Create or update a consolidated traceability document. (`docs/invoices-traceability.md`)

## Findings
- **COMPLIANT:** received-only split; staff CRUD; client 403; min fields; soft-delete + SET NULL.
- **Accounting layout:** staff gate already existed; aligned to `isWorkshopStaff` and locked with `layout.test.tsx` (client → `/dashboard`).
- **Residual:** DELETE already-deleted → 404; stale ports comment.

## Next Step
- Start `suppliers` (last P1 accounting module), or leave residuals.

## Progress
- [x] Planning
- [x] Audit
- [x] Documentation refinement
- [x] Review
- [x] RDD Enabled (globally on)
