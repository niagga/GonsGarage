## ADDED Requirements

### Requirement: Warm dark comfort (gentleman-cute)

For `html[data-theme='dark']`, the canonical dark palette in `frontend/src/styles/tokens.css` SHALL use warm charcoal/stone surfaces (documented as gentleman-cute). Shadcn HSL variables in `shadcn-theme.css` SHALL stay aligned with those surfaces. MVP route CSS modules SHALL NOT use literal `white` / `#ffffff` for panel or card backgrounds when `--surface-panel` or `--surface-header` apply.

#### Scenario: Client dashboard panels in dark

- **GIVEN** `data-theme='dark'` and a client on `/client`
- **WHEN** they view stat cards and list panels
- **THEN** backgrounds derive from `--surface-panel` or `--surface-header`
- **AND** primary and secondary text use `--text-primary`, `--text-secondary`, or `--text-muted`
- **AND** panel backgrounds are not hardcoded to `#ffffff`

#### Scenario: Contract guard for client shell CSS

- **GIVEN** the frontend test suite
- **WHEN** `gentleman-cute-theme.contract.test.ts` runs
- **THEN** `client.module.css` contains no `background: white` / `background-color: white` panel literals
- **AND** it references `--surface-panel` and `--surface-header`
