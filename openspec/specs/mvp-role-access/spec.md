# mvp-role-access Specification

> **Promoción:** catálogo principal desde el change archivado `openspec/changes/archive/2026-04-20-mvp-role-verification/` (2026-04-20). Actualizado desde `openspec/changes/archive/2026-04-20-admin-provision-user-roles/` — fila de aprovisionamiento staff y CI (2026-04-20). Actualizado desde `openspec/changes/archive/2026-04-21-ui-admin-users-discoverability/` — matriz UI `/admin/users`, CI UI y `openspec/specs/staff-user-management-ui/spec.md` (2026-04-21). Actualizado desde `openspec/changes/archive/2026-04-22-workshop-mechanic-vehicle-lifecycle/` — taller (service job), fila de matriz y pruebas CI (2026-04-22). Actualizado desde `openspec/changes/archive/2026-04-24-spare-parts-inventory/` — inventario de peças `/api/v1/parts` y shell `/admin/parts` (`canManageUsers`), matriz, CI y spec `openspec/specs/parts-inventory/spec.md`.

## Purpose

Reproducible **MVP verification** by **role** (`client`, `employee`, `manager`, `admin`): expected **API/UI** access, **idempotent dev seeds** (one user per role via env), and **automated tests** that MUST catch authorization regressions in CI.

## Normative matrix (summary)

| Area | client | employee | manager | admin |
|------|--------|----------|---------|-------|
| Auth + shell (dashboard, cars, appointments nav) | MUST | MUST | MUST | MUST |
| Cars / appointments (tenant rules) | MUST | MUST | MUST | MUST |
| Repairs read by car | MUST (own car) | MUST | MUST | MUST |
| Repairs POST/PUT/DELETE | MUST NOT | MUST* | MUST* | MUST* |
| `/api/v1/employees` | MUST NOT | MUST NOT | MUST | MUST |
| Staff user provisioning `POST /api/v1/admin/users` | MUST NOT | MUST NOT | MUST** | MUST |
| Staff user management UI (`/admin/users` via shell) | MUST NOT | MUST NOT | MUST | MUST |
| Parts inventory HTTP (`GET/POST/PATCH/DELETE /api/v1/parts`) and UI (`/admin/parts` via shell, `canManageUsers`) | MUST NOT | MUST NOT | MUST | MUST |
| Workshop / service job (taller) `POST/PUT/PATCH` bajo `/api/v1/service-jobs` y *checklists* (muta) | MUST NOT | MUST* | MUST* | MUST* |
| Issued invoices HTTP (`/api/v1/invoices`, `/api/v1/invoices/me`) + UI (`/accounting/issued-invoices`, `/my-invoices`) | MUST (own-only) | MUST (staff CRUD) | MUST (staff CRUD) | MUST (staff CRUD) |
| Received invoices HTTP (`/api/v1/received-invoices`) + UI (`/accounting/received-invoices`) | MUST NOT | MUST | MUST | MUST |
| Billing documents HTTP (`/api/v1/billing-documents`, kinds payroll/irs/other) + UI (`/accounting/billing-documents`) | MUST NOT | MUST | MUST | MUST |

\*Subject to existing service rules (car ownership, etc.).  
\*\*`manager` **MAY** create only `employee` and `client` per `openspec/specs/staff-user-provisioning/spec.md`.  
Staff trio on billing/received/issued invoices: `admin` = `manager` = `employee` (`RequireWorkshopStaff` / `User.IsEmployee`). Historical MVP v1 deferral of accounting HTTP is in `openspec/specs/p1-accounting-defer/spec.md` (history, not current P1 HTTP).

## Requirements

### Requirement: Published role–surface matrix

MVP verification docs **SHALL** expose the matrix above (or equivalent), including **staff user provisioning**, **parts inventory** (manager/admin stock), and **workshop (service job)** (taller visit) como filas de API/superficie distintas, alineadas con `openspec/specs/workshop-repair-execution/spec.md` donde aplica.

#### Scenario: Four roles listed

- GIVEN the linked checklist or spec excerpt
- WHEN a reviewer scans role coverage
- THEN all four roles appear with MUST/MUST NOT per row

#### Scenario: Issued invoices own-only for client, staff CRUD

- GIVEN the matrix
- WHEN the issued invoices row is read (`/api/v1/invoices`, `/api/v1/invoices/me`, UI `/accounting/issued-invoices` and `/my-invoices`)
- THEN `client` is MUST own-only and `employee` / `manager` / `admin` are MUST staff CRUD

#### Scenario: Received invoices staff-only

- GIVEN the matrix
- WHEN the received invoices row is read (`/api/v1/received-invoices`)
- THEN `client` is MUST NOT and `employee` / `manager` / `admin` are MUST

#### Scenario: Billing documents staff-only

