package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gaston-garcia-cegid/gonsgarage/internal/core/ports"
	"github.com/gaston-garcia-cegid/gonsgarage/internal/domain"
	"github.com/gaston-garcia-cegid/gonsgarage/internal/middleware"
	repopg "github.com/gaston-garcia-cegid/gonsgarage/internal/repository/postgres"
	partsvc "github.com/gaston-garcia-cegid/gonsgarage/internal/service/part"
	repairsvc "github.com/gaston-garcia-cegid/gonsgarage/internal/service/repair"
	"github.com/gaston-garcia-cegid/gonsgarage/internal/service/servicejob"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func testJWT(t *testing.T, secret string, userID uuid.UUID, role string) string {
	t.Helper()
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"userID": userID.String(),
		"role":   role,
		"email":  role + "@test.local",
	})
	s, err := tok.SignedString([]byte(secret))
	require.NoError(t, err)
	return s
}

// --- Employees: RequireStaffManagers (spec client + employee forbidden; manager allowed) ---

func TestMVPAccess_EmployeesGET_ClientForbidden(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)
	secret := "mvp-role-access-secret"
	am := middleware.NewAuthMiddleware(secret)
	uid := uuid.New()

	r := gin.New()
	api := r.Group("/api/v1")
	api.Use(middleware.GinBearerJWT(am))
	employees := api.Group("/employees")
	employees.Use(middleware.RequireStaffManagers())
	employees.GET("", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/employees", nil)
	req.Header.Set("Authorization", "Bearer "+testJWT(t, secret, uid, domain.RoleClient))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestMVPAccess_EmployeesGET_EmployeeForbidden(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)
	secret := "mvp-role-access-secret-emp"
	am := middleware.NewAuthMiddleware(secret)
	uid := uuid.New()

	r := gin.New()
	api := r.Group("/api/v1")
	api.Use(middleware.GinBearerJWT(am))
	employees := api.Group("/employees")
	employees.Use(middleware.RequireStaffManagers())
	employees.GET("", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/employees", nil)
	req.Header.Set("Authorization", "Bearer "+testJWT(t, secret, uid, domain.RoleEmployee))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestMVPAccess_EmployeesGET_ManagerReachesHandler(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)
	secret := "mvp-role-access-secret-mgr"
	am := middleware.NewAuthMiddleware(secret)
	uid := uuid.New()

	r := gin.New()
	api := r.Group("/api/v1")
	api.Use(middleware.GinBearerJWT(am))
	employees := api.Group("/employees")
	employees.Use(middleware.RequireStaffManagers())
	employees.GET("", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/employees", nil)
	req.Header.Set("Authorization", "Bearer "+testJWT(t, secret, uid, domain.RoleManager))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "ok")
}

func TestMVPAccess_EmployeesGET_AdminReachesHandler(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)
	secret := "mvp-role-access-secret-adm"
	am := middleware.NewAuthMiddleware(secret)
	uid := uuid.New()

	r := gin.New()
	api := r.Group("/api/v1")
	api.Use(middleware.GinBearerJWT(am))
	employees := api.Group("/employees")
	employees.Use(middleware.RequireStaffManagers())
	employees.GET("", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/employees", nil)
	req.Header.Set("Authorization", "Bearer "+testJWT(t, secret, uid, domain.RoleAdmin))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "ok")
}

// --- Parts inventory: RequireStaffManagers (spec: client + employee forbidden; manager + admin allowed) ---

func partsStaffStubRouter(secret string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	am := middleware.NewAuthMiddleware(secret)
	r := gin.New()
	api := r.Group("/api/v1")
	api.Use(middleware.GinBearerJWT(am))
	parts := api.Group("/parts")
	parts.Use(middleware.RequireStaffManagers())
	{
		parts.GET("", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"items": []any{}, "total": int64(0)})
		})
		parts.POST("", func(c *gin.Context) {
			c.JSON(http.StatusCreated, gin.H{"ok": true})
		})
	}
	return r
}

