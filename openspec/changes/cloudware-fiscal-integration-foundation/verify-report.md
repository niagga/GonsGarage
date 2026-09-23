```yaml
schema: gentle-ai.verify-result/v1
evidence_revision: sha256:64cb4306252c1e7723aef87b5bfa551541c3bf9d39e3fb057b301582717aa8b4
verdict: fail
blockers: 1
critical_findings: 1
requirements: 26/29
scenarios: 54/58
remediation_note: "2026-09-23 mid-change post-WU10: Cloudware security shell green (crypto AAD, hashed OAuth, gates block before httptest, mutationsEnabled=false default); WU1–WU10 focused correctness OK; PDF ArtifactService nil + artifactId PARTIAL retained; refresh→document connection_action_required and license remediation PARTIAL; PG Fiscal* skipped (FISCAL_TEST_DATABASE_URL unset in process); WU11 ops + WU12 live UNTESTED(deferred); archive gate FAIL at 46/58"
test_command: cd backend && CGO_ENABLED=0 go test ./internal/platform/crypto/ ./internal/service/fiscal/ ./internal/integration/fiscal/cloudware/ ./internal/handler/ ./internal/domain/... ./internal/integration/fiscal/mock/... ./internal/platform/fiscalartifact/... ./internal/core/ports/... -count=1 ; cd frontend && pnpm test -- --passWithNoTests && pnpm typecheck
test_exit_code: 0
test_output_hash: sha256:416273401c320d1da35f5bfe1629c84a62b519019c6406db5648844c42eb316e
build_command: cd backend && go vet ./internal/platform/crypto/ ./internal/service/fiscal/ ./internal/integration/fiscal/cloudware/ ./internal/handler/ ./internal/domain/... ./internal/repository/postgres/... ; go build -o NUL ./cmd/api/ ; go build -o NUL ./cmd/fiscal-worker/
build_exit_code: 0
build_output_hash: sha256:b1a8fbbab1dbb706ce0caa7696aafb04b008f4be2287093bffeab66babe755c7
```

## Verification Report

**Change**: cloudware-fiscal-integration-foundation
**Version**: N/A (OpenSpec change; four delta specs)
**Mode**: Standard (focused mid-change verify after WU1–WU10; full suite / archive not claimed)
**Artifact store**: openspec
**Validator**: `gentle-ai sdd-verify-validate` **unavailable** on installed gentle-ai (command not present; only `sdd-status` / `sdd-continue` / `sdd-attempt grant`). Report persisted per parent Persistence instruction; admission tooling could not attest bytes.

### Completeness

| Metric | Value |
|--------|-------|
| Tasks total | 58 |
| Tasks complete | 46 (1.1–1.5, 2.1–2.8, 3.1–3.4, 4.1–4.4, 5.1–5.4, 6.1–6.4, 7.1–7.4, 8.1–8.4, 9.1–9.4, 10.1–10.4, 13.1) |
| Tasks incomplete | 12 (WU11 11.1–11.4, WU12 12.1–12.4 BLOCKED, parent 13.2–13.5) |
| Change-level archive gate | **FAIL** — `allComplete: false` |
| WU1–WU10 local finish claims | Satisfied in `tasks.md` / `apply-progress.md` checkboxes |

Scope of this verify: **completed work units only** (schema through Cloudware security shell). Remaining WU11 ops and WU12 live Cloudware are expected deferred, not false PASS.

### Build & Tests Execution

**Build**: ✅ Passed (`cmd/api` + `cmd/fiscal-worker` + focused vet)
```text
cd backend
go vet ./internal/platform/crypto/ ./internal/service/fiscal/ ./internal/integration/fiscal/cloudware/ ./internal/handler/ ./internal/domain/... ./internal/repository/postgres/...
→ VET:0
go build -o NUL ./cmd/api/
→ BUILD_API:0
go build -o NUL ./cmd/fiscal-worker/
→ BUILD_WORKER:0
build_output_hash: sha256:b1a8fbbab1dbb706ce0caa7696aafb04b008f4be2287093bffeab66babe755c7
```

