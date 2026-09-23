```yaml
schema: gentle-ai.verify-result/v1
evidence_revision: sha256:47fb0559b67cf48c8e974dd9a4430c21df35a85ad9b7d8e1c2deb5f0a288e965
verdict: fail
blockers: 1
critical_findings: 1
requirements: 21/29
scenarios: 44/58
remediation_note: "2026-09-23 mid-change post-WU8: staff fiscal UI green (151 FE tests); WU1–WU8 focused correctness OK; API ArtifactService nil + projection omits artifactId → PDF download PARTIAL; WU9+ client/admin and WU10+ Cloudware deferred UNTESTED; archive gate FAIL at 38/58"
test_command: cd backend && go test ./internal/domain/... ./internal/service/fiscal/... ./internal/service/invoice/... ./internal/integration/fiscal/mock/... ./internal/platform/fiscalartifact/... ./internal/handler/... ./internal/core/ports/... -count=1 ; FISCAL_TEST_DATABASE_URL=<redacted> go test ./tests/integration/ -count=1 -run Fiscal(Migration|Repository|Mock|Outbox|Artifact) -timeout 180s ; cd frontend && pnpm test -- --passWithNoTests && pnpm lint && pnpm typecheck
test_exit_code: 0
test_output_hash: sha256:727ed002927b1fa5a51fa1a46212dd8084f200d98607288543f4f6ff15887354
build_command: cd backend && go vet ./internal/domain/... ./internal/service/fiscal/... ./internal/service/invoice/... ./internal/integration/fiscal/mock/... ./internal/platform/fiscalartifact/... ./internal/handler/... ./internal/core/ports/... ./internal/repository/postgres/... ; go build -o NUL ./cmd/api/ ; go build -o NUL ./cmd/fiscal-worker/
build_exit_code: 0
build_output_hash: sha256:1feef24cc5f14b2eece990b911e5e17d703ef993281a79d2bec25ad8455a2a0a
```

## Verification Report

**Change**: cloudware-fiscal-integration-foundation
**Version**: N/A (OpenSpec change; four delta specs)
**Mode**: Standard (focused mid-change verify after WU1–WU8; full suite not claimed)
**Artifact store**: openspec
**Validator**: `gentle-ai sdd-verify-validate` **unavailable** on installed gentle-ai 3.5.0 (command not present; only `sdd-status` / `sdd-continue` / `sdd-attempt grant`). Report persisted per parent Persistence instruction; admission tooling could not attest bytes.

### Completeness

| Metric | Value |
|--------|-------|
| Tasks total | 58 |
| Tasks complete | 38 (1.1–1.5, 2.1–2.8, 3.1–3.4, 4.1–4.4, 5.1–5.4, 6.1–6.4, 7.1–7.4, 8.1–8.4, 13.1) |
| Tasks incomplete | 20 (WU9–WU12 implementation + parent 13.2–13.5) |
| Change-level archive gate | **FAIL** — `allComplete: false` |
| WU1–WU8 local finish claims | Satisfied in `tasks.md` / `apply-progress.md` checkboxes |

Scope of this verify: **completed work units only** (schema through staff fiscal UI). Remaining WU9–WU12 scenarios are expected deferred, not false PASS.

### Build & Tests Execution

**Build**: ✅ Passed (`cmd/api` + `cmd/fiscal-worker` + focused vet)
```text
cd backend
go vet ./internal/domain/... ./internal/service/fiscal/... ./internal/service/invoice/... ./internal/integration/fiscal/mock/... ./internal/platform/fiscalartifact/... ./internal/handler/... ./internal/core/ports/... ./internal/repository/postgres/...
→ VET_EXIT:0
go build -o NUL ./cmd/api/
→ BUILD_API:0
go build -o NUL ./cmd/fiscal-worker/
→ BUILD_WORKER:0
build_output_hash: sha256:1feef24cc5f14b2eece990b911e5e17d703ef993281a79d2bec25ad8455a2a0a
```

