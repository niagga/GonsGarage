package handler

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gaston-garcia-cegid/gonsgarage/internal/core/ports"
	"github.com/gaston-garcia-cegid/gonsgarage/internal/domain"
	"github.com/gaston-garcia-cegid/gonsgarage/internal/middleware"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubFiscalIntegrationService struct {
	statusByKey  map[string]*ports.FiscalConnectionStatus
	storeStatus  *ports.FiscalConnectionStatus
	verifyStatus *ports.FiscalConnectionStatus
	revokeStatus *ports.FiscalConnectionStatus
	storeReq     ports.FiscalConnectionSetupRequest
	statusCalls  int
	verifyCalls  int
	revokeCalls  int
}

func (s *stubFiscalIntegrationService) key(scopeKey, providerKey string) string {
	return scopeKey + "|" + providerKey
}

func (s *stubFiscalIntegrationService) Status(ctx context.Context, requestingUserID uuid.UUID, scopeKey, providerKey string) (*ports.FiscalConnectionStatus, error) {
	s.statusCalls++
	if s.statusByKey != nil {
		if status := s.statusByKey[s.key(scopeKey, providerKey)]; status != nil {
			clone := status.Clone()
			return &clone, nil
		}
	}
	if s.storeStatus != nil {
		clone := s.storeStatus.Clone()
		return &clone, nil
	}
	return &ports.FiscalConnectionStatus{ScopeKey: scopeKey, ProviderKey: providerKey, State: ports.FiscalConnectionStateDisconnected}, nil
}

func (s *stubFiscalIntegrationService) StoreCredentials(ctx context.Context, requestingUserID uuid.UUID, req ports.FiscalConnectionSetupRequest) (*ports.FiscalConnectionStatus, error) {
	s.storeReq = req
	if s.storeStatus != nil {
		clone := s.storeStatus.Clone()
		return &clone, nil
	}
	return &ports.FiscalConnectionStatus{ScopeKey: req.ScopeKey, ProviderKey: req.ProviderKey, State: ports.FiscalConnectionStateAuthorizing, HasCredentials: true}, nil
}

func (s *stubFiscalIntegrationService) Verify(ctx context.Context, requestingUserID uuid.UUID, scopeKey, providerKey string) (*ports.FiscalConnectionStatus, error) {
	s.verifyCalls++
	if s.verifyStatus != nil {
		clone := s.verifyStatus.Clone()
		return &clone, nil
	}
	return &ports.FiscalConnectionStatus{ScopeKey: scopeKey, ProviderKey: providerKey, State: ports.FiscalConnectionStateConnected, HasCredentials: true}, nil
}

func (s *stubFiscalIntegrationService) Revoke(ctx context.Context, requestingUserID uuid.UUID, scopeKey, providerKey string) (*ports.FiscalConnectionStatus, error) {
	s.revokeCalls++
	if s.revokeStatus != nil {
		clone := s.revokeStatus.Clone()
		return &clone, nil
	}
	return &ports.FiscalConnectionStatus{ScopeKey: scopeKey, ProviderKey: providerKey, State: ports.FiscalConnectionStateRevoked}, nil
}

func (s *stubFiscalIntegrationService) StartOAuthConnect(ctx context.Context, requestingUserID uuid.UUID, req ports.FiscalOAuthStartRequest) (*ports.FiscalOAuthStartResult, error) {
	return nil, ports.ErrFiscalOAuthUnavailable
}

func (s *stubFiscalIntegrationService) CompleteOAuthCallback(ctx context.Context, req ports.FiscalOAuthCallbackRequest) (*ports.FiscalConnectionStatus, error) {
	return nil, ports.ErrFiscalOAuthUnavailable
}

func (s *stubFiscalIntegrationService) RefreshCredentials(ctx context.Context, requestingUserID uuid.UUID, scopeKey, providerKey string) (*ports.FiscalConnectionStatus, error) {
	return nil, ports.ErrFiscalOAuthUnavailable
}

func (s *stubFiscalIntegrationService) Readiness(ctx context.Context, requestingUserID uuid.UUID) (*ports.FiscalReadinessReport, error) {
	return &ports.FiscalReadinessReport{Ready: false, Gates: []ports.FiscalEnablementGateView{{Name: "oauth_security", Status: "pending", Guidance: "Complete OAuth evidence."}}}, nil
}

