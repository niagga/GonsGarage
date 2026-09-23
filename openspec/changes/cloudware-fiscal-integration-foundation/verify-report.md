```yaml
schema: gentle-ai.verify-result/v1
evidence_revision: sha256:de6bb928b108077567f5e077eb9e9de96f4d87cdb66c04c4d750b1d3c685c85c
verdict: fail
blockers: 1
critical_findings: 1
requirements: 16/29
scenarios: 35/58
remediation_note: "2026-09-23 remediated 3 covering tests remain COMPLIANT; WU5 outbox scenarios now covered"
test_command: cd backend && go test ./internal/domain/... ./internal/service/fiscal/... ./internal/integration/fiscal/mock/... ./internal/core/ports/... -count=1 ; FISCAL_TEST_DATABASE_URL=<redacted> go test ./tests/integration/ -count=1 -run Fiscal(Migration|Repository|Mock|Outbox) -timeout 180s
test_exit_code: 0
test_output_hash: sha256:0423cce5e1140b4b6b083d036562f1cb56d44b70567386018bf29e136a9d0fd6
build_command: cd backend && go vet ./internal/domain/... ./internal/service/fiscal/... ./internal/integration/fiscal/mock/... ./internal/core/ports/... ./internal/repository/postgres/... ; go test -c -o NUL ./internal/domain/ ./internal/service/fiscal/ ./internal/integration/fiscal/mock/ ; go build -o NUL ./cmd/fiscal-worker/
build_exit_code: 0
build_output_hash: sha256:2e69bf22c3dceee1b4939615c03d112aaf69dda09bfc1fb03140e5cfc6295119
```

## Verification Report

**Change**: cloudware-fiscal-integration-foundation
**Version**: N/A (OpenSpec change; four delta specs)
**Mode**: Standard (focused mid-change verify after WU1–WU5; full suite not claimed)
**Artifact store**: openspec
**Validator**: `gentle-ai sdd-verify-validate` **unavailable** on gentle-ai 3.5.0 (command not present; `sdd-attempt grant` is edit-authority only). Report persisted per parent Persistence instruction; admission tooling could not attest bytes.

### Completeness

| Metric | Value |
|--------|-------|
| Tasks total | 58 |
| Tasks complete | 26 (1.1–1.5, 2.1–2.8, 3.1–3.4, 4.1–4.4, 5.1–5.4, 13.1) |
| Tasks incomplete | 32 (WU6–WU12 implementation + parent 13.2–13.5) |
| Change-level archive gate | **FAIL** — `allComplete: false` |
| WU1–WU5 local finish claims | Satisfied in `tasks.md` / `apply-progress.md` checkboxes |

Scope of this verify: **completed work units only** (schema, domain, persistence, mock, outbox worker). Remaining WU6–WU12 scenarios are expected deferred, not false PASS.

### Build & Tests Execution

**Build**: ✅ Passed (focused packages + fiscal-worker)
```text
cd backend
go vet ./internal/domain/... ./internal/service/fiscal/... ./internal/integration/fiscal/mock/... ./internal/core/ports/... ./internal/repository/postgres/...
→ VET_EXIT:0
go test -c -o NUL ./internal/domain/ ; ./internal/service/fiscal/ ; ./internal/integration/fiscal/mock/
→ BUILD_DOMAIN:0 BUILD_SVC:0 BUILD_MOCK:0
go build -o NUL ./cmd/fiscal-worker/
→ BUILD_WORKER:0
build_output_hash: sha256:2e69bf22c3dceee1b4939615c03d112aaf69dda09bfc1fb03140e5cfc6295119
```

