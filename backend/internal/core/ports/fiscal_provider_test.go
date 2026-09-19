package ports

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type noopFiscalProvider struct{}

func (noopFiscalProvider) VerifyConnection(context.Context, FiscalVerifyConnectionRequest) (FiscalVerifyConnectionResult, error) {
	return FiscalVerifyConnectionResult{}, nil
}

func (noopFiscalProvider) Issue(context.Context, FiscalIssueRequest) (FiscalIssueResult, error) {
	return FiscalIssueResult{}, nil
}

func (noopFiscalProvider) Reconcile(context.Context, FiscalReconcileRequest) (FiscalReconcileResult, error) {
	return FiscalReconcileResult{}, nil
}

func (noopFiscalProvider) Void(context.Context, FiscalVoidRequest) (FiscalVoidResult, error) {
	return FiscalVoidResult{}, nil
}

func (noopFiscalProvider) FetchArtifact(context.Context, FiscalFetchArtifactRequest) (FiscalFetchArtifactResult, error) {
	return FiscalFetchArtifactResult{}, nil
}

var _ FiscalProvider = noopFiscalProvider{}

func TestFiscalErrorClassInvariants(t *testing.T) {
	t.Parallel()

	valid := []FiscalErrorClass{
		FiscalErrorClassValidation,
		FiscalErrorClassConflict,
		FiscalErrorClassUnauthorized,
		FiscalErrorClassNotFound,
		FiscalErrorClassTransient,
		FiscalErrorClassPermanent,
		FiscalErrorClassUnsupported,
	}
	for _, class := range valid {
		assert.True(t, class.IsValid(), class)
	}
	assert.False(t, FiscalErrorClass("unknown").IsValid())
}

func TestFiscalProviderErrorFormattingAndWrapping(t *testing.T) {
	t.Parallel()

	cause := errors.New("upstream timeout")
	err := NewFiscalProviderError(FiscalErrorClassTransient, "timeout", "provider did not respond", cause)
	require.NotNil(t, err)
	assert.Equal(t, FiscalErrorClassTransient, err.Class)
	assert.Equal(t, "timeout", err.Code)
	assert.Equal(t, "provider did not respond", err.Message)
	assert.True(t, err.IsRetryable())
	assert.True(t, err.IsValid())
	assert.ErrorIs(t, err, cause)
	assert.Contains(t, err.Error(), string(FiscalErrorClassTransient))
	assert.Contains(t, err.Error(), "timeout")
}

func TestFiscalOperationRequestAndResponseClone(t *testing.T) {
	t.Parallel()

	request := FiscalOperationRequest{
		ConnectionID:   uuid.New(),
		ProviderKey:    "mock",
		OperationKey:   "issue-01",
		CorrelationKey: "corr-01",
		Metadata:       map[string]string{"a": "1"},
	}
	requestClone := request.Clone()
	request.Metadata["a"] = "2"
	assert.Equal(t, "1", requestClone.Metadata["a"])

	now := time.Date(2026, 1, 2, 16, 0, 0, 0, time.UTC)
	response := FiscalOperationResponse{
		ProviderKey:       "mock",
		OperationKey:      "issue-01",
		ProviderReference: "ref-1",
		CorrelationKey:    "corr-01",
		ObservedAt:        now,
		Metadata:          map[string]string{"b": "2"},
	}
	responseClone := response.Clone()
	response.Metadata["b"] = "3"
	assert.Equal(t, "2", responseClone.Metadata["b"])
	assert.Equal(t, now, responseClone.ObservedAt)
}
