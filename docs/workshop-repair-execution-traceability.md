# Workshop Repair Execution Traceability Document

## Context
The **service job** (visita) is the workshop visit aggregate: reception, close, optional OBD/estimate stubs, day listing, and repair links.

**Primary specification:** [openspec/specs/workshop-repair-execution/spec.md](openspec/specs/workshop-repair-execution/spec.md)

**Reference archives:**
- [openspec/changes/archive/2026-04-22-workshop-mechanic-vehicle-lifecycle/](openspec/changes/archive/2026-04-22-workshop-mechanic-vehicle-lifecycle/)
- [openspec/changes/archive/2026-04-20-taller-benchmark-mvp-priorities/](openspec/changes/archive/2026-04-20-taller-benchmark-mvp-priorities/)

## HTTP / UI
- HTTP: `/api/v1/service-jobs` (`RequireWorkshopStaff`)
  - `POST /` create (`car_id`, status `open`)
  - `GET ?opened_on=YYYY-MM-DD` UTC day list
  - `GET /:id` job + reception + handover + `repair_ids`
  - `PUT /:id/reception`, `PUT /:id/handover`
  - `GET /:id/obd` → **501** stub
- UI: `/workshop` (list + Nova visita modal), `/workshop/[id]`, `/workshop/recepcion`

## Traceability Matrix

| Requirement | Spec rule | Audited implementation | Status |
| :--- | :--- | :--- | :--- |
| Visit aggregate | Own aggregate; `car_id` required; `appointment_id` MAY null; no forced repair backfill | `domain.ServiceJob`; `Repair.ServiceJobID` optional | Compliant |
| Reception + close | Structured versioned payload; incomplete MUST NOT 2xx; close → closed; who/when on GET | Child tables + `schema_version`; handover sets `closed`. **`odometer_km` MUST be present** (explicit `0` allowed; missing/`{}` → 400; negative still invalid) | **Compliant** (GAP remediado) |
| OBD / estimate stubs | Predictable; MUST NOT 200 with fake business state | OBD **501**; estimate unexposed (404) | Compliant |
| Day listing | Documented day; empty not 5xx | `opened_on` UTC `[00:00, next)`; UI “Hoje (UTC)” | Compliant |
| `repair_ids` on GET | Zero/one/many; MUST NOT invent work | `repair_ids` `[]` if none | Compliant |
| UI detail | Success readable; error actionable | loading / error + Voltar / status | Compliant |
| List-first create | Modal on list, not jump-only | Dialog on `/workshop` then push detail | Compliant |
| RBAC | Client MUST NOT mutate; staff trio | `RequireWorkshopStaff` + `requireWorkshopUser` | Compliant |

## Residual / GAP
- **Incomplete reception (remediado):** `PUT .../reception` requires `odometer_km` present in the JSON body. Missing field (`{}` or notes-only) is 400; explicit `0` is 2xx; `OdometerKM < 0` remains invalid in the service. Oil/coolant/tires/notes stay optional.
- Estimate has no stub route (404 vs 501). Spec is MAY.
- No HTTP test that a linked `repairs.service_job_id` appears in GET `repair_ids`.
