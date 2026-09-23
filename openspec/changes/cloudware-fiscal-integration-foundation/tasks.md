# Tasks: Cloudware fiscal integration foundation

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | 7,000–12,000 across roughly 55–80 source, test, migration, deployment, and documentation files |
| 400-line budget risk | High — the confirmed session budget is 600 changed lines, and the forecast materially exceeds it |
| Chained PRs recommended | Yes |
| Suggested split | PR 1 schema/domain → PR 2 persistence/API foundation → PR 3 mock/worker/artifacts → PR 4 staff/client UI → PR 5 Cloudware security shell/operations; live enablement remains separately gated |
| Delivery strategy | single-pr |
| Chain strategy | pending |

Decision needed before apply: Yes
Chained PRs recommended: Yes
Chain strategy: pending
400-line budget risk: High

The requested `single-pr` strategy conflicts with the 600-line review budget. Before apply, the user must either change the delivery strategy or explicitly accept `size:exception`; no exception is inferred here. The agent must not create commits or PRs.

## Work-unit boundaries

| Unit | Start state | Finished and verified state | Rollback boundary |
|---|---|---|---|
| WU1 — schema and legacy isolation | Migration `010` is current and fiscal schema is absent. | Migration `011`, fail-fast schema checks, and PostgreSQL tests prove legacy rows remain isolated. | Down migration is allowed only before fiscal data or credentials exist; otherwise forward-fix. |
| WU2 — exact domain model | No fiscal domain types or approved-policy calculator exist. | Decimal, calculator, lifecycle, immutability, and action-matrix tests pass without provider dependencies. | Remove the isolated domain package and dependency before persistence adopts it. |
| WU3 — aggregate persistence | Schema and domain contracts pass. | Draft/finalization/retry/reconcile/void repositories are atomic, fenced, and preserve invoice compatibility. | Disable fiscal feature routes; retain schema and any frozen evidence. |
| WU4 — provider-neutral mock | Gateway port exists with no executable provider. | Non-production mock deterministically covers FT/FR, failures, reconciliation, voiding, and labeled PDFs. | Disable mock selection and remove only mock operation rows in non-production. |
| WU5 — outbox worker | Finalization can write durable work, but nothing dispatches it. | Lease fencing, crash ambiguity, explicit recovery, and graceful worker shutdown pass PostgreSQL tests. | Stop new claims; allow leases to finish or expire conservatively. |
| WU6 — artifact archive | Issuance can succeed without a private archive. | Immutable local test storage, metadata, recovery, integrity checks, and authorized streaming pass. | Disable new issuance while retaining read access to existing artifacts. |
| WU7 — additive HTTP API | Services are internal only. | Nested fiscal routes return specified status codes and leave legacy invoice contracts unchanged. | Disable fiscal routes; database deletion guards continue protecting retained evidence. |
| WU8 — staff UI | Existing issued-invoice pages show only operational fields. | Staff can prepare drafts and authorized managers/admins can act using provider-neutral projections. | Hide fiscal controls while leaving operational invoice UI unchanged. |
| WU9 — client/admin UI | Client invoice and admin navigation have no fiscal surfaces. | Own-only client status/PDF and privileged integration settings pass component tests. | Remove additive navigation/panels without changing notes or invoice pages. |
| WU10 — Cloudware security shell | Only provider-neutral and mock behavior is enabled. | Encrypted credentials, OAuth-state handling, evidenced FT/FR fixtures, and fail-closed gate evaluation pass with fake HTTP. | Keep Cloudware mutations disabled and revoke/remove active credentials without deleting audit data. |
| WU11 — operations and rollout | Feature behavior exists but is not production-operable. | CI, migration command, deployment switches, observability, security checks, and runbooks pass with finalization off by default. | Turn off finalization/worker; retain reads, evidence, and archived artifacts. |
| WU12 — live Cloudware enablement | All Cloudware mutations are disabled. | Only credentialed acceptance evidence can satisfy gates and permit an explicitly approved controlled rollout. | Disable mutations, stop claims, preserve frozen documents/artifacts, and use legal void workflows rather than data deletion. |

Each unit is intended to map to a reviewable manual commit or, if the delivery strategy changes, an autonomous chained PR. Tests and documentation stay with the behavior they verify.

## Phase 1 — Migration, schema validation, and legacy compatibility (WU1)

