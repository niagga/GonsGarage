# GonsGarage — Design system

Synced from the project's own sources on 2026-11-27. Read-only over `frontend/`; no project file was modified.

**Sources read**

| Source | What it provides |
| --- | --- |
| `frontend/src/styles/tokens.css` | Brand block, gray ramp, surfaces, text, spacing, radius, shadows, motion, focus rings, chips, loading sizes, and the full `html[data-theme='dark']` override |
| `frontend/src/styles/shadcn-theme.css` | shadcn/ui HSL triples (no `hsl()` wrapper) for `:root` and `html[data-theme='dark']` |
| `frontend/tailwind.config.ts` | `darkMode: ["selector", '[data-theme="dark"]']` and the Tailwind color aliases that consume the HSL triples |

**Fonts in the real app:** `--font-sans` resolves `var(--font-geist-sans, 'Geist')` (Geist injected by `next/font`). Artboards render the documented fallback stack with system fonts only, because external font files are not allowed on the canvas.

---

## How theming works

- The theme is an **attribute selector**, not a class: `html[data-theme='light']` and `html[data-theme='dark']` (`tokens.css:155,159`; `shadcn-theme.css:7,29`).
- `tailwind.config.ts:5` declares `darkMode: ["selector", '[data-theme="dark"]']`, but **no `dark:` utility is used anywhere in `src/**/*.tsx`** — the theme works purely through token overrides.
- The preference is stored in `localStorage['gonsgarage-theme']` and read before paint by `lib/theme-inline-script.ts:6`, injected `beforeInteractive` at `app/layout.tsx:35-39`.
- The switcher is `components/theme/ThemeSwitcher.tsx` (select: sistema / claro / escuro), mounted globally at `app/layout.tsx:41`. `prefers-color-scheme` is honoured only while the preference is `system`.

## Brand

`tokens.css:15-21`

| Token | Value | Usage |
| --- | --- | --- |
| `--brand-navy` | `#0a1628` | Dark foundation, marketing footer background |
| `--brand-navy-muted` | `#132337` | Logo bar, secondary dark surface |
| `--brand-steel` | `#334155` | Neutral structural tone |
| `--brand-accent` | `#1d4ed8` | Trust blue; the light-theme interaction color |
| `--brand-accent-soft` | `#3b82f6` | Gradient partner for the accent |
| `--brand-signal` | `#dc2626` | Signage red and destructive actions |
| `--brand-signal-hover` | `color-mix(in srgb, var(--brand-signal) 86%, #000)` | Destructive hover |

## Semantic colors

| Token | Light | Dark (`html[data-theme='dark']`) |
| --- | --- | --- |
| `--color-primary` | `#1d4ed8` | `#38BDF8` |
| `--color-primary-hover` | `#1e40af` | `#5FCCFA` |
| `--color-secondary` | `#64748b` | `#A9B7C6` |
| `--color-success` | `#059669` | `#34D399` |
| `--color-warning` | `#d97706` | `#F59E0B` |
| `--color-error` | `#dc2626` | `#F87171` |

Source: `tokens.css:23-35` (light), `tokens.css:163-172` (dark). Dark palette = **Taller nocturno**, replaced on 2026-11-27.

## Neutral ramp

| Token | Light | Dark |
| --- | --- | --- |
| `--color-white` | `#ffffff` | `#0B0F14` |
| `--color-black` | `#020617` | `#F2F6FA` |
| `--color-gray-50` | `#f4f6f9` | `#0B0F14` |
| `--color-gray-100` | `#eef2f7` | `#0E141A` |
| `--color-gray-200` | `#e2e8f0` | `#141C26` |
| `--color-gray-300` | `#cbd5e1` | `#1A232C` |
| `--color-gray-400` | `#94a3b8` | `#7E8C9B` |
| `--color-gray-500` | `#64748b` | `#A9B7C6` |
| `--color-gray-600` | `#475569` | `#C6D0DB` |
| `--color-gray-700` | `#334155` | `#DCE4EC` |
| `--color-gray-800` | `#1e293b` | `#EBF1F7` |
| `--color-gray-900` | `#0f172a` | `#E9F0F7` |

Source: `tokens.css:37-50`, `tokens.css:175-186`. The dark ramp is a **cool blue-grey steel**, and the ramp inverts (white becomes the darkest value).