**Tests**: ✅ Backend WU10-focused + domain/mock/artifact/ports + frontend full suite / ⚠️ PG skipped / ⚠️ race N/A
```text
# Unit (CGO_ENABLED=0) — WU10 attention packages + retained WU1–WU9 packages
go test ./internal/platform/crypto/ ./internal/service/fiscal/ ./internal/integration/fiscal/cloudware/ ./internal/handler/ ./internal/domain/... ./internal/integration/fiscal/mock/... ./internal/platform/fiscalartifact/... ./internal/core/ports/... -count=1
→ ok crypto, fiscal, cloudware, handler, domain, mock, fiscalartifact, ports (exit 0)

# PostgreSQL Fiscal* integration
FISCAL_TEST_DATABASE_URL not present in process env (safer path avoided reading .env).
→ PG_EXIT:skip — prior mid-change (post-WU9) had Fiscal* PG green; WU10 adds no new Fiscal* PG suite. WARNING only.

# Frontend confirm (vitest ran full suite despite file filter argv)
cd frontend && pnpm test -- --passWithNoTests
→ Test Files 34 passed (34) / Tests 175 passed (175) (exit 0)
pnpm typecheck → TC_EXIT:0

# Live Cloudware network
cloudware_test.go uses net/http/httptest only; NewAdapter defaults mutationsEnabled=false;
TestEnablementEvaluatorBlocksMutationBeforeHTTP asserts httpCalls==0.
→ NO live Cloudware mutations in ordinary tests.

# Race detector
CGO/gcc unavailable on this Windows agent → N/A (WARNING; CI/Linux remains authoritative).

test_output_hash: sha256:416273401c320d1da35f5bfe1629c84a62b519019c6406db5648844c42eb316e
Authoritative confirm exit for envelope: test_exit_code 0.
```

**Coverage**: ➖ Not measured this run (no coverage threshold enforced in focused verify)

### Spec Compliance Matrix

Authoritative totals from retrieved specs: **29 requirements**, **58 scenarios**.

Legend: `UNTESTED (deferred WUn)` = expected not yet implemented; does not prove WU1–WU10 incorrect.

#### fiscal-documents (9 requirements / 24 scenarios)

| Requirement | Scenario | Test | Result |
|-------------|----------|------|--------|
| Fiscal drafts distinct | Employee prepares an FT draft | `draft_finalization_test.go` + FE `FiscalDraftForm.test.tsx` | ✅ COMPLIANT |
| Fiscal drafts distinct | Client attempts draft preparation | `TestDraftServiceEmployeeCRUDAndClientDenial` | ✅ COMPLIANT |
| Fiscal drafts distinct | Optional source record later changes | `TestCalculate_SourceReferenceSurvivesAndCanonicalIgnoresExternalMutation` + FE | ✅ COMPLIANT |
| Fiscal drafts distinct | Unsupported document kind | `TestDraftServiceRejectsUnsupportedDocumentKind` | ✅ COMPLIANT |
| Complete fiscal data | Required fiscal identity is incomplete | policy approval gate + FE readiness | ✅ COMPLIANT |
| Complete fiscal data | Legacy floating-point amount is supplied | draft decimal independence + FE | ✅ COMPLIANT |
| Deterministic arithmetic | Arithmetic policy is unavailable | `TestPolicyVersionResolve_ApprovalGateAndImmutability` | ✅ COMPLIANT |
| Deterministic arithmetic | Totals reconcile | `TestCalculate_*` + FE server totals | ✅ COMPLIANT |
| Deterministic arithmetic | Submitted totals do not reconcile | `TestCalculate_DocumentRoundingAdjustmentAndDeclaredMismatch` | ✅ COMPLIANT |
| Explicit finalization | Manager finalizes an eligible draft | finalization + PG + FE manager finalize | ✅ COMPLIANT |
| Explicit finalization | Employee attempts finalization | auth matrix + FE hide | ✅ COMPLIANT |
| Explicit finalization | Two finalization requests race | concurrent finalize PG + FE double-click | ✅ COMPLIANT |
| Frozen immutable | Customer or invoice changes after finalization | domain freeze + migration immutability | ✅ COMPLIANT |
| Frozen immutable | Staff attempts to edit frozen lines | `TestFiscalMigrationConstraintsAndImmutability` | ✅ COMPLIANT |
| Lifecycle guarded | Definite validation rejection | `TestFiscalDocumentFinalizeFreezeAndTransitions` | ✅ COMPLIANT |
| Lifecycle guarded | Ambiguous issuance result | unknown retry prohibition + FE guidance | ✅ COMPLIANT |
| Lifecycle guarded | Definite void failure | same + failed void → issued | ✅ COMPLIANT |
| Privileged recovery/void | Manager retries after connection recovery | role matrix + FE privileged | ✅ COMPLIANT |
| Privileged recovery/void | Employee requests reconciliation or void | role matrix + handler + FE | ✅ COMPLIANT |
| Privileged recovery/void | Void succeeds | domain + mock void + worker | ✅ COMPLIANT |
| Legacy isolation | Upgrade with historical invoices | migration legacy + FE badge | ✅ COMPLIANT |
| Additive APIs/UI | Existing invoice contract remains stable | legacy snapshot + FE columns | ✅ COMPLIANT |
| Additive APIs/UI | Owning client reads fiscal status | handler own-only + FE my-invoices | ✅ COMPLIANT |
| Additive APIs/UI | Protected invoice deletion | finalize protected delete + HTTP 409 | ✅ COMPLIANT |

