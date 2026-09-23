package fiscalartifact_test

import (
	"testing"

	"github.com/gaston-garcia-cegid/gonsgarage/internal/core/ports"
	"github.com/gaston-garcia-cegid/gonsgarage/internal/platform/fiscalartifact"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProductionReadiness_FailClosedUntilObjectBackendPasses(t *testing.T) {
	local := fiscalartifact.ProductionReadiness{Backend: fiscalartifact.BackendLocal}
	require.ErrorIs(t, fiscalartifact.AssertProductionIssuanceAllowed(local), ports.ErrFiscalArtifactBackendNotReady)
	assert.Contains(t, local.Missing(), "backend_not_object")

	partial := fiscalartifact.ProductionReadiness{
		Backend: fiscalartifact.BackendObject, PrivateACL: true, EncryptionAtRest: true,
	}
	err := fiscalartifact.AssertProductionIssuanceAllowed(partial)
	require.ErrorIs(t, err, ports.ErrFiscalArtifactBackendNotReady)
	assert.Contains(t, err.Error(), "retention")

	ready := fiscalartifact.ProductionReadiness{
		Backend: fiscalartifact.BackendObject, PrivateACL: true, EncryptionAtRest: true,
		RetentionConfigured: true, BackupRestoreEvidenced: true, AccessLoggingEnabled: true,
	}
	require.NoError(t, fiscalartifact.AssertProductionIssuanceAllowed(ready))

	store := fiscalartifact.NewObjectStore("https://objects.internal", "fiscal-private", ready)
	require.NoError(t, store.EnsureReady())

	unready := fiscalartifact.NewObjectStore("https://objects.internal", "fiscal-private", fiscalartifact.ProductionReadiness{})
	require.ErrorIs(t, unready.EnsureReady(), ports.ErrFiscalArtifactBackendNotReady)
}