func TestMVPAccess_PartsGET_ClientForbidden(t *testing.T) {
	t.Parallel()
	secret := "mvp-parts-get-client"
	uid := uuid.New()
	r := partsStaffStubRouter(secret)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/parts", nil)
	req.Header.Set("Authorization", "Bearer "+testJWT(t, secret, uid, domain.RoleClient))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestMVPAccess_PartsGET_EmployeeForbidden(t *testing.T) {
	t.Parallel()
	secret := "mvp-parts-get-emp"
	uid := uuid.New()
	r := partsStaffStubRouter(secret)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/parts", nil)
	req.Header.Set("Authorization", "Bearer "+testJWT(t, secret, uid, domain.RoleEmployee))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestMVPAccess_PartsGET_ManagerReachesHandler(t *testing.T) {
	t.Parallel()
	secret := "mvp-parts-get-mgr"
	uid := uuid.New()
	r := partsStaffStubRouter(secret)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/parts", nil)
	req.Header.Set("Authorization", "Bearer "+testJWT(t, secret, uid, domain.RoleManager))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestMVPAccess_PartsGET_AdminReachesHandler(t *testing.T) {
	t.Parallel()
	secret := "mvp-parts-get-adm"
	uid := uuid.New()
	r := partsStaffStubRouter(secret)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/parts", nil)
	req.Header.Set("Authorization", "Bearer "+testJWT(t, secret, uid, domain.RoleAdmin))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestMVPAccess_PartsPOST_ClientForbidden(t *testing.T) {
	t.Parallel()
	secret := "mvp-parts-post-client"
	uid := uuid.New()
	r := partsStaffStubRouter(secret)
	body := []byte(`{"reference":"r","brand":"b","name":"n","barcode":"x","quantity":1,"uom":"unit"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/parts", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+testJWT(t, secret, uid, domain.RoleClient))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestMVPAccess_PartsPOST_EmployeeForbidden(t *testing.T) {
	t.Parallel()
	secret := "mvp-parts-post-emp"
	uid := uuid.New()
	r := partsStaffStubRouter(secret)
	body := []byte(`{"reference":"r","brand":"b","name":"n","barcode":"x","quantity":1,"uom":"unit"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/parts", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+testJWT(t, secret, uid, domain.RoleEmployee))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestMVPAccess_PartsPOST_ManagerReachesHandler(t *testing.T) {
	t.Parallel()
	secret := "mvp-parts-post-mgr"
	uid := uuid.New()
	r := partsStaffStubRouter(secret)
	body := []byte(`{"reference":"r","brand":"b","name":"n","barcode":"x","quantity":1,"uom":"unit"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/parts", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+testJWT(t, secret, uid, domain.RoleManager))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)
}

func partsRouterWithPartHandler(t *testing.T, secret string, h *PartHandler) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	am := middleware.NewAuthMiddleware(secret)
	r := gin.New()
	api := r.Group("/api/v1")
	api.Use(middleware.GinBearerJWT(am))
	p := api.Group("/parts")
	p.Use(middleware.RequireStaffManagers())
	{
		p.GET("", h.ListParts)
		p.POST("", h.CreatePartItem)
		p.GET("/:id", h.GetPartItem)
		p.PATCH("/:id", h.UpdatePartItem)
		p.DELETE("/:id", h.DeletePartItem)
	}
	return r
}

func TestMVPAccess_PartsGET_ManagerRealHandlerEmptyList(t *testing.T) {
	t.Parallel()
	secret := "mvp-parts-real-list"
	mgrID := uuid.New()
	mgr, err := domain.NewUser("mgr-parts-list@test", "x", "M", "L", domain.RoleManager)
	require.NoError(t, err)
	mgr.ID = mgrID
	ur := &mvpUserRepo{byID: map[uuid.UUID]*domain.User{mgrID: mgr}}

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&domain.PartItem{}))
	t.Cleanup(func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			_ = sqlDB.Close()
		}
	})
	repo := repopg.NewPostgresPartItemRepository(db)
	svc := partsvc.NewPartService(repo, ur)
	h := NewPartHandler(svc)
	r := partsRouterWithPartHandler(t, secret, h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/parts", nil)
	req.Header.Set("Authorization", "Bearer "+testJWT(t, secret, mgrID, domain.RoleManager))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var payload struct {
		Items []json.RawMessage `json:"items"`
		Total int64             `json:"total"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &payload))
	assert.Equal(t, int64(0), payload.Total)
	assert.Len(t, payload.Items, 0)
}

func TestMVPAccess_PartsPOST_ManagerRealHandlerCreated(t *testing.T) {
	t.Parallel()
	secret := "mvp-parts-real-create"
	mgrID := uuid.New()
	mgr, err := domain.NewUser("mgr-parts-post@test", "x", "M", "P", domain.RoleManager)
	require.NoError(t, err)
	mgr.ID = mgrID
	ur := &mvpUserRepo{byID: map[uuid.UUID]*domain.User{mgrID: mgr}}

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&domain.PartItem{}))
	t.Cleanup(func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			_ = sqlDB.Close()
		}
	})
	repo := repopg.NewPostgresPartItemRepository(db)
	svc := partsvc.NewPartService(repo, ur)
	h := NewPartHandler(svc)
	r := partsRouterWithPartHandler(t, secret, h)

	body := map[string]any{
		"reference": "R1", "brand": "B", "name": "Filter", "barcode": "5900000000999",
		"quantity": 3.0, "uom": "unit",
	}
	b, err := json.Marshal(body)
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/parts", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+testJWT(t, secret, mgrID, domain.RoleManager))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	var created PartItemResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created))
	assert.Equal(t, "R1", created.Reference)
	assert.Equal(t, "5900000000999", created.Barcode)
}

func TestMVPAccess_PartsPOST_ClientForbidden_RealHandler(t *testing.T) {
	t.Parallel()
	secret := "mvp-parts-real-client-post"
	clientID := uuid.New()
	cl, err := domain.NewUser("cl-parts@test", "x", "C", "L", domain.RoleClient)
	require.NoError(t, err)
	cl.ID = clientID
	ur := &mvpUserRepo{byID: map[uuid.UUID]*domain.User{clientID: cl}}

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&domain.PartItem{}))
	t.Cleanup(func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			_ = sqlDB.Close()
		}
	})
	repo := repopg.NewPostgresPartItemRepository(db)
	svc := partsvc.NewPartService(repo, ur)
	h := NewPartHandler(svc)
	r := partsRouterWithPartHandler(t, secret, h)

	body := map[string]any{
		"reference": "R1", "brand": "B", "name": "N", "barcode": "x", "quantity": 1.0, "uom": "unit",
	}
	b, err := json.Marshal(body)
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/parts", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+testJWT(t, secret, clientID, domain.RoleClient))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestMVPAccess_PartsPOST_EmployeeForbidden_RealHandler(t *testing.T) {
	t.Parallel()
	secret := "mvp-parts-real-emp-post"
	empID := uuid.New()
	emp, err := domain.NewUser("emp-parts@test", "x", "E", "M", domain.RoleEmployee)
	require.NoError(t, err)
	emp.ID = empID
	ur := &mvpUserRepo{byID: map[uuid.UUID]*domain.User{empID: emp}}

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&domain.PartItem{}))
	t.Cleanup(func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			_ = sqlDB.Close()
		}
	})
	repo := repopg.NewPostgresPartItemRepository(db)
	svc := partsvc.NewPartService(repo, ur)
	h := NewPartHandler(svc)
	r := partsRouterWithPartHandler(t, secret, h)

	body := map[string]any{
		"reference": "R1", "brand": "B", "name": "N", "barcode": "x", "quantity": 1.0, "uom": "unit",
	}
	b, err := json.Marshal(body)
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/parts", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+testJWT(t, secret, empID, domain.RoleEmployee))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

