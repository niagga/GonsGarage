package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	// ErrFiscalConnectionStateInvalid is returned when a connection state is unknown.
	ErrFiscalConnectionStateInvalid = errors.New("invalid fiscal connection state")
	// ErrFiscalConnectionFixed is returned when a fixed provider value would change.
	ErrFiscalConnectionFixed = errors.New("fiscal connection provider is fixed")
	// ErrFiscalConnectionRoleDenied is returned when the role cannot manage the connection.
	ErrFiscalConnectionRoleDenied = errors.New("fiscal connection role is not allowed")
)

// FiscalConnectionState models the provider connection lifecycle.
type FiscalConnectionState string

const (
	FiscalConnectionStateDisconnected FiscalConnectionState = "disconnected"
	FiscalConnectionStateAuthorizing  FiscalConnectionState = "authorizing"
	FiscalConnectionStateConnected    FiscalConnectionState = "connected"
	FiscalConnectionStateActionNeeded FiscalConnectionState = "action_required"
	FiscalConnectionStateRevoked      FiscalConnectionState = "revoked"
)

// IsValid reports whether the state is recognized.
func (s FiscalConnectionState) IsValid() bool {
	switch s {
	case FiscalConnectionStateDisconnected,
		FiscalConnectionStateAuthorizing,
		FiscalConnectionStateConnected,
		FiscalConnectionStateActionNeeded,
		FiscalConnectionStateRevoked:
		return true
	default:
		return false
	}
}

// IsReady reports whether the connection can be used for issuance.
func (s FiscalConnectionState) IsReady() bool {
	return s == FiscalConnectionStateConnected
}

// FiscalConnectionAction identifies the connection management matrix.
type FiscalConnectionAction string

const (
	FiscalConnectionActionView       FiscalConnectionAction = "view"
	FiscalConnectionActionVerify     FiscalConnectionAction = "verify"
	FiscalConnectionActionReconnect  FiscalConnectionAction = "reconnect"
	FiscalConnectionActionDisconnect FiscalConnectionAction = "disconnect"
	FiscalConnectionActionRevoke     FiscalConnectionAction = "revoke"
)

// IsValid reports whether the action is recognized.
func (a FiscalConnectionAction) IsValid() bool {
	switch a {
	case FiscalConnectionActionView,
		FiscalConnectionActionVerify,
		FiscalConnectionActionReconnect,
		FiscalConnectionActionDisconnect,
		FiscalConnectionActionRevoke:
		return true
	default:
		return false
	}
}

