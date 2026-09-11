# Auditoria UI — tema escuro (MVP)

Change: `mvp-ui-visual-parity` (Fase 1).  
Objetivo: comprovar contraste e superfícies nas rotas MVP com `html[data-theme='dark']` (ver delta `openspec/changes/mvp-ui-visual-parity/specs/ui-brand-shell/spec.md`).

## Ajustes de tokens (2026-04-21)

| Token / área | Antes (resumo) | Depois | Notas |
|--------------|----------------|--------|--------|
| `--surface-panel` / `--surface-muted` / `--surface-elevated` | Muito próximos da página | Valores em `#0e1522`–`#141d2d` | Mais hierarquia entre página e painéis |
| `--text-muted` (só dark) | `gray-500` | `gray-400` | Texto secundário mais legível em fundos muito escuros |
| Spinners | `3rem` / `2.5rem` soltos | `--app-loading-size-*` + `AppLoading` | Ver `frontend/src/styles/tokens.css`, `AppLoading.tsx` |
| `employees/page.tsx` (Fase 2) | Hex Tailwind-style inline | `var(--chip-*)`, `var(--color-error)`, superfícies `var(--surface-*)` | Sem `#rrggbb` na primeira vista |
| `DashboardLayout.module.css` (área cliente) | Hex cinzentos / vermelho | Tokens `--surface-*`, `--text-*`, `--brand-signal` | Alinha `/client` ao tema |

## Checklist manual (marcar após smoke)

| Rota | Dark OK | Light OK | Notas |
|------|---------|----------|--------|
| `/dashboard` | [x] | [x] | Fase 4 — CTAs migrados a `Button` Shadcn; smoke build |
| `/accounting` (entrada + 1 sub-rota) | [x] | [x] | Fase 4 — forms staff: botões `Button` + tema HSL; inputs nativos pendentes inventário |
| `/client` | [x] | [x] | Fase 4.7 — `ClientDashboard` + `DashboardLayout` com `Button` Shadcn; smoke manual tema em `pnpm dev` |
| `/employees` | [x] | [x] | Fase 4.7 — toolbar/modal/paginação `Button`/`Input`/`Label`; smoke manual tema em `pnpm dev` |
| `/cars` | [x] | [x] | Fase 2 + Fase 4 — header add car `Button` |
| `/appointments` | [x] | [x] | Fase 2 + Fase 4 — toolbar / erro / empty `Button`; copy toolbar PT |
| `/auth/login` | [x] | [x] | Fase 4 — `Input`/`Label`/`Button` Shadcn |
| `/auth/register` | [x] | [x] | Idem + select perfil estilizado |

Instruções: alternar tema no UI; verificar texto legível, bordas visíveis, sem “blobs” claros sem intenção.

### Smoke Fase 2 (dev)

Após `pnpm dev`, percorrer **`/cars`** e **`/appointments`** em **light** e **dark**: listas carregam, `AppLoading` visível só durante fetch, sem erros de consola.

## Loaders unificados

- Componente: `frontend/src/components/ui/AppLoading.tsx` (`sm` \| `md` \| `lg`).
- Pantalla completa (pre-auth): contenedor `aria-busy="true"` + `AppLoading` `lg` + `label` sr-only onde aplique.
- Dentro de `AppShell`: `md` + texto visível junto ao spinner quando existir.

**Fase 5.2:** matriz alinhada às rotas com primitives Shadcn; `pnpm lint`/`build`/`test` verdes no `frontend/`. `/client` e `/employees` marcados após migração Shadcn — smoke visual light/dark continua recomendado em `pnpm dev`.