#### fiscal-provider-foundation (9 / 14)

| Requirement | Scenario | Test | Result |
|-------------|----------|------|--------|
| Gateway normalized | Provider becomes unavailable after finalization | worker reconcile frozen provider/key | ✅ COMPLIANT |
| Finalization atomic | Dispatch event persistence fails | outbox rollback service + PG | ✅ COMPLIANT |
| Finalization atomic | Process stops after commit | outbox e2e issue with mock | ✅ COMPLIANT |
| Outbox crash-safe | Concurrent workers claim one event | SKIP LOCKED single claim | ✅ COMPLIANT |
| Outbox crash-safe | Worker crashes while leased | lease expiry + stale→unknown | ✅ COMPLIANT |
| Idempotent correlation | User double-clicks finalization | concurrent finalize + handler + FE | ✅ COMPLIANT |
| Failures classified | Rate limit response is definite | mock rate-limit scenario | ✅ COMPLIANT |
| Failures classified | Timeout may have followed issuance | ambiguous timeout never auto-resubmit | ✅ COMPLIANT |
| Ambiguous reconciliation | Reconciliation finds an issued document | mock ambiguous→reconcile matched | ✅ COMPLIANT |
| Ambiguous reconciliation | Reconciliation cannot establish a result | inconclusive reconcile retains unknown | ✅ COMPLIANT |
| Attempts redacted | Provider returns a secret-bearing error | redact + sanitize projection | ✅ COMPLIANT |
| Outages isolate core | Provider is offline | legacy invoice independent; `/ready` isolation remains WU11 | ⚠️ PARTIAL |
| Deterministic mock | Same mock issuance is repeated | FT/FR stable labeled mock | ✅ COMPLIANT |
| Deterministic mock | Production selects mock | production rejects mock/legal | ✅ COMPLIANT |

#### fiscal-artifacts (5 / 8)

| Requirement | Scenario | Test | Result |
|-------------|----------|------|--------|
| Immutable PDF archive | Issuance returns a PDF | local store + artifact service + PG path (prior) | ✅ COMPLIANT |
| Immutable PDF archive | Archived bytes are altered | compromised refusal | ✅ COMPLIANT |
| No duplicate issuance | PDF retrieval fails after confirmed issuance | retain issued + recover FetchArtifact only | ✅ COMPLIANT |
| Authz downloads | Owning client downloads PDF | handler/FE wired; **API `ArtifactService` nil; projection omits `artifactId`** | ⚠️ PARTIAL |
| Authz downloads | Different client requests PDF | 404 non-enumeration + FE | ✅ COMPLIANT |
| Legal vs mock | Developer downloads mock PDF | SEM VALIDADE FISCAL — MOCK label | ✅ COMPLIANT |
| Production readiness | Artifact backend is not production-ready | fail-closed readiness | ✅ COMPLIANT |
| Production readiness | New issuance is disabled | read while issuance disabled | ✅ COMPLIANT |

#### cloudware-fiscal-enablement (6 / 12) — **WU10 in scope**

