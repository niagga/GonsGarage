package fiscal

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gaston-garcia-cegid/gonsgarage/internal/core/ports"
	"github.com/gaston-garcia-cegid/gonsgarage/internal/domain"
	fiscalcrypto "github.com/gaston-garcia-cegid/gonsgarage/internal/platform/crypto"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubFiscalConnectionRepo struct {
	byScope map[string]*ports.FiscalConnectionRecord
}

func (r *stubFiscalConnectionRepo) key(scopeKey, providerKey string) string {
	return scopeKey + "|" + providerKey
}

func (r *stubFiscalConnectionRepo) GetByID(ctx context.Context, id uuid.UUID) (*ports.FiscalConnectionRecord, error) {
	for _, record := range r.byScope {
		if record != nil && record.ID == id {
			clone := record.Clone()
			return &clone, nil
		}
	}
	return nil, nil
}

func (r *stubFiscalConnectionRepo) GetByScopeProvider(ctx context.Context, scopeKey, providerKey string) (*ports.FiscalConnectionRecord, error) {
	if r.byScope == nil {
		return nil, nil
	}
	record := r.byScope[r.key(scopeKey, providerKey)]
	if record == nil {
		return nil, nil
	}
	clone := record.Clone()
	return &clone, nil
}

func (r *stubFiscalConnectionRepo) Save(ctx context.Context, record *ports.FiscalConnectionRecord) error {
	if r.byScope == nil {
		r.byScope = make(map[string]*ports.FiscalConnectionRecord)
	}
	clone := record.Clone()
	r.byScope[r.key(record.ScopeKey, record.ProviderKey)] = &clone
	return nil
}

type stubFiscalUserRepo struct {
	users map[uuid.UUID]*domain.User
}

func (r *stubFiscalUserRepo) Create(ctx context.Context, user *domain.User) error { return nil }
func (r *stubFiscalUserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	return nil, nil
}
func (r *stubFiscalUserRepo) GetByRole(ctx context.Context, role string, limit, offset int) ([]*domain.User, error) {
	return nil, nil
}
func (r *stubFiscalUserRepo) List(ctx context.Context, limit, offset int) ([]*domain.User, error) {
	return nil, nil
}
func (r *stubFiscalUserRepo) Update(ctx context.Context, user *domain.User) error { return nil }
func (r *stubFiscalUserRepo) Delete(ctx context.Context, id uuid.UUID) error      { return nil }
func (r *stubFiscalUserRepo) UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string) error {
	return nil
}
func (r *stubFiscalUserRepo) GetActiveUsers(ctx context.Context, limit, offset int) ([]*domain.User, error) {
	return nil, nil
}
func (r *stubFiscalUserRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	user := r.users[id]
	if user == nil {
		return nil, nil
	}
	clone := *user
	return &clone, nil
}

type stubFiscalProvider struct {
	verifyResult ports.FiscalVerifyConnectionResult
	verifyErr    error
	lastVerify   ports.FiscalVerifyConnectionRequest
}

func (p *stubFiscalProvider) VerifyConnection(ctx context.Context, req ports.FiscalVerifyConnectionRequest) (ports.FiscalVerifyConnectionResult, error) {
	p.lastVerify = req
	if p.verifyErr != nil {
		return ports.FiscalVerifyConnectionResult{}, p.verifyErr
	}
	return p.verifyResult, nil
}
func (p *stubFiscalProvider) Issue(context.Context, ports.FiscalIssueRequest) (ports.FiscalIssueResult, error) {
	return ports.FiscalIssueResult{}, nil
}
func (p *stubFiscalProvider) Reconcile(context.Context, ports.FiscalReconcileRequest) (ports.FiscalReconcileResult, error) {
	return ports.FiscalReconcileResult{}, nil
}
func (p *stubFiscalProvider) Void(context.Context, ports.FiscalVoidRequest) (ports.FiscalVoidResult, error) {
	return ports.FiscalVoidResult{}, nil
}
func (p *stubFiscalProvider) FetchArtifact(context.Context, ports.FiscalFetchArtifactRequest) (ports.FiscalFetchArtifactResult, error) {
	return ports.FiscalFetchArtifactResult{}, nil
}