## Surfaces and borders

| Token | Light | Dark |
| --- | --- | --- |
| `--surface-page` | `var(--color-gray-50)` → `#f4f6f9` | `#0B0F14` |
| `--surface-panel` | `#ffffff` | `#131A22` |
| `--surface-elevated` | `#ffffff` | `#1A232C` |
| `--surface-muted` | `#eef2f7` | `#0E141A` |
| `--surface-header` | `#ffffff` | `#101821` |
| `--border-default` | `#e2e8f0` | `#253040` |
| `--border-strong` | `#cbd5e1` | `#36455A` |

Source: `tokens.css:52-58`, `tokens.css:188-195`.

## Text

| Token | Light | Dark |
| --- | --- | --- |
| `--text-primary` | `var(--color-gray-900)` → `#0f172a` | `#E9F0F7` |
| `--text-secondary` | `var(--color-gray-600)` → `#475569` | `#A9B7C6` |
| `--text-muted` | `var(--color-gray-500)` → `#64748b` | `#7E8C9B` |
| `--text-on-primary` | `#ffffff` | `#041016` |
| `--text-on-signal` | `#ffffff` | `#ffffff` |

Source: `tokens.css:60-64`, `tokens.css:197-202`.

## Typography

| Token | Value |
| --- | --- |
| `--font-sans` | `var(--font-geist-sans, 'Geist'), -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif` |
| `--font-mono` | `var(--font-geist-mono, 'Geist Mono'), ui-monospace, monospace` |

Source: `tokens.css:66-67`. No type scale is declared as tokens: sizes come from the Tailwind default scale and from per-module CSS. Any new theme must therefore state its own scale explicitly.

## Spacing

`--space-1` `0.25rem` · `--space-2` `0.5rem` · `--space-3` `0.75rem` · `--space-4` `1rem` · `--space-5` `1.25rem` · `--space-6` `1.5rem` · `--space-8` `2rem` · `--space-10` `2.5rem` · `--space-12` `3rem` · `--space-16` `4rem` · `--space-20` `5rem`

Source: `tokens.css:69-79`.

## Containers

| Token | Value |
| --- | --- |
| `--container-sm` | `640px` |
| `--container-md` | `768px` |
| `--container-lg` | `1024px` |
| `--container-xl` | `1280px` |
| `--container-2xl` | `1536px` |

Source: `tokens.css:81-85`. Note: the app shell does **not** use these — `AppShell.module.css:13,132,179` hardcodes `max-width: 1200px`.

## Radius

| Token | Value |
| --- | --- |
| `--radius-sm` | `0.125rem` |
| `--radius` | `0.375rem` |
| `--radius-md` | `0.5rem` |
| `--radius-lg` | `0.75rem` |
| `--radius-xl` | `1rem` |
| `--radius-full` | `9999px` |

Source: `tokens.css:87-92`.

> **Conflict to resolve:** `--radius` is declared twice — `0.375rem` in `tokens.css:88` and `0.5rem` in `shadcn-theme.css:26`. The winner depends on import order in `globals.css`. Any theme proposal must state which value it assumes.

## Shadows

| Token | Light | Dark |
| --- | --- | --- |
| `--shadow-sm` | `0 1px 2px 0 rgb(15 23 42 / 0.04)` | `0 1px 2px 0 rgb(0 0 0 / 0.2)` |
| `--shadow` | `0 1px 3px 0 rgb(15 23 42 / 0.06), 0 1px 2px -1px rgb(15 23 42 / 0.06)` | `0 4px 6px -1px rgb(0 0 0 / 0.3)` |
| `--shadow-md` | `0 4px 12px -2px rgb(15 23 42 / 0.08), 0 2px 6px -2px rgb(15 23 42 / 0.06)` | `0 10px 15px -3px rgb(0 0 0 / 0.4)` |
| `--shadow-lg` | `0 12px 24px -4px rgb(15 23 42 / 0.1)` | `0 16px 40px -8px rgb(0 0 0 / 0.5)` |

Source: `tokens.css:94-97`, `tokens.css:205-208`.

## Motion and focus

