# ui-brand-shell Specification

## Purpose

Define how the GonsGarage web UI SHALL present a **minimal, professional** look aligned with the **brand palette** (logo-derived), with **consistent light/dark** behaviour on the surfaces in scope.

## Requirements

### Requirement: Documented brand palette

The product SHALL maintain a **single documented source of truth** for brand-related colours (navy/carbon, trust blue, signal red, neutrals) used by the application shell and in-scope routes, so designers and implementers can verify alignment with the logo.

#### Scenario: Brand section exists

- GIVEN the design tokens stylesheet for the frontend
- WHEN a maintainer opens it to audit colours against the logo asset
- THEN a dedicated **Brand** (or equivalent) comment block lists the role of each brand token relative to the logo
- AND primary interaction colours are traceable to those tokens

#### Scenario: No orphan brand hex in shell

- GIVEN the authenticated application shell (header, primary nav, logout)
- WHEN the theme is toggled between light and dark
- THEN destructive and accent states remain legible without introducing **raw hex literals** for colours that already have a semantic token

### Requirement: Theme coherence on priority routes

The **vehicles** and **appointments** user-facing views that ship with this change SHALL NOT rely on hardcoded colour literals for backgrounds, borders, and text that duplicate the token palette, so light and dark themes stay visually coherent.

#### Scenario: Cars view respects theme

- GIVEN a user on the cars area with dark theme enabled
- WHEN they browse lists and forms in that area
- THEN surfaces and status colours derive from shared tokens or shared utilities, not ad-hoc hex unrelated to the token system

#### Scenario: Appointments view respects theme

- GIVEN a user on the appointments area
- WHEN status chips or alerts are shown
- THEN their colours map to semantic tokens (success, warning, error, info) or documented utilities, not standalone hex blocks that ignore dark mode

### Requirement: Lint warning budget on default frontend gate

A release or merge that touches the **Next.js frontend package** **SHALL** satisfy the repository default **`pnpm lint`** outcome **without warnings**, so React Hooks and related rules deferred as `warn` during migration **MUST NOT** accumulate unchecked.

#### Scenario: Clean lint output on review

- **GIVEN** a change ready for review that modifies files under `frontend/`
- **WHEN** a maintainer runs `pnpm lint` in the frontend package as documented in `openspec/config.yaml`
- **THEN** the command **SHALL** complete with **zero warnings** in the aggregate lint report for that package
- **AND** `pnpm build` in the same package **SHALL** still complete successfully unless the change explicitly documents a temporary build exception approved outside this requirement

#### Scenario: Capped, documented waivers

- **GIVEN** a specific warning cannot be removed in the same change without unacceptable product risk
- **WHEN** the change documents the exception in its proposal or verify notes with rationale
- **THEN** at most **three** source files **MAY** retain a targeted suppression
- **AND** each suppression **SHALL** be scoped (rule + line or narrow block), not a file-wide disable of the default hook ruleset

### Requirement: Non-regression quality gate

A release that claims this capability SHALL keep the frontend **lint-clean (no errors and no warnings on the default `pnpm lint` invocation)** and **buildable** after styling refactors and related UI refactors, subject only to the **Lint warning budget** waiver rules above.

(Previously: success required no new **errors** from lint; **warnings** were not contractually capped.)

#### Scenario: CI-quality commands succeed locally

- **GIVEN** the change is ready for review
- **WHEN** `pnpm lint` and `pnpm build` are run in the frontend package
- **THEN** both complete successfully with **no errors and no warnings** introduced or left unresolved by this work, except waivers that meet the **Lint warning budget** requirement
- **AND** any waiver **SHALL** appear in the active change documentation with rationale

---

## Merged additions (change `mvp-ui-visual-parity`, archived 2026-04-21)

The following sections **add** MVP-wide dark, loading, route homogeneity, toolchain spike, and Shadcn-canonical behaviour. They **do not** remove requirements above.

### Phase 1 — Dark usable + loading contract

#### Requirement: Dark theme minimum quality (MVP)

Authenticated MVP surfaces **SHALL** present legible contrast in `html[data-theme='dark']` for primary/secondary text, borders, and panel backgrounds, without relying on raw `#rrggbb` in components when an equivalent token exists.

##### Scenario: Dashboard readable in dark

- **GIVEN** `data-theme='dark'` and an authenticated user
- **WHEN** they open `/dashboard`
- **THEN** metric text and links meet at least **WCAG AA** contrast against backgrounds **OR** are justified in code with an issue link
- **AND** there are no unintended light “ghost” blocks without a surface token