- [x] 1.1 RED — Add PostgreSQL integration tests in `backend/tests/integration/fiscal_migration_test.go` that apply `backend/migrations/011_fiscal_integration_foundation.up.sql` over historical invoices and assert `legacy_unfiscalized`, zero fiscal documents/outbox events, required constraints/indexes/triggers, and rejection of forbidden frozen/history mutations. <!-- sdd-owner: implementation -->
- [x] 1.2 GREEN — Implement `backend/migrations/011_fiscal_integration_foundation.up.sql` with the invoice eligibility column and all configuration, connection, aggregate, transition, outbox, attempt, artifact, access-log, mock-operation, gate, immutability, audit, and invoice-delete protections specified in `design.md`; insert Cloudware gates only as `pending` and seed no legal policy/profile. <!-- sdd-owner: implementation -->
- [x] 1.3 TRIANGULATE — Extend `backend/tests/integration/fiscal_migration_test.go` for partial-current-intent uniqueness, operation/provider-reference uniqueness, artifact consistency, append-only history, draft-versus-frozen invoice deletion, and old/direct invoice inserts failing safe as legacy. <!-- sdd-owner: implementation -->
- [x] 1.4 REFACTOR — Move `backend/scripts/run_migrations.go` into a reusable runner plus `backend/cmd/migrate/main.go`, add `backend/internal/repository/postgres/fiscal_schema.go`, and prove fiscal startup fails on missing migration objects while unrelated legacy startup remains unchanged. <!-- sdd-owner: implementation -->
- [x] 1.5 VERIFICATION REMEDIATION — Reject fiscal-line INSERT/UPDATE/DELETE after snapshot freeze even while the document is draft, and fail startup when any required migration 011 table, column, constraint, index, or table-scoped trigger is missing. <!-- sdd-owner: implementation -->

## Phase 2 — Exact decimal arithmetic and lifecycle domain (WU2)

- [x] 2.1 RED — Create `backend/internal/domain/fiscal_decimal_test.go` and `backend/internal/domain/fiscal_policy_test.go` covering strict non-exponent decimal strings, negative-zero normalization, scale limits, quantity × price, discounts, IVA/exemption, rounding points, adjustment limits, total mismatch codes, and fail-closed missing/unapproved policy inputs. <!-- sdd-owner: implementation -->
- [x] 2.2 GREEN — Add `github.com/shopspring/decimal` to `backend/go.mod`/`backend/go.sum` and implement `backend/internal/domain/fiscal_decimal.go` plus `backend/internal/domain/fiscal_policy.go` so all canonical calculations avoid `float64`, return decimal strings, and require an immutable approved policy version. <!-- sdd-owner: implementation -->
- [x] 2.3 TRIANGULATE — Add table/property-style cases in the same domain tests for FT and FR, taxable and exempt lines, boundary precision, over-discount, unsupported currency/tax codes, deterministic line ordering, and stable canonical SHA-256 across equivalent inputs. <!-- sdd-owner: implementation -->
- [x] 2.4 REFACTOR — Centralize validation/error codes and canonical serialization in `backend/internal/domain/fiscal_decimal.go` and `backend/internal/domain/fiscal_policy.go`; run `cd backend && go test ./internal/domain/... -count=1` and `gofmt` on the package. <!-- sdd-owner: implementation -->
- [x] 2.5 RED — Create `backend/internal/domain/fiscal_document_test.go`, `fiscal_connection_test.go`, and `fiscal_artifact_test.go` for every allowed/forbidden lifecycle transition, immutable frozen fields, rejected-intent supersession, provider fixation, role/action matrix, artifact-state independence, and mock/legal classification. <!-- sdd-owner: implementation -->
- [x] 2.6 GREEN — Implement `backend/internal/domain/fiscal_document.go`, `fiscal_connection.go`, and `fiscal_artifact.go` with the eleven lifecycle states, guarded transition function, frozen DTO, safe errors, server-derived allowed actions, and provider-neutral presentation mapping. <!-- sdd-owner: implementation -->
- [x] 2.7 TRIANGULATE — Expand domain tests for employee/client denial, unknown-state retry prohibition, definite failed void returning to `issued`, source-reference traceability after source deletion, and byte-stable requests after invoice/customer/issuer changes. <!-- sdd-owner: implementation -->
- [x] 2.8 REFACTOR — Keep Cloudware names and DTOs out of `backend/internal/domain` except the provider key value, document invariants beside constructors, and run domain tests with the race detector. <!-- sdd-owner: implementation -->

## Phase 3 — Fiscal repositories and atomic application services (WU3)

