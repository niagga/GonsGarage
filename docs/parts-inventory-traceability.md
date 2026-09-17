# Parts Inventory Traceability Document

## Context
This document provides a consolidated traceability overview for the `parts-inventory` module. It links the catalogue specification, archived OpenSpec changes, and the audited implementation surfaces.

**Primary specification:** [openspec/specs/parts-inventory/spec.md](openspec/specs/parts-inventory/spec.md)

**Reference archives:**
- [openspec/changes/archive/2026-04-24-spare-parts-inventory/](openspec/changes/archive/2026-04-24-spare-parts-inventory/) — catalogue promotion (stock, UoM, barcode, role gate)
- [openspec/changes/archive/2026-04-27-ui-homogeneity-modal-workshop-parts/](openspec/changes/archive/2026-04-27-ui-homogeneity-modal-workshop-parts/) — list-first create flow (modal / query)

## Glossary
- **Part item:** Inventory spare with stable id, internal code/reference, brand, name, quantity ≥ 0, and a closed UoM set (`unit`, `liter`).
- **Barcode:** Optional unique identifier. Two items MUST NOT share a non-empty barcode.
- **Staff managers:** `admin` and `manager` roles. Only these roles MAY use inventory HTTP routes and the inventory UI.

## Scope Overview
The module covers:
- Role-gated CRUD and reads for spare-parts stock
- Quantity + UoM persistence
- Optional unique barcode and search/pre-fill without implicit persist
- List-first create flow (modal/query); dedicated `/new` is compatibility only

## Traceability Matrix

| Requirement | Spec rule | Audited implementation | Status |
| :--- | :--- | :--- | :--- |
| Role gate | Only `manager`/`admin` get 2xx and see the nav link; `employee`/`client` MUST NOT | Backend: `RequireStaffManagers` in `backend/internal/middleware/role_middleware.go`, applied to `/api/v1/parts` in `backend/cmd/api/main.go`. Frontend: nav in `frontend/src/components/layout/AppShell.tsx`; route guard in `frontend/src/app/admin/parts/layout.tsx` | Compliant |
| CRUD item | Persist id, code, brand, name, quantity ≥ 0, closed UoM; reject negative quantity | Domain: `PartItem.Validate()` in `backend/internal/domain/part_item.go`. Service: `PartService` Create/Update. Persistence: `postgresPartItemRepository` → `part_items` | Compliant |
| Barcode uniqueness | Non-empty barcode MUST be unique | Service: `ensureNoDuplicateBarcode` in `backend/internal/service/part/part_service.go` returns `domain.ErrPartItemDuplicateBarcode` | Compliant |
| UI homogeneity | Create MUST start from the list (modal/query). Dedicated `/new` MAY redirect; MUST NOT be the only path | List: `frontend/src/app/admin/parts/page.tsx` opens `PartCreateModal`. Legacy: `frontend/src/app/admin/parts/new/page.tsx` redirects to `/admin/parts?create=1` | Compliant |

## Related Documentation
- Working ODD task: [odd/tasks/parts-inventory.md](odd/tasks/parts-inventory.md)
- Role catalogue: [openspec/specs/mvp-role-access/](openspec/specs/mvp-role-access/)
