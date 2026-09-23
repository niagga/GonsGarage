package fiscal

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/gaston-garcia-cegid/gonsgarage/internal/core/ports"
	"github.com/gaston-garcia-cegid/gonsgarage/internal/domain"
	"github.com/gaston-garcia-cegid/gonsgarage/internal/integration/fiscal/cloudware"
	fiscalcrypto "github.com/gaston-garcia-cegid/gonsgarage/internal/platform/crypto"
	"github.com/google/uuid"
)

const (
	fiscalCredentialFormatVersion = 1
	oauthStateBytes               = 16 // 128-bit
	oauthStateTTL                 = 10 * time.Minute
)

// ConnectionService manages the provider-neutral connection lifecycle.
type ConnectionService struct {
	repo                 ports.FiscalConnectionRepository
	userRepo             ports.UserRepository
	provider             ports.FiscalProvider
	cipher               *fiscalcrypto.FiscalCredentialCipher
	credentialKeyVersion string
	oauthStore           ports.FiscalOAuthStore
	tokenExchanger       ports.FiscalOAuthTokenExchanger
	authorizeURL         string
	oauthClientID        string
	gateCatalog          *cloudware.GateCatalog
}

var _ ports.FiscalConnectionService = (*ConnectionService)(nil)

// NewConnectionService builds the manager/admin connection service.
func NewConnectionService(repo ports.FiscalConnectionRepository, userRepo ports.UserRepository, provider ports.FiscalProvider, cipher *fiscalcrypto.FiscalCredentialCipher, credentialKeyVersion string) *ConnectionService {
	return &ConnectionService{
		repo:                 repo,
		userRepo:             userRepo,
		provider:             provider,
		cipher:               cipher,
		credentialKeyVersion: strings.TrimSpace(credentialKeyVersion),
		gateCatalog:          cloudware.NewGateCatalog("production"),
	}
}

// WithOAuth wires hashed-state OAuth connect/callback/refresh support.
func (s *ConnectionService) WithOAuth(store ports.FiscalOAuthStore, exchanger ports.FiscalOAuthTokenExchanger, authorizeURL, clientID string) *ConnectionService {
	if s == nil {
		return nil
	}
	s.oauthStore = store
	s.tokenExchanger = exchanger
	s.authorizeURL = strings.TrimSpace(authorizeURL)
	s.oauthClientID = strings.TrimSpace(clientID)
	return s
}

// WithGateCatalog replaces the default Cloudware enablement catalog (tests/wiring).
func (s *ConnectionService) WithGateCatalog(catalog *cloudware.GateCatalog) *ConnectionService {
	if s == nil {
		return nil
	}
	if catalog == nil {
		catalog = cloudware.NewGateCatalog("production")
	}
	s.gateCatalog = catalog
	return s
}

func (s *ConnectionService) Status(ctx context.Context, requestingUserID uuid.UUID, scopeKey, providerKey string) (*ports.FiscalConnectionStatus, error) {
	if err := s.requireManager(ctx, requestingUserID); err != nil {
		return nil, err
	}
	record, err := s.load(ctx, scopeKey, providerKey)
	if err != nil {
		return nil, err
	}
	if record == nil {
		return &ports.FiscalConnectionStatus{
			ScopeKey:    normalizeScopeKey(scopeKey),
			ProviderKey: normalizeProviderKey(providerKey),
			State:       ports.FiscalConnectionStateDisconnected,
		}, nil
	}
	return toStatus(record), nil
}

