# Apply Progress: Cloudware fiscal integration foundation

## Status
WU1–WU10 remain complete. **WU11 `operations and rollout` is complete**: independent fiscal switches (feature/finalization/worker/provider) default off; production rejects mock/local/default secrets/missing schema; `/ready` isolates provider health; worker readiness gates; shared artifact composition wired into API (closes prior PDF ArtifactService nil PARTIAL for composition); CI PostgreSQL 16 + migration smoke + race tests + api/worker/migrate builds; runbook + traceability docs. Finalization and Cloudware mutations remain off. No live Cloudware. No commit or PR. Stopped before WU12.

## Completed tasks and persisted checkboxes
- [x] 1.1–1.5 WU1 schema/migration (retained).
- [x] 2.1–2.8 WU2 exact domain model (retained).
- [x] 3.1–3.4 WU3 aggregate persistence (retained).
- [x] 4.1–4.4 WU4 provider-neutral mock (retained).
- [x] 5.1–5.4 WU5 outbox worker (retained).
- [x] 6.1–6.4 WU6 artifact archive (retained).
- [x] 7.1–7.4 WU7 additive HTTP API (retained).
- [x] 8.1–8.4 WU8 staff fiscal UI (retained).
- [x] 9.1–9.4 WU9 client/admin UI (retained).
- [x] 10.1–10.4 WU10 Cloudware security shell (retained).
- [x] 11.1 RED — configuration/readiness tests (cmd/api, cmd/fiscal-worker, service/fiscal). <!-- sdd-owner: implementation -->
- [x] 11.2 GREEN — API/worker/migrate/compose/env wiring; allowlisted logs/metrics; dependency status; graceful worker shutdown. <!-- sdd-owner: implementation -->
- [x] 11.3 TRIANGULATE — CI PostgreSQL 16, migration smoke, fiscal matrix tests, builds. <!-- sdd-owner: implementation -->
- [x] 11.4 REFACTOR — runbook + traceability docs linked to all four specs. <!-- sdd-owner: implementation -->
- [x] 13.1 Parent delivery decision (retained).

## Files changed (WU11 batch)
| File | Action | What was done |
|------|--------|---------------|
| `backend/internal/service/fiscal/runtime_config.go` | Created | Independent switches, production validation, core/worker readiness, dependency status |
| `backend/internal/service/fiscal/runtime_config_test.go` | Created | RED/GREEN readiness and production fail-closed tests |
| `backend/internal/service/fiscal/runtime_config_matrix_test.go` | Created | Feature-off, Cloudware fail-closed, outage isolation, artifact-recovery matrix |
| `backend/internal/service/fiscal/observability.go` | Created | Allowlisted fiscal log fields + metrics snapshot |
| `backend/internal/service/fiscal/observability_test.go` | Created | PII/secret exclusion assertions |
| `backend/internal/service/fiscal/compose_artifacts.go` | Created | Shared API/worker artifact composition + provider key resolve |
| `backend/internal/service/fiscal/finalization_service.go` | Modified | Default-off `WithEnabled` gate for finalize/retry/void |
| `backend/internal/service/fiscal/document_service.go` | Modified | Projection `artifactId`; clear on client draft hide |
| `backend/internal/core/ports/fiscalization_service.go` | Modified | `ArtifactID` on projection |
| `backend/cmd/api/fiscal_runtime.go` | Created | `/ready` isolation helpers + production composition gate |
| `backend/cmd/api/fiscal_runtime_test.go` | Created | Provider-outage `/ready` + dependency-status tests |
| `backend/cmd/api/main.go` | Modified | RuntimeConfig wiring, ArtifactService composition, dependency route |
| `backend/cmd/fiscal-worker/main.go` | Modified | Startup gates, shared compose, allowlisted start log |
| `backend/cmd/fiscal-worker/readiness.go` | Created | Worker startup validation |
| `backend/cmd/fiscal-worker/readiness_test.go` | Created | Worker enable/schema/DB/mock production tests |
| `backend/Dockerfile.migrate` | Created | One-shot migrate image |
| `backend/Dockerfile.fiscal-worker` | Created | Worker image |
| `docker-compose.yml` | Modified | Fiscal profile: migrate + worker |
| `docker-compose.prod.yml` | Modified | Fiscal env + migrate/worker profiles; defaults off |
| `.env.prod.example` | Modified | Fiscal dark-launch env documentation |
| `.github/workflows/ci.yml` | Modified | Postgres 16, migration smoke, race, builds |
| `deploy/README.md` | Modified | Link fiscal dark-launch section |
| `docs/fiscal-integration-runbook.md` | Created | Ops runbook |
| `docs/fiscal-integration-traceability.md` | Created | Four-spec scenario matrix |
| `openspec/.../tasks.md` | Modified | Mark 11.1–11.4 `[x]` |
| `openspec/.../apply-progress.md` | Modified | Cumulative WU1–WU11 progress |

