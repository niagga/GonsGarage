package handler

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/gaston-garcia-cegid/gonsgarage/internal/domain"
	"github.com/gaston-garcia-cegid/gonsgarage/internal/service/servicejob"
)

// ServiceJobHandler exposes workshop service-job HTTP routes.
type ServiceJobHandler struct {
	svc *servicejob.Service
}

func NewServiceJobHandler(svc *servicejob.Service) *ServiceJobHandler {
	return &ServiceJobHandler{svc: svc}
}

type createServiceJobJSON struct {
	CarID string `json:"car_id" binding:"required"`
}

type serviceJobDetailResponse struct {
	Job        domain.ServiceJob            `json:"job"`
	Reception  *domain.ServiceJobReception `json:"reception,omitempty"`
	Handover   *domain.ServiceJobHandover  `json:"handover,omitempty"`
	RepairIDs  []uuid.UUID                   `json:"repair_ids"`
}

type putReceptionJSON struct {
	OdometerKM   *int   `json:"odometer_km"`
	OilLevel     string `json:"oil_level"`
	CoolantLevel string `json:"coolant_level"`
	TiresNote    string `json:"tires_note"`
	GeneralNotes string `json:"general_notes"`
}

type putHandoverJSON struct {
	OdometerKM   int    `json:"odometer_km"`
	TiresNote    string `json:"tires_note"`
	GeneralNotes string `json:"general_notes"`
}

// CreateServiceJob POST /api/v1/service-jobs
// @Summary     Abrir visita (service job)
// @Tags        service-jobs
// @Security    BearerAuth
// @Accept      json
// @Param       body body createServiceJobJSON true "car_id"
// @Success     201 {object} domain.ServiceJob
// @Failure     400,401,403,500
// @Router      /api/v1/service-jobs [post]
func (h *ServiceJobHandler) CreateServiceJob(c *gin.Context) {
	uid, ok := parseGinUserID(c)
	if !ok {
		return
	}
	var req createServiceJobJSON
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	carID, err := uuid.Parse(strings.TrimSpace(req.CarID))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid car_id"})
		return
	}
	job, err := h.svc.CreateServiceJob(c.Request.Context(), carID, uid)
	if err != nil {
		if err == domain.ErrUnauthorizedAccess {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		if err == domain.ErrCarNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "car not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, job)
}

// GetServiceJob GET /api/v1/service-jobs/:id
// @Tags        service-jobs
// @Param       id path string true "UUID service job"
// @Success     200 {object} serviceJobDetailResponse
// @Router      /api/v1/service-jobs/{id} [get]
func (h *ServiceJobHandler) GetServiceJob(c *gin.Context) {
	uid, ok := parseGinUserID(c)
	if !ok {
		return
	}
	jid, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	job, rec, ho, repIDs, err := h.svc.GetWithDetails(c.Request.Context(), jid, uid)
	if err != nil {
		if err == domain.ErrServiceJobNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "service job not found"})
			return
		}
		if err == domain.ErrUnauthorizedAccess {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		if err == domain.ErrCarNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "car not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, serviceJobDetailResponse{Job: *job, Reception: rec, Handover: ho, RepairIDs: repIDs})
}

