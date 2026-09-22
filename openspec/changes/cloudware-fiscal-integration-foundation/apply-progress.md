# Apply Progress: Cloudware fiscal integration foundation

## Status
WU1–WU3 remain complete. **WU4 `provider-neutral mock` is complete**: injected `MockScenarioRegistry`, persisted `fiscal_mock_operations`, deterministic labeled PDFs, normalized error classes (`rate_limit`/`ambiguous`/…), and production fail-closed composition guards. No commit or PR was created. Work stayed inside the WU4 boundary (no outbox worker, artifact archive service, HTTP, or UI).

## Completed tasks and persisted checkboxes
- [x] 1.1–1.5 WU1 schema/migration (retained).
- [x] 2.1–2.8 WU2 exact domain model (retained).
- [x] 3.1–3.4 WU3 aggregate persistence (retained).
- [x] 4.1 RED — `mock_provider_test.go` + `fiscal_mock_repository_test.go` covering FT/FR, stability, concurrency, validation, expired connection, transient/rate-limit, ambiguous-then-reconcile, voids, reload, deterministic PDF. <!-- sdd-owner: implementation -->
- [x] 4.2 GREEN — Ports error-contract expansion + persisted mock under `integration/fiscal/mock/*` with registry + memory/Postgres stores. <!-- sdd-owner: implementation -->
- [x] 4.3 TRIANGULATE — Stable refs/PDF checksums for identical key+canonical; change with different input; survive store reload; all provider-foundation mock scenarios exercised. <!-- sdd-owner: implementation -->
- [x] 4.4 REFACTOR — `APP_ENV=production` rejects mock provider/store; `legal` classification forbidden for mock; PDF label `SEM VALIDADE FISCAL — MOCK`; `cmd/api/main.go` composition guards. <!-- sdd-owner: implementation -->

## Files changed (WU4 batch)
| File | Action | What was done |
|------|--------|---------------|
| `backend/internal/core/ports/fiscal_provider.go` | Modified | Added `authorization`/`rate_limit`/`ambiguous` classes + `DefinitiveNonAcceptance` |
| `backend/internal/integration/fiscal/mock/provider.go` | Rewritten | Registry+store driven Issue/Reconcile/Void/Fetch/Verify |
| `backend/internal/integration/fiscal/mock/scenario.go` | Created | Scenario constants + injected `ScenarioRegistry` |
| `backend/internal/integration/fiscal/mock/store.go` | Created | `OperationStore` + memory PutIfAbsent/MarkVoided |
| `backend/internal/integration/fiscal/mock/postgres_store.go` | Created | `fiscal_mock_operations` persistence |
| `backend/internal/integration/fiscal/mock/pdf.go` | Created | Deterministic PDF renderer with mock label |
| `backend/internal/integration/fiscal/mock/guard.go` | Created | Production / legal-classification fail-closed helpers |
| `backend/internal/integration/fiscal/mock/mock_provider_test.go` | Created | Unit coverage for all WU4 scenarios |
| `backend/internal/integration/fiscal/mock/provider_test.go` | Rewritten | Approval-style verify scenarios on new API |
| `backend/tests/integration/fiscal_mock_repository_test.go` | Created | PostgreSQL persist + process reload |
| `backend/cmd/api/main.go` | Modified | Wire mock only when non-production; Postgres store |
| `openspec/.../tasks.md` | Modified | Mark 4.1–4.4 `[x]` |
| `openspec/.../apply-progress.md` | Modified | Cumulative WU1–WU4 progress |

## Verification
- `cd backend && go test ./internal/integration/fiscal/mock/ -count=1` → PASS
- `FISCAL_TEST_DATABASE_URL=postgres://admindb:***@localhost:5432/gonsgarage?sslmode=disable` + `go test ./tests/integration/ -count=1 -run FiscalMock -timeout 60s` → PASS
- `cd backend && go test ./internal/service/fiscal/ -count=1` → PASS (connection + draft/finalization unchanged)
- `cd backend && go test ./internal/core/ports/ -count=1 -run Fiscal` → PASS
- `gofmt` applied to touched Go files
- Race detector: `go test -race` requires CGO; Windows agent reports CGO/gcc unavailable (same limitation as WU2/WU3). Non-race tests pass; CI/Linux should run `-race`.

