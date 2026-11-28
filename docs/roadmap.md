# Roadmap: documentación, configuración y alineación tipo Arnela

**Última reconciliación contra el código:** 27 de noviembre de 2026 (`main` en `98d7f96`).
Los ítems de las Fases 3 y 5 estaban desactualizados: figuraban como pendientes cuando ya estaban entregados. Se corrigieron con la evidencia correspondiente.

**Post‑MVP v1 (prioridades operativas):** ver [mvp-next-steps.md](./mvp-next-steps.md) (P0/P1/P2). **Estado del MVP:** ver [mvp-status-report.md](./mvp-status-report.md). **Orden sugerido y deploy con Postgres Arnela** (`arnela-postgres` / `COMPOSE_OVERRIDE`): [mvp-next-steps — Orden e incidente](./mvp-next-steps.md#orden-sugerido). Este roadmap ordena el trabajo en fases. La parte “como Arnela” depende de completar las especificaciones reales en [arnela-specs.md](./arnela-specs.md).

## Fase 0 — Especificaciones Arnela (bloqueante para paridad)

- [x] Ruta local confirmada: **`D:\Repos\Arnela`** (README, `arnela-rules/`, `docs/DOCUMENTATION_INDEX.md`).
- [x] Resumen en GonsGarage: [specs/arnela/ARNELA_SYNOPSIS.md](./specs/arnela/ARNELA_SYNOPSIS.md) e índice [specs/arnela/README.md](./specs/arnela/README.md); matriz en [arnela-specs.md](./arnela-specs.md).
- [ ] Revisión conjunta con el equipo: priorizar filas de la matriz (compose, CI, auth, migraciones) y convertir en issues/tareas. — **Sin equipo: no aplica como reunión.** La priorización se resolvió unilateralmente en el congelamiento de alcance de MVP v1 ([checklist Fase 1](./mvp-solo-checklist.md#decisiones-cerradas-mvp-v1), 2026-04-17). Queda abierto solo para el backlog de paridad P2.

### P2 — Paridad Arnela (backlog)

Change SDD (archivado): [openspec/changes/archive/2026-04-21-arnela-parity/](../openspec/changes/archive/2026-04-21-arnela-parity/) (`proposal.md`, `tasks.md`). Después de cerrar Fase 0, usar la matriz en [arnela-specs.md](./arnela-specs.md) y los **títulos de issue sugeridos** en la sección **P2 — Paridad Arnela** de [mvp-next-steps.md](./mvp-next-steps.md).

## Fase 1 — Documentación y descubribilidad (este repo)

- [x] Carpeta `docs/` con índice, análisis de aplicación y guía de desarrollo.
- [x] README raíz y Swagger: comandos alineados con `cmd/api` y `docker-compose.yml` en raíz; `deployment/README.md` apunta al compose unificado.
- [x] `frontend/README.md`: enlaces a guía de desarrollo y `.env.local.example`.
- [ ] Opcional: mover notas sueltas (`frontend_migration_plan.md`, etc.) a `docs/history/` con un índice breve.

## Fase 2 — Configuración unificada (objetivo: un solo “camino feliz”)

- [x] `docker-compose.yml` en raíz: **PostgreSQL + Redis** alineados con defaults de `cmd/api` y `backend/.env.example`.
- [x] `backend/.env.example` y `frontend/.env.local.example`.
- [x] [development-guide.md](./development-guide.md) actualizado al flujo actual.
- [x] Perfiles compose para construir/ejecutar **backend + frontend** en contenedor — entregado como stack de despliegue: `docker-compose.prod.yml`, `docker-compose.prod.arnela-network.yml`, `backend/Dockerfile`, `frontend/Dockerfile`. Para dev local se mantiene el flujo nativo (`go run` + `pnpm dev`), que es lo que pedía el ítem.

## Fase 3 — Paridad funcional y API

- [x] Rutas REST de **repairs** (listado por coche + `POST`/`GET`/`PATCH`/`DELETE` por id, permisos staff) — `backend/cmd/api/main.go:622-629`; ver [`application-analysis.md`](./application-analysis.md) y Swagger en `/swagger/`.
- [x] Parte de permisos por rol: **`/employees/*`** solo **admin/manager**; **`/auth/me`** para sesión; registro con rol por defecto **client** (ver [mvp-minimum-phases.md](./mvp-minimum-phases.md) Fase B).
- [x] Swagger regenerado con rutas Gin (`go run … swag init` en `backend/`); tipos de cliente en `frontend/src/types/api.ts` y `lib/api.ts` alineados con repairs CRUD.

## Fase 4 — Prácticas de ingeniería (espejo de Arnela, según Fase 0)

Ajustar según lo que marque la matriz Arnela vs GonsGarage. Candidatos típicos:

- [x] Pipeline CI: `.github/workflows/ci.yml` (Go + vet + test con CGO; pnpm install + lint + typecheck + test + build frontend).
- [x] Workflow deploy: `.github/workflows/deploy.yml` documenta **solo manual** (`workflow_dispatch` + paso que referencia `deploy/README.md`); se le suman `deploy-production.yml` y `build-push-images.yml`.
- [x] Política de versionado de API: prefijo **`/api/v1`** documentado en el encabezado de [`CHANGELOG.md`](../CHANGELOG.md) y en el README.
- [ ] **Mantenimiento del changelog:** `CHANGELOG.md` sigue siendo un borrador — la última entrada es del 2026-06-02 y no cubre las entregas posteriores (facturación fiscal WU1–WU11, fixes de CI del frontend, cutover a Postgres dedicado). Requiere una pasada de cierre.
- [ ] Plantillas de PR / issues: `.github/` solo contiene workflows; no hay `.github/pull_request_template.md` ni `ISSUE_TEMPLATE/`.
- [ ] Observabilidad: no hay logging estructurado (sin `slog`/`logrus`/`zap` en `backend/`) ni endpoint de métricas (`/metrics` inexistente). `/health` y `/ready` existen pero no hay health agregado ni métricas.

## Fase 5 — Producción y seguridad

- [x] Secretos solo por variables de entorno; eliminar defaults inseguros en producción — `.env.prod.example` como plantilla y `.env.prod` excluido en `.gitignore`; criterio en `deploy/README.md` § *Secretos*.
- [x] CORS restrictivo en `release` — `CORS_ORIGINS` parseado en `backend/cmd/api/main.go:445-447` y documentado en `.env.prod.example` + `deploy/README.md` § *CORS*.
- [x] Imagen Docker de backend/frontend y documentación de despliegue acorde al entorno real — `backend/Dockerfile`, `backend/Dockerfile.migrate`, `backend/Dockerfile.fiscal-worker`, `frontend/Dockerfile`, `docker-compose.prod.yml`, `deploy/README.md` (smoke, backup, rollback, HTTPS).

---

**Prioridad sugerida:** las Fases 3 y 5 están cerradas; la Fase 4 tiene dos ítems accionables de bajo costo (**changelog** y **plantillas de PR/issues**) y uno de mayor alcance (**observabilidad**, que conviene decidir antes de exponer el servidor fuera de la LAN). La Fase 1 solo tiene una tarea opcional de ordenamiento documental.