## Verification
- Focused: `go test ./internal/service/fiscal/ ./cmd/api/ ./cmd/fiscal-worker/ ./internal/handler/ ./internal/core/ports/ -count=1` → **PASS**
- Config matrix filter → **PASS**
- `go build ./cmd/api ./cmd/fiscal-worker ./cmd/migrate` → **PASS**
- `go test -race` → **N/A** on this Windows agent (requires cgo/`CGO_ENABLED=1` + gcc); CI ubuntu job runs `-race`

## Work Unit Evidence (WU11)

| Evidence | Result |
|---|---|
| Focused test command | `go test ./internal/service/fiscal/ ./cmd/api/ ./cmd/fiscal-worker/ -count=1 -run "TestRuntimeConfig_|TestCoreAPIReady_|TestWorker|TestReadyHandler_|TestFiscalDependency|TestValidate|TestResolveProvider|TestArtifactRecovery|TestFiscalLogEvent_|TestFiscalMetricsSnapshot_|TestFinalizationDisabled"` → **PASS** |
| Runtime harness | Compose/CI config + `/ready` httptest isolation; live Cloudware mutations remain blocked (WU12 / parent 13.2–13.4); PostgreSQL integration suite runs in CI via `FISCAL_TEST_DATABASE_URL` |
| Rollback boundary | Revert WU11 runtime/observability/compose/CI/docs and disable fiscal env switches; WU1–WU10 domain/provider/UI remain; turn off finalization/worker without deleting evidence |

## TDD Cycle Evidence (WU11)

| Task | Test file/layer | Safety net | RED | GREEN | TRIANGULATE | REFACTOR |
|---|---|---|---|---|---|---|
| 11.1 | `runtime_config_test.go`, `observability_test.go`, `cmd/api/fiscal_runtime_test.go`, `cmd/fiscal-worker/readiness_test.go` / Unit+HTTP | N/A (new config surface) | ✅ Written (undefined types) | — | Covered in 11.3 | N/A in RED |
| 11.2 | same + production modules | Prior fiscal package green | From 11.1 | ✅ packages PASS + builds | — | Composition helpers extracted |
| 11.3 | `runtime_config_matrix_test.go` + CI | ✅ focused green | Matrix cases | Behaviors PASS | ✅ feature-off/mock/Cloudware/outage/artifact | CI yaml |
| 11.4 | Docs (runbook/traceability) | Approval via existing green tests | Docs only | N/A code | Spec links | ✅ Ops clarity |

## Prior work unit evidence (retained)

### WU10
| Evidence | Result |
|---|---|
| Focused tests | crypto + fiscal service + cloudware + handler PASS |
| Runtime harness | N/A — fake HTTP only |
| Rollback | WU10 crypto/OAuth/cloudware; mock path remains |

### WU9–WU1
Retained: client/admin UI; staff UI; additive HTTP; artifacts; worker; mock; repos; domain; migration 011.

## Deviations, budget, and remaining work
- **Budget / size:exception**: Authored WU11 volume exceeds the preferred ~600-line budget (runtime config + observability + cmd wiring + compose/CI/Dockerfiles + docs + tests). Completing WU11 coherently required the full ops slice (same pattern as WU3–WU10). Documented as **size:exception** under parent-authorized manual WU11 slice.
- **API ArtifactService**: Composed for non-production local backend when feature enabled; production object bytes remain readiness-only until WU12/cloud SDK — projection now includes `artifactId` when metadata exists.
- **Live Cloudware**: Explicitly out of scope. Mutations stay disabled; WU12 blocked on parent 13.2–13.4.
- **Race detector**: N/A without gcc/cgo on this agent host; CI with cgo remains authoritative.
- Delivery: Parent authorized manual WU11 only; agent created **no commits/PRs**. Stopped before WU12.