| Token | Value |
| --- | --- |
| `--transition-fast` | `150ms ease-in-out` |
| `--transition-base` | `250ms ease-in-out` |
| `--transition-slow` | `350ms ease-in-out` |
| `--focus-ring` | light `0 0 0 3px rgb(37 99 235 / 0.25)` · dark `0 0 0 3px rgb(56 189 248 / 0.4)` |
| `--focus-ring-error` | light `0 0 0 3px rgb(220 38 38 / 0.2)` · dark `0 0 0 3px rgb(248 113 113 / 0.35)` |

Source: `tokens.css:99-106`, `tokens.css:210-211`.

## Status chips

| Chip | Background | Foreground | Border |
| --- | --- | --- | --- |
| `--chip-info-*` | `color-mix(in srgb, var(--color-primary) 10%, var(--surface-panel))` | `var(--brand-accent)` (light) / `var(--color-primary)` (dark) | `color-mix(in srgb, var(--color-primary) 28%, var(--border-default))` |
| `--chip-success-*` | `color-mix(in srgb, var(--color-success) 12%, var(--surface-panel))` | `color-mix(… 55%, var(--color-gray-900))` | `color-mix(in srgb, var(--color-success) 32%, var(--border-default))` |
| `--chip-warning-*` | `color-mix(in srgb, var(--color-warning) 14%, var(--surface-panel))` | `color-mix(… 70%, var(--color-gray-900))` | `color-mix(in srgb, var(--color-warning) 38%, var(--border-default))` |
| `--chip-danger-*` | `color-mix(in srgb, var(--color-error) 10%, var(--surface-panel))` | `var(--color-error)` | `color-mix(in srgb, var(--color-error) 32%, var(--border-default))` |
| `--chip-neutral-*` | `var(--surface-muted)` | `var(--text-secondary)` | `var(--border-default)` |

Source: `tokens.css:129-146`.

## Misc

| Token | Value | Usage |
| --- | --- | --- |
| `--overlay-scrim` | `rgb(15 23 42 / 0.45)` | Modal scrim |
| `--overlay-scrim-strong` | `rgb(0 0 0 / 0.55)` | Stronger scrim |
| `--shadow-primary-glow` | `0 4px 6px color-mix(in srgb, var(--color-primary) 22%, transparent)` | Primary button lift |
| `--ring-focus-input` | `0 0 0 3px color-mix(in srgb, var(--color-primary) 14%, transparent)` | Input focus |
| `--app-loading-size-sm` | `1.25rem` | Small spinner |
| `--app-loading-size-md` | `1.75rem` | Medium spinner |
| `--app-loading-size-lg` | `2.25rem` | Large spinner |

Source: `tokens.css:118-119,147-151,153-155`.

## shadcn HSL triples

The shadcn layer carries the same palette as bare HSL components so Tailwind utilities (`hsl(var(--primary))`) can consume it.

| Token | Light (`:root`) | Dark (`html[data-theme='dark']`) |
| --- | --- | --- |
| `--background` | `210 20% 97%` | `213 29% 6%` |
| `--foreground` | `222 47% 11%` | `210 47% 94%` |
| `--card` | `0 0% 100%` | `213 28% 10%` |
| `--card-foreground` | `222 47% 11%` | `210 47% 94%` |
| `--popover` | `0 0% 100%` | `210 26% 14%` |
| `--popover-foreground` | `222 47% 11%` | `210 47% 94%` |
| `--primary` | `221 83% 53%` | `198 93% 60%` |
| `--primary-foreground` | `0 0% 100%` | `198 60% 4%` |
| `--secondary` | `215 16% 47%` | `211 20% 72%` |
| `--secondary-foreground` | `222 47% 11%` | `210 47% 94%` |
| `--muted` | `210 20% 94%` | `210 30% 8%` |
| `--muted-foreground` | `215 16% 40%` | `211 13% 55%` |
| `--accent` | `210 20% 94%` | `210 26% 14%` |
| `--accent-foreground` | `222 47% 11%` | `210 47% 94%` |
| `--destructive` | `0 72% 51%` | `0 91% 71%` |
| `--destructive-foreground` | `0 0% 100%` | `0 0% 100%` |
| `--border` | `214 32% 91%` | `216 27% 20%` |
| `--input` | `214 32% 91%` | `210 31% 13%` |
| `--ring` | `221 83% 53%` | `198 93% 60%` |
| `--radius` | `0.5rem` | `0.5rem` |