func (s *ConnectionService) StoreCredentials(ctx context.Context, requestingUserID uuid.UUID, req ports.FiscalConnectionSetupRequest) (*ports.FiscalConnectionStatus, error) {
	if err := s.requireManager(ctx, requestingUserID); err != nil {
		return nil, err
	}
	if len(req.Credential) == 0 {
		return nil, ports.ErrFiscalConnectionCredentialsMissing
	}
	if s.disabled() {
		return nil, ports.ErrFiscalConnectionUnavailable
	}
	scopeKey := normalizeScopeKey(req.ScopeKey)
	providerKey := normalizeProviderKey(req.ProviderKey)
	if scopeKey == "" || providerKey == "" {
		return nil, fmt.Errorf("scope and provider keys are required")
	}
	if strings.TrimSpace(s.credentialKeyVersion) == "" {
		return nil, ports.ErrFiscalConnectionUnavailable
	}
	record, err := s.load(ctx, scopeKey, providerKey)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	if record == nil {
		record = &ports.FiscalConnectionRecord{ID: uuid.New(), ScopeKey: scopeKey, ProviderKey: providerKey, CreatedBy: requestingUserID, CreatedAt: now, Version: 1}
	}
	binding := fiscalcrypto.FiscalCredentialBinding{ConnectionID: record.ID, ProviderKey: providerKey, ScopeKey: scopeKey}
	envelope, err := s.cipher.EncryptWithBinding(append([]byte(nil), req.Credential...), binding)
	if err != nil {
		return nil, err
	}
	updatedBy := requestingUserID
	record.ScopeKey = scopeKey
	record.ProviderKey = providerKey
	record.State = ports.FiscalConnectionStateAuthorizing
	record.ProviderReference = ""
	record.GrantedScopes = append([]string(nil), req.GrantedScopes...)
	record.AccessExpiresAt = nil
	record.CredentialCiphertext = append([]byte(nil), envelope.Ciphertext...)
	record.CredentialNonce = append([]byte(nil), envelope.Nonce...)
	record.CredentialKeyVersion = s.credentialKeyVersion
	record.CredentialFormatVersion = fiscalCredentialFormatVersion
	record.LastVerifiedAt = nil
	record.ConnectedAt = nil
	record.RevokedAt = nil
	record.UpdatedBy = &updatedBy
	record.UpdatedAt = now
	if record.CreatedAt.IsZero() {
		record.CreatedAt = now
	}
	if err := s.repo.Save(ctx, record); err != nil {
		return nil, err
	}
	return s.Status(ctx, requestingUserID, scopeKey, providerKey)
}

func (s *ConnectionService) Verify(ctx context.Context, requestingUserID uuid.UUID, scopeKey, providerKey string) (*ports.FiscalConnectionStatus, error) {
	if err := s.requireManager(ctx, requestingUserID); err != nil {
		return nil, err
	}
	if s.disabled() {
		return nil, ports.ErrFiscalConnectionUnavailable
	}
	record, err := s.mustLoadRecord(ctx, scopeKey, providerKey)
	if err != nil {
		return nil, err
	}
	if len(record.CredentialCiphertext) == 0 || len(record.CredentialNonce) == 0 {
		return nil, ports.ErrFiscalConnectionCredentialsMissing
	}
	binding := fiscalcrypto.FiscalCredentialBinding{ConnectionID: record.ID, ProviderKey: record.ProviderKey, ScopeKey: record.ScopeKey}
	envelope := fiscalcrypto.FiscalCredentialEnvelope{
		FormatVersion: record.CredentialFormatVersion,
		KeyVersion:    record.CredentialKeyVersion,
		Nonce:         append([]byte(nil), record.CredentialNonce...),
		Ciphertext:    append([]byte(nil), record.CredentialCiphertext...),
	}
	if _, err := s.cipher.DecryptWithBinding(envelope, binding); err != nil {
		// Legacy envelopes may use empty binding from pre-WU10 rows.
		if _, legacyErr := s.cipher.Decrypt(envelope); legacyErr != nil {
			return nil, err
		}
	}
	observedAt := time.Now().UTC()
	request := ports.FiscalVerifyConnectionRequest{
		FiscalOperationRequest: ports.FiscalOperationRequest{
			ConnectionID:   record.ID,
			ProviderKey:    record.ProviderKey,
			OperationKey:   buildOperationKey(scopeKey, providerKey, "verify"),
			CorrelationKey: buildCorrelationKey(record.ID, providerKey, "verify"),
			Metadata: map[string]string{
				"scopeKey":             record.ScopeKey,
				"providerKey":          record.ProviderKey,
				"connectionId":         record.ID.String(),
				"credentialKeyVersion": record.CredentialKeyVersion,
				"hasCredentials":       "true",
			},
		},
		CredentialCiphertext:    append([]byte(nil), record.CredentialCiphertext...),
		CredentialNonce:         append([]byte(nil), record.CredentialNonce...),
		CredentialKeyVersion:    record.CredentialKeyVersion,
		CredentialFormatVersion: record.CredentialFormatVersion,
	}
	result, err := s.provider.VerifyConnection(ctx, request)
	if err != nil {
		return s.failVerification(ctx, requestingUserID, record, err)
	}
	if !result.State.IsValid() {
		return s.failVerification(ctx, requestingUserID, record, fmt.Errorf("provider returned invalid connection state %q", result.State))
	}
	record.State = result.State
	if len(result.GrantedScopes) > 0 {
		record.GrantedScopes = append([]string(nil), result.GrantedScopes...)
	}
	if result.AccessExpiresAt != nil {
		record.AccessExpiresAt = timePtrClone(result.AccessExpiresAt)
	}
	if result.ConnectedAt != nil {
		record.ConnectedAt = timePtrClone(result.ConnectedAt)
	} else if record.State == ports.FiscalConnectionStateConnected {
		record.ConnectedAt = &observedAt
	}
	if record.State == ports.FiscalConnectionStateRevoked {
		record.RevokedAt = timePtrClone(result.RevokedAt)
	} else {
		record.RevokedAt = nil
	}
	record.ProviderReference = result.OrganizationRef
	record.LastVerifiedAt = &observedAt
	record.UpdatedBy = &requestingUserID
	record.UpdatedAt = observedAt
	if err := s.repo.Save(ctx, record); err != nil {
		return nil, err
	}
	return s.Status(ctx, requestingUserID, scopeKey, providerKey)
}

