# Proposal: MVP — homogeneidad visual, tema oscuro y alineación toolchain (fases)

## Intent

Paridad Arnela fue documental. El MVP se ve heterogéneo (dark, loaders, rutas fuera de `ui-brand-shell`). Incluye **Shadcn desde cero** (Fase 4), UX en Fase 1–2 y spike toolchain en Fase 3.

## Scope

### In Scope

- **Fase 1:** auditoría dark/light MVP; `tokens.css` / utilidades; **loading unificado** (`sm|md|lg`, a11y).
- **Fase 2:** `ui-brand-shell` en employees, client, dashboard, accounting; menos hex/spinners sueltos.
- **Fase 3 (opcional):** spike Next 16 + Tailwind v4 (ADR, branch); no bloquea Fase 1–2.
- **Fase 4:** Shadcn **desde cero** — fundación (`components/ui/` o ADR), primitives, migración MVP completa; mapa marca↔tema; sin forms críticos solo legacy sin plan.

### Out of Scope

- i18n masivo (solo copy mínima si bloquea contraste).
- Cambios de API o dominio.

## Capabilities

### New Capabilities

- **`ui-component-system`**: Shadcn greenfield + migración MVP + tema marca (`specs/ui-component-system/spec.md`).

### Modified Capabilities

- **`ui-brand-shell`**: coherencia tema, loading unificado, rutas MVP; integración con tokens del sistema Shadcn en Fase 4.

## Approach

1. Fase 1–2: inventario, tokens, loading, homogeneidad rutas.  
2. Fase 3: spike Next/TW si prereq Shadcn.  
3. Fase 4: PR fundación → migrar por dominio; “hecho” en `tasks.md`.  
4. Rollback por rama/revert.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `frontend/src/styles/*` | Modified | Tokens / tema hacia Shadcn |
| `frontend/src/components/ui/**` | New | Primitives Shadcn |
| `frontend/src/app/**` | Modified | Reescritura UI sobre sistema |
| `components/*` legacy | Modified/Removed | Sustitución progresiva |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Alcance Fase 4 muy largo | Alto | Hitos por ruta; excepciones fechadas en verify |
| Tailwind v4 + Shadcn acoplan riesgo | Alto | Fase 3 GO antes de fundación o ADR de excepción |
| Regresión a11y | Med | checklist Radix/shadcn por PR |

## Rollback Plan

Revertir por PR/milestone; mantener `main` sin fundación Shadcn hasta PR explícito; si Fase 4 parcial, documentar estado en `tasks.md`.

## Dependencies

- Arnela (`D:\Repos\Arnela`) solo inspiración. Shadcn oficial / docs del stack elegido.

## Success Criteria

- [x] Fase 1–2: como antes (dark, loading, homogeneidad).
- [x] Fase 3: ADR GO/NO-GO Next 16 + Tailwind v4.
- [x] Fase 4: fundación Shadcn mergeada; rutas MVP migradas salvo excepciones listadas en `docs/ui-forms-shadcn-inventory.md`; guía marca↔tema en `docs/ui-shadcn-theme.md`.
- [x] `pnpm lint` + `pnpm build` (+ tests) verdes por hito.