**Tests**: ✅ Backend focused + Fiscal* PostgreSQL + frontend full suite / ⚠️ race unavailable
```text
# Unit / service / mock / platform / handler / invoice / ports (CGO_ENABLED=0)
go test ./internal/domain/... ./internal/service/fiscal/... ./internal/service/invoice/... ./internal/integration/fiscal/mock/... ./internal/platform/fiscalartifact/... ./internal/handler/... ./internal/core/ports/... -count=1
→ ok (exit 0); unit hash sha256:f3bc3b4dd4d1416c6362d6e7ed803b26398614af82333a8b5268aea0fb8c2669

# PostgreSQL integration (FISCAL_TEST_DATABASE_URL from backend/.env, redacted)
go test ./tests/integration/ -count=1 -run 'Fiscal(Migration|Repository|Mock|Outbox|Artifact)' -timeout 180s
→ ok tests/integration 45.468s (exit 0); pg hash sha256:fa80c334400efbcf959f4b6745fd2c98ac96a10bca8adb6220dcfac138c94e8f

# Frontend WU8 + regression
cd frontend && pnpm test -- --passWithNoTests
→ Test Files 32 passed (32) / Tests 151 passed (151) (exit 0)
→ focused WU8 files: 34 passed
→ fe hash sha256:6e342711e3de634e6ab2d2623e9dee9dcf755db22991662a94d471b53a70de37
pnpm lint → exit 0
pnpm typecheck → exit 0

test_output_hash (combined unit+pg+fe): sha256:727ed002927b1fa5a51fa1a46212dd8084f200d98607288543f4f6ff15887354

# Race detector (documented limit)
CGO_ENABLED=1 go test ./internal/domain/ -count=1 -race
→ FAIL build: cgo: C compiler "gcc" not found
→ Not treated as scenario FAIL; CI/Linux must run -race. WARNING only.
```

**Coverage**: ➖ Not measured this run (no coverage threshold enforced in focused verify)

### Spec Compliance Matrix

Authoritative totals from retrieved specs: **29 requirements**, **58 scenarios**.

Legend for deferred rows: `UNTESTED (deferred WUn)` = expected not yet implemented; does not prove WU1–WU8 incorrect.

#### fiscal-documents (9 requirements / 24 scenarios)

