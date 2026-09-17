# UI Brand Shell Traceability Document

## Context
Brand palette as a single source of truth, plus light/dark coherence on cars and appointments.

**Primary specification:** [openspec/specs/ui-brand-shell/spec.md](openspec/specs/ui-brand-shell/spec.md)

## Traceability Matrix

| Requirement | Implementation | Status |
| :--- | :--- | :--- |
| Documented brand palette | `frontend/src/styles/tokens.css` `## Brand (vs. logo)` — navy, accent, signal | Compliant |
| AppShell no orphan brand hex | `AppShell.module.css` uses tokens (`--color-primary`, `--brand-signal`, …) | Compliant |
| Cars theme coherence | List cards tokenized; **CarModal** error uses chip-danger; **car-details** staff form uses surface/border/text tokens | Remediado |
| Appointments theme coherence | Live `AppointmentCard.module.css` chips/actions, `EmptyState`, `NewAppointmentModal` overlay use semantic tokens | Remediado |
| Lint/build warning budget | Not run in this audit | Not verified |

## GAP detail
Cars/appointments GAP remediado: orphan hex replaced with existing semantic tokens (no new tokens; `tokens.css` unchanged).
- `frontend/src/app/cars/components/CarModal.tsx` — error alert → `--chip-danger-*`
- `frontend/src/app/cars/[id]/car-details.module.css` — staff form → `--surface-*` / `--border-*` / `--text-*` / `--chip-danger-*`
- `frontend/src/components/appointments/AppointmentCard.module.css` — chips → `--chip-*`; actions → `--color-success` / `--color-primary` / `--color-error` / `--brand-accent-soft` / `--color-warning`
- `EmptyState.module.css` — `--text-primary` / `--text-secondary` / `--color-primary` / `--color-primary-hover`
- `NewAppointmentModal.module.css` — overlay → `--overlay-scrim`
