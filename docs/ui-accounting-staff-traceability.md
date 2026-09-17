# UI Accounting Staff Traceability Document

## Context
Four Contabilidade lists create records in-context via modal. Domain rules stay in P1 specs.

**Primary specification:** [openspec/specs/ui-accounting-staff/spec.md](openspec/specs/ui-accounting-staff/spec.md)

**Archive:** [openspec/changes/archive/2026-04-21-ui-accounting-modal-create-flows/](openspec/changes/archive/2026-04-21-ui-accounting-modal-create-flows/)

## Traceability Matrix

| Requirement | Implementation | Status |
| :--- | :--- | :--- |
| Modal create on four lists | suppliers, received-invoices, billing-documents, issued-invoices — toolbar + empty CTA | Compliant |
| Success closes + reloads | `handleCreated` closes dialog and `load()` | Compliant |
| Cancel does not persist | `onCancel` / dialog close; no create call | Compliant |
| Legacy `*/new` | each `new/page.tsx` → `?create=1` on the list | Compliant |
| pt_PT actions | Novo/Nova…, Criar, Cancelar, Fechar | Compliant |

## Residual
- No dedicated tests for the four `new/page.tsx` files (destination `?create=1` is tested on the lists).
- Issued-invoice status placeholder still English `open`.
