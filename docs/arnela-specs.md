# Especificaciones del proyecto Arnela

## Fuente local (canónica)

El proyecto **Arnela** está disponible en esta máquina en:

**`D:\Repos\Arnela`**

No forma parte del workspace de GonsGarage; la información se consulta **leyendo ese repo directamente**.

## Resumen en GonsGarage

Para no duplicar mantenimiento, el equipo puede usar:

- **[docs/specs/arnela/README.md](./specs/arnela/README.md)** — Índice y punteros.  
- **[docs/specs/arnela/ARNELA_SYNOPSIS.md](./specs/arnela/ARNELA_SYNOPSIS.md)** — Resumen de producto, stack, convenciones y diferencias frente a GonsGarage.  
- **Change SDD (paridad / backlog, archivado):** [openspec/changes/archive/2026-04-21-arnela-parity/](../openspec/changes/archive/2026-04-21-arnela-parity/) (`proposal.md`, `tasks.md`).

## Matriz de comparación

**Última revisión:** 2026-04-20 (change `arnela-parity`). Revisar Arnela en `D:\Repos\Arnela` cuando cambie su README o `arnela-rules/`.

| Tema | Arnela | GonsGarage hoy | Acción propuesta | Última revisión |
|------|--------|----------------|------------------|-----------------|
| **Compose / DB** | `docker-compose.yml` en raíz: PostgreSQL + Redis; `.env.example` | `docker-compose.yml` raíz + `backend/.env.example`; prod: `docker-compose.prod.yml` + opción red Arnela (`docker-compose.prod.arnela-network.yml`) | Mantener docs deploy; perfiles opcionales app en contenedor si hace falta | 2026-04-20 |
| **Migraciones** | golang-migrate, SQL en `migrations/` | GORM `AutoMigrate` en `cmd/api` + SQL idempotente en `backend/migrations/` | Issue P2: evaluar trazabilidad SQL vs solo AutoMigrate | 2026-04-20 |
| **CI** | GitHub Actions | **`.github/workflows/ci.yml`** (Go vet + test `-race` en Linux; pnpm lint, typecheck, test, build frontend) + `deploy.yml` manual | Mantener verde en PR; ampliar cobertura si aplica | 2026-04-20 |
| **Estructura docs** | `docs/DOCUMENTATION_INDEX.md` + `arnela-rules/` | `docs/` con índice, guías, roadmap, esta matriz | Issue P2: `docs/DOCUMENTATION_INDEX.md` propio (opcional) | 2026-04-20 |
| **Auth / API** | JWT; `GET /api/v1/auth/me`; rate limiting; CRUD documentado | JWT; **`GET /api/v1/auth/me`** (`auth_handler.Me`); register/login; cars, appointments, employees, repairs, accounting P1, etc. | Issue P2: rate limiting y políticas finas vs Arnela | 2026-04-20 |
| **Frontend** | Next 16, pnpm, Tailwind v4, Shadcn, grupos de rutas | **Next 16**, **pnpm**, App Router, sin Tailwind v4/Shadcn replicados | Mantener paridad de major y cerrar brechas de UI por fases | 2026-09-20 |
| **Observabilidad** | `/health` y readiness (p. ej. `/readiness` en README Arnela) | **`GET /health`** y **`GET /ready`** en API; nginx proxifica ambos en el mismo origen (`:8102`) | Documentar equivalencia nombre; métricas = roadmap Fase 4 | 2026-04-20 |

## Si la ruta cambia de máquina

Actualizar la ruta absoluta en este archivo y en `docs/specs/arnela/ARNELA_SYNOPSIS.md`. Alternativa estable: clonar Arnela como submódulo bajo `docs/external/arnela` y enlazar desde aquí.
