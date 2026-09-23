package fiscal

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/gaston-garcia-cegid/gonsgarage/internal/core/ports"
	"github.com/gaston-garcia-cegid/gonsgarage/internal/domain"
	fiscalcrypto "github.com/gaston-garcia-cegid/gonsgarage/internal/platform/crypto"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubOAuthStore struct {
	byHash map[string]*ports.FiscalOAuthAuthorization
}

func (s *stubOAuthStore) SaveOAuthAuthorization(ctx context.Context, auth *ports.FiscalOAuthAuthorization) error {
	if s.byHash == nil {
		s.byHash = make(map[string]*ports.FiscalOAuthAuthorization)
	}
	clone := auth.Clone()
	s.byHash[auth.StateSHA256] = &clone
	return nil
}

func (s *stubOAuthStore) ConsumeOAuthAuthorization(ctx context.Context, stateSHA256 string, now time.Time) (*ports.FiscalOAuthAuthorization, error) {
	if s.byHash == nil {
		return nil, ports.ErrFiscalOAuthStateInvalid
	}
	auth := s.byHash[stateSHA256]
	if auth == nil {
		return nil, ports.ErrFiscalOAuthStateInvalid
	}
	if auth.ConsumedAt != nil {
		return nil, ports.ErrFiscalOAuthStateReused
	}
	if !auth.ExpiresAt.After(now) {
		return nil, ports.ErrFiscalOAuthStateExpired
	}
	consumed := now.UTC()
	auth.ConsumedAt = &consumed
	clone := auth.Clone()
	return &clone, nil
}

type stubTokenExchanger struct {
	exchangeErr error
	tokens      ports.FiscalOAuthTokenSet
	lastCode    string
	refreshErr  error
	refreshed   bool
}

func (e *stubTokenExchanger) ExchangeAuthorizationCode(ctx context.Context, code, redirectURI string) (ports.FiscalOAuthTokenSet, error) {
	e.lastCode = code
	if e.exchangeErr != nil {
		return ports.FiscalOAuthTokenSet{}, e.exchangeErr
	}
	return e.tokens, nil
}

func (e *stubTokenExchanger) RefreshAccessToken(ctx context.Context, refreshToken string) (ports.FiscalOAuthTokenSet, error) {
	e.refreshed = true
	if e.refreshErr != nil {
		return ports.FiscalOAuthTokenSet{}, e.refreshErr
	}
	return e.tokens, nil
}