##### Scenario: Accounting shell readable in dark

- **GIVEN** `data-theme='dark'`
- **WHEN** they navigate under `/accounting`
- **THEN** headers, tables, and forms use `--surface-*`, `--text-*`, or documented utilities aligned to tokens

#### Requirement: Unified loading indicator

The app **SHALL** expose a reusable loading pattern (component or documented class) with exactly **three** nominal sizes (`sm`, `md`, `lg`) with fixed proportions; MVP screens **SHALL NOT** introduce new ad hoc sizes except via documented `docs/` entry or `// UI exception:`.

##### Scenario: Size variants documented

- **GIVEN** a developer opens the unified indicator source
- **WHEN** they look for size variants
- **THEN** they find exactly three nominal variants with dimensions in `rem` in a single shared module or stylesheet

##### Scenario: Full-page load uses large variant only

- **GIVEN** progressive substitution on MVP routes
- **WHEN** a view shows full-page loading
- **THEN** it uses the `lg` variant (or named equivalent) and announces busy state with `aria-busy="true"` **OR** associated accessible text

### Phase 2 — Cross-route homogeneity

#### Requirement: MVP route coverage for theme coherence

Beyond **cars** and **appointments**, **employees**, **client**, **dashboard**, **accounting** (and sub-routes) **SHALL** follow the same policy as “Theme coherence on priority routes”: no duplicated colour literals for background/border/text on first-view components that contradict tokens.

##### Scenario: Employees list respects tokens in both themes

- **GIVEN** staff on `/employees`
- **WHEN** toggling light ↔ dark
- **THEN** rows, header, and primary actions do not end up with incompatible inherited backgrounds (e.g. pure white on white) from local literals

##### Scenario: Client home respects tokens

- **GIVEN** client role on `/client`
- **WHEN** dark theme
- **THEN** cards and empty states use tokenized surfaces and text

#### Requirement: Loader migration completeness (MVP)

For each file under `frontend/src/app/` that used `spinnerLg`, `spinnerMd`, local `.spinner` in a CSS module, or undocumented equivalent, the team **SHALL** migrate to the unified pattern **OR** document the exception in the phase PR.

##### Scenario: Inventory gate before closing Phase 2

- **GIVEN** Phase 2 closure
- **WHEN** searching the repo for agreed patterns (`spinnerLg`, `className={styles.spinner}`)
- **THEN** zero occurrences **OR** an explicit approved exception list in `verify-report.md`

### Phase 3 — Platform spike (Tailwind v4 + Next 16)

#### Requirement: Upgrade spike decision record

Before adopting **Next.js 16** and **Tailwind CSS v4** on `main`, the project **SHALL** keep a document (ADR in `docs/` or change section) with: benefit hypothesis, branches tried, `pnpm build` + `pnpm test` outcomes, and **GO** / **NO-GO** / **DEFER** decision. When the ADR records **GO** for this migration, **`package.json` on `main` SHALL list Next.js 16.x and Tailwind CSS v4** as the adopted dependency versions (not only an optional spike without merging those dependencies).

##### Scenario: NO-GO preserves current stack

- **GIVEN** a NO-GO outcome
- **WHEN** the phase is archived
- **THEN** `package.json` on `main` stays on Next 15.x without mandatory Tailwind v4
- **AND** the ADR explains why

##### Scenario: GO documents migration steps

- **GIVEN** a GO outcome
- **WHEN** the spike merges
- **THEN** the ADR lists contributor migration steps (commands, known breaking changes)
- **AND** `package.json` on `main` **SHALL** list Next.js 16.x and Tailwind CSS v4 as adopted dependency versions

#### Requirement: Spike non-regression

Any branch exercising Phase 3 **SHALL** pass the same quality commands as the base `ui-brand-shell` spec (`pnpm lint`, `pnpm build`) before proposing merge to `main`.

##### Scenario: CI parity on spike branch

- **GIVEN** a spike PR
- **WHEN** the repo CI workflow runs
- **THEN** configured frontend jobs **SHALL** finish green or the PR is not mergeable

### Phase 4 — Full Shadcn-style redesign

#### Requirement: Canonical component system (greenfield)

The product **SHALL** adopt a **Shadcn-style** component system (see `ui-component-system` spec) as the single reference for new UI and migrations; redesign **SHALL** be treated as **greenfield** on that base (no stray legacy CSS patches without an exit path).

##### Scenario: Legacy escape hatch documented

