# Apply Progress: Cloudware fiscal integration foundation

## Status
WU1–WU7 remain complete. **WU8 `staff fiscal UI` is complete**: provider-neutral fiscalization service + types, issued-invoices list batch summaries/badges, detail `FiscalDraftForm` / `FiscalActions` (decimal-string drafts, readiness, role-gated privileged actions, 202→capped polling, double-submit guard, mock label, PDF download wiring). Operational invoice columns/create/edit controls preserved. No commit or PR was created. Work stayed inside the WU8 boundary (stopped before WU9 client/admin UI).

## Completed tasks and persisted checkboxes
- [x] 1.1–1.5 WU1 schema/migration (retained).
- [x] 2.1–2.8 WU2 exact domain model (retained).
- [x] 3.1–3.4 WU3 aggregate persistence (retained).
- [x] 4.1–4.4 WU4 provider-neutral mock (retained).
- [x] 5.1–5.4 WU5 outbox worker (retained).
- [x] 6.1–6.4 WU6 artifact archive (retained).
- [x] 7.1–7.4 WU7 additive HTTP API (retained).
- [x] 8.1 RED — Vitest/Testing Library: `fiscalization.service.test.ts`, `issued-invoices/page.test.tsx`, `[id]/FiscalDraftForm.test.tsx`, `FiscalActions.test.tsx`, `page.test.tsx` for summaries, FT/FR decimals, readiness, roles, 202 polling, cancellation, double-click, operational edits. <!-- sdd-owner: implementation -->
- [x] 8.2 GREEN — `types/fiscal.ts`, `fiscalization.service.ts`, `FiscalDraftForm.tsx`, `FiscalActions.tsx`; extend list + detail pages. <!-- sdd-owner: implementation -->
- [x] 8.3 TRIANGULATE — employee vs manager/admin, FT/FR/exempt, stale 409, unavailable/unknown, source refs, server totals, mock labels, capped polling stop; `fiscal.test.ts` helpers. <!-- sdd-owner: implementation -->
- [x] 8.4 REFACTOR — reuse Button/form/apiClient patterns; preserve create modal + operational fields; pt_PT provider-neutral copy; `pnpm test`, `pnpm lint`, `pnpm typecheck`. <!-- sdd-owner: implementation -->
- [x] 13.1 Parent delivery decision (retained).

## Files changed (WU8 batch)
| File | Action | What was done |
|------|--------|---------------|
| `frontend/src/types/fiscal.ts` | Created | Decimal-string DTOs, presentation helpers, privileged-action gate |
| `frontend/src/types/fiscal.test.ts` | Created | Helper triangulation (labels, polling states, roles) |
| `frontend/src/types/index.ts` | Modified | Re-export fiscal types/helpers |
| `frontend/src/lib/services/fiscalization.service.ts` | Created | Draft/actions/summaries/artifact client |
| `frontend/src/lib/services/fiscalization.service.test.ts` | Created | Contract tests for paths + decimal payloads + PDF fetch |
| `frontend/src/lib/services/index.ts` | Modified | Export fiscalization service |
| `frontend/src/app/accounting/issued-invoices/page.tsx` | Modified | Batch summaries + Fiscalização column |
| `frontend/src/app/accounting/issued-invoices/page.test.tsx` | Modified | Summary badges + legacy guidance; keep create-modal safety net |
| `frontend/src/app/accounting/issued-invoices/[id]/page.tsx` | Modified | Fiscal panel beside operational edit form |
| `frontend/src/app/accounting/issued-invoices/[id]/page.test.tsx` | Created | Operational controls + projection load |
| `frontend/src/app/accounting/issued-invoices/[id]/FiscalDraftForm.tsx` | Created | Manual FT/FR lines, decimals, readiness, source refs |
| `frontend/src/app/accounting/issued-invoices/[id]/FiscalDraftForm.test.tsx` | Created | Draft form behavioral coverage |
| `frontend/src/app/accounting/issued-invoices/[id]/FiscalActions.tsx` | Created | Privileged actions, polling, double-submit, PDF, mock label |
| `frontend/src/app/accounting/issued-invoices/[id]/FiscalActions.test.tsx` | Created | Role/polling/conflict/mock coverage |
| `openspec/.../tasks.md` | Modified | Mark 8.1–8.4 `[x]` |
| `openspec/.../apply-progress.md` | Modified | Cumulative WU1–WU8 progress |

