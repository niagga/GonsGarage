# Task: Billing Audit & Traceability

## Context
This task tracks the audit and traceability of the P1 **billing** module (documents **emitted** by the workshop). It must stay distinct from **invoices** (documents **received**). Spec: `openspec/specs/billing/spec.md`. Archive: `openspec/changes/archive/2026-04-20-p1-invoices-billing-suppliers/`.

## Scope & Constraints
- **Goal:** Verify emitted-only domain, CRUD + role matrix (`mvp-role-access`), and client visibility (own documents only).
- **Constraint:** Do not mix purchase invoices into this domain.
- **Actionable:** Audit implementation against the two P1 requirements, then publish a PM/auditor traceability document.

## Decisions (locked, remediating)
1. [x] Source of truth for client-emitted invoices is `domain.Invoice` (`/api/v1/invoices`, UI `/accounting/issued-invoices` + `/my-invoices`). Remove `client_invoice` from `BillingDocumentKind`. Payroll/IRS/other stay on `billing_documents`.
2. [x] Do NOT rename HTTP routes. Document the map: spec `invoices` (received) → `/api/v1/received-invoices`; emitted client invoices → `/api/v1/invoices`.
3. [x] Do NOT change RBAC. Document staff trio (admin = manager = employee) vs client on billing/received/issued invoices.

## Action Plan
1. [x] Audit domain separation: billing (emitted) vs invoices (received).
2. [x] Audit CRUD + role × operation matrix (staff vs client; payroll/IRS must stay staff-only).
3. [x] Audit client visibility: a client SHALL see only billing emitted to their account.
4. [x] Create or update a consolidated traceability document.

## Findings
- **COMPLIANT:** received invoices stay out of billing; payroll/IRS are staff-only; client cannot mutate another client's issued invoice (service layer).
- **REMEDIATED — dual model:** `billing_documents.kind` is payroll/irs/other only; client invoices are `domain.Invoice`.
- **DOCUMENTED — name inversion:** HTTP `/invoices` = emitted client billing; spec `invoices` = `/received-invoices` (routes unchanged).
- **REMEDIATED — “Cliente ve o seu”:** clients never see `billing_documents`; own docs are only `/invoices/me`. Spec and matrix updated.
- **DOCUMENTED — mvp-role-access:** issued / received / billing-documents rows match code; staff trio; `p1-accounting-defer` kept as history.

## Next Step
- Closed after independent verification (domain tests + billing-documents vitest PASS). Residual: legacy `client_invoice` rows if any; RDD inspect blocked on `managed_assets_outdated`.

## Progress
- [x] Planning
- [x] Audit
- [x] Documentation refinement
- [x] Review
- [x] RDD Enabled (globally on)
