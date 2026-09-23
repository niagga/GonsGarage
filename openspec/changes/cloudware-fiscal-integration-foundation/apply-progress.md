# Apply Progress: Cloudware fiscal integration foundation

## Status
WU1–WU5 remain complete. **WU6 `artifact archive` is complete**: immutable local private store (create-if-absent + collision), PostgreSQL artifact metadata/access logs, ownership-authorized streaming with checksum compromise refusal, worker `recover_artifact` wired through `ArtifactService.RecoverFromProvider` (FetchArtifact only — never Issue), and production object-store readiness boundary fail-closed until ACL/encryption/retention/backup/access-log pass. No commit or PR was created. Work stayed inside the WU6 boundary (no HTTP handlers / WU7).

## Completed tasks and persisted checkboxes
- [x] 1.1–1.5 WU1 schema/migration (retained).
- [x] 2.1–2.8 WU2 exact domain model (retained).
- [x] 3.1–3.4 WU3 aggregate persistence (retained).
- [x] 4.1–4.4 WU4 provider-neutral mock (retained).
- [x] 5.1–5.4 WU5 outbox worker (retained).
- [x] 6.1 RED — `artifact_service_test.go`, `local_test.go`, `fiscal_artifact_test.go` for create-if-absent, collision, failed archive, recovery without Issue, ownership, access logs, media/signature/size, compromised refusal. <!-- sdd-owner: implementation -->
- [x] 6.2 GREEN — ports `FiscalArtifactStore`/`FiscalArtifactRepository`, `platform/fiscalartifact/local.go`, `postgres/fiscal_artifact_repository.go`, `service/fiscal/artifact_service.go`; worker + `cmd/fiscal-worker` recover path. <!-- sdd-owner: implementation -->
- [x] 6.3 TRIANGULATE — owning vs non-owning clients, staff roles, unavailable/compromised, repeated archive, safe filenames/headers, no URL leakage, read while issuance disabled. <!-- sdd-owner: implementation -->
- [x] 6.4 REFACTOR — `platform/fiscalartifact/object.go` readiness (private ACL, encryption, retention, backup/restore, access logging); production composition fail-closed on local/incomplete readiness. <!-- sdd-owner: implementation -->

## Files changed (WU6 batch)
| File | Action | What was done |
|------|--------|---------------|
| `backend/internal/core/ports/fiscal_artifact_store.go` | Modified | PutImmutable/Open/Stat + repository/access-log ports + typed errors |
| `backend/internal/platform/fiscalartifact/local.go` | Created | Private FS store: streaming SHA-256, PDF checks, create-if-absent |
| `backend/internal/platform/fiscalartifact/local_test.go` | Created | Store RED/GREEN coverage |
| `backend/internal/platform/fiscalartifact/object.go` | Created | Production adapter boundary + readiness fail-closed |
| `backend/internal/platform/fiscalartifact/object_test.go` | Created | Readiness triangulation |
| `backend/internal/repository/postgres/fiscal_artifact_repository.go` | Created | Metadata upsert, status marks, access log, SQL invoice reader |
| `backend/internal/service/fiscal/artifact_service.go` | Created | Archive / RecoverFromProvider / OpenDownload |
| `backend/internal/service/fiscal/artifact_service_test.go` | Created | Ownership, recovery, compromise, issuance-disabled reads |
| `backend/internal/service/fiscal/worker.go` | Modified | Optional ArtifactService on recover_artifact |
| `backend/cmd/fiscal-worker/main.go` | Modified | Compose local store + production readiness gate |
| `backend/tests/integration/fiscal_artifact_test.go` | Created | PostgreSQL archive/access-log/compromised path |
| `openspec/.../tasks.md` | Modified | Mark 6.1–6.4 `[x]` |
| `openspec/.../apply-progress.md` | Modified | Cumulative WU1–WU6 progress |

## Verification
- `cd backend && go test ./internal/platform/fiscalartifact/ ./internal/service/fiscal/ -count=1` → PASS
- `FISCAL_TEST_DATABASE_URL=<redacted from backend/.env>` + `go test ./tests/integration/ -count=1 -run FiscalArtifact -timeout 180s` → PASS
- `cd backend && go build ./cmd/fiscal-worker/` → PASS
- `gofmt` applied to touched Go files
- Race detector: `go test -race` requires CGO; Windows agent reports CGO/gcc unavailable (same limitation as WU2–WU5). Non-race tests pass; CI/Linux should run `-race`.

