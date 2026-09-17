# Staff User Management UI Traceability Document

## Context
Staff who `canManageUsers` (`admin`/`manager`) reach `/admin/users` from AppShell without typing the URL.

**Primary specification:** [openspec/specs/staff-user-management-ui/spec.md](openspec/specs/staff-user-management-ui/spec.md)

**Archive:** [openspec/changes/archive/2026-04-21-ui-admin-users-discoverability/](openspec/changes/archive/2026-04-21-ui-admin-users-discoverability/)

## Traceability Matrix

| Requirement | Implementation | Status |
| :--- | :--- | :--- |
| Nav «Utilizadores» iff `canManageUsers` | `AppShell.tsx`; tests manager/admin vs client/employee | Compliant |
| Active nav on `/admin/users` | `activeNav="admin_users"` | Compliant |
| Single nav source | Inline AppShell; no `navigation.ts` | Compliant |
| Toolbar + modal create | `page.tsx` + `ProvisionUserModal`; cancel stays on page | Compliant |

## Residual
- `frontend_structure_analysis.md` may still mention `navigation.ts` (file absent).
