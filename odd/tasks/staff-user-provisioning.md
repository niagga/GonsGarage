# Task: Staff User Provisioning Audit

## Context
Authenticated `POST /api/v1/admin/users` with caller×target role matrix. Spec: `openspec/specs/staff-user-provisioning/spec.md`.

## Action Plan
1. [x] Audit JWT required (no token MUST NOT 2xx).
2. [x] Audit caller matrix: admin vs manager vs employee vs client.
3. [x] Audit no admin escalation and unknown roles rejected.
4. [x] Create traceability document. (`docs/staff-user-provisioning-traceability.md`)

## Findings
- **COMPLIANT** on JWT, matrix, no admin via this flow, unknown roles.
- Residual: some HTTP test holes; Register (other endpoint) may still allow admin.

## Next Step
- Closed with UI sibling (`odd/tasks/staff-user-management-ui.md`). Next module: `client-auth-shell`.

## Progress
- [x] Planning
- [x] Audit
- [x] Documentation refinement
- [x] Review
- [x] RDD Enabled (globally on)
