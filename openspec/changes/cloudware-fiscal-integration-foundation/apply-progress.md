# Apply Progress: Cloudware fiscal integration foundation

## Status
WU1–WU8 remain complete. **WU9 `client/admin UI` is complete**: own-only client fiscal status/PDF on `/my-invoices`, manager/admin fiscal integration settings page + AppShell nav, shared provider-neutral `FiscalStatusBadge`, connection/readiness service without token leakage. Notes and auth flows preserved. No commit or PR was created. Work stayed inside the WU9 boundary (stopped before WU10 Cloudware security shell).

## Completed tasks and persisted checkboxes
- [x] 1.1–1.5 WU1 schema/migration (retained).
- [x] 2.1–2.8 WU2 exact domain model (retained).
- [x] 3.1–3.4 WU3 aggregate persistence (retained).
- [x] 4.1–4.4 WU4 provider-neutral mock (retained).
- [x] 5.1–5.4 WU5 outbox worker (retained).
- [x] 6.1–6.4 WU6 artifact archive (retained).
- [x] 7.1–7.4 WU7 additive HTTP API (retained).
- [x] 8.1–8.4 WU8 staff fiscal UI (retained).
- [x] 9.1 RED — Extend my-invoices + AppShell tests; add admin fiscal page tests for own-only status/PDF, notes, hidden privileged actions, role-gated nav, readiness. <!-- sdd-owner: implementation -->
- [x] 9.2 GREEN — Update MyInvoicesList/Detail; add `fiscal-integration.service.ts`, `admin/integrations/fiscal/page.tsx`, AppShell manager/admin nav. <!-- sdd-owner: implementation -->
- [x] 9.3 TRIANGULATE — Non-owner 404, compromised artifacts, employee/client nav absence, OAuth-return strip, AT/e-Fatura only when evidenced. <!-- sdd-owner: implementation -->
- [x] 9.4 REFACTOR — Shared `FiscalStatusBadge`; connection details confined to admin page; full frontend test/lint/typecheck/build. <!-- sdd-owner: implementation -->
- [x] 13.1 Parent delivery decision (retained).

## Files changed (WU9 batch)
| File | Action | What was done |
|------|--------|---------------|
| `frontend/src/types/fiscal.ts` | Modified | Client presentation helpers, connection/readiness types, download gate |
| `frontend/src/types/fiscal.test.ts` | Modified | Helper triangulation for client/connection/download |
| `frontend/src/types/index.ts` | Modified | Re-export new fiscal helpers/types |
| `frontend/src/lib/services/fiscal-integration.service.ts` | Created | Manager/admin connection + readiness client |
| `frontend/src/lib/services/fiscal-integration.service.test.ts` | Created | Path/contract tests without secrets |
| `frontend/src/lib/services/index.ts` | Modified | Export fiscal-integration service |
| `frontend/src/components/fiscal/FiscalStatusBadge.tsx` | Created | Shared provider-neutral badge |
| `frontend/src/app/my-invoices/MyInvoicesListClient.tsx` | Modified | Summaries + simplified badges |
| `frontend/src/app/my-invoices/MyInvoicesListClient.test.tsx` | Modified | Badge + no-privileged-action coverage |
| `frontend/src/app/my-invoices/[id]/MyInvoiceDetailClient.tsx` | Modified | Status, PDF, notes preserved |
| `frontend/src/app/my-invoices/[id]/MyInvoiceDetailClient.test.tsx` | Modified | PDF/unavailable/compromised/404 cases |
| `frontend/src/app/admin/integrations/fiscal/page.tsx` | Created | Connection state, verify/disconnect, readiness, OAuth strip |
| `frontend/src/app/admin/integrations/fiscal/page.test.tsx` | Created | Readiness, OAuth, AT field gating |
| `frontend/src/components/layout/AppShell.tsx` | Modified | Manager/admin Integração fiscal → `/admin/integrations/fiscal` |
| `frontend/src/components/layout/AppShell.test.tsx` | Modified | Role-gated fiscal nav cases |
| `frontend/src/app/accounting/issued-invoices/page.tsx` | Modified | Reuse FiscalStatusBadge |
| `frontend/src/app/accounting/issued-invoices/[id]/FiscalActions.tsx` | Modified | Reuse FiscalStatusBadge |
| `openspec/.../tasks.md` | Modified | Mark 9.1–9.4 `[x]` |
| `openspec/.../apply-progress.md` | Modified | Cumulative WU1–WU9 progress |

