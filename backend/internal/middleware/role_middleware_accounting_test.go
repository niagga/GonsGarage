package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gaston-garcia-cegid/gonsgarage/internal/domain"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestRequireAccountingAccess_ClientForbidden(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)
	secret := "acc-mgr-secret-client"
	am := NewAuthMiddleware(secret)
	uid := uuid.New()

	r := gin.New()
	api := r.Group("/api/v1")
	api.Use(GinBearerJWT(am))
	api.Use(RequireAccountingAccess())
	api.GET("/billing-documents", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })

	req := httptest.NewRequest(http.MethodGet, "/api/v1/billing-documents", nil)
	req.Header.Set("Authorization", "Bearer "+testJWTWorkshop(t, secret, uid, domain.RoleClient))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestRequireAccountingAccess_EmployeeForbidden(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)
	secret := "acc-mgr-secret-emp"
	am := NewAuthMiddleware(secret)
	uid := uuid.New()

	r := gin.New()
	api := r.Group("/api/v1")
	api.Use(GinBearerJWT(am))
	api.Use(RequireAccountingAccess())
	api.GET("/billing-documents", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })

	req := httptest.NewRequest(http.MethodGet, "/api/v1/billing-documents", nil)
	req.Header.Set("Authorization", "Bearer "+testJWTWorkshop(t, secret, uid, domain.RoleEmployee))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestRequireAccountingAccess_ManagerOK(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)
	secret := "acc-mgr-secret-manager"
	am := NewAuthMiddleware(secret)
	uid := uuid.New()

	r := gin.New()
	api := r.Group("/api/v1")
	api.Use(GinBearerJWT(am))
	api.Use(RequireAccountingAccess())
	api.GET("/billing-documents", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })

	req := httptest.NewRequest(http.MethodGet, "/api/v1/billing-documents", nil)
	req.Header.Set("Authorization", "Bearer "+testJWTWorkshop(t, secret, uid, domain.RoleManager))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRequireAccountingAccess_AdminOK(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)
	secret := "acc-mgr-secret-admin"
	am := NewAuthMiddleware(secret)
	uid := uuid.New()

	r := gin.New()
	api := r.Group("/api/v1")
	api.Use(GinBearerJWT(am))
	api.Use(RequireAccountingAccess())
	api.GET("/billing-documents", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })

	req := httptest.NewRequest(http.MethodGet, "/api/v1/billing-documents", nil)
	req.Header.Set("Authorization", "Bearer "+testJWTWorkshop(t, secret, uid, domain.RoleAdmin))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
