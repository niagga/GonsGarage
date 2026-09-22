```yaml
schema: gentle-ai.verify-result/v1
evidence_revision: sha256:6b5dbb9332520789cb53fc09993d24d60f7b9501a42ef442989feb70b3e9e9fa
verdict: fail
blockers: 2
critical_findings: 5
requirements: 7/29
scenarios: 25/58
test_command: cd backend && go test ./internal/domain/... ./internal/service/fiscal/... ./internal/integration/fiscal/mock/... ./internal/core/ports/... -count=1 ; FISCAL_TEST_DATABASE_URL=postgres://admindb:***@localhost:5432/gonsgarage?sslmode=disable go test ./tests/integration/ -count=1 -run Fiscal(Migration|Repository|Mock) -timeout 120s
test_exit_code: 0
test_output_hash: sha256:4ff8b21727e25077cf0a36b53e00fe59f5670ce1d9a7438673937863f8ae813c
build_command: cd backend && go vet ./internal/domain/... ./internal/service/fiscal/... ./internal/integration/fiscal/mock/... ./internal/core/ports/... ./internal/repository/postgres/... ; go test -c -o NUL ./internal/domain/ ./internal/service/fiscal/ ./internal/integration/fiscal/mock/
build_exit_code: 0
build_output_hash: sha256:47100eb16365ae44d654bb6e5038b8126271d01420e4b06d928ac345f681a22e
```

## Verification Report

**Change**: cloudware-fiscal-integration-foundation
**Version**: N/A (OpenSpec change; four delta specs)
**Mode**: Standard (focused mid-change verify after WU1–WU4; full suite not claimed)
**Artifact store**: openspec
**Validator**: `gentle-ai sdd-verify-validate` **unavailable** on gentle-ai 3.5.0 (command not present; `sdd-attempt grant` is edit-authority only). Report persisted per parent Persistence instruction; admission tooling could not attest bytes.

### Completeness

| Metric | Value |
|--------|-------|
| Tasks total | 58 |
| Tasks complete | 22 (1.1–1.5, 2.1–2.8, 3.1–3.4, 4.1–4.4, 13.1) |
| Tasks incomplete | 36 (WU5–WU12 implementation + parent 13.2–13.5) |
| Change-level archive gate | **FAIL** — `allComplete: false` |
| WU1–WU4 local finish claims | Satisfied in `tasks.md` / `apply-progress.md` checkboxes |

Scope of this verify: **completed work units only** (schema/domain/persistence/mock). Remaining WU5–WU12 scenarios are expected deferred, not false PASS.

### Build & Tests Execution

**Build**: ✅ Passed (focused packages)
```text
cd backend
go vet ./internal/domain/... ./internal/service/fiscal/... ./internal/integration/fiscal/mock/... ./internal/core/ports/... ./internal/repository/postgres/...
→ VET_EXIT:0
go test -c -o NUL ./internal/domain/ ; ./internal/service/fiscal/ ; ./internal/integration/fiscal/mock/
→ BUILD_DOMAIN:0 BUILD_SVC:0 BUILD_MOCK:0
```

**Tests**: ✅ Focused packages passed / ⚠️ race unavailable
```text
# Unit / service / mock / ports (CGO_ENABLED=0)
go test ./internal/domain/... ./internal/service/fiscal/... ./internal/integration/fiscal/mock/... ./internal/core/ports/... -count=1
→ ok domain, service/fiscal, integration/fiscal/mock, core/ports (exit 0)

# PostgreSQL integration (FISCAL_TEST_DATABASE_URL set from local .env DATABASE_URL)
go test ./tests/integration/ -count=1 -run 'Fiscal(Migration|Repository|Mock)' -timeout 120s
→ ok tests/integration 25.260s (exit 0)
  covers FiscalMigration*, FiscalRepository*, FiscalMock*

# Race detector (documented limit)
CGO_ENABLED=1 go test ./internal/domain/ -count=1 -race
→ FAIL build: cgo: C compiler "gcc" not found
→ Not treated as scenario FAIL; CI/Linux must run -race. WARNING only.
```

**Coverage**: ➖ Not measured this run (no coverage threshold enforced in focused verify)

### Spec Compliance Matrix

Authoritative totals from retrieved specs: **29 requirements**, **58 scenarios**.

Legend for deferred rows: `UNTESTED (deferred WUn)` = expected not yet implemented; does not prove WU1–WU4 incorrect.

#### fiscal-documents (9 requirements / 24 scenarios)

