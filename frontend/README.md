# GonsGarage frontend

Next.js **15.5** (App Router), React **19**, TypeScript, Zustand. Package manager: **pnpm 9** (see `packageManager` in [`package.json`](./package.json)).

- **Monorepo overview & setup:** [../README.md](../README.md)
- **Detailed dev guide:** [../docs/development-guide.md](../docs/development-guide.md)
- **API client notes:** [docs/api-client.md](./docs/api-client.md)

## Commands

```bash
pnpm install
pnpm dev           # http://localhost:3000
pnpm lint
pnpm typecheck
pnpm test          # Vitest (default; same as CI)
pnpm test:coverage # Vitest + V8 coverage (text + coverage/coverage-summary.json)
pnpm e2e           # Playwright end-to-end tests (frontend + backend running)
pnpm e2e:headed    # Playwright in headed mode
pnpm e2e:ui        # Playwright interactive UI
pnpm build
```

Copy [`./.env.local.example`](./.env.local.example) to `.env.local` and set `NEXT_PUBLIC_API_URL` if your API is not at `http://localhost:8080`.

For Playwright, keep the frontend running on `http://localhost:3000` and the API on `http://localhost:8080` (or set `PLAYWRIGHT_BASE_URL` to point elsewhere before running `pnpm e2e`).

## Create Next App

This app was originally bootstrapped with [`create-next-app`](https://nextjs.org/docs/app/api-reference/cli/create-next-app). For framework docs see [Next.js Documentation](https://nextjs.org/docs).

## UI canónica (Shadcn)

- **Primitives:** `src/components/ui/` — ficheiros estilo shadcn (`button.tsx`, `input.tsx`, `label.tsx`, `dialog.tsx`, …) + `AppLoading.tsx` e outros legados por pastas até migração.
- **Utilidades:** `pnpm dlx shadcn@latest add <componente>` na pasta `frontend/` (lê `components.json`). Depois ajustar imports a `@/components/ui/...`.
- **Tema:** ver [../docs/ui-shadcn-theme.md](../docs/ui-shadcn-theme.md) e `src/styles/shadcn-theme.css` (HSL) + `tailwind.config.ts`.
- **Stack:** [ADR 0002](../docs/adr/0002-shadcn-stack.md).
