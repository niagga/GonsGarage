package ports

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
)

// FiscalProvider executes normalized fiscal operations.
type FiscalProvider interface {
	VerifyConnection(ctx context.Context, req FiscalVerifyConnectionRequest) (FiscalVerifyConnectionResult, error)
	Issue(ctx context.Context, req FiscalIssueRequest) (FiscalIssueResult, error)
	Reconcile(ctx context.Context, req FiscalReconcileRequest) (FiscalReconcileResult, error)
	Void(ctx context.Context, req FiscalVoidRequest) (FiscalVoidResult, error)
	FetchArtifact(ctx context.Context, req FiscalFetchArtifactRequest) (FiscalFetchArtifactResult, error)
}

// FiscalErrorClass classifies provider-agnostic failures.
type FiscalErrorClass string

const (
	FiscalErrorClassValidation   FiscalErrorClass = "validation"
	FiscalErrorClassConflict     FiscalErrorClass = "conflict"
	FiscalErrorClassUnauthorized FiscalErrorClass = "unauthorized"
	FiscalErrorClassNotFound     FiscalErrorClass = "not_found"
	FiscalErrorClassTransient    FiscalErrorClass = "transient"
	FiscalErrorClassPermanent    FiscalErrorClass = "permanent"
	FiscalErrorClassUnsupported  FiscalErrorClass = "unsupported"
)

// IsValid reports whether the class is recognized.
func (c FiscalErrorClass) IsValid() bool {
	switch c {
	case FiscalErrorClassValidation,
		FiscalErrorClassConflict,
		FiscalErrorClassUnauthorized,
		FiscalErrorClassNotFound,
		FiscalErrorClassTransient,
		FiscalErrorClassPermanent,
		FiscalErrorClassUnsupported:
		return true
	default:
		return false
	}
}

// FiscalProviderError is the normalized error wrapper used by providers.
type FiscalProviderError struct {
	Class     FiscalErrorClass
	Code      string
	Message   string
	Retryable bool
	Cause     error
}

// Error implements error.
func (e *FiscalProviderError) Error() string {
	if e == nil {
		return "fiscal provider error"
	}
	parts := []string{"fiscal provider error"}
	if e.Class != "" {
		parts = append(parts, string(e.Class))
	}
	if e.Code != "" {
		parts = append(parts, e.Code)
	}
	if e.Message != "" {
		parts = append(parts, e.Message)
	}
	return strings.Join(parts, ": ")
}

// Unwrap exposes the wrapped cause.
func (e *FiscalProviderError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

// IsRetryable reports whether the failure may be retried safely.
func (e FiscalProviderError) IsRetryable() bool {
	return e.Retryable || e.Class == FiscalErrorClassTransient
}

// IsValid reports whether the class is set correctly.
func (e FiscalProviderError) IsValid() bool {
	return e.Class.IsValid()
}

// NewFiscalProviderError builds a normalized provider error.
func NewFiscalProviderError(class FiscalErrorClass, code, message string, cause error) *FiscalProviderError {
	return &FiscalProviderError{
		Class:     class,
		Code:      strings.TrimSpace(code),
		Message:   strings.TrimSpace(message),
		Retryable: class == FiscalErrorClassTransient,
		Cause:     cause,
	}
}

// FiscalOperationRequest contains the shared normalized request fields.
type FiscalOperationRequest struct {
	ConnectionID   uuid.UUID
	ProviderKey    string
	OperationKey   string
	CorrelationKey string
	Metadata       map[string]string
}

// Clone returns a copy with cloned metadata.
func (r FiscalOperationRequest) Clone() FiscalOperationRequest {
	return FiscalOperationRequest{
		ConnectionID:   r.ConnectionID,
		ProviderKey:    r.ProviderKey,
		OperationKey:   r.OperationKey,
		CorrelationKey: r.CorrelationKey,
		Metadata:       cloneStringMap(r.Metadata),
	}
}

// FiscalOperationResponse contains the shared normalized response fields.
type FiscalOperationResponse struct {
	ProviderKey       string
	OperationKey      string
	ProviderReference string
	CorrelationKey    string
	ObservedAt        time.Time
	Metadata          map[string]string
}

// Clone returns a copy with cloned metadata.
func (r FiscalOperationResponse) Clone() FiscalOperationResponse {
	return FiscalOperationResponse{
		ProviderKey:       r.ProviderKey,
		OperationKey:      r.OperationKey,
		ProviderReference: r.ProviderReference,
		CorrelationKey:    r.CorrelationKey,
		ObservedAt:        r.ObservedAt,
		Metadata:          cloneStringMap(r.Metadata),
	}
}

// FiscalVerifyConnectionRequest verifies the stored credential against a provider.
type FiscalVerifyConnectionRequest struct {
	FiscalOperationRequest
	CredentialCiphertext    []byte
	CredentialNonce         []byte
	CredentialKeyVersion    string
	CredentialFormatVersion int
}

// FiscalVerifyConnectionResult reports the normalized connection state.
type FiscalVerifyConnectionResult struct {
	FiscalOperationResponse
	State           FiscalConnectionState
	GrantedScopes   []string
	AccessExpiresAt *time.Time
	ConnectedAt     *time.Time
	RevokedAt       *time.Time
	OrganizationRef string
}

// FiscalIssueRequest asks a provider to issue a fiscal document.
type FiscalIssueRequest struct {
	FiscalOperationRequest
	DocumentID uuid.UUID
	Payload    []byte
}

// FiscalIssueResult reports the normalized issue outcome.
type FiscalIssueResult struct {
	FiscalOperationResponse
	DocumentID     uuid.UUID
	ArtifactID     uuid.UUID
	ProviderNumber string
	IssuedAt       *time.Time
}

// FiscalReconcileRequest asks a provider to reconcile a prior issue.
type FiscalReconcileRequest struct {
	FiscalOperationRequest
	DocumentID        uuid.UUID
	ProviderReference string
}

// FiscalReconcileResult reports the normalized reconciliation outcome.
type FiscalReconcileResult struct {
	FiscalOperationResponse
	DocumentID uuid.UUID
	Matched    bool
	ResolvedAt *time.Time
}

// FiscalVoidRequest asks a provider to void an issued document.
type FiscalVoidRequest struct {
	FiscalOperationRequest
	DocumentID        uuid.UUID
	ProviderReference string
	Reason            string
}

// FiscalVoidResult reports the normalized void outcome.
type FiscalVoidResult struct {
	FiscalOperationResponse
	DocumentID uuid.UUID
	VoidedAt   *time.Time
}

// FiscalFetchArtifactRequest asks a provider to fetch a generated artifact.
type FiscalFetchArtifactRequest struct {
	FiscalOperationRequest
	ArtifactID uuid.UUID
}

// FiscalFetchArtifactResult reports the fetched artifact payload.
type FiscalFetchArtifactResult struct {
	FiscalOperationResponse
	ArtifactID uuid.UUID
	Content    []byte
	MediaType  string
	Sha256     string
	FetchedAt  *time.Time
}

func cloneStringMap(values map[string]string) map[string]string {
	if values == nil {
		return nil
	}
	cloned := make(map[string]string, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}