Source: `shadcn-theme.css:7-27` (light), `shadcn-theme.css:29-46` (dark).

## Component rules

These are the project's real components. Any artboard that shows app chrome must follow them.

### App shell (`components/layout/AppShell.tsx`)

- **No sidebar exists.** The shell is a top header, a horizontally scrolling tab strip, and a centered `main`.
- Container: `max-width: 1200px`, `overflow-x: clip` (`AppShell.module.css:4,13,132,179`).
- Header bar: `min-height: 64px` (`:24`).
- Nav strip: `display: flex`, `flex-wrap: nowrap`, `overflow-x: auto`, tabs `min-height: 2.75rem` (44px) (`:132-158`).
- `main` padding: `var(--space-6) var(--space-4)` → `1.5rem 1rem` (`:186`).
- Responsive: `≤768px` reduces padding (`:206`); `≤640px` the header stacks with the logo on its own row and the user row full width (`:225`); `≤480px` the page toolbar becomes a column (`:246`).
- Coverage: 23 pages across 7 route groups.

### Client dashboard shell (`components/layouts/DashboardLayout.tsx`)

A near-copy of AppShell with a `navigationItems` prop, used by exactly one route (`app/client/page.tsx:84`). Its CSS duplicates AppShell's container and 64px header (`DashboardLayout.module.css:5,17,24`).

### Auth shell (`components/auth/AuthShell.tsx`)

Centered column, `min-height: 100vh`, `max-width: 28rem` (448px) (`AuthShell.module.css:1-17`).

### Buttons

- `ui/button.tsx:20-23` — heights `h-9` (36px) default, `h-8` (32px) `sm`, `h-11` (44px) `lg`, `h-9 w-9` icon; Tailwind focus ring at `:8`.
- `utilities.css:25,42,74,101` — legacy `.btn*` classes use `min-height: 2.5rem` (40px) with `focus-visible` at `:55,87,113`.
- **Both scales are below the 44px touch goal** except `lg`. A theme proposal that cares about phones must raise them.

### Inputs

`ui/input.tsx:11` — `h-9` (36px), `md:` prefix used once, Tailwind focus ring. Labels come from `ui/label.tsx`; form wiring from `ui/form.tsx`.

### Tables

There is **no table component**. Each page renders `<table className={styles.table}>` inside a per-page `.tableWrap` that owns the horizontal scroll:

- `accounting/accounting.module.css:93` (used by suppliers, received-invoices, issued-invoices, billing-documents, my-invoices)
- `admin/users/admin-users.module.css:72`
- `admin/parts/admin-parts.module.css:34`

`workshop/page.tsx:181,226` renders tables with **no wrapper** (`workshop.module.css:70` sets only `width: 100%`), so four columns compress at phone width instead of scrolling. No wrapper declares a `min-width` on the table, so columns always squeeze.

### Cards

`components/ui/Card/Card.module.css` exists with **no `Card.tsx`**. The `.card` utility at `utilities.css:148` is largely unused. Pages duplicate card markup locally.

### Modals and dialogs

`ui/dialog.tsx:41` — `w-full max-w-lg`, no horizontal inset, so it goes edge-to-edge at 360px. `ui/Modal/ConfirmModal.module.css:28,72,154` — `max-width: 450px`, `48px` control, `min-width: 100px` button. `NewAppointmentModal.module.css:15` — `max-width: 720px`.

### Loading, empty, and error states

`ui/AppLoading`, `ui/Loading/LoadingSpinner` (sizes from `--app-loading-size-*`), `ui/Error/ErrorAlert`, `components/empty-states/EmptyCarState` and `EmptyAppointmentState`.

---

## Self-check reference

An artboard that follows this system must:

1. Express color only through the tokens above (no loose hex in component rules).
2. Use `system-ui`-first stacks, never a webfont or CDN.
3. Reach 44px touch targets on primary phone controls.
4. Keep `:focus-visible` on every interactive element, distinct from hover.
5. Give every table a scroll container or a stacked narrow layout.
6. Never rely on `dark:` utilities — the theme is the `data-theme` attribute.