func (s *ConnectionService) Revoke(ctx context.Context, requestingUserID uuid.UUID, scopeKey, providerKey string) (*ports.FiscalConnectionStatus, error) {
	if err := s.requireManager(ctx, requestingUserID); err != nil {
		return nil, err
	}
	if s.disabled() {
		return nil, ports.ErrFiscalConnectionUnavailable
	}
	record, err := s.load(ctx, scopeKey, providerKey)
	if err != nil {
		return nil, err
	}
	if record == nil {
		return &ports.FiscalConnectionStatus{
			ScopeKey:    normalizeScopeKey(scopeKey),
			ProviderKey: normalizeProviderKey(providerKey),
			State:       ports.FiscalConnectionStateDisconnected,
		}, nil
	}
	now := time.Now().UTC()
	record.State = ports.FiscalConnectionStateRevoked
	record.ProviderReference = ""
	record.GrantedScopes = nil
	record.AccessExpiresAt = nil
	record.CredentialCiphertext = nil
	record.CredentialNonce = nil
	record.CredentialKeyVersion = ""
	record.CredentialFormatVersion = 0
	record.LastVerifiedAt = nil
	record.ConnectedAt = nil
	record.RevokedAt = &now
	record.UpdatedBy = &requestingUserID
	record.UpdatedAt = now
	if err := s.repo.Save(ctx, record); err != nil {
		return nil, err
	}
	return s.Status(ctx, requestingUserID, scopeKey, providerKey)
}

// StartOAuthConnect creates/reuses the scoped connection and a one-time hashed OAuth state.
func (s *ConnectionService) StartOAuthConnect(ctx context.Context, requestingUserID uuid.UUID, req ports.FiscalOAuthStartRequest) (*ports.FiscalOAuthStartResult, error) {
	if err := s.requireManager(ctx, requestingUserID); err != nil {
		return nil, err
	}
	if s.oauthStore == nil || s.tokenExchanger == nil || s.authorizeURL == "" || s.oauthClientID == "" {
		return nil, ports.ErrFiscalOAuthUnavailable
	}
	scopeKey := normalizeScopeKey(req.ScopeKey)
	providerKey := normalizeProviderKey(req.ProviderKey)
	redirectURI := strings.TrimSpace(req.RedirectURI)
	if scopeKey == "" || providerKey == "" || redirectURI == "" {
		return nil, fmt.Errorf("scope, provider, and redirect URI are required")
	}
	record, err := s.load(ctx, scopeKey, providerKey)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	if record == nil {
		record = &ports.FiscalConnectionRecord{
			ID: uuid.New(), ScopeKey: scopeKey, ProviderKey: providerKey,
			CreatedBy: requestingUserID, CreatedAt: now, Version: 1,
		}
	}
	record.State = ports.FiscalConnectionStateAuthorizing
	record.UpdatedBy = &requestingUserID
	record.UpdatedAt = now
	if err := s.repo.Save(ctx, record); err != nil {
		return nil, err
	}

	rawState, err := randomState()
	if err != nil {
		return nil, err
	}
	expiresAt := now.Add(oauthStateTTL)
	auth := &ports.FiscalOAuthAuthorization{
		ID:           uuid.New(),
		ConnectionID: record.ID,
		ActorID:      requestingUserID,
		StateSHA256:  hashState(rawState),
		RedirectURI:  redirectURI,
		ExpiresAt:    expiresAt,
		CreatedAt:    now,
	}
	if err := s.oauthStore.SaveOAuthAuthorization(ctx, auth); err != nil {
		return nil, err
	}
	authURL, err := buildAuthorizeURL(s.authorizeURL, s.oauthClientID, redirectURI, rawState)
	if err != nil {
		return nil, err
	}
	status := toStatus(record)
	return &ports.FiscalOAuthStartResult{
		Status:           status,
		AuthorizationURL: authURL,
		RawState:         rawState,
		ExpiresAt:        expiresAt,
	}, nil
}

