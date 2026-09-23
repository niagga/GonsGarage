# Fiscal integration scenario traceability

Maps acceptance scenarios from the four OpenSpec change specs to implementation and operational evidence for `cloudware-fiscal-integration-foundation`.

Specs:

1. [`openspec/changes/cloudware-fiscal-integration-foundation/specs/fiscal-documents/spec.md`](../openspec/changes/cloudware-fiscal-integration-foundation/specs/fiscal-documents/spec.md)
2. [`openspec/changes/cloudware-fiscal-integration-foundation/specs/fiscal-provider-foundation/spec.md`](../openspec/changes/cloudware-fiscal-integration-foundation/specs/fiscal-provider-foundation/spec.md)
3. [`openspec/changes/cloudware-fiscal-integration-foundation/specs/fiscal-artifacts/spec.md`](../openspec/changes/cloudware-fiscal-integration-foundation/specs/fiscal-artifacts/spec.md)
4. [`openspec/changes/cloudware-fiscal-integration-foundation/specs/cloudware-fiscal-enablement/spec.md`](../openspec/changes/cloudware-fiscal-integration-foundation/specs/cloudware-fiscal-enablement/spec.md)

Operational procedures: [`docs/fiscal-integration-runbook.md`](fiscal-integration-runbook.md).

## fiscal-documents

| Scenario theme | Spec requirement | Evidence / control |
|----------------|------------------|--------------------|
| Draft preparation vs privileged finalization | Explicit finalization; employee denial | `FinalizationService` + handler authz; finalization default off (`FISCAL_FINALIZATION_ENABLED`) |
| Exact policy arithmetic fail-closed | Approved policy required | Domain calculator + readiness issues on projection |
| Intent uniqueness / race | One current intent | Migration partial unique + repository finalize |
| Frozen immutability | No post-finalization rewrite | DB triggers + repository guards |
| Additive HTTP contracts | Nested routes; legacy unchanged | `RegisterInvoiceAndFiscalRoutes`; handler tests |
| Staff/client presentation | Provider-neutral projections | Document service + FE fiscal components |

## fiscal-provider-foundation

| Scenario theme | Spec requirement | Evidence / control |
|----------------|------------------|--------------------|
| Atomic finalize + outbox | Same transaction | `FinalizeDraft` repository |
| Lease / SKIP LOCKED | One active worker lease | Outbox claim + worker tests; CI PostgreSQL |
| Ambiguous crash → unknown | No blind Issue retry | Worker recover-stale path |
| Deterministic mock (non-prod) | Stable FT/FR/PDF; labeled non-legal | `integration/fiscal/mock`; production rejects mock |
| Production selects mock | Must fail closed | `RuntimeConfig.ValidateProductionComposition`, `ResolveProviderKey` |
| Provider outage isolation | Core API stays ready | `/ready` via `CoreAPIReady`; `/fiscal/dependency-status` |

## fiscal-artifacts

| Scenario theme | Spec requirement | Evidence / control |
|----------------|------------------|--------------------|
| Immutable archive + checksum | Put-if-absent; SHA-256 | `ArtifactService` + local/object readiness |
| Issued survives PDF failure | Recovery never calls Issue | Worker artifact recovery path |
| Mock vs legal labeling | Visible non-legal mock | Mock PDF label; production classification guards |
| Production storage readiness | Fail closed without ACL/encryption/retention/backup/access log | `fiscalartifact.ProductionReadiness` |
| Authorized download | Own-only / staff roles; no URL leakage | Document service `OpenArtifact`; projection `artifactId` |

## cloudware-fiscal-enablement

| Scenario theme | Spec requirement | Evidence / control |
|----------------|------------------|--------------------|
| OAuth connect / callback | Hashed one-time state | Connection service + crypto AAD |
| Gate fail-closed before HTTP | Incomplete gates block mutations | `CloudwareEnablementEvaluator`; `mutationsEnabled=false` default |
| FT/FR mapping only | No FS/credit/standalone receipt | Cloudware adapter fixtures |
| Readiness endpoint | Gate names + guidance | `GET /api/v1/fiscal-integrations/cloudware/readiness` |
| Live enablement | Credentialed acceptance | **Blocked** — WU12 + parent 13.2–13.4 |

## Ops / CI matrix (WU11)

| Config posture | Expected |
|----------------|----------|
| Feature off | Composition validation skipped; legacy API unchanged |
| Mock non-production | Allowed; finalization still off unless explicitly enabled |
| Cloudware fail-closed production | Mutations off; mock/local/default secrets rejected |
| Provider outage | `/ready` OK when DB healthy; dependency-status reports `provider_unhealthy` |
| Artifact recovery settings | Issuance/archive gated by `FISCAL_FINALIZATION_ENABLED` |

CI: `.github/workflows/ci.yml` — PostgreSQL 16, migration 011 smoke, `go test -race`, API/worker/migrate builds, frontend lint/typecheck/test/build.