## Work Unit Evidence (WU4)

| Evidence | Result |
|---|---|
| Focused test command | `go test ./internal/integration/fiscal/mock/ -count=1` → PASS; `go test ./tests/integration/ -count=1 -run FiscalMock` → PASS |
| Runtime harness | PostgreSQL 16 via `FISCAL_TEST_DATABASE_URL` + migration 011 isolated schemas → PASS (`fiscal_mock_operations` row survives provider reload) |
| Rollback boundary | Disable mock selection (`APP_ENV=production` or stop wiring); delete only non-production `fiscal_mock_operations` rows. Revert WU4 mock/ports/main without touching WU5+ |

## TDD Cycle Evidence (WU4)

| Task | Test file/layer | Safety net | RED | GREEN | TRIANGULATE | REFACTOR |
|---|---|---|---|---|---|---|
| 4.1 | `mock_provider_test.go` + `fiscal_mock_repository_test.go` / unit+PG | ✅ existing mock package PASS before rewrite | Failing contracts written first (registry/store/PDF/production) | — | Covered with 4.3 | N/A in RED |
| 4.2 | same | N/A (rewrite) | Ports + mock APIs referenced by RED | Provider/store/PDF/guards implemented; tests PASS | — | Contracts kept provider-neutral |
| 4.3 | same | unit green | Different canonical + reload + FR capability cases | Stable MOCK- refs/PDF SHA; change on different input; PG reload | All foundation mock scenarios | Cleaned void/scenario switches |
| 4.4 | unit + main composition | ✅ mock+service PASS | Production/legal rejection tests | Guards in `guard.go` + `main.go` | production vs development | Composition fail-closed |

## Prior work unit evidence (retained)

### WU3
| Evidence | Result |
|---|---|
| Focused tests | draft/finalization + FiscalRepository PG → PASS |
| Runtime harness | PostgreSQL 16 migration 011 → PASS |
| Rollback | Disable fiscal services; retain schema 011 |

### WU2 / WU1
Retained from prior apply-progress: domain decimal/lifecycle packages and migration 011 with fail-fast schema checks.

## Deviations, budget, and remaining work
- **Budget / size:exception**: Authored WU4 volume is estimated ~1,100–1,400 lines (mock package rewrite + ports + PG integration + main wiring), above the preferred ~600-line manual slice. Completing WU4 coherently required full registry/store/PDF/guards + dual test layers (same pattern as WU3 ~1650). Documented as **size:exception** under parent-authorized manual WU4 slice.
- **Deviation**: Existing `FiscalProvider` port retained (already used by connection service) rather than introducing a separate `FiscalProviderGateway` type from design pseudocode; normalized error classes and mock behavior match design intent.
- **Deviation**: PDF bytes are regenerated from stored canonical/kind when the Postgres row has only `pdf_sha256` (table has no bytea column); checksum must match.
- Scenario selection no longer parses magic operation-key suffixes or metadata (old scaffolding removed); only injected registry keys.
- Race/`gcc`: deferred to CI (same as WU2/WU3).
- Delivery: Parent authorized manual WU4 only; agent created **no commits/PRs**. Native gentle-ai authority bypass honored (OpenSpec file-based apply only).

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
taskProgress: { total: 58, complete: 22, remaining: 36 }
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
    - "WU4 authored ~1100-1400 lines (>600 preferred); size:exception like WU3"
    - "go test -race unavailable locally: CGO/gcc missing on Windows agent host"
    - "Native gentle-ai apply blocked (corrupt_authority); OpenSpec-only apply per parent"
nextRecommended: sdd-verify
isNonAuthoritative: false
```

WU4 finish state is satisfied: non-production mock deterministically covers FT/FR, failures, reconciliation, voiding, and labeled PDFs; production rejects mock. Next unit starts at Phase 5 (outbox worker) only after parent/verify.

## Remaining implementation tasks (verbatim start of next unit)

- [ ] 5.1 RED — Add `backend/tests/integration/fiscal_outbox_test.go` and `backend/internal/service/fiscal/worker_test.go` for `SKIP LOCKED` single claims, lease-token/owner fencing, lease expiry, started-attempt-before-call, stale started mutation becoming unknown, result transaction atomicity, and graceful shutdown. <!-- sdd-owner: implementation -->