// FiscalConnection is the provider-neutral connection aggregate.
type FiscalConnection struct {
	ID                      uuid.UUID             `json:"id" gorm:"type:uuid;primaryKey"`
	ScopeKey                string                `json:"scopeKey" gorm:"type:varchar(80);not null;default:default;index"`
	ProviderKey             string                `json:"providerKey" gorm:"type:varchar(40);not null;index"`
	State                   FiscalConnectionState `json:"state" gorm:"type:varchar(24);not null;index"`
	ProviderOrganizationRef string                `json:"providerOrganizationRef,omitempty" gorm:"column:provider_organization_ref;type:varchar(255)"`
	GrantedScopes           []string              `json:"grantedScopes,omitempty" gorm:"type:text[]"`
	AccessExpiresAt         *time.Time            `json:"accessExpiresAt,omitempty" gorm:"column:access_expires_at"`
	CredentialCiphertext    []byte                `json:"credentialCiphertext,omitempty" gorm:"column:credential_ciphertext"`
	CredentialNonce         []byte                `json:"credentialNonce,omitempty" gorm:"column:credential_nonce"`
	CredentialKeyVersion    string                `json:"credentialKeyVersion,omitempty" gorm:"column:credential_key_version;type:varchar(40)"`
	CredentialFormatVersion int                   `json:"credentialFormatVersion,omitempty" gorm:"column:credential_format_version"`
	LastVerifiedAt          *time.Time            `json:"lastVerifiedAt,omitempty" gorm:"column:last_verified_at"`
	ConnectedAt             *time.Time            `json:"connectedAt,omitempty" gorm:"column:connected_at"`
	RevokedAt               *time.Time            `json:"revokedAt,omitempty" gorm:"column:revoked_at"`
	CreatedBy               uuid.UUID             `json:"createdBy" gorm:"type:uuid;not null;index"`
	UpdatedBy               *uuid.UUID            `json:"updatedBy,omitempty" gorm:"type:uuid;index"`
	CreatedAt               time.Time             `json:"createdAt" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt               time.Time             `json:"updatedAt" gorm:"column:updated_at;autoUpdateTime"`
	Version                 int                   `json:"version" gorm:"not null;default:1"`
}

func (FiscalConnection) TableName() string {
	return "fiscal_provider_connections"
}

// Validate checks the structural invariants used by the domain slice.
func (c *FiscalConnection) Validate() error {
	if c == nil {
		return errors.New("fiscal connection is nil")
	}
	if strings.TrimSpace(c.ScopeKey) == "" {
		return errors.New("scope key is required")
	}
	if strings.TrimSpace(c.ProviderKey) == "" {
		return errors.New("provider key is required")
	}
	if !c.State.IsValid() {
		return ErrFiscalConnectionStateInvalid
	}
	return nil
}

// Snapshot returns a deep-copied connection view.
func (c FiscalConnection) Snapshot() FiscalConnection {
	return FiscalConnection{
		ID:                      c.ID,
		ScopeKey:                c.ScopeKey,
		ProviderKey:             c.ProviderKey,
		State:                   c.State,
		ProviderOrganizationRef: c.ProviderOrganizationRef,
		GrantedScopes:           cloneStrings(c.GrantedScopes),
		AccessExpiresAt:         cloneTimePtr(c.AccessExpiresAt),
		CredentialCiphertext:    cloneBytes(c.CredentialCiphertext),
		CredentialNonce:         cloneBytes(c.CredentialNonce),
		CredentialKeyVersion:    c.CredentialKeyVersion,
		CredentialFormatVersion: c.CredentialFormatVersion,
		LastVerifiedAt:          cloneTimePtr(c.LastVerifiedAt),
		ConnectedAt:             cloneTimePtr(c.ConnectedAt),
		RevokedAt:               cloneTimePtr(c.RevokedAt),
		CreatedBy:               c.CreatedBy,
		UpdatedBy:               cloneUUIDPtr(c.UpdatedBy),
		CreatedAt:               c.CreatedAt,
		UpdatedAt:               c.UpdatedAt,
		Version:                 c.Version,
	}
}

// ProviderFixed reports whether the provider key is already bound.
func (c FiscalConnection) ProviderFixed() bool {
	return strings.TrimSpace(c.ProviderKey) != ""
}

// FixProvider binds the connection to a provider key exactly once.
func (c *FiscalConnection) FixProvider(providerKey string) error {
	if c == nil {
		return errors.New("fiscal connection is nil")
	}
	if strings.TrimSpace(providerKey) == "" {
		return fmt.Errorf("%w: provider key is required", ErrFiscalConnectionFixed)
	}
	if c.ProviderKey != "" && c.ProviderKey != providerKey {
		return ErrFiscalConnectionFixed
	}
	c.ProviderKey = providerKey
	return nil
}

// CanUseForIssue reports whether the connection can back an issuance request.
func (c FiscalConnection) CanUseForIssue() bool {
	return c.State.IsReady()
}

// AllowedActions returns the role/action matrix for connection management.
func (c FiscalConnection) AllowedActions(role FiscalActorRole) []FiscalConnectionAction {
	actions := make([]FiscalConnectionAction, 0, 4)
	if !role.CanManage() {
		return actions
	}
	actions = append(actions, FiscalConnectionActionView)
	switch c.State {
	case FiscalConnectionStateDisconnected:
		actions = append(actions, FiscalConnectionActionReconnect)
	case FiscalConnectionStateAuthorizing:
		actions = append(actions, FiscalConnectionActionVerify, FiscalConnectionActionDisconnect, FiscalConnectionActionRevoke)
	case FiscalConnectionStateConnected:
		actions = append(actions, FiscalConnectionActionVerify, FiscalConnectionActionDisconnect, FiscalConnectionActionRevoke)
	case FiscalConnectionStateActionNeeded:
		actions = append(actions, FiscalConnectionActionVerify, FiscalConnectionActionReconnect, FiscalConnectionActionDisconnect, FiscalConnectionActionRevoke)
	case FiscalConnectionStateRevoked:
		// view only
	}
	return dedupeFiscalConnectionActions(actions)
}

// CanAct reports whether the role may perform the requested action.
func (c FiscalConnection) CanAct(role FiscalActorRole, action FiscalConnectionAction) bool {
	for _, allowed := range c.AllowedActions(role) {
		if allowed == action {
			return true
		}
	}
	return false
}

func dedupeFiscalConnectionActions(actions []FiscalConnectionAction) []FiscalConnectionAction {
	seen := make(map[FiscalConnectionAction]bool, len(actions))
	result := make([]FiscalConnectionAction, 0, len(actions))
	for _, action := range actions {
		if seen[action] {
			continue
		}
		seen[action] = true
		result = append(result, action)
	}
	return result
}

func cloneStrings(values []string) []string {
	if values == nil {
		return nil
	}
	return append([]string(nil), values...)
}

func cloneBytes(values []byte) []byte {
	if values == nil {
		return nil
	}
	return append([]byte(nil), values...)
}