| Requirement | Scenario | Test | Result |
|-------------|----------|------|--------|
| Scoped connections | Manager starts connection setup | `TestConnectionServiceStartOAuthBindsHashedStateToActorRedirectExpiry` + handler `CloudwareConnect` | ✅ COMPLIANT |
| Scoped connections | Employee calls connection endpoint | `TestConnectionServiceEmployeeCannotStartOAuth` + `TestFiscalIntegrationHandler_EmployeeLegalActionsForbidden` | ✅ COMPLIANT |
| OAuth protected | OAuth callback succeeds | `TestConnectionServiceOAuthCallbackRejectsInvalidAndReusedState` success path (encrypted at rest; status without tokens) | ✅ COMPLIANT |
| OAuth protected | Callback state is invalid | same test invalid + reused state; no code exchange | ✅ COMPLIANT |
| Evidenced FT/FR mapping | Frozen FR is mapped | `TestMapFrozenFTAndFROnly` + FR associated-receipt fixture | ✅ COMPLIANT |
| Evidenced FT/FR mapping | Unsupported receipt workflow is requested | `TestMapRejectsUnsupportedKindsAndReceiptWorkflows` | ✅ COMPLIANT |
| Evidence-gated mutation | One applicable gate lacks evidence | `TestEnablementEvaluatorBlocksMutationBeforeHTTP` (httpCalls==0; mutationsEnabled default false) | ✅ COMPLIANT |
| Evidence-gated mutation | A gate is not applicable | `TestGateNotApplicableRequiresRationale` | ✅ COMPLIANT |
| No undocumented behavior | Provider times out before idempotency evidence exists | enablement block + adapter `Reconcile` unevidenced + worker no blind resubmit | ✅ COMPLIANT |
| No undocumented behavior | UI presents issued status | FE badges; AT/e-Fatura only when `atCommunicationStatus` evidenced | ✅ COMPLIANT |
| Normalized errors | Refresh token fails | `TestConnectionServiceRefreshFailureSetsActionRequiredWithoutLeakingSecrets` — connection → `action_required`; **document → `connection_action_required` not asserted/wired** | ⚠️ PARTIAL |
| Normalized errors | Active GC license is absent | `TestNormalizeErrorsRefreshLicenseAmbiguous` maps `active_license_required`; **full remediation + frozen-intent retention path not end-to-end asserted** | ⚠️ PARTIAL |

**Compliance summary**: **54/58** scenarios ✅ COMPLIANT; **4** ⚠️ PARTIAL; **0** ❌ UNTESTED in completed WUs. Deferred (not matrix rows as UNTESTED for archive): WU11 ops/CI/runbook, WU12 live credentialed acceptance / parent 13.2–13.4. Fully green requirements: **26/29**. Mid-change archive gate remains **FAIL** (46/58 tasks).

### WU10 Cloudware security shell attention

| Design / hard constraint | Evidence | Result |
|--------------------------|----------|--------|
| Fake HTTP / test credentials only | `httptest.NewServer` in `cloudware_test.go`; stub OAuth/token exchangers | ✅ |
| `mutationsEnabled=false` default | `NewAdapter` sets `mutationsEnabled: false`; `WithMutationsEnabled` reserved WU12 | ✅ |
| Incomplete gates block before HTTP | `TestEnablementEvaluatorBlocksMutationBeforeHTTP` asserts `httpCalls==0` | ✅ |
| AES-GCM AAD binding | `TestFiscalCredentialCipherAADBindingRejectsWrongConnection` + `EncryptWithBinding` | ✅ |
| Keyring rotation / production keyring | rotation + `TestValidateProductionKeyringRejectsDefaultAndShortKeys` | ✅ |
| Hashed one-time OAuth state (actor/redirect/expiry) | `TestConnectionServiceStartOAuthBindsHashedStateToActorRedirectExpiry` | ✅ |
| Invalid/reused callback | `TestConnectionServiceOAuthCallbackRejectsInvalidAndReusedState` | ✅ |
| Redaction | crypto envelope redaction + `TestAdapterRedactsSecretsFromDiagnostics` + refresh no token leak | ✅ |
| Bounded TLS client / allowlist / no mutation retries | `TestHTTPClientAllowlistDeadlinesAndNoMutationRetry` | ✅ |
| Adapter isolation | `TestDomainAndHandlerDoNotImportCloudware` / `TestCloudwareImportIsolation` | ✅ |
| No live Cloudware network in tests | allowlisted `httptest` hosts only; no production host calls observed | ✅ |
| Race detector | N/A without gcc/cgo | ⚠️ documented |

### Correctness (Static Evidence — WU1–WU10)

| Requirement area | Status | Notes |
|------------------|--------|-------|
| Migration 011 + legacy isolation | ✅ Implemented | Prior PG green; not re-run this session |
| Exact decimal + policy calculator | ✅ Implemented | shopspring/decimal |
| Lifecycle domain (11 states) | ✅ Implemented | includes `connection_action_required` state |
| Aggregate persistence / finalize+outbox | ✅ Implemented | |
| Provider-neutral ports + mock | ✅ Implemented | |
| Outbox worker lease/crash protocol | ✅ Implemented | |
| Immutable artifact archive + recovery | ✅ Implemented | |
| Additive fiscal HTTP API | ✅ Implemented | |
| Staff fiscal UI (WU8) | ✅ Implemented | |
| Client/admin UI (WU9) | ✅ Implemented | FE 175 confirm |
| Cloudware security shell (WU10) | ✅ Implemented | crypto/OAuth/gates/adapter/handler readiness+connect+callback |
| API ArtifactService composition | ⚠️ Partial | `cmd/api/main.go` still `NewDocumentService(..., nil, ...)`; projection omits `artifactId` |
| Refresh → fiscal document state | ⚠️ Partial | connection `action_required` only |
| Ops / CI / runbooks (WU11) | ❌ Deferred | |
| Live Cloudware enablement (WU12) | ❌ Blocked | parent 13.2–13.4 |

