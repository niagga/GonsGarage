# Task: UI Brand Shell Audit

## Context
Brand palette, light/dark on in-scope routes, lint warning budget. Spec: `openspec/specs/ui-brand-shell/spec.md`.

## Action Plan
1. [x] Audit documented brand tokens vs logo (Brand block in tokens.css).
2. [x] Audit shell + cars/appointments: no orphan hex, semantic tokens.
3. [x] Note lint/build quality gate (run only if cheap).
4. [x] Create traceability document. (`docs/ui-brand-shell-traceability.md`)

## Findings
- **COMPLIANT:** brand block + AppShell tokens.
- **GAP remediado:** cars (CarModal + car-details form) and appointments (`AppointmentCard` chips/actions, empty/new-modal) now use semantic tokens. No new tokens; `tokens.css` unchanged.
- Lint/build not run.

## Next Step
- Review (parent).

## Progress
- [x] Planning
- [x] Audit
- [x] Documentation refinement
- [x] GAP remediado (cars/appointments hex → tokens)
- [ ] Review
- [x] RDD Enabled (globally on)
