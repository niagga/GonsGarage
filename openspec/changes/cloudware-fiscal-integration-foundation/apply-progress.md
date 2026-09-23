# Apply Progress: Cloudware fiscal integration foundation

## Status
WU1–WU4 remain complete. **WU5 `outbox worker` is complete**: PostgreSQL `SKIP LOCKED` claims with lease-token/owner fencing, pre-call started attempts, crash-safe stale→unknown recovery (no blind Issue resubmit), atomic completion, redacted diagnostics, and `cmd/fiscal-worker` dispatch for issue / reconcile-issue / void / reconcile-void / recover-artifact outside HTTP. No commit or PR was created. Work stayed inside the WU5 boundary (no artifact archive service, HTTP, or UI).

## Completed tasks and persisted checkboxes
- [x] 1.1–1.5 WU1 schema/migration (retained).
- [x] 2.1–2.8 WU2 exact domain model (retained).
- [x] 3.1–3.4 WU3 aggregate persistence (retained).
- [x] 4.1–4.4 WU4 provider-neutral mock (retained).
- [x] 5.1 RED — `fiscal_outbox_test.go` + `worker_test.go` for SKIP LOCKED single claims, lease fencing, expiry, started-before-call, stale→unknown, result atomicity, graceful shutdown. <!-- sdd-owner: implementation -->
- [x] 5.2 GREEN — `fiscal_outbox_repository.go`, attempt persistence/redaction, `worker.go`, `cmd/fiscal-worker/main.go` with all five event dispatch paths. <!-- sdd-owner: implementation -->
- [x] 5.3 TRIANGULATE — Crash windows before/during/after provider call and after lease loss; definite non-acceptance → retryable/rejected; ambiguous never auto-resubmit; reconcile uses same frozen provider/key. <!-- sdd-owner: implementation -->
- [x] 5.4 REFACTOR — Separate claim / pre-call / gateway / completion transactions; bounded lease defaults + `RedactSecrets`; focused tests green (race N/A without gcc — same as WU2–WU4). <!-- sdd-owner: implementation -->

## Files changed (WU5 batch)
| File | Action | What was done |
|------|--------|---------------|
| `backend/internal/core/ports/fiscal_outbox.go` | Created | Outbox/attempt ports, claim/begin/recover/complete commands |
| `backend/internal/repository/postgres/fiscal_outbox_repository.go` | Created | SKIP LOCKED claim, lease fencing, started attempts, atomic complete |
| `backend/internal/service/fiscal/worker.go` | Created | Lease worker + classification/redaction + 5 event types |
| `backend/internal/service/fiscal/worker_test.go` | Created | Unit coverage for fencing, crashes, reconcile, void, redaction |
| `backend/tests/integration/fiscal_outbox_test.go` | Created | PostgreSQL SKIP LOCKED, fencing, expiry, atomicity, mock e2e |
| `backend/cmd/fiscal-worker/main.go` | Created | Standalone worker composition + graceful SIGINT/SIGTERM shutdown |
| `openspec/.../tasks.md` | Modified | Mark 5.1–5.4 `[x]` |
| `openspec/.../apply-progress.md` | Modified | Cumulative WU1–WU5 progress |

## Verification
- `cd backend && go test ./internal/service/fiscal/ -count=1` → PASS (includes Worker_* + prior draft/finalization)
- `FISCAL_TEST_DATABASE_URL=postgres://admindb:***@localhost:5432/gonsgarage?sslmode=disable` + `go test ./tests/integration/ -count=1 -run "FiscalOutbox|FiscalRepository|FiscalMock" -timeout 180s` → PASS
- `cd backend && go build ./cmd/fiscal-worker/` → PASS
- `gofmt` applied to touched Go files
- Race detector: `go test -race` requires CGO; Windows agent reports CGO/gcc unavailable (same limitation as WU2–WU4). Non-race tests pass; CI/Linux should run `-race`.

## Work Unit Evidence (WU5)

| Evidence | Result |
|---|---|
| Focused test command | `go test ./internal/service/fiscal/ -count=1` → PASS; `go test ./tests/integration/ -count=1 -run FiscalOutbox` → PASS (5 tests) |
| Runtime harness | PostgreSQL 16 via `FISCAL_TEST_DATABASE_URL` + migration 011 isolated schemas → PASS (SKIP LOCKED 8-way claim, lease expiry recovery, mock e2e issue) |
| Rollback boundary | Stop `fiscal-worker` / set `FISCAL_WORKER_ENABLED=false`; allow leased events to expire into conservative unknown recovery. Revert WU5 outbox/worker/cmd without touching WU6+ |

