package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/gaston-garcia-cegid/gonsgarage/internal/core/ports"
	"github.com/gaston-garcia-cegid/gonsgarage/internal/domain"
)

func parseAppointmentScheduledAt(raw string) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, fmt.Errorf("scheduled time required")
	}
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t, nil
	}
	if t, err := time.Parse("2006-01-02T15:04:05Z07:00", raw); err == nil {
		return t, nil
	}
	// HTML datetime-local (no timezone): interpret in local wall clock
	if t, err := time.ParseInLocation("2006-01-02T15:04", raw, time.Local); err == nil {
		return t, nil
	}
	return time.ParseInLocation("2006-01-02T15:04:05", raw, time.Local)
}

type AppointmentHandler struct {
	appointmentService ports.AppointmentService
}

func NewAppointmentHandler(appointmentService ports.AppointmentService) *AppointmentHandler {
	return &AppointmentHandler{
		appointmentService: appointmentService,
	}
}

// CreateAppointmentRequest represents the request payload for creating an appointment
type CreateAppointmentRequest struct {
	CustomerID    string `json:"customerID"`    // ✅ camelCase
	EmployeeID    string `json:"employeeID"`    // ✅ camelCase
	CarID         string `json:"carId"`         // frontend / Agent.md
	ScheduledTime string `json:"scheduledTime"` // legacy
	ScheduledAt   string `json:"scheduledAt"`   // ✅ camelCase
	Reason        string `json:"reason"`
	Notes         string `json:"notes"`
	Status        string `json:"status"`
	ServiceType   string `json:"serviceType"` // ✅ camelCase
}

// UpdateAppointmentRequest represents the request payload for updating an appointment
type UpdateAppointmentRequest struct {
	CustomerID    string `json:"customerID"`
	EmployeeID    string `json:"employeeID"`
	CarID         string `json:"carId"`         // ✅ camelCase
	ScheduledTime string `json:"scheduledTime"` // ✅ camelCase
	ScheduledAt   string `json:"scheduledAt"`   // ✅ camelCase
	Reason        string `json:"reason"`
	Notes         string `json:"notes"`
	Status        string `json:"status"`
	ServiceType   string `json:"serviceType"` // ✅ camelCase
}

// AppointmentResponse represents the response payload for an appointment
type AppointmentResponse struct {
	ID         string  `json:"id"`
	ClientName string  `json:"clientName"`
	CarID      string  `json:"carId"`
	Service    string  `json:"service"`
	Date       string  `json:"date"`
	Time       string  `json:"time"`
	Status     string  `json:"status"`
	Notes      string  `json:"notes"`
	CreatedAt  string  `json:"createdAt"`
	UpdatedAt  string  `json:"updatedAt"`
	DeletedAt  *string `json:"deletedAt,omitempty"`
}

