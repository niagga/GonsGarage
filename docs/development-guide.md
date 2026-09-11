# Guía de desarrollo GonsGarage

## Servidor de pruebas / LAN (checklist Fase 4)

Plantilla Docker + nginx (puerto **8102**), `.env.prod`, scripts y runbook (secretos, verificación, HTTP LAN, rollback): **[`deploy/README.md`](../deploy/README.md)**. URLs de ejemplo en [`mvp-solo-checklist.md`](./mvp-solo-checklist.md) → *Entorno remoto*.

## Demo local (secuencia mínima, checklist 3.1)

Orden para una máquina “limpia” siguiendo también el [README en la raíz](../README.md):

1. **Clonar** el repo y entrar en la carpeta raíz.
2. **Variables:** `backend/.env` desde `.env.example` y `frontend/.env.local` desde `.env.local.example` (en Windows: `Copy-Item` como en los bloques de abajo).
3. **Bases de datos:** desde la raíz, `docker compose up -d` (Postgres + Redis; Redis no es opcional si usás el compose por defecto).
4. **Backend:** carpeta `backend/`, `go mod download`, luego **`go run ./cmd/api`** (equivale a ejecutar el `main` del paquete `cmd/api`; AutoMigrate crea tablas al arrancar).
5. **Comprobar:** `GET http://localhost:8080/health` y, si querés datos demo, `go run ./cmd/seed-mvp-users` y `go run ./cmd/seed-test-client` (misma consola en `backend/`, con Postgres en marcha; idempotentes).
6. **Frontend:** carpeta `frontend/`, `pnpm install`, `pnpm dev`.
7. **Abrir** `http://localhost:3000` con el API en `http://localhost:8080` (o el valor de `NEXT_PUBLIC_API_URL` sin sufijo `/api/v1`).

Si algo falla, revisá firewall/puertos **5432** (Postgres) y **6379** (Redis) no ocupados por otra instancia.

### Flujo manual mínimo (checklist 3.2)

Objetivo: comprobar en el navegador **login → coche → cita → reparaciones en el detalle del coche** (MVP v1).

