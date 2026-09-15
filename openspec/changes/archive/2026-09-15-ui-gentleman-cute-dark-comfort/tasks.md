# Tasks: ui-gentleman-cute-dark-comfort

## Phase 1 — Tokens
- [x] 1.1 Warm dark palette in `tokens.css` (gentleman-cute canonical `data-theme=dark`)
- [x] 1.2 Align `shadcn-theme.css` HSL with warm surfaces

## Phase 2 — Surfaces (CSS modules)
- [x] 2.1 Fix `client.module.css` — no literal `white` panels; semantic `--surface-*` / `--text-*`
- [x] 2.2 Sweep other MVP modules (`dashboard`, `cars`, modals, empty state, appointment cards)

## Phase 3 — Verify
- [x] 3.1 Contract test: client CSS avoids hardcoded white panel backgrounds
- [x] 3.2 `pnpm typecheck` + `pnpm test` in frontend