- GIVEN the matrix
- WHEN the billing documents row is read (`/api/v1/billing-documents`, kinds payroll/irs/other)
- THEN `client` is MUST NOT and `employee` / `manager` / `admin` are MUST

Historical pointer: MVP v1 deferral of accounting HTTP remains in `openspec/specs/p1-accounting-defer/spec.md` as history, not as current deferral of P1 HTTP.

#### Scenario: Provisioning row present

- GIVEN the matrix excerpt
- WHEN a reviewer scans API rows
- THEN the staff user provisioning row appears with correct MUST/MUST NOT per column

#### Scenario: Matrix row for workshop (service job)

- GIVEN the normative matrix summary
- WHEN the matrix is read
- THEN a row documents `Workshop / service job` and distinguishes `client` **MUST NOT** on mutating those routes from staff (`employee` / `manager` / `admin`) **MUST\*** for the HTTP surface in `workshop-repair-execution` scope

#### Scenario: Matrix row for parts inventory

- GIVEN the normative matrix summary
- WHEN the matrix is read
- THEN a row documents **Parts inventory** for `/api/v1/parts` and shell `/admin/parts`, with `client` and `employee` **MUST NOT** and `manager` and `admin` **MUST**

### Requirement: Staff user provisioning HTTP surface

The normative matrix **SHALL** include one row for **staff user provisioning** (authenticated `POST` that creates a `User` with a discrete `role`), aligned with `openspec/specs/staff-user-provisioning/spec.md`.

#### Scenario: Matrix row exists after change

- GIVEN MVP role verification materials
- WHEN the matrix is read
- THEN a row documents `POST /api/v1/admin/users` for `client`, `employee`, `manager`, and `admin` columns per MUST/MUST NOT

### Requirement: Staff user management UI in published matrix

The normative matrix (summary) **SHALL** include one row for **staff user management UI**: in-app navigation to `/admin/users` for provisioning/management, aligned with `openspec/specs/staff-user-provisioning/spec.md` and `openspec/specs/staff-user-management-ui/spec.md` (see matrix row above).

#### Scenario: Matrix documents UI access

- GIVEN MVP role verification materials
- WHEN the matrix is read
- THEN a row documents shell access to `/admin/users` with MUST for `manager` and `admin`, MUST NOT for `client` and `employee`

#### Scenario: Discoverability not URL-only

- GIVEN `admin` or `manager` using only the shell (no manual URL)
- WHEN they use primary navigation
- THEN they **SHALL** be able to open `/admin/users`

### Requirement: CI coverage for provisioning authorization

Automated tests **SHALL** fail CI if `client` or `employee` gains 2xx from the staff user provisioning `POST`, or if `admin` gains 2xx when the body requests `role=admin`.

#### Scenario: Client denied provisioning

- GIVEN `go test` in CI
- WHEN provisioning authorization tests run for JWT `client`
- THEN no 2xx success on the provisioning route

#### Scenario: Admin escalation denied

- GIVEN `go test` in CI
- WHEN a test sends `role=admin` on the provisioning route with JWT `admin`
- THEN the response **MUST NOT** be 2xx success

### Requirement: Idempotent dev seeds per role

The repo **SHALL** ship idempotent seed command(s) creating ≥1 user per role when missing; email/password **SHALL** come from env (no secrets in git).

#### Scenario: First run creates users

- GIVEN dev DB without those emails
- WHEN seed runs with valid `DATABASE_URL` and `SEED_*`
- THEN rows exist for all four roles

#### Scenario: Re-run is safe

- GIVEN users already exist for those emails
- WHEN seed runs again
- THEN exit success without duplicate errors or silent password overwrite

### Requirement: Parts inventory API and UI for manager and admin only

Spare-parts inventory **SHALL** use the same **staff-managers** surface as employees and staff user provisioning: HTTP bajo `/api/v1/parts` con `RequireStaffManagers()`, and in-app UI bajo `/admin/parts` only when `canManageUsers` is true (manager + admin). Domain detail: `openspec/specs/parts-inventory/spec.md`.

#### Scenario: Client forbidden on parts list

- GIVEN JWT role `client`
- WHEN `GET /api/v1/parts`
- THEN response is forbidden (e.g. 403) before part business logic

#### Scenario: Employee forbidden on parts mutator

- GIVEN JWT role `employee`
- WHEN `POST /api/v1/parts` with otherwise valid payload
- THEN response is not 2xx success **solely** due to staff-manager authorization

#### Scenario: Manager allowed past parts role gate

- GIVEN JWT role `manager`
- WHEN `GET /api/v1/parts` or `POST /api/v1/parts` with valid payload
- THEN response is not forbidden **solely** by `RequireStaffManagers` (service may still return 4xx for validation)

#### Scenario: Parts shell nav present for manager in UI tests

- GIVEN `pnpm test` (or equivalent) in CI
- WHEN a test renders `AppShell` with a `manager` user
- THEN a navigation control linking to `/admin/parts` is present (e.g. **Peças (stock)**)

