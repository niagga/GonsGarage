```yaml
schema: gentle-ai.verify-result/v1
evidence_revision: sha256:2cf9766c1d9cceb3bb67b1c488c916f631e99bc77190ea6158e6f202ffac152f
verdict: fail
blockers: 1
critical_findings: 1
requirements: 28/29
scenarios: 56/58
remediation_note: "2026-09-23 mid-change post-WU11: ops/CI/runbooks green (switches default off, production fail-closed, /ready isolates provider, CI Postgres 16, runbook+traceability); ArtifactService composition + artifactId closes prior PDF PARTIAL; WU1–WU11 focused correctness OK; refresh→document connection_action_required and license remediation PARTIAL retained; PG Fiscal* skipped (FISCAL_TEST_DATABASE_URL unset); go test -race N/A without gcc (CI authoritative); WU12 live UNTESTED(deferred); archive gate FAIL at 50/58"
test_command: cd backend && CGO_ENABLED=0 go test ./internal/service/fiscal/ ./cmd/api/ ./cmd/fiscal-worker/ ./internal/handler/ ./internal/core/ports/ ./internal/platform/crypto/ ./internal/integration/fiscal/cloudware/ ./internal/domain/... ./internal/integration/fiscal/mock/... ./internal/platform/fiscalartifact/... -count=1 ; go test ./internal/service/fiscal/ ./cmd/api/ ./cmd/fiscal-worker/ -count=1 -run "TestRuntimeConfig_|TestCoreAPIReady_|TestWorker|TestReadyHandler_|TestFiscalDependency|TestValidate|TestResolveProvider|TestArtifactRecovery|TestFiscalLogEvent_|TestFiscalMetricsSnapshot_|TestFinalizationDisabled"
test_exit_code: 0
test_output_hash: sha256:a41e1f4b6606f8d6973b7b6dea2aba5f8c5345469c0f291580308e4c3b9bc7f6
build_command: cd backend && go vet ./internal/service/fiscal/ ./cmd/api/ ./cmd/fiscal-worker/ ./internal/handler/ ./internal/domain/... ./internal/platform/crypto/ ./internal/integration/fiscal/cloudware/ ; go build -o NUL ./cmd/api/ ; go build -o NUL ./cmd/fiscal-worker/ ; go build -o NUL ./cmd/migrate/
build_exit_code: 0
build_output_hash: sha256:0c0bfc913f2d94b92513ae826ee3e2f07a590ae4cbf2a9110189e4b7e6c1f55d
```

## Verification Report

**Change**: cloudware-fiscal-integration-foundation
**Version**: N/A (OpenSpec change; four delta specs)
**Mode**: Standard (focused mid-change verify after WU1–WU11; full suite / archive not claimed)
**Artifact store**: openspec (primary) + Engram upsert
**Validator**: `gentle-ai sdd-verify-validate` **unavailable** on installed gentle-ai 3.5.0 (command not present; only `sdd-status` / `sdd-continue` / `sdd-attempt grant`). Report persisted per orchestrator Persistence instruction; admission tooling could not attest bytes.

### Completeness

| Metric | Value |
|--------|-------|
| Tasks total | 58 |
| Tasks complete | 50 (1.1–1.5, 2.1–2.8, 3.1–3.4, 4.1–4.4, 5.1–5.4, 6.1–6.4, 7.1–7.4, 8.1–8.4, 9.1–9.4, 10.1–10.4, 11.1–11.4, 13.1) |
| Tasks incomplete | 8 (WU12 12.1–12.4 **BLOCKED**, parent 13.2–13.5) |
| Change-level archive gate | **FAIL** — `allComplete: false` |
| WU11 local finish claims | Satisfied in `tasks.md` / `apply-progress.md` (11.1–11.4 `[x]`) |

Scope of this verify: **completed work units WU1–WU11** (schema through ops/CI/runbooks). Remaining WU12 live Cloudware is expected deferred, not a WU11 implementation failure.

### Build & Tests Execution

**Build**: ✅ Passed (`cmd/api` + `cmd/fiscal-worker` + `cmd/migrate` + focused vet)
```text
cd backend
go vet ./internal/service/fiscal/ ./cmd/api/ ./cmd/fiscal-worker/ ./internal/handler/ ./internal/domain/... ./internal/platform/crypto/ ./internal/integration/fiscal/cloudware/
→ VET_EXIT:0
go build -o NUL ./cmd/api/
→ BUILD_API:0
go build -o NUL ./cmd/fiscal-worker/
→ BUILD_WORKER:0
go build -o NUL ./cmd/migrate/
→ BUILD_MIGRATE:0
build_output_hash: sha256:0c0bfc913f2d94b92513ae826ee3e2f07a590ae4cbf2a9110189e4b7e6c1f55d
```