| Requirement | Scenario | Test | Result |
|-------------|----------|------|--------|
| Fiscal drafts distinct | Employee prepares an FT draft | `draft_finalization_test.go` > `TestDraftServiceEmployeeCRUDAndClientDenial` | ✅ COMPLIANT |
| Fiscal drafts distinct | Client attempts draft preparation | same | ✅ COMPLIANT |
| Fiscal drafts distinct | Optional source record later changes | `fiscal_policy_test.go` > `TestCalculate_SourceReferenceSurvivesAndCanonicalIgnoresExternalMutation` | ✅ COMPLIANT |
| Fiscal drafts distinct | Unsupported document kind | impl `DocumentKind.IsValid` + `draft_service` guard; **no covering test** | ❌ UNTESTED |
| Complete fiscal data | Required fiscal identity is incomplete | `fiscal_policy_test.go` > `TestPolicyVersionResolve_ApprovalGateAndImmutability` + finalization readiness paths | ✅ COMPLIANT |
| Complete fiscal data | Legacy floating-point amount is supplied | no dedicated test that draft ignores `Invoice.Amount float64` | ❌ UNTESTED |
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
| Privileged recovery/void | Void succeeds | domain + mock void; **no outbox worker end-to-end** | ⚠️ PARTIAL |
| Legacy isolation | Upgrade with historical invoices | `TestFiscalMigrationLegacyIsolationAndSchema` | ✅ COMPLIANT |
| Additive APIs/UI | Existing invoice contract remains stable | invoice service regression retained; **HTTP snapshot is WU7** | ⚠️ PARTIAL |
| Additive APIs/UI | Owning client reads fiscal status | — | ❌ UNTESTED (deferred WU7/WU9) |
| Additive APIs/UI | Protected invoice deletion | `TestFiscalRepositoryFinalizeAtomicAndProtectedDelete` | ✅ COMPLIANT |

#### fiscal-provider-foundation (9 / 14)

| Requirement | Scenario | Test | Result |
|-------------|----------|------|--------|
| Gateway normalized | Provider becomes unavailable after finalization | — | ❌ UNTESTED (deferred WU5/WU11) |
| Finalization atomic | Dispatch event persistence fails | `TestFinalizationServiceOutboxFailureRollsBack` + `TestFiscalRepositoryOutboxInsertFailureRollsBack` | ✅ COMPLIANT |
| Finalization atomic | Process stops after commit | — | ❌ UNTESTED (deferred WU5) |
| Outbox crash-safe | Concurrent workers claim one event | — | ❌ UNTESTED (deferred WU5) |
| Outbox crash-safe | Worker crashes while leased | — | ❌ UNTESTED (deferred WU5) |
| Idempotent correlation | User double-clicks finalization | concurrent finalize PG + service idempotency | ✅ COMPLIANT |
| Failures classified | Rate limit response is definite | `TestMockProvider_ScenariosCoverValidationExpiredTransientRateLimitAmbiguousReconcileAndVoids` | ✅ COMPLIANT |
| Failures classified | Timeout may have followed issuance | mock `ambiguous` class; **lease/crash protocol is WU5** | ⚠️ PARTIAL |
| Ambiguous reconciliation | Reconciliation finds an issued document | mock ambiguous→Reconcile matched | ✅ COMPLIANT |
| Ambiguous reconciliation | Reconciliation cannot establish a result | no inconclusive/not-matched mock assertion | ❌ UNTESTED |
| Attempts redacted | Provider returns a secret-bearing error | — | ❌ UNTESTED (deferred WU5/WU10) |
| Outages isolate core | Provider is offline | `TestConnectionServiceProviderFailureFailsClosed` (partial surface) | ⚠️ PARTIAL |
| Deterministic mock | Same mock issuance is repeated | `TestMockProvider_FTAndFRIssuanceAreStableAndLabeled` + concurrent + reload | ✅ COMPLIANT |
| Deterministic mock | Production selects mock | `TestMockProvider_ProductionRejectsSelectionAndLegalClassification` | ✅ COMPLIANT |

#### fiscal-artifacts (5 / 8)

| Requirement | Scenario | Test | Result |
|-------------|----------|------|--------|
| Immutable PDF archive | Issuance returns a PDF | mock deterministic PDF bytes; **archive store WU6** | ⚠️ PARTIAL |
| Immutable PDF archive | Archived bytes are altered | — | ❌ UNTESTED (deferred WU6) |
| No duplicate issuance | PDF retrieval fails after confirmed issuance | — | ❌ UNTESTED (deferred WU6) |
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

