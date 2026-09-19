package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFiscalConnectionStateAndFixation(t *testing.T) {
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

	conn := testFiscalConnection(FiscalConnectionStateConnected)
	snapshot := conn.Snapshot()
	conn.GrantedScopes[0] = "changed"
	conn.CredentialCiphertext[0] = 9
	conn.CredentialNonce[0] = 8
	require.NotNil(t, snapshot.AccessExpiresAt)
	assert.Equal(t, []string{"invoice:read", "invoice:write"}, snapshot.GrantedScopes)
	assert.Equal(t, byte(1), snapshot.CredentialCiphertext[0])
	assert.Equal(t, byte(4), snapshot.CredentialNonce[0])

	require.NoError(t, conn.FixProvider("mock"))
	require.NoError(t, conn.FixProvider("mock"))
	assert.ErrorIs(t, conn.FixProvider("other"), ErrFiscalConnectionFixed)
	assert.True(t, conn.ProviderFixed())
	assert.True(t, conn.CanUseForIssue())
}

func TestFiscalConnectionRoleActionMatrix(t *testing.T) {
	t.Parallel()

	connected := testFiscalConnection(FiscalConnectionStateConnected)
	assert.ElementsMatch(t, []FiscalConnectionAction{FiscalConnectionActionView, FiscalConnectionActionVerify, FiscalConnectionActionDisconnect, FiscalConnectionActionRevoke}, connected.AllowedActions(FiscalActorRoleAdmin))
	assert.ElementsMatch(t, []FiscalConnectionAction{FiscalConnectionActionView, FiscalConnectionActionVerify, FiscalConnectionActionDisconnect, FiscalConnectionActionRevoke}, connected.AllowedActions(FiscalActorRoleManager))
	assert.Empty(t, connected.AllowedActions(FiscalActorRoleEmployee))
	assert.False(t, connected.CanAct(FiscalActorRoleEmployee, FiscalConnectionActionVerify))
	assert.True(t, connected.CanAct(FiscalActorRoleManager, FiscalConnectionActionDisconnect))

	actionNeeded := testFiscalConnection(FiscalConnectionStateActionNeeded)
	assert.True(t, actionNeeded.CanAct(FiscalActorRoleAdmin, FiscalConnectionActionReconnect))
	assert.True(t, actionNeeded.CanAct(FiscalActorRoleManager, FiscalConnectionActionVerify))
	assert.False(t, actionNeeded.CanAct(FiscalActorRoleClient, FiscalConnectionActionView))

	disconnected := testFiscalConnection(FiscalConnectionStateDisconnected)
	assert.True(t, disconnected.CanAct(FiscalActorRoleAdmin, FiscalConnectionActionReconnect))
	assert.False(t, disconnected.CanAct(FiscalActorRoleEmployee, FiscalConnectionActionReconnect))
}

func testFiscalConnection(state FiscalConnectionState) FiscalConnection {
	accessExpiresAt := time.Date(2026, 1, 2, 12, 0, 0, 0, time.UTC)
	verifiedAt := time.Date(2026, 1, 2, 12, 5, 0, 0, time.UTC)
	connectedAt := time.Date(2026, 1, 2, 12, 10, 0, 0, time.UTC)
	revokedAt := time.Date(2026, 1, 2, 12, 15, 0, 0, time.UTC)
	updatedBy := uuid.New()
	return FiscalConnection{
		ID:                      uuid.New(),
		ScopeKey:                "default",
		ProviderKey:             "mock",
		State:                   state,
		ProviderOrganizationRef: "ORG-1",
		GrantedScopes:           []string{"invoice:read", "invoice:write"},
		AccessExpiresAt:         &accessExpiresAt,
		CredentialCiphertext:    []byte{1, 2, 3},
		CredentialNonce:         []byte{4, 5, 6},
		CredentialKeyVersion:    "v1",
		CredentialFormatVersion: 1,
		LastVerifiedAt:          &verifiedAt,
		ConnectedAt:             &connectedAt,
		RevokedAt:               &revokedAt,
		CreatedBy:               uuid.New(),
		UpdatedBy:               &updatedBy,
		CreatedAt:               time.Unix(1, 0).UTC(),
		UpdatedAt:               time.Unix(2, 0).UTC(),
		Version:                 1,
	}
}
