package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/gaston-garcia-cegid/gonsgarage/internal/core/ports"
	"github.com/gaston-garcia-cegid/gonsgarage/internal/domain"
)

type CarHandler struct {
	carService ports.CarService
}

func NewCarHandler(carService ports.CarService) *CarHandler {
	return &CarHandler{
		carService: carService,
	}
}

// CreateCarRequest represents the request payload for creating a car
type CreateCarRequest struct {
	Make         string `json:"make"`
	Model        string `json:"model"`
	Year         int    `json:"year"`
	LicensePlate string `json:"licensePlate"`
	VIN          string `json:"vin"`
	Color        string `json:"color"`
	Mileage      int    `json:"mileage"`
	// OwnerID optional: solo admin/manager asigna el cliente dueño.
	OwnerID string `json:"ownerID"`
}

// UpdateCarRequest represents the request payload for updating a car
type UpdateCarRequest struct {
	Make         string `json:"make"`
	Model        string `json:"model"`
	Year         int    `json:"year"`
	LicensePlate string `json:"licensePlate"` // ✅ camelCase
	VIN          string `json:"vin"`
	Color        string `json:"color"`
	Mileage      int    `json:"mileage"`
}

// CarResponse represents the response payload for a car
type CarResponse struct {
	ID           string `json:"id"`
	Make         string `json:"make"`
	Model        string `json:"model"`
	Year         int    `json:"year"`
	LicensePlate string `json:"licensePlate"` // ✅ camelCase
	VIN          string `json:"vin"`
	Color        string `json:"color"`
	Mileage      int    `json:"mileage"`
	OwnerID      string `json:"ownerID"`   // ✅ camelCase
	CreatedAt    string `json:"createdAt"` // ✅ camelCase
	UpdatedAt    string `json:"updatedAt"` // ✅ camelCase
}

// CreateCar registra un coche (cliente: dueño automático; admin/manager: ownerID obligatorio).
// @Summary     Crear coche
// @Tags        cars
// @Security    BearerAuth
// @Accept      json
// @Produce     json
// @Param       body body CreateCarRequest true "Datos del vehículo"
// @Success     201 {object} CarResponse
// @Failure     400 {object} SwaggerMessage
// @Failure     401 {object} SwaggerMessage
// @Failure     403 {object} SwaggerMessage
// @Failure     409 {object} SwaggerMessage
// @Router      /api/v1/cars [post]
func (h *CarHandler) CreateCar(c *gin.Context) {
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

	// Parse request
	var req CreateCarRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	ownerID := userID
	if roleVal, ok := c.Get("userRole"); ok {
		if roleStr, ok := roleVal.(string); ok && roleStr != domain.RoleClient && req.OwnerID != "" {
			parsed, perr := uuid.Parse(req.OwnerID)
			if perr != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ownerID"})
				return
			}
			ownerID = parsed
		}
	}

	car := &domain.Car{
		OwnerID:      ownerID,
		Make:         req.Make,
		Model:        req.Model,
		Year:         req.Year,
		LicensePlate: req.LicensePlate,
		VIN:          req.VIN,
		Color:        req.Color,
		Mileage:      req.Mileage,
	}

	// Create car (service will validate permissions)
	createdCar, err := h.carService.CreateCar(c.Request.Context(), car, userID)
	if err != nil {
		if errors.Is(err, domain.ErrUnauthorizedAccess) {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		if errors.Is(err, domain.ErrCarAlreadyExists) {
			c.JSON(http.StatusConflict, gin.H{"error": "car with this license plate already exists"})
			return
		}
		if errors.Is(err, domain.ErrCarOwnerRequiredForStaff) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Escolha um cliente existente ou crie um novo cliente antes de guardar o automóvel."})
			return
		}
		if errors.Is(err, domain.ErrCarOwnerNotFound) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "O cliente selecionado não existe."})
			return
		}
		if errors.Is(err, domain.ErrCarOwnerMustBeClient) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "O proprietário do automóvel tem de ser um cliente."})
			return
		}
		if errors.Is(err, domain.ErrInvalidCarData) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid car data"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	// Convert to response
	response := h.toCarResponse(createdCar)
	c.JSON(http.StatusCreated, response)
}

// GetCar obtiene un coche por ID (cliente: solo propios; taller: cualquiera).
// @Summary     Obtener coche
// @Tags        cars
// @Security    BearerAuth
// @Produce     json
// @Param       id path string true "UUID del coche"
// @Success     200 {object} CarResponse
// @Failure     400 {object} SwaggerMessage
// @Failure     401 {object} SwaggerMessage
// @Failure     403 {object} SwaggerMessage
// @Failure     404 {object} SwaggerMessage
// @Router      /api/v1/cars/{id} [get]
func (h *CarHandler) GetCar(c *gin.Context) {
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

	// Parse car ID from URL parameter
	carIDStr := c.Param("id")
	carID, err := uuid.Parse(carIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid car ID"})
		return
	}

	car, err := h.carService.GetCar(c.Request.Context(), carID, userID)
	if err != nil {
		if errors.Is(err, domain.ErrCarNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "car not found"})
			return
		}
		if errors.Is(err, domain.ErrUnauthorizedAccess) {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	response := h.toCarResponse(car)
	c.JSON(http.StatusOK, response)
}

