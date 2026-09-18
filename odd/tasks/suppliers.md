# Task: Suppliers Audit & Traceability

## Context
P1 **suppliers** is the workshop vendor master, optionally linked to received invoices. Spec: `openspec/specs/suppliers/spec.md`. Archive: `openspec/changes/archive/2026-04-20-p1-invoices-billing-suppliers/`.

## Scope & Constraints
- **Goal:** Verify staff CRUD, client denial, min fields, optional `supplier_id` on received invoices.
- HTTP names stay as-is.

## Action Plan
1. [x] Audit staff CRUD + client 403.
2. [x] Audit min fields (commercial id, contact, optional tax id, optional notes) and delete policy.
3. [x] Audit optional link from received invoices (`supplier_id` nullable).
4. [x] Create or update a consolidated traceability document. (`docs/suppliers-traceability.md`)

## Findings
- **COMPLIANT:** staff CRUD, client 403, min fields, optional `supplier_id` on received invoices.
- **Residual:** contact fields optional-empty; no name-search API. Soft-delete now nulls `received_invoices.supplier_id` before `suppliers.deleted_at` (honors SQL `ON DELETE SET NULL` intent without hard-delete).

## Next Step
- P1 accounting cluster (defer / parts-inventory / billing / invoices / suppliers / accounting layout) is audited. Pick next module outside this cluster, or remediate residuals.

## Progress
- [x] Planning
- [x] Audit
- [x] Documentation refinement
- [x] Review
- [x] RDD Enabled (globally on)