## Work Unit Evidence (WU6)

| Evidence | Result |
|---|---|
| Focused test command | `go test ./internal/platform/fiscalartifact/ ./internal/service/fiscal/ -count=1` → PASS; `go test ./tests/integration/ -count=1 -run FiscalArtifact` → PASS |
| Runtime harness | PostgreSQL 16 via `FISCAL_TEST_DATABASE_URL` + migration 011 isolated schema → PASS (archive metadata, access log, compromised refusal, issued retained) |
| Rollback boundary | Disable new issuance (`FISCAL_FINALIZATION_ENABLED=false` / stop worker claims); retain local/object bytes and `fiscal_artifacts` reads. Revert WU6 store/service/repo/worker wiring without touching WU7 HTTP |

## TDD Cycle Evidence (WU6)

| Task | Test file/layer | Safety net | RED | GREEN | TRIANGULATE | REFACTOR |
|---|---|---|---|---|---|---|
| 6.1 | `local_test.go` + `artifact_service_test.go` + `fiscal_artifact_test.go` / unit+PG | ✅ `go test ./internal/service/fiscal` PASS before changes | Failing refs to NewLocalStore / ArtifactService / FiscalArtifactRepository (compile fail) | — | Covered with 6.3 | N/A in RED |
| 6.2 | same | N/A (new) | Ports + APIs from RED | Local store + repo + service + worker recover wiring; tests PASS | — | Streaming hash + private keys |
| 6.3 | service + object_test cases | unit green | Staff/client/unavailable/compromised/issuance-disabled/headers | Behaviors pass; no URL/key leakage | All listed authorization scenarios | Safe filename helper |
| 6.4 | `object_test.go` | ✅ fiscalartifact PASS | Production readiness fail-closed | `object.go` AssertProductionIssuanceAllowed + worker compose gate | Local vs complete object readiness | Boundary kept SDK-free |

## Prior work unit evidence (retained)

### WU5
| Evidence | Result |
|---|---|
| Focused tests | fiscal service + FiscalOutbox PG → PASS |
| Runtime harness | PostgreSQL SKIP LOCKED / lease expiry → PASS |
| Rollback | Stop fiscal-worker; leases expire conservatively |

### WU4 / WU3 / WU2 / WU1
Retained: mock provider; draft/finalization repos; domain packages; migration 011 fail-fast schema checks.

## Deviations, budget, and remaining work
- **Budget / size:exception**: Authored WU6 volume is estimated ~1,400–1,800 lines (ports + local/object + repo + service + dual test layers + worker/cmd wiring), above the preferred ~600-line manual slice. Completing WU6 coherently required store + metadata + auth download + recover wiring + readiness (same pattern as WU3–WU5). Documented as **size:exception** under parent-authorized manual WU6 slice.
- **Deviation**: Production `ObjectStore` is a readiness/composition boundary only in WU6 — no cloud SDK PutImmutable yet. Production worker composition refuses local backend and incomplete readiness; full object I/O remains for enablement/later units.
- **Closed WU5 gap**: `recover_artifact` now persists PDF bytes via `ArtifactService.RecoverFromProvider` when ArtifactService is wired (cmd/fiscal-worker does so for non-production local store).
- Race/`gcc`: deferred to CI (same as WU2–WU5).
- Delivery: Parent authorized manual WU6 only; agent created **no commits/PRs**.

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
taskProgress: { total: 58, complete: 30, remaining: 28 }
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
    - "WU6 authored ~1400-1800 lines (>600 preferred); size:exception like WU3–WU5"
    - "go test -race unavailable locally: CGO/gcc missing on Windows agent host"
    - "Production ObjectStore is readiness boundary only; cloud SDK PutImmutable deferred"
nextRecommended: sdd-verify
isNonAuthoritative: false
```

WU6 finish state is satisfied: immutable local test storage, metadata, recovery, integrity checks, and authorized streaming pass unit + PostgreSQL tests. Next unit starts at Phase 7 (HTTP API) only after parent/verify.

## Remaining implementation tasks (verbatim start of next unit)

- [ ] 7.1 RED — Add handler/route tests in `backend/internal/handler/fiscal_handler_test.go`, `fiscal_integration_handler_test.go`, and `invoice_handler_test.go` for static summary-route ordering, strict decimal JSON, body/line limits, 403/404/409/422/202 mappings, repeated actions, own-only projection/artifacts, and manager/admin-only legal actions. <!-- sdd-owner: implementation -->
