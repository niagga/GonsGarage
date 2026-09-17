# Client Auth Shell Traceability Document

## Context
Login (`/auth/login`) and registration (`/auth/register`) share one auth experience: shell, tokens, feedback, and a single `useAuth` consumer.

**Primary specification:** [openspec/specs/client-auth-shell/spec.md](openspec/specs/client-auth-shell/spec.md)

## Traceability Matrix

| Requirement | Implementation | Status |
| :--- | :--- | :--- |
| Shared page shell | Both wrap `AuthShell`; card/background from `tokens.css` vars | Compliant |
| Form / feedback | Shared `bannerSuccess` / `bannerError`; login `?message=`; field errors `text-destructive` | Compliant |
| Single auth consumer | Both `useAuth` from `@/stores` (Zustand); no register Context path | Compliant |
| Cross-nav secondary | `AuthShellFooter` + `Button variant="link"` on both | Compliant |
| Register copy pt_PT | `E-mail` / `O seu e-mail` match login; `Confirmar palavra-passe`; no `Error:` prefix | Compliant |
| Quality gate lint/typecheck | Not run in this audit | Not verified |

## Residuals
- Unused CSS modules (`login.module.css` / `register.module.css`) still contain dead ad-hoc colours.
- Register `isAuthenticated` redirect is `/employees`; login is `/dashboard` (out of the five shell requirements).
