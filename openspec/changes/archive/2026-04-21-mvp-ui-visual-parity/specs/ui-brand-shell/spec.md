# ui-brand-shell — Delta (change `mvp-ui-visual-parity`)

**Merge target:** `openspec/specs/ui-brand-shell/spec.md`  
**Normativa:** requisitos aquí **ADD** comportamiento al spec principal; al archivar, integrar sin borrar requisitos existentes.

---

## Phase 1 — Dark usable + loading contract

### Requirement: Dark theme minimum quality (MVP)

Surfaces autenticadas del MVP **SHALL** presentar en `html[data-theme='dark']` contraste legible para texto primario, secundario, bordes y fondos de panel, sin depender de literales `#rrggbb` en componentes cuando exista token equivalente.

#### Scenario: Dashboard readable in dark

- **GIVEN** `data-theme='dark'` y un usuario autenticado
- **WHEN** abre `/dashboard`
- **THEN** el texto de métricas y enlaces cumple al menos contraste **WCAG AA** frente al fondo **O** está justificado en comentario de código con enlace a issue
- **AND** no hay bloques de fondo claro “fantasma” sin token de superficie

#### Scenario: Accounting shell readable in dark

- **GIVEN** `data-theme='dark'`
- **WHEN** navega a cualquier ruta bajo `/accounting`
- **THEN** cabeceras, tablas y formularios usan `--surface-*`, `--text-*` o utilidades documentadas alineadas a tokens

### Requirement: Unified loading indicator

La aplicación **SHALL** exponer un patrón de loading reutilizable (componente o clase documentada) con **tres** tamaños nominales (`sm`, `md`, `lg`) con proporciones fijas; las pantallas MVP **SHALL NOT** introducir nuevos tamaños ad hoc salvo excepción registrada en `docs/` o comentario `// UI exception:`.

#### Scenario: Size variants documented

- **GIVEN** un desarrollador abre el código del indicador unificado
- **WHEN** busca variantes de tamaño
- **THEN** encuentra exactamente las tres variantes nominales con dimensiones en `rem` declaradas en un solo módulo o hoja compartida

#### Scenario: Full-page load uses large variant only

- **GIVEN** sustitución progresiva en rutas MVP
- **WHEN** una vista muestra loading de página completa
- **THEN** usa la variante `lg` (o la nombrada equivalente en el contrato) y anuncia estado ocupado con `aria-busy="true"` **O** texto accesible asociado

---

## Phase 2 — Cross-route homogeneity

### Requirement: MVP route coverage for theme coherence

Además de **cars** y **appointments**, las vistas **employees**, **client**, **dashboard**, **accounting** (y subrutas) **SHALL** cumplir la misma política que “Theme coherence on priority routes”: sin literales de color duplicados que contradigan tokens para fondo/borde/texto en componentes de primera vista (lista, detalle, formulario principal).

#### Scenario: Employees list respects tokens in both themes

- **GIVEN** usuario staff en `/employees`
- **WHEN** alterna light ↔ dark
- **THEN** filas, cabecera y botones primarios no quedan con fondo heredado incompatible (p. ej. blanco puro sobre blanco) por literales locales

#### Scenario: Client home respects tokens

- **GIVEN** rol cliente en `/client`
- **WHEN** tema dark
- **THEN** tarjetas y estados vacíos usan superficies y texto tokenizados

### Requirement: Loader migration completeness (MVP)

Para cada archivo bajo `frontend/src/app/` que hoy use `spinnerLg`, `spinnerMd`, `.spinner` local en CSS module, o equivalente no documentado, el equipo **SHALL** migrar a el patrón unificado **O** documentar la excepción en el PR de la fase.

#### Scenario: Inventory gate before closing Phase 2

- **GIVEN** cierre de Fase 2
- **WHEN** se ejecuta búsqueda en repo por patrones acordados (`spinnerLg`, `className={styles.spinner}`)
- **THEN** cero ocurrencias **O** una lista explícita de excepciones aprobadas en `verify-report.md`

---

## Phase 3 — Platform spike (Tailwind v4 + Next 16)

### Requirement: Upgrade spike decision record

Antes de adoptar **Next.js 16** y **Tailwind CSS v4** en `main`, el proyecto **SHALL** mantener un documento (ADR en `docs/` o sección en este change) con: hipótesis de beneficio, ramas probadas, resultado de `pnpm build` + `pnpm test`, y decisión **GO** / **NO-GO** / **DEFER**.

#### Scenario: NO-GO preserves current stack

- **GIVEN** resultado NO-GO
- **WHEN** se archiva la fase
- **THEN** `package.json` de `main` permanece en Next 15.x sin Tailwind v4 obligatorio
- **AND** el ADR explica el motivo

#### Scenario: GO documents migration steps

- **GIVEN** resultado GO
- **WHEN** se mergea el spike
- **THEN** el ADR lista pasos de migración para contribuidores (comandos, breaking changes conocidos)

### Requirement: Spike non-regression

Cualquier rama que pruebe Fase 3 **SHALL** pasar los mismos comandos de calidad que el spec base de `ui-brand-shell` (`pnpm lint`, `pnpm build`) antes de propuesta de merge a `main`.

#### Scenario: CI parity on spike branch

- **GIVEN** PR del spike
- **WHEN** ejecuta el workflow de CI del repo
- **THEN** jobs de frontend configurados **SHALL** concluir en verde o el PR no es mergeable

---

## Phase 4 — Rediseño completo tipo Shadcn desde cero

### Requirement: Canonical component system (greenfield)

El producto **SHALL** adoptar un **sistema de componentes estilo Shadcn** (ver capability `ui-component-system`) como referencia única para UI nueva y migraciones; el rediseño **SHALL** tratarse como **greenfield** sobre esa base (no parches sueltos en CSS legacy sin ruta de salida).

#### Scenario: Legacy escape hatch documented

- **GIVEN** una pantalla aún no migrada
- **WHEN** se mergea código que la toca
- **THEN** el PR enlaza tarea de migración Shadcn **O** amplía excepción en `verify-report.md` con propietario y fecha

#### Scenario: Post–Phase 4 MVP visual consistency

- **GIVEN** Fase 4 cerrada según `tasks.md`
- **WHEN** un usuario recorre rutas MVP en light y dark
- **THEN** controles interactivos compartidos (botón primario, input, modal, tabla) comparten el mismo lenguaje visual del sistema Shadcn