func sha256Hex(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func TestConnectionServiceStartOAuthBindsHashedStateToActorRedirectExpiry(t *testing.T) {
	t.Parallel()

	cipher, err := fiscalcrypto.NewFiscalCredentialCipher("v1", bytes.Repeat([]byte{0x11}, 32))
	require.NoError(t, err)
	adminID, admin := newManagerUser(t)
	repo := &stubFiscalConnectionRepo{}
	oauth := &stubOAuthStore{}
	exchanger := &stubTokenExchanger{tokens: ports.FiscalOAuthTokenSet{
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
		ExpiresIn:    3600,
		Scopes:       []string{"documents"},
	}}
	svc := NewConnectionService(repo, &stubFiscalUserRepo{users: map[uuid.UUID]*domain.User{adminID: admin}}, &stubFiscalProvider{}, cipher, "v1").
		WithOAuth(oauth, exchanger, "https://auth.test/authorize", "client-id")

	started, err := svc.StartOAuthConnect(context.Background(), adminID, ports.FiscalOAuthStartRequest{
		ScopeKey:    "default",
		ProviderKey: "cloudware",
		RedirectURI: "https://app.test/oauth/callback",
	})
	require.NoError(t, err)
	require.NotNil(t, started)
	assert.NotEmpty(t, started.RawState)
	assert.Contains(t, started.AuthorizationURL, started.RawState)
	assert.Contains(t, started.AuthorizationURL, "client-id")
	assert.True(t, started.ExpiresAt.After(time.Now().UTC()))
	assert.Equal(t, ports.FiscalConnectionStateAuthorizing, started.Status.State)

	// Raw state must never be persisted — only SHA-256.
	require.Len(t, oauth.byHash, 1)
	var stored *ports.FiscalOAuthAuthorization
	for _, auth := range oauth.byHash {
		stored = auth
	}
	require.NotNil(t, stored)
	assert.Equal(t, sha256Hex(started.RawState), stored.StateSHA256)
	assert.Equal(t, adminID, stored.ActorID)
	assert.Equal(t, "https://app.test/oauth/callback", stored.RedirectURI)
	assert.Equal(t, started.Status.ID, stored.ConnectionID)
	assert.Nil(t, stored.ConsumedAt)
	assert.NotContains(t, stored.StateSHA256, started.RawState)
}

func TestConnectionServiceOAuthCallbackRejectsInvalidAndReusedState(t *testing.T) {
	t.Parallel()

	cipher, err := fiscalcrypto.NewFiscalCredentialCipher("v1", bytes.Repeat([]byte{0x22}, 32))
	require.NoError(t, err)
	adminID, admin := newManagerUser(t)
	repo := &stubFiscalConnectionRepo{}
	oauth := &stubOAuthStore{}
	exchanger := &stubTokenExchanger{tokens: ports.FiscalOAuthTokenSet{
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
		ExpiresIn:    3600,
		Organization: "org-1",
		Scopes:       []string{"documents"},
	}}
	svc := NewConnectionService(repo, &stubFiscalUserRepo{users: map[uuid.UUID]*domain.User{adminID: admin}}, &stubFiscalProvider{}, cipher, "v1").
		WithOAuth(oauth, exchanger, "https://auth.test/authorize", "client-id")

	started, err := svc.StartOAuthConnect(context.Background(), adminID, ports.FiscalOAuthStartRequest{
		ScopeKey: "default", ProviderKey: "cloudware", RedirectURI: "https://app.test/oauth/callback",
	})
	require.NoError(t, err)

	_, err = svc.CompleteOAuthCallback(context.Background(), ports.FiscalOAuthCallbackRequest{
		State: "not-the-real-state",
		Code:  "auth-code",
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ports.ErrFiscalOAuthStateInvalid)
	assert.Empty(t, exchanger.lastCode)

	status, err := svc.CompleteOAuthCallback(context.Background(), ports.FiscalOAuthCallbackRequest{
		State: started.RawState,
		Code:  "auth-code",
	})
	require.NoError(t, err)
	require.NotNil(t, status)
	assert.Equal(t, ports.FiscalConnectionStateConnected, status.State)
	assert.True(t, status.HasCredentials)
	assert.Equal(t, "org-1", status.ProviderReference)
	assert.Equal(t, "auth-code", exchanger.lastCode)
	assert.NotContains(t, status.ProviderReference, "access-token")

	_, err = svc.CompleteOAuthCallback(context.Background(), ports.FiscalOAuthCallbackRequest{
		State: started.RawState,
		Code:  "second-code",
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ports.ErrFiscalOAuthStateReused)
}

func TestConnectionServiceRefreshFailureSetsActionRequiredWithoutLeakingSecrets(t *testing.T) {
	t.Parallel()

	cipher, err := fiscalcrypto.NewFiscalCredentialCipher("v1", bytes.Repeat([]byte{0x33}, 32))
	require.NoError(t, err)
	adminID, admin := newManagerUser(t)
	repo := &stubFiscalConnectionRepo{}
	oauth := &stubOAuthStore{}
	exchanger := &stubTokenExchanger{
		tokens: ports.FiscalOAuthTokenSet{
			AccessToken: "access-token", RefreshToken: "refresh-token", ExpiresIn: 60, Organization: "org-1",
		},
		refreshErr: ports.NewFiscalProviderError(ports.FiscalErrorClassUnauthorized, "refresh_failed", "token rejected", nil),
	}
	svc := NewConnectionService(repo, &stubFiscalUserRepo{users: map[uuid.UUID]*domain.User{adminID: admin}}, &stubFiscalProvider{}, cipher, "v1").
		WithOAuth(oauth, exchanger, "https://auth.test/authorize", "client-id")

	started, err := svc.StartOAuthConnect(context.Background(), adminID, ports.FiscalOAuthStartRequest{
		ScopeKey: "default", ProviderKey: "cloudware", RedirectURI: "https://app.test/oauth/callback",
	})
	require.NoError(t, err)
	_, err = svc.CompleteOAuthCallback(context.Background(), ports.FiscalOAuthCallbackRequest{State: started.RawState, Code: "code"})
	require.NoError(t, err)

	status, err := svc.RefreshCredentials(context.Background(), adminID, "default", "cloudware")
	require.Error(t, err)
	var providerErr *ports.FiscalProviderError
	require.True(t, errors.As(err, &providerErr))
	assert.Equal(t, "refresh_failed", providerErr.Code)
	assert.NotContains(t, err.Error(), "refresh-token")
	assert.NotContains(t, err.Error(), "access-token")
	require.NotNil(t, status)
	assert.Equal(t, ports.FiscalConnectionStateActionNeeded, status.State)
	assert.True(t, exchanger.refreshed)

	record, err := repo.GetByScopeProvider(context.Background(), "default", "cloudware")
	require.NoError(t, err)
	require.NotNil(t, record)
	assert.Equal(t, ports.FiscalConnectionStateActionNeeded, record.State)
	assert.True(t, len(record.CredentialCiphertext) > 0)
	assert.False(t, strings.Contains(string(record.CredentialCiphertext), "refresh-token"))
}

func TestConnectionServiceEmployeeCannotStartOAuth(t *testing.T) {
	t.Parallel()

	cipher, err := fiscalcrypto.NewFiscalCredentialCipher("v1", bytes.Repeat([]byte{0x44}, 32))
	require.NoError(t, err)
	clientID, client := newClientUser(t)
	svc := NewConnectionService(&stubFiscalConnectionRepo{}, &stubFiscalUserRepo{users: map[uuid.UUID]*domain.User{clientID: client}}, &stubFiscalProvider{}, cipher, "v1").
		WithOAuth(&stubOAuthStore{}, &stubTokenExchanger{}, "https://auth.test/authorize", "client-id")

	_, err = svc.StartOAuthConnect(context.Background(), clientID, ports.FiscalOAuthStartRequest{
		ScopeKey: "default", ProviderKey: "cloudware", RedirectURI: "https://app.test/callback",
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrPermissionDenied)
}
