# Apply Progress: Cloudware fiscal integration foundation

## Status
WU1 `schema and legacy isolation` is complete after bounded independent-verification remediation within the authoritative repo-local edit roots. The current attempt used the parent-authorized `WU1 verification remediation` slice and 300-line hard cap; no commit or PR was created. The native status CLI was unavailable, so shape-compatible status was produced from the OpenSpec artifacts and injected action context.

## Completed tasks and persisted checkboxes
- [x] 1.1 PostgreSQL migration RED coverage. <!-- sdd-owner: implementation -->
- [x] 1.2 Explicit fiscal foundation migration. <!-- sdd-owner: implementation -->
- [x] 1.3 Constraint and immutability triangulation. <!-- sdd-owner: implementation -->
- [x] 1.4 Reusable migration runner and fail-fast schema verification. <!-- sdd-owner: implementation -->
- [x] 1.5 Independent-verification remediation for frozen lines and exhaustive schema checks. <!-- sdd-owner: implementation -->

## Files changed
`backend/tests/integration/fiscal_migration_test.go`; `backend/migrations/011_fiscal_integration_foundation.up.sql`; `backend/scripts/run_migrations.go` moved to `backend/cmd/migrate/migrations/runner.go`; `backend/cmd/migrate/main.go`; `backend/internal/repository/postgres/fiscal_schema.go`; `backend/cmd/api/main.go`; this progress file; and WU1 checkboxes in `tasks.md`.

## Verification
- RED: `go test ./internal/repository/postgres ./tests/integration -run Fiscal -count=1` failed because the runner was not importable and schema verification symbols did not exist.
- GREEN/TRIANGULATE: PostgreSQL 16 container, `FISCAL_TEST_DATABASE_URL=... go test ./tests/integration -run FiscalMigration -count=1 -v` — 2/2 passed after migration, constraint, uniqueness, immutability, legacy-isolation, and missing-schema cases.
- REFACTOR: `go test ./internal/repository/postgres ./cmd/api ./cmd/migrate/... -run Fiscal -count=1` passed; `go test ./... -count=1` passed against PostgreSQL 16; `go vet ./...` passed.

## TDD Cycle Evidence
| Task | Test file/layer | Safety net | RED | GREEN | TRIANGULATE | REFACTOR |
|---|---|---|---|---|---|---|
| 1.1 | `tests/integration/fiscal_migration_test.go` / PostgreSQL | Repository/schema packages passed; existing integration files were CGO-excluded | Compile failure confirmed | Legacy and zero-work assertions passed | Historical plus direct inserts passed | Shared setup retained |
| 1.2 | same / PostgreSQL | New migration | Test written first | All required table families and pending-only gates passed | Frozen/history constraints passed | Migration remains transaction-wrapper-free |
| 1.3 | same / PostgreSQL | 1.1–1.2 green | Additional cases written | Passed | Current-intent, operation/reference, artifact, history, and deletion paths passed | Assertions use real PostgreSQL behavior |
| 1.4 | same / PostgreSQL | Existing API/repository tests passed | Missing runner/schema symbols failed | Runner and validator passed | Missing versus complete schema paths passed | Full backend tests and vet passed |

## Deviations, budget, and remaining work
Deviation: the invoice-delete trigger removes draft-only fiscal rows itself, preserving legacy deletion before WU3 adds its service hook; every non-draft row remains protected. PostgreSQL-specific acceptance was run on PostgreSQL 16, not inferred from SQLite. Rename-aware implementation delta is 550 changed lines (OpenSpec bookkeeping excluded), below the 600-line cap. No WU1 task remains unchecked; WU2 and later implementation rows remain untouched and outside this slice.

## Deferred parent lifecycle actions
- [ ] 13.2 Production fiscal-policy, issuer/series, retention, storage, backup, access-log, and legal-void decisions. <!-- sdd-owner: parent -->
- [ ] 13.3 Approved Cloudware validation account, credentials, keyring, and license evidence. <!-- sdd-owner: parent -->
- [ ] 13.4 Approval of every applicable Cloudware enablement gate. <!-- sdd-owner: parent -->
- [ ] 13.5 Ordinary repository review after each work unit and final cross-spec traceability. <!-- sdd-owner: parent -->

## WU1 independent-verification remediation