**Tests**: ✅ Focused packages passed / ⚠️ race unavailable
```text
# Unit / service / mock / ports (CGO_ENABLED=0)
go test ./internal/domain/... ./internal/service/fiscal/... ./internal/integration/fiscal/mock/... ./internal/core/ports/... -count=1
→ ok domain, service/fiscal (Worker_* + draft/finalization), integration/fiscal/mock, core/ports (exit 0)

# PostgreSQL integration (FISCAL_TEST_DATABASE_URL set privately from local DATABASE_URL)
go test ./tests/integration/ -count=1 -run 'Fiscal(Migration|Repository|Mock|Outbox)' -timeout 180s
→ ok tests/integration 13.209s (exit 0)
  covers FiscalMigration*, FiscalRepository*, FiscalMock*, FiscalOutbox*

test_output_hash: sha256:0423cce5e1140b4b6b083d036562f1cb56d44b70567386018bf29e136a9d0fd6

# Race detector (documented limit)
CGO_ENABLED=1 go test ./internal/domain/ -count=1 -race
→ FAIL build: cgo: C compiler "gcc" not found
→ Not treated as scenario FAIL; CI/Linux must run -race. WARNING only.
```

**Coverage**: ➖ Not measured this run (no coverage threshold enforced in focused verify)

### Spec Compliance Matrix

Authoritative totals from retrieved specs: **29 requirements**, **58 scenarios**.

Legend for deferred rows: `UNTESTED (deferred WUn)` = expected not yet implemented; does not prove WU1–WU5 incorrect.

#### fiscal-documents (9 requirements / 24 scenarios)

| Requirement | Scenario | Test | Result |
|-------------|----------|------|--------|
| Fiscal drafts distinct | Employee prepares an FT draft | `draft_finalization_test.go` > `TestDraftServiceEmployeeCRUDAndClientDenial` | ✅ COMPLIANT |
| Fiscal drafts distinct | Client attempts draft preparation | same | ✅ COMPLIANT |
| Fiscal drafts distinct | Optional source record later changes | `fiscal_policy_test.go` > `TestCalculate_SourceReferenceSurvivesAndCanonicalIgnoresExternalMutation` | ✅ COMPLIANT |
| Fiscal drafts distinct | Unsupported document kind | `TestDraftServiceRejectsUnsupportedDocumentKind` (2026-09-23 remediation) | ✅ COMPLIANT |
| Complete fiscal data | Required fiscal identity is incomplete | `fiscal_policy_test.go` > `TestPolicyVersionResolve_ApprovalGateAndImmutability` + finalization readiness paths | ✅ COMPLIANT |
| Complete fiscal data | Legacy floating-point amount is supplied | `TestDraftServiceStoresCanonicalDecimalsIndependentOfLegacyFloatAmount` (2026-09-23 remediation) | ✅ COMPLIANT |
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
| Privileged recovery/void | Employee requests reconciliation or void | `TestFiscalDocumentRoleActionMatrixAndSupersession` | ✅ COMPLIANT |
| Privileged recovery/void | Void succeeds | domain + mock void + `TestWorker_VoidDispatchAndArtifactRecoveryPaths` | ✅ COMPLIANT |
| Legacy isolation | Upgrade with historical invoices | `TestFiscalMigrationLegacyIsolationAndSchema` | ✅ COMPLIANT |
| Additive APIs/UI | Existing invoice contract remains stable | invoice service regression retained; **HTTP snapshot is WU7** | ⚠️ PARTIAL |
| Additive APIs/UI | Owning client reads fiscal status | — | ❌ UNTESTED (deferred WU7/WU9) |
| Additive APIs/UI | Protected invoice deletion | `TestFiscalRepositoryFinalizeAtomicAndProtectedDelete` | ✅ COMPLIANT |

#### fiscal-provider-foundation (9 / 14)