// CompleteOAuthCallback consumes one-time state before exchanging the authorization code.
func (s *ConnectionService) CompleteOAuthCallback(ctx context.Context, req ports.FiscalOAuthCallbackRequest) (*ports.FiscalConnectionStatus, error) {
	if s.oauthStore == nil || s.tokenExchanger == nil || s.cipher == nil {
		return nil, ports.ErrFiscalOAuthUnavailable
	}
	rawState := strings.TrimSpace(req.State)
	code := strings.TrimSpace(req.Code)
	if rawState == "" || code == "" {
		return nil, ports.ErrFiscalOAuthStateInvalid
	}
	now := time.Now().UTC()
	auth, err := s.oauthStore.ConsumeOAuthAuthorization(ctx, hashState(rawState), now)
	if err != nil {
		return nil, err
	}
	record, err := s.repo.GetByID(ctx, auth.ConnectionID)
	if err != nil {
		return nil, err
	}
	if record == nil {
		return nil, ports.ErrFiscalConnectionCredentialsMissing
	}
	tokens, err := s.tokenExchanger.ExchangeAuthorizationCode(ctx, code, auth.RedirectURI)
	if err != nil {
		return nil, err
	}
	if err := s.persistTokenSet(ctx, record, auth.ActorID, tokens, now); err != nil {
		return nil, err
	}
	return toStatus(record), nil
}

// RefreshCredentials refreshes the access token; failures become action_required without leaking secrets.
func (s *ConnectionService) RefreshCredentials(ctx context.Context, requestingUserID uuid.UUID, scopeKey, providerKey string) (*ports.FiscalConnectionStatus, error) {
	if err := s.requireManager(ctx, requestingUserID); err != nil {
		return nil, err
	}
	if s.tokenExchanger == nil || s.cipher == nil {
		return nil, ports.ErrFiscalOAuthUnavailable
	}
	record, err := s.mustLoadRecord(ctx, scopeKey, providerKey)
	if err != nil {
		return nil, err
	}
	tokens, err := s.decryptTokenSet(record)
	if err != nil {
		return nil, err
	}
	refreshed, err := s.tokenExchanger.RefreshAccessToken(ctx, tokens.RefreshToken)
	if err != nil {
		now := time.Now().UTC()
		record.State = ports.FiscalConnectionStateActionNeeded
		record.UpdatedBy = &requestingUserID
		record.UpdatedAt = now
		_ = s.repo.Save(ctx, record)
		return toStatus(record), err
	}
	now := time.Now().UTC()
	if err := s.persistTokenSet(ctx, record, requestingUserID, refreshed, now); err != nil {
		return nil, err
	}
	return toStatus(record), nil
}

