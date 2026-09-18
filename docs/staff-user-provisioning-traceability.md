# Staff User Provisioning Traceability Document

## Context
Authenticated `POST /api/v1/admin/users` with a caller × target role matrix. No self-service **admin** creation on this flow.

**Primary specification:** [openspec/specs/staff-user-provisioning/spec.md](openspec/specs/staff-user-provisioning/spec.md)

**Archive:** [openspec/changes/archive/2026-04-20-admin-provision-user-roles/](openspec/changes/archive/2026-04-20-admin-provision-user-roles/)

## Surface
`GinBearerJWT` → `RequireStaffManagers` → `AdminUserHandler.ProvisionUser` → `AuthService.ProvisionUser`

## Matrix (audited)

| Caller → target | Spec | Status |
| :--- | :--- | :--- |
| admin → manager / employee / client | MUST | Compliant |
| admin → admin | MUST NOT | Compliant (400) |
| manager → employee / client | MUST | Compliant |
| manager → manager / admin | MUST NOT | Compliant |
| employee / client → anyone | MUST NOT | Compliant (403) |
| unauthenticated | MUST NOT 2xx | Compliant (401) |
| unknown target role | MUST NOT 2xx | Compliant (400) |

## Residuals
- HTTP matrix coverage is in `backend/internal/handler/admin_user_provision_test.go` (admin→manager/employee/client, manager→employee/client, manager→admin 400, unknown target `superuser` 400, unknown JWT role 403).
- Role normalisation is trim-only, not case-fold (`Client` → 400).
- Public `Register` no longer allows `admin` or `manager` (self-service is `client` default and `employee` only). Staff creation stays on `POST /api/v1/admin/users` (`ProvisionUser`).
