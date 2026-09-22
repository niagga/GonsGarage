package integration

import (
	"context"
	"database/sql"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	migrations "github.com/gaston-garcia-cegid/gonsgarage/cmd/migrate/migrations"
	"github.com/gaston-garcia-cegid/gonsgarage/internal/core/ports"
	"github.com/gaston-garcia-cegid/gonsgarage/internal/domain"
	postgresrepo "github.com/gaston-garcia-cegid/gonsgarage/internal/repository/postgres"
)

func setupFiscalRepoFixture(t *testing.T) (*sql.DB, *postgresrepo.FiscalRepository, fiscalRepoSeed) {
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

	return db, postgresrepo.NewFiscalRepository(db), fiscalRepoSeed{
		UserID: userID, InvoiceID: invoiceID, ConnectionID: connectionID, PolicyID: policyID, IssuerID: issuerID,
	}
}

type fiscalRepoSeed struct {
	UserID, InvoiceID, ConnectionID, PolicyID, IssuerID uuid.UUID
}

func draftSnapshot() ports.FiscalSnapshotRecord {
	return ports.FiscalSnapshotRecord{
		Issuer: json.RawMessage(`{}`), Customer: json.RawMessage(`{"nif":"123456789"}`), BillingAddress: json.RawMessage(`{}`),
		Currency: "EUR", GrossTotal: "10", DiscountTotal: "0", NetTotal: "10", TaxTotal: "2.3", RoundingAdj: "0", PayableTotal: "12.3",
		Lines: []ports.FiscalLineRecord{{
			Position: 1, Description: "labor", UnitCode: "unit", Quantity: "1", UnitPrice: "10", GrossAmount: "10",
			DiscountKind: "none", DiscountValue: "0", DiscountAmount: "0", NetAmount: "10", TaxRate: "0.23",
			TaxAmount: "2.3", LineTotal: "12.3", TaxTreatmentCode: "normal",
		}},
	}
}

func TestFiscalRepositoryDraftCRUDOptimisticAndIntent(t *testing.T) {
	_, repo, seed := setupFiscalRepoFixture(t)
	ctx := context.Background()

	created, err := repo.CreateDraft(ctx, ports.CreateDraftInput{
		SourceInvoiceID: seed.InvoiceID, Kind: domain.DocumentKindFT, CreatedBy: seed.UserID, Snapshot: draftSnapshot(),
	})
	require.NoError(t, err)
	require.EqualValues(t, 1, created.Document.Version)

	_, err = repo.CreateDraft(ctx, ports.CreateDraftInput{
		SourceInvoiceID: seed.InvoiceID, Kind: domain.DocumentKindFR, CreatedBy: seed.UserID, Snapshot: draftSnapshot(),
	})
	require.ErrorIs(t, err, ports.ErrFiscalIntentConflict)

	updated, err := repo.UpdateDraft(ctx, created.Document.ID, 1, draftSnapshot())
	require.NoError(t, err)
	require.EqualValues(t, 2, updated.Document.Version)

	_, err = repo.UpdateDraft(ctx, created.Document.ID, 1, draftSnapshot())
	require.ErrorIs(t, err, ports.ErrFiscalVersionConflict)
}

