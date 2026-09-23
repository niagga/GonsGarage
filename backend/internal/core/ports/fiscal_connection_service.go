package ports

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	// ErrFiscalConnectionUnavailable reports that the integration is disabled or misconfigured.
	ErrFiscalConnectionUnavailable = errors.New("fiscal connection service unavailable")
	// ErrFiscalConnectionCredentialsMissing reports that verification was requested without stored credentials.
	ErrFiscalConnectionCredentialsMissing = errors.New("fiscal connection credentials are missing")
)

// FiscalConnectionSetupRequest stores provider-neutral opaque credentials.
type FiscalConnectionSetupRequest struct {
	ScopeKey      string
	ProviderKey   string
	Credential    []byte
	GrantedScopes []string
}

// FiscalConnectionStatus is a safe read model for handlers and callers.
type FiscalConnectionStatus struct {
	ID                      uuid.UUID
	ScopeKey                string
	ProviderKey             string
	State                   FiscalConnectionState
	ProviderReference       string
	GrantedScopes           []string
	AccessExpiresAt         *time.Time
	LastVerifiedAt          *time.Time
	ConnectedAt             *time.Time
	RevokedAt               *time.Time
	CreatedBy               uuid.UUID
	UpdatedBy               *uuid.UUID
	CredentialKeyVersion    string
	CredentialFormatVersion int
	HasCredentials          bool
	CreatedAt               time.Time
	UpdatedAt               time.Time
	Version                 int
}

// Clone returns a deep copy of the status.
func (s FiscalConnectionStatus) Clone() FiscalConnectionStatus {
	return FiscalConnectionStatus{
		ID:                      s.ID,
		ScopeKey:                s.ScopeKey,
		ProviderKey:             s.ProviderKey,
		State:                   s.State,
		ProviderReference:       s.ProviderReference,
		GrantedScopes:           cloneStrings(s.GrantedScopes),
		AccessExpiresAt:         cloneTimePtr(s.AccessExpiresAt),
		LastVerifiedAt:          cloneTimePtr(s.LastVerifiedAt),
		ConnectedAt:             cloneTimePtr(s.ConnectedAt),
		RevokedAt:               cloneTimePtr(s.RevokedAt),
		CreatedBy:               s.CreatedBy,
		UpdatedBy:               cloneUUIDPtr(s.UpdatedBy),
		CredentialKeyVersion:    s.CredentialKeyVersion,
		CredentialFormatVersion: s.CredentialFormatVersion,
		HasCredentials:          s.HasCredentials,
		CreatedAt:               s.CreatedAt,
		UpdatedAt:               s.UpdatedAt,
		Version:                 s.Version,
	}
}

// FiscalConnectionService manages provider-neutral connection lifecycle.
type FiscalConnectionService interface {
	Status(ctx context.Context, requestingUserID uuid.UUID, scopeKey, providerKey string) (*FiscalConnectionStatus, error)
	StoreCredentials(ctx context.Context, requestingUserID uuid.UUID, req FiscalConnectionSetupRequest) (*FiscalConnectionStatus, error)
	Verify(ctx context.Context, requestingUserID uuid.UUID, scopeKey, providerKey string) (*FiscalConnectionStatus, error)
	Revoke(ctx context.Context, requestingUserID uuid.UUID, scopeKey, providerKey string) (*FiscalConnectionStatus, error)
	StartOAuthConnect(ctx context.Context, requestingUserID uuid.UUID, req FiscalOAuthStartRequest) (*FiscalOAuthStartResult, error)
	CompleteOAuthCallback(ctx context.Context, req FiscalOAuthCallbackRequest) (*FiscalConnectionStatus, error)
	RefreshCredentials(ctx context.Context, requestingUserID uuid.UUID, scopeKey, providerKey string) (*FiscalConnectionStatus, error)
	Readiness(ctx context.Context, requestingUserID uuid.UUID) (*FiscalReadinessReport, error)
}