// --- Repairs POST: client 403; employee 201 with stubs ---

type mvpUserRepo struct {
	byID map[uuid.UUID]*domain.User
}

func (m *mvpUserRepo) Create(context.Context, *domain.User) error {
	return errors.New("mvpUserRepo.Create not used")
}
func (m *mvpUserRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.User, error) {
	u, ok := m.byID[id]
	if !ok {
		return nil, domain.ErrUserNotFound
	}
	return u, nil
}
func (m *mvpUserRepo) GetByEmail(context.Context, string) (*domain.User, error) {
	return nil, domain.ErrUserNotFound
}
func (m *mvpUserRepo) GetByRole(context.Context, string, int, int) ([]*domain.User, error) {
	return nil, nil
}
func (m *mvpUserRepo) List(context.Context, int, int) ([]*domain.User, error) { return nil, nil }
func (m *mvpUserRepo) Update(context.Context, *domain.User) error             { return errors.New("not used") }
func (m *mvpUserRepo) Delete(context.Context, uuid.UUID) error                { return errors.New("not used") }
func (m *mvpUserRepo) UpdatePassword(context.Context, uuid.UUID, string) error {
	return errors.New("not used")
}
func (m *mvpUserRepo) GetActiveUsers(context.Context, int, int) ([]*domain.User, error) {
	return nil, nil
}