### Coherence (Design — Cloudware security shell)

| Decision | Followed? | Notes |
|----------|-----------|-------|
| Cloudware package owns DTOs/OAuth/mapping/errors/gates | ✅ Yes | `integration/fiscal/cloudware/*`; domain/handler do not import |
| Production mutations fail-closed via `CloudwareEnablementEvaluator` | ✅ Yes | `GateCatalog.AllowMutation` + `guardMutation` + `mutationsEnabled` |
| OAuth state one-time, hashed, actor/redirect/expiry bound | ✅ Yes | `StartOAuthConnect` / `ConsumeOAuthAuthorization` |
| AES-256-GCM AAD: connection ID, provider, scope, format | ✅ Yes | `FiscalCredentialBinding` + AAD tests |
| Dedicated keyring / no JWT secret reuse | ✅ Yes | production keyring validation |
| No webhook route; no assumed idempotency/AT | ✅ Yes | optional webhooks N/A with rationale; reconcile unevidenced |
| FT/FR only; no FS/receipt/correction | ✅ Yes | mapping + workflow validation |
| Secret-free logs/API/outbox | ✅ Yes | redaction tests; status omits tokens |
| API PDF streaming via composed ArtifactService | ⚠️ Deviation | still nil in `cmd/api` |
| `FiscalProviderGateway` type name | ⚠️ Deviation | retained `FiscalProvider` port |

### Issues Found

**CRITICAL**:
1. Change-level incompleteness: **12/58 tasks pending** (WU11 + WU12 + parent 13.2–13.5) — archive gate must remain blocked.

**WARNING**:
1. Production API `ArtifactService` remains `nil`; projection still omits `artifactId`/`classification` → PDF download PARTIAL.
2. Refresh-token failure sets connection `action_required` but does not yet prove fiscal document transition to `connection_action_required`.
3. Active-GC-license path is normalized at HTTP error layer only; manager remediation + frozen-intent retention not end-to-end covered for Cloudware.
4. Fiscal* PostgreSQL suite skipped this run (`FISCAL_TEST_DATABASE_URL` unset in process; `.env` not read).
5. `go test -race` blocked locally (CGO/`gcc` missing). Non-race tests green; CI must run race.
6. `gentle-ai sdd-verify-validate` unavailable — cannot machine-admit report bytes.
7. Design deviations retained: port naming `FiscalProvider`; ObjectStore readiness-only; ArtifactService nil.
8. WU10 authored above preferred ~600-line slice (`size:exception`, documented in apply-progress).
9. Provider-offline `/ready` isolation remains PARTIAL pending WU11.

**SUGGESTION**:
1. Continue `/sdd-apply` at **WU11** (ops/CI/deployment/runbooks). Do **not** start WU12 until parent 13.2–13.4.
2. Do **not** archive until 58/58 tasks complete and a full verify PASS with validator admission when tooling is available.
3. Optionally wire API ArtifactService + projection `artifactId` during WU11 composition.
4. Extend refresh failure path to assert document `connection_action_required` when an intent is in flight.
5. Ensure CI Linux runners execute `go test -race` and Fiscal* PG with an explicit test DB URL.

### Verdict

**FAIL**

Change-level verification cannot PASS while 12 tasks remain. WU1–WU10 focused correctness is green (crypto + fiscal service + cloudware + handler + domain/mock/artifact/ports exit 0; frontend 175 + typecheck exit 0; `cmd/api` and `cmd/fiscal-worker` build). Cloudware enablement scenarios moved from deferred UNTESTED to mostly COMPLIANT with proof of fail-closed gates, hashed OAuth, AAD, and no live mutations. Honest PARTIALs remain for API PDF composition, refresh→document state, license remediation E2E, and provider `/ready` isolation. Proceed to WU11 apply. **Not archive-ready.**

### Verification scope note

This is an honest **mid-change** verify requested after WU1–WU10. It is **not** a claim that the OpenSpec change is complete. Native next step remains `sdd-apply` (WU11) with 12 pending tasks.
