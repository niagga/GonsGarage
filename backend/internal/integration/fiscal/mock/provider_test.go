package mock

import (
	"context"
	"errors"
	"testing"

	"github.com/gaston-garcia-cegid/gonsgarage/internal/core/ports"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVerifyConnectionExpiredAndSuccess(t *testing.T) {
	t.Parallel()
	registry := NewScenarioRegistry(map[string]Scenario{
		"verify-ok":      ScenarioSuccess,
		"verify-expired": ScenarioExpiredConnection,
	})
	provider, err := NewProvider(Options{AppEnv: "development", Registry: registry, Store: NewMemoryOperationStore()})
	require.NoError(t, err)

	ok, err := provider.VerifyConnection(context.Background(), ports.FiscalVerifyConnectionRequest{
		FiscalOperationRequest: ports.FiscalOperationRequest{
			ConnectionID: uuid.MustParse("11111111-1111-1111-1111-111111111111"),
			ProviderKey:  "mock", OperationKey: "verify-ok", CorrelationKey: "c1",
		},
	})
	require.NoError(t, err)
	assert.True(t, ok.State.IsReady())
	assert.NotEmpty(t, ok.GrantedScopes)

	_, err = provider.VerifyConnection(context.Background(), ports.FiscalVerifyConnectionRequest{
		FiscalOperationRequest: ports.FiscalOperationRequest{
			ConnectionID: uuid.MustParse("11111111-1111-1111-1111-111111111111"),
			ProviderKey:  "mock", OperationKey: "verify-expired",
		},
	})
	var providerErr *ports.FiscalProviderError
	require.True(t, errors.As(err, &providerErr))
	assert.Equal(t, ports.FiscalErrorClassUnauthorized, providerErr.Class)
}
