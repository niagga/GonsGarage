# Apply Progress: Cloudware fiscal integration foundation

## Status
WU1–WU9 remain complete. **WU10 `Cloudware security shell` is complete**: AES-256-GCM credential AAD + keyring rotation, hashed one-time OAuth state with actor/redirect/expiry binding, Cloudware adapter FT/FR mapping with fail-closed `CloudwareEnablementEvaluator`, bounded fake-HTTP client, readiness/connect/callback routes, and secret-free responses. Fake HTTP and test credentials only — no live Cloudware mutations. No commit or PR was created. Stopped before WU11.

## Completed tasks and persisted checkboxes
- [x] 1.1–1.5 WU1 schema/migration (retained).
- [x] 2.1–2.8 WU2 exact domain model (retained).
- [x] 3.1–3.4 WU3 aggregate persistence (retained).
- [x] 4.1–4.4 WU4 provider-neutral mock (retained).
- [x] 5.1–5.4 WU5 outbox worker (retained).
- [x] 6.1–6.4 WU6 artifact archive (retained).
- [x] 7.1–7.4 WU7 additive HTTP API (retained).
- [x] 8.1–8.4 WU8 staff fiscal UI (retained).
- [x] 9.1–9.4 WU9 client/admin UI (retained).
- [x] 10.1 RED — Cloudware/crypto/connection tests for OAuth state, AES-GCM AAD, gates blocking before HTTP. <!-- sdd-owner: implementation -->
- [x] 10.2 GREEN — `fiscal_credentials.go`, connection repo/service/handler, `integration/fiscal/cloudware/*` OAuth/DTOs/mapping/errors/`CloudwareEnablementEvaluator`. <!-- sdd-owner: implementation -->
- [x] 10.3 TRIANGULATE — finalize/void/PDF fixtures, FR associated-receipt, active-license, unknown→ambiguous, no FS/receipt/correction, gate `not_applicable` rationale. <!-- sdd-owner: implementation -->
- [x] 10.4 REFACTOR — bounded TLS client, allowlist, response limits, no mutation retries, secret-free logs/API, production keyring validation, import isolation tests. <!-- sdd-owner: implementation -->
- [x] 13.1 Parent delivery decision (retained).

## Files changed (WU10 batch)
| File | Action | What was done |
|------|--------|---------------|
| `backend/internal/platform/crypto/fiscal_credentials.go` | Modified | AAD binding, keyring rotation, production keyring validation |
| `backend/internal/platform/crypto/fiscal_credentials_aad_test.go` | Created | AAD/rotation/production keyring RED→GREEN |
| `backend/internal/core/ports/fiscal_oauth.go` | Created | OAuth state/token/readiness ports |
| `backend/internal/core/ports/fiscal_connection_service.go` | Modified | OAuth + readiness service methods |
| `backend/internal/service/fiscal/connection_service.go` | Modified | OAuth start/callback/refresh, AAD encrypt, readiness |
| `backend/internal/service/fiscal/connection_oauth_test.go` | Created | Hashed state, invalid/reused callback, refresh fail-closed |
| `backend/internal/integration/fiscal/cloudware/*` | Created | Mapping, enablement, errors, HTTP client, adapter, isolation |
| `backend/internal/repository/postgres/fiscal_connection_repository.go` | Modified | OAuth authorization save/consume |
| `backend/internal/handler/fiscal_integration_handler.go` | Modified | readiness/connect/callback/status/verify/disconnect |
| `backend/cmd/api/main.go` | Modified | Cloudware integration routes + production keyring gate |
| `openspec/.../tasks.md` | Modified | Mark 10.1–10.4 `[x]` |
| `openspec/.../apply-progress.md` | Modified | Cumulative WU1–WU10 progress |

## Verification
- Focused: `go test ./internal/platform/crypto/ ./internal/service/fiscal/ ./internal/integration/fiscal/cloudware/ ./internal/handler/ -count=1` → **PASS**
- `go build ./cmd/api` → **PASS**
- `go test -race` → **N/A** on this Windows agent (requires cgo/`CGO_ENABLED=1` + gcc); documented, not silently skipped as green

