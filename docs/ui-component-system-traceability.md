# UI Component System Traceability Document

## Context
Shadcn-style canonical primitives in `frontend/src/components/ui/`, mapped to brand tokens.

**Primary specification:** [openspec/specs/ui-component-system/spec.md](openspec/specs/ui-component-system/spec.md)

## Traceability Matrix

| Requirement | Implementation | Status |
| :--- | :--- | :--- |
| Foundation dir | `dialog.tsx`, `button.tsx`, `input.tsx`, `label.tsx`, `select.tsx`; `components.json` | Compliant |
| Auth on system primitives | Login/register use `Button`/`Input`/`Label` | Compliant |
| Parts create | `PartCreateModal` uses `Dialog`/`Button`/`Input`/`Label` | Compliant |
| Workshop Nova visita | `Dialog` + `Button` on `/workshop` | Compliant |
| Theme mapping | `docs/ui-shadcn-theme.md` + `shadcn-theme.css` | Compliant |

## Residuals
- Native `<select>` still used (register Perfil, parts UoM, workshop car picker) while `select.tsx` exists.
- Legacy widgets remain in the same `components/ui/` tree (`ConfirmModal`, CSS-module FormField).
- Theme table does not name `--brand-navy`.