**Tests**: ✅ WU11-focused + retained fiscal packages / ⚠️ PG skipped / ⚠️ race N/A
```text
# Unit (CGO_ENABLED=0) — WU11 attention + retained WU1–WU10 packages
go test ./internal/service/fiscal/ ./cmd/api/ ./cmd/fiscal-worker/ ./internal/handler/ ./internal/core/ports/ ./internal/platform/crypto/ ./internal/integration/fiscal/cloudware/ ./internal/domain/... ./internal/integration/fiscal/mock/... ./internal/platform/fiscalartifact/... -count=1
→ ok fiscal, cmd/api, cmd/fiscal-worker, handler, ports, crypto, cloudware, domain, mock, fiscalartifact (exit 0)

# WU11 config/readiness matrix
go test ./internal/service/fiscal/ ./cmd/api/ ./cmd/fiscal-worker/ -count=1 -run "TestRuntimeConfig_|TestCoreAPIReady_|TestWorker|TestReadyHandler_|TestFiscalDependency|TestValidate|TestResolveProvider|TestArtifactRecovery|TestFiscalLogEvent_|TestFiscalMetricsSnapshot_|TestFinalizationDisabled"
→ MATRIX_EXIT:0

# PostgreSQL Fiscal* integration
FISCAL_TEST_DATABASE_URL not present in process env (safer path avoided reading .env).
→ PG_EXIT:skip — prior mid-change had Fiscal* PG green; CI postgres:16 now applies migration 011 smoke + go test -race. WARNING only.

# Live Cloudware network
cloudware adapter defaults mutationsEnabled=false; gate evaluator blocks before HTTP.
→ NO live Cloudware mutations in ordinary tests. WU12 remains blocked.

# Race detector
go test -race → requires cgo; CGO/gcc unavailable on this Windows agent → N/A (WARNING; CI ubuntu job runs -race with CGO_ENABLED=1).

test_output_hash: sha256:a41e1f4b6606f8d6973b7b6dea2aba5f8c5345469c0f291580308e4c3b9bc7f6
Authoritative confirm exit for envelope: test_exit_code 0.
```

**Coverage**: ➖ Not measured this run (no coverage threshold enforced in focused verify)

### Spec Compliance Matrix

Authoritative totals from retrieved specs: **29 requirements**, **58 scenarios**.

Legend: `UNTESTED (deferred WU12)` = expected blocked live enablement; does not prove WU1–WU11 incorrect.

#### fiscal-documents (9 requirements / 24 scenarios)

| Requirement | Scenario | Test | Result |
|-------------|----------|------|--------|
| Fiscal drafts distinct | Employee prepares an FT draft | `draft_finalization_test.go` + FE `FiscalDraftForm.test.tsx` | ✅ COMPLIANT |
| Fiscal drafts distinct | Client attempts draft preparation | `TestDraftServiceEmployeeCRUDAndClientDenial` | ✅ COMPLIANT |
| Fiscal drafts distinct | Optional source record later changes | domain + FE source reference | ✅ COMPLIANT |
| Fiscal drafts distinct | Unsupported document kind | `TestDraftServiceRejectsUnsupportedDocumentKind` | ✅ COMPLIANT |
| Complete fiscal data | Required fiscal identity is incomplete | policy approval gate + FE readiness | ✅ COMPLIANT |
| Complete fiscal data | Legacy floating-point amount is supplied | draft decimal independence + FE | ✅ COMPLIANT |
| Deterministic arithmetic | Arithmetic policy is unavailable | policy approval gate tests | ✅ COMPLIANT |
| Deterministic arithmetic | Totals reconcile | `TestCalculate_*` + FE server totals | ✅ COMPLIANT |
| Deterministic arithmetic | Submitted totals do not reconcile | declared mismatch tests | ✅ COMPLIANT |
| Explicit finalization | Manager finalizes an eligible draft | finalization + FE; default-off `WithEnabled` | ✅ COMPLIANT |
| Explicit finalization | Employee attempts finalization | auth matrix + FE hide | ✅ COMPLIANT |
| Explicit finalization | Two finalization requests race | concurrent finalize + FE double-click | ✅ COMPLIANT |
| Frozen immutable | Customer or invoice changes after finalization | domain freeze + migration immutability | ✅ COMPLIANT |
| Frozen immutable | Staff attempts to edit frozen lines | migration immutability | ✅ COMPLIANT |
| Lifecycle guarded | Definite validation rejection | domain transition tests | ✅ COMPLIANT |
| Lifecycle guarded | Ambiguous issuance result | unknown retry prohibition + FE | ✅ COMPLIANT |
| Lifecycle guarded | Definite void failure | failed void → issued | ✅ COMPLIANT |
| Privileged recovery/void | Manager retries after connection recovery | role matrix + FE | ✅ COMPLIANT |
| Privileged recovery/void | Employee requests reconciliation or void | role matrix + handler + FE | ✅ COMPLIANT |
| Privileged recovery/void | Void succeeds | domain + mock void + worker | ✅ COMPLIANT |
| Legacy isolation | Upgrade with historical invoices | migration legacy + FE badge | ✅ COMPLIANT |
| Additive APIs/UI | Existing invoice contract remains stable | legacy snapshot + FE columns | ✅ COMPLIANT |
| Additive APIs/UI | Owning client reads fiscal status | handler own-only + FE my-invoices | ✅ COMPLIANT |
| Additive APIs/UI | Protected invoice deletion | finalize protected delete + HTTP 409 | ✅ COMPLIANT |