| Requirement | Scenario | Test | Result |
|-------------|----------|------|--------|
| Fiscal drafts distinct | Employee prepares an FT draft | `draft_finalization_test.go` > `TestDraftServiceEmployeeCRUDAndClientDenial` + FE `FiscalDraftForm.test.tsx` > submits FT draft with decimal-string quantity and unit price | ✅ COMPLIANT |
| Fiscal drafts distinct | Client attempts draft preparation | `TestDraftServiceEmployeeCRUDAndClientDenial` | ✅ COMPLIANT |
| Fiscal drafts distinct | Optional source record later changes | `fiscal_policy_test.go` > `TestCalculate_SourceReferenceSurvivesAndCanonicalIgnoresExternalMutation` + FE source-ref display | ✅ COMPLIANT |
| Fiscal drafts distinct | Unsupported document kind | `TestDraftServiceRejectsUnsupportedDocumentKind` | ✅ COMPLIANT |
| Complete fiscal data | Required fiscal identity is incomplete | `fiscal_policy_test.go` > `TestPolicyVersionResolve_ApprovalGateAndImmutability` + FE readiness guidance | ✅ COMPLIANT |
| Complete fiscal data | Legacy floating-point amount is supplied | `TestDraftServiceStoresCanonicalDecimalsIndependentOfLegacyFloatAmount` + FE operational amount separate from fiscal lines | ✅ COMPLIANT |
| Deterministic arithmetic | Arithmetic policy is unavailable | `TestPolicyVersionResolve_ApprovalGateAndImmutability` | ✅ COMPLIANT |
| Deterministic arithmetic | Totals reconcile | `TestCalculate_*` + FE shows server-calculated totals | ✅ COMPLIANT |
| Deterministic arithmetic | Submitted totals do not reconcile | `TestCalculate_DocumentRoundingAdjustmentAndDeclaredMismatch` | ✅ COMPLIANT |
| Explicit finalization | Manager finalizes an eligible draft | `TestFinalizationServiceAtomicityAndAuth` + PG finalize + FE `FiscalActions` manager finalize | ✅ COMPLIANT |
| Explicit finalization | Employee attempts finalization | `TestFinalizationServiceAtomicityAndAuth` + FE hides privileged actions for employee | ✅ COMPLIANT |
| Explicit finalization | Two finalization requests race | `TestFiscalRepositoryConcurrentFinalizationOneIntent` + FE double-click guard | ✅ COMPLIANT |
| Frozen immutable | Customer or invoice changes after finalization | domain freeze + `TestFiscalMigrationConstraintsAndImmutability` | ✅ COMPLIANT |
| Frozen immutable | Staff attempts to edit frozen lines | `TestFiscalMigrationConstraintsAndImmutability` | ✅ COMPLIANT |
| Lifecycle guarded | Definite validation rejection | `TestFiscalDocumentFinalizeFreezeAndTransitions` | ✅ COMPLIANT |
| Lifecycle guarded | Ambiguous issuance result | `TestFiscalDocumentUnknownRetryProhibitionAndFailedVoidReturnsIssued` + FE unavailable/unknown guidance | ✅ COMPLIANT |
| Lifecycle guarded | Definite void failure | same | ✅ COMPLIANT |
| Privileged recovery/void | Manager retries after connection recovery | domain transitions + role matrix + FE privileged actions | ✅ COMPLIANT |
| Privileged recovery/void | Employee requests reconciliation or void | `TestFiscalDocumentRoleActionMatrixAndSupersession` + `TestFiscalHandler_ManagerAdminOnlyLegalActions` + FE employee hide | ✅ COMPLIANT |
| Privileged recovery/void | Void succeeds | domain + mock void + worker void path | ✅ COMPLIANT |
| Legacy isolation | Upgrade with historical invoices | `TestFiscalMigrationLegacyIsolationAndSchema` + FE legacy_unfiscalized badge | ✅ COMPLIANT |
| Additive APIs/UI | Existing invoice contract remains stable | `TestInvoiceHandler_LegacyContractSnapshot` + FE issued-invoices create/edit columns preserved + operational patch without finalize | ✅ COMPLIANT |
| Additive APIs/UI | Owning client reads fiscal status | `TestFiscalHandler_ClientOwnOnlyProjectionAndArtifact` (API); **client UI deferred WU9** | ✅ COMPLIANT |
| Additive APIs/UI | Protected invoice deletion | `TestFiscalRepositoryFinalizeAtomicAndProtectedDelete` + `TestInvoiceHandler_DeleteProtectedFiscalHistory_409` | ✅ COMPLIANT |

#### fiscal-provider-foundation (9 / 14)

| Requirement | Scenario | Test | Result |
|-------------|----------|------|--------|
| Gateway normalized | Provider becomes unavailable after finalization | `TestWorker_ReconcileUsesSameFrozenProviderAndKey` + domain provider fixation | ✅ COMPLIANT |
| Finalization atomic | Dispatch event persistence fails | `TestFinalizationServiceOutboxFailureRollsBack` + `TestFiscalRepositoryOutboxInsertFailureRollsBack` | ✅ COMPLIANT |
| Finalization atomic | Process stops after commit | `TestFiscalOutboxWorkerEndToEndIssueWithMock` + claimable durable events | ✅ COMPLIANT |
| Outbox crash-safe | Concurrent workers claim one event | `TestFiscalOutboxSkipLockedSingleClaim` | ✅ COMPLIANT |
| Outbox crash-safe | Worker crashes while leased | `TestFiscalOutboxLeaseExpiryRecoversWithoutBlindResubmit` + `TestWorker_StaleStartedAttemptBecomesUnknownWithoutResubmit` | ✅ COMPLIANT |
| Idempotent correlation | User double-clicks finalization | concurrent finalize PG + `TestFiscalHandler_StatusMappingsAndRepeatedFinalize` + FE `does not call finalize twice on double-click` | ✅ COMPLIANT |
| Failures classified | Rate limit response is definite | `TestMockProvider_ScenariosCoverValidationExpiredTransientRateLimitAmbiguousReconcileAndVoids` | ✅ COMPLIANT |
| Failures classified | Timeout may have followed issuance | `TestWorker_AmbiguousTimeoutNeverAutoResubmitsIssue` | ✅ COMPLIANT |
| Ambiguous reconciliation | Reconciliation finds an issued document | mock ambiguous→Reconcile matched + worker reconcile path | ✅ COMPLIANT |
| Ambiguous reconciliation | Reconciliation cannot establish a result | `TestMockProvider_InconclusiveReconcileRemainsUnmatchedWithoutBlindRetry` + `TestWorker_ReconcileInconclusiveRetainsUnknownWithoutIssue` | ✅ COMPLIANT |
| Attempts redacted | Provider returns a secret-bearing error | `TestRedactSecrets_StripsBearerAndTokens` + `TestSanitizeFiscalizationProjection_OmitsSecrets` + FE no secret text in unknown guidance | ✅ COMPLIANT |
| Outages isolate core | Provider is offline | Legacy invoice handler/service + FE operational edit independent of fiscal; full provider-health vs `/ready` isolation remains WU11 | ⚠️ PARTIAL |
| Deterministic mock | Same mock issuance is repeated | `TestMockProvider_FTAndFRIssuanceAreStableAndLabeled` + concurrent + reload | ✅ COMPLIANT |
| Deterministic mock | Production selects mock | `TestMockProvider_ProductionRejectsSelectionAndLegalClassification` | ✅ COMPLIANT |