- [x] 3.1 RED — Define fakes and failing service tests under `backend/internal/service/fiscal/*_test.go` plus PostgreSQL tests under `backend/tests/integration/fiscal_repository_test.go` for employee draft CRUD, client denial, optimistic versions, one current intent, atomic finalize/outbox rollback, repeated/concurrent finalization, and protected invoice deletion. <!-- sdd-owner: implementation -->
- [x] 3.2 GREEN — Add provider-neutral ports in `backend/internal/core/ports/fiscal_repository.go`, `fiscal_provider.go`, `fiscal_artifact_store.go`, and `credential_cipher.go`; implement aggregate/config/action queries in `backend/internal/repository/postgres/fiscal_repository.go` and draft/finalization commands in `backend/internal/service/fiscal/draft_service.go` and `finalization_service.go`. <!-- sdd-owner: implementation -->
- [x] 3.3 TRIANGULATE — Add PostgreSQL race tests proving one frozen intent and one initial issue event, complete rollback when event insertion fails, retry/reconcile/void sequence numbering, same provider/connection/operation key reuse, and no action from unknown states except reconciliation. <!-- sdd-owner: implementation -->
- [x] 3.4 REFACTOR — Add narrow `FiscalProtectionReader` and eligibility persistence through `backend/internal/core/ports/repositories.go`, `backend/internal/domain/invoice.go`, `backend/internal/repository/postgres/invoice_repository.go`, and `backend/internal/service/invoice/invoice_service.go`; retain the legacy JSON DTO and verify existing invoice service/repository tests unchanged. <!-- sdd-owner: implementation -->

## Phase 4 — Provider-neutral gateway and deterministic mock (WU4)

- [x] 4.1 RED — Add `backend/internal/integration/fiscal/mock/mock_provider_test.go` and repository integration tests for successful FT/FR issuance, stable repeated/concurrent calls, validation rejection, expired connection, definite transient/rate-limit errors, ambiguous-then-reconcile, permitted/refused voids, process reload, and deterministic PDF bytes. <!-- sdd-owner: implementation -->
- [x] 4.2 GREEN — Implement the normalized gateway/result/error contracts in `backend/internal/core/ports/fiscal_provider.go` and the persisted mock in `backend/internal/integration/fiscal/mock/*`, selecting scenarios only through an injected registry keyed by operation key and never through fiscal identity fields. <!-- sdd-owner: implementation -->
- [x] 4.3 TRIANGULATE — Verify mock references and PDF checksums remain stable for identical operation key plus canonical input, change for different input, survive repository reload, and expose every scenario required by `specs/fiscal-provider-foundation/spec.md`. <!-- sdd-owner: implementation -->
- [x] 4.4 REFACTOR — Add runtime guards in the mock package and `backend/cmd/api/main.go` composition so `APP_ENV=production` rejects mock provider selection, mock repository access, or `legal` artifact classification; keep all mock PDFs visibly labeled `SEM VALIDADE FISCAL — MOCK`. <!-- sdd-owner: implementation -->

## Phase 5 — Transactional outbox, worker, and reconciliation (WU5)

- [x] 5.1 RED — Add `backend/tests/integration/fiscal_outbox_test.go` and `backend/internal/service/fiscal/worker_test.go` for `SKIP LOCKED` single claims, lease-token/owner fencing, lease expiry, started-attempt-before-call, stale started mutation becoming unknown, result transaction atomicity, and graceful shutdown. <!-- sdd-owner: implementation -->
- [x] 5.2 GREEN — Implement `backend/internal/repository/postgres/fiscal_outbox_repository.go`, attempt persistence/redaction, and `backend/internal/service/fiscal/worker.go`; add `backend/cmd/fiscal-worker/main.go` with issue, reconcile-issue, void, reconcile-void, and artifact-recovery dispatch outside HTTP requests. <!-- sdd-owner: implementation -->
- [x] 5.3 TRIANGULATE — Exercise process-crash windows before call, during call, after provider response, and after lease loss; prove only definite non-acceptance reaches retryable/rejected states, ambiguous mutations never auto-resubmit, and reconciliations use the same frozen provider/key. <!-- sdd-owner: implementation -->
- [x] 5.4 REFACTOR — Separate claim, pre-call, gateway, and completion transactions; centralize bounded lease settings and redaction; verify `cd backend && go test ./internal/service/fiscal ./tests/integration -count=1 -race -timeout=2m`. <!-- sdd-owner: implementation -->