#### fiscal-provider-foundation (9 / 14) — **WU11 closes /ready isolation**

| Requirement | Scenario | Test | Result |
|-------------|----------|------|--------|
| Gateway normalized | Provider becomes unavailable after finalization | worker reconcile frozen provider/key | ✅ COMPLIANT |
| Finalization atomic | Dispatch event persistence fails | outbox rollback | ✅ COMPLIANT |
| Finalization atomic | Process stops after commit | outbox e2e with mock | ✅ COMPLIANT |
| Outbox crash-safe | Concurrent workers claim one event | SKIP LOCKED single claim | ✅ COMPLIANT |
| Outbox crash-safe | Worker crashes while leased | lease expiry + stale→unknown | ✅ COMPLIANT |
| Idempotent correlation | User double-clicks finalization | concurrent finalize + handler + FE | ✅ COMPLIANT |
| Failures classified | Rate limit response is definite | mock rate-limit | ✅ COMPLIANT |
| Failures classified | Timeout may have followed issuance | ambiguous never auto-resubmit | ✅ COMPLIANT |
| Ambiguous reconciliation | Reconciliation finds an issued document | mock ambiguous→reconcile | ✅ COMPLIANT |
| Ambiguous reconciliation | Reconciliation cannot establish a result | inconclusive retains unknown | ✅ COMPLIANT |
| Attempts redacted | Provider returns a secret-bearing error | redact + `TestFiscalLogEvent_AllowlistsSafeFieldsOnly` | ✅ COMPLIANT |
| Outages isolate core | Provider is offline | `TestCoreAPIReady_IgnoresProviderHealth`, `TestReadyHandler_IgnoresFiscalProviderOutage`, `TestRuntimeConfig_ProviderOutageIsolatedFromCoreReady`, `/fiscal/dependency-status` | ✅ COMPLIANT |
| Deterministic mock | Same mock issuance is repeated | FT/FR stable labeled mock | ✅ COMPLIANT |
| Deterministic mock | Production selects mock | `TestRuntimeConfig_ProductionRejectsDefaultsMockLocalMissingSchema`, `TestResolveProviderKey_DefaultsMockOnlyOutsideProduction`, `TestValidateAPIFiscalComposition_ProductionRejectsMock`, `TestWorkerStartupGate_ProductionRejectsMockProvider` | ✅ COMPLIANT |

#### fiscal-artifacts (5 / 8) — **WU11 closes API ArtifactService composition**

| Requirement | Scenario | Test | Result |
|-------------|----------|------|--------|
| Immutable PDF archive | Issuance returns a PDF | local store + artifact service | ✅ COMPLIANT |
| Immutable PDF archive | Archived bytes are altered | compromised refusal | ✅ COMPLIANT |
| No duplicate issuance | PDF retrieval fails after confirmed issuance | retain issued + recover FetchArtifact only | ✅ COMPLIANT |
| Authz downloads | Owning client downloads PDF | handler/FE; `ComposeArtifactService` wired in `cmd/api/main.go`; projection `artifactId` set when metadata exists | ✅ COMPLIANT |
| Authz downloads | Different client requests PDF | 404 non-enumeration + FE | ✅ COMPLIANT |
| Legal vs mock | Developer downloads mock PDF | SEM VALIDADE FISCAL — MOCK label | ✅ COMPLIANT |
| Production readiness | Artifact backend is not production-ready | `ValidateProductionComposition` + `ComposeArtifactService` production object readiness fail-closed | ✅ COMPLIANT |
| Production readiness | New issuance is disabled | `WithEnabled(false)` default; `TestArtifactRecoveryConfig_IssuanceFollowsFinalizationSwitch`; read path retained | ✅ COMPLIANT |