**Compliance summary**: **25/58** scenarios ✅ COMPLIANT; **6** ⚠️ PARTIAL; **27** ❌ UNTESTED (of which **3** are in-scope for completed WUs and **24** are expected deferred). Fully green requirements: **7/29**.

### Correctness (Static Evidence — WU1–WU4)

| Requirement area | Status | Notes |
|------------------|--------|-------|
| Migration 011 + legacy isolation | ✅ Implemented | PG tests green; `VerifyFiscalSchema` fail-fast |
| Exact decimal + policy calculator | ✅ Implemented | shopspring/decimal; no float64 in fiscal calc |
| Lifecycle domain (11 states) | ✅ Implemented | transitions, role matrix, freeze invariants |
| Aggregate persistence / finalize+outbox | ✅ Implemented | atomic finalize, concurrent one-intent, delete protection |
| Provider-neutral ports + mock | ✅ Implemented | registry-keyed scenarios; persisted `fiscal_mock_operations`; prod guards |
| Outbox worker / artifact archive / HTTP / UI / Cloudware | ❌ Not in scope yet | WU5–WU12 pending |

### Coherence (Design — completed units)

| Decision | Followed? | Notes |
|----------|-----------|-------|
| FiscalDocument aggregate separate from Invoice | ✅ Yes | eligibility column; draft/finalize services |
| Exact decimal + unapproved policy fail-closed | ✅ Yes | domain package |
| Finalization + initial outbox one transaction | ✅ Yes | service + PG rollback tests |
| Gateway Issue/Reconcile/Void/Fetch/Verify | ✅ Yes | ports + mock |
| Deterministic mock via injected registry | ✅ Yes | no magic identity fields |
| Production rejects mock / legal mock classification | ✅ Yes | `guard.go` + `cmd/api/main.go` |
| `FiscalProviderGateway` type name | ⚠️ Deviation | Retained `FiscalProvider` port (apply-progress); behavior matches |
| PDF bytes only checksum in PG mock table | ⚠️ Deviation | regenerate from canonical; checksum enforced |
| Worker SKIP LOCKED / crash→unknown | ➖ Deferred | WU5 |
| Private artifact store / OAuth / Cloudware gates | ➖ Deferred | WU6/WU10 |

### Issues Found

**CRITICAL**:
1. Change-level incompleteness: **36/58 tasks pending** — archive gate must remain blocked.
2. In-scope UNTESTED: **Unsupported document kind** — validation exists in code, no passing covering test.
3. In-scope UNTESTED: **Legacy floating-point amount is supplied** — no dedicated test proving draft ignores `Invoice.Amount`.
4. In-scope UNTESTED: **Reconciliation cannot establish a result** — mock covers matched reconcile only.
5. `gentle-ai sdd-verify-validate` unavailable on installed 3.5.0 — cannot machine-admit report bytes (parent still required OpenSpec+Engram persistence).

**WARNING**:
1. `go test -race` blocked locally (CGO/`gcc` missing on Windows agent). Non-race tests green; CI must run race.
2. Design deviations: port naming `FiscalProvider` vs design `FiscalProviderGateway`; mock PDF regen-from-canonical when row stores sha only.
3. WU4 authored size ~1100–1400 lines (`size:exception`) above preferred 600-line manual slice.
4. Six PARTIAL scenarios in completed surface (void E2E, invoice HTTP stability, timeout/worker protocol, provider-offline breadth, mock PDF archive/download).

**SUGGESTION**:
1. Before WU5 apply, add small RED tests for unsupported kind, legacy amount independence, and inconclusive reconcile.
2. Continue `/sdd-apply` at **WU5** (outbox worker) — native next after mid-change verify.
3. Do **not** archive until 58/58 tasks complete and a full verify PASS with validator admission.

### Verdict

**FAIL**

Change-level verification cannot PASS while 36 tasks remain and three in-scope scenarios lack covering tests. WU1–WU4 focused correctness is largely green (unit + PostgreSQL FiscalMigration/FiscalRepository/FiscalMock exit 0); proceed to WU5 apply after optionally remediating the three in-scope UNTESTED gaps. **Not archive-ready.**

### Verification scope note

This is an honest **mid-change** verify requested after WU1–WU4. It is **not** a claim that the OpenSpec change is complete. Native `nextRecommended` remaining `apply` with 36 pending tasks is expected and correct.
