# Siguientes pasos (post‑MVP v1)

**Última revisión:** 2026-11-27.

Priorización después del cierre operativo del checklist MVP (fases 1–6) y de [`mvp-role-access`](../openspec/specs/mvp-role-access/spec.md). La **contabilidad P1** (API + UI) ya está en `main`; el spec [`p1-accounting-defer`](../openspec/specs/p1-accounting-defer/spec.md) documenta el aplazamiento histórico del MVP v1 original frente a lo entregado después.

## Orden sugerido

1. **P0** — filas P0 de la [tabla](#tabla-p0--p1--p2) (seeds, CI `-race`, secretos).  
2. **Fiabilidad deploy (Arnela)** — **cerrado en código:** ambos caminos de deploy rechazan el acople a la red/DB de Arnela (ver [Incidente DNS arnela-postgres](#incidente-dns-arnela-postgres) y [`deploy/README.md`](../deploy/README.md)). Queda **una acción manual en el host**: si `.env.prod` todavía usa `arnela-postgres`, hacer el cutover a `postgres` antes del próximo deploy.  
3. **P1** — misma tabla (admin `/employees`, issue GitHub).  
4. **P2** — misma tabla y [`roadmap.md`](./roadmap.md).

## Incidente DNS arnela-postgres

Con **`.env.prod`** apuntando a **`arnela-postgres`** pero un `docker compose` **solo** con `docker-compose.prod.yml`, el API queda en la red `gonsgarage-network` y **no** resuelve ese hostname (Postgres vive en la red de Arnela): *lookup arnela-postgres … no such host*, contenedor en **reinicio**, **502** en `/health`. **Fix:** `export COMPOSE_OVERRIDE=docker-compose.prod.arnela-network.yml` (y verificar `name:` de la red externa en ese YAML) antes de `up`, o el equivalente manual con dos `-f`. Detalle: [`deploy/README.md`](../deploy/README.md).

**Estado:** ese camino quedó **cerrado**. Con el cutover P0-T1, producción usa su propio servicio `postgres` y las guardas de deploy rechazan el acople:

| Camino | Qué rechaza | Dónde |
|--------|-------------|-------|
| Servidor (`git pull` + compose) | `COMPOSE_OVERRIDE` con `arnela`, `DATABASE_URL` con `arnela-postgres`, compose sin servicio `postgres` | `scripts/deploy-prod.sh` → `assert_gonsgarage_isolation()` |
| Desde tu PC (SCP) | el override Arnela en tu máquina y, en el servidor, antes de cada `up`: `.env.prod` con `arnela-postgres` y compose sin `postgres` | `deploy.ps1` (probar con `-DryRun`) |

`docker-compose.prod.arnela-network.yml` queda como legado y `deploy.ps1` ya no lo copia. Lo que sigue pendiente no es código sino la acción en el host: confirmar que `.env.prod` apunta a `postgres` y no a `arnela-postgres`.

## Tabla P0 / P1 / P2

| P | Ítem | Estado | Criterio / evidencia |
|---|------|--------|----------------------|
| **P0** | Seeds en Postgres dev | **Hecho** 2026-06-02 | `seed-mvp-users` ×2: 1.ª creó 3 usuarios; 2.ª log `already exists … Skip`, exit 0. |
| **P0** | CI backend con carrera de datos | **CI** (local requiere gcc) | `.github/workflows/ci.yml` → `go test ./... -race`. En Windows sin gcc: tests unitarios OK con `CGO_ENABLED=0`; integración (`//go:build cgo`) solo en Linux CI. |
| **P0** | Aislamiento Arnela en los dos caminos de deploy | **Hecho** 2026-11-27 | `deploy.ps1` aplica las mismas guardas que `scripts/deploy-prod.sh` (override, `DATABASE_URL`, servicio `postgres`) y expone `-DryRun`; verificado con `.env.prod` falso, compose sin `postgres`, override Arnela y override inexistente. |
| **P0** | Secretos en servidor real | **Doc** | `.env.prod` en `.gitignore`; runbook [`deploy/README.md`](../deploy/README.md). Revisión periódica manual en el host. |
| **P1** | Cobertura middleware `admin` en `/employees` | **Hecho** | `TestMVPAccess_EmployeesGET_AdminReachesHandler` — PASS (2026-06-02). |
| **P1** | Issue en GitHub (tracker) | **Pendiente** | Plantilla: [`github-issue-p1-employees-admin.md`](./github-issue-p1-employees-admin.md). Crear en GitHub y marcar checkbox abajo. |
| **P2** | Matriz Arnela Fase 0 | Pendiente | Issues desde [`arnela-specs.md`](./arnela-specs.md). |
| **P2** | Changelog / versionado API | Pendiente | Ver [`CHANGELOG.md`](../CHANGELOG.md) (borrador iniciado 2026-06-02). |

## P2 — Paridad Arnela (issues sugeridos)

Change SDD (archivado): [`openspec/changes/archive/2026-04-21-arnela-parity/`](../openspec/changes/archive/2026-04-21-arnela-parity/) — matriz en [`arnela-specs.md`](./arnela-specs.md). Crear issues (o tareas) con estos títulos; cuerpo: enlace a este doc + fila de la matriz.

1. **`docs: evaluar golang-migrate vs GORM AutoMigrate`** — Trazabilidad SQL y revisión de esquema al estilo Arnela sin romper el arranque actual en `cmd/api`.
2. **`docs: índice DOCUMENTATION_INDEX.md`** — Espejo opcional de Arnela para descubribilidad (`docs/DOCUMENTATION_INDEX.md`).
3. **`api: rate limiting (paridad Arnela)`** — Alinear con políticas de Arnela si el producto lo exige.
4. **`spike: Next / Tailwind / Shadcn (alcance mínimo)`** — Upgrade solo si el beneficio justifica el churn (matriz: Arnela Next 16 + Tailwind v4).
5. **`docs: matriz roles GonsGarage vs arnela-rules`** — Comparación explícita de permisos por rol frente a `arnela-rules/`.

## Checklist issue GitHub (P1)

- [ ] Issue creado en el repositorio (título sugerido: `test: GET /api/v1/employees accepts admin JWT`). Cuerpo listo para pegar: [`github-issue-p1-employees-admin.md`](./github-issue-p1-employees-admin.md).

> **Nota:** el test ya existe y pasa; el issue sirve como trazabilidad en el tracker (roadmap Fase 0 / P1), no como trabajo de implementación pendiente.

## Referencias

- Checklist MVP: [`mvp-solo-checklist.md`](./mvp-solo-checklist.md)
- Plan épico archivado: `openspec/changes/archive/2026-04-20-mvp-funcionando-plan/proposal.md`
- Change SDD archivado: [`mvp-post-v1-followup` proposal](../openspec/changes/archive/2026-04-20-mvp-post-v1-followup/proposal.md)
