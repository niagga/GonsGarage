# Task: Client Auth Shell Audit

## Context
Login (`/auth/login`) and register (`/auth/register`) as one coherent auth experience. Spec: `openspec/specs/client-auth-shell/spec.md`.

## Action Plan
1. [x] Audit shared page shell + token-driven visuals.
2. [x] Audit form/feedback consistency and pt_PT copy (email wording, confirm password, no English `Error:` prefix).
3. [x] Audit single `useAuth` consumer and cross-nav secondary actions.
4. [x] Create traceability document. (`docs/client-auth-shell-traceability.md`)

## Findings
- **COMPLIANT** on shell, feedback, `useAuth`, cross-nav, pt_PT copy.
- Quality gate (`pnpm lint` / `typecheck`) not run.
- Residual: unused CSS modules; authenticated register redirect `/employees` vs login `/dashboard`.

## Next Step
- Remaining catalogue modules: `ui-brand-shell`, `ui-component-system`, `ui-accounting-staff`, `mvp-role-access` (already updated during billing).

## Progress
- [x] Planning
- [x] Audit
- [x] Documentation refinement
- [x] Review
- [x] RDD Enabled (globally on)
