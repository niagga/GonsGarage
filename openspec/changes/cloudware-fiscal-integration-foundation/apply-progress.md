# Apply Progress: Cloudware fiscal integration foundation

## Status
WU1–WU6 remain complete. **WU7 `additive HTTP API` is complete**: nested fiscalization routes (draft/detail/finalize/retry/reconcile/void), batch summaries registered before `/:id`, PDF streaming, strict decimal-string JSON + body/line limits, status mappings 403/404/409/422/202/503, client own-only projections/artifacts, manager/admin-only legal actions, legacy invoice contract regression, centralized safe response mapping, and swagger updates. No commit or PR was created. Work stayed inside the WU7 boundary (no WU8 UI).

## Completed tasks and persisted checkboxes
- [x] 1.1–1.5 WU1 schema/migration (retained).
- [x] 2.1–2.8 WU2 exact domain model (retained).
- [x] 3.1–3.4 WU3 aggregate persistence (retained).
- [x] 4.1–4.4 WU4 provider-neutral mock (retained).
- [x] 5.1–5.4 WU5 outbox worker (retained).
- [x] 6.1–6.4 WU6 artifact archive (retained).
- [x] 7.1 RED — `fiscal_handler_test.go`, `fiscal_integration_handler_test.go`, `invoice_handler_test.go` for route ordering, strict decimals, body/line limits, 403/404/409/422/202, repeated actions, own-only, manager/admin-only legal actions. <!-- sdd-owner: implementation -->
- [x] 7.2 GREEN — `fiscal_handler.go` DTOs/endpoints; `document_service.go`; register `/invoices/fiscalization-summaries` before `/:id` in `cmd/api/main.go`; service-level auth repeated. <!-- sdd-owner: implementation -->
- [x] 7.3 TRIANGULATE — `p1_accounting_routes_test.go` + invoice handler/service tests: legacy list/create/own/detail/PATCH/delete envelopes, RFC3339, notes-only client patch, draft-vs-frozen delete conflicts. <!-- sdd-owner: implementation -->
- [x] 7.4 REFACTOR — `fiscal_response.go` centralized error/projection sanitization; swagger models + regenerated `docs/swagger.yaml|json|docs.go`; secret/URL omission verified. <!-- sdd-owner: implementation -->

## Files changed (WU7 batch)
| File | Action | What was done |
|------|--------|---------------|
| `backend/internal/core/ports/fiscalization_service.go` | Created | Invoice-scoped FiscalizationService port + safe projection types |
| `backend/internal/service/fiscal/document_service.go` | Created | DocumentService wrapping draft/finalization/artifact with ownership auth |
| `backend/internal/handler/fiscal_handler.go` | Created | Nested fiscal HTTP endpoints + route registration helper |
| `backend/internal/handler/fiscal_response.go` | Created | Centralized error mapping + projection sanitization |
| `backend/internal/handler/fiscal_handler_test.go` | Created | RED/GREEN/triangulation HTTP coverage |
| `backend/internal/handler/invoice_handler_test.go` | Created | Legacy snapshot + protected-history 409 |
| `backend/internal/handler/fiscal_integration_handler_test.go` | Modified | Employee 403 on connection legal actions |
| `backend/internal/handler/p1_accounting_routes_test.go` | Modified | Legacy invoice contract + additive summaries triangulation |
| `backend/internal/handler/invoice_handler.go` | Modified | Map ErrFiscalHistoryProtected → 409 |
| `backend/internal/handler/swagger_models.go` | Modified | Fiscalization swagger DTOs |
| `backend/internal/service/invoice/invoice_service_test.go` | Modified | Frozen vs draft delete protection |
| `backend/internal/domain/fiscal_document.go` | Modified | `legacy_unfiscalized` presentation constant |
| `backend/cmd/api/main.go` | Modified | Wire DocumentService + RegisterInvoiceAndFiscalRoutes |
| `backend/docs/swagger.yaml`, `swagger.json`, `docs.go` | Modified | Regenerated with fiscalization paths (LeftDelim stripped for swag v1.8.12) |
| `openspec/.../tasks.md` | Modified | Mark 7.1–7.4 `[x]` |
| `openspec/.../apply-progress.md` | Modified | Cumulative WU1–WU7 progress |

