# Apply Progress: Cloudware fiscal integration foundation

## Status
WU1 `schema and legacy isolation` remains complete. WU2 `exact domain model` is complete after reconciliation: production domain types were already present on `main`, triangulation gaps for tasks 2.3/2.7 were filled with additional domain tests, and Phase 2 checkboxes 2.1–2.8 are marked `[x]`. No commit or PR was created. Work stayed inside the WU2 boundary (domain package + OpenSpec bookkeeping only).

## Completed tasks and persisted checkboxes
- [x] 1.1 PostgreSQL migration RED coverage. <!-- sdd-owner: implementation -->
- [x] 1.2 Explicit fiscal foundation migration. <!-- sdd-owner: implementation -->
- [x] 1.3 Constraint and immutability triangulation. <!-- sdd-owner: implementation -->
- [x] 1.4 Reusable migration runner and fail-fast schema verification. <!-- sdd-owner: implementation -->
- [x] 1.5 Independent-verification remediation for frozen lines and exhaustive schema checks. <!-- sdd-owner: implementation -->
- [x] 2.1 RED — Decimal and policy domain tests for strict strings, normalization, scales, arithmetic, IVA/exemption, rounding, adjustments, mismatches, and fail-closed policy. <!-- sdd-owner: implementation -->
- [x] 2.2 GREEN — `shopspring/decimal` plus `fiscal_decimal.go` / `fiscal_policy.go` exact calculator requiring approved policy. <!-- sdd-owner: implementation -->
- [x] 2.3 TRIANGULATE — FT/FR, taxable/exempt, boundary precision, over-discount, unsupported currency/tax, ordering, SHA-256 stability. <!-- sdd-owner: implementation -->
- [x] 2.4 REFACTOR — Centralized validation/error codes and canonical serialization; domain package tests + gofmt. <!-- sdd-owner: implementation -->
- [x] 2.5 RED — Document/connection/artifact lifecycle, freeze, supersession, fixation, role matrix, mock/legal tests. <!-- sdd-owner: implementation -->
- [x] 2.6 GREEN — Eleven-state document lifecycle, connection/artifact aggregates, frozen DTO, allowed actions, presentation. <!-- sdd-owner: implementation -->
- [x] 2.7 TRIANGULATE — Employee/client denial, unknown retry prohibition, failed void → issued, source-ref survival, canonical stability. <!-- sdd-owner: implementation -->
- [x] 2.8 REFACTOR — No Cloudware DTOs in domain (provider key value only); invariants beside constructors; race attempted (see note). <!-- sdd-owner: implementation -->

## WU2 reconciliation — already present vs newly implemented

### Already present (production + baseline tests on `main`)
- `backend/go.mod` / `go.sum`: `github.com/shopspring/decimal v1.4.0`
- `backend/internal/domain/fiscal_decimal.go` — strict non-exponent parse, negative-zero normalization, exact arithmetic, rounding modes, string JSON
- `backend/internal/domain/fiscal_policy.go` — approved-policy resolve, immutable config digest, pure `Calculate`, canonical bytes/SHA-256, fail-closed error codes
- `backend/internal/domain/fiscal_document.go` — eleven states, transitions, finalize/freeze, action matrix, supersession, presentation
- `backend/internal/domain/fiscal_connection.go` — connection states, fixation, role/action matrix, snapshot cloning
- `backend/internal/domain/fiscal_artifact.go` — status/classification, mock vs legal, role/action matrix, production serve gate
- Baseline tests in `fiscal_*_test.go` covering core RED/GREEN scenarios for 2.1–2.6

### Newly implemented in this apply batch
- Expanded triangulation tests only (+182 authored lines):
  - `fiscal_decimal_test.go` — additional invalid-string cases
  - `fiscal_policy_test.go` — boundary precision, over-discount, scale overflow, unsupported currency/tax, source-ref survival, canonical independence from external mutation
  - `fiscal_document_test.go` — unknown-state retry prohibition, employee/client denial on privileged actions, void_pending → issued, opaque `cloudware` provider key
- OpenSpec bookkeeping: Phase 2 checkboxes and this progress file

## Files changed (this batch)
| File | Action | What was done |
|------|--------|---------------|
| `backend/internal/domain/fiscal_decimal_test.go` | Modified | Extra invalid decimal string triangulation |
| `backend/internal/domain/fiscal_policy_test.go` | Modified | 2.3/2.7 calculator triangulation |
| `backend/internal/domain/fiscal_document_test.go` | Modified | 2.7 lifecycle triangulation + opaque provider key |
| `openspec/changes/.../tasks.md` | Modified | Mark 2.1–2.8 `[x]` |
| `openspec/changes/.../apply-progress.md` | Modified | Merge WU1+WU2 progress |

## Verification
- Safety net: `cd backend && go test ./internal/domain/... -count=1` — PASS before edits.
- After triangulation: `cd backend && go test ./internal/domain/... -count=1` — PASS.
- Focused: `go test ./internal/domain/ -count=1 -run 'Fiscal|Decimal|Policy|Calculate' -v` — all listed fiscal tests PASS.
- `gofmt` applied to edited test files.
- Race detector: `go test -race` requires CGO + gcc; this Windows environment has no `gcc` (`cgo: C compiler "gcc" not found`). Race verification is deferred to CI/Linux; domain tests without `-race` pass. Documented as environment limitation for task 2.8.
- Runtime harness: N/A for WU2 — pure domain package with no provider/persistence boundary in this unit.

## Work Unit Evidence (WU2)

