package handler

import (
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/gaston-garcia-cegid/gonsgarage/internal/core/ports"
	"github.com/gaston-garcia-cegid/gonsgarage/internal/domain"
)

type FiscalIntegrationHandler struct {
	svc ports.FiscalConnectionService
}

func NewFiscalIntegrationHandler(svc ports.FiscalConnectionService) *FiscalIntegrationHandler {
	return &FiscalIntegrationHandler{svc: svc}
}

type FiscalConnectionSetupRequest struct {
	CredentialBase64 string   `json:"credentialBase64"`
	GrantedScopes    []string `json:"grantedScopes"`
}

type FiscalConnectionStatusResponse struct {
	ID                      string     `json:"id"`
	ScopeKey                string     `json:"scopeKey"`
	ProviderKey             string     `json:"providerKey"`
	State                   string     `json:"state"`
	ProviderReference       string     `json:"providerReference,omitempty"`
	GrantedScopes           []string   `json:"grantedScopes,omitempty"`
	AccessExpiresAt         *time.Time `json:"accessExpiresAt,omitempty"`
	LastVerifiedAt          *time.Time `json:"lastVerifiedAt,omitempty"`
	ConnectedAt             *time.Time `json:"connectedAt,omitempty"`
	RevokedAt               *time.Time `json:"revokedAt,omitempty"`
	CreatedBy               string     `json:"createdBy,omitempty"`
	UpdatedBy               *string    `json:"updatedBy,omitempty"`
	CredentialKeyVersion    string     `json:"credentialKeyVersion,omitempty"`
	CredentialFormatVersion int        `json:"credentialFormatVersion,omitempty"`
	HasCredentials          bool       `json:"hasCredentials"`
	CreatedAt               time.Time  `json:"createdAt"`
	UpdatedAt               time.Time  `json:"updatedAt"`
	Version                 int        `json:"version"`
}

func statusResponseFromStatus(status *ports.FiscalConnectionStatus) FiscalConnectionStatusResponse {
	if status == nil {
		return FiscalConnectionStatusResponse{}
	}
	resp := FiscalConnectionStatusResponse{
		ID:                      status.ID.String(),
		ScopeKey:                status.ScopeKey,
		ProviderKey:             status.ProviderKey,
		State:                   string(status.State),
		ProviderReference:       status.ProviderReference,
		GrantedScopes:           append([]string(nil), status.GrantedScopes...),
		AccessExpiresAt:         status.AccessExpiresAt,
		LastVerifiedAt:          status.LastVerifiedAt,
		ConnectedAt:             status.ConnectedAt,
		RevokedAt:               status.RevokedAt,
		CreatedBy:               status.CreatedBy.String(),
		CredentialKeyVersion:    status.CredentialKeyVersion,
		CredentialFormatVersion: status.CredentialFormatVersion,
		HasCredentials:          status.HasCredentials,
		CreatedAt:               status.CreatedAt.UTC(),
		UpdatedAt:               status.UpdatedAt.UTC(),
		Version:                 status.Version,
	}
	if status.UpdatedBy != nil {
		updatedBy := status.UpdatedBy.String()
		resp.UpdatedBy = &updatedBy
	}
	return resp
}

func parseFiscalConnectionPath(c *gin.Context) (string, string, bool) {
	scopeKey := strings.TrimSpace(c.Param("scopeKey"))
	providerKey := strings.TrimSpace(c.Param("providerKey"))
	if scopeKey == "" || providerKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "scopeKey and providerKey are required"})
		return "", "", false
	}
	return scopeKey, providerKey, true
}

func writeFiscalConnectionError(c *gin.Context, err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ports.ErrFiscalConnectionUnavailable) {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "fiscal integration unavailable"})
		return true
	}
	if errors.Is(err, ports.ErrFiscalConnectionCredentialsMissing) {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return true
	}
	if errors.Is(err, domain.ErrPermissionDenied) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return true
	}
	if errors.Is(err, domain.ErrUserNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return true
	}
	var providerErr *ports.FiscalProviderError
	if errors.As(err, &providerErr) {
		c.JSON(http.StatusBadGateway, gin.H{
			"error":   "provider verification failed",
			"class":   string(providerErr.Class),
			"code":    providerErr.Code,
			"message": providerErr.Message,
		})
		return true
	}
	c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	return true
}

