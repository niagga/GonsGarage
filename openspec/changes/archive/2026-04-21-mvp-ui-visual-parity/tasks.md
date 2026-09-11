# Tasks: MVP UI visual parity

## Phase 1 — Dark + loading

- [x] 1.1 `docs/ui-audit-mvp-dark.md`: checklist dark para dashboard, accounting, client, employees.
- [x] 1.2 Ajustar dark en `frontend/src/styles/tokens.css`; enlazar pares revisados al doc 1.1.
- [x] 1.3 Tres tamaños spinner `sm|md|lg` en `rem`, una sola fuente en `frontend/src/styles/utilities.css` (o hoja importada).
- [x] 1.4 `frontend/src/components/ui/AppLoading.tsx`: prop `size`, `aria-busy`, label opcional sr-only.
- [x] 1.5 Migrar loaders en `frontend/src/app/dashboard/page.tsx`, `cars/page.tsx`, `cars/[id]/page.tsx` a `AppLoading` (`lg` pantalla completa).

## Phase 2 — Homogeneidad rutas

- [x] 2.1 `employees/page.tsx`: sin hex sueltos en primera vista; tokens/utilidades.
- [x] 2.2 `client/page.tsx`: superficies/texto tokenizados light/dark.
- [x] 2.3 `app/accounting/**`: cabeceras/tablas/forms alineados a tokens.
- [x] 2.4 `dashboard` + CSS: quitar `styles.spinner` donde exista; unificar a `AppLoading`.
- [x] 2.5 `rg spinnerLg|spinnerMd|styles\.spinner frontend/src/app` → migrar o fila en `docs/ui-loader-exceptions.md` (ruta, motivo, fecha).
- [x] 2.6 Smoke `/cars`, `/appointments` light/dark.

## Phase 3 — Spike Next + TW (opcional)

- [x] 3.1 `docs/adr/0001-next16-tailwind4-spike.md`: hipótesis, comandos, GO/NO-GO, tabla CI local.
- [x] 3.2 Rama spike: bump Next/TW en `frontend/` según docs oficiales.
- [x] 3.3 En rama: `pnpm lint`, `typecheck`, `test`, `build`; pegar salida en ADR.
- [x] 3.4 Decisión ADR; NO-GO sin merge; GO con pasos para contribuidores.

## Phase 4 — Shadcn desde cero

- [x] 4.1 `docs/adr/0002-shadcn-stack.md`: stack, carpeta `frontend/src/components/ui/`, relación Fase 3.
- [x] 4.2 PR fundación: shadcn init + `Button` `Input` `Label` `Dialog` en `components/ui/` sin rutas MVP aún; build+test verdes.
- [x] 4.3 `docs/ui-shadcn-theme.md`: tabla marca ↔ variables tema shadcn.
- [x] 4.4 PR auth: `app/auth/login`, `register`, `AuthShell.tsx` solo con `components/ui` en CTA/campos.
- [x] 4.5 PR shell: `AppShell.tsx` nav/header con primitives.
- [x] 4.6 PR: `app/dashboard`, `cars`, `appointments` — controles repetidos a `components/ui`.
- [x] 4.7 PR: `employees`, `client`, `app/accounting/**` — mismo; sin form crítico legacy sin issue.
- [x] 4.8 Inventario forms sin `components/ui`: cero o lista fechada en verify (propietario).
- [x] 4.9 `frontend/README.md`: cómo añadir componentes shadcn; UI canónica = `components/ui`.

## Phase 5 — Cierre verify

- [x] 5.1 `pnpm lint`, `build`, `test` en `frontend/` tras hito Fase 4.
- [x] 5.2 Completar matriz light/dark en `docs/ui-audit-mvp-dark.md` para rutas migradas.
