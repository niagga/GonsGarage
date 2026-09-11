package handler

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/gaston-garcia-cegid/gonsgarage/internal/domain"
)

// ListRepairsByCar returns repairs for a car (client: own car only).
// @Summary     Listar reparaciones por coche
// @Description Cliente: solo su coche. Personal del taller según reglas del dominio.
// @Tags        repairs
// @Security    BearerAuth
// @Produce     json
// @Param       carId path string true "UUID del coche"
// @Success     200 {array} RepairResponse
// @Failure     400 {object} SwaggerMessage
// @Failure     401 {object} SwaggerMessage
// @Failure     403 {object} SwaggerMessage
// @Failure     500 {object} SwaggerMessage
// @Router      /api/v1/repairs/car/{carId} [get]
func (h *RepairHandler) ListRepairsByCar(c *gin.Context) {
	userIDStr, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user ID"})
		return
	}

	carIDStr := c.Param("carId")
	if carIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "car ID is required"})
		return
	}

	carID, err := uuid.Parse(carIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid car ID"})
		return
	}

	repairs, err := h.repairService.GetRepairsByCarID(c.Request.Context(), carID, userID)
	if err != nil {
		if err == domain.ErrUnauthorizedAccess {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if repairs == nil {
		c.JSON(http.StatusOK, []domain.Repair{})
		return
	}

	out := make([]domain.Repair, 0, len(repairs))
	for _, r := range repairs {
		if r != nil {
			out = append(out, *r)
		}
	}
	c.JSON(http.StatusOK, out)
}

type createRepairJSON struct {
	CarID       string  `json:"car_id" binding:"required"`
	Description string  `json:"description" binding:"required"`
	Status      string  `json:"status"`
	Cost        float64 `json:"cost"`
	StartedAt   *string `json:"started_at"`
	StartDate   *string `json:"start_date"`
}

type updateRepairJSON struct {
	Description *string  `json:"description"`
	Status        *string  `json:"status"`
	Cost          *float64 `json:"cost"`
	StartedAt     *string  `json:"started_at"`
	CompletedAt   *string  `json:"completed_at"`
}

func parseOptionalTime(s *string) (*time.Time, error) {
	if s == nil {
		return nil, nil
	}
	v := strings.TrimSpace(*s)
	if v == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, v)
	if err == nil {
		return &t, nil
	}
	t2, err2 := time.Parse("2006-01-02", v)
	if err2 != nil {
		return nil, fmt.Errorf("invalid datetime: %w", err)
	}
	return &t2, nil
}

func firstNonEmptyTimePtr(a, b *string) *string {
	if a != nil && strings.TrimSpace(*a) != "" {
		return a
	}
	if b != nil && strings.TrimSpace(*b) != "" {
		return b
	}
	return nil
}

// CreateRepair creates a repair (employee/manager/admin only; enforced in service).
// @Summary     Crear reparación
// @Description Personal del taller. Clientes reciben 403.
// @Tags        repairs
// @Security    BearerAuth
// @Accept      json
// @Produce     json
// @Param       body body createRepairJSON true "Datos (snake_case; started_at o start_date RFC3339 o fecha)"
// @Success     201 {object} RepairAPIModel
// @Failure     400 {object} SwaggerMessage
// @Failure     401 {object} SwaggerMessage
// @Failure     403 {object} SwaggerMessage
// @Failure     500 {object} SwaggerMessage
// @Router      /api/v1/repairs [post]
func (h *RepairHandler) GinCreateRepair(c *gin.Context) {
	userIDStr, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user ID"})
		return
	}

	var req createRepairJSON
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	carID, err := uuid.Parse(strings.TrimSpace(req.CarID))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid car ID"})
		return
	}

	startedAt, err := parseOptionalTime(firstNonEmptyTimePtr(req.StartedAt, req.StartDate))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	st := domain.RepairStatus(req.Status)
	if strings.TrimSpace(req.Status) == "" {
		st = domain.RepairStatusPending
	}

	repair := &domain.Repair{
		CarID:       carID,
		Description: strings.TrimSpace(req.Description),
		Status:      st,
		Cost:        req.Cost,
		StartedAt:   startedAt,
	}

	created, err := h.repairService.CreateRepair(c.Request.Context(), repair, userID)
	if err != nil {
		if err == domain.ErrUnauthorizedAccess {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, created)
}