## Phase 6 — Immutable private artifact storage and recovery (WU6)

- [x] 6.1 RED — Add `backend/internal/service/fiscal/artifact_service_test.go`, `backend/internal/platform/fiscalartifact/local_test.go`, and PostgreSQL artifact tests for create-if-absent checksum matching, collision rejection, failed archive retaining `issued`, recovery without `Issue`, ownership authorization, access logs, media/signature/size limits, and compromised-byte refusal. <!-- sdd-owner: implementation -->
- [x] 6.2 GREEN — Implement `backend/internal/core/ports/fiscal_artifact_store.go`, `backend/internal/platform/fiscalartifact/local.go`, `backend/internal/repository/postgres/fiscal_artifact_repository.go`, and `backend/internal/service/fiscal/artifact_service.go` with deterministic private keys, streaming SHA-256, immutable writes, metadata, and current ownership checks. <!-- sdd-owner: implementation -->
- [x] 6.3 TRIANGULATE — Test owning versus non-owning clients, all staff roles, unavailable and compromised artifacts, repeated archive attempts, safe filenames/headers, no storage/provider URL leakage, and continued read access while issuance is disabled. <!-- sdd-owner: implementation -->
- [x] 6.4 REFACTOR — Introduce the production adapter boundary at `backend/internal/platform/fiscalartifact/object.go` and readiness checks for private ACL, encryption, retention, backup/restore, and access logging; keep production issuance fail-closed until the chosen backend passes those checks. <!-- sdd-owner: implementation -->

## Phase 7 — Additive fiscal HTTP APIs and legacy regression (WU7)

- [x] 7.1 RED — Add handler/route tests in `backend/internal/handler/fiscal_handler_test.go`, `fiscal_integration_handler_test.go`, and `invoice_handler_test.go` for static summary-route ordering, strict decimal JSON, body/line limits, 403/404/409/422/202 mappings, repeated actions, own-only projection/artifacts, and manager/admin-only legal actions. <!-- sdd-owner: implementation -->
- [x] 7.2 GREEN — Implement DTOs and endpoints in `backend/internal/handler/fiscal_handler.go` for nested draft/detail/finalize/retry/reconcile/void, batch summaries, and PDF streaming; register `/invoices/fiscalization-summaries` before `/:id` in `backend/cmd/api/main.go` and repeat authorization in services. <!-- sdd-owner: implementation -->
- [x] 7.3 TRIANGULATE — Extend `backend/internal/handler/p1_accounting_routes_test.go` and invoice handler/service tests to snapshot existing list/create/own-list/detail/PATCH/delete fields, envelopes, pagination, RFC 3339 timestamps, staff roles, and client notes-only behavior while testing draft deletion and frozen-history conflicts. <!-- sdd-owner: implementation -->
- [x] 7.4 REFACTOR — Centralize fiscal response/error mapping and safe projections in `backend/internal/handler/fiscal_handler.go`/`swagger_models.go`, update `backend/docs/swagger.yaml`, `swagger.json`, and `docs.go`, and verify no credentials, raw provider bodies, diagnostics, keys, or URLs can serialize. <!-- sdd-owner: implementation -->

## Phase 8 — Staff fiscal UI (WU8)

- [ ] 8.1 RED — Add Vitest/Testing Library coverage in `frontend/src/lib/services/fiscalization.service.test.ts`, `frontend/src/app/accounting/issued-invoices/page.test.tsx`, and new `[id]/*test.tsx` files for batch summaries, manual FT/FR lines, decimal strings, readiness guidance, role-specific actions, 202 progress, polling cancellation, double-click safety, and unchanged operational edit controls. <!-- sdd-owner: implementation -->
- [ ] 8.2 GREEN — Add `frontend/src/types/fiscal.ts` and `frontend/src/lib/services/fiscalization.service.ts`; extend `frontend/src/app/accounting/issued-invoices/page.tsx` and `[id]/page.tsx` with `FiscalDraftForm.tsx` and `FiscalActions.tsx` for provider-neutral draft, readiness, lifecycle, privileged actions, and PDF download. <!-- sdd-owner: implementation -->
- [ ] 8.3 TRIANGULATE — Cover employee versus manager/admin controls, FT/FR and exempt-line validation, stale-version conflicts, unavailable/unknown guidance, source references, server-calculated totals, mock labels outside production, and stable states stopping capped polling. <!-- sdd-owner: implementation -->
- [ ] 8.4 REFACTOR — Reuse existing API/error/form primitives, preserve issued-invoice columns/create/edit behavior, keep Portuguese UI copy provider-neutral, and run `cd frontend && pnpm test -- --passWithNoTests`, `pnpm lint`, and `pnpm typecheck`. <!-- sdd-owner: implementation -->

