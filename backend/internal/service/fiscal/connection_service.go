package fiscal

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gaston-garcia-cegid/gonsgarage/internal/core/ports"
	"github.com/gaston-garcia-cegid/gonsgarage/internal/domain"
	fiscalcrypto "github.com/gaston-garcia-cegid/gonsgarage/internal/platform/crypto"
	"github.com/google/uuid"
)

const fiscalCredentialFormatVersion = 1

// ConnectionService manages the provider-neutral connection lifecycle.
type ConnectionService struct {
	repo                 ports.FiscalConnectionRepository
	userRepo             ports.UserRepository
	provider             ports.FiscalProvider
	cipher               *fiscalcrypto.FiscalCredentialCipher
	credentialKeyVersion string
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
	}
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
	envelope, err := s.cipher.Encrypt(append([]byte(nil), req.Credential...))
	if err != nil {
		return nil, err
	}
	record, err := s.load(ctx, scopeKey, providerKey)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	if record == nil {
		record = &ports.FiscalConnectionRecord{ID: uuid.New(), ScopeKey: scopeKey, ProviderKey: providerKey, CreatedBy: requestingUserID, CreatedAt: now, Version: 1}
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
	envelope := fiscalcrypto.FiscalCredentialEnvelope{
		FormatVersion: record.CredentialFormatVersion,
		KeyVersion:    record.CredentialKeyVersion,
		Nonce:         append([]byte(nil), record.CredentialNonce...),
		Ciphertext:    append([]byte(nil), record.CredentialCiphertext...),
	}
	if _, err := s.cipher.Decrypt(envelope); err != nil {
		return nil, err
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