// CreateAppointment agenda una cita (cliente: para sí; taller: customerID opcional).
// @Summary     Crear cita
// @Tags        appointments
// @Security    BearerAuth
// @Accept      json
// @Produce     json
// @Param       body body CreateAppointmentRequest true "carID, scheduledAt (RFC3339), serviceType, etc."
// @Success     201 {object} AppointmentResponse
// @Failure     400 {object} SwaggerMessage
// @Failure     401 {object} SwaggerMessage
// @Failure     403 {object} SwaggerMessage
// @Router      /api/v1/appointments [post]
func (h *AppointmentHandler) CreateAppointment(c *gin.Context) {
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

	var req CreateAppointmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	carID, err := uuid.Parse(strings.TrimSpace(req.CarID))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid car ID"})
		return
	}
	schedRaw := strings.TrimSpace(req.ScheduledAt)
	if schedRaw == "" {
		schedRaw = strings.TrimSpace(req.ScheduledTime)
	}
	scheduledAt, err := parseAppointmentScheduledAt(schedRaw)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid scheduled time format"})
		return
	}

		// Clients: customer = self. Staff: leave Nil unless explicitly set so the
		// service can derive the car owner (avoids 403 when FE omits customerID).
		customerID := uuid.Nil
		roleVal, _ := c.Get("userRole")
		roleStr, _ := roleVal.(string)
		if roleStr == "" || roleStr == domain.RoleClient {
			customerID = userID
		} else if strings.TrimSpace(req.CustomerID) != "" {
			customerID, err = uuid.Parse(strings.TrimSpace(req.CustomerID))
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid customerID"})
				return
			}
		}

	appointment := &domain.Appointment{
		CustomerID:  customerID,
		CarID:       carID,
		ScheduledAt: scheduledAt,
		Status:      domain.AppointmentStatus(req.Status),
		Notes:       req.Notes,
		ServiceType: req.ServiceType,
	}

	createdAppointment, err := h.appointmentService.CreateAppointment(c.Request.Context(), appointment, userID)
	if err != nil {
		if err == domain.ErrUnauthorizedAccess {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		if err == domain.ErrAppointmentOutsideBusinessHours {
			c.JSON(http.StatusBadRequest, gin.H{"error": "El horario debe estar entre 9:30 y 12:30 o entre 14:00 y 17:30."})
			return
		}
		if err == domain.ErrAppointmentDailyCapReached {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Ya hay 8 turnos agendados ese día; elegí otra fecha u horario."})
			return
		}
		if err == domain.ErrAppointmentAlreadyExists {
			c.JSON(http.StatusConflict, gin.H{"error": "appointment with this ID already exists"})
			return
		}
		if err == domain.ErrInvalidAppointmentData {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid appointment data"})
			return
		}
		if err == domain.ErrUserNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	response := h.toAppointmentResponse(createdAppointment)
	c.JSON(http.StatusCreated, response)
}

// GetAppointment obtiene una cita por ID.
// @Summary     Obtener cita
// @Tags        appointments
// @Security    BearerAuth
// @Produce     json
// @Param       id path string true "UUID de la cita"
// @Success     200 {object} AppointmentResponse
// @Failure     400 {object} SwaggerMessage
// @Failure     401 {object} SwaggerMessage
// @Failure     403 {object} SwaggerMessage
// @Failure     404 {object} SwaggerMessage
// @Router      /api/v1/appointments/{id} [get]
func (h *AppointmentHandler) GetAppointment(c *gin.Context) {
	// Get user from Gin context
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
	// Parse appointment ID from URL parameter
	appointmentIDStr := c.Param("id")
	appointmentID, err := uuid.Parse(appointmentIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid appointment ID"})
		return
	}
	appointment, err := h.appointmentService.GetAppointment(c.Request.Context(), appointmentID, userID)
	if err != nil {
		if err == domain.ErrAppointmentNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "appointment not found"})
			return
		}
		if err == domain.ErrUnauthorizedAccess {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		if err == domain.ErrUserNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	// Convert to response
	response := h.toAppointmentResponse(appointment)

	c.JSON(http.StatusOK, response)
}

// ListAppointments lista citas (cliente: solo las suyas; taller: filtros opcionales).
// @Summary     Listar citas
// @Tags        appointments
// @Security    BearerAuth
// @Produce     json
// @Param       customerId query string false "Filtro UUID cliente (staff)"
// @Param       carId query string false "Filtro UUID coche"
// @Param       status query string false "Estado (scheduled, confirmed, ...)"
// @Param       limit query int false "Límite (default 10)"
// @Param       offset query int false "Offset"
// @Param       sortBy query string false "created_at o scheduled_at"
// @Param       sortOrder query string false "ASC o DESC"
// @Success     200 {array} AppointmentResponse
// @Failure     400 {object} SwaggerMessage
// @Failure     401 {object} SwaggerMessage
// @Failure     403 {object} SwaggerMessage
// @Router      /api/v1/appointments [get]
func (h *AppointmentHandler) ListAppointments(c *gin.Context) {
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

	filters := &ports.AppointmentFilters{}
	if cid := c.Query("customerId"); cid != "" {
		id, perr := uuid.Parse(cid)
		if perr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid customerId"})
			return
		}
		filters.CustomerID = &id
	}
	if carIDStr := c.Query("carId"); carIDStr != "" {
		id, perr := uuid.Parse(carIDStr)
		if perr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid carId"})
			return
		}
		filters.CarID = &id
	}
	if st := c.Query("status"); st != "" {
		filters.Status = &st
	}
	filters.SortBy = c.DefaultQuery("sortBy", "created_at")
	filters.SortOrder = c.DefaultQuery("sortOrder", "DESC")
	filters.Limit, _ = strconv.Atoi(c.DefaultQuery("limit", "10"))
	filters.Offset, _ = strconv.Atoi(c.DefaultQuery("offset", "0"))

	appointments, _, err := h.appointmentService.ListAppointments(c.Request.Context(), userID, filters)
	if err != nil {
		if err == domain.ErrUnauthorizedAccess {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		if err == domain.ErrUserNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	responses := make([]AppointmentResponse, 0, len(appointments))
	for _, appointment := range appointments {
		responses = append(responses, h.toAppointmentResponse(appointment))
	}
	c.JSON(http.StatusOK, responses)
}

// UpdateAppointment modifica una cita (scheduledAt/carId opcionales).
// @Summary     Actualizar cita
// @Tags        appointments
// @Security    BearerAuth
// @Accept      json
// @Produce     json
// @Param       id path string true "UUID de la cita"
// @Param       body body UpdateAppointmentRequest true "Campos a fusionar"
// @Success     200 {object} AppointmentResponse
// @Failure     400 {object} SwaggerMessage
// @Failure     401 {object} SwaggerMessage
// @Failure     403 {object} SwaggerMessage
// @Failure     404 {object} SwaggerMessage
// @Router      /api/v1/appointments/{id} [put]
func (h *AppointmentHandler) UpdateAppointment(c *gin.Context) {
	// Get user from Gin context
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

	// Parse appointment ID from URL parameter
	appointmentIDStr := c.Param("id")
	appointmentID, err := uuid.Parse(appointmentIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid appointment ID"})
		return
	}

	// Parse request
	var req UpdateAppointmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	// Convert to domain object
	// customerUUID, err := uuid.Parse(req.CustomerID)
	// if err != nil {
	// 	c.JSON(http.StatusBadRequest, gin.H{"error": "invalid customer ID"})
	// 	return
	// }

	patch := &domain.Appointment{ID: appointmentID}
	schedRaw := strings.TrimSpace(req.ScheduledAt)
	if schedRaw == "" {
		schedRaw = strings.TrimSpace(req.ScheduledTime)
	}
	if schedRaw != "" {
		t, perr := parseAppointmentScheduledAt(schedRaw)
		if perr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid scheduled time format"})
			return
		}
		patch.ScheduledAt = t
	}
	carPatch := strings.TrimSpace(req.CarID)
	if carPatch != "" {
		carUUID, perr := uuid.Parse(carPatch)
		if perr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid car ID"})
			return
		}
		patch.CarID = carUUID
	}
	if req.Status != "" {
		patch.Status = domain.AppointmentStatus(req.Status)
	}
	patch.Notes = req.Notes
	if req.ServiceType != "" {
		patch.ServiceType = req.ServiceType
	}

	appointment, err := h.appointmentService.UpdateAppointment(c.Request.Context(), patch, userID)
	if err != nil {
		if err == domain.ErrAppointmentNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "appointment not found"})
			return
		}
		if err == domain.ErrUnauthorizedAccess {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		if err == domain.ErrAppointmentOutsideBusinessHours {
			c.JSON(http.StatusBadRequest, gin.H{"error": "El horario debe estar entre 9:30 y 12:30 o entre 14:00 y 17:30."})
			return
		}
		if err == domain.ErrAppointmentDailyCapReached {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Ya hay 8 turnos agendados ese día; elegí otra fecha u horario."})
			return
		}
		if err == domain.ErrInvalidAppointmentData {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid appointment data"})
			return
		}
		if err == domain.ErrUserNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	// Convert to response
	response := h.toAppointmentResponse(appointment)
	c.JSON(http.StatusOK, response)
}

