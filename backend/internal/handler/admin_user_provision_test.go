package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gaston-garcia-cegid/gonsgarage/internal/core/ports"
	"github.com/gaston-garcia-cegid/gonsgarage/internal/domain"
	"github.com/gaston-garcia-cegid/gonsgarage/internal/middleware"
	authsvc "github.com/gaston-garcia-cegid/gonsgarage/internal/service/auth"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// provisionTestUserRepo is a minimal UserRepository for admin provision HTTP tests.
type provisionTestUserRepo struct {
	byEmail   map[string]*domain.User
	createErr error
}

func newProvisionTestUserRepo() *provisionTestUserRepo {
	return &provisionTestUserRepo{byEmail: make(map[string]*domain.User)}
}

func (s *provisionTestUserRepo) Create(ctx context.Context, user *domain.User) error {
	if s.createErr != nil {
		return s.createErr
	}
	s.byEmail[user.Email] = user
	return nil
}

func (s *provisionTestUserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	u, ok := s.byEmail[email]
	if !ok {
		return nil, nil
	}
	return u, nil
}

func (s *provisionTestUserRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	for _, u := range s.byEmail {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, errors.New("not found")
}

func (s *provisionTestUserRepo) GetByRole(ctx context.Context, role string, limit, offset int) ([]*domain.User, error) {
	items := make([]*domain.User, 0)
	for _, u := range s.byEmail {
		if u.Role == role {
			items = append(items, u)
		}
	}
	return items, nil
}
func (s *provisionTestUserRepo) List(ctx context.Context, limit, offset int) ([]*domain.User, error) {
	return nil, nil
}
func (s *provisionTestUserRepo) Update(ctx context.Context, user *domain.User) error { return nil }
func (s *provisionTestUserRepo) Delete(ctx context.Context, id uuid.UUID) error      { return nil }
func (s *provisionTestUserRepo) UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string) error {
	return nil
}
func (s *provisionTestUserRepo) GetActiveUsers(ctx context.Context, limit, offset int) ([]*domain.User, error) {
	return nil, nil
}

func newProvisionTestRouter(t *testing.T, secret string, repo ports.UserRepository) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	am := middleware.NewAuthMiddleware(secret)
	authService := authsvc.NewAuthService(repo, secret, 24)
	h := NewAdminUserHandler(authService, repo)

	r := gin.New()
	api := r.Group("/api/v1")
	api.Use(middleware.GinBearerJWT(am))
	admin := api.Group("/admin")
	admin.Use(middleware.RequireStaffManagers())
	admin.POST("/users", h.ProvisionUser)
	admin.GET("/users/clients", h.ListClients)
	return r
}

func TestProvisionUser_NoJWT_Not2xx(t *testing.T) {
	t.Parallel()
	repo := newProvisionTestUserRepo()
	r := newProvisionTestRouter(t, "prov-secret-nojwt", repo)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users", bytes.NewReader([]byte(`{}`)))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.NotEqual(t, http.StatusCreated, w.Code)
	assert.True(t, w.Code == http.StatusUnauthorized || w.Code == http.StatusBadRequest)
}

