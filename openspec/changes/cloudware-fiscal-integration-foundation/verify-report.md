```yaml
schema: gentle-ai.verify-result/v1
evidence_revision: sha256:bbe6720cadc33ce2097ad0c895d246098ad870601e29bcd43ed3f715a131520a
verdict: fail
blockers: 1
critical_findings: 1
requirements: 21/29
scenarios: 44/58
remediation_note: "2026-09-23 mid-change post-WU7: additive fiscal HTTP API + legacy invoice regression green; API ArtifactService remains nil (PDF download PARTIAL); WU8+ UI and WU10+ Cloudware deferred UNTESTED; archive gate FAIL at 34/58"
test_command: cd backend && go test ./internal/domain/... ./internal/service/fiscal/... ./internal/service/invoice/... ./internal/integration/fiscal/mock/... ./internal/platform/fiscalartifact/... ./internal/handler/... ./internal/core/ports/... -count=1 ; FISCAL_TEST_DATABASE_URL=<redacted> go test ./tests/integration/ -count=1 -run Fiscal(Migration|Repository|Mock|Outbox|Artifact) -timeout 180s
test_exit_code: 0
test_output_hash: sha256:d289161f9b2e06309e57c080cfce343b31c0a1a6b184ceb4659e6c57f5a409ee
build_command: cd backend && go vet ./internal/domain/... ./internal/service/fiscal/... ./internal/service/invoice/... ./internal/integration/fiscal/mock/... ./internal/platform/fiscalartifact/... ./internal/handler/... ./internal/core/ports/... ./internal/repository/postgres/... ; go build -o NUL ./cmd/api/ ; go build -o NUL ./cmd/fiscal-worker/
build_exit_code: 0
build_output_hash: sha256:244ed73bd4da9df3b4e03508fff914ce25e8275f16fe9a9ce599f58cc03304cb
```

## Verification Report

**Change**: cloudware-fiscal-integration-foundation
**Version**: N/A (OpenSpec change; four delta specs)
**Mode**: Standard (focused mid-change verify after WU1–WU7; full suite not claimed)
**Artifact store**: openspec
**Validator**: `gentle-ai sdd-verify-validate` **unavailable** on installed gentle-ai (command not present; only `sdd-status` / `sdd-continue` / `sdd-attempt grant`). Report persisted per parent Persistence instruction; admission tooling could not attest bytes.

### Completeness

| Metric | Value |
|--------|-------|
| Tasks total | 58 |
| Tasks complete | 34 (1.1–1.5, 2.1–2.8, 3.1–3.4, 4.1–4.4, 5.1–5.4, 6.1–6.4, 7.1–7.4, 13.1) |
| Tasks incomplete | 24 (WU8–WU12 implementation + parent 13.2–13.5) |
| Change-level archive gate | **FAIL** — `allComplete: false` |
| WU1–WU7 local finish claims | Satisfied in `tasks.md` / `apply-progress.md` checkboxes |

Scope of this verify: **completed work units only** (schema, domain, persistence, mock, outbox worker, artifact archive, additive HTTP API). Remaining WU8–WU12 scenarios are expected deferred, not false PASS.

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
build_output_hash: sha256:244ed73bd4da9df3b4e03508fff914ce25e8275f16fe9a9ce599f58cc03304cb
```

**Tests**: ✅ Focused packages + Fiscal* PostgreSQL passed / ⚠️ race unavailable
```text
# Unit / service / mock / platform / handler / invoice / ports (CGO_ENABLED=0)
go test ./internal/domain/... ./internal/service/fiscal/... ./internal/service/invoice/... ./internal/integration/fiscal/mock/... ./internal/platform/fiscalartifact/... ./internal/handler/... ./internal/core/ports/... -count=1
→ ok domain, service/fiscal, service/invoice, integration/fiscal/mock, platform/fiscalartifact, handler, core/ports (exit 0)

# PostgreSQL integration (FISCAL_TEST_DATABASE_URL loaded privately from DATABASE_URL)
go test ./tests/integration/ -count=1 -run 'Fiscal(Migration|Repository|Mock|Outbox|Artifact)' -timeout 180s
→ ok tests/integration 30.007s (exit 0)

test_output_hash: sha256:d289161f9b2e06309e57c080cfce343b31c0a1a6b184ceb4659e6c57f5a409ee