// ListServiceJobsByOpenedOn GET /api/v1/service-jobs?opened_on=YYYY-MM-DD
// Visits with OpenedAt in [date 00:00 UTC, next day 00:00 UTC).
// @Param       opened_on query string true "Calendar day in UTC (YYYY-MM-DD)"
// @Success     200 {array} domain.ServiceJob
// @Router      /api/v1/service-jobs [get]
func (h *ServiceJobHandler) ListServiceJobsByOpenedOn(c *gin.Context) {
	uid, ok := parseGinUserID(c)
	if !ok {
		return
	}
	q := strings.TrimSpace(c.Query("opened_on"))
	if q == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "opened_on is required (YYYY-MM-DD, UTC day window)"})
		return
	}
	day, err := time.Parse("2006-01-02", q)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid opened_on"})
		return
	}
	list, err := h.svc.ListOpenedOn(c.Request.Context(), day, uid)
	if err != nil {
		if err == domain.ErrUnauthorizedAccess {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if list == nil {
		list = []*domain.ServiceJob{}
	}
	c.JSON(http.StatusOK, list)
}

// ListServiceJobsByCar GET /api/v1/service-jobs/car/:carId
// @Param       carId path string true "UUID coche"
// @Success     200 {array} domain.ServiceJob
// @Router      /api/v1/service-jobs/car/{carId} [get]
func (h *ServiceJobHandler) ListServiceJobsByCar(c *gin.Context) {
	uid, ok := parseGinUserID(c)
	if !ok {
		return
	}
	carID, err := uuid.Parse(c.Param("carId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid carId"})
		return
	}
	list, err := h.svc.ListByCarID(c.Request.Context(), carID, uid)
	if err != nil {
		if err == domain.ErrUnauthorizedAccess {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		if err == domain.ErrCarNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "car not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if list == nil {
		list = []*domain.ServiceJob{}
	}
	c.JSON(http.StatusOK, list)
}

// PutReception PUT /api/v1/service-jobs/:id/reception
// @Router      /api/v1/service-jobs/{id}/reception [put]
func (h *ServiceJobHandler) PutReception(c *gin.Context) {
	uid, ok := parseGinUserID(c)
	if !ok {
		return
	}
	jid, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var body putReceptionJSON
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	if body.OdometerKM == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "odometer_km is required"})
		return
	}
	out, err := h.svc.SaveReception(c.Request.Context(), jid, servicejob.SaveReceptionInput{
		OdometerKM:   *body.OdometerKM,
		OilLevel:     strings.TrimSpace(body.OilLevel),
		CoolantLevel: strings.TrimSpace(body.CoolantLevel),
		TiresNote:    strings.TrimSpace(body.TiresNote),
		GeneralNotes: strings.TrimSpace(body.GeneralNotes),
	}, uid)
	if err != nil {
		if err == domain.ErrUnauthorizedAccess {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		if err == domain.ErrServiceJobNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "service job not found"})
			return
		}
		if err == domain.ErrInvalidServiceJobData {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid reception data"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, out)
}

// PutHandover PUT /api/v1/service-jobs/:id/handover
// @Router      /api/v1/service-jobs/{id}/handover [put]
func (h *ServiceJobHandler) PutHandover(c *gin.Context) {
	uid, ok := parseGinUserID(c)
	if !ok {
		return
	}
	jid, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var body putHandoverJSON
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	out, err := h.svc.SaveHandover(c.Request.Context(), jid, servicejob.SaveHandoverInput{
		OdometerKM:   body.OdometerKM,
		TiresNote:    strings.TrimSpace(body.TiresNote),
		GeneralNotes: strings.TrimSpace(body.GeneralNotes),
	}, uid)
	if err != nil {
		if err == domain.ErrUnauthorizedAccess {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		if err == domain.ErrServiceJobNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "service job not found"})
			return
		}
		if err == domain.ErrReceptionRequiredBeforeHandover {
			c.JSON(http.StatusBadRequest, gin.H{"error": "reception required before handover"})
			return
		}
		if err == domain.ErrInvalidServiceJobData {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid handover data"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, out)
}

// StubOBD GET /api/v1/service-jobs/:id/obd — OBD not implemented in MVP1 (spec stub).
// @Router      /api/v1/service-jobs/{id}/obd [get]
func (h *ServiceJobHandler) StubOBD(c *gin.Context) {
	_, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	c.JSON(http.StatusNotImplemented, gin.H{"error": "OBD not implemented", "code": "obd_not_implemented"})
}

func parseGinUserID(c *gin.Context) (uuid.UUID, bool) {
	userIDStr, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return uuid.UUID{}, false
	}
	uid, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user ID"})
		return uuid.UUID{}, false
	}
	return uid, true
}