| Evidence | Result |
|---|---|
| Focused test command | `cd backend && go test ./internal/domain/ -count=1 -run 'Fiscal\|Decimal\|Policy\|Calculate'` → PASS (all fiscal domain cases) |
| Package command | `cd backend && go test ./internal/domain/... -count=1` → PASS |
| Runtime harness | N/A — WU2 is pure domain; no HTTP/worker/provider runtime in this unit |
| Rollback boundary | Revert the three `fiscal_*_test.go` edits and uncheck 2.1–2.8; production domain package already on `main` can be removed later only before WU3 persistence adopts it |

## TDD Cycle Evidence (WU2)

| Task | Test file/layer | Safety net | RED | GREEN | TRIANGULATE | REFACTOR |
|---|---|---|---|---|---|---|
| 2.1 | `fiscal_decimal_test.go`, `fiscal_policy_test.go` / unit | Domain package already green | Pre-existing tests described strict decimals and fail-closed policy | `fiscal_decimal.go` / `fiscal_policy.go` already present and green | Expanded invalid-string cases added this batch | Error codes already centralized |
| 2.2 | same | shopspring already in go.mod | Covered by existing failing-first history on branch precursors | Exact calculator already avoids float64 | Covered via Calculate cases | gofmt + package green |
| 2.3 | `fiscal_policy_test.go` | Existing FT/FR/SHA cases green | New cases for over-discount/scale/currency/tax written against existing API | Existing `Calculate` already returned correct error codes — new tests green without production edits | Boundary, over-discount, unsupported inputs, ordering, SHA stability covered | No production refactor needed |
| 2.4 | package | Domain green | N/A structural | Already implemented | Covered | `go test ./internal/domain/... -count=1` + gofmt |
| 2.5 | `fiscal_document_test.go`, `fiscal_connection_test.go`, `fiscal_artifact_test.go` | Domain green | Pre-existing lifecycle/matrix tests | Aggregates already present | Extended in 2.7 | Provider-neutral naming retained |
| 2.6 | same production files | Domain green | Covered by 2.5 | Eleven states, transitions, frozen DTO, actions present | Covered | No Cloudware DTOs |
| 2.7 | document + policy tests | Domain green | New tests for retry prohibition, failed void→issued, source refs, denial | Existing transition map/action matrix satisfied tests | All 2.7 behaviors covered | No production change |
| 2.8 | domain package | Domain green | Opaque `cloudware` key test | Provider key is opaque string only | No Cloudware types in domain package (grep) | Race blocked by missing gcc locally; CI expected |

## Deviations, budget, and remaining work
- Deviation: Strict TDD RED→GREEN for production files was reconciled rather than rewritten because production domain code and baseline tests already existed on `main`. This batch added missing triangulation only.
- Deviation: Task 2.8 race detector could not run locally (no gcc/CGO). Non-race domain tests pass; parent/CI should run `-race` on Linux.
- Budget: This batch is +182 authored lines in three test files plus OpenSpec bookkeeping — well under the 600-line WU preference. No WU3 files were touched.
- Delivery: Parent authorized manual WU2 slice; no commits/PRs by agent.

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
taskProgress: { total: 53, complete: 13, remaining: 40 }
deferredParentActions: { total: 5, complete: 1, remaining: 4 }
taskArtifactErrors: []
applyState: ready
dependencies: { apply: ready, verify: blocked, sync: blocked, archive: blocked }
actionContext:
  mode: repo-local
  workspaceRoot: D:/Repos/GonsGarage
  allowedEditRoots:
    - backend/internal/domain/
    - openspec/changes/cloudware-fiscal-integration-foundation/apply-progress.md
    - openspec/changes/cloudware-fiscal-integration-foundation/tasks.md
  warnings:
    - "go test -race unavailable locally: gcc/CGO missing on Windows agent host"
nextRecommended: sdd-verify
isNonAuthoritative: false
```

The broader change remains `applyState: ready` for WU3+. WU2 finish state is satisfied: decimal, calculator, lifecycle, immutability, and action-matrix domain tests pass without provider dependencies. Rollback remains the isolated domain package (already on main) plus this batch's test/bookkeeping delta.

## Remaining implementation tasks (verbatim start of next unit)

- [ ] 3.1 RED — Define fakes and failing service tests under `backend/internal/service/fiscal/*_test.go` plus PostgreSQL tests under `backend/tests/integration/fiscal_repository_test.go` for employee draft CRUD, client denial, optimistic versions, one current intent, atomic finalize/outbox rollback, repeated/concurrent finalization, and protected invoice deletion. <!-- sdd-owner: implementation -->
- [ ] 3.2 GREEN — Add provider-neutral ports in `backend/internal/core/ports/fiscal_repository.go`, `fiscal_provider.go`, `fiscal_artifact_store.go`, and `credential_cipher.go`; implement aggregate/config/action queries in `backend/internal/repository/postgres/fiscal_repository.go` and draft/finalization commands in `backend/internal/service/fiscal/draft_service.go` and `finalization_service.go`. <!-- sdd-owner: implementation -->
- [ ] 3.3 TRIANGULATE — Add PostgreSQL race tests proving one frozen intent and one initial issue event, complete rollback when event insertion fails, retry/reconcile/void sequence numbering, same provider/connection/operation key reuse, and no action from unknown states except reconciliation. <!-- sdd-owner: implementation -->
- [ ] 3.4 REFACTOR — Add narrow `FiscalProtectionReader` and eligibility persistence through `backend/internal/core/ports/repositories.go`, `backend/internal/domain/invoice.go`, `backend/internal/repository/postgres/invoice_repository.go`, and `backend/internal/service/invoice/invoice_service.go`; retain the legacy JSON DTO and verify existing invoice service/repository tests unchanged. <!-- sdd-owner: implementation -->

## WU1 verification evidence (retained)

WU1 PostgreSQL 16 migration/schema work remains complete as previously recorded. No WU1 files were modified in the WU2 batch.
