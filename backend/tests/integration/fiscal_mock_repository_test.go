package integration

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	migrations "github.com/gaston-garcia-cegid/gonsgarage/cmd/migrate/migrations"
	"github.com/gaston-garcia-cegid/gonsgarage/internal/core/ports"
	fiscalmock "github.com/gaston-garcia-cegid/gonsgarage/internal/integration/fiscal/mock"
)

func TestFiscalMockOperationsPersistAcrossProviderReload(t *testing.T) {
	db := fiscalTestDB(t)
	_, err := db.Exec(`
CREATE TABLE users (id uuid PRIMARY KEY);
CREATE TABLE invoices (
 id uuid PRIMARY KEY, customer_id uuid NOT NULL REFERENCES users(id), amount double precision NOT NULL,
 status varchar(40) NOT NULL DEFAULT 'open', notes text, created_at timestamptz DEFAULT now(), updated_at timestamptz DEFAULT now()
);`)
	require.NoError(t, err)
	require.NoError(t, migrations.Run(db, migrationDir(t)))

	store := fiscalmock.NewPostgresOperationStore(db)
	registry := fiscalmock.NewScenarioRegistry(map[string]fiscalmock.Scenario{
		"pg-op-ft": fiscalmock.ScenarioSuccess,
		"pg-op-fr": fiscalmock.ScenarioSuccess,
	})
	provider, err := fiscalmock.NewProvider(fiscalmock.Options{
		AppEnv: "development", Registry: registry, Store: store,
	})
	require.NoError(t, err)

	ftReq := ports.FiscalIssueRequest{
		FiscalOperationRequest: ports.FiscalOperationRequest{
			ConnectionID: uuid.MustParse("33333333-3333-3333-3333-333333333333"),
			ProviderKey:  "mock", OperationKey: "pg-op-ft", CorrelationKey: "pg-corr-ft",
			Metadata: map[string]string{"documentKind": "FT"},
		},
		DocumentID: uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"),
		Payload:    []byte(`{"kind":"FT","pg":1}`),
	}
	first, err := provider.Issue(context.Background(), ftReq)
	require.NoError(t, err)
	require.True(t, len(first.ProviderReference) > 5 && first.ProviderReference[:5] == "MOCK-")

	art1, err := provider.FetchArtifact(context.Background(), ports.FiscalFetchArtifactRequest{
		FiscalOperationRequest: ftReq.FiscalOperationRequest, ArtifactID: first.ArtifactID,
	})
	require.NoError(t, err)
	require.Contains(t, string(art1.Content), fiscalmock.MockPDFLabel)
	sum := sha256.Sum256(art1.Content)
	require.Equal(t, hex.EncodeToString(sum[:]), art1.Sha256)

	var count int
	require.NoError(t, db.QueryRow(`SELECT count(*) FROM fiscal_mock_operations WHERE operation_key=$1`, "pg-op-ft").Scan(&count))
	require.Equal(t, 1, count)

	reloaded, err := fiscalmock.NewProvider(fiscalmock.Options{
		AppEnv: "development", Registry: registry, Store: fiscalmock.NewPostgresOperationStore(db),
	})
	require.NoError(t, err)
	second, err := reloaded.Issue(context.Background(), ftReq)
	require.NoError(t, err)
	assert.Equal(t, first.ProviderReference, second.ProviderReference)
	assert.Equal(t, first.ProviderNumber, second.ProviderNumber)
	assert.Equal(t, first.ArtifactID, second.ArtifactID)

	art2, err := reloaded.FetchArtifact(context.Background(), ports.FiscalFetchArtifactRequest{
		FiscalOperationRequest: ftReq.FiscalOperationRequest, ArtifactID: first.ArtifactID,
	})
	require.NoError(t, err)
	assert.Equal(t, art1.Sha256, art2.Sha256)
	assert.Equal(t, art1.Content, art2.Content)

	frReq := ftReq
	frReq.OperationKey = "pg-op-fr"
	frReq.CorrelationKey = "pg-corr-fr"
	frReq.Metadata = map[string]string{"documentKind": "FR"}
	frReq.Payload = []byte(`{"kind":"FR","pg":1}`)
	fr, err := reloaded.Issue(context.Background(), frReq)
	require.NoError(t, err)
	assert.NotEqual(t, first.ProviderReference, fr.ProviderReference)
	assert.Equal(t, "true", fr.Metadata["associatedReceiptCapability"])

	require.ErrorIs(t, fiscalmock.AssertMockAllowed("production"), fiscalmock.ErrMockForbiddenInProduction)
	_, err = store.Get(context.Background(), "pg-op-ft")
	require.NoError(t, err)
	require.ErrorIs(t, fiscalmock.AssertMockStoreAllowed("production"), fiscalmock.ErrMockStoreForbiddenInProduction)
}