// Readiness returns Cloudware enablement gate status for managers/admins.
func (s *ConnectionService) Readiness(ctx context.Context, requestingUserID uuid.UUID) (*ports.FiscalReadinessReport, error) {
	if err := s.requireManager(ctx, requestingUserID); err != nil {
		return nil, err
	}
	if s.gateCatalog == nil {
		s.gateCatalog = cloudware.NewGateCatalog("production")
	}
	report := s.gateCatalog.Readiness()
	out := &ports.FiscalReadinessReport{
		Ready:                 report.Ready,
		ATCommunicationStatus: report.ATCommunicationStatus,
		Gates:                 make([]ports.FiscalEnablementGateView, 0, len(report.Gates)),
	}
	for _, g := range report.Gates {
		status := string(g.Status)
		if status == string(cloudware.GateSatisfied) {
			status = "passed"
		}
		out.Gates = append(out.Gates, ports.FiscalEnablementGateView{
			Name:      g.Key,
			Status:    status,
			Guidance:  g.Guidance,
			Rationale: g.Rationale,
		})
	}
	return out, nil
}

func (s *ConnectionService) persistTokenSet(ctx context.Context, record *ports.FiscalConnectionRecord, actorID uuid.UUID, tokens ports.FiscalOAuthTokenSet, now time.Time) error {
	raw, err := json.Marshal(tokens)
	if err != nil {
		return err
	}
	binding := fiscalcrypto.FiscalCredentialBinding{
		ConnectionID: record.ID,
		ProviderKey:  record.ProviderKey,
		ScopeKey:     record.ScopeKey,
	}
	envelope, err := s.cipher.EncryptWithBinding(raw, binding)
	if err != nil {
		return err
	}
	record.CredentialCiphertext = append([]byte(nil), envelope.Ciphertext...)
	record.CredentialNonce = append([]byte(nil), envelope.Nonce...)
	record.CredentialKeyVersion = envelope.KeyVersion
	record.CredentialFormatVersion = envelope.FormatVersion
	record.State = ports.FiscalConnectionStateConnected
	record.ProviderReference = tokens.Organization
	record.GrantedScopes = append([]string(nil), tokens.Scopes...)
	if tokens.ExpiresIn > 0 {
		exp := now.Add(time.Duration(tokens.ExpiresIn) * time.Second)
		record.AccessExpiresAt = &exp
	}
	record.ConnectedAt = &now
	record.LastVerifiedAt = &now
	record.RevokedAt = nil
	record.UpdatedBy = &actorID
	record.UpdatedAt = now
	return s.repo.Save(ctx, record)
}

func (s *ConnectionService) decryptTokenSet(record *ports.FiscalConnectionRecord) (ports.FiscalOAuthTokenSet, error) {
	envelope := fiscalcrypto.FiscalCredentialEnvelope{
		FormatVersion: record.CredentialFormatVersion,
		KeyVersion:    record.CredentialKeyVersion,
		Nonce:         append([]byte(nil), record.CredentialNonce...),
		Ciphertext:    append([]byte(nil), record.CredentialCiphertext...),
	}
	binding := fiscalcrypto.FiscalCredentialBinding{
		ConnectionID: record.ID,
		ProviderKey:  record.ProviderKey,
		ScopeKey:     record.ScopeKey,
	}
	raw, err := s.cipher.DecryptWithBinding(envelope, binding)
	if err != nil {
		return ports.FiscalOAuthTokenSet{}, err
	}
	var tokens ports.FiscalOAuthTokenSet
	if err := json.Unmarshal(raw, &tokens); err != nil {
		return ports.FiscalOAuthTokenSet{}, err
	}
	return tokens, nil
}

func (s *ConnectionService) failVerification(ctx context.Context, requestingUserID uuid.UUID, record *ports.FiscalConnectionRecord, cause error) (*ports.FiscalConnectionStatus, error) {
	if record == nil {
		return nil, cause
	}
	now := time.Now().UTC()
	record.LastVerifiedAt = nil
	record.ConnectedAt = nil
	record.RevokedAt = nil
	record.AccessExpiresAt = nil
	record.ProviderReference = ""
	if isRetryableProviderError(cause) {
		record.State = ports.FiscalConnectionStateAuthorizing
	} else {
		record.State = ports.FiscalConnectionStateActionNeeded
	}
	record.UpdatedBy = &requestingUserID
	record.UpdatedAt = now
	if err := s.repo.Save(ctx, record); err != nil {
		return nil, err
	}
	return nil, cause
}