| Requirement | Scenario | Test | Result |
|-------------|----------|------|--------|
| Gateway normalized | Provider becomes unavailable after finalization | `TestWorker_ReconcileUsesSameFrozenProviderAndKey` + domain provider fixation | ✅ COMPLIANT |
| Finalization atomic | Dispatch event persistence fails | `TestFinalizationServiceOutboxFailureRollsBack` + `TestFiscalRepositoryOutboxInsertFailureRollsBack` | ✅ COMPLIANT |
| Finalization atomic | Process stops after commit | `TestFiscalOutboxWorkerEndToEndIssueWithMock` + claimable durable events | ✅ COMPLIANT |
| Outbox crash-safe | Concurrent workers claim one event | `TestFiscalOutboxSkipLockedSingleClaim` | ✅ COMPLIANT |
| Outbox crash-safe | Worker crashes while leased | `TestFiscalOutboxLeaseExpiryRecoversWithoutBlindResubmit` + `TestWorker_StaleStartedAttemptBecomesUnknownWithoutResubmit` | ✅ COMPLIANT |
| Idempotent correlation | User double-clicks finalization | concurrent finalize PG + service idempotency | ✅ COMPLIANT |
| Failures classified | Rate limit response is definite | `TestMockProvider_ScenariosCoverValidationExpiredTransientRateLimitAmbiguousReconcileAndVoids` | ✅ COMPLIANT |
| Failures classified | Timeout may have followed issuance | `TestWorker_AmbiguousTimeoutNeverAutoResubmitsIssue` | ✅ COMPLIANT |
| Ambiguous reconciliation | Reconciliation finds an issued document | mock ambiguous→Reconcile matched + worker reconcile path | ✅ COMPLIANT |
| Ambiguous reconciliation | Reconciliation cannot establish a result | `TestMockProvider_InconclusiveReconcileRemainsUnmatchedWithoutBlindRetry` (2026-09-23) + `TestWorker_ReconcileInconclusiveRetainsUnknownWithoutIssue` | ✅ COMPLIANT |
| Attempts redacted | Provider returns a secret-bearing error | `TestRedactSecrets_StripsBearerAndTokens` | ✅ COMPLIANT |
| Outages isolate core | Provider is offline | `TestConnectionServiceProviderFailureFailsClosed` (partial surface; full invoice isolation WU7/WU11) | ⚠️ PARTIAL |
| Deterministic mock | Same mock issuance is repeated | `TestMockProvider_FTAndFRIssuanceAreStableAndLabeled` + concurrent + reload | ✅ COMPLIANT |
| Deterministic mock | Production selects mock | `TestMockProvider_ProductionRejectsSelectionAndLegalClassification` | ✅ COMPLIANT |

#### fiscal-artifacts (5 / 8)

| Requirement | Scenario | Test | Result |
|-------------|----------|------|--------|
| Immutable PDF archive | Issuance returns a PDF | mock deterministic PDF bytes; **archive store WU6** | ⚠️ PARTIAL |
| Immutable PDF archive | Archived bytes are altered | — | ❌ UNTESTED (deferred WU6) |
| No duplicate issuance | PDF retrieval fails after confirmed issuance | worker `recover_artifact` path exists; **immutable archive WU6** | ❌ UNTESTED (deferred WU6) |
| Authz downloads | Owning client downloads PDF | — | ❌ UNTESTED (deferred WU6/WU7) |
| Authz downloads | Different client requests PDF | — | ❌ UNTESTED (deferred WU6/WU7) |
| Legal vs mock | Developer downloads mock PDF | labeled PDF generation (`SEM VALIDADE FISCAL — MOCK`); download API deferred | ⚠️ PARTIAL |
| Production readiness | Artifact backend is not production-ready | — | ❌ UNTESTED (deferred WU6/WU11) |
| Production readiness | New issuance is disabled | — | ❌ UNTESTED (deferred WU11) |

#### cloudware-fiscal-enablement (6 / 12)

| Requirement | Scenario | Test | Result |
|-------------|----------|------|--------|
| Scoped connections | Manager starts connection setup | — | ❌ UNTESTED (deferred WU10) |
| Scoped connections | Employee calls connection endpoint | — | ❌ UNTESTED (deferred WU10) |
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

**Compliance summary**: **35/58** scenarios ✅ COMPLIANT; **4** ⚠️ PARTIAL; **19** ❌ UNTESTED (all expected deferred to later WUs). Fully green requirements: **16/29**. Mid-change archive gate remains **FAIL** (26/58 tasks). Remediations from 2026-09-23 remain green.

### Correctness (Static Evidence — WU1–WU5)