type mvpCarRepo struct {
	car *domain.Car
}

func (m *mvpCarRepo) Create(context.Context, *domain.Car) error { return errors.New("not used") }
func (m *mvpCarRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.Car, error) {
	if m.car != nil && m.car.ID == id {
		return m.car, nil
	}
	return nil, fmt.Errorf("car not found")
}
func (m *mvpCarRepo) GetByOwnerID(context.Context, uuid.UUID) ([]*domain.Car, error) {
	return nil, nil
}
func (m *mvpCarRepo) GetByLicensePlate(context.Context, string) (*domain.Car, error) {
	return nil, domain.ErrUserNotFound
}
func (m *mvpCarRepo) List(context.Context, int, int) ([]*domain.Car, error) { return nil, nil }
func (m *mvpCarRepo) Update(context.Context, *domain.Car) error             { return errors.New("not used") }
func (m *mvpCarRepo) Delete(context.Context, uuid.UUID) error               { return errors.New("not used") }
func (m *mvpCarRepo) GetWithRepairs(context.Context, uuid.UUID) (*domain.Car, error) {
	return nil, nil
}
func (m *mvpCarRepo) GetDeletedByLicensePlate(context.Context, string) (*domain.Car, error) {
	return nil, nil
}
func (m *mvpCarRepo) Restore(context.Context, uuid.UUID) error { return errors.New("not used") }

type mvpRepairRepo struct {
	byCar map[uuid.UUID][]*domain.Repair
}

func (m *mvpRepairRepo) Create(context.Context, *domain.Repair) error { return nil }
func (m *mvpRepairRepo) GetByID(context.Context, uuid.UUID) (*domain.Repair, error) {
	return nil, domain.ErrRepairNotFound
}
func (m *mvpRepairRepo) Update(context.Context, *domain.Repair) error { return errors.New("not used") }
func (m *mvpRepairRepo) Delete(context.Context, uuid.UUID) error      { return errors.New("not used") }
func (m *mvpRepairRepo) GetByCarID(_ context.Context, carID uuid.UUID) ([]*domain.Repair, error) {
	if m.byCar == nil {
		return nil, nil
	}
	return m.byCar[carID], nil
}

func (m *mvpRepairRepo) ListIDsByServiceJobID(_ context.Context, _ uuid.UUID) ([]uuid.UUID, error) {
	return nil, nil
}

var _ ports.UserRepository = (*mvpUserRepo)(nil)
var _ ports.CarRepository = (*mvpCarRepo)(nil)
var _ ports.RepairRepository = (*mvpRepairRepo)(nil)

// mvpSJRepo is a minimal in-memory ServiceJobRepository for auth / flow tests.
type mvpSJRepo struct {
	byID  map[uuid.UUID]*domain.ServiceJob
	byCar map[uuid.UUID][]*domain.ServiceJob
	rec   map[uuid.UUID]*domain.ServiceJobReception
	ho    map[uuid.UUID]*domain.ServiceJobHandover
}