#### Scenario: Parts shell nav absent for client and employee in UI tests

- GIVEN `pnpm test` in CI
- WHEN a test renders `AppShell` with a `client` or `employee` user
- THEN no navigation control links to `/admin/parts`

### Requirement: Employees API for manager and admin only

`/api/v1/employees` **SHALL** return forbidden for `client` and `employee` before business logic.

#### Scenario: Client forbidden

- GIVEN JWT role `client`
- WHEN `GET /api/v1/employees`
- THEN response is forbidden (e.g. 403)

#### Scenario: Manager allowed past role gate

- GIVEN JWT role `manager`
- WHEN `GET /api/v1/employees`
- THEN response is not forbidden **solely** by staff-manager middleware

### Requirement: Client cannot mutate repairs via HTTP

`client` **MUST NOT** get 2xx from repair **POST/PUT/DELETE** **nor** from any **workshop service job** (taller visit) or subresource (checklists) **POST/PUT/PATCH** that mutates; staff **SHALL** mutate per service rules.

(Previously, antes do change `workshop-mechanic-vehicle-lifecycle`: solo `repairs` HTTP, sen *service job* normativo.)

#### Scenario: Client POST repair fails

- GIVEN JWT role `client`
- WHEN `POST /api/v1/repairs` with otherwise valid payload
- THEN response is not 2xx success

#### Scenario: Client mutates service job is denied

- GIVEN JWT role `client`
- WHEN `POST`/`PUT`/`PATCH` to a documented service-job or checklist path for the workshop feature
- THEN response is not 2xx success

#### Scenario: Employee may mutate

- GIVEN JWT role `employee` and payload allowed by `RepairService` (or workshop rules when they apply to the request)
- WHEN `POST /api/v1/repairs`
- THEN response is not rejected **only** because role is `employee`

### Requirement: CI must fail if client mutates service job

Automated tests **SHALL** fail CI if a `client` gets 2xx from mutating a documented service-job or checklist `POST/PUT/PATCH` route in scope of the `workshop-repair-execution` feature.

#### Scenario: CI client denied on create visit

- GIVEN `go test` in CI
- WHEN a test sends a service-job *create* with JWT `client`
- THEN the response is not 2xx success

### Requirement: Invoices not in MVP acceptance (historical)

MVP **v1** completion **SHALL NOT** have depended on invoice Gin routes nor Next invoice pages. That deferral is history in `openspec/specs/p1-accounting-defer/spec.md`. Current P1 HTTP/UI for issued invoices, received invoices, and billing documents **SHALL** match the matrix rows above (not a current deferral).

#### Scenario: Checklist omits invoice HTTP (MVP v1 history)

- GIVEN MVP v1 role verification sign-off
- WHEN historical criteria are checked
- THEN success did not require invoice handlers in `cmd/api`; P1 implementation is documented separately in the matrix and `p1-accounting-defer`

### Requirement: CI authorization regression tests

Tests **SHALL** fail CI if employees list opens to `client`/`employee`, if `client` gains 2xx repair mutation, if `client` gains 2xx on mutating a workshop service job (see prior requirement), if staff user provisioning rules in `staff-user-provisioning` are violated, if `AppShell` omits the `/admin/users` nav entry when `canManageUsers` is true (regression on staff user management UI discovery), **or** if parts inventory authorization regresses (`/api/v1/parts` for `client`/`employee`, or `AppShell` omits `/admin/parts` for manager when `canManageUsers` is true).

#### Scenario: Employees gate tested

- GIVEN `go test` in CI
- WHEN employees authorization tests run
- THEN non-manager non-admin receives forbidden on list route

#### Scenario: Repair mutation denial for client

- GIVEN `go test` in CI
- WHEN client repair mutation tests run
- THEN at least one assertion denies 2xx for `client`

#### Scenario: Provisioning gate tested

- GIVEN `go test` in CI
- WHEN staff user provisioning authorization tests run
- THEN forbidden callers do not receive 2xx and admin-in-body is rejected

#### Scenario: Staff users nav present for manager in UI tests

- GIVEN `pnpm test` (or equivalent) in CI
- WHEN a test renders `AppShell` with a `manager` user
- THEN a navigation control linking to `/admin/users` is present

#### Scenario: Staff users nav absent for client in UI tests

- GIVEN `pnpm test` in CI
- WHEN a test renders `AppShell` with a `client` user
- THEN no navigation control links to `/admin/users`

#### Scenario: Parts HTTP gate tested in Go

- GIVEN `go test` in CI
- WHEN MVP role access tests run for `/api/v1/parts`
- THEN `client` and `employee` do not receive 2xx on list or mutating methods guarded like employees, and `manager`/`admin` pass the role gate (see `backend/internal/handler/mvp_role_access_test.go`)