## Phase 9 — Client status/PDF and admin integration UI (WU9)

- [ ] 9.1 RED — Extend `frontend/src/app/my-invoices/MyInvoicesListClient.test.tsx` and `[id]/MyInvoiceDetailClient.test.tsx`, and add `frontend/src/app/admin/integrations/fiscal/page.test.tsx` plus `frontend/src/components/layout/AppShell.test.tsx` cases for own-only simplified status/PDF, unchanged notes, hidden privileged data/actions, role-gated settings navigation, and safe readiness guidance. <!-- sdd-owner: implementation -->
- [ ] 9.2 GREEN — Update `frontend/src/app/my-invoices/MyInvoicesListClient.tsx` and `[id]/MyInvoiceDetailClient.tsx` with pending/finalized/voided/unavailable presentation and authorized PDF actions while preserving notes; add `frontend/src/lib/services/fiscal-integration.service.ts`, `frontend/src/app/admin/integrations/fiscal/page.tsx`, and manager/admin navigation in `frontend/src/components/layout/AppShell.tsx`. <!-- sdd-owner: implementation -->
- [ ] 9.3 TRIANGULATE — Test non-owner denial without enumeration, unavailable/compromised artifacts, employee/client absence of integration controls, OAuth-return status without token/query leakage, and no claims about AT/e-Fatura unless an evidenced projection field explicitly supports them. <!-- sdd-owner: implementation -->
- [ ] 9.4 REFACTOR — Keep connection/provider details out of ordinary client/staff components, share provider-neutral badges/actions, preserve `/my-invoices` authentication and notes flows, and run the complete frontend test/lint/typecheck/build commands. <!-- sdd-owner: implementation -->

## Phase 10 — Cloudware adapter preparation, encryption, and fail-closed gates (WU10)

This phase uses fake HTTP and test credentials only. It does not authorize or perform live Cloudware mutations.

- [ ] 10.1 RED — Add `backend/internal/integration/fiscal/cloudware/*_test.go`, `backend/internal/platform/crypto/fiscal_credentials_test.go`, and connection service tests for one-time hashed OAuth state, actor/redirect/expiry binding, invalid/reused callbacks, AES-256-GCM AAD, key rotation, refresh failure, redaction, evidenced FT/FR-only fixtures, and every incomplete enablement gate blocking before HTTP. <!-- sdd-owner: implementation -->
- [ ] 10.2 GREEN — Implement `backend/internal/platform/crypto/fiscal_credentials.go`, `backend/internal/repository/postgres/fiscal_connection_repository.go`, `backend/internal/service/fiscal/connection_service.go`, `backend/internal/handler/fiscal_integration_handler.go`, and `backend/internal/integration/fiscal/cloudware/*` for server-side OAuth/token envelopes, adapter-local DTOs, documented FT/FR mapping, normalized errors, and `CloudwareEnablementEvaluator`. <!-- sdd-owner: implementation -->
- [ ] 10.3 TRIANGULATE — Add contract fixtures for documented finalize, void, PDF request, FR associated-receipt behavior, active-license remediation, unknown mutation response defaulting to ambiguous, no unsupported FS/receipt/correction mapping, and gate `not_applicable` requiring explicit rationale. <!-- sdd-owner: implementation -->
- [ ] 10.4 REFACTOR — Enforce bounded TLS HTTP clients, allowlisted base URLs, response limits, deadlines, no transport-level mutation retries, secret-free logs/outbox/API payloads, production keyring validation, and adapter isolation verified by import/dependency tests. <!-- sdd-owner: implementation -->

## Phase 11 — Observability, PostgreSQL CI, deployment, and rollout documentation (WU11)