#### fiscal-artifacts (5 / 8)

| Requirement | Scenario | Test | Result |
|-------------|----------|------|--------|
| Immutable PDF archive | Issuance returns a PDF | `TestLocalStore_PutImmutableCreateIfAbsentMatchingChecksum` + `TestArtifactService_ArchiveCreateIfAbsentAndOwnershipDownload` + PG artifact tests | ✅ COMPLIANT |
| Immutable PDF archive | Archived bytes are altered | `TestArtifactService_UnavailableCompromisedAndStaffRoles` + PG compromised path | ✅ COMPLIANT |
| No duplicate issuance | PDF retrieval fails after confirmed issuance | `TestArtifactService_FailedArchiveRetainsIssuedAndRecoverySkipsIssue` + worker recover_artifact → FetchArtifact only | ✅ COMPLIANT |
| Authz downloads | Owning client downloads PDF | Handler streaming covered by `TestFiscalHandler_ClientOwnOnlyProjectionAndArtifact`; FE `downloadArtifact` wired; **production `cmd/api` passes `nil` ArtifactService; projection omits `artifactId`** | ⚠️ PARTIAL |
| Authz downloads | Different client requests PDF | same ownership denial → HTTP 404 (non-enumeration) | ✅ COMPLIANT |
| Legal vs mock | Developer downloads mock PDF | labeled mock PDF bytes (`SEM VALIDADE FISCAL — MOCK`) + FE `shows mock label outside production when classification is mock`; live projection still omits classification (WARNING) | ✅ COMPLIANT |
| Production readiness | Artifact backend is not production-ready | `TestProductionReadiness_FailClosedUntilObjectBackendPasses` + fiscal-worker production compose gate | ✅ COMPLIANT |
| Production readiness | New issuance is disabled | `TestArtifactService_ReadWhileIssuanceDisabled` | ✅ COMPLIANT |

#### cloudware-fiscal-enablement (6 / 12)

