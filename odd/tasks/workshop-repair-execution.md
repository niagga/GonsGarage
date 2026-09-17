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
- Reception GAP remediating/done (presence required). Leave Review for parent.

## Progress
- [x] Planning
- [x] Audit
- [x] Documentation refinement
- [x] Reception GAP remediating/done (`odometer_km` presence required)
- [ ] Review
- [x] RDD Enabled (globally on)