# Race detector (documented limit)
CGO_ENABLED=1 go test ./internal/domain/ -count=1 -race
→ FAIL build: cgo: C compiler "gcc" not found
→ Not treated as scenario FAIL; CI/Linux must run -race. WARNING only.
```

**Coverage**: ➖ Not measured this run (no coverage threshold enforced in focused verify)

### Spec Compliance Matrix

Authoritative totals from retrieved specs: **29 requirements**, **58 scenarios**.

Legend for deferred rows: `UNTESTED (deferred WUn)` = expected not yet implemented; does not prove WU1–WU7 incorrect.

#### fiscal-documents (9 requirements / 24 scenarios)

| Requirement | Scenario | Test | Result |
|-------------|----------|------|--------|
| Fiscal drafts distinct | Employee prepares an FT draft | `draft_finalization_test.go` > `TestDraftServiceEmployeeCRUDAndClientDenial` | ✅ COMPLIANT |
| Fiscal drafts distinct | Client attempts draft preparation | same | ✅ COMPLIANT |
| Fiscal drafts distinct | Optional source record later changes | `fiscal_policy_test.go` > `TestCalculate_SourceReferenceSurvivesAndCanonicalIgnoresExternalMutation` | ✅ COMPLIANT |
| Fiscal drafts distinct | Unsupported document kind | `TestDraftServiceRejectsUnsupportedDocumentKind` | ✅ COMPLIANT |
| Complete fiscal data | Required fiscal identity is incomplete | `fiscal_policy_test.go` > `TestPolicyVersionResolve_ApprovalGateAndImmutability` + finalization readiness paths | ✅ COMPLIANT |
| Complete fiscal data | Legacy floating-point amount is supplied | `TestDraftServiceStoresCanonicalDecimalsIndependentOfLegacyFloatAmount` | ✅ COMPLIANT |
| Deterministic arithmetic | Arithmetic policy is unavailable | `TestPolicyVersionResolve_ApprovalGateAndImmutability` | ✅ COMPLIANT |
| Deterministic arithmetic | Totals reconcile | `TestCalculate_*` | ✅ COMPLIANT |
| Deterministic arithmetic | Submitted totals do not reconcile | `TestCalculate_DocumentRoundingAdjustmentAndDeclaredMismatch` | ✅ COMPLIANT |
| Explicit finalization | Manager finalizes an eligible draft | `TestFinalizationServiceAtomicityAndAuth` + `TestFiscalRepositoryFinalizeAtomicAndProtectedDelete` | ✅ COMPLIANT |
| Explicit finalization | Employee attempts finalization | `TestFinalizationServiceAtomicityAndAuth` | ✅ COMPLIANT |
| Explicit finalization | Two finalization requests race | `TestFiscalRepositoryConcurrentFinalizationOneIntent` | ✅ COMPLIANT |
| Frozen immutable | Customer or invoice changes after finalization | domain freeze + `TestFiscalMigrationConstraintsAndImmutability` | ✅ COMPLIANT |
| Frozen immutable | Staff attempts to edit frozen lines | `TestFiscalMigrationConstraintsAndImmutability` | ✅ COMPLIANT |
| Lifecycle guarded | Definite validation rejection | `TestFiscalDocumentFinalizeFreezeAndTransitions` | ✅ COMPLIANT |
| Lifecycle guarded | Ambiguous issuance result | `TestFiscalDocumentUnknownRetryProhibitionAndFailedVoidReturnsIssued` | ✅ COMPLIANT |
| Lifecycle guarded | Definite void failure | same | ✅ COMPLIANT |
| Privileged recovery/void | Manager retries after connection recovery | domain transitions + role matrix | ✅ COMPLIANT |
| Privileged recovery/void | Employee requests reconciliation or void | `TestFiscalDocumentRoleActionMatrixAndSupersession` + `TestFiscalHandler_ManagerAdminOnlyLegalActions` | ✅ COMPLIANT |
| Privileged recovery/void | Void succeeds | domain + mock void + `TestWorker_VoidDispatchAndArtifactRecoveryPaths` | ✅ COMPLIANT |
| Legacy isolation | Upgrade with historical invoices | `TestFiscalMigrationLegacyIsolationAndSchema` | ✅ COMPLIANT |
| Additive APIs/UI | Existing invoice contract remains stable | `invoice_handler_test.go` > `TestInvoiceHandler_LegacyContractSnapshot` + `p1_accounting_routes_test.go` + invoice service regression | ✅ COMPLIANT |
| Additive APIs/UI | Owning client reads fiscal status | `fiscal_handler_test.go` > `TestFiscalHandler_ClientOwnOnlyProjectionAndArtifact` | ✅ COMPLIANT |
| Additive APIs/UI | Protected invoice deletion | `TestFiscalRepositoryFinalizeAtomicAndProtectedDelete` + `TestInvoiceHandler_DeleteProtectedFiscalHistory_409` | ✅ COMPLIANT |

#### fiscal-provider-foundation (9 / 14)

| Requirement | Scenario | Test | Result |
|-------------|----------|------|--------|
| Gateway normalized | Provider becomes unavailable after finalization | `TestWorker_ReconcileUsesSameFrozenProviderAndKey` + domain provider fixation | ✅ COMPLIANT |
| Finalization atomic | Dispatch event persistence fails | `TestFinalizationServiceOutboxFailureRollsBack` + `TestFiscalRepositoryOutboxInsertFailureRollsBack` | ✅ COMPLIANT |
| Finalization atomic | Process stops after commit | `TestFiscalOutboxWorkerEndToEndIssueWithMock` + claimable durable events | ✅ COMPLIANT |
| Outbox crash-safe | Concurrent workers claim one event | `TestFiscalOutboxSkipLockedSingleClaim` | ✅ COMPLIANT |
| Outbox crash-safe | Worker crashes while leased | `TestFiscalOutboxLeaseExpiryRecoversWithoutBlindResubmit` + `TestWorker_StaleStartedAttemptBecomesUnknownWithoutResubmit` | ✅ COMPLIANT |
| Idempotent correlation | User double-clicks finalization | concurrent finalize PG + `TestFiscalHandler_StatusMappingsAndRepeatedFinalize` | ✅ COMPLIANT |
| Failures classified | Rate limit response is definite | `TestMockProvider_ScenariosCoverValidationExpiredTransientRateLimitAmbiguousReconcileAndVoids` | ✅ COMPLIANT |
| Failures classified | Timeout may have followed issuance | `TestWorker_AmbiguousTimeoutNeverAutoResubmitsIssue` | ✅ COMPLIANT |
| Ambiguous reconciliation | Reconciliation finds an issued document | mock ambiguous→Reconcile matched + worker reconcile path | ✅ COMPLIANT |
| Ambiguous reconciliation | Reconciliation cannot establish a result | `TestMockProvider_InconclusiveReconcileRemainsUnmatchedWithoutBlindRetry` + `TestWorker_ReconcileInconclusiveRetainsUnknownWithoutIssue` | ✅ COMPLIANT |
| Attempts redacted | Provider returns a secret-bearing error | `TestRedactSecrets_StripsBearerAndTokens` + `TestSanitizeFiscalizationProjection_OmitsSecrets` | ✅ COMPLIANT |
| Outages isolate core | Provider is offline | Legacy invoice handler/service continue under existing contract (WU7 regression); full provider-health vs `/ready` isolation remains WU11 | ⚠️ PARTIAL |
| Deterministic mock | Same mock issuance is repeated | `TestMockProvider_FTAndFRIssuanceAreStableAndLabeled` + concurrent + reload | ✅ COMPLIANT |
| Deterministic mock | Production selects mock | `TestMockProvider_ProductionRejectsSelectionAndLegalClassification` | ✅ COMPLIANT |

#### fiscal-artifacts (5 / 8)

| Requirement | Scenario | Test | Result |
|-------------|----------|------|--------|
| Immutable PDF archive | Issuance returns a PDF | `TestLocalStore_PutImmutableCreateIfAbsentMatchingChecksum` + `TestArtifactService_ArchiveCreateIfAbsentAndOwnershipDownload` + `TestFiscalArtifactRepository_ArchiveMetadataAccessLogAndCompromised` | ✅ COMPLIANT |
| Immutable PDF archive | Archived bytes are altered | `TestArtifactService_UnavailableCompromisedAndStaffRoles` + PG compromised path | ✅ COMPLIANT |
| No duplicate issuance | PDF retrieval fails after confirmed issuance | `TestArtifactService_FailedArchiveRetainsIssuedAndRecoverySkipsIssue` + worker recover_artifact → `RecoverFromProvider` (FetchArtifact only) | ✅ COMPLIANT |
| Authz downloads | Owning client downloads PDF | Handler streaming covered by `TestFiscalHandler_ClientOwnOnlyProjectionAndArtifact`; **production `cmd/api` wires `NewDocumentService(..., nil /* ArtifactService */, ...)` so OpenArtifact returns unavailable until store composed for API** | ⚠️ PARTIAL |
| Authz downloads | Different client requests PDF | same ownership denial → HTTP 404 (non-enumeration) | ✅ COMPLIANT |
| Legal vs mock | Developer downloads mock PDF | labeled mock PDF (`SEM VALIDADE FISCAL — MOCK`) + OpenDownload; UI presentation deferred WU8 | ✅ COMPLIANT |
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
| No undocumented behavior | UI presents issued status | — | ❌ UNTESTED (deferred WU8/WU9) |
| Normalized errors | Refresh token fails | — | ❌ UNTESTED (deferred WU10) |
| Normalized errors | Active GC license is absent | — | ❌ UNTESTED (deferred WU10) |

**Compliance summary**: **44/58** scenarios ✅ COMPLIANT; **2** ⚠️ PARTIAL; **12** ❌ UNTESTED (all expected deferred to WU8+). Fully green requirements: **21/29**. Mid-change archive gate remains **FAIL** (34/58 tasks). WU7 closed prior HTTP deferrals for invoice contract stability and owning-client projection; production API PDF download remains PARTIAL.

### Correctness (Static Evidence — WU1–WU7)

| Requirement area | Status | Notes |
|------------------|--------|-------|
| Migration 011 + legacy isolation | ✅ Implemented | PG tests green; `VerifyFiscalSchema` fail-fast |
| Exact decimal + policy calculator | ✅ Implemented | shopspring/decimal; no float64 in fiscal calc |
| Lifecycle domain (11 states) | ✅ Implemented | transitions, role matrix, freeze invariants |
| Aggregate persistence / finalize+outbox | ✅ Implemented | atomic finalize, concurrent one-intent, delete protection |
| Provider-neutral ports + mock | ✅ Implemented | registry-keyed scenarios; prod guards; labeled mock PDFs |
| Outbox worker lease/crash protocol | ✅ Implemented | SKIP LOCKED, lease fencing, started-before-call, stale→unknown, `cmd/fiscal-worker` |
| Immutable artifact archive + recovery | ✅ Implemented | LocalStore PutImmutable; ArtifactService Archive/RecoverFromProvider/OpenDownload; PG metadata/access log |
| Additive fiscal HTTP API | ✅ Implemented | Nested draft/detail/finalize/retry/reconcile/void; summaries before `/:id`; strict decimal strings; 403/404/409/422/202; client own-only; manager/admin legal actions; swagger secret omission |
| API ArtifactService composition | ⚠️ Partial | `cmd/api/main.go` passes `nil` ArtifactService; worker has store; downloads fail closed as unavailable |
| Staff/client UI / Cloudware enablement | ❌ Not in scope yet | WU8–WU12 pending |

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
| Additive HTTP under `/api/v1`; legacy invoice contracts unchanged | ✅ Yes | `RegisterInvoiceAndFiscalRoutes`; legacy snapshot tests |
| Static `/fiscalization-summaries` before `/:id` | ✅ Yes | registration order + `TestFiscalHandler_SummaryRouteRegisteredBeforeID` |
| Non-owning client 404 (no enumeration) | ✅ Yes | handler + DocumentService ownership checks |
| Safe projections omit credentials/keys/URLs | ✅ Yes | `fiscal_response.go` + sanitize test + swagger models |
| API PDF streaming via composed ArtifactService | ⚠️ Deviation | Handler path exists; API composition leaves ArtifactService nil (documented apply-progress) |
| `FiscalProviderGateway` type name | ⚠️ Deviation | Retained `FiscalProvider` port (apply-progress); behavior matches |
| OAuth / Cloudware gates / staff UI | ➖ Deferred | WU8–WU10 |

### Issues Found

**CRITICAL**:
1. Change-level incompleteness: **24/58 tasks pending** — archive gate must remain blocked.

**WARNING**:
1. Production API `ArtifactService` is `nil` (`cmd/api/main.go` → `NewDocumentService(..., nil, ...)`). PDF download HTTP handlers exist and unit-test with stubs, but real API downloads return unavailable until local/object store is composed for the API process (worker already has store). Scenario marked PARTIAL.
2. `go test -race` blocked locally (CGO/`gcc` missing on Windows agent). Non-race tests green; CI must run race.
3. `gentle-ai sdd-verify-validate` unavailable — cannot machine-admit report bytes (parent still required OpenSpec+Engram persistence).
4. Design deviations retained: port naming `FiscalProvider` vs design `FiscalProviderGateway`; production ObjectStore readiness-only (no cloud SDK PutImmutable yet).
5. WU7 authored size ~1400–2000 lines (`size:exception`) above preferred 600-line manual slice (documented in apply-progress).
6. Provider-offline scenario remains PARTIAL pending WU11 readiness isolation breadth.

**SUGGESTION**:
1. Continue `/sdd-apply` at **WU8** (staff fiscal UI), or first wire API ArtifactService if PDF downloads are needed before UI.
2. Do **not** archive until 58/58 tasks complete and a full verify PASS with validator admission when tooling is available.
3. Ensure CI Linux runners execute `go test -race` for fiscal packages.

### Verdict

**FAIL**

Change-level verification cannot PASS while 24 tasks remain. WU1–WU7 focused correctness is green (unit + handler + invoice + PostgreSQL FiscalMigration/FiscalRepository/FiscalMock/FiscalOutbox/FiscalArtifact exit 0; `cmd/api` and `cmd/fiscal-worker` build). Honest PARTIAL remains for production API PDF downloads (nil ArtifactService). Proceed to WU8 apply. **Not archive-ready.**

### Verification scope note

This is an honest **mid-change** verify requested after WU1–WU7. It is **not** a claim that the OpenSpec change is complete. Native `nextRecommended` remaining `apply` with 24 pending tasks is expected and correct.