1. Abrí `http://localhost:3000/auth/login`. Tras seeds: cliente **cliente.demo@gonsgarage.local** / **ClienteDemo123**; staff **admin.demo@gonsgarage.local** / **AdminDemo123** (ver tabla en [README](../README.md#demo-users)).
2. **Coches:** `/cars` — añadí un coche o abrí uno existente; el detalle está en **`/cars/{id}`** (también hay enlaces desde el dashboard y desde las citas).
3. **Cita:** **`/appointments/new`** — reservá un servicio para el coche elegido; en **`/appointments`** debe aparecer en la lista.
4. **Reparaciones:** en **`/cars/{id}`** la sección de reparaciones llama a **`GET /api/v1/repairs/car/{carId}`** (puede estar vacía hasta que el taller registre reparaciones). Si la API devolvía error SQL por columna **`technician_id`** en bases creadas antes de ese campo, **actualizá el repo y reiniciá** `go run ./cmd/api`: al arrancar se ejecuta un `ALTER` idempotente que asegura esa columna en la tabla `repairs`.

## Requisitos

- Go 1.25+ (directiva `go` en `backend/go.mod`)
- Node.js 22+ con **pnpm** 9+ (`corepack enable` recomendado)
- PostgreSQL 16+ (o Docker)
- Redis: el `docker compose` de la raíz levanta Redis en **6379**; el API tolera fallo de conexión en desarrollo, pero conviene tenerlo en marcha para el mismo comportamiento que en servidor de pruebas

## Variables de entorno (backend)

Definidas o usadas en `backend/cmd/api/main.go`:

| Variable | Uso | Valor por defecto (si vacío) |
|----------|-----|-------------------------------|
| `DATABASE_URL` | DSN PostgreSQL | `postgres://admindb:gonsgarage123@localhost:5432/gonsgarage?sslmode=disable` |
| `REDIS_URL` | Host Redis | `localhost:6379` |
| `JWT_SECRET` | Firma JWT | Clave de desarrollo (log de advertencia) |
| `SERVER_PORT` | Puerto HTTP | `8080` |
| `GIN_MODE` | `release` desactiva modo debug Gin | — |
| `RESET_DATABASE` | `true` elimina tablas antes de migrar (solo desarrollo) | — |
| `CORS_ORIGINS` | Orígenes extra permitidos con **`GIN_MODE=release`** (coma-separado, sin espacios tras coma si podés). Localhost ya está permitido. | — (vacío = solo localhost en release) |

## Levantar PostgreSQL y Redis (Docker)

Desde la **raíz del repositorio** (archivo `docker-compose.yml`):

```powershell
docker compose up -d
```

Credenciales y nombres coinciden con `backend/.env.example` y con los valores por defecto en `backend/cmd/api/main.go` si no defines `DATABASE_URL`.

## Cuentas demo (seeds)

Con la API usando la misma base PostgreSQL (la tabla `users` debe existir: levantá **`go run ./cmd/api`** al menos una vez para AutoMigrate; los seeds **no** migran el esquema).

```powershell
Set-Location backend
go run ./cmd/seed-mvp-users    # admin, manager, employee (solo dev)
go run ./cmd/seed-test-client  # cliente demo
```

| Comando | Usuarios por defecto |
|---------|----------------------|
| `seed-mvp-users` | `admin.demo@gonsgarage.local`, `manager.demo@gonsgarage.local`, `employee.demo@gonsgarage.local` |
| `seed-test-client` | `cliente.demo@gonsgarage.local` / `ClienteDemo123` |

Variables opcionales: `SEED_*_EMAIL`, `SEED_*_PASSWORD`, `DATABASE_URL` (ver comentarios en cada `main.go`).

### Comportamiento idempotente (checklist 3.3)

Ambos comandos consultan el email antes de insertar. Si **ya existe**, log de skip y **exit 0** (no duplican filas). Criterio P0 en [mvp-next-steps.md](./mvp-next-steps.md): ejecutar `seed-mvp-users` dos veces seguidas sin error.

**No** ejecutar seeds contra producción.

## Backend

```powershell
Set-Location backend
Copy-Item .env.example .env   # primera vez
go mod download
go run ./cmd/api
```

- API: `http://localhost:8080` (o `SERVER_PORT`)
- Swagger: `http://localhost:8080/swagger/index.html`
- Health: `http://localhost:8080/health`
- Perfil autenticado: `GET http://localhost:8080/api/v1/auth/me` con cabecera `Authorization: Bearer <token>`
- Empleados: `GET/POST /api/v1/employees/...` solo para roles **admin** o **manager**

### Coches y citas (roles)

- **GET /api/v1/cars** — si el JWT es **client**, solo devuelve sus coches. Si el rol es **employee**, **manager** o **admin**, admite `?ownerId=<uuid>` (coches de ese cliente) o, sin `ownerId`, lista paginada del taller (`limit` por defecto 50, `offset` 0).
- **POST /api/v1/cars** — el cliente no envía `ownerID` (el dueño es siempre él). El taller puede enviar **`ownerID`** (UUID del usuario cliente) para registrar un coche a su nombre.
- **GET /api/v1/appointments** — el cliente solo ve sus citas (el backend fuerza su `customer_id`). El taller puede usar `customerId`, `carId`, `status`, `limit`, `offset`, `sortBy`, `sortOrder`.
- **POST /api/v1/appointments** — el cliente reserva para sí; el taller puede enviar **`customerID`** (camelCase) para crear la cita en nombre del cliente. El coche debe pertenecer a ese cliente.

## Frontend

```powershell
Set-Location frontend
Copy-Item .env.local.example .env.local   # primera vez
pnpm install
pnpm dev
```

`NEXT_PUBLIC_API_URL` en `.env.local` debe ser la **URL base del API sin** path `/api/v1` (por defecto `http://localhost:8080`). El cliente en [`frontend/src/lib/api-client.ts`](../frontend/src/lib/api-client.ts) concatena `/api/v1`. El módulo legacy [`frontend/src/lib/api.ts`](../frontend/src/lib/api.ts) usa el mismo criterio en su `baseURL` por defecto.

## Tests (backend)

```powershell
Set-Location backend
go test ./...
```

## Documentación de código y OpenAPI (swag)

Los artefactos generados viven en **`backend/docs/`** (`docs.go`, `swagger.json`, `swagger.yaml`). El servidor expone la UI en **`http://localhost:8080/swagger/index.html`**.

Regenerar tras cambiar anotaciones `// @Summary`, `// @Router`, etc.:

```powershell
Set-Location backend
go run github.com/swaggo/swag/cmd/swag@v1.8.12 init -g main.go -o docs -d ./cmd/api,./internal/handler,./internal/core/ports --parseInternal
```

- El **general API** (título, `BasePath`, seguridad `BearerAuth`) está en `cmd/api/main.go`.
- Las rutas documentadas están en `internal/handler`; **`/health`** y **`/ready`** se anclan en `cmd/api/swagger_health.go` y `cmd/api/swagger_ready.go`.
- Tipos de petición compartidos (`ports.RegisterRequest`, empleados, etc.) requieren incluir **`internal/core/ports`** en `-d`.

## Persistencia híbrida GORM + sqlx (Fase 2)

- **`internal/platform/sqlxdb`**: `WrapPostgres(*sql.DB)` reutiliza el pool de GORM (`GET /ready` hace `PingContext` con ese handle).
- **`internal/repository/postgres/user_repository.go`**: con dialector **postgres** / **pgx**, todo el CRUD de usuarios va por **sqlx** (lecturas, `EmailExists`, listados, `Create`/`Update`/`UpdatePassword`, soft `Delete`). Con **sqlite** en tests se sigue usando **GORM** en esos métodos.
- **`internal/repository/postgres/car_repository.go`**: con **postgres** / **pgx**, coches por **sqlx** (`Create`, lecturas con dueños vía `IN` batch, `Update`, soft `Delete`, `Restore`, matrícula borrada). **`GetWithRepairs`** carga el coche con sqlx y las reparaciones con **GORM** (`Find` sobre `repairs`) hasta alinear el modelo SQL de reparaciones.
- Helpers compartidos: **`internal/repository/postgres/gorm_sqlx.go`** (`sqlxFromGORM`), **`internal/repository/postgres/fetch_user_models.go`** (`fetchUserModelsByIDs`).
- **`internal/repository/postgres/appointment_repository.go`**: con **postgres** / **pgx**, `Create`, `GetByID`, `Update`, `Delete` (soft) y **`List`** (conteo + filtros `customer_id` / `car_id` / `status`, orden `created_at` o `scheduled_at`) por **sqlx**.
- **`internal/repository/postgres/repair_repository.go`**: con **postgres** / **pgx**, CRUD y listados (`GetByCarID`, `GetByClientID`, `List`, `GetByLicensePlate`) por **sqlx**; **`toDomainRepair`** mapea dominio completo; `INSERT` rellena campos denormalizados del modelo con valores neutros (el modelo GORM ya los tenía).
- **`internal/repository/postgres/employee_repository.go`**: con **postgres** / **pgx**, `Create`, `FindByID`, `Update`, soft `Delete` y **`List`** (conteo + filtros `department` / `is_active`, orden seguro) por **sqlx**; el usuario asociado se enriquece con un `IN` batch vía `fetchUserModelsByIDs` (mismo patrón que coches). Con **sqlite** en tests se sigue usando **GORM**.
- Próximos pasos: repositorios Redis/mock si aplica, y alinear migraciones SQL con nombres de columnas reales (`repairs` / `appointments`) para reducir divergencia con AutoMigrate.

## Tests (cliente / JWT / facturas)

- **Backend**: `go test ./... -short` — incluye `internal/service/invoice` (RU factura propia del cliente) y `internal/middleware` (`GinBearerJWT`).
- **Integración SQLite** (`tests/integration/*`, build tag `cgo`): requiere **CGO habilitado** (p. ej. Linux en CI). Incluye RBAC HTTP de citas y coches con JWT; en Windows sin CGO el paquete se omite en `go test ./...`.
- **Frontend**: `pnpm test` ejecuta **Vitest** (`src/**/*.test.ts`). Suites legacy con `jest.mock` siguen en `pnpm test:jest`.