func newManagerUser(t *testing.T) (uuid.UUID, *domain.User) {
	t.Helper()
	user, err := domain.NewUser("manager@example.com", "password", "Manager", "One", domain.RoleManager)
	require.NoError(t, err)
	userID := uuid.New()
	user.ID = userID
	return userID, user
}

func newClientUser(t *testing.T) (uuid.UUID, *domain.User) {
	t.Helper()
	user, err := domain.NewUser("client@example.com", "password", "Client", "One", domain.RoleClient)
	require.NoError(t, err)
	userID := uuid.New()
	user.ID = userID
	return userID, user
}

func TestConnectionServiceStoreVerifyAndRevoke(t *testing.T) {
	t.Parallel()

	cipher, err := fiscalcrypto.NewFiscalCredentialCipher("v1", bytes.Repeat([]byte{0x11}, 32))
	require.NoError(t, err)
	adminID, admin := newManagerUser(t)
	provider := &stubFiscalProvider{
		verifyResult: ports.FiscalVerifyConnectionResult{
			FiscalOperationResponse: ports.FiscalOperationResponse{
				ProviderReference: "org-123",
				ObservedAt:        time.Date(2025, 1, 2, 10, 0, 0, 0, time.UTC),
			},
			State:           ports.FiscalConnectionStateConnected,
			GrantedScopes:   []string{"issue", "void"},
			ConnectedAt:     func() *time.Time { v := time.Date(2025, 1, 2, 10, 0, 0, 0, time.UTC); return &v }(),
			AccessExpiresAt: func() *time.Time { v := time.Date(2025, 2, 2, 10, 0, 0, 0, time.UTC); return &v }(),
			OrganizationRef: "org-123",
		},
	}
	repo := &stubFiscalConnectionRepo{}
	svc := NewConnectionService(repo, &stubFiscalUserRepo{users: map[uuid.UUID]*domain.User{adminID: admin}}, provider, cipher, "v1")

	stored, err := svc.StoreCredentials(context.Background(), adminID, ports.FiscalConnectionSetupRequest{
		ScopeKey:      "default",
		ProviderKey:   "mock",
		Credential:    []byte("top-secret"),
		GrantedScopes: []string{"issue"},
	})
	require.NoError(t, err)
	require.NotNil(t, stored)
	assert.True(t, stored.HasCredentials)
	assert.Equal(t, ports.FiscalConnectionStateAuthorizing, stored.State)
	assert.Equal(t, adminID, stored.CreatedBy)
	assert.NotNil(t, stored.UpdatedBy)
	assert.Equal(t, adminID, *stored.UpdatedBy)

	record, err := repo.GetByScopeProvider(context.Background(), "default", "mock")
	require.NoError(t, err)
	require.NotNil(t, record)
	assert.NotEqual(t, []byte("top-secret"), record.CredentialCiphertext)
	assert.Len(t, record.CredentialNonce, 12)
	decrypted, err := cipher.DecryptWithBinding(fiscalcrypto.FiscalCredentialEnvelope{
		FormatVersion: record.CredentialFormatVersion,
		KeyVersion:    record.CredentialKeyVersion,
		Nonce:         record.CredentialNonce,
		Ciphertext:    record.CredentialCiphertext,
	}, fiscalcrypto.FiscalCredentialBinding{
		ConnectionID: record.ID,
		ProviderKey:  record.ProviderKey,
		ScopeKey:     record.ScopeKey,
	})
	require.NoError(t, err)
	assert.Equal(t, []byte("top-secret"), decrypted)

	verified, err := svc.Verify(context.Background(), adminID, "default", "mock")
	require.NoError(t, err)
	require.NotNil(t, verified)
	assert.Equal(t, ports.FiscalConnectionStateConnected, verified.State)
	assert.True(t, verified.HasCredentials)
	assert.Equal(t, "org-123", verified.ProviderReference)
	assert.ElementsMatch(t, []string{"issue", "void"}, verified.GrantedScopes)
	assert.NotNil(t, verified.LastVerifiedAt)
	assert.NotNil(t, verified.ConnectedAt)

	assert.Equal(t, record.CredentialCiphertext, provider.lastVerify.CredentialCiphertext)
	assert.Equal(t, record.CredentialNonce, provider.lastVerify.CredentialNonce)
	assert.Equal(t, record.CredentialKeyVersion, provider.lastVerify.CredentialKeyVersion)
	assert.Equal(t, record.CredentialFormatVersion, provider.lastVerify.CredentialFormatVersion)

	revoked, err := svc.Revoke(context.Background(), adminID, "default", "mock")
	require.NoError(t, err)
	require.NotNil(t, revoked)
	assert.Equal(t, ports.FiscalConnectionStateRevoked, revoked.State)
	assert.False(t, revoked.HasCredentials)
	assert.Nil(t, revoked.AccessExpiresAt)
	assert.NotNil(t, revoked.RevokedAt)

	record, err = repo.GetByScopeProvider(context.Background(), "default", "mock")
	require.NoError(t, err)
	require.NotNil(t, record)
	assert.Empty(t, record.CredentialCiphertext)
	assert.Empty(t, record.CredentialNonce)
	assert.Equal(t, ports.FiscalConnectionStateRevoked, record.State)
}