- [ ] 11.1 RED — Add configuration/readiness tests near `backend/cmd/api`, `backend/cmd/fiscal-worker`, and `backend/internal/service/fiscal` for independent feature/finalization/worker/provider switches, production rejection of defaults/mock/local storage/missing schema, provider-health isolation from `/ready`, and worker readiness failures. <!-- sdd-owner: implementation -->
- [ ] 11.2 GREEN — Wire API, worker, migration, gateway, credential, and artifact composition in `backend/cmd/api/main.go`, `backend/cmd/fiscal-worker/main.go`, `backend/cmd/migrate/main.go`, `.env.prod.example`, `docker-compose*.yml`, and `deploy/`; add structured allowlisted fiscal logs, metrics, dependency status, and graceful lease shutdown without logging PII or secrets. <!-- sdd-owner: implementation -->
- [ ] 11.3 TRIANGULATE — Update `.github/workflows/ci.yml` with PostgreSQL 16, migration application, fiscal integration/concurrency tests, API/worker/migrate builds, and existing backend/frontend quality commands; test feature-off, mock non-production, Cloudware fail-closed, provider outage, and artifact-recovery configurations. <!-- sdd-owner: implementation -->
- [ ] 11.4 REFACTOR — Document schema dark launch, mock validation, production prerequisites, monitoring/alerts, credential rotation, backup/restore, artifact retention, forward-fix rollback, and legal-history preservation in `docs/fiscal-integration-runbook.md` and `docs/fiscal-integration-traceability.md`; link scenarios to all four change specs. <!-- sdd-owner: implementation -->

## Phase 12 — Credential-dependent Cloudware validation and production enablement (WU12, BLOCKED)

**Blocked:** Do not start these implementation tasks until Parent tasks 13.2–13.4 record approved credentials, evidence, policy/storage readiness, and acceptance references. Production mutations remain disabled throughout partial completion.

- [ ] 12.1 RED — Using only an approved controlled environment and secret-handling procedure, add failing credentialed acceptance cases in an isolated `backend/tests/cloudware_acceptance/` target for OAuth/reconnect, FT/FR responses, idempotency/reconciliation lookup, rate limits, PDF authority/lifetime, voiding, active-license behavior, and AT/e-Fatura claims; keep this target out of ordinary CI when credentials are absent. <!-- sdd-owner: implementation -->
- [ ] 12.2 GREEN — Update only `backend/internal/integration/fiscal/cloudware/*`, gate records through the approved administrative mechanism, and deployment secret/configuration references to implement observed contracts; never copy live secrets or raw provider payloads into fixtures, source, logs, or OpenSpec artifacts. <!-- sdd-owner: implementation -->
- [ ] 12.3 TRIANGULATE — Re-run credentialed acceptance for timeouts, malformed/unknown responses, token expiry/rotation, repeated operation keys, exact reconciliation matches, PDF recovery, FT/FR differences, void refusal/success, and license failure; retain unknown behavior as a blocking gate rather than guessing. <!-- sdd-owner: implementation -->
- [ ] 12.4 REFACTOR — Enable only evidenced capabilities behind `CloudwareEnablementEvaluator`, preserve explicit no-webhook/no-auto-retry behavior where applicable, update `docs/fiscal-integration-runbook.md` with evidence references rather than secrets, and prove disabling Cloudware leaves invoice reads/notes and archived artifact reads available. <!-- sdd-owner: implementation -->

## Phase 13 — Parent-owned decisions and bounded review

These actions are intentionally separated from implementation work. Task 13.1 is a prerequisite to any apply session; tasks 13.2–13.4 are prerequisites to Phase 12 or legal production enablement.

- [x] 13.1 Resolve the forecast conflict: the user selected manual work-unit slices with a 600-line cap, starting with WU1 only. The agent MUST NOT create commits or PRs and MUST stop before WU2 or before exceeding the cap. <!-- sdd-owner: parent -->
- [ ] 13.2 Record approved fiscal-policy, issuer/series, retention, private-storage, backup/restore, access-log, and legal-void decisions in the production configuration/evidence system described by `docs/fiscal-integration-runbook.md`; keep finalization disabled for every missing item. <!-- sdd-owner: parent -->
- [ ] 13.3 Provision an approved Cloudware test company or controlled-validation account, OAuth credentials, dedicated encryption keyring, and active-GC-license evidence outside the repository; authorize the bounded credentialed acceptance target only after security review. <!-- sdd-owner: parent -->
- [ ] 13.4 Review and approve every `fiscal_enablement_gates` record with evidence, decision, and passing acceptance reference—including explicit rationale for `not_applicable`—before allowing Phase 12 to enable any production mutation. <!-- sdd-owner: parent -->
- [ ] 13.5 After each manually managed work unit, apply the repository’s ordinary review policy to its listed finish/verification/rollback boundary, and perform final scenario traceability against all four files under `openspec/changes/cloudware-fiscal-integration-foundation/specs/` before rollout. <!-- sdd-owner: parent -->
