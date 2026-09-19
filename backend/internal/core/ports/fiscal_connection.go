package ports

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// FiscalConnectionState is the neutral lifecycle for a provider connection.
type FiscalConnectionState string

const (
	FiscalConnectionStateDisconnected FiscalConnectionState = "disconnected"
	FiscalConnectionStateAuthorizing  FiscalConnectionState = "authorizing"
	FiscalConnectionStateConnected    FiscalConnectionState = "connected"
	FiscalConnectionStateActionNeeded FiscalConnectionState = "action_required"
	FiscalConnectionStateRevoked      FiscalConnectionState = "revoked"
)

// IsValid reports whether the state is recognized by the contract.
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

// IsReady reports whether the connection can be used for an operation.
func (s FiscalConnectionState) IsReady() bool {
	return s == FiscalConnectionStateConnected
}

// FiscalConnectionRecord is the minimal provider-neutral connection record.
type FiscalConnectionRecord struct {
	ID                      uuid.UUID
	ScopeKey                string
	ProviderKey             string
	State                   FiscalConnectionState
	ProviderReference       string
	GrantedScopes           []string
	AccessExpiresAt         *time.Time
	CredentialCiphertext    []byte
	CredentialNonce         []byte
	CredentialKeyVersion    string
	CredentialFormatVersion int
	LastVerifiedAt          *time.Time
	ConnectedAt             *time.Time
	RevokedAt               *time.Time
	CreatedAt               time.Time
	UpdatedAt               time.Time
	Version                 int
}

// Clone returns a deep copy of the record.
func (r FiscalConnectionRecord) Clone() FiscalConnectionRecord {
	return FiscalConnectionRecord{
		ID:                      r.ID,
		ScopeKey:                r.ScopeKey,
		ProviderKey:             r.ProviderKey,
		State:                   r.State,
		ProviderReference:       r.ProviderReference,
		GrantedScopes:           cloneStrings(r.GrantedScopes),
		AccessExpiresAt:         cloneTimePtr(r.AccessExpiresAt),
		CredentialCiphertext:    cloneBytes(r.CredentialCiphertext),
		CredentialNonce:         cloneBytes(r.CredentialNonce),
		CredentialKeyVersion:    r.CredentialKeyVersion,
		CredentialFormatVersion: r.CredentialFormatVersion,
		LastVerifiedAt:          cloneTimePtr(r.LastVerifiedAt),
		ConnectedAt:             cloneTimePtr(r.ConnectedAt),
		RevokedAt:               cloneTimePtr(r.RevokedAt),
		CreatedAt:               r.CreatedAt,
		UpdatedAt:               r.UpdatedAt,
		Version:                 r.Version,
	}
}

// FiscalConnectionRepository persists provider-neutral connection records.
type FiscalConnectionRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*FiscalConnectionRecord, error)
	Save(ctx context.Context, record *FiscalConnectionRecord) error
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

func cloneTimePtr(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	clone := *value
	return &clone
}