- **GIVEN** a screen not yet migrated
- **WHEN** code touching it merges
- **THEN** the PR links a Shadcn migration task **OR** extends an exception in `verify-report.md` with owner and date

##### Scenario: Post–Phase 4 MVP visual consistency

- **GIVEN** Phase 4 closed per `tasks.md`
- **WHEN** a user walks MVP routes in light and dark
- **THEN** shared interactive controls (primary button, input, modal, table) share the same Shadcn system visual language

---

## Merged additions (change `ui-homogeneity-modal-workshop-parts`, archived 2026-04-27)

### Requirement: Staff inventário e taller alinhados ao shell homogéneo

As vistas autenticadas de **inventário de peças** (`admin` / `manager`) e **taller** (staff de oficina) **SHALL** seguir o mesmo padrão de **AppShell + toolbar** que áreas staff já homogéneas: **criação** de recurso primário **SHALL NOT** ser a única acção disponível como página isolada sem continuidade com a lista, quando a lista é o contexto principal de trabalho.

#### Scenario: Criar peça desde o contexto da lista

- **GIVEN** um utilizador `admin` ou `manager` na lista de peças
- **WHEN** inicia **Nova peça**
- **THEN** o fluxo de criação **SHALL** manter a coerência com rotas que usam **sobreposição** (modal ou query na mesma rota) alinhada a accounting/appointments
- **AND** o utilizador **SHALL** conseguir cancelar e regressar à lista sem perda de contexto de navegação principal

#### Scenario: Tema e tokens nas superfícies tocadas

- **GIVEN** tema claro ou escuro activo
- **WHEN** o utilizador percorre formulários ou modais de inventário ou taller alterados por esta mudança
- **THEN** fundos, bordas e texto de interacção primária **SHALL NOT** introduzir literais de cor que dupliquem o papel de tokens já definidos para o mesmo semântica

---

## Merged additions (change `nextjs-16-react19-migration`, archived 2026-04-27)

### Requirement: Production stack on main (Next.js 16 + Tailwind v4)

Tras este change, a linha **oficial** em `main` **SHALL** incluir **Next.js 16.x** (ou superior acordado) e **Tailwind CSS v4** como ferramentas de build de estilos do frontend; **MUST NOT** ficar o frontend preso a Next 15.5 + Tailwind v3 só por omissão pós-merge. O produto **SHALL** manter **ADR GO** (ou documento equivalente no change) con hipótese de benefício, pasos de migración e resultados de `pnpm build`, `pnpm test`, `pnpm lint` e **CI** alineados ao `openspec/config.yaml`.

#### Scenario: CI and local quality gate

- **GIVEN** o merge do change concluído em `main`
- **WHEN** corren os comandos de qualidade frontend acordados no CI (lint, typecheck, test, build)
- **THEN** **SHALL** completar sen erros introducidos por esta migración
- **AND** o ADR **GO** **SHALL** estar ligado ou referenciado no repositório

#### Scenario: Theme and shell without functional regression

- **GIVEN** tema claro e escuro e rotas MVP en escopo do checklist do change
- **WHEN** un usuario percorre shell, navegación primaria e vistas tocadas
- **THEN** contraste e legibilidade **SHALL** permanecer alinhados a tokens documentados
- **AND** **MUST NOT** introducirse regresión funcional documentada como bloqueante no `verify-report` do change

### Requirement: Documentación de cambios importantes na UI

Cada alteración **estructural** na capa de presentación (config Tailwind v4, tokens globais, `next.config`, pipeline CSS) que afecte o comportamento observábel **SHALL** aparecer na tabla “antes/después” do ADR ou `design.md` do change, con unha liña de motivación testábel.

#### Scenario: Maintainer finds rationale

- **GIVEN** un mantenedor revisa un diff non obvio (p. ex. renomeo de utilidades Tailwind)
- **WHEN** abre o ADR ou design do change
- **THEN** **SHALL** atopar entrada que ligue o cambio a requisito ou risco mitigado

---

## Merged additions (change `frontend-eslint-warnings-cleanup`, archived 2026-04-27)

Integráronse no bloque **`## Requirements`** (antes desta sección de arquivo): o requisito **Lint warning budget on default frontend gate** e a versión actualizada de **Non-regression quality gate** (delta en `openspec/changes/archive/2026-04-27-frontend-eslint-warnings-cleanup/specs/ui-brand-shell/spec.md`).

---

## Merged additions (change `ui-gentleman-cute-dark-comfort`, archived 2026-09-15)

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