| Requirement | Scenario | Test | Result |
|-------------|----------|------|--------|
| Scoped connections | Manager starts connection setup | — | ❌ UNTESTED (deferred WU10) |
| Scoped connections | Employee calls connection endpoint | `TestFiscalIntegrationHandler_EmployeeLegalActionsForbidden` covers early 403 surface only; full Cloudware OAuth connect flow deferred | ❌ UNTESTED (deferred WU10) |
| OAuth protected | OAuth callback succeeds | — | ❌ UNTESTED (deferred WU10) |
| OAuth protected | Callback state is invalid | — | ❌ UNTESTED (deferred WU10) |
| Evidenced FT/FR mapping | Frozen FR is mapped | — | ❌ UNTESTED (deferred WU10) |
| Evidenced FT/FR mapping | Unsupported receipt workflow is requested | — | ❌ UNTESTED (deferred WU10) |
| Evidence-gated mutation | One applicable gate lacks evidence | — | ❌ UNTESTED (deferred WU10/WU12) |
| Evidence-gated mutation | A gate is not applicable | — | ❌ UNTESTED (deferred WU10/WU12) |
| No undocumented behavior | Provider times out before idempotency evidence exists | — | ❌ UNTESTED (deferred WU10) |
| No undocumented behavior | UI presents issued status | Staff FE `fiscalPresentationLabel` / badges / mock label — no AT/e-Fatura claims; **client my-invoices deferred WU9** | ⚠️ PARTIAL |
| Normalized errors | Refresh token fails | — | ❌ UNTESTED (deferred WU10) |
| Normalized errors | Active GC license is absent | — | ❌ UNTESTED (deferred WU10) |

**Compliance summary**: **44/58** scenarios ✅ COMPLIANT; **3** ⚠️ PARTIAL; **11** ❌ UNTESTED (all expected deferred to WU9+). Fully green requirements: **21/29**. Mid-change archive gate remains **FAIL** (38/58 tasks). WU8 closed staff UI deferrals for draft/actions/list badges; PDF end-to-end remains PARTIAL (nil ArtifactService + missing projection `artifactId`).

### WU8 staff UI attention

| Design / task expectation | Evidence | Result |
|---------------------------|----------|--------|
| Batch summaries on list (`Fiscalização` column) | `page.test.tsx` loads summaries + legacy guidance; `listSummaries` service test | ✅ |
| `FiscalDraftForm` FT/FR decimals, exempt, source refs, readiness | `FiscalDraftForm.test.tsx` (5) | ✅ |
| `FiscalActions` role gate, 202 polling, cancel, double-submit, 409, mock label | `FiscalActions.test.tsx` (8) | ✅ |
| Operational create/edit preserved | list/detail page tests | ✅ |
| Navigation only to existing pages | Links/replace → `/accounting`, `/accounting/issued-invoices`, `/accounting/issued-invoices/[id]` (all have `page.tsx`) | ✅ |
| Provider-neutral pt_PT copy | labels via `fiscalPresentationLabel`; no Cloudware/AT claims in UI strings | ✅ |
| PDF download usable end-to-end | UI wired; API composition nil + projection lacks `artifactId` | ⚠️ PARTIAL |

### Correctness (Static Evidence — WU1–WU8)

| Requirement area | Status | Notes |
|------------------|--------|-------|
| Migration 011 + legacy isolation | ✅ Implemented | PG tests green; `VerifyFiscalSchema` fail-fast |
| Exact decimal + policy calculator | ✅ Implemented | shopspring/decimal; no float64 in fiscal calc |
| Lifecycle domain (11 states) | ✅ Implemented | transitions, role matrix, freeze invariants |
| Aggregate persistence / finalize+outbox | ✅ Implemented | atomic finalize, concurrent one-intent, delete protection |
| Provider-neutral ports + mock | ✅ Implemented | registry-keyed scenarios; prod guards; labeled mock PDFs |
| Outbox worker lease/crash protocol | ✅ Implemented | SKIP LOCKED, lease fencing, started-before-call, stale→unknown |
| Immutable artifact archive + recovery | ✅ Implemented | LocalStore; ArtifactService; PG metadata/access log |
| Additive fiscal HTTP API | ✅ Implemented | Nested routes; summaries before `/:id`; 403/404/409/422/202 |
| Staff fiscal UI (WU8) | ✅ Implemented | types/service + list column + detail draft/actions; 151 FE tests green |
| API ArtifactService composition | ⚠️ Partial | `cmd/api/main.go` → `NewDocumentService(..., nil, ...)`; projection has `artifactStatus` only (no `artifactId` / classification) |
| Client/admin UI / Cloudware enablement | ❌ Not in scope yet | WU9–WU12 pending |

### Coherence (Design — completed units)