// DeleteAppointment elimina una cita.
// @Summary     Eliminar cita
// @Tags        appointments
// @Security    BearerAuth
// @Param       id path string true "UUID de la cita"
// @Success     204 "Sin cuerpo"
// @Failure     400 {object} SwaggerMessage
// @Failure     401 {object} SwaggerMessage
// @Failure     403 {object} SwaggerMessage
// @Failure     404 {object} SwaggerMessage
// @Router      /api/v1/appointments/{id} [delete]
func (h *AppointmentHandler) DeleteAppointment(c *gin.Context) {
	// Get user from Gin context
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

	// Parse appointment ID from URL parameter
	appointmentIDStr := c.Param("id")
	appointmentID, err := uuid.Parse(appointmentIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid appointment ID"})
		return
	}

	// Delete appointment
	err = h.appointmentService.DeleteAppointment(c.Request.Context(), appointmentID, userID)
	if err != nil {
		if err == domain.ErrAppointmentNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "appointment not found"})
			return
		}
		if err == domain.ErrUnauthorizedAccess {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.Status(http.StatusNoContent)
}

// Helper methods

func (h *AppointmentHandler) toAppointmentResponse(appointment *domain.Appointment) AppointmentResponse {
	// Extrai nome do cliente (podes buscar do domínio ou da DB, ou deixar vazio se não existir)
	clientName := "" // TODO: buscar nome do cliente se necessário

	// Extrai service (pode ser appointment.ServiceType ou outro campo)
	service := appointment.ServiceType

	// Divide data/hora
	date := appointment.ScheduledAt.Format("2006-01-02")
	time := appointment.ScheduledAt.Format("15:04")

	var deletedAt *string
	if appointment.DeletedAt != nil {
		s := appointment.DeletedAt.Format("2006-01-02T15:04:05Z07:00")
		deletedAt = &s
	}

	return AppointmentResponse{
		ID:         appointment.ID.String(),
		ClientName: clientName,
		CarID:      appointment.CarID.String(),
		Service:    service,
		Date:       date,
		Time:       time,
		Status:     string(appointment.Status),
		Notes:      appointment.Notes,
		CreatedAt:  appointment.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:  appointment.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		DeletedAt:  deletedAt,
	}
}
