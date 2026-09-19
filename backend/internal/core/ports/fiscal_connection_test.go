package ports

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type noopFiscalConnectionRepository struct{}

func (noopFiscalConnectionRepository) GetByID(context.Context, uuid.UUID) (*FiscalConnectionRecord, error) {
	return nil, nil
}

func (noopFiscalConnectionRepository) Save(context.Context, *FiscalConnectionRecord) error {
	return nil
}

var _ FiscalConnectionRepository = noopFiscalConnectionRepository{}

func TestFiscalConnectionStateInvariants(t *testing.T) {
	t.Parallel()

	states := []FiscalConnectionState{
		FiscalConnectionStateDisconnected,
		FiscalConnectionStateAuthorizing,
		FiscalConnectionStateConnected,
		FiscalConnectionStateActionNeeded,
		FiscalConnectionStateRevoked,
	}

	require.Len(t, states, 5)
	for _, state := range states {
		assert.True(t, state.IsValid(), state)
	}
	assert.False(t, FiscalConnectionState("unknown").IsValid())
	assert.True(t, FiscalConnectionStateConnected.IsReady())
	assert.False(t, FiscalConnectionStateDisconnected.IsReady())
}

func TestFiscalConnectionRecordClone(t *testing.T) {
	t.Parallel()

	accessExpiresAt := time.Date(2026, 1, 2, 12, 0, 0, 0, time.UTC)
	lastVerifiedAt := time.Date(2026, 1, 2, 12, 5, 0, 0, time.UTC)
	connectedAt := time.Date(2026, 1, 2, 12, 10, 0, 0, time.UTC)
	revokedAt := time.Date(2026, 1, 2, 12, 15, 0, 0, time.UTC)

	record := FiscalConnectionRecord{
		ID:                      uuid.New(),
		ScopeKey:                "default",
		ProviderKey:             "mock",
		State:                   FiscalConnectionStateConnected,
		ProviderReference:       "ORG-1",
		GrantedScopes:           []string{"invoice:read", "invoice:write"},
		AccessExpiresAt:         &accessExpiresAt,
		CredentialCiphertext:    []byte{1, 2, 3},
		CredentialNonce:         []byte{4, 5, 6},
		CredentialKeyVersion:    "v1",
		CredentialFormatVersion: 1,
		LastVerifiedAt:          &lastVerifiedAt,
		ConnectedAt:             &connectedAt,
		RevokedAt:               &revokedAt,
		CreatedAt:               time.Unix(1, 0).UTC(),
		UpdatedAt:               time.Unix(2, 0).UTC(),
		Version:                 1,
	}

	clone := record.Clone()

	record.GrantedScopes[0] = "changed"
	record.CredentialCiphertext[0] = 9
	record.CredentialNonce[0] = 8
	accessExpiresAt = accessExpiresAt.Add(time.Hour)
	lastVerifiedAt = lastVerifiedAt.Add(time.Hour)
	connectedAt = connectedAt.Add(time.Hour)
	revokedAt = revokedAt.Add(time.Hour)

	require.NotNil(t, clone.AccessExpiresAt)
	require.NotNil(t, clone.LastVerifiedAt)
	require.NotNil(t, clone.ConnectedAt)
	require.NotNil(t, clone.RevokedAt)
	assert.Equal(t, []string{"invoice:read", "invoice:write"}, clone.GrantedScopes)
	assert.Equal(t, byte(1), clone.CredentialCiphertext[0])
	assert.Equal(t, byte(4), clone.CredentialNonce[0])
	assert.Equal(t, time.Date(2026, 1, 2, 12, 0, 0, 0, time.UTC), *clone.AccessExpiresAt)
	assert.Equal(t, time.Date(2026, 1, 2, 12, 5, 0, 0, time.UTC), *clone.LastVerifiedAt)
	assert.Equal(t, time.Date(2026, 1, 2, 12, 10, 0, 0, time.UTC), *clone.ConnectedAt)
	assert.Equal(t, time.Date(2026, 1, 2, 12, 15, 0, 0, time.UTC), *clone.RevokedAt)
}