## TDD Cycle Evidence (WU5)

| Task | Test file/layer | Safety net | RED | GREEN | TRIANGULATE | REFACTOR |
|---|---|---|---|---|---|---|
| 5.1 | `worker_test.go` + `fiscal_outbox_test.go` / unit+PG | ✅ `go test ./internal/service/fiscal` PASS before worker | Failing contracts referencing `NewWorker` / `FiscalOutboxRepository` (compile fail) | — | Covered with 5.3 | N/A in RED |
| 5.2 | same | N/A (new) | Ports + worker APIs from RED | Outbox repo + worker + cmd implemented; tests PASS | — | Tx boundaries kept separate |
| 5.3 | `worker_test.go` crash/classify cases | unit green | Crash before/during/after + permanent reject + inconclusive reconcile + void/artifact | States map correctly; no Issue on stale/ambiguous | All foundation crash/classify scenarios | Cleaned classification helpers |
| 5.4 | unit + integration + build | ✅ fiscal+FiscalOutbox PASS | Lease defaults/`RedactSecrets` tests | `WorkerConfig.Normalize` + constants | Default vs override lease | Documented 4-tx protocol on Worker |

## Prior work unit evidence (retained)

### WU4
| Evidence | Result |
|---|---|
| Focused tests | mock package + FiscalMock PG → PASS |
| Runtime harness | PostgreSQL 16 migration 011 → PASS |
| Rollback | Disable mock selection; remove non-prod mock rows |

### WU3 / WU2 / WU1
Retained: draft/finalization + FiscalRepository PG; domain packages; migration 011 fail-fast schema checks.

## Deviations, budget, and remaining work
- **Budget / size:exception**: Authored WU5 volume is estimated ~1,500 lines (ports + outbox repo + worker + dual test layers + cmd), above the preferred ~600-line manual slice. Completing WU5 coherently required claim/fencing + crash protocol + all five dispatch kinds (same pattern as WU3/WU4). Documented as **size:exception** under parent-authorized manual WU5 slice.
- **Deviation**: Existing scaffolding `dispatcher.go` / `GetNextPending` left in place (pre-WU5 poll-by-document-state helper). Production dispatch path is the new outbox `Worker`; dispatcher is not wired into `cmd/fiscal-worker`.
- **Deviation**: Artifact recovery completes the outbox event and records the attempt but does not yet persist artifact bytes (WU6 owns immutable archive). Legal document state is unchanged on recover_artifact, matching design.
- Race/`gcc`: deferred to CI (same as WU2–WU4).
- Delivery: Parent authorized manual WU5 only; agent created **no commits/PRs**. Native gentle-ai authority bypass honored (OpenSpec file-based apply only).

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
  verifyReport: []
  syncReport: []
artifacts: { proposal: done, specs: done, design: done, tasks: done, applyProgress: done, verifyReport: missing, syncReport: missing }
taskProgress: { total: 58, complete: 26, remaining: 32 }
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
    - "WU5 authored ~1500 lines (>600 preferred); size:exception like WU3/WU4"
    - "go test -race unavailable locally: CGO/gcc missing on Windows agent host"
    - "Native gentle-ai apply blocked (corrupt_authority); OpenSpec-only apply per parent"
nextRecommended: sdd-verify
isNonAuthoritative: false
```

WU5 finish state is satisfied: lease fencing, crash ambiguity, explicit recovery, and graceful worker shutdown pass PostgreSQL + unit tests. Next unit starts at Phase 6 (artifact archive) only after parent/verify.

## Remaining implementation tasks (verbatim start of next unit)

- [ ] 6.1 RED — Add `backend/internal/service/fiscal/artifact_service_test.go`, `backend/internal/platform/fiscalartifact/local_test.go`, and PostgreSQL artifact tests for create-if-absent checksum matching, collision rejection, failed archive retaining `issued`, recovery without `Issue`, ownership authorization, access logs, media/signature/size limits, and compromised-byte refusal. <!-- sdd-owner: implementation -->