| Decision | Followed? | Notes |
|----------|-----------|-------|
| FiscalDocument aggregate separate from Invoice | ✅ Yes | eligibility column; draft/finalize services |
| Exact decimal + unapproved policy fail-closed | ✅ Yes | domain package; HTTP rejects JSON numbers |
| Finalization + initial outbox one transaction | ✅ Yes | service + PG rollback tests |
| Gateway Issue/Reconcile/Void/Fetch/Verify | ✅ Yes | ports + mock |
| Deterministic mock via injected registry | ✅ Yes | no magic identity fields |
| Production rejects mock / legal mock classification | ✅ Yes | `guard.go` + `cmd/api/main.go` |
| Worker SKIP LOCKED + lease-token/owner fencing | ✅ Yes | outbox repository ClaimNext + Complete fencing |
| Crashed started mutation → unknown, no blind Issue | ✅ Yes | RecoverStaleAttempt + worker crash tests |
| Private immutable FiscalArtifactStore | ✅ Yes | ports + `local.go`; worker composed |
| Issued independent of artifact; recovery never Issue | ✅ Yes | FailedArchiveRetainsIssued; RecoverFromProvider uses FetchArtifact only |
| Additive HTTP under `/api/v1`; legacy invoice contracts unchanged | ✅ Yes | legacy snapshot tests |
| Static `/fiscalization-summaries` before `/:id` | ✅ Yes | registration order tests |
| Staff UI on issued-invoices list/detail | ✅ Yes | design paths match; create modal preserved |
| Navigation to existing issued-invoices pages only | ✅ Yes | no invented `/edit` routes |
| 202 → capped polling; double-submit guard | ✅ Yes | FiscalActions + tests |
| API PDF streaming via composed ArtifactService | ⚠️ Deviation | Handler path exists; API composition leaves ArtifactService nil; projection omits artifactId |
| `FiscalProviderGateway` type name | ⚠️ Deviation | Retained `FiscalProvider` port (apply-progress); behavior matches |
| Client my-invoices / admin integrations / OAuth | ➖ Deferred | WU9–WU10 |

### Issues Found

**CRITICAL**:
1. Change-level incompleteness: **20/58 tasks pending** — archive gate must remain blocked.

**WARNING**:
1. Production API `ArtifactService` is `nil` (`cmd/api/main.go`). PDF handlers exist and unit-test with stubs, but real API downloads return unavailable until store is composed for the API process.
2. `FiscalizationProjection` emits `artifactStatus` only — no `artifactId` (and no `artifactClassification`), so staff PDF download and live mock badges cannot complete end-to-end despite UI wiring/tests with injected fields.
3. `go test -race` blocked locally (CGO/`gcc` missing on Windows agent). Non-race tests green; CI must run race.
4. `gentle-ai sdd-verify-validate` unavailable — cannot machine-admit report bytes (parent still required OpenSpec+Engram persistence).
5. Design deviations retained: port naming `FiscalProvider` vs design `FiscalProviderGateway`; production ObjectStore readiness-only.
6. WU8 authored size ~1200–1800 lines (`size:exception`) above preferred 600-line manual slice (documented in apply-progress).
7. Provider-offline scenario remains PARTIAL pending WU11 readiness isolation breadth.

**SUGGESTION**:
1. Continue `/sdd-apply` at **WU9** (client status/PDF + admin integration UI). Optionally wire API ArtifactService + projection `artifactId`/`classification` before or during a later ops slice if staff PDF UX is needed sooner.
2. Do **not** archive until 58/58 tasks complete and a full verify PASS with validator admission when tooling is available.
3. Ensure CI Linux runners execute `go test -race` for fiscal packages.

### Verdict

**FAIL**

Change-level verification cannot PASS while 20 tasks remain. WU1–WU8 focused correctness is green (backend unit + Fiscal* PostgreSQL + frontend 151 tests + lint/typecheck exit 0; `cmd/api` and `cmd/fiscal-worker` build). Honest PARTIAL remains for production API PDF downloads and live mock presentation (nil ArtifactService; projection omits id/classification). Proceed to WU9 apply. **Not archive-ready.**

### Verification scope note

This is an honest **mid-change** verify requested after WU1–WU8. It is **not** a claim that the OpenSpec change is complete. Native `nextRecommended` remaining `apply` with 20 pending tasks is expected and correct.