### Findings and implementation evidence
- The independent verifier found that `fiscal_lines_immutable` omitted INSERT and ignored `fiscal_snapshots.frozen_at` for line UPDATE/DELETE. PostgreSQL RED tests reproduced all three mutations while the document remained `draft`; the migration trigger now rejects INSERT/UPDATE/DELETE whenever either the parent document is non-draft or its snapshot is frozen.
- The independent verifier found incomplete and unscoped startup validation. `fiscal_schema.go` now inventories all 376 migration requirements across the migration marker, tables, columns, constraints, indexes, and table-scoped triggers, including the two intent/void indexes and three omitted append/config triggers named in the finding.
- PostgreSQL sabotage cases independently remove each specifically reported index/trigger, a required column, and a required constraint. A decoy trigger with the expected name on the wrong table proves trigger discovery is scoped to the expected current schema and table.

### Verification commands
- RED on PostgreSQL 16: `cd backend && go test ./tests/integration -run FiscalMigration -count=1 -v` failed for all eight missing-object cases and for frozen-snapshot line INSERT before production changes.
- GREEN/TRIANGULATE/REFACTOR on isolated `postgres:16-alpine`: `cd backend && go test ./tests/integration -run FiscalMigration -count=1 -v` passed all three top-level tests and eight missing-object subtests after the catalog-query refactor.
- Regression: `cd backend && go test ./... -count=1` passed.
- Static analysis: `cd backend && go vet ./...` passed.

### TDD Cycle Evidence
| Task | Test file/layer | Safety net | RED | GREEN | TRIANGULATE | REFACTOR |
|---|---|---|---|---|---|---|
| 1.5 | `backend/tests/integration/fiscal_migration_test.go` / PostgreSQL 16 | Existing 2 fiscal migration tests passed on PostgreSQL 16 | Frozen-draft INSERT and all eight schema sabotage cases failed as expected | INSERT/UPDATE/DELETE protection and complete schema verification passed | Eight independent missing-object paths plus wrong-table trigger decoy passed | Catalog inspection reduced verification to six bounded queries; focused tests stayed green |

### Structured status and boundary
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
taskProgress: { total: 53, complete: 5, remaining: 48 }
deferredParentActions: { total: 5, complete: 1, remaining: 4 }
taskArtifactErrors: []
applyState: ready
dependencies: { apply: ready, verify: blocked, sync: blocked, archive: blocked }
actionContext:
  mode: repo-local
  workspaceRoot: D:/repos/gonsgarage
  allowedEditRoots:
    - backend/migrations/011_fiscal_integration_foundation.up.sql
    - backend/tests/integration/fiscal_migration_test.go
    - backend/internal/repository/postgres/fiscal_schema.go
    - openspec/changes/cloudware-fiscal-integration-foundation/apply-progress.md
    - openspec/changes/cloudware-fiscal-integration-foundation/tasks.md
  warnings: []
