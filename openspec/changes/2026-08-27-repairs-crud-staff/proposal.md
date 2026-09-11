# Propuesta: CRUD de Reparaciones (Staff)

## Objetivo
Habilitar la gestión de reparaciones (creación y actualización) para roles de staff (admin, manager, employee).

## Alcance
- API (Backend):
    - `POST /api/v1/repairs`: Crear una nueva reparación.
    - `PATCH /api/v1/repairs/:id`: Actualizar una reparación existente.
- UI (Frontend):
    - Formulario de creación de reparación.
    - Formulario de edición de reparación.
- Documentación:
    - Actualizar Swagger (`docs/swagger/`).

## Requisitos de Seguridad
- Solo `admin`, `manager`, `employee` pueden acceder a estos endpoints.
- Validaciones estrictas en el backend (campos obligatorios, formatos).

## Impacto
- `backend/internal/handler/repair.go`
- `backend/internal/core/services/repair.go`
- `frontend/src/app/(staff)/repairs/...` (estructura nueva o existente)