func TestProvisionUser_ClientForbidden(t *testing.T) {
	t.Parallel()
	secret := "prov-secret-client"
	repo := newProvisionTestUserRepo()
	r := newProvisionTestRouter(t, secret, repo)
	uid := uuid.New()

	body := map[string]string{
		"email": "new@example.com", "password": "secret12", "firstName": "A", "lastName": "B", "role": domain.RoleClient,
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users", bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer "+testJWT(t, secret, uid, domain.RoleClient))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestProvisionUser_EmployeeForbidden(t *testing.T) {
	t.Parallel()
	secret := "prov-secret-emp"
	repo := newProvisionTestUserRepo()
	r := newProvisionTestRouter(t, secret, repo)
	uid := uuid.New()

	body := map[string]string{
		"email": "new2@example.com", "password": "secret12", "firstName": "A", "lastName": "B", "role": domain.RoleClient,
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users", bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer "+testJWT(t, secret, uid, domain.RoleEmployee))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestProvisionUser_AdminBodyAdminRole_Not2xx(t *testing.T) {
	t.Parallel()
	secret := "prov-secret-adm-admin"
	repo := newProvisionTestUserRepo()
	r := newProvisionTestRouter(t, secret, repo)
	uid := uuid.New()

	body := map[string]string{
		"email": "adm@example.com", "password": "secret12", "firstName": "A", "lastName": "B", "role": domain.RoleAdmin,
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users", bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer "+testJWT(t, secret, uid, domain.RoleAdmin))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.NotEqual(t, http.StatusCreated, w.Code)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestProvisionUser_AdminCreatesClient_201(t *testing.T) {
	t.Parallel()
	secret := "prov-secret-adm-ok"
	repo := newProvisionTestUserRepo()
	r := newProvisionTestRouter(t, secret, repo)
	uid := uuid.New()

	body := map[string]string{
		"email": "clientnew@example.com", "password": "secret12", "firstName": "A", "lastName": "B", "role": domain.RoleClient,
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users", bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer "+testJWT(t, secret, uid, domain.RoleAdmin))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code)
	var out struct {
		User domain.User `json:"user"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out))
	assert.Equal(t, domain.RoleClient, out.User.Role)
	assert.Equal(t, "clientnew@example.com", out.User.Email)
	assert.Empty(t, out.User.Password)
}

func TestProvisionUser_ManagerCreatesManager_Not2xx(t *testing.T) {
	t.Parallel()
	secret := "prov-secret-mgr-mgr"
	repo := newProvisionTestUserRepo()
	r := newProvisionTestRouter(t, secret, repo)
	uid := uuid.New()

	body := map[string]string{
		"email": "mgrdup@example.com", "password": "secret12", "firstName": "A", "lastName": "B", "role": domain.RoleManager,
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users", bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer "+testJWT(t, secret, uid, domain.RoleManager))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.NotEqual(t, http.StatusCreated, w.Code)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestProvisionUser_ManagerCreatesEmployee_201(t *testing.T) {
	t.Parallel()
	secret := "prov-secret-mgr-emp"
	repo := newProvisionTestUserRepo()
	r := newProvisionTestRouter(t, secret, repo)
	uid := uuid.New()

	body := map[string]string{
		"email": "empnew@example.com", "password": "secret12", "firstName": "A", "lastName": "B", "role": domain.RoleEmployee,
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users", bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer "+testJWT(t, secret, uid, domain.RoleManager))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code)
	var out struct {
		User domain.User `json:"user"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out))
	assert.Equal(t, domain.RoleEmployee, out.User.Role)
}

func TestProvisionUser_AdminCreatesManager_201(t *testing.T) {
	t.Parallel()
	secret := "prov-secret-adm-mgr"
	repo := newProvisionTestUserRepo()
	r := newProvisionTestRouter(t, secret, repo)

	body := map[string]string{
		"email": "mgrnew@example.com", "password": "secret12", "firstName": "A", "lastName": "B", "role": domain.RoleManager,
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users", bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer "+testJWT(t, secret, uuid.New(), domain.RoleAdmin))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code)
	var out struct {
		User domain.User `json:"user"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out))
	assert.Equal(t, domain.RoleManager, out.User.Role)
	assert.Empty(t, out.User.Password)
}

func TestProvisionUser_AdminCreatesEmployee_201(t *testing.T) {
	t.Parallel()
	secret := "prov-secret-adm-emp"
	repo := newProvisionTestUserRepo()
	r := newProvisionTestRouter(t, secret, repo)

	body := map[string]string{
		"email": "empfromadm@example.com", "password": "secret12", "firstName": "A", "lastName": "B", "role": domain.RoleEmployee,
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users", bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer "+testJWT(t, secret, uuid.New(), domain.RoleAdmin))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code)
	var out struct {
		User domain.User `json:"user"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out))
	assert.Equal(t, domain.RoleEmployee, out.User.Role)
}