nextRecommended: parent-lifecycle
isNonAuthoritative: false
```

The broader change remains `applyState: ready` because WU2–WU12 are intentionally unchecked. This 276-line remediation delta stayed below the 300-line hard cap, crossed no WU1 boundary, and leaves ordinary review/verification lifecycle work to the parent.

## Remaining implementation tasks (verbatim)

- [ ] 2.1 RED — Create `backend/internal/domain/fiscal_decimal_test.go` and `backend/internal/domain/fiscal_policy_test.go` covering strict non-exponent decimal strings, negative-zero normalization, scale limits, quantity × price, discounts, IVA/exemption, rounding points, adjustment limits, total mismatch codes, and fail-closed missing/unapproved policy inputs. <!-- sdd-owner: implementation -->
- [ ] 2.2 GREEN — Add `github.com/shopspring/decimal` to `backend/go.mod`/`backend/go.sum` and implement `backend/internal/domain/fiscal_decimal.go` plus `backend/internal/domain/fiscal_policy.go` so all canonical calculations avoid `float64`, return decimal strings, and require an immutable approved policy version. <!-- sdd-owner: implementation -->
- [ ] 2.3 TRIANGULATE — Add table/property-style cases in the same domain tests for FT and FR, taxable and exempt lines, boundary precision, over-discount, unsupported currency/tax codes, deterministic line ordering, and stable canonical SHA-256 across equivalent inputs. <!-- sdd-owner: implementation -->
- [ ] 2.4 REFACTOR — Centralize validation/error codes and canonical serialization in `backend/internal/domain/fiscal_decimal.go` and `backend/internal/domain/fiscal_policy.go`; run `cd backend && go test ./internal/domain/... -count=1` and `gofmt` on the package. <!-- sdd-owner: implementation -->
- [ ] 2.5 RED — Create `backend/internal/domain/fiscal_document_test.go`, `fiscal_connection_test.go`, and `fiscal_artifact_test.go` for every allowed/forbidden lifecycle transition, immutable frozen fields, rejected-intent supersession, provider fixation, role/action matrix, artifact-state independence, and mock/legal classification. <!-- sdd-owner: implementation -->
- [ ] 2.6 GREEN — Implement `backend/internal/domain/fiscal_document.go`, `fiscal_connection.go`, and `fiscal_artifact.go` with the eleven lifecycle states, guarded transition function, frozen DTO, safe errors, server-derived allowed actions, and provider-neutral presentation mapping. <!-- sdd-owner: implementation -->
- [ ] 2.7 TRIANGULATE — Expand domain tests for employee/client denial, unknown-state retry prohibition, definite failed void returning to `issued`, source-reference traceability after source deletion, and byte-stable requests after invoice/customer/issuer changes. <!-- sdd-owner: implementation -->
- [ ] 2.8 REFACTOR — Keep Cloudware names and DTOs out of `backend/internal/domain` except the provider key value, document invariants beside constructors, and run domain tests with the race detector. <!-- sdd-owner: implementation -->
- [ ] 3.1 RED — Define fakes and failing service tests under `backend/internal/service/fiscal/*_test.go` plus PostgreSQL tests under `backend/tests/integration/fiscal_repository_test.go` for employee draft CRUD, client denial, optimistic versions, one current intent, atomic finalize/outbox rollback, repeated/concurrent finalization, and protected invoice deletion. <!-- sdd-owner: implementation -->
- [ ] 3.2 GREEN — Add provider-neutral ports in `backend/internal/core/ports/fiscal_repository.go`, `fiscal_provider.go`, `fiscal_artifact_store.go`, and `credential_cipher.go`; implement aggregate/config/action queries in `backend/internal/repository/postgres/fiscal_repository.go` and draft/finalization commands in `backend/internal/service/fiscal/draft_service.go` and `finalization_service.go`. <!-- sdd-owner: implementation -->
- [ ] 3.3 TRIANGULATE — Add PostgreSQL race tests proving one frozen intent and one initial issue event, complete rollback when event insertion fails, retry/reconcile/void sequence numbering, same provider/connection/operation key reuse, and no action from unknown states except reconciliation. <!-- sdd-owner: implementation -->
- [ ] 3.4 REFACTOR — Add narrow `FiscalProtectionReader` and eligibility persistence through `backend/internal/core/ports/repositories.go`, `backend/internal/domain/invoice.go`, `backend/internal/repository/postgres/invoice_repository.go`, and `backend/internal/service/invoice/invoice_service.go`; retain the legacy JSON DTO and verify existing invoice service/repository tests unchanged. <!-- sdd-owner: implementation -->
- [ ] 4.1 RED — Add `backend/internal/integration/fiscal/mock/mock_provider_test.go` and repository integration tests for successful FT/FR issuance, stable repeated/concurrent calls, validation rejection, expired connection, definite transient/rate-limit errors, ambiguous-then-reconcile, permitted/refused voids, process reload, and deterministic PDF bytes. <!-- sdd-owner: implementation -->
- [ ] 4.2 GREEN — Implement the normalized gateway/result/error contracts in `backend/internal/core/ports/fiscal_provider.go` and the persisted mock in `backend/internal/integration/fiscal/mock/*`, selecting scenarios only through an injected registry keyed by operation key and never through fiscal identity fields. <!-- sdd-owner: implementation -->
- [ ] 4.3 TRIANGULATE — Verify mock references and PDF checksums remain stable for identical operation key plus canonical input, change for different input, survive repository reload, and expose every scenario required by `specs/fiscal-provider-foundation/spec.md`. <!-- sdd-owner: implementation -->
- [ ] 4.4 REFACTOR — Add runtime guards in the mock package and `backend/cmd/api/main.go` composition so `APP_ENV=production` rejects mock provider selection, mock repository access, or `legal` artifact classification; keep all mock PDFs visibly labeled `SEM VALIDADE FISCAL — MOCK`. <!-- sdd-owner: implementation -->
- [ ] 5.1 RED — Add `backend/tests/integration/fiscal_outbox_test.go` and `backend/internal/service/fiscal/worker_test.go` for `SKIP LOCKED` single claims, lease-token/owner fencing, lease expiry, started-attempt-before-call, stale started mutation becoming unknown, result transaction atomicity, and graceful shutdown. <!-- sdd-owner: implementation -->
- [ ] 5.2 GREEN — Implement `backend/internal/repository/postgres/fiscal_outbox_repository.go`, attempt persistence/redaction, and `backend/internal/service/fiscal/worker.go`; add `backend/cmd/fiscal-worker/main.go` with issue, reconcile-issue, void, reconcile-void, and artifact-recovery dispatch outside HTTP requests. <!-- sdd-owner: implementation -->
- [ ] 5.3 TRIANGULATE — Exercise process-crash windows before call, during call, after provider response, and after lease loss; prove only definite non-acceptance reaches retryable/rejected states, ambiguous mutations never auto-resubmit, and reconciliations use the same frozen provider/key. <!-- sdd-owner: implementation -->
- [ ] 5.4 REFACTOR — Separate claim, pre-call, gateway, and completion transactions; centralize bounded lease settings and redaction; verify `cd backend && go test ./internal/service/fiscal ./tests/integration -count=1 -race -timeout=2m`. <!-- sdd-owner: implementation -->
- [ ] 6.1 RED — Add `backend/internal/service/fiscal/artifact_service_test.go`, `backend/internal/platform/fiscalartifact/local_test.go`, and PostgreSQL artifact tests for create-if-absent checksum matching, collision rejection, failed archive retaining `issued`, recovery without `Issue`, ownership authorization, access logs, media/signature/size limits, and compromised-byte refusal. <!-- sdd-owner: implementation -->
- [ ] 6.2 GREEN — Implement `backend/internal/core/ports/fiscal_artifact_store.go`, `backend/internal/platform/fiscalartifact/local.go`, `backend/internal/repository/postgres/fiscal_artifact_repository.go`, and `backend/internal/service/fiscal/artifact_service.go` with deterministic private keys, streaming SHA-256, immutable writes, metadata, and current ownership checks. <!-- sdd-owner: implementation -->
- [ ] 6.3 TRIANGULATE — Test owning versus non-owning clients, all staff roles, unavailable and compromised artifacts, repeated archive attempts, safe filenames/headers, no storage/provider URL leakage, and continued read access while issuance is disabled. <!-- sdd-owner: implementation -->
- [ ] 6.4 REFACTOR — Introduce the production adapter boundary at `backend/internal/platform/fiscalartifact/object.go` and readiness checks for private ACL, encryption, retention, backup/restore, and access logging; keep production issuance fail-closed until the chosen backend passes those checks. <!-- sdd-owner: implementation -->
- [ ] 7.1 RED — Add handler/route tests in `backend/internal/handler/fiscal_handler_test.go`, `fiscal_integration_handler_test.go`, and `invoice_handler_test.go` for static summary-route ordering, strict decimal JSON, body/line limits, 403/404/409/422/202 mappings, repeated actions, own-only projection/artifacts, and manager/admin-only legal actions. <!-- sdd-owner: implementation -->
- [ ] 7.2 GREEN — Implement DTOs and endpoints in `backend/internal/handler/fiscal_handler.go` for nested draft/detail/finalize/retry/reconcile/void, batch summaries, and PDF streaming; register `/invoices/fiscalization-summaries` before `/:id` in `backend/cmd/api/main.go` and repeat authorization in services. <!-- sdd-owner: implementation -->
- [ ] 7.3 TRIANGULATE — Extend `backend/internal/handler/p1_accounting_routes_test.go` and invoice handler/service tests to snapshot existing list/create/own-list/detail/PATCH/delete fields, envelopes, pagination, RFC 3339 timestamps, staff roles, and client notes-only behavior while testing draft deletion and frozen-history conflicts. <!-- sdd-owner: implementation -->
- [ ] 7.4 REFACTOR — Centralize fiscal response/error mapping and safe projections in `backend/internal/handler/fiscal_handler.go`/`swagger_models.go`, update `backend/docs/swagger.yaml`, `swagger.json`, and `docs.go`, and verify no credentials, raw provider bodies, diagnostics, keys, or URLs can serialize. <!-- sdd-owner: implementation -->
- [ ] 8.1 RED — Add Vitest/Testing Library coverage in `frontend/src/lib/services/fiscalization.service.test.ts`, `frontend/src/app/accounting/issued-invoices/page.test.tsx`, and new `[id]/*test.tsx` files for batch summaries, manual FT/FR lines, decimal strings, readiness guidance, role-specific actions, 202 progress, polling cancellation, double-click safety, and unchanged operational edit controls. <!-- sdd-owner: implementation -->
- [ ] 8.2 GREEN — Add `frontend/src/types/fiscal.ts` and `frontend/src/lib/services/fiscalization.service.ts`; extend `frontend/src/app/accounting/issued-invoices/page.tsx` and `[id]/page.tsx` with `FiscalDraftForm.tsx` and `FiscalActions.tsx` for provider-neutral draft, readiness, lifecycle, privileged actions, and PDF download. <!-- sdd-owner: implementation -->
- [ ] 8.3 TRIANGULATE — Cover employee versus manager/admin controls, FT/FR and exempt-line validation, stale-version conflicts, unavailable/unknown guidance, source references, server-calculated totals, mock labels outside production, and stable states stopping capped polling. <!-- sdd-owner: implementation -->
- [ ] 8.4 REFACTOR — Reuse existing API/error/form primitives, preserve issued-invoice columns/create/edit behavior, keep Portuguese UI copy provider-neutral, and run `cd frontend && pnpm test -- --passWithNoTests`, `pnpm lint`, and `pnpm typecheck`. <!-- sdd-owner: implementation -->
- [ ] 9.1 RED — Extend `frontend/src/app/my-invoices/MyInvoicesListClient.test.tsx` and `[id]/MyInvoiceDetailClient.test.tsx`, and add `frontend/src/app/admin/integrations/fiscal/page.test.tsx` plus `frontend/src/components/layout/AppShell.test.tsx` cases for own-only simplified status/PDF, unchanged notes, hidden privileged data/actions, role-gated settings navigation, and safe readiness guidance. <!-- sdd-owner: implementation -->
- [ ] 9.2 GREEN — Update `frontend/src/app/my-invoices/MyInvoicesListClient.tsx` and `[id]/MyInvoiceDetailClient.tsx` with pending/finalized/voided/unavailable presentation and authorized PDF actions while preserving notes; add `frontend/src/lib/services/fiscal-integration.service.ts`, `frontend/src/app/admin/integrations/fiscal/page.tsx`, and manager/admin navigation in `frontend/src/components/layout/AppShell.tsx`. <!-- sdd-owner: implementation -->
- [ ] 9.3 TRIANGULATE — Test non-owner denial without enumeration, unavailable/compromised artifacts, employee/client absence of integration controls, OAuth-return status without token/query leakage, and no claims about AT/e-Fatura unless an evidenced projection field explicitly supports them. <!-- sdd-owner: implementation -->
- [ ] 9.4 REFACTOR — Keep connection/provider details out of ordinary client/staff components, share provider-neutral badges/actions, preserve `/my-invoices` authentication and notes flows, and run the complete frontend test/lint/typecheck/build commands. <!-- sdd-owner: implementation -->
- [ ] 10.1 RED — Add `backend/internal/integration/fiscal/cloudware/*_test.go`, `backend/internal/platform/crypto/fiscal_credentials_test.go`, and connection service tests for one-time hashed OAuth state, actor/redirect/expiry binding, invalid/reused callbacks, AES-256-GCM AAD, key rotation, refresh failure, redaction, evidenced FT/FR-only fixtures, and every incomplete enablement gate blocking before HTTP. <!-- sdd-owner: implementation -->
- [ ] 10.2 GREEN — Implement `backend/internal/platform/crypto/fiscal_credentials.go`, `backend/internal/repository/postgres/fiscal_connection_repository.go`, `backend/internal/service/fiscal/connection_service.go`, `backend/internal/handler/fiscal_integration_handler.go`, and `backend/internal/integration/fiscal/cloudware/*` for server-side OAuth/token envelopes, adapter-local DTOs, documented FT/FR mapping, normalized errors, and `CloudwareEnablementEvaluator`. <!-- sdd-owner: implementation -->
- [ ] 10.3 TRIANGULATE — Add contract fixtures for documented finalize, void, PDF request, FR associated-receipt behavior, active-license remediation, unknown mutation response defaulting to ambiguous, no unsupported FS/receipt/correction mapping, and gate `not_applicable` requiring explicit rationale. <!-- sdd-owner: implementation -->
- [ ] 10.4 REFACTOR — Enforce bounded TLS HTTP clients, allowlisted base URLs, response limits, deadlines, no transport-level mutation retries, secret-free logs/outbox/API payloads, production keyring validation, and adapter isolation verified by import/dependency tests. <!-- sdd-owner: implementation -->
- [ ] 11.1 RED — Add configuration/readiness tests near `backend/cmd/api`, `backend/cmd/fiscal-worker`, and `backend/internal/service/fiscal` for independent feature/finalization/worker/provider switches, production rejection of defaults/mock/local storage/missing schema, provider-health isolation from `/ready`, and worker readiness failures. <!-- sdd-owner: implementation -->
- [ ] 11.2 GREEN — Wire API, worker, migration, gateway, credential, and artifact composition in `backend/cmd/api/main.go`, `backend/cmd/fiscal-worker/main.go`, `backend/cmd/migrate/main.go`, `.env.prod.example`, `docker-compose*.yml`, and `deploy/`; add structured allowlisted fiscal logs, metrics, dependency status, and graceful lease shutdown without logging PII or secrets. <!-- sdd-owner: implementation -->
- [ ] 11.3 TRIANGULATE — Update `.github/workflows/ci.yml` with PostgreSQL 16, migration application, fiscal integration/concurrency tests, API/worker/migrate builds, and existing backend/frontend quality commands; test feature-off, mock non-production, Cloudware fail-closed, provider outage, and artifact-recovery configurations. <!-- sdd-owner: implementation -->
- [ ] 11.4 REFACTOR — Document schema dark launch, mock validation, production prerequisites, monitoring/alerts, credential rotation, backup/restore, artifact retention, forward-fix rollback, and legal-history preservation in `docs/fiscal-integration-runbook.md` and `docs/fiscal-integration-traceability.md`; link scenarios to all four change specs. <!-- sdd-owner: implementation -->
- [ ] 12.1 RED — Using only an approved controlled environment and secret-handling procedure, add failing credentialed acceptance cases in an isolated `backend/tests/cloudware_acceptance/` target for OAuth/reconnect, FT/FR responses, idempotency/reconciliation lookup, rate limits, PDF authority/lifetime, voiding, active-license behavior, and AT/e-Fatura claims; keep this target out of ordinary CI when credentials are absent. <!-- sdd-owner: implementation -->
- [ ] 12.2 GREEN — Update only `backend/internal/integration/fiscal/cloudware/*`, gate records through the approved administrative mechanism, and deployment secret/configuration references to implement observed contracts; never copy live secrets or raw provider payloads into fixtures, source, logs, or OpenSpec artifacts. <!-- sdd-owner: implementation -->
- [ ] 12.3 TRIANGULATE — Re-run credentialed acceptance for timeouts, malformed/unknown responses, token expiry/rotation, repeated operation keys, exact reconciliation matches, PDF recovery, FT/FR differences, void refusal/success, and license failure; retain unknown behavior as a blocking gate rather than guessing. <!-- sdd-owner: implementation -->
- [ ] 12.4 REFACTOR — Enable only evidenced capabilities behind `CloudwareEnablementEvaluator`, preserve explicit no-webhook/no-auto-retry behavior where applicable, update `docs/fiscal-integration-runbook.md` with evidence references rather than secrets, and prove disabling Cloudware leaves invoice reads/notes and archived artifact reads available. <!-- sdd-owner: implementation -->

## WU1 verification evidence continuation
- Added independent sabotage cases for missing `fiscal_mock_operations` and the deleted migration 011 marker; both assert the exact missing-object identity from `VerifyFiscalSchema`.
- Safety net: the focused PostgreSQL 16 suite passed before editing (3 top-level tests and 8 existing sabotage subtests).
- Verification: `cd backend && go test ./tests/integration -run FiscalMigration -count=1 -v` passed with 10 sabotage subtests; `go test ./... -count=1` and `go vet ./...` passed.
- No production code, WU2 task, task checkbox, commit, or PR changed. The continuation changed two test-table rows plus this evidence and remained within the parent-authorized WU1 boundary and 300-line cumulative cap.

### TDD Cycle Evidence
| Task | Test file/layer | Safety net | RED | GREEN | TRIANGULATE | REFACTOR |
|---|---|---|---|---|---|---|
| 1.5 continuation | `backend/tests/integration/fiscal_migration_test.go` / PostgreSQL 16 | 3 top-level tests and 8 existing sabotage cases passed | Independent verification identified two absent sabotage cases; no production RED was expected because the checker already supported both objects | Both added cases passed against the existing checker | Standalone table and migration-row removal independently exercised different catalog paths | No production refactor was needed; focused, full, and vet checks stayed green |

Native status retained `applyState: ready` for the broader change but reported `nextRecommended: resolve-blockers` because the local CLI could not decode the parent-issued runtime authority field `max_changed_lines_explicit`. The parent-provided continuation was explicitly `proceed`; its narrower `repo-local` edit roots limited this slice to the test and progress files.