func TestConnectionServiceAuthorizationAndMissingCredentials(t *testing.T) {
	t.Parallel()

	cipher, err := fiscalcrypto.NewFiscalCredentialCipher("v1", bytes.Repeat([]byte{0x22}, 32))
	require.NoError(t, err)
	managerID, manager := newManagerUser(t)
	clientID, client := newClientUser(t)
	repo := &stubFiscalConnectionRepo{}
	svc := NewConnectionService(repo, &stubFiscalUserRepo{users: map[uuid.UUID]*domain.User{managerID: manager, clientID: client}}, &stubFiscalProvider{}, cipher, "v1")

	_, err = svc.StoreCredentials(context.Background(), clientID, ports.FiscalConnectionSetupRequest{ScopeKey: "default", ProviderKey: "mock", Credential: []byte("secret")})
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrPermissionDenied)

	_, err = svc.Verify(context.Background(), managerID, "default", "mock")
	require.Error(t, err)
	assert.ErrorIs(t, err, ports.ErrFiscalConnectionCredentialsMissing)
}

func TestConnectionServiceProviderFailureFailsClosed(t *testing.T) {
	t.Parallel()

	cipher, err := fiscalcrypto.NewFiscalCredentialCipher("v1", bytes.Repeat([]byte{0x33}, 32))
	require.NoError(t, err)
	adminID, admin := newManagerUser(t)
	provider := &stubFiscalProvider{verifyErr: ports.NewFiscalProviderError(ports.FiscalErrorClassTransient, "timeout", "temporary outage", nil)}
	repo := &stubFiscalConnectionRepo{}
	svc := NewConnectionService(repo, &stubFiscalUserRepo{users: map[uuid.UUID]*domain.User{adminID: admin}}, provider, cipher, "v1")

	_, err = svc.StoreCredentials(context.Background(), adminID, ports.FiscalConnectionSetupRequest{ScopeKey: "default", ProviderKey: "mock", Credential: []byte("secret")})
	require.NoError(t, err)

	_, err = svc.Verify(context.Background(), adminID, "default", "mock")
	require.Error(t, err)
	var providerErr *ports.FiscalProviderError
	require.True(t, errors.As(err, &providerErr))
	assert.Equal(t, ports.FiscalErrorClassTransient, providerErr.Class)

	record, err := repo.GetByScopeProvider(context.Background(), "default", "mock")
	require.NoError(t, err)
	require.NotNil(t, record)
	assert.Equal(t, ports.FiscalConnectionStateAuthorizing, record.State)
	assert.Nil(t, record.LastVerifiedAt)
	assert.Nil(t, record.ConnectedAt)
}