func (m *mvpSJRepo) Create(_ context.Context, j *domain.ServiceJob) error {
	if m.byID == nil {
		m.byID = make(map[uuid.UUID]*domain.ServiceJob)
	}
	if m.byCar == nil {
		m.byCar = make(map[uuid.UUID][]*domain.ServiceJob)
	}
	m.byID[j.ID] = j
	m.byCar[j.CarID] = append(m.byCar[j.CarID], j)
	return nil
}

func (m *mvpSJRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.ServiceJob, error) {
	j, ok := m.byID[id]
	if !ok {
		return nil, domain.ErrServiceJobNotFound
	}
	return j, nil
}

func (m *mvpSJRepo) Update(_ context.Context, j *domain.ServiceJob) error {
	if _, ok := m.byID[j.ID]; !ok {
		return domain.ErrServiceJobNotFound
	}
	m.byID[j.ID] = j
	return nil
}

func (m *mvpSJRepo) ListByCarID(_ context.Context, carID uuid.UUID) ([]*domain.ServiceJob, error) {
	if m.byCar == nil {
		return nil, nil
	}
	return m.byCar[carID], nil
}

func (m *mvpSJRepo) ListByOpenedOn(_ context.Context, day time.Time) ([]*domain.ServiceJob, error) {
	if m.byID == nil {
		return []*domain.ServiceJob{}, nil
	}
	y, mth, d := day.UTC().Date()
	start := time.Date(y, mth, d, 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)
	var out []*domain.ServiceJob
	for _, j := range m.byID {
		if j.OpenedAt.Before(start) || !j.OpenedAt.Before(end) {
			continue
		}
		cj := *j
		out = append(out, &cj)
	}
	if out == nil {
		out = []*domain.ServiceJob{}
	}
	return out, nil
}

func (m *mvpSJRepo) SaveReception(_ context.Context, r *domain.ServiceJobReception) error {
	if m.rec == nil {
		m.rec = make(map[uuid.UUID]*domain.ServiceJobReception)
	}
	cp := *r
	m.rec[r.ServiceJobID] = &cp
	return nil
}

func (m *mvpSJRepo) GetReception(_ context.Context, id uuid.UUID) (*domain.ServiceJobReception, error) {
	if m.rec == nil {
		return nil, nil
	}
	return m.rec[id], nil
}

func (m *mvpSJRepo) SaveHandover(_ context.Context, h *domain.ServiceJobHandover) error {
	if m.ho == nil {
		m.ho = make(map[uuid.UUID]*domain.ServiceJobHandover)
	}
	cp := *h
	m.ho[h.ServiceJobID] = &cp
	return nil
}

func (m *mvpSJRepo) GetHandover(_ context.Context, id uuid.UUID) (*domain.ServiceJobHandover, error) {
	if m.ho == nil {
		return nil, nil
	}
	return m.ho[id], nil
}

var _ ports.ServiceJobRepository = (*mvpSJRepo)(nil)

func serviceJobWorkshopRouter(t *testing.T, secret string, userRepo ports.UserRepository, carRepo ports.CarRepository, jobRepo ports.ServiceJobRepository) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	am := middleware.NewAuthMiddleware(secret)
	rep := &mvpRepairRepo{}
	svc := servicejob.NewService(jobRepo, carRepo, userRepo, rep)
	h := NewServiceJobHandler(svc)

	r := gin.New()
	api := r.Group("/api/v1")
	api.Use(middleware.GinBearerJWT(am))
	sj := api.Group("/service-jobs")
	sj.Use(middleware.RequireWorkshopStaff())
	{
		sj.POST("", h.CreateServiceJob)
		sj.GET("", h.ListServiceJobsByOpenedOn)
		sj.GET("/car/:carId", h.ListServiceJobsByCar)
		sj.GET("/:id/obd", h.StubOBD)
		sj.GET("/:id", h.GetServiceJob)
		sj.PUT("/:id/reception", h.PutReception)
		sj.PUT("/:id/handover", h.PutHandover)
	}
	return r
}

