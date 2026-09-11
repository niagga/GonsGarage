# Pruebas y TDD (GonsGarage)

## Principio

**Todo el código de producto se escribe con TDD** (Test-Driven Development): escribir el test que falla, implementar lo mínimo para verde, refactorizar. Los cambios en lógica de negocio o contratos HTTP/API deben ir acompañados de tests automatizados en el mismo PR salvo excepción acordada por el equipo.

## Backend (Go)

- Framework: `testing` + `testify` (`require` / `assert`).
- **Unit tests**: servicios y dominio con dependencias sustituidas por stubs o mocks (ver `internal/service/auth`, `internal/service/car`, `internal/service/appointment`).
- **Tests que usan SQLite (GORM + `go-sqlite3`)**: requieren **CGO** (`CGO_ENABLED=1`). En Windows sin toolchain C esos archivos llevan `//go:build cgo` y no se ejecutan localmente; en **CI (Ubuntu)** el workflow activa CGO para incluir esos paquetes cuando existan bajo el módulo.

## Frontend (Next.js + pnpm)

- Gestor de paquetes: **pnpm** (ver `packageManager` en `frontend/package.json`).
- Tests: **Jest** + Testing Library (`pnpm test`).
- Typecheck: `pnpm typecheck` (TypeScript estricto).

### Suites temporalmente excluidas de Jest

Algunas pruebas de página legacy (`__tests__/app/appointments/page.test.tsx`, `src/app/cars/__tests__/cars.test.tsx`) están en `testPathIgnorePatterns` de `jest.config.js` hasta alinear mocks (`useCars`, hooks) con el árbol actual de `src/`. No eliminar los archivos: reactivarlos quitando la exclusión cuando los mocks estén listos.

## CI

El workflow `.github/workflows/ci.yml` ejecuta `go test`, `pnpm lint`, `pnpm typecheck`, `pnpm test` y `pnpm build`. Debe permanecer verde en `main`.