// Status returns the connection management state.
func (h *FiscalIntegrationHandler) Status(c *gin.Context) {
	if h == nil || h.svc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "fiscal integration unavailable"})
		return
	}
	userID, err := ContextUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	scopeKey, providerKey, ok := parseFiscalConnectionPath(c)
	if !ok {
		return
	}
	status, err := h.svc.Status(c.Request.Context(), userID, scopeKey, providerKey)
	if err != nil {
		writeFiscalConnectionError(c, err)
		return
	}
	c.JSON(http.StatusOK, statusResponseFromStatus(status))
}

// SetupConnection stores encrypted credentials for the active scope/provider pairing.
func (h *FiscalIntegrationHandler) SetupConnection(c *gin.Context) {
	if h == nil || h.svc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "fiscal integration unavailable"})
		return
	}
	userID, err := ContextUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	scopeKey, providerKey, ok := parseFiscalConnectionPath(c)
	if !ok {
		return
	}
	var req FiscalConnectionSetupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	credential, err := base64.StdEncoding.DecodeString(strings.TrimSpace(req.CredentialBase64))
	if err != nil || len(credential) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid credentialBase64"})
		return
	}
	status, err := h.svc.StoreCredentials(c.Request.Context(), userID, ports.FiscalConnectionSetupRequest{
		ScopeKey:      scopeKey,
		ProviderKey:   providerKey,
		Credential:    credential,
		GrantedScopes: append([]string(nil), req.GrantedScopes...),
	})
	if err != nil {
		writeFiscalConnectionError(c, err)
		return
	}
	c.JSON(http.StatusOK, statusResponseFromStatus(status))
}

// VerifyConnection asks the configured provider to validate the stored credential.
func (h *FiscalIntegrationHandler) VerifyConnection(c *gin.Context) {
	if h == nil || h.svc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "fiscal integration unavailable"})
		return
	}
	userID, err := ContextUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	scopeKey, providerKey, ok := parseFiscalConnectionPath(c)
	if !ok {
		return
	}
	status, err := h.svc.Verify(c.Request.Context(), userID, scopeKey, providerKey)
	if err != nil {
		writeFiscalConnectionError(c, err)
		return
	}
	c.JSON(http.StatusOK, statusResponseFromStatus(status))
}

// RevokeConnection clears the stored credential and marks the connection revoked.
func (h *FiscalIntegrationHandler) RevokeConnection(c *gin.Context) {
	if h == nil || h.svc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "fiscal integration unavailable"})
		return
	}
	userID, err := ContextUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	scopeKey, providerKey, ok := parseFiscalConnectionPath(c)
	if !ok {
		return
	}
	status, err := h.svc.Revoke(c.Request.Context(), userID, scopeKey, providerKey)
	if err != nil {
		writeFiscalConnectionError(c, err)
		return
	}
	c.JSON(http.StatusOK, statusResponseFromStatus(status))
}

// CloudwareReadiness returns enablement gate status without secrets.
func (h *FiscalIntegrationHandler) CloudwareReadiness(c *gin.Context) {
	if h == nil || h.svc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "fiscal integration unavailable"})
		return
	}
	userID, err := ContextUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	report, err := h.svc.Readiness(c.Request.Context(), userID)
	if err != nil {
		writeFiscalConnectionError(c, err)
		return
	}
	gates := make([]gin.H, 0, len(report.Gates))
	for _, g := range report.Gates {
		gates = append(gates, gin.H{
			"name":      g.Name,
			"status":    g.Status,
			"guidance":  g.Guidance,
			"rationale": g.Rationale,
		})
	}
	resp := gin.H{
		"ready": report.Ready,
		"gates": gates,
	}
	if report.ATCommunicationStatus != "" {
		resp["atCommunicationStatus"] = report.ATCommunicationStatus
	}
	c.JSON(http.StatusOK, resp)
}

