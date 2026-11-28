# Archive report — repairs-crud-staff

**Archived**: 2026-11-27
**Destination**: `openspec/changes/archive/2026-08-27-repairs-crud-staff/`
**Reason for archiving**: the change describes work that was already delivered months before the proposal was written. It was never executed as an SDD change and cannot be: its 12 tasks remain `[ ]` and the file paths they name no longer exist.

## Verification gate

**No `verify-report.md` exists and none ever did.** This change has no `design.md`, no `apply-progress.md`, no `specs/` and no TDD evidence. The verdict here is not a verification pass — it is an **evidence-based closure**: the described capability is present in `main` and was closed through the MVP checklist instead of through this change.

## Delivery evidence

The feature exists and is exercised in production-shaped paths:

| Layer | Evidence |
|-------|----------|
| API routes | `backend/cmd/api/main.go:622-629` — `GET /repairs/car/:carId`, `POST /repairs`, `GET /repairs/:id`, `PATCH /repairs/:id`, `DELETE /repairs/:id` |
| Handlers | `backend/internal/handler/repair_handler_gin.go` — `GinCreateRepair:137`, `GinGetRepair:204`, `GinPatchRepair:251`, `GinDeleteRepair:339` |
| Service + tests | `backend/internal/service/repair/repair_service.go`, `repair_service_test.go` |
| Swagger | `backend/docs/swagger.json:1766` (`/api/v1/repairs`), `:1829` (`/api/v1/repairs/car/{carId}`) |
| Role gate | `backend/internal/handler/mvp_role_access_test.go:641` — `POST /repairs` registered under the staff RBAC matrix |
| Frontend | `frontend/src/app/(staff)/repairs/new/page.tsx`, `.../[id]/page.tsx`, `.../[id]/edit/page.tsx` |
| Process closure | `docs/mvp-solo-checklist.md` Fase 6, ítem **6.1 — Hecho (2026-04-19)** |

## Why the task list stayed at 0/12

This proposal was written on 2026-08-27, after the capability had already shipped on 2026-04-19. The tasks therefore never had a chance to be applied, and their references are stale against the current layout:

| `tasks.md` / `proposal.md` says | Actual location in the repo |
|---------------------------------|-----------------------------|
| `backend/internal/core/services/repair.go` | `backend/internal/service/repair/repair_service.go` |
| `backend/internal/handler/repair.go` | `backend/internal/handler/repair_handler_gin.go` |
| `go test ./internal/core/services/repair_test.go` | `backend/internal/service/repair/repair_service_test.go` |
| `frontend/src/app/(staff)/repairs/...` *(new or existing)* | Exists; all three pages present |
| `PATCH /api/v1/repairs/:id` | Delivered as `PATCH`, handler `GinPatchRepair` |

## Archive bundle

- `proposal.md` (unchanged, kept as historical record)
- `tasks.md` (unchanged, kept with its 12 open boxes to preserve the audit trail of a proposal that was never executed)
- `archive-report.md` (this file)

## Consequence

**No spec exists for this domain.** `openspec/specs/` has no `repairs/` entry, so the delivered CRUD is governed only by `docs/mvp-solo-checklist.md` ítem 6.1, `docs/application-analysis.md` and `openspec/specs/mvp-role-access/`. That gap is recorded in `docs/mvp-status-report.md` as open documentation debt and is not resolved by this archive.