| Requirement area | Status | Notes |
|------------------|--------|-------|
| Migration 011 + legacy isolation | ✅ Implemented | PG tests green; `VerifyFiscalSchema` fail-fast |
| Exact decimal + policy calculator | ✅ Implemented | shopspring/decimal; no float64 in fiscal calc |
| Lifecycle domain (11 states) | ✅ Implemented | transitions, role matrix, freeze invariants |
| Aggregate persistence / finalize+outbox | ✅ Implemented | atomic finalize, concurrent one-intent, delete protection |
| Provider-neutral ports + mock | ✅ Implemented | registry-keyed scenarios; prod guards; labeled mock PDFs |
| Outbox worker lease/crash protocol | ✅ Implemented | SKIP LOCKED, lease fencing, started-before-call, stale→unknown, 4-tx boundaries, `cmd/fiscal-worker` |
| Artifact archive / HTTP / UI / Cloudware | ❌ Not in scope yet | WU6–WU12 pending |

### Coherence (Design — completed units)

| Decision | Followed? | Notes |
|----------|-----------|-------|
| FiscalDocument aggregate separate from Invoice | ✅ Yes | eligibility column; draft/finalize services |
| Exact decimal + unapproved policy fail-closed | ✅ Yes | domain package |
| Finalization + initial outbox one transaction | ✅ Yes | service + PG rollback tests |
| Gateway Issue/Reconcile/Void/Fetch/Verify | ✅ Yes | ports + mock |
| Deterministic mock via injected registry | ✅ Yes | no magic identity fields |
| Production rejects mock / legal mock classification | ✅ Yes | `guard.go` + `cmd/api/main.go` |
| Worker SKIP LOCKED + lease-token/owner fencing | ✅ Yes | `fiscal_outbox_repository.go` ClaimNext + Complete fencing |
| Crashed started mutation → unknown, no blind Issue | ✅ Yes | RecoverStaleAttempt + worker crash tests |
| Separate claim / pre-call / gateway / completion txs | ✅ Yes | documented on `Worker`; unit coverage |
| Graceful worker shutdown | ✅ Yes | `TestWorker_GracefulShutdownStopsNewClaims` + SIGINT/SIGTERM in cmd |
| `FiscalProviderGateway` type name | ⚠️ Deviation | Retained `FiscalProvider` port (apply-progress); behavior matches |
| PDF bytes only checksum in PG mock table | ⚠️ Deviation | regenerate from canonical; checksum enforced |
| Artifact recovery persists PDF archive | ➖ Deferred | WU5 completes event only; WU6 owns immutable store |
| Private artifact store / OAuth / Cloudware gates | ➖ Deferred | WU6/WU10 |

### Issues Found

**CRITICAL**:
1. Change-level incompleteness: **32/58 tasks pending** — archive gate must remain blocked.

**WARNING**:
1. `go test -race` blocked locally (CGO/`gcc` missing on Windows agent). Non-race tests green; CI must run race.
2. `gentle-ai sdd-verify-validate` unavailable on installed 3.5.0 — cannot machine-admit report bytes (parent still required OpenSpec+Engram persistence).
3. Design deviations: port naming `FiscalProvider` vs design `FiscalProviderGateway`; mock PDF regen-from-canonical when row stores sha only; artifact recovery without byte persistence until WU6.
4. WU5 authored size ~1500 lines (`size:exception`) above preferred 600-line manual slice (documented in apply-progress).
5. Four PARTIAL scenarios remain in completed surface (invoice HTTP stability, provider-offline breadth, mock PDF archive/download).

**SUGGESTION**:
1. Continue `/sdd-apply` at **WU6** (immutable artifact archive).
2. Do **not** archive until 58/58 tasks complete and a full verify PASS with validator admission when tooling is available.
3. Ensure CI Linux runners execute `go test -race` for fiscal packages.

### Verdict

**FAIL**

Change-level verification cannot PASS while 32 tasks remain. WU1–WU5 focused correctness is green (unit + PostgreSQL FiscalMigration/FiscalRepository/FiscalMock/FiscalOutbox exit 0); WU5 outbox/worker scenarios are COMPLIANT. Proceed to WU6 apply. **Not archive-ready.**

### Verification scope note

This is an honest **mid-change** verify requested after WU1–WU5. It is **not** a claim that the OpenSpec change is complete. Native `nextRecommended` remaining `apply` with 32 pending tasks is expected and correct.