## Verification
- Focused: WU9 vitest files → **45 passed**
- Full: `cd frontend && pnpm test -- --passWithNoTests` → **34 files / 175 tests passed**
- `pnpm lint` → PASS (exit 0)
- `pnpm typecheck` → PASS (exit 0)
- `pnpm build` → PASS (includes `/admin/integrations/fiscal`)

## Work Unit Evidence (WU9)

| Evidence | Result |
|---|---|
| Focused test command | `pnpm exec vitest run` on my-invoices list/detail, AppShell, admin fiscal page, fiscal-integration.service, fiscal.test → **45 passed** |
| Runtime harness | N/A — WU9 is Vitest/Testing Library against mocked API client; no browser E2E in tooling inventory |
| Rollback boundary | Revert WU9 frontend files (my-invoices fiscal panels, AppShell nav item, admin/integrations/fiscal, fiscal-integration.service, FiscalStatusBadge); notes/auth and staff WU8 UI remain |

## TDD Cycle Evidence (WU9)

| Task | Test file/layer | Safety net | RED | GREEN | TRIANGULATE | REFACTOR |
|---|---|---|---|---|---|---|
| 9.1 | my-invoices + AppShell + admin page tests / RTL | ✅ 18/18 prior my-invoices+AppShell PASS | ✅ Written (missing UI / missing page import) | — | Covered with 9.3 | N/A in RED |
| 9.2 | same + production modules | N/A (new admin page/service) | From 9.1 | ✅ list/detail/service/page/nav; focused PASS | — | Shared badge later |
| 9.3 | + service + fiscal helpers + AT/compromised | unit green | 404/compromised/OAuth/AT/employee-client nav | Behaviors PASS | ✅ All listed scenarios | Helpers extracted |
| 9.4 | full frontend quality | ✅ focused green | Approval tests preserved | lint+typecheck+test+build | — | FiscalStatusBadge shared; OAuth notice without setState-in-effect |

## Prior work unit evidence (retained)

### WU8
| Evidence | Result |
|---|---|
| Focused tests | 34 passed fiscalization UI |
| Full frontend | 151 tests (pre-WU9) |
| Rollback | Hide fiscal column/panel |

### WU7 / WU6 / WU5 / WU4 / WU3 / WU2 / WU1
Retained: additive HTTP API; artifact archive; outbox worker; mock provider; draft/finalization repos; domain packages; migration 011.

## Deviations, budget, and remaining work
- **Budget / size:exception**: Authored WU9 volume is estimated ~900–1,400 lines including tests/components (preferred ~600). Completing WU9 coherently required client list/detail + admin page + service + AppShell + RTL coverage (same pattern as WU3–WU8). Documented as **size:exception** under parent-authorized manual WU9 slice.
- **Readiness API**: Admin UI calls `/fiscal-integrations/cloudware/readiness`; when unavailable, shows fail-closed local guidance. Full Cloudware OAuth connect remains WU10.
- **PDF download**: Still depends on projection `artifactId` + API ArtifactService (same WU8 note).
- Delivery: Parent authorized manual WU9 only; agent created **no commits/PRs**. Stopped before WU10.

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
  verifyReport: []
  syncReport: []
artifacts: { proposal: done, specs: done, design: done, tasks: done, applyProgress: done, verifyReport: missing, syncReport: missing }
taskProgress: { total: 58, complete: 42, remaining: 16 }
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
    - "WU9 authored ~900-1400 lines (>600 preferred); size:exception like WU3–WU8"
    - "Cloudware readiness route may 503 until WU10; UI fail-closes with safe guidance"
    - "PDF download still needs artifactId on projection + API ArtifactService"
nextRecommended: sdd-verify
isNonAuthoritative: false
```

WU9 finish state is satisfied: owning clients see simplified fiscal status/PDF; managers/admins reach integration settings; employees/clients lack those controls. Next: `sdd-verify`, then WU10 only after parent/verify.

## Remaining implementation tasks (verbatim start of next unit)

- [ ] 10.1 RED — Add Cloudware adapter/crypto/connection tests for OAuth state, AES-GCM, gates blocking before HTTP. <!-- sdd-owner: implementation -->