// CloudwareConnect starts the manager/admin OAuth authorization redirect.
func (h *FiscalIntegrationHandler) CloudwareConnect(c *gin.Context) {
	if h == nil || h.svc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "fiscal integration unavailable"})
		return
	}
	userID, err := ContextUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	var body struct {
		ScopeKey    string `json:"scopeKey"`
		RedirectURI string `json:"redirectUri"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	if strings.TrimSpace(body.ScopeKey) == "" {
		body.ScopeKey = "default"
	}
	result, err := h.svc.StartOAuthConnect(c.Request.Context(), userID, ports.FiscalOAuthStartRequest{
		ScopeKey:    body.ScopeKey,
		ProviderKey: "cloudware",
		RedirectURI: body.RedirectURI,
	})
	if err != nil {
		if errors.Is(err, ports.ErrFiscalOAuthUnavailable) {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "cloudware oauth unavailable"})
			return
		}
		writeFiscalConnectionError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"authorizationUrl": result.AuthorizationURL,
		"expiresAt":        result.ExpiresAt.UTC(),
		"connection":       statusResponseFromStatus(result.Status),
	})
}

// CloudwareOAuthCallback consumes one-time state and exchanges the authorization code.
func (h *FiscalIntegrationHandler) CloudwareOAuthCallback(c *gin.Context) {
	if h == nil || h.svc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "fiscal integration unavailable"})
		return
	}
	state := strings.TrimSpace(c.Query("state"))
	code := strings.TrimSpace(c.Query("code"))
	status, err := h.svc.CompleteOAuthCallback(c.Request.Context(), ports.FiscalOAuthCallbackRequest{State: state, Code: code})
	if err != nil {
		if errors.Is(err, ports.ErrFiscalOAuthStateInvalid) || errors.Is(err, ports.ErrFiscalOAuthStateReused) || errors.Is(err, ports.ErrFiscalOAuthStateExpired) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid oauth state"})
			return
		}
		writeFiscalConnectionError(c, err)
		return
	}
	c.JSON(http.StatusOK, statusResponseFromStatus(status))
}

const cloudwareDefaultScopeKey = "default"
const cloudwareProviderKey = "cloudware"

// CloudwareConnectionStatus returns the Cloudware connection for the default scope.
func (h *FiscalIntegrationHandler) CloudwareConnectionStatus(c *gin.Context) {
	if h == nil || h.svc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "fiscal integration unavailable"})
		return
	}
	userID, err := ContextUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	status, err := h.svc.Status(c.Request.Context(), userID, cloudwareDefaultScopeKey, cloudwareProviderKey)
	if err != nil {
		writeFiscalConnectionError(c, err)
		return
	}
	c.JSON(http.StatusOK, statusResponseFromStatus(status))
}

// CloudwareVerify verifies the Cloudware connection credentials.
func (h *FiscalIntegrationHandler) CloudwareVerify(c *gin.Context) {
	if h == nil || h.svc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "fiscal integration unavailable"})
		return
	}
	userID, err := ContextUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	status, err := h.svc.Verify(c.Request.Context(), userID, cloudwareDefaultScopeKey, cloudwareProviderKey)
	if err != nil {
		writeFiscalConnectionError(c, err)
		return
	}
	c.JSON(http.StatusOK, statusResponseFromStatus(status))
}

// CloudwareDisconnect revokes the Cloudware connection credentials.
func (h *FiscalIntegrationHandler) CloudwareDisconnect(c *gin.Context) {
	if h == nil || h.svc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "fiscal integration unavailable"})
		return
	}
	userID, err := ContextUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	status, err := h.svc.Revoke(c.Request.Context(), userID, cloudwareDefaultScopeKey, cloudwareProviderKey)
	if err != nil {
		writeFiscalConnectionError(c, err)
		return
	}
	c.JSON(http.StatusOK, statusResponseFromStatus(status))
}