// ListCars lista coches del cliente o inventario/por dueño para personal del taller.
// @Summary     Listar coches
// @Tags        cars
// @Security    BearerAuth
// @Produce     json
// @Param       ownerId query string false "UUID del cliente dueño (solo staff)"
// @Param       limit query int false "Límite (staff sin ownerId; default 50)"
// @Param       offset query int false "Offset paginación"
// @Success     200 {array} CarResponse
// @Failure     400 {object} SwaggerMessage
// @Failure     401 {object} SwaggerMessage
// @Failure     403 {object} SwaggerMessage
// @Router      /api/v1/cars [get]
func (h *CarHandler) ListCars(c *gin.Context) {
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

	var cars []*domain.Car
	if roleVal, ok := c.Get("userRole"); ok {
		if roleStr, ok := roleVal.(string); ok && roleStr != domain.RoleClient {
			var ownerFilter *uuid.UUID
			if oid := c.Query("ownerId"); oid != "" {
				parsed, perr := uuid.Parse(oid)
				if perr != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ownerId"})
					return
				}
				ownerFilter = &parsed
			}
			limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
			offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
			cars, err = h.carService.ListCars(c.Request.Context(), userID, ownerFilter, limit, offset)
		} else {
			cars, err = h.carService.GetCarsByOwner(c.Request.Context(), userID, userID)
		}
	} else {
		cars, err = h.carService.GetCarsByOwner(c.Request.Context(), userID, userID)
	}
	if err != nil {
		if errors.Is(err, domain.ErrUnauthorizedAccess) {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	// Convert to response
	responses := make([]CarResponse, len(cars))
	for i, car := range cars {
		responses[i] = h.toCarResponse(car)
	}

	c.JSON(http.StatusOK, responses)
}

// UpdateCar actualiza un coche.
// @Summary     Actualizar coche
// @Tags        cars
// @Security    BearerAuth
// @Accept      json
// @Produce     json
// @Param       id path string true "UUID del coche"
// @Param       body body UpdateCarRequest true "Campos a actualizar"
// @Success     200 {object} CarResponse
// @Failure     400 {object} SwaggerMessage
// @Failure     401 {object} SwaggerMessage
// @Failure     403 {object} SwaggerMessage
// @Failure     404 {object} SwaggerMessage
// @Router      /api/v1/cars/{id} [put]
func (h *CarHandler) UpdateCar(c *gin.Context) {
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

	// Parse car ID from URL parameter
	carIDStr := c.Param("id")
	carID, err := uuid.Parse(carIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid car ID"})
		return
	}

	// Parse request
	var req UpdateCarRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	// Convert to domain object
	car := &domain.Car{
		ID:           carID,
		Make:         req.Make,
		Model:        req.Model,
		Year:         req.Year,
		LicensePlate: req.LicensePlate,
		VIN:          req.VIN,
		Color:        req.Color,
		Mileage:      req.Mileage,
	}

	// Update car
	updatedCar, err := h.carService.UpdateCar(c.Request.Context(), car, userID)
	if err != nil {
		if errors.Is(err, domain.ErrCarNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "car not found"})
			return
		}
		if errors.Is(err, domain.ErrUnauthorizedAccess) {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		if errors.Is(err, domain.ErrInvalidCarData) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid car data"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	// Convert to response
	response := h.toCarResponse(updatedCar)

	c.JSON(http.StatusOK, response)
}

// DeleteCar elimina un coche (baja lógica según repositorio).
// @Summary     Eliminar coche
// @Tags        cars
// @Security    BearerAuth
// @Param       id path string true "UUID del coche"
// @Success     204 "Sin cuerpo"
// @Failure     400 {object} SwaggerMessage
// @Failure     401 {object} SwaggerMessage
// @Failure     403 {object} SwaggerMessage
// @Failure     404 {object} SwaggerMessage
// @Router      /api/v1/cars/{id} [delete]
func (h *CarHandler) DeleteCar(c *gin.Context) {
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

	// Parse car ID from URL parameter
	carIDStr := c.Param("id")
	carID, err := uuid.Parse(carIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid car ID"})
		return
	}

	// Delete car
	err = h.carService.DeleteCar(c.Request.Context(), carID, userID)
	if err != nil {
		if errors.Is(err, domain.ErrCarNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "car not found"})
			return
		}
		if errors.Is(err, domain.ErrUnauthorizedAccess) {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.Status(http.StatusNoContent)
}

// Helper methods

func (h *CarHandler) toCarResponse(car *domain.Car) CarResponse {
	return CarResponse{
		ID:           car.ID.String(),
		Make:         car.Make,
		Model:        car.Model,
		Year:         car.Year,
		LicensePlate: car.LicensePlate,
		VIN:          car.VIN,
		Color:        car.Color,
		Mileage:      car.Mileage,
		OwnerID:      car.OwnerID.String(),
		CreatedAt:    car.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:    car.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
