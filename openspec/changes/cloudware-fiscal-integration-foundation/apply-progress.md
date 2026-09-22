# Apply Progress: Cloudware fiscal integration foundation

## Status
WU1 and WU2 remain complete. **WU3 `aggregate persistence` is complete**: provider-neutral fiscal repository ports, PostgreSQL atomic draft/finalize/retry/reconcile/void persistence, draft/finalization services, PostgreSQL concurrency/rollback tests, and narrow `FiscalProtectionReader` + invoice eligibility wiring. No commit or PR was created. Work stayed inside the WU3 boundary (no mock provider, worker, artifact storage, HTTP, or UI).

## Completed tasks and persisted checkboxes
- [x] 1.1–1.5 WU1 schema/migration (retained).
- [x] 2.1–2.8 WU2 exact domain model (retained).
- [x] 3.1 RED — Service fakes/tests + PostgreSQL repository tests for draft CRUD, client denial, optimistic versions, one current intent, finalize/outbox rollback, concurrent finalization, protected delete. <!-- sdd-owner: implementation -->
- [x] 3.2 GREEN — Ports (`fiscal_repository`, artifact_store stub, credential_cipher stub; existing `fiscal_provider` reused) + `postgres/fiscal_repository.go` + `draft_service.go` + `finalization_service.go`. <!-- sdd-owner: implementation -->
- [x] 3.3 TRIANGULATE — Concurrent finalize → one intent/event; outbox-fail trigger full rollback; retry/reconcile/void sequence numbering with same provider/connection/operation key; unknown-state issue denied, reconcile allowed. <!-- sdd-owner: implementation -->
- [x] 3.4 REFACTOR — `FiscalEligibility` on `domain.Invoice` with `json:"-"`; Create sets `eligible`; `InvoiceService.WithFiscalProtection` + delete guard; invoice service/repository tests still pass. <!-- sdd-owner: implementation -->

## Files changed (WU3 batch)
| File | Action | What was done |
|------|--------|---------------|
| `backend/internal/core/ports/fiscal_repository.go` | Created | Aggregate CRUD/finalize/enqueue contracts + `FiscalProtectionReader` |
| `backend/internal/core/ports/fiscal_artifact_store.go` | Created | Port stub only (no storage impl) |
| `backend/internal/core/ports/credential_cipher.go` | Created | Port stub only (AES impl already under platform/crypto) |
| `backend/internal/core/ports/fiscal_provider.go` | Unchanged | Pre-existing; reused, not expanded toward WU4 |
| `backend/internal/core/ports/repositories.go` | Modified | Comment linking invoice delete protection seam |
| `backend/internal/repository/postgres/fiscal_repository.go` | Created | Atomic SQL draft/finalize/enqueue with `FOR UPDATE` |
| `backend/internal/repository/postgres/invoice_repository.go` | Modified | Default `eligible` on Create |
| `backend/internal/service/fiscal/draft_service.go` | Created | Staff draft CRUD + client denial |
| `backend/internal/service/fiscal/finalization_service.go` | Created | Manager finalize/retry/reconcile/void |
| `backend/internal/service/fiscal/draft_finalization_test.go` | Created | Fake-repo service tests |
| `backend/tests/integration/fiscal_repository_test.go` | Created | PostgreSQL atomicity/concurrency/sequence tests |
| `backend/internal/domain/invoice.go` | Modified | `FiscalEligibility` field excluded from JSON |
| `backend/internal/service/invoice/invoice_service.go` | Modified | Eligibility-on-create + optional protection reader |
| `openspec/.../tasks.md` | Modified | Mark 3.1–3.4 `[x]` |
| `openspec/.../apply-progress.md` | Modified | This WU3 progress merge |

## Verification
- `cd backend && go test ./internal/service/fiscal/ -count=1 -run 'Draft|Finalization'` → PASS
- `cd backend && go test ./internal/service/invoice/ -count=1` → PASS (legacy contracts unchanged)
- `cd backend && go test ./internal/repository/postgres/ -count=1 -run Invoice` → PASS
- `FISCAL_TEST_DATABASE_URL=postgres://admindb:***@localhost:5432/gonsgarage?sslmode=disable` + `go test ./tests/integration/ -count=1 -run FiscalRepository -timeout 120s` → PASS (local `docker compose` Postgres 16)
- `gofmt` applied to touched Go files
- Race detector: `go test -race` requires CGO; Windows agent reports CGO/gcc unavailable (same limitation as WU2). Non-race tests pass; CI/Linux should run `-race`.