func TestProvisionUser_ManagerCreatesClient_201(t *testing.T) {
	t.Parallel()
	secret := "prov-secret-mgr-client"
	repo := newProvisionTestUserRepo()
	r := newProvisionTestRouter(t, secret, repo)

	body := map[string]string{
		"email": "clientfrommgr@example.com", "password": "secret12", "firstName": "A", "lastName": "B", "role": domain.RoleClient,
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users", bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer "+testJWT(t, secret, uuid.New(), domain.RoleManager))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code)
	var out struct {
		User domain.User `json:"user"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out))
	assert.Equal(t, domain.RoleClient, out.User.Role)
}

func TestProvisionUser_ManagerCreatesAdmin_Not2xx(t *testing.T) {
	t.Parallel()
	secret := "prov-secret-mgr-admin"
	repo := newProvisionTestUserRepo()
	r := newProvisionTestRouter(t, secret, repo)

	body := map[string]string{
		"email": "admfrommgr@example.com", "password": "secret12", "firstName": "A", "lastName": "B", "role": domain.RoleAdmin,
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users", bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer "+testJWT(t, secret, uuid.New(), domain.RoleManager))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.NotEqual(t, http.StatusCreated, w.Code)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestProvisionUser_AdminUnknownRole_Not2xx(t *testing.T) {
	t.Parallel()
	secret := "prov-secret-adm-super"
	repo := newProvisionTestUserRepo()
	r := newProvisionTestRouter(t, secret, repo)

	body := map[string]string{
		"email": "super@example.com", "password": "secret12", "firstName": "A", "lastName": "B", "role": "superuser",
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users", bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer "+testJWT(t, secret, uuid.New(), domain.RoleAdmin))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.NotEqual(t, http.StatusCreated, w.Code)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestProvisionUser_UnknownJWTRole_Forbidden(t *testing.T) {
	t.Parallel()
	secret := "prov-secret-jwt-unknown"
	repo := newProvisionTestUserRepo()
	r := newProvisionTestRouter(t, secret, repo)

	body := map[string]string{
		"email": "x@example.com", "password": "secret12", "firstName": "A", "lastName": "B", "role": domain.RoleClient,
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users", bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer "+testJWT(t, secret, uuid.New(), "superuser"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestListClients_ManagerGetsClientUsersOnly(t *testing.T) {
	t.Parallel()
	secret := "prov-secret-list-clients"
	repo := newProvisionTestUserRepo()

	client, err := domain.NewUser("client@example.com", "secret12", "Cli", "Ent", domain.RoleClient)
	require.NoError(t, err)
	client.ID = uuid.New()
	repo.byEmail[client.Email] = client

	employee, err := domain.NewUser("emp@example.com", "secret12", "Emp", "Loyee", domain.RoleEmployee)
	require.NoError(t, err)
	employee.ID = uuid.New()
	repo.byEmail[employee.Email] = employee

	r := newProvisionTestRouter(t, secret, repo)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users/clients", nil)
	req.Header.Set("Authorization", "Bearer "+testJWT(t, secret, uuid.New(), domain.RoleManager))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var out struct {
		Items []struct {
			ID       string `json:"id"`
			Email    string `json:"email"`
			FirstName string `json:"firstName"`
			LastName  string `json:"lastName"`
		} `json:"items"`
		Total int `json:"total"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out))
	require.Len(t, out.Items, 1)
	assert.Equal(t, "client@example.com", out.Items[0].Email)
	assert.Equal(t, 1, out.Total)
}

func TestListClients_EmployeeForbidden(t *testing.T) {
	t.Parallel()
	secret := "prov-secret-list-emp"
	repo := newProvisionTestUserRepo()
	r := newProvisionTestRouter(t, secret, repo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users/clients", nil)
	req.Header.Set("Authorization", "Bearer "+testJWT(t, secret, uuid.New(), domain.RoleEmployee))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}