func TestFiscalRepositoryFinalizeAtomicAndProtectedDelete(t *testing.T) {
	db, repo, seed := setupFiscalRepoFixture(t)
	ctx := context.Background()
	created, err := repo.CreateDraft(ctx, ports.CreateDraftInput{
		SourceInvoiceID: seed.InvoiceID, Kind: domain.DocumentKindFT, CreatedBy: seed.UserID, Snapshot: draftSnapshot(),
	})
	require.NoError(t, err)

	result, err := repo.FinalizeDraft(ctx, ports.FinalizeDraftCommand{
		DocumentID: created.Document.ID, ExpectedVersion: 1, ActorID: seed.UserID, ProviderKey: "mock",
		ConnectionID: seed.ConnectionID, IntentKey: uuid.New(), IssueOpKey: "fiscal:issue:" + uuid.New().String(),
		FrozenAt: time.Now().UTC(), PolicyVersionID: seed.PolicyID, IssuerProfileID: seed.IssuerID,
		Issuer: created.Snapshot.Issuer, Customer: created.Snapshot.Customer, BillingAddress: created.Snapshot.BillingAddress,
		Currency: "EUR", GrossTotal: "10", DiscountTotal: "0", NetTotal: "10", TaxTotal: "2.3", RoundingAdj: "0", PayableTotal: "12.3",
		CanonicalBytes: []byte(`{"ok":true}`), CanonicalSHA256: "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
		Lines: created.Snapshot.Lines,
	})
	require.NoError(t, err)
	require.Equal(t, domain.FiscalDocumentStatePending, result.Document.State)
	require.Equal(t, 1, result.Sequence)

	var events int
	require.NoError(t, db.QueryRow(`SELECT count(*) FROM fiscal_outbox_events WHERE fiscal_document_id=$1`, created.Document.ID).Scan(&events))
	require.Equal(t, 1, events)

	protected, err := repo.HasProtectedFiscalHistory(ctx, seed.InvoiceID)
	require.NoError(t, err)
	require.True(t, protected)
	_, err = db.Exec(`DELETE FROM invoices WHERE id=$1`, seed.InvoiceID)
	require.Error(t, err)
}

func TestFiscalRepositoryConcurrentFinalizationOneIntent(t *testing.T) {
	db, repo, seed := setupFiscalRepoFixture(t)
	ctx := context.Background()
	created, err := repo.CreateDraft(ctx, ports.CreateDraftInput{
		SourceInvoiceID: seed.InvoiceID, Kind: domain.DocumentKindFT, CreatedBy: seed.UserID, Snapshot: draftSnapshot(),
	})
	require.NoError(t, err)

	var wg sync.WaitGroup
	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			intent := uuid.New()
			_, err := repo.FinalizeDraft(ctx, ports.FinalizeDraftCommand{
				DocumentID: created.Document.ID, ExpectedVersion: 1, ActorID: seed.UserID, ProviderKey: "mock",
				ConnectionID: seed.ConnectionID, IntentKey: intent, IssueOpKey: "fiscal:issue:" + intent.String(),
				FrozenAt: time.Now().UTC(), PolicyVersionID: seed.PolicyID, IssuerProfileID: seed.IssuerID,
				Issuer: created.Snapshot.Issuer, Customer: created.Snapshot.Customer, BillingAddress: created.Snapshot.BillingAddress,
				Currency: "EUR", GrossTotal: "10", DiscountTotal: "0", NetTotal: "10", TaxTotal: "2.3", RoundingAdj: "0", PayableTotal: "12.3",
				CanonicalBytes: []byte(`{"ok":true}`), CanonicalSHA256: "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
				Lines: created.Snapshot.Lines,
			})
			results <- err
		}()
	}
	wg.Wait()
	close(results)
	for err := range results {
		require.NoError(t, err)
	}

	var docs, events, intents int
	require.NoError(t, db.QueryRow(`SELECT count(*) FROM fiscal_documents WHERE source_invoice_id=$1 AND superseded_at IS NULL`, seed.InvoiceID).Scan(&docs))
	require.NoError(t, db.QueryRow(`SELECT count(*) FROM fiscal_outbox_events WHERE event_type='issue'`).Scan(&events))
	require.NoError(t, db.QueryRow(`SELECT count(DISTINCT intent_key) FROM fiscal_documents WHERE source_invoice_id=$1`, seed.InvoiceID).Scan(&intents))
	assert.Equal(t, 1, docs)
	assert.Equal(t, 1, events)
	assert.Equal(t, 1, intents)
}

