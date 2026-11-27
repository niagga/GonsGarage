# Tareas: CRUD de Reparaciones (Staff)

## Fase 1: Backend
- [ ] Definir DTOs para `CreateRepairRequest` y `UpdateRepairRequest`.
- [ ] Implementar validaciones en el handler Gin.
- [ ] Implementar lógica en el Service (GORM).
- [ ] Registrar rutas en el router Gin.
- [ ] Actualizar anotaciones Swagger para los nuevos endpoints.
- [ ] Ejecutar `go test ./internal/core/services/repair_test.go` (o crear tests unitarios).

## Fase 2: Frontend
- [ ] Crear/actualizar `src/types/api.ts` para incluir tipos de `Repair`.
- [ ] Implementar servicios API en `frontend/src/lib/api.ts`.
- [ ] Desarrollar formulario con `react-hook-form` y `shadcn/ui`.
- [ ] Conectar formulario con el store de Zustand/API.

## Fase 3: Integración
- [ ] Verificación cruzada (staff UI vs Swagger).
- [ ] Test E2E de creación/edición.
