# Task: Staff User Management UI Audit

## Context
Shell nav + `/admin/users` toolbar/modal for `canManageUsers`. Spec: `openspec/specs/staff-user-management-ui/spec.md`.

## Action Plan
1. [x] Audit AppShell nav (Utilizadores) iff `canManageUsers`; client hidden; active state.
2. [x] Audit single navigation source of truth.
3. [x] Audit list-first toolbar + modal provisioning (not inline-only form).
4. [x] Create or update traceability document. (`docs/staff-user-management-ui-traceability.md`)

## Findings
- **COMPLIANT** on nav gate, active state, single source, toolbar+modal.

## Next Step
- Closed. Next module: `client-auth-shell`.

## Progress
- [x] Planning
- [x] Audit
- [x] Documentation refinement
- [x] Review
- [x] RDD Enabled (globally on)