## Verification
- Focused: `pnpm exec vitest run` on WU8 test files → **34 passed**
- Full: `cd frontend && pnpm test -- --passWithNoTests` → **32 files / 151 tests passed**
- `pnpm lint` → PASS (exit 0)
- `pnpm typecheck` → PASS (exit 0)

## Work Unit Evidence (WU8)

| Evidence | Result |
|---|---|
| Focused test command | `pnpm exec vitest run src/lib/services/fiscalization.service.test.ts src/types/fiscal.test.ts src/app/accounting/issued-invoices/page.test.tsx src/app/accounting/issued-invoices/[id]/FiscalDraftForm.test.tsx src/app/accounting/issued-invoices/[id]/FiscalActions.test.tsx src/app/accounting/issued-invoices/[id]/page.test.tsx` → **34 passed** |
| Runtime harness | N/A — WU8 is Vitest/Testing Library against mocked API client; no browser E2E in tooling inventory |
| Rollback boundary | Hide fiscal column/panel (revert WU8 frontend files); operational issued-invoice list/create/edit remain; no backend schema change |

## TDD Cycle Evidence (WU8)

| Task | Test file/layer | Safety net | RED | GREEN | TRIANGULATE | REFACTOR |
|---|---|---|---|---|---|---|
| 8.1 | service + page + `[id]/*` tests / unit+RTL | ✅ prior issued-invoices create-modal tests PASS (123 suite baseline) | ✅ Written (import fail / missing UI) | — | Covered with 8.3 | N/A in RED |
| 8.2 | same + production modules | N/A (new) | From 8.1 | ✅ types/service/forms/pages; focused PASS | — | Shared Button/styles/apiClient |
| 8.3 | + `fiscal.test.ts` | unit green | Role/FT/FR/exempt/409/unknown/mock/poll cases | Behaviors PASS | ✅ All listed scenarios | Helpers extracted |
| 8.4 | full frontend quality | ✅ focused green | Approval tests preserved | lint+typecheck+full test | — | Exports + alert role + submit ref |

## Prior work unit evidence (retained)

### WU7
| Evidence | Result |
|---|---|
| Focused tests | handler/fiscal + invoice + domain PASS |
| Runtime harness | N/A httptest stubs |
| Rollback | Disable `FISCAL_FEATURE_ENABLED` |

### WU6 / WU5 / WU4 / WU3 / WU2 / WU1
Retained: artifact archive; outbox worker; mock provider; draft/finalization repos; domain packages; migration 011.

## Deviations, budget, and remaining work
- **Budget / size:exception**: Authored WU8 volume is estimated ~1,200–1,800 lines including tests/components (preferred ~600). Completing WU8 coherently required service + types + list/detail wiring + RTL coverage (same pattern as WU3–WU7). Documented as **size:exception** under parent-authorized manual WU8 slice.
- **PDF download**: UI wires `downloadArtifact` when `artifactId`/`artifactStatus` present. WU7 projection still omits `artifactId`; API process `ArtifactService` may remain nil (503 / unavailable guidance). Optional note: PDF UX needs projection `artifactId` + API ArtifactService composition for reliable downloads.
- Delivery: Parent authorized manual WU8 only; agent created **no commits/PRs**. Stopped before WU9.

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
taskProgress: { total: 58, complete: 38, remaining: 20 }
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
    - "WU8 authored ~1200-1800 lines (>600 preferred); size:exception like WU3–WU7"
    - "PDF download needs artifactId on projection + API ArtifactService (may be nil)"
nextRecommended: sdd-verify
isNonAuthoritative: false
```

WU8 finish state is satisfied: staff can prepare drafts and authorized managers/admins can act using provider-neutral projections on issued-invoice pages. Next: `sdd-verify`, then WU9 only after parent/verify.

## Remaining implementation tasks (verbatim start of next unit)

- [ ] 9.1 RED — Extend my-invoices tests and add admin fiscal integration page tests for own-only status/PDF and role-gated settings. <!-- sdd-owner: implementation -->