func TestFiscalRepositoryOutboxInsertFailureRollsBack(t *testing.T) {
	db, repo, seed := setupFiscalRepoFixture(t)
	ctx := context.Background()
	created, err := repo.CreateDraft(ctx, ports.CreateDraftInput{
		SourceInvoiceID: seed.InvoiceID, Kind: domain.DocumentKindFT, CreatedBy: seed.UserID, Snapshot: draftSnapshot(),
	})
	require.NoError(t, err)
	_, err = db.Exec(`
CREATE OR REPLACE FUNCTION fiscal_fail_outbox() RETURNS trigger AS $$
BEGIN RAISE EXCEPTION 'forced outbox failure'; END; $$ LANGUAGE plpgsql;
CREATE TRIGGER fiscal_fail_outbox BEFORE INSERT ON fiscal_outbox_events
FOR EACH ROW EXECUTE FUNCTION fiscal_fail_outbox();`)
	require.NoError(t, err)

	intent := uuid.New()
	_, err = repo.FinalizeDraft(ctx, ports.FinalizeDraftCommand{
		DocumentID: created.Document.ID, ExpectedVersion: 1, ActorID: seed.UserID, ProviderKey: "mock",
		ConnectionID: seed.ConnectionID, IntentKey: intent, IssueOpKey: "fiscal:issue:" + intent.String(),
		FrozenAt: time.Now().UTC(), PolicyVersionID: seed.PolicyID, IssuerProfileID: seed.IssuerID,
		Issuer: created.Snapshot.Issuer, Customer: created.Snapshot.Customer, BillingAddress: created.Snapshot.BillingAddress,
		Currency: "EUR", GrossTotal: "10", DiscountTotal: "0", NetTotal: "10", TaxTotal: "2.3", RoundingAdj: "0", PayableTotal: "12.3",
		CanonicalBytes: []byte(`{"ok":true}`), CanonicalSHA256: "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
		Lines: created.Snapshot.Lines,
	})
	require.Error(t, err)

	agg, err := repo.GetAggregate(ctx, created.Document.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.FiscalDocumentStateDraft, agg.Document.State)
	var events int
	require.NoError(t, db.QueryRow(`SELECT count(*) FROM fiscal_outbox_events`).Scan(&events))
	assert.Equal(t, 0, events)
}