## Verification
- `cd backend && go test ./internal/handler/ ./internal/service/fiscal/ ./internal/service/invoice/ ./internal/domain/ -count=1` → PASS
- `cd backend && go build ./cmd/api/` → PASS
- `gofmt` applied to touched Go files
- Race detector: `go test -race` requires CGO; Windows agent reports CGO/gcc unavailable (same limitation as WU2–WU6). Non-race tests pass; CI/Linux should run `-race`.

## Work Unit Evidence (WU7)

| Evidence | Result |
|---|---|
| Focused test command | `go test ./internal/handler/ ./internal/service/fiscal/ ./internal/service/invoice/ ./internal/domain/ -count=1` → PASS |
| Runtime harness | N/A — WU7 is HTTP handler/service unit coverage with gin httptest stubs; no new runtime/provider boundary beyond existing fiscal feature flag wiring in `cmd/api` |
| Rollback boundary | Disable `FISCAL_FEATURE_ENABLED` (fiscalHandler nil → only legacy invoice routes); revert WU7 handler/service/port/docs without touching WU1–WU6 persistence/worker |

## TDD Cycle Evidence (WU7)

| Task | Test file/layer | Safety net | RED | GREEN | TRIANGULATE | REFACTOR |
|---|---|---|---|---|---|---|
| 7.1 | `fiscal_handler_test.go` + integration/invoice handler tests / unit+httptest | ✅ `go test ./internal/handler` PASS before changes | Compile-fail refs to NewFiscalHandler / RegisterInvoiceAndFiscalRoutes / MaxFiscal* | — | Covered with 7.3 | N/A in RED |
| 7.2 | same + `document_service.go` | N/A (new) | Ports + handler APIs from RED | Handler + DocumentService + main wiring; tests PASS | — | Route helper + MaxBytesReader |
| 7.3 | `p1_accounting_routes_test.go` + `invoice_service_test.go` | unit green | Legacy envelope/RFC3339/notes-only + frozen delete 409 | Behaviors pass; summaries additive | All listed legacy scenarios | Stub CreateInvoice timestamps |
| 7.4 | `fiscal_response.go` + sanitize test | ✅ handler PASS | Secret/URL redaction assertions | Centralized mapping; swagger regen | Projection omit secrets | Extract fiscal_response.go |

## Prior work unit evidence (retained)

### WU6
| Evidence | Result |
|---|---|
| Focused tests | fiscalartifact + fiscal service + FiscalArtifact PG → PASS |
| Runtime harness | PostgreSQL archive/access-log → PASS |
| Rollback | Disable issuance; retain bytes/metadata |

### WU5 / WU4 / WU3 / WU2 / WU1
Retained: outbox worker; mock provider; draft/finalization repos; domain packages; migration 011.

## Deviations, budget, and remaining work
- **Budget / size:exception**: Authored WU7 volume is estimated ~1,400–2,000 lines including new handler/service/port/tests (plus large swagger regeneration churn). Completing WU7 coherently required HTTP surface + DocumentService + legacy regression + swagger (same pattern as WU3–WU6). Documented as **size:exception** under parent-authorized manual WU7 slice (~600 preferred).
- **Deviation**: Artifact download wiring in production `cmd/api` leaves ArtifactService nil until object/local store is composed for the API process (worker already has it); OpenArtifact returns unavailable until wired — projection/status paths still work.
- **Swagger**: `swag init` (CLI v1.16.4) emitted `LeftDelim`/`RightDelim` incompatible with module `swag v1.8.12`; fields removed post-generation so `go build ./cmd/api` succeeds.
- Race/`gcc`: deferred to CI (same as WU2–WU6).
- Delivery: Parent authorized manual WU7 only; agent created **no commits/PRs**.

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
taskProgress: { total: 58, complete: 34, remaining: 24 }
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
    - "WU7 authored ~1400-2000 lines (>600 preferred); size:exception like WU3–WU6"
    - "go test -race unavailable locally: CGO/gcc missing on Windows agent host"
    - "API process ArtifactService nil until local/object store composed for downloads"
nextRecommended: sdd-verify
isNonAuthoritative: false
```

WU7 finish state is satisfied: additive fiscal HTTP APIs, legacy invoice regression, and safe projections pass unit tests. Next unit starts at Phase 8 (frontend) only after parent/verify.

## Remaining implementation tasks (verbatim start of next unit)

- [ ] 8.1 RED — Add Vitest coverage for fiscalization service client, shared components, and accounting badge integration covering decimal-string payloads, 202→poll, own-only client views, and unchanged legacy invoice columns/actions. <!-- sdd-owner: implementation -->