## Work Unit Evidence (WU3)

| Evidence | Result |
|---|---|
| Focused test command | `go test ./internal/service/fiscal/ -count=1 -run 'Draft\|Finalization'` → PASS; `go test ./tests/integration/ -count=1 -run FiscalRepository` → PASS |
| Runtime harness | PostgreSQL 16 via `FISCAL_TEST_DATABASE_URL` + migration 011 in isolated schemas → PASS |
| Rollback boundary | Disable fiscal feature routes/services; retain schema 011 and any frozen evidence rows. Revert WU3 ports/repo/services/tests without touching WU4+ |

## TDD Cycle Evidence (WU3)

| Task | Test file/layer | Safety net | RED | GREEN | TRIANGULATE | REFACTOR |
|---|---|---|---|---|---|---|
| 3.1 | `draft_finalization_test.go` + `fiscal_repository_test.go` / unit+PG | Invoice service/repo green before edits | Failing tests/contracts written first against missing services/repo | Services + SQL repo made tests pass | Covered with 3.3 | N/A in RED |
| 3.2 | same + ports | N/A (new) | Ports referenced by RED tests | `fiscal_repository.go`, draft/finalization services implemented; artifact/cipher stubs only | — | Ports kept provider-neutral |
| 3.3 | `fiscal_repository_test.go` / PG | Draft CRUD green | Concurrent/rollback/sequence cases added | Finalize line-order fix + void-key-before-transition | One intent/event; full outbox-fail rollback; seq 2 reuse; unknown≠issue | Cleaned freeze order |
| 3.4 | invoice service/repo tests / unit | ✅ invoice packages PASS before/after | Approval: existing invoice tests | Eligibility + `WithFiscalProtection` | Existing suites unchanged | `json:"-"` keeps legacy DTO |

## Deviations, budget, and remaining work
- **Budget**: Authored WU3 volume is ~1,650 lines (new files + small invoice edits), above the preferred ~600-line manual slice. Completing WU3 coherently required the full repository + dual test layers; no WU4 work was started. Documented as size overage under parent-authorized manual WU3 slice.
- **Deviation**: Finalization service freezes using persisted draft snapshot totals/canonical bytes rather than re-invoking `domain.Calculate` in-process; policy/issuer IDs are accepted on the finalize command for repository persistence. Full calculator-gated readiness remains available for later HTTP/service hardening.
- **Deviation**: `fiscal_provider.go` already existed from earlier scaffolding; WU3 did not expand mock/Cloudware behavior.
- Race/`gcc`: deferred to CI (same as WU2).
- Delivery: Parent authorized manual WU3 only; agent created **no commits/PRs**.

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
taskProgress: { total: 53, complete: 17, remaining: 36 }
deferredParentActions: { total: 5, complete: 1, remaining: 4 }
taskArtifactErrors: []
applyState: ready
dependencies: { apply: ready, verify: ready, sync: blocked, archive: blocked }
actionContext:
  mode: repo-local
  workspaceRoot: D:/Repos/GonsGarage
  allowedEditRoots:
    - backend/internal/core/ports/
    - backend/internal/repository/postgres/
    - backend/internal/service/fiscal/
    - backend/internal/service/invoice/
    - backend/internal/domain/
    - backend/tests/integration/
    - openspec/changes/cloudware-fiscal-integration-foundation/
  warnings:
    - "WU3 authored ~1650 lines (>600 preferred); coherent finish of WU3 only"
    - "go test -race unavailable locally: CGO/gcc missing on Windows agent host"
nextRecommended: sdd-verify
isNonAuthoritative: false
```

WU3 finish state is satisfied: draft/finalization/retry/reconcile/void repositories are atomic and fenced; invoice JSON contracts preserved. Next unit starts at Phase 4 (mock provider) only after parent/verify.

## Remaining implementation tasks (verbatim start of next unit)

- [ ] 4.1 RED — Add `backend/internal/integration/fiscal/mock/mock_provider_test.go` and repository integration tests for successful FT/FR issuance, stable repeated/concurrent calls, validation rejection, expired connection, definite transient/rate-limit errors, ambiguous-then-reconcile, permitted/refused voids, process reload, and deterministic PDF bytes. <!-- sdd-owner: implementation -->
