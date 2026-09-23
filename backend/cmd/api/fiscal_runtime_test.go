package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	fiscalsvc "github.com/gaston-garcia-cegid/gonsgarage/internal/service/fiscal"
)

func TestReadyHandler_IgnoresFiscalProviderOutage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	registerCoreReadyRoute(router, func() error { return nil }, func() fiscalsvc.FiscalDependencyStatus {
		return fiscalsvc.BuildFiscalDependencyStatus(fiscalsvc.DependencyStatusInput{
			FeatureEnabled: true, ProviderKey: "cloudware", ProviderHealthy: false, SchemaPresent: true,
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, "ready", body["status"])
	_, hasProvider := body["provider"]
	require.False(t, hasProvider, "/ready must not embed provider health")
}

func TestFiscalDependencyStatusRoute_SurfacesProviderOutage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	registerFiscalDependencyRoute(router, func() fiscalsvc.FiscalDependencyStatus {
		return fiscalsvc.BuildFiscalDependencyStatus(fiscalsvc.DependencyStatusInput{
			FeatureEnabled: true, ProviderKey: "cloudware", ProviderHealthy: false, SchemaPresent: true,
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/fiscal/dependency-status", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, false, body["providerHealthy"])
	issues, ok := body["issues"].([]any)
	require.True(t, ok)
	require.NotEmpty(t, issues)
}

func TestValidateAPIFiscalComposition_ProductionRejectsMock(t *testing.T) {
	err := validateAPIFiscalComposition(fiscalsvc.RuntimeConfig{
		AppEnv: "production", FeatureEnabled: true, ProviderKey: "mock",
		ArtifactBackend: "object", CredentialKeyConfigured: true, CredentialKeyVersion: "v1",
		SchemaPresent: true,
	})
	require.Error(t, err)
	require.Contains(t, stringsToLower(err.Error()), "mock")
}

func stringsToLower(s string) string {
	b := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		b[i] = c
	}
	return string(b)
}
