# ui-component-system — Delta (change `mvp-ui-visual-parity`)

**Merge target (nuevo):** `openspec/specs/ui-component-system/spec.md` al archivar el change.

## Purpose

Definir el **rediseño desde cero** con un sistema tipo **Shadcn** (componentes copiables, Radix, tokens de tema) como capa canónica de UI del MVP, sustituyendo patrones ad hoc actuales de forma **planificada por fases** tras Fase 1–3 o en paralelo donde el equipo acuerde.

## Requirements

### Requirement: Greenfield Shadcn foundation

El frontend **SHALL** incorporar la base recomendada por el stack elegido (p. ej. **shadcn/ui** + Tailwind v4 cuando Fase 3 sea GO, o variante documentada en ADR) en un **directorio dedicado** (`components/ui/` o ruta acordada), sin mezclar primitives nuevas con estilos legacy en el mismo archivo.

#### Scenario: Foundation PR is self-contained

- **GIVEN** el primer PR de Fase 4
- **WHEN** se revisa el diff
- **THEN** introduce configuración, dependencias y primitives base **sin** reescribir aún todas las rutas MVP
- **AND** `pnpm build` y `pnpm test` permanecen verdes

### Requirement: MVP route migration to system components

Cada vista MVP autenticada **SHALL** migrarse a componentes del nuevo sistema (botones, inputs, diálogos, tablas) según plan en `tasks.md`; no se considera “desde cero” cumplido hasta que **no** queden formularios críticos solo en CSS modules legacy sin plan de sustitución.

#### Scenario: Auth shell on new system

- **GIVEN** cierre parcial de Fase 4 acordado en tasks
- **WHEN** usuario abre login o registro
- **THEN** la UI usa exclusivamente primitives del sistema para campos y acciones primarias **O** el verify documenta excepción temporal con fecha límite

### Requirement: Theme parity with brand

El tema del sistema Shadcn **SHALL** mapear colores de marca GonsGarage (navy, accent, signal) a variables de tema del stack, de modo que dark/light permanezcan coherentes con `ui-brand-shell` tras merge del delta de shell.

#### Scenario: Token mapping documented

- **GIVEN** merge de Fase 4 a `main`
- **WHEN** un maintainer abre la guía de tema
- **THEN** existe tabla o comentario que enlaza tokens de marca ↔ variables del sistema de componentes
