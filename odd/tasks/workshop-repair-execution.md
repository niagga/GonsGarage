# Task: Workshop Repair Execution Audit & Traceability

## Context
Workshop **service job** is the visit aggregate (reception, close; OBD/estimate stubs). Spec: `openspec/specs/workshop-repair-execution/spec.md`. Archives: `2026-04-22-workshop-mechanic-vehicle-lifecycle`, `2026-04-20-taller-benchmark-mvp-priorities`.

## Scope & Constraints
- **Goal:** Verify visit aggregate, reception/close, stubs, traces, day listing, repair_ids on read, staff vs client RBAC.
- Historical repairs without a visit MUST remain valid (no forced backfill).

## Action Plan
1. [x] Audit service-job aggregate (car_id required, appointment_id optional, not repairs-only).
2. [x] Audit reception + close (versioned structured payload, reject incomplete, status closed).
3. [x] Audit OBD/estimate stubs (predictable, not fake 200 business state).
4. [x] Audit day listing, repair_ids on GET, and role gate vs `mvp-role-access`.
5. [x] Create or update a consolidated traceability document. (`docs/workshop-repair-execution-traceability.md`)

## Findings
- **COMPLIANT:** visit aggregate, OBD 501 stub, UTC day list, repair_ids, UI detail, list-first modal, staff RBAC.
- **GAP (remediado):** incomplete reception (`{}` / missing `odometer_km`) MUST NOT be 2xx. Handler requires presence; explicit `0` remains 2xx; negative still invalid in the service.

## Next Step
- Closed (2026-11-27).

## Progress
- [x] Planning
- [x] Audit
- [x] Documentation refinement
- [x] Reception GAP remediating/done (`odometer_km` presence required)
- [x] Review
- [x] RDD Enabled (globally on)

### Review outcome (2026-11-27)
Parent review closed. Reception GAP verified in code: missing `odometer_km` returns `400 {"error":"odometer_km is required"}` (`backend/internal/handler/service_job_handler_gin.go:210`) with coverage in `backend/internal/handler/mvp_role_access_test.go`. Route group `/service-jobs` registered at `backend/cmd/api/main.go:631`; `docs/workshop-repair-execution-traceability.md` present (37 lines).
