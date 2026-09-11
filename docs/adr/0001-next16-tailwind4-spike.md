# ADR 0001 — Spike Next.js 16 + Tailwind CSS v4

**Estado:** resultado de spike documentado — **NO-GO** para fusionar en `main` hasta `pnpm lint` verde (o política explícita de reglas).  
**Fecha:** 2026-04-21  
**Rama de trabajo:** `spike/next16-tailwind4`

## Contexto

El frontend vive en App Router (Next 15.x) con CSS propio (`tokens.css`, `utilities.css`) y sin Tailwind. La Fase 3 del change `mvp-ui-visual-parity` pide un spike controlado: subir Next 16, integrar Tailwind v4 según documentación oficial, y registrar evidencia de CI local antes de decidir merge.

## Hipótesis

| ID | Hipótesis | Resultado |
|----|-----------|-----------|
| H1 | Subir `next` y `eslint-config-next` a 16.x con el `next.config.ts` actual (incl. `turbopack.root`) no rompe `typecheck` / `test` / `build`. | **Cumplida** — los tres comandos pasaron en esta rama. |
| H2 | Tailwind v4 con `@tailwindcss/postcss` + `@import "tailwindcss"` en `globals.css` compila junto a las hojas existentes. | **Cumplida** — `next build` completó sin errores de CSS. |
| H3 | `pnpm lint` sigue verde sin refactors de datos/efectos. | **Rechazada** — nuevas reglas estrictas de `eslint-config-next` 16 (p. ej. `react-hooks/set-state-in-effect`) fallan en código MVP existente. |

## Comandos ejecutados (oficial)

Paquetes (resumen):

- `pnpm add next@16.2.4 react@19.1.0 react-dom@19.1.0`
- `pnpm add -D eslint-config-next@16.2.4 tailwindcss @tailwindcss/postcss`

Archivos añadidos o tocados en el spike:

- `frontend/postcss.config.mjs` — plugin `@tailwindcss/postcss` ([Next.js — CSS / PostCSS](https://nextjs.org/docs/app/getting-started/css)).
- `frontend/src/app/globals.css` — primera línea `@import "tailwindcss";` ([Tailwind v4 — import](https://github.com/tailwindlabs/tailwindcss.com/blob/main/src/blog/tailwindcss-v4/index.mdx)).
- `frontend/eslint.config.mjs` — configuración **flat** nativa (`eslint/config` + `eslint-config-next/core-web-vitals` + `typescript`); `FlatCompat` + `extends("next/...")` rompe con error de estructura circular al validar.

Referencias: documentación Next (CSS + ESLint flat), blog / docs Tailwind v4.

## CI local (evidencia)

Máquina: Windows; repo en `d:\Repos\GonsGarage`; directorio `frontend/`.

| Comando | Salida | Código salida |
|---------|--------|----------------|
| `pnpm lint` | Falla: **26 errores** (principalmente `react-hooks/set-state-in-effect` en páginas accounting, appointments, `ThemeSwitcher`, `useAuthHydrationReady`, etc.; además `react-hooks/immutability` en `AuthContext.tsx` por `checkAuthStatus` usado antes de declararse). Quedan advertencias menores (`no-unused-vars`, etc.). | 1 |
| `pnpm typecheck` | OK (`tsc --noEmit`) | 0 |
| `pnpm test` | OK — **26** pruebas en **6** archivos (Vitest) | 0 |
| `pnpm build` | OK — Next **16.2.4** (Turbopack), generación de rutas completada | 0 |

Nota: `next build` informó ajuste automático de `tsconfig.json` (`jsx`: `preserve` → `react-jsx`), alineado con el runtime automático de React en Next.

## GO / NO-GO / DEFER

- **GO (merge a `main`):** no recomendado en el estado actual — **CI del frontend rompería en lint** tal como está el código.
- **NO-GO:** adoptar el bump en `main` sin antes corregir reglas o patrones — **decisión tomada para proteger la barra de calidad del repo.**
- **DEFER:** no aplica como “posponer el spike”: el spike **ya se ejecutó** en rama; lo que queda aplazado es el **merge** hasta una PR dedicada de compatibilidad ESLint / hooks.

## Pasos para contribuidores (cuando se persiga GO)

1. Partir de la rama `spike/next16-tailwind4` o recrear los mismos bumps en una rama nueva desde `main` actualizado.
2. Hacer `pnpm lint` y resolver bloqueos:
   - Refactor de efectos que disparan `setState` de forma síncrona (o patrón aceptado por el equipo con issue enlazada y `eslint-disable` acotado).
   - En `AuthContext`, reordenar o extraer `checkAuthStatus` (p. ej. `useCallback` declarado antes del `useEffect` que lo usa).
3. Volver a ejecutar `pnpm lint`, `pnpm typecheck`, `pnpm test`, `pnpm build`.
4. PR único o dividido (bump vs. fixes de lint) según política del equipo; actualizar `openspec/config.yaml` (contexto “Next.js 16”) cuando `main` quede en 16.x.

## Consecuencias

- El spike demuestra que **compilación y pruebas** pueden mantenerse verdes con Next 16 + Tailwind v4 mínimo.
- El cuello de botella real para adopción es **ESLint / reglas de hooks** más estrictas en la stack Next 16, no el bundler ni TypeScript en este recorte.