#### cloudware-fiscal-enablement (6 / 12)

| Requirement | Scenario | Test | Result |
|-------------|----------|------|--------|
| Scoped connections | Manager starts connection setup | OAuth start + hashed state | ✅ COMPLIANT |
| Scoped connections | Employee calls connection endpoint | employee denial tests | ✅ COMPLIANT |
| OAuth protected | OAuth callback succeeds | encrypted at rest; status without tokens | ✅ COMPLIANT |
| OAuth protected | Callback state is invalid | invalid/reused state; no code exchange | ✅ COMPLIANT |
| Evidenced FT/FR mapping | Frozen FR is mapped | FT/FR mapping fixtures | ✅ COMPLIANT |
| Evidenced FT/FR mapping | Unsupported receipt workflow is requested | unsupported kind/receipt rejection | ✅ COMPLIANT |
| Evidence-gated mutation | One applicable gate lacks evidence | `TestEnablementEvaluatorBlocksMutationBeforeHTTP` (httpCalls==0; mutationsEnabled default false) | ✅ COMPLIANT |
| Evidence-gated mutation | A gate is not applicable | `TestGateNotApplicableRequiresRationale` | ✅ COMPLIANT |
| No undocumented behavior | Provider times out before idempotency evidence exists | enablement block + no blind resubmit | ✅ COMPLIANT |
| No undocumented behavior | UI presents issued status | FE badges; AT only when evidenced | ✅ COMPLIANT |
| Normalized errors | Refresh token fails | connection → `action_required`; **document → `connection_action_required` not end-to-end asserted** | ⚠️ PARTIAL |
| Normalized errors | Active GC license is absent | error normalization present; **full remediation + frozen-intent retention E2E not asserted** | ⚠️ PARTIAL |

**Compliance summary**: **56/58** scenarios ✅ COMPLIANT; **2** ⚠️ PARTIAL; **0** ❌ UNTESTED / ❌ FAILING in completed WUs. Deferred (not counted as WU11 failure): WU12 credentialed acceptance / parent 13.2–13.4. Fully green requirements: **28/29**. Mid-change archive gate remains **FAIL** (50/58 tasks).

### WU11 Operations / rollout attention

| Design / hard constraint | Evidence | Result |
|--------------------------|----------|--------|
| Independent switches default OFF | `ParseRuntimeConfig` + `TestRuntimeConfig_IndependentSwitchesDefaultOff`; `.env.prod.example` / compose defaults false | ✅ |
| Production fail-closed (mock/local/default JWT/missing schema/keyring) | `ValidateProductionComposition` + production reject tests + worker startup gate | ✅ |
| `/ready` isolates provider health | `CoreAPIReady` ignores provider; `TestReadyHandler_IgnoresFiscalProviderOutage` | ✅ |
| Fiscal dependency status separate | `GET /fiscal/dependency-status` + route test | ✅ |
| Worker readiness gates | `EvaluateWorkerReadiness` + `TestWorkerStartupGate_*` | ✅ |
| Finalization default-off gate | `NewFinalizationService` enabled=false; `WithEnabled` from env | ✅ |
| Allowlisted logs / metrics | `TestFiscalLogEvent_AllowlistsSafeFieldsOnly`, `TestFiscalMetricsSnapshot_TracksOperationalSignals` | ✅ |
| Artifact composition shared API/worker | `ComposeArtifactService`; API wires non-nil local service outside production | ✅ |
| CI PostgreSQL 16 + migration smoke + race + builds | `.github/workflows/ci.yml` postgres:16-alpine; migrate smoke; `go test -race`; api/worker/migrate builds | ✅ (static CI evidence; not executed on this agent) |
| Runbook + four-spec traceability | `docs/fiscal-integration-runbook.md`, `docs/fiscal-integration-traceability.md` | ✅ |
| No live Cloudware mutations | `mutationsEnabled=false`; `FISCAL_CLOUDWARE_MUTATIONS` default false; WU12 blocked | ✅ |
| Race detector local | N/A without gcc/cgo | ⚠️ documented; CI authoritative |

### Correctness (Static Evidence — WU1–WU11)