func repairPOSTRouter(t *testing.T, secret string, userRepo ports.UserRepository, carRepo ports.CarRepository, repRepo ports.RepairRepository) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	am := middleware.NewAuthMiddleware(secret)
	svc := repairsvc.NewRepairService(repRepo, carRepo, userRepo)
	h := NewRepairHandler(svc)

	r := gin.New()
	api := r.Group("/api/v1")
	api.Use(middleware.GinBearerJWT(am))
	api.POST("/repairs", h.GinCreateRepair)
	return r
}

func TestMVPAccess_RepairPOST_ClientForbidden(t *testing.T) {
	t.Parallel()
	secret := "mvp-repair-client-secret"
	clientID := uuid.New()
	carID := uuid.New()

	u, err := domain.NewUser("c@test.local", "x", "C", "L", domain.RoleClient)
	require.NoError(t, err)
	u.ID = clientID

	users := &mvpUserRepo{byID: map[uuid.UUID]*domain.User{clientID: u}}
	cars := &mvpCarRepo{car: &domain.Car{ID: carID, OwnerID: clientID}}
	repairs := &mvpRepairRepo{byCar: map[uuid.UUID][]*domain.Repair{carID: {}}}

	r := repairPOSTRouter(t, secret, users, cars, repairs)

	body := map[string]any{
		"car_id":      carID.String(),
		"description": "Brake inspection",
		"status":      "pending",
		"cost":        0.0,
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/repairs", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+testJWT(t, secret, clientID, domain.RoleClient))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestMVPAccess_RepairPOST_EmployeeCreated(t *testing.T) {
	t.Parallel()
	secret := "mvp-repair-emp-secret"
	empID := uuid.New()
	carID := uuid.New()
	ownerID := uuid.New()

	emp, err := domain.NewUser("e@test.local", "x", "E", "M", domain.RoleEmployee)
	require.NoError(t, err)
	emp.ID = empID

	users := &mvpUserRepo{byID: map[uuid.UUID]*domain.User{empID: emp}}
	cars := &mvpCarRepo{car: &domain.Car{ID: carID, OwnerID: ownerID}}
	repairs := &mvpRepairRepo{byCar: map[uuid.UUID][]*domain.Repair{carID: {}}}

	r := repairPOSTRouter(t, secret, users, cars, repairs)

	body := map[string]any{
		"car_id":      carID.String(),
		"description": "Oil change",
		"status":      "pending",
		"cost":        10.0,
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/repairs", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+testJWT(t, secret, empID, domain.RoleEmployee))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
}

// --- Service jobs: client 403 (workshop staff only); employee 201 ---

func TestMVPAccess_ServiceJobPOST_ClientForbidden(t *testing.T) {
	t.Parallel()
	secret := "mvp-sj-client"
	clientID := uuid.New()
	carID := uuid.New()
	u, err := domain.NewUser("c2@test.local", "x", "C", "L", domain.RoleClient)
	require.NoError(t, err)
	u.ID = clientID
	users := &mvpUserRepo{byID: map[uuid.UUID]*domain.User{clientID: u}}
	cars := &mvpCarRepo{car: &domain.Car{ID: carID, OwnerID: clientID}}
	sj := &mvpSJRepo{byID: map[uuid.UUID]*domain.ServiceJob{}, byCar: map[uuid.UUID][]*domain.ServiceJob{carID: {}}}

	r := serviceJobWorkshopRouter(t, secret, users, cars, sj)
	body, _ := json.Marshal(map[string]any{"car_id": carID.String()})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/service-jobs", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+testJWT(t, secret, clientID, domain.RoleClient))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestMVPAccess_ServiceJobPOST_EmployeeCreated(t *testing.T) {
	t.Parallel()
	secret := "mvp-sj-emp"
	empID := uuid.New()
	carID := uuid.New()
	ownerID := uuid.New()
	emp, err := domain.NewUser("e2@test.local", "x", "E", "M", domain.RoleEmployee)
	require.NoError(t, err)
	emp.ID = empID
	users := &mvpUserRepo{byID: map[uuid.UUID]*domain.User{empID: emp}}
	cars := &mvpCarRepo{car: &domain.Car{ID: carID, OwnerID: ownerID}}
	sj := &mvpSJRepo{byID: map[uuid.UUID]*domain.ServiceJob{}, byCar: map[uuid.UUID][]*domain.ServiceJob{carID: {}}}

	r := serviceJobWorkshopRouter(t, secret, users, cars, sj)
	body, _ := json.Marshal(map[string]any{"car_id": carID.String()})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/service-jobs", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+testJWT(t, secret, empID, domain.RoleEmployee))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestMVPAccess_ServiceJobGET_ByOpenedOn_MissingParam_BadRequest(t *testing.T) {
	t.Parallel()
	secret := "mvp-sj-list-bad"
	empID := uuid.New()
	carID := uuid.New()
	emp, err := domain.NewUser("e-list-bad@test.local", "x", "E", "M", domain.RoleEmployee)
	require.NoError(t, err)
	emp.ID = empID
	users := &mvpUserRepo{byID: map[uuid.UUID]*domain.User{empID: emp}}
	cars := &mvpCarRepo{car: &domain.Car{ID: carID, OwnerID: uuid.New()}}
	sj := &mvpSJRepo{byID: map[uuid.UUID]*domain.ServiceJob{}, byCar: map[uuid.UUID][]*domain.ServiceJob{carID: {}}}

	r := serviceJobWorkshopRouter(t, secret, users, cars, sj)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/service-jobs", nil)
	req.Header.Set("Authorization", "Bearer "+testJWT(t, secret, empID, domain.RoleEmployee))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestMVPAccess_ServiceJobGET_ByOpenedOn_EmployeeOK(t *testing.T) {
	t.Parallel()
	secret := "mvp-sj-list-ok"
	empID := uuid.New()
	carID := uuid.New()
	emp, err := domain.NewUser("e-list-ok@test.local", "x", "E", "M", domain.RoleEmployee)
	require.NoError(t, err)
	emp.ID = empID
	users := &mvpUserRepo{byID: map[uuid.UUID]*domain.User{empID: emp}}
	cars := &mvpCarRepo{car: &domain.Car{ID: carID, OwnerID: uuid.New()}}
	sj := &mvpSJRepo{byID: map[uuid.UUID]*domain.ServiceJob{}, byCar: map[uuid.UUID][]*domain.ServiceJob{carID: {}}}

	r := serviceJobWorkshopRouter(t, secret, users, cars, sj)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/service-jobs?opened_on=1990-06-01", nil)
	req.Header.Set("Authorization", "Bearer "+testJWT(t, secret, empID, domain.RoleEmployee))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "[]", strings.TrimSpace(w.Body.String()))
}

func TestMVPAccess_ServiceJobFlow_ReceptionHandoverGetClosed(t *testing.T) {
	t.Parallel()
	secret := "mvp-sj-flow"
	empID := uuid.New()
	carID := uuid.New()
	ownerID := uuid.New()
	emp, err := domain.NewUser("eflow@test.local", "x", "E", "M", domain.RoleEmployee)
	require.NoError(t, err)
	emp.ID = empID
	users := &mvpUserRepo{byID: map[uuid.UUID]*domain.User{empID: emp}}
	cars := &mvpCarRepo{car: &domain.Car{ID: carID, OwnerID: ownerID}}
	sj := &mvpSJRepo{byID: map[uuid.UUID]*domain.ServiceJob{}, byCar: map[uuid.UUID][]*domain.ServiceJob{carID: {}}}

	r := serviceJobWorkshopRouter(t, secret, users, cars, sj)
	auth := "Bearer " + testJWT(t, secret, empID, domain.RoleEmployee)

	postBody, _ := json.Marshal(map[string]any{"car_id": carID.String()})
	preq := httptest.NewRequest(http.MethodPost, "/api/v1/service-jobs", bytes.NewReader(postBody))
	preq.Header.Set("Content-Type", "application/json")
	preq.Header.Set("Authorization", auth)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, preq)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	var created domain.ServiceJob
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created))
	jid := created.ID.String()

	putR, _ := json.Marshal(map[string]any{
		"odometer_km":   5000,
		"general_notes": "reception ok",
	})
	rr := httptest.NewRequest(http.MethodPut, "/api/v1/service-jobs/"+jid+"/reception", bytes.NewReader(putR))
	rr.Header.Set("Content-Type", "application/json")
	rr.Header.Set("Authorization", auth)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, rr)
	require.Equal(t, http.StatusOK, w2.Code, w2.Body.String())

	putH, _ := json.Marshal(map[string]any{"odometer_km": 5010, "general_notes": "handover ok"})
	hr := httptest.NewRequest(http.MethodPut, "/api/v1/service-jobs/"+jid+"/handover", bytes.NewReader(putH))
	hr.Header.Set("Content-Type", "application/json")
	hr.Header.Set("Authorization", auth)
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, hr)
	require.Equal(t, http.StatusOK, w3.Code, w3.Body.String())

	gr := httptest.NewRequest(http.MethodGet, "/api/v1/service-jobs/"+jid, nil)
	gr.Header.Set("Authorization", auth)
	w4 := httptest.NewRecorder()
	r.ServeHTTP(w4, gr)
	require.Equal(t, http.StatusOK, w4.Code, w4.Body.String())
	var detail serviceJobDetailResponse
	require.NoError(t, json.Unmarshal(w4.Body.Bytes(), &detail))
	assert.Equal(t, domain.ServiceJobStatusClosed, detail.Job.Status)
	assert.NotNil(t, detail.Handover)
}