## Deferred parent lifecycle actions
- [ ] 13.2 Production fiscal-policy, issuer/series, retention, storage, backup, access-log, and legal-void decisions. <!-- sdd-owner: parent -->
- [ ] 13.3 Approved Cloudware validation account, credentials, keyring, and license evidence. <!-- sdd-owner: parent -->
- [ ] 13.4 Approval of every applicable Cloudware enablement gate. <!-- sdd-owner: parent -->
- [ ] 13.5 Ordinary repository review after each work unit and final cross-spec traceability. <!-- sdd-owner: parent -->

## Structured status and boundary
```yaml
schemaName: spec-driven
changeName: cloudware-fiscal-integration-foundation
artifactStore: openspec
planningHome: { root: openspec, changesDir: openspec/changes }
changeRoot: openspec/changes/cloudware-fiscal-integration-foundation
artifactPaths:
  proposal: [openspec/changes/cloudware-fiscal-integration-foundation/proposal.md]
  specs: [openspec/changes/cloudware-fiscal-integration-foundation/specs/*/spec.md]
  design: [openspec/changes/cloudware-fiscal-integration-foundation/design.md]
  tasks: [openspec/changes/cloudware-fiscal-integration-foundation/tasks.md]
  applyProgress: [openspec/changes/cloudware-fiscal-integration-foundation/apply-progress.md]
  verifyReport: [openspec/changes/cloudware-fiscal-integration-foundation/verify-report.md]
  syncReport: [openspec/changes/cloudware-fiscal-integration-foundation/sync-report.md]
contextFiles:
  proposal: [openspec/changes/cloudware-fiscal-integration-foundation/proposal.md]
  specs: [openspec/changes/cloudware-fiscal-integration-foundation/specs/fiscal-documents/spec.md, openspec/changes/cloudware-fiscal-integration-foundation/specs/fiscal-provider-foundation/spec.md, openspec/changes/cloudware-fiscal-integration-foundation/specs/fiscal-artifacts/spec.md, openspec/changes/cloudware-fiscal-integration-foundation/specs/cloudware-fiscal-enablement/spec.md]
  design: [openspec/changes/cloudware-fiscal-integration-foundation/design.md]
  tasks: [openspec/changes/cloudware-fiscal-integration-foundation/tasks.md]
  applyProgress: [openspec/changes/cloudware-fiscal-integration-foundation/apply-progress.md]
  verifyReport: [openspec/changes/cloudware-fiscal-integration-foundation/verify-report.md]
  syncReport: []
artifacts: { proposal: done, specs: done, design: done, tasks: done, applyProgress: done, verifyReport: present, syncReport: missing }
taskProgress: { total: 58, complete: 50, remaining: 8 }
deferredParentActions: { total: 5, complete: 1, remaining: 4 }
taskArtifactErrors: []
applyState: ready
dependencies: { apply: ready, verify: ready, sync: blocked, archive: blocked }
actionContext:
  mode: repo-local
  workspaceRoot: D:/Repos/GonsGarage
  allowedEditRoots:
    - D:/Repos/GonsGarage
  warnings:
    - "WU11 authored above ~600 preferred lines; size:exception like WU3–WU10"
    - "Live Cloudware mutations remain fail-closed until WU12 / parent 13.2–13.4"
    - "go test -race N/A without gcc/cgo on this agent host; CI runs -race"
nextRecommended: sdd-verify
isNonAuthoritative: false
```

WU11 finish state is satisfied: dark-launch switches, production fail-closed composition, `/ready` isolation, worker readiness, CI Postgres 16, runbooks. Next: `sdd-verify`. WU12 remains **BLOCKED** pending parent 13.2–13.4.

## Remaining implementation tasks (verbatim start of next unit)

- [ ] 12.1 … (WU12 BLOCKED — do not start without parent 13.2–13.4)
