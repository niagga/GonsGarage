package integration

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"io"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	migrations "github.com/gaston-garcia-cegid/gonsgarage/cmd/migrate/migrations"
	"github.com/gaston-garcia-cegid/gonsgarage/internal/core/ports"
	"github.com/gaston-garcia-cegid/gonsgarage/internal/domain"
	"github.com/gaston-garcia-cegid/gonsgarage/internal/platform/fiscalartifact"
	postgresrepo "github.com/gaston-garcia-cegid/gonsgarage/internal/repository/postgres"
	fiscalsvc "github.com/gaston-garcia-cegid/gonsgarage/internal/service/fiscal"
)

func setupArtifactFixture(t *testing.T) (*sql.DB, *postgresrepo.FiscalArtifactRepository, fiscalRepoSeed, uuid.UUID) {
	t.Helper()
	db := fiscalTestDB(t)
	_, err := db.Exec(`
CREATE TABLE users (id uuid PRIMARY KEY);
CREATE TABLE invoices (
 id uuid PRIMARY KEY, customer_id uuid NOT NULL REFERENCES users(id), amount double precision NOT NULL,
 status varchar(40) NOT NULL DEFAULT 'open', notes text, created_at timestamptz DEFAULT now(), updated_at timestamptz DEFAULT now()
);`)
	require.NoError(t, err)
	require.NoError(t, migrations.Run(db, migrationDir(t)))

	userID := uuid.New()
	invoiceID := uuid.New()
	connectionID := uuid.New()
	policyID := uuid.New()
	issuerID := uuid.New()
	_, err = db.Exec(`INSERT INTO users(id) VALUES ($1)`, userID)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO invoices(id,customer_id,amount,fiscal_eligibility) VALUES ($1,$2,12.3,'eligible')`, invoiceID, userID)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO fiscal_provider_connections(id,provider_key,state) VALUES ($1,'mock','connected')`, connectionID)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO fiscal_policy_versions(id,policy_key,version,schema_version,classification,status,config,config_sha256)
VALUES ($1,'test',1,1,'mock','approved','{}',repeat('a',64))`, policyID)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO fiscal_issuer_profiles(id,version,status,legal_identity,billing_address,tax_profile,series_config,config_sha256)
VALUES ($1,1,'approved','{}','{}','{}','{}',repeat('b',64))`, issuerID)
	require.NoError(t, err)

	repo := postgresrepo.NewFiscalRepository(db)
	created, err := repo.CreateDraft(context.Background(), ports.CreateDraftInput{
		SourceInvoiceID: invoiceID, Kind: domain.DocumentKindFT, CreatedBy: userID, Snapshot: draftSnapshot(),
	})
	require.NoError(t, err)
	canon := []byte(`{"artifact":true}`)
	sum := sha256.Sum256(canon)
	finalized, err := repo.FinalizeDraft(context.Background(), ports.FinalizeDraftCommand{
		DocumentID: created.Document.ID, ExpectedVersion: 1, ActorID: userID, ProviderKey: "mock",
		ConnectionID: connectionID, IntentKey: uuid.New(), IssueOpKey: "fiscal:issue:" + uuid.New().String(),
		FrozenAt: time.Now().UTC(), PolicyVersionID: policyID, IssuerProfileID: issuerID,
		Issuer: created.Snapshot.Issuer, Customer: created.Snapshot.Customer, BillingAddress: created.Snapshot.BillingAddress,
		Currency: "EUR", GrossTotal: "10", DiscountTotal: "0", NetTotal: "10", TaxTotal: "2.3", RoundingAdj: "0", PayableTotal: "12.3",
		CanonicalBytes: canon, CanonicalSHA256: hex.EncodeToString(sum[:]), Lines: created.Snapshot.Lines,
	})
	require.NoError(t, err)
	_, err = db.Exec(`UPDATE fiscal_documents SET state='dispatching' WHERE id=$1`, finalized.Document.ID)
	require.NoError(t, err)
	_, err = db.Exec(`UPDATE fiscal_documents SET state='issued', provider_reference='MOCK-REF' WHERE id=$1`, finalized.Document.ID)
	require.NoError(t, err)

	return db, postgresrepo.NewFiscalArtifactRepository(db), fiscalRepoSeed{
		UserID: userID, InvoiceID: invoiceID, ConnectionID: connectionID, PolicyID: policyID, IssuerID: issuerID,
	}, finalized.Document.ID
}

func TestFiscalArtifactRepository_ArchiveMetadataAccessLogAndCompromised(t *testing.T) {
	db, artRepo, seed, docID := setupArtifactFixture(t)
	store, err := fiscalartifact.NewLocalStore(t.TempDir())
	require.NoError(t, err)

	invoiceRepo := postgresrepo.NewSQLInvoiceReader(db)
	svc := fiscalsvc.NewArtifactService(store, artRepo, invoiceRepo)

	pdf := []byte("%PDF-1.4\n%\xE2\xE3\xCF\xD3\n%%EOF\npg-archive\n")
	archived, err := svc.Archive(context.Background(), fiscalsvc.ArchiveArtifactCommand{
		DocumentID: docID, SourceInvoiceID: seed.InvoiceID, Environment: "test",
		Classification:    domain.FiscalArtifactClassificationMock,
		ProviderReference: "MOCK-REF", MediaType: "application/pdf", Body: bytes.NewReader(pdf),
	})
	require.NoError(t, err)
	require.Equal(t, domain.FiscalArtifactStatusAvailable, archived.Status)

	var status string
	require.NoError(t, db.QueryRow(`SELECT status FROM fiscal_artifacts WHERE id=$1`, archived.ID).Scan(&status))
	assert.Equal(t, "available", status)

	dl, err := svc.OpenDownload(context.Background(), fiscalsvc.DownloadArtifactCommand{
		ArtifactID: archived.ID, ActorID: seed.UserID, ActorRole: domain.FiscalActorRoleClient,
	})
	require.NoError(t, err)
	got, err := io.ReadAll(dl.Body)
	_ = dl.Body.Close()
	require.NoError(t, err)
	assert.Equal(t, pdf, got)

	var accessCount int
	require.NoError(t, db.QueryRow(`SELECT count(*) FROM fiscal_artifact_access_log WHERE artifact_id=$1 AND outcome='allowed'`, archived.ID).Scan(&accessCount))
	assert.Equal(t, 1, accessCount)

	// Collision rejection at store layer keeps issued document untouched.
	_, err = store.PutImmutable(context.Background(), archived.StorageKey, "application/pdf", bytes.NewReader([]byte("%PDF-1.4\nother\n")))
	require.ErrorIs(t, err, ports.ErrFiscalArtifactCollision)
	var docState string
	require.NoError(t, db.QueryRow(`SELECT state FROM fiscal_documents WHERE id=$1`, docID).Scan(&docState))
	assert.Equal(t, "issued", docState)

	require.NoError(t, artRepo.MarkCompromised(context.Background(), archived.ID, "checksum_mismatch", "bytes altered"))
	_, err = svc.OpenDownload(context.Background(), fiscalsvc.DownloadArtifactCommand{
		ArtifactID: archived.ID, ActorID: seed.UserID, ActorRole: domain.FiscalActorRoleClient,
	})
	require.ErrorIs(t, err, ports.ErrFiscalArtifactCompromised)
}