func openServiceJobForReception(t *testing.T, secret string) (*gin.Engine, string, string) {
	t.Helper()
	empID := uuid.New()
	carID := uuid.New()
	ownerID := uuid.New()
	emp, err := domain.NewUser("erec@test.local", "x", "E", "M", domain.RoleEmployee)
	require.NoError(t, err)
	emp.ID = empID
	users := &mvpUserRepo{byID: map[uuid.UUID]*domain.User{empID: emp}}
	cars := &mvpCarRepo{car: &domain.Car{ID: carID, OwnerID: ownerID}}
	sj := &mvpSJRepo{byID: map[uuid.UUID]*domain.ServiceJob{}, byCar: map[uuid.UUID][]*domain.ServiceJob{carID: {}}}

	r := serviceJobWorkshopRouter(t, secret, users, cars, sj)
	auth := "Bearer " + testJWT(t, secret, empID, domain.RoleEmployee)

	postBody, _ := json.Marshal(map[string]any{"car_id": carID.String()})
	preq := httptest.NewRequest(http.MethodPost, "/api/v1/service-jobs", bytes.NewReader(postBody))
	preq.Header.Set("Content-Type", "application/json")
	preq.Header.Set("Authorization", auth)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, preq)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	var created domain.ServiceJob
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created))
	return r, created.ID.String(), auth
}

func putServiceJobReception(r *gin.Engine, jobID, auth, raw string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPut, "/api/v1/service-jobs/"+jobID+"/reception", strings.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", auth)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestMVPAccess_ServiceJobReception_EmptyBody_Not2xx(t *testing.T) {
	t.Parallel()
	r, jid, auth := openServiceJobForReception(t, "mvp-sj-rec-empty")
	w := putServiceJobReception(r, jid, auth, `{}`)
	assert.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
}

func TestMVPAccess_ServiceJobReception_NotesWithoutOdometer_Not2xx(t *testing.T) {
	t.Parallel()
	r, jid, auth := openServiceJobForReception(t, "mvp-sj-rec-notes")
	w := putServiceJobReception(r, jid, auth, `{"general_notes":"x"}`)
	assert.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
}

func TestMVPAccess_ServiceJobReception_OdometerZero_PresentOK(t *testing.T) {
	t.Parallel()
	r, jid, auth := openServiceJobForReception(t, "mvp-sj-rec-zero")
	w := putServiceJobReception(r, jid, auth, `{"odometer_km":0}`)
	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
}