## Work Unit Evidence (WU10)

| Evidence | Result |
|---|---|
| Focused test command | `go test ./internal/platform/crypto/ ./internal/service/fiscal/ ./internal/integration/fiscal/cloudware/ ./internal/handler/ -count=1` → **PASS** (crypto, fiscal service, cloudware, handler) |
| Runtime harness | N/A — WU10 uses `httptest` fake HTTP + in-memory OAuth/token stubs; live Cloudware mutations remain blocked (WU12 / parent 13.2–13.4) |
| Rollback boundary | Revert WU10 crypto AAD/keyring, cloudware package, OAuth connection methods/repo, Cloudware handler routes/wiring; mock provider path and WU1–WU9 remain |

## TDD Cycle Evidence (WU10)

| Task | Test file/layer | Safety net | RED | GREEN | TRIANGULATE | REFACTOR |
|---|---|---|---|---|---|---|
| 10.1 | `fiscal_credentials_aad_test.go`, `connection_oauth_test.go`, `cloudware/cloudware_test.go` / Unit | ✅ crypto+fiscal PASS pre-change | ✅ Written (undefined types/methods) | — | Covered with 10.3 | N/A in RED |
| 10.2 | same + production modules | N/A (new cloudware pkg) | From 10.1 | ✅ packages PASS | — | Bounded client later |
| 10.3 | finalize/void/PDF/FR/license/ambiguous/FS/N/A rationale | unit green | Contract fixtures + error cases | Behaviors PASS | ✅ All listed scenarios | Helpers extracted |
| 10.4 | HTTP allowlist/no-retry, isolation import walk, production keyring | ✅ focused green | Approval via existing round-trip | PASS + `go build ./cmd/api` | — | Isolation test; keyring validation in API wiring |

## Prior work unit evidence (retained)

### WU9
| Evidence | Result |
|---|---|
| Focused tests | 45 passed frontend WU9 |
| Full frontend | 175 tests |
| Rollback | my-invoices + admin fiscal + AppShell |

### WU8–WU1
Retained: staff UI; additive HTTP API; artifact archive; outbox worker; mock provider; draft/finalization repos; domain packages; migration 011.

## Deviations, budget, and remaining work
- **Budget / size:exception**: Authored WU10 volume exceeds the preferred ~600-line budget (crypto+OAuth+cloudware adapter+handler routes+tests). Completing WU10 coherently required the full security shell (same pattern as WU3–WU9). Documented as **size:exception** under parent-authorized manual WU10 slice.
- **Live Cloudware**: Explicitly out of scope. Adapter `mutationsEnabled` defaults false; gates remain pending from migration seed.
- **Race detector**: N/A without gcc/cgo on this agent host; CI with cgo remains authoritative.
- Delivery: Parent authorized manual WU10 only; agent created **no commits/PRs**. Stopped before WU11.

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
  verifyReport: [openspec/changes/cloudware-fiscal-integration-foundation/verify-report.md]
  syncReport: []
artifacts: { proposal: done, specs: done, design: done, tasks: done, applyProgress: done, verifyReport: present, syncReport: missing }
taskProgress: { total: 58, complete: 46, remaining: 12 }
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
    - "WU10 authored above ~600 preferred lines; size:exception like WU3–WU9"
    - "Live Cloudware mutations remain fail-closed until WU12 / parent 13.2–13.4"
    - "go test -race N/A without gcc/cgo on this agent host"
nextRecommended: sdd-verify
isNonAuthoritative: false
```

WU10 finish state is satisfied: encrypted credentials with AAD/keyring, OAuth hashed state, Cloudware FT/FR mapping, fail-closed gates before HTTP, readiness endpoint for admin UI, no live mutations. Next: `sdd-verify`, then WU11 only after parent/verify.

## Remaining implementation tasks (verbatim start of next unit)

- [ ] 11.1 RED — Add deployment/CI/config tests for feature/finalization/worker flags, mock isolation, Cloudware fail-closed production defaults, and migration/schema guards. <!-- sdd-owner: implementation -->