func TestFiscalRepositoryRetryReconcileVoidSequences(t *testing.T) {
	db, repo, seed := setupFiscalRepoFixture(t)
	ctx := context.Background()
	created, err := repo.CreateDraft(ctx, ports.CreateDraftInput{
		SourceInvoiceID: seed.InvoiceID, Kind: domain.DocumentKindFT, CreatedBy: seed.UserID, Snapshot: draftSnapshot(),
	})
	require.NoError(t, err)
	intent := uuid.New()
	issueKey := "fiscal:issue:" + intent.String()
	finalized, err := repo.FinalizeDraft(ctx, ports.FinalizeDraftCommand{
		DocumentID: created.Document.ID, ExpectedVersion: 1, ActorID: seed.UserID, ProviderKey: "mock",
		ConnectionID: seed.ConnectionID, IntentKey: intent, IssueOpKey: issueKey,
		FrozenAt: time.Now().UTC(), PolicyVersionID: seed.PolicyID, IssuerProfileID: seed.IssuerID,
		Issuer: created.Snapshot.Issuer, Customer: created.Snapshot.Customer, BillingAddress: created.Snapshot.BillingAddress,
		Currency: "EUR", GrossTotal: "10", DiscountTotal: "0", NetTotal: "10", TaxTotal: "2.3", RoundingAdj: "0", PayableTotal: "12.3",
		CanonicalBytes: []byte(`{"ok":true}`), CanonicalSHA256: "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
		Lines: created.Snapshot.Lines,
	})
	require.NoError(t, err)

	_, err = db.Exec(`UPDATE fiscal_outbox_events SET status='completed' WHERE fiscal_document_id=$1`, created.Document.ID)
	require.NoError(t, err)
	_, err = db.Exec(`UPDATE fiscal_documents SET state='dispatching',version=$2 WHERE id=$1`, created.Document.ID, finalized.Document.Version)
	require.NoError(t, err)
	_, err = db.Exec(`UPDATE fiscal_documents SET state='retryable_failure',version=$2 WHERE id=$1`, created.Document.ID, finalized.Document.Version)
	require.NoError(t, err)

	retried, err := repo.EnqueueAction(ctx, ports.EnqueueActionCommand{
		DocumentID: created.Document.ID, ExpectedVersion: finalized.Document.Version, ActorID: seed.UserID,
		EventType: "issue", OperationKey: issueKey, NextState: domain.FiscalDocumentStatePending,
	})
	require.NoError(t, err)
	require.Equal(t, domain.FiscalDocumentStatePending, retried.State)

	var seq int
	require.NoError(t, db.QueryRow(`SELECT MAX(sequence_no) FROM fiscal_outbox_events WHERE fiscal_document_id=$1 AND event_type='issue'`, created.Document.ID).Scan(&seq))
	assert.Equal(t, 2, seq)

	var opKey, provider, connection string
	require.NoError(t, db.QueryRow(`SELECT operation_key FROM fiscal_outbox_events WHERE fiscal_document_id=$1 AND sequence_no=2`, created.Document.ID).Scan(&opKey))
	require.NoError(t, db.QueryRow(`SELECT provider_key,connection_id::text FROM fiscal_documents WHERE id=$1`, created.Document.ID).Scan(&provider, &connection))
	assert.Equal(t, issueKey, opKey)
	assert.Equal(t, "mock", provider)
	assert.Equal(t, seed.ConnectionID.String(), connection)

	_, err = db.Exec(`UPDATE fiscal_outbox_events SET status='completed' WHERE fiscal_document_id=$1 AND status='ready'`, created.Document.ID)
	require.NoError(t, err)
	_, err = db.Exec(`UPDATE fiscal_documents SET state='dispatching',version=$2 WHERE id=$1`, created.Document.ID, retried.Version)
	require.NoError(t, err)
	_, err = db.Exec(`UPDATE fiscal_documents SET state='outcome_unknown',version=$2 WHERE id=$1`, created.Document.ID, retried.Version)
	require.NoError(t, err)

	_, err = repo.EnqueueAction(ctx, ports.EnqueueActionCommand{
		DocumentID: created.Document.ID, ExpectedVersion: retried.Version, ActorID: seed.UserID,
		EventType: "issue", OperationKey: issueKey, NextState: domain.FiscalDocumentStatePending,
	})
	require.Error(t, err)

	reconciled, err := repo.EnqueueAction(ctx, ports.EnqueueActionCommand{
		DocumentID: created.Document.ID, ExpectedVersion: retried.Version, ActorID: seed.UserID,
		EventType: "reconcile_issue", OperationKey: issueKey, NextState: domain.FiscalDocumentStateOutcomeUnknown,
	})
	require.NoError(t, err)
	assert.Equal(t, domain.FiscalDocumentStateOutcomeUnknown, reconciled.State)

	_, err = db.Exec(`UPDATE fiscal_outbox_events SET status='completed' WHERE status='ready'`)
	require.NoError(t, err)
	_, err = db.Exec(`UPDATE fiscal_documents SET state='issued',version=$2 WHERE id=$1`, created.Document.ID, reconciled.Version)
	require.NoError(t, err)
	voidKey := "fiscal:void:" + intent.String()
	voided, err := repo.EnqueueAction(ctx, ports.EnqueueActionCommand{
		DocumentID: created.Document.ID, ExpectedVersion: reconciled.Version, ActorID: seed.UserID,
		EventType: "void", OperationKey: voidKey, NextState: domain.FiscalDocumentStateVoidPending, VoidOperationKey: voidKey,
	})
	require.NoError(t, err)
	assert.Equal(t, domain.FiscalDocumentStateVoidPending, voided.State)
	assert.Equal(t, voidKey, voided.VoidOperationKey)
}