| Requirement area | Status | Notes |
|------------------|--------|-------|
| Migration 011 + legacy isolation | ✅ Implemented | Prior PG green; CI migration smoke present |
| Exact decimal + policy calculator | ✅ Implemented | |
| Lifecycle domain (11 states) | ✅ Implemented | |
| Aggregate persistence / finalize+outbox | ✅ Implemented | |
| Provider-neutral ports + mock | ✅ Implemented | |
| Outbox worker lease/crash protocol | ✅ Implemented | |
| Immutable artifact archive + recovery | ✅ Implemented | |
| Additive fiscal HTTP API | ✅ Implemented | |
| Staff fiscal UI (WU8) | ✅ Implemented | |
| Client/admin UI (WU9) | ✅ Implemented | |
| Cloudware security shell (WU10) | ✅ Implemented | mutations fail-closed |
| Ops / CI / runbooks (WU11) | ✅ Implemented | switches, readiness, compose, CI, docs |
| API ArtifactService composition | ✅ Implemented | non-prod local; prod object readiness-only until WU12 SDK |
| Refresh → fiscal document state | ⚠️ Partial | connection `action_required` only |
| Live Cloudware enablement (WU12) | ❌ Blocked | parent 13.2–13.4 |

### Coherence (Design — WU11 ops)

| Decision | Followed? | Notes |
|----------|-----------|-------|
| Separate feature/finalization/worker/provider/Cloudware mutation switches | ✅ Yes | `RuntimeConfig` + env examples |
| Production refuses mock, local artifact, default secrets, missing schema/keyring | ✅ Yes | `ValidateProductionComposition` |
| Provider health not part of core `/ready` | ✅ Yes | `CoreAPIReady` + httptest proof |
| Worker readiness separate from API | ✅ Yes | worker startup validation |
| API and worker share domain/services; migrate one-shot | ✅ Yes | Dockerfiles + compose fiscal profile |
| Structured allowlisted logs; dependency status endpoint | ✅ Yes | observability + route |
| Dark-launch finalization off by default | ✅ Yes | `WithEnabled(false)` |
| Runbook stages + forward-fix rollback + legal-history preservation | ✅ Yes | runbook sections |
| Production object PDF bytes readiness-only until cloud SDK | ⚠️ Accepted deviation | documented; fail-closed until WU12 |
| `FiscalProviderGateway` type name | ⚠️ Deviation | retained `FiscalProvider` port (pre-WU11) |

### Issues Found

**CRITICAL**:
1. Change-level incompleteness: **8/58 tasks pending** (WU12 12.1–12.4 BLOCKED + parent 13.2–13.5) — archive gate must remain blocked. **Not a WU11 correctness failure.**

**WARNING**:
1. Refresh-token failure sets connection `action_required` but does not yet prove fiscal document transition to `connection_action_required`.
2. Active-GC-license path is normalized at HTTP error layer only; manager remediation + frozen-intent retention not end-to-end covered for Cloudware.
3. Fiscal* PostgreSQL suite skipped this run (`FISCAL_TEST_DATABASE_URL` unset in process; `.env` not read). CI postgres:16 is the authoritative PG path.
4. `go test -race` blocked locally (CGO/`gcc` missing). Non-race tests green; CI must run race.
5. `gentle-ai sdd-verify-validate` unavailable — cannot machine-admit report bytes.
6. Production object artifact byte path remains readiness-only (intentional until WU12/cloud SDK).
7. Design deviation retained: port naming `FiscalProvider` vs design `FiscalProviderGateway`.
8. WU11 authored above preferred ~600-line slice (`size:exception`, documented in apply-progress).

**SUGGESTION**:
1. Do **not** archive. Continue change only after parent **13.2–13.4**; then WU12. Parent **13.5** review remains open.
2. Optionally extend refresh failure path to assert document `connection_action_required` when an intent is in flight.
3. Ensure CI Linux runners remain the authority for `-race` and Fiscal* PG.
4. When validator tooling is available, re-admit this report bytes before final archive verify.

### Verdict

**FAIL**

Change-level verification cannot PASS while 8 tasks remain (archive completeness). **WU11 correctness is green**: focused fiscal packages exit 0; config/readiness matrix exit 0; `cmd/api`, `cmd/fiscal-worker`, and `cmd/migrate` build; switches default off; production fail-closed; `/ready` isolates provider; CI Postgres 16 + runbooks present; no live Cloudware mutations. Prior PDF ArtifactService / `/ready` PARTIALs closed. Honest PARTIALs remain for refresh→document state and license remediation E2E. **Not archive-ready.** Next: parent gates / continue change — **not** `sdd-archive`.

### Verification scope note

This is an honest **mid-change** verify requested after WU11 apply. It is **not** a claim that the OpenSpec change is complete. Native next step is continue the change (WU12 blocked on parent 13.2–13.4); do not archive.