// GetRepair returns one repair by ID (client: only for own car).
// @Summary     Obtener reparación por ID
// @Tags        repairs
// @Security    BearerAuth
// @Produce     json
// @Param       id path string true "UUID reparación"
// @Success     200 {object} RepairAPIModel
// @Failure     400 {object} SwaggerMessage
// @Failure     401 {object} SwaggerMessage
// @Failure     403 {object} SwaggerMessage
// @Failure     404 {object} SwaggerMessage
// @Router      /api/v1/repairs/{id} [get]
func (h *RepairHandler) GinGetRepair(c *gin.Context) {
	userIDStr, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user ID"})
		return
	}
	repairID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid repair ID"})
		return
	}

	repair, err := h.repairService.GetRepair(c.Request.Context(), repairID, userID)
	if err != nil {
		if err == domain.ErrRepairNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "repair not found"})
			return
		}
		if err == domain.ErrUnauthorizedAccess {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, repair)
}

// PatchRepair updates a repair (employee/manager/admin only).
// @Summary     Actualizar reparación
// @Tags        repairs
// @Security    BearerAuth
// @Accept      json
// @Produce     json
// @Param       id path string true "UUID reparación"
// @Param       body body updateRepairJSON true "Campos opcionales"
// @Success     200 {object} RepairAPIModel
// @Failure     400 {object} SwaggerMessage
// @Failure     401 {object} SwaggerMessage
// @Failure     403 {object} SwaggerMessage
// @Failure     404 {object} SwaggerMessage
// @Router      /api/v1/repairs/{id} [patch]
func (h *RepairHandler) GinPatchRepair(c *gin.Context) {
	userIDStr, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user ID"})
		return
	}
	repairID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid repair ID"})
		return
	}

	var req updateRepairJSON
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	existing, err := h.repairService.GetRepair(c.Request.Context(), repairID, userID)
	if err != nil {
		if err == domain.ErrRepairNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "repair not found"})
			return
		}
		if err == domain.ErrUnauthorizedAccess {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	merged := *existing
	if req.Description != nil {
		merged.Description = strings.TrimSpace(*req.Description)
	}
	if req.Status != nil {
		merged.Status = domain.RepairStatus(strings.TrimSpace(*req.Status))
	}
	if req.Cost != nil {
		merged.Cost = *req.Cost
	}
	if req.StartedAt != nil {
		t, err := parseOptionalTime(req.StartedAt)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		merged.StartedAt = t
	}
	if req.CompletedAt != nil {
		t, err := parseOptionalTime(req.CompletedAt)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		merged.CompletedAt = t
	}

	updated, err := h.repairService.UpdateRepair(c.Request.Context(), &merged, userID)
	if err != nil {
		if err == domain.ErrUnauthorizedAccess {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, updated)
}

// DeleteRepair soft-deletes a repair (employee/manager/admin only).
// @Summary     Eliminar reparación (soft delete)
// @Tags        repairs
// @Security    BearerAuth
// @Produce     json
// @Param       id path string true "UUID reparación"
// @Success     204 "Sin cuerpo"
// @Failure     400 {object} SwaggerMessage
// @Failure     401 {object} SwaggerMessage
// @Failure     403 {object} SwaggerMessage
// @Failure     404 {object} SwaggerMessage
// @Router      /api/v1/repairs/{id} [delete]
func (h *RepairHandler) GinDeleteRepair(c *gin.Context) {
	userIDStr, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user ID"})
		return
	}
	repairID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid repair ID"})
		return
	}

	err = h.repairService.DeleteRepair(c.Request.Context(), repairID, userID)
	if err != nil {
		if err == domain.ErrUnauthorizedAccess {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		if err == domain.ErrRepairNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "repair not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}