func (s *ConnectionService) requireManager(ctx context.Context, requestingUserID uuid.UUID) error {
	if s.disabled() {
		return ports.ErrFiscalConnectionUnavailable
	}
	if s.userRepo == nil {
		return ports.ErrFiscalConnectionUnavailable
	}
	user, err := s.userRepo.GetByID(ctx, requestingUserID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return domain.ErrUserNotFound
	}
	if !user.CanManageUsers() {
		return domain.ErrPermissionDenied
	}
	return nil
}

func (s *ConnectionService) load(ctx context.Context, scopeKey, providerKey string) (*ports.FiscalConnectionRecord, error) {
	if s.disabled() {
		return nil, ports.ErrFiscalConnectionUnavailable
	}
	if s.repo == nil {
		return nil, ports.ErrFiscalConnectionUnavailable
	}
	return s.repo.GetByScopeProvider(ctx, normalizeScopeKey(scopeKey), normalizeProviderKey(providerKey))
}

func (s *ConnectionService) mustLoadRecord(ctx context.Context, scopeKey, providerKey string) (*ports.FiscalConnectionRecord, error) {
	record, err := s.load(ctx, scopeKey, providerKey)
	if err != nil {
		return nil, err
	}
	if record == nil {
		return nil, ports.ErrFiscalConnectionCredentialsMissing
	}
	return record, nil
}

func (s *ConnectionService) disabled() bool {
	return s == nil || s.repo == nil || s.userRepo == nil || s.provider == nil || s.cipher == nil || strings.TrimSpace(s.credentialKeyVersion) == ""
}

func toStatus(record *ports.FiscalConnectionRecord) *ports.FiscalConnectionStatus {
	if record == nil {
		return nil
	}
	return &ports.FiscalConnectionStatus{
		ID:                      record.ID,
		ScopeKey:                record.ScopeKey,
		ProviderKey:             record.ProviderKey,
		State:                   record.State,
		ProviderReference:       record.ProviderReference,
		GrantedScopes:           append([]string(nil), record.GrantedScopes...),
		AccessExpiresAt:         timePtrClone(record.AccessExpiresAt),
		LastVerifiedAt:          timePtrClone(record.LastVerifiedAt),
		ConnectedAt:             timePtrClone(record.ConnectedAt),
		RevokedAt:               timePtrClone(record.RevokedAt),
		CreatedBy:               record.CreatedBy,
		UpdatedBy:               uuidPtrClone(record.UpdatedBy),
		CredentialKeyVersion:    record.CredentialKeyVersion,
		CredentialFormatVersion: record.CredentialFormatVersion,
		HasCredentials:          len(record.CredentialCiphertext) > 0 && len(record.CredentialNonce) > 0,
		CreatedAt:               record.CreatedAt,
		UpdatedAt:               record.UpdatedAt,
		Version:                 record.Version,
	}
}

func normalizeScopeKey(scopeKey string) string {
	return strings.TrimSpace(scopeKey)
}

func normalizeProviderKey(providerKey string) string {
	return strings.TrimSpace(providerKey)
}

func buildOperationKey(scopeKey, providerKey, operation string) string {
	return strings.Join([]string{normalizeScopeKey(scopeKey), normalizeProviderKey(providerKey), operation}, ":")
}

func buildCorrelationKey(id uuid.UUID, providerKey, operation string) string {
	return strings.Join([]string{id.String(), normalizeProviderKey(providerKey), operation}, ":")
}

func isRetryableProviderError(err error) bool {
	var providerErr *ports.FiscalProviderError
	if !errors.As(err, &providerErr) {
		return false
	}
	return providerErr.IsRetryable()
}

func timePtrClone(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	clone := *value
	return &clone
}

func uuidPtrClone(value *uuid.UUID) *uuid.UUID {
	if value == nil {
		return nil
	}
	clone := *value
	return &clone
}

func randomState() (string, error) {
	buf := make([]byte, oauthStateBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func hashState(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func buildAuthorizeURL(base, clientID, redirectURI, state string) (string, error) {
	u, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Set("response_type", "code")
	q.Set("client_id", clientID)
	q.Set("redirect_uri", redirectURI)
	q.Set("state", state)
	u.RawQuery = q.Encode()
	return u.String(), nil
}
