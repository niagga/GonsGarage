# Tema Shadcn ↔ marca GonsGarage

Variables HSL (sin función `hsl()`) definidas en `frontend/src/styles/shadcn-theme.css`. Tailwind las expone como `bg-primary`, `text-foreground`, etc. vía `frontend/tailwind.config.ts`.

| Token marca / app (`tokens.css`) | Variable Shadcn | Uso típico |
|----------------------------------|-----------------|------------|
| `--color-primary` / `--brand-accent` (claro) | `--primary` | Botón primario, anillos focus |
| `--text-on-primary` | `--primary-foreground` | Texto sobre primario |
| `--surface-page` (sensación visual) | `--background` | Fondo base en utilidades Shadcn |
| `--text-primary` | `--foreground` | Texto principal |
| `--color-error` / `--brand-signal` | `--destructive` | Eliminar / acciones destructivas |
| `--border-default` | `--border`, `--input` | Bordes de campos y tarjetas |
| `--surface-muted` | `--muted`, `--accent` | Superficies secundarias / hover |
| `--text-muted` | `--muted-foreground` | Texto secundario en forms |

**Dark:** `html[data-theme='dark']` sobrescribe las mismas variables Shadcn para alinearlas con la rampa oscura de `tokens.css`. Tema **Taller nocturno** (2026-11-27): acero azulado frío — página `#0B0F14`, panel `#131A22`, elevado `#1A232C`, bordes `#253040`/`#36455A`, texto `#E9F0F7`, primario `#38BDF8`, destructivo `#F87171`.

**Mantenimiento:** al cambiar una marca en `tokens.css`, revisar si hace falta ajustar el triple HSL correspondiente en `shadcn-theme.css` y esta tabla.