func TestFiscalIntegrationHandler_StatusSetupVerifyRevoke(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)
	secret := "fiscal-handler-secret"
	am := middleware.NewAuthMiddleware(secret)
	managerID := uuid.New()

	connectedAt := time.Date(2025, 1, 2, 10, 0, 0, 0, time.UTC)
	status := &ports.FiscalConnectionStatus{
		ID:                uuid.New(),
		ScopeKey:          "default",
		ProviderKey:       "mock",
		State:             ports.FiscalConnectionStateConnected,
		ProviderReference: "org-1",
		GrantedScopes:     []string{"issue"},
		ConnectedAt:       &connectedAt,
		HasCredentials:    true,
		CreatedAt:         time.Now().UTC(),
		UpdatedAt:         time.Now().UTC(),
		Version:           3,
	}
	service := &stubFiscalIntegrationService{statusByKey: map[string]*ports.FiscalConnectionStatus{"default|mock": status}, storeStatus: status, verifyStatus: status, revokeStatus: &ports.FiscalConnectionStatus{ScopeKey: "default", ProviderKey: "mock", State: ports.FiscalConnectionStateRevoked, HasCredentials: false}}
	h := NewFiscalIntegrationHandler(service)

	r := gin.New()
	api := r.Group("/api/v1")
	api.Use(middleware.GinBearerJWT(am))
	fiscal := api.Group("/fiscal")
	fiscal.Use(middleware.RequireStaffManagers())
	connections := fiscal.Group("/connections/:scopeKey/:providerKey")
	connections.GET("", h.Status)
	connections.PUT("", h.SetupConnection)
	connections.POST("/verify", h.VerifyConnection)
	connections.DELETE("", h.RevokeConnection)

	statusReq := httptest.NewRequest(http.MethodGet, "/api/v1/fiscal/connections/default/mock", nil)
	statusReq.Header.Set("Authorization", "Bearer "+testJWTHandler(t, secret, managerID, domain.RoleManager))
	statusResp := httptest.NewRecorder()
	r.ServeHTTP(statusResp, statusReq)
	require.Equal(t, http.StatusOK, statusResp.Code)
	var statusBody map[string]any
	require.NoError(t, json.Unmarshal(statusResp.Body.Bytes(), &statusBody))
	assert.Equal(t, "mock", statusBody["providerKey"])
	assert.Equal(t, true, statusBody["hasCredentials"])
	assert.NotContains(t, statusResp.Body.String(), "credentialCiphertext")
	assert.NotContains(t, statusResp.Body.String(), "credentialNonce")

	payload := map[string]any{
		"credentialBase64": base64.StdEncoding.EncodeToString([]byte("super-secret")),
		"grantedScopes":    []string{"issue", "void"},
	}
	payloadBytes, err := json.Marshal(payload)
	require.NoError(t, err)
	setupReq := httptest.NewRequest(http.MethodPut, "/api/v1/fiscal/connections/default/mock", bytes.NewReader(payloadBytes))
	setupReq.Header.Set("Authorization", "Bearer "+testJWTHandler(t, secret, managerID, domain.RoleManager))
	setupReq.Header.Set("Content-Type", "application/json")
	setupResp := httptest.NewRecorder()
	r.ServeHTTP(setupResp, setupReq)
	require.Equal(t, http.StatusOK, setupResp.Code)
	assert.Equal(t, []byte("super-secret"), service.storeReq.Credential)
	assert.Equal(t, "default", service.storeReq.ScopeKey)
	assert.Equal(t, "mock", service.storeReq.ProviderKey)
	assert.NotContains(t, setupResp.Body.String(), "super-secret")

	verifyReq := httptest.NewRequest(http.MethodPost, "/api/v1/fiscal/connections/default/mock/verify", nil)
	verifyReq.Header.Set("Authorization", "Bearer "+testJWTHandler(t, secret, managerID, domain.RoleManager))
	verifyResp := httptest.NewRecorder()
	r.ServeHTTP(verifyResp, verifyReq)
	require.Equal(t, http.StatusOK, verifyResp.Code)
	assert.Equal(t, 1, service.verifyCalls)

	revokeReq := httptest.NewRequest(http.MethodDelete, "/api/v1/fiscal/connections/default/mock", nil)
	revokeReq.Header.Set("Authorization", "Bearer "+testJWTHandler(t, secret, managerID, domain.RoleManager))
	revokeResp := httptest.NewRecorder()
	r.ServeHTTP(revokeResp, revokeReq)
	require.Equal(t, http.StatusOK, revokeResp.Code)
	assert.Equal(t, 1, service.revokeCalls)
}

func TestFiscalIntegrationHandler_EmployeeLegalActionsForbidden(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)
	secret := "fiscal-int-employee"
	employeeID := uuid.New()
	service := &stubFiscalIntegrationService{}
	h := NewFiscalIntegrationHandler(service)

	r := gin.New()
	api := r.Group("/api/v1")
	api.Use(middleware.GinBearerJWT(middleware.NewAuthMiddleware(secret)))
	fiscal := api.Group("/fiscal")
	fiscal.Use(middleware.RequireStaffManagers())
	connections := fiscal.Group("/connections/:scopeKey/:providerKey")
	connections.GET("", h.Status)
	connections.PUT("", h.SetupConnection)
	connections.POST("/verify", h.VerifyConnection)
	connections.DELETE("", h.RevokeConnection)

	token := testJWTHandler(t, secret, employeeID, domain.RoleEmployee)
	for _, tc := range []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodGet, "/api/v1/fiscal/connections/default/mock", ""},
		{http.MethodPut, "/api/v1/fiscal/connections/default/mock", `{"credentialBase64":"` + base64.StdEncoding.EncodeToString([]byte("x")) + `"}`},
		{http.MethodPost, "/api/v1/fiscal/connections/default/mock/verify", ""},
		{http.MethodDelete, "/api/v1/fiscal/connections/default/mock", ""},
	} {
		var bodyReader *bytes.Reader
		if tc.body != "" {
			bodyReader = bytes.NewReader([]byte(tc.body))
		} else {
			bodyReader = bytes.NewReader(nil)
		}
		req := httptest.NewRequest(tc.method, tc.path, bodyReader)
		req.Header.Set("Authorization", "Bearer "+token)
		if tc.body != "" {
			req.Header.Set("Content-Type", "application/json")
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusForbidden, w.Code, "%s %s", tc.method, tc.path)
	}
	assert.Equal(t, 0, service.statusCalls)
	assert.Equal(t, 0, service.verifyCalls)
	assert.Equal(t, 0, service.revokeCalls)
}
