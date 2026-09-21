package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gaston-garcia-cegid/gonsgarage/internal/core/ports"
	"github.com/gaston-garcia-cegid/gonsgarage/internal/domain"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AdminUserHandler struct {
	authService ports.AuthService
	userRepo    ports.UserRepository
}

func NewAdminUserHandler(authService ports.AuthService, userRepo ports.UserRepository) *AdminUserHandler {
	return &AdminUserHandler{authService: authService, userRepo: userRepo}
}

// ProvisionUser creates a user (roles manager, employee, or client only). Requires JWT; only admin and manager reach this handler.
// @Summary     Aprovisionar utilizador (staff)
// @Description Cria utilizador com papel manager, employee ou client. Admin pode todos; manager não pode criar manager. Nunca cria admin por este fluxo.
// @Tags        admin
// @Security    BearerAuth
// @Accept      json
// @Produce     json
// @Param       body body ports.ProvisionUserRequest true "Dados do novo utilizador"
// @Success     201 {object} SwaggerProvisionUserOK
// @Failure     400 {object} SwaggerMessage
// @Failure     403 {object} SwaggerMessage
// @Failure     409 {object} SwaggerMessage
// @Failure     500 {object} SwaggerMessage
// @Router      /api/v1/admin/users [post]
func (h *AdminUserHandler) ProvisionUser(c *gin.Context) {
	userIDStr, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	uidStr, ok := userIDStr.(string)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user id"})
		return
	}
	callerID, err := uuid.Parse(uidStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user id"})
		return
	}

	roleVal, ok := c.Get("userRole")
	if !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	callerRole, _ := roleVal.(string)

	var req ports.ProvisionUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	user, err := h.authService.ProvisionUser(c.Request.Context(), callerID, callerRole, req)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrUserAlreadyExists):
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		case errors.Is(err, domain.ErrPermissionDenied):
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		case errors.Is(err, domain.ErrInvalidRole):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{"user": user})
}

// ListClients returns client users so staff can associate car ownership during car creation.
// @Summary     Listar clientes (staff)
// @Description Lista utilizadores com papel client para associação de viaturas.
// @Tags        admin
// @Security    BearerAuth
// @Produce     json
// @Param       q query string false "Filtro por nome ou email"
// @Param       limit query int false "Límite (default 100, max 200)"
// @Param       offset query int false "Offset"
// @Success     200 {object} map[string]interface{}
// @Failure     403 {object} SwaggerMessage
// @Failure     500 {object} SwaggerMessage
// @Router      /api/v1/admin/users/clients [get]
func (h *AdminUserHandler) ListClients(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if limit <= 0 {
		limit = 100
	}
	if limit > 200 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}

	users, err := h.userRepo.GetByRole(c.Request.Context(), domain.RoleClient, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list clients"})
		return
	}

	q := strings.TrimSpace(strings.ToLower(c.Query("q")))
	items := make([]gin.H, 0, len(users))
	for _, u := range users {
		if u == nil {
			continue
		}
		if q != "" {
			fullName := strings.ToLower(strings.TrimSpace(u.FirstName + " " + u.LastName))
			email := strings.ToLower(strings.TrimSpace(u.Email))
			if !strings.Contains(fullName, q) && !strings.Contains(email, q) {
				continue
			}
		}
		items = append(items, gin.H{
			"id":        u.ID,
			"email":     u.Email,
			"firstName": u.FirstName,
			"lastName":  u.LastName,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"items": items,
		"total": len(items),
	})
}
