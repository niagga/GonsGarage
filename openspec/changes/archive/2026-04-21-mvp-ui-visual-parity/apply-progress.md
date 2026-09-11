# Apply progress — mvp-ui-visual-parity

**Mode:** Strict TDD (`openspec/config.yaml`).

## TDD cycle evidence

### Phase 1

| Task | Layer | RED | GREEN | Refactor |
|------|-------|-----|--------|----------|
| 1.1 | Doc | N/A | ✅ `docs/ui-audit-mvp-dark.md` | — |
| 1.2 | CSS tokens | N/A | ✅ `tokens.css` dark + loading vars | — |
| 1.3 | CSS | N/A | ✅ `utilities.css` `.app-loading` | — |
| 1.4 | React | ✅ `AppLoading.test.tsx` | ✅ `AppLoading.tsx` | — |
| 1.5 | Pages | N/A | ✅ dashboard, cars, car `[id]` | — |

### Phase 2

| Task | Layer | RED | GREEN | Refactor |
|------|-------|-----|--------|----------|
| 2.1 | employees | N/A (CSS inline → tokens) | ✅ hex → `var(--chip-*\|--color-error\|--surface-*)`, loading `AppLoading` | — |
| 2.2 | client shell | N/A | ✅ `DashboardLayout.module.css` + `client/page.tsx` loading | — |
| 2.3 | accounting | N/A | ✅ `layout.tsx` + 4× `[id]/page` `AppLoading`; `accounting.module.css` já tokens | — |
| 2.4 | dashboard CSS | N/A | ✅ `.spinner` removido de `dashboard.module.css` e `car-details.module.css` | — |
| 2.5 | inventory | N/A | ✅ `docs/ui-loader-exceptions.md`; `rg` 0 em `app/` | — |
| 2.6 | smoke | N/A | ✅ notas em `ui-audit-mvp-dark.md` | — |

### Phase 3 (spike Next 16 + Tailwind v4)

| Task | Layer | RED | GREEN | Refactor |
|------|-------|-----|--------|----------|
| 3.1 | ADR | N/A | ✅ `docs/adr/0001-next16-tailwind4-spike.md` | — |
| 3.2 | deps + CSS pipeline | N/A | ✅ rama `spike/next16-tailwind4`: `next@16.2.4`, `eslint-config-next@16.2.4`, `tailwindcss` + `@tailwindcss/postcss`, `postcss.config.mjs`, `@import "tailwindcss"` en `globals.css` | — |
| 3.3 | CI local | N/A | ⚠️ lint **falla** (26 errores hooks); typecheck / test / build **OK** — volcado en ADR | — |
| 3.4 | decisión | N/A | ✅ ADR: **NO-GO** merge a `main` hasta lint verde; pasos si GO | — |

### Phase 4 — Shadcn (Tailwind 3 + primitives)

| Task | Layer | RED | GREEN | Refactor |
|------|-------|-----|--------|----------|
| 4.1 | ADR | N/A | ✅ `docs/adr/0002-shadcn-stack.md` | — |
| 4.2 | tooling + UI | ✅ `utils.test.ts`, `button.test.tsx` | ✅ `tailwind.config.ts`, `postcss.config.mjs`, `shadcn-theme.css`, `components.json`, `button/input/label/dialog.tsx`, `lib/utils.ts` | — |
| 4.3 | Doc | N/A | ✅ `docs/ui-shadcn-theme.md` | — |
| 4.4 | auth | N/A (tests existentes Login/Register) | ✅ `LoginForm.tsx`, `register/page.tsx` con Shadcn | — |
| 4.5 | shell | N/A | ✅ `AppShell.tsx` nav + logout `Button` | — |
| 4.6 | rutas | N/A | ✅ `dashboard/page.tsx`, `CarsContainer`, `appointments/page.tsx` | — |
| 4.7 | employees/client/accounting | ✅ `ClientDashboard.test.tsx`, `DashboardLayout.test.tsx`, `employees/page.test.tsx` (approval + interação; baseline 30 → 38 tests) | ✅ `employees/page.tsx` (`Button`/`Input`/`Label`, modal); `ClientDashboard.tsx`; `DashboardLayout.tsx` | ✅ Mantido verde após migração |
| 4.8 | inventário | N/A | ✅ `docs/ui-forms-shadcn-inventory.md` | — |
| 4.9 | README | N/A | ✅ `frontend/README.md` secção Shadcn | — |

### Phase 5 — Cierre

| Task | Layer | RED | GREEN | Refactor |
|------|-------|-----|--------|----------|
| 5.1 | CI | N/A | ✅ `pnpm lint` (0 erros), `typecheck`, `test` 38, `build` | — |
| 5.2 | Doc audit | N/A | ✅ matriz actualizada em `ui-audit-mvp-dark.md` | — |

## Files changed (cumulative)

**Fase 1:** ver histórico git.  
**Fase 2:** `employees/page.tsx`, `client/page.tsx`, `components/layouts/DashboardLayout.module.css`, `accounting/layout.tsx`, `accounting/**/[id]/page.tsx` (4), `my-invoices/layout.tsx`, `my-invoices/[id]/page.tsx`, `appointments/page.tsx`, `appointments/new/page.tsx`, `app/page.tsx`, `dashboard.module.css`, `car-details.module.css`, `docs/ui-loader-exceptions.md`, `docs/ui-audit-mvp-dark.md`.

**Fase 4–5 (esta sesión):** Tailwind 3 + shadcn primitives, migración auth/shell/dashboard/cars/appointments/accounting botones, **4.7** `employees/page.tsx`, `client/components/ClientDashboard.tsx`, `components/layouts/DashboardLayout.tsx`, tests `ClientDashboard`/`DashboardLayout`/`employees/page`, docs ADR0002 + tema + inventario, README, tests `utils`/`button`, matriz auditoría.

**Estado:** **26/26** tarefas. Fase 4.7 fechada (`employees` + área `client`). Listo para verify/archive.

**Pós-verify (2026-04-21):** `docs/ui-audit-mvp-dark.md` + `proposal.md` alinhados; `employees/page.tsx` (hooks/catches, sem `style jsx`); `button.test.tsx` comportamental; `@vitest/coverage-v8` + `vitest.config.ts` coverage `src/`; `openspec/config.yaml` + `frontend/README.md` atualizados; `verify-report.md` addendum.
