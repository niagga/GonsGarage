package integration

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	migrations "github.com/gaston-garcia-cegid/gonsgarage/cmd/migrate/migrations"
	"github.com/gaston-garcia-cegid/gonsgarage/internal/core/ports"
	"github.com/gaston-garcia-cegid/gonsgarage/internal/domain"
	fiscalmock "github.com/gaston-garcia-cegid/gonsgarage/internal/integration/fiscal/mock"
	postgresrepo "github.com/gaston-garcia-cegid/gonsgarage/internal/repository/postgres"
	fiscalsvc "github.com/gaston-garcia-cegid/gonsgarage/internal/service/fiscal"
)

func setupOutboxFixture(t *testing.T) (*ports.FinalizeDraftResult, ports.FiscalOutboxRepository, *sql.DB, fiscalRepoSeed) {
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
	outbox := postgresrepo.NewFiscalOutboxRepository(db)
	seed := fiscalRepoSeed{UserID: userID, InvoiceID: invoiceID, ConnectionID: connectionID, PolicyID: policyID, IssuerID: issuerID}

	created, err := repo.CreateDraft(context.Background(), ports.CreateDraftInput{
		SourceInvoiceID: seed.InvoiceID, Kind: domain.DocumentKindFT, CreatedBy: seed.UserID, Snapshot: draftSnapshot(),
	})
	require.NoError(t, err)
	canon := []byte(`{"ok":true,"outbox":1}`)
	sum := sha256.Sum256(canon)
	finalized, err := repo.FinalizeDraft(context.Background(), ports.FinalizeDraftCommand{
		DocumentID: created.Document.ID, ExpectedVersion: 1, ActorID: seed.UserID, ProviderKey: "mock",
		ConnectionID: seed.ConnectionID, IntentKey: uuid.New(), IssueOpKey: "fiscal:issue:" + uuid.New().String(),
		FrozenAt: time.Now().UTC(), PolicyVersionID: seed.PolicyID, IssuerProfileID: seed.IssuerID,
		Issuer: created.Snapshot.Issuer, Customer: created.Snapshot.Customer, BillingAddress: created.Snapshot.BillingAddress,
		Currency: "EUR", GrossTotal: "10", DiscountTotal: "0", NetTotal: "10", TaxTotal: "2.3", RoundingAdj: "0", PayableTotal: "12.3",
		CanonicalBytes: canon, CanonicalSHA256: hex.EncodeToString(sum[:]), Lines: created.Snapshot.Lines,
	})
	require.NoError(t, err)
	return finalized, outbox, db, seed
}

func TestFiscalOutboxSkipLockedSingleClaim(t *testing.T) {
	finalized, outbox, _, _ := setupOutboxFixture(t)

	var wg sync.WaitGroup
	claims := make(chan *ports.ClaimedOutboxEvent, 8)
	errs := make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(owner string) {
			defer wg.Done()
			claimed, err := outbox.ClaimNext(context.Background(), owner, 30*time.Second)
			if err != nil {
				errs <- err
				return
			}
			claims <- claimed
		}(uuid.NewString())
	}
	wg.Wait()
	close(claims)
	close(errs)

	var won []*ports.ClaimedOutboxEvent
	for c := range claims {
		won = append(won, c)
	}
	require.Len(t, won, 1, "exactly one worker must win SKIP LOCKED claim")
	assert.Equal(t, ports.FiscalOutboxLeased, won[0].Event.Status)
	assert.Equal(t, finalized.Document.ID, won[0].Document.ID)
	assert.NotEqual(t, uuid.Nil, won[0].Event.LeaseToken)

	empty := 0
	for err := range errs {
		if assert.ErrorIs(t, err, ports.ErrFiscalOutboxEmpty) {
			empty++
		}
	}
	assert.Equal(t, 7, empty)
}

func TestFiscalOutboxLeaseOwnerTokenFencing(t *testing.T) {
	finalized, outbox, db, _ := setupOutboxFixture(t)
	claimed, err := outbox.ClaimNext(context.Background(), "owner-a", time.Minute)
	require.NoError(t, err)

	att, err := outbox.BeginLegalCall(context.Background(), ports.BeginLegalCallCommand{
		EventID: claimed.Event.ID, DocumentID: claimed.Document.ID,
		LeaseOwner: "owner-a", LeaseToken: claimed.Event.LeaseToken,
		ProviderKey: "mock", Operation: ports.FiscalOutboxIssue, OperationKey: claimed.Event.OperationKey,
		RequestSHA256:     hex.EncodeToString(sha256Sum([]byte(`{"ok":true,"outbox":1}`))),
		NextDocumentState: domain.FiscalDocumentStateDispatching,
	})
	require.NoError(t, err)

	err = outbox.CompleteCall(context.Background(), ports.CompleteOutboxCallCommand{
		EventID: claimed.Event.ID, DocumentID: claimed.Document.ID, AttemptID: att.ID,
		LeaseOwner: "owner-b", LeaseToken: claimed.Event.LeaseToken,
		AttemptStatus: ports.FiscalAttemptSucceeded, NextDocumentState: domain.FiscalDocumentStateIssued,
		ProviderReference: "MOCK-X",
	})
	require.ErrorIs(t, err, ports.ErrFiscalLeaseLost)

	err = outbox.CompleteCall(context.Background(), ports.CompleteOutboxCallCommand{
		EventID: claimed.Event.ID, DocumentID: claimed.Document.ID, AttemptID: att.ID,
		LeaseOwner: "owner-a", LeaseToken: uuid.New(),
		AttemptStatus: ports.FiscalAttemptSucceeded, NextDocumentState: domain.FiscalDocumentStateIssued,
		ProviderReference: "MOCK-X",
	})
	require.ErrorIs(t, err, ports.ErrFiscalLeaseLost)

	err = outbox.CompleteCall(context.Background(), ports.CompleteOutboxCallCommand{
		EventID: claimed.Event.ID, DocumentID: claimed.Document.ID, AttemptID: att.ID,
		LeaseOwner: "owner-a", LeaseToken: claimed.Event.LeaseToken,
		AttemptStatus: ports.FiscalAttemptSucceeded, Definitive: true,
		NextDocumentState: domain.FiscalDocumentStateIssued, ProviderReference: "MOCK-OK", ProviderNumber: "FT 9",
	})
	require.NoError(t, err)

	var state, status string
	require.NoError(t, db.QueryRow(`SELECT state FROM fiscal_documents WHERE id=$1`, finalized.Document.ID).Scan(&state))
	require.NoError(t, db.QueryRow(`SELECT status FROM fiscal_outbox_events WHERE id=$1`, claimed.Event.ID).Scan(&status))
	assert.Equal(t, string(domain.FiscalDocumentStateIssued), state)
	assert.Equal(t, string(ports.FiscalOutboxCompleted), status)
}

func TestFiscalOutboxLeaseExpiryRecoversWithoutBlindResubmit(t *testing.T) {
	finalized, outbox, db, _ := setupOutboxFixture(t)
	claimed, err := outbox.ClaimNext(context.Background(), "owner-crash", 50*time.Millisecond)
	require.NoError(t, err)

	att, err := outbox.BeginLegalCall(context.Background(), ports.BeginLegalCallCommand{
		EventID: claimed.Event.ID, DocumentID: claimed.Document.ID,
		LeaseOwner: "owner-crash", LeaseToken: claimed.Event.LeaseToken,
		ProviderKey: "mock", Operation: ports.FiscalOutboxIssue, OperationKey: claimed.Event.OperationKey,
		RequestSHA256:     hex.EncodeToString(sha256Sum([]byte(`x`))),
		NextDocumentState: domain.FiscalDocumentStateDispatching,
	})
	require.NoError(t, err)

	time.Sleep(80 * time.Millisecond)
	recovered, err := outbox.ClaimNext(context.Background(), "owner-recover", time.Minute)
	require.NoError(t, err)
	require.True(t, recovered.HasStartedAttempt)
	assert.Equal(t, att.ID, recovered.StartedAttemptID)
	assert.Equal(t, domain.FiscalDocumentStateDispatching, recovered.Document.State)

	providerCalls := 0
	provider := &countingIssueProvider{onIssue: func() { providerCalls++ }}
	worker := fiscalsvc.NewWorker(outbox, provider, fiscalsvc.WorkerConfig{Owner: "owner-recover", Lease: time.Minute})
	require.NoError(t, worker.ProcessClaim(context.Background(), recovered))
	assert.Equal(t, 0, providerCalls)

	var state, attemptStatus, eventStatus string
	require.NoError(t, db.QueryRow(`SELECT state FROM fiscal_documents WHERE id=$1`, finalized.Document.ID).Scan(&state))
	require.NoError(t, db.QueryRow(`SELECT status FROM fiscal_provider_attempts WHERE id=$1`, att.ID).Scan(&attemptStatus))
	require.NoError(t, db.QueryRow(`SELECT status FROM fiscal_outbox_events WHERE id=$1`, recovered.Event.ID).Scan(&eventStatus))
	assert.Equal(t, string(domain.FiscalDocumentStateOutcomeUnknown), state)
	assert.Equal(t, string(ports.FiscalAttemptUnknown), attemptStatus)
	assert.Equal(t, string(ports.FiscalOutboxCompleted), eventStatus)
}

func TestFiscalOutboxResultTransactionAtomicity(t *testing.T) {
	finalized, outbox, db, _ := setupOutboxFixture(t)
	claimed, err := outbox.ClaimNext(context.Background(), "owner-a", time.Minute)
	require.NoError(t, err)
	att, err := outbox.BeginLegalCall(context.Background(), ports.BeginLegalCallCommand{
		EventID: claimed.Event.ID, DocumentID: claimed.Document.ID,
		LeaseOwner: "owner-a", LeaseToken: claimed.Event.LeaseToken,
		ProviderKey: "mock", Operation: ports.FiscalOutboxIssue, OperationKey: claimed.Event.OperationKey,
		RequestSHA256:     hex.EncodeToString(sha256Sum([]byte(`x`))),
		NextDocumentState: domain.FiscalDocumentStateDispatching,
	})
	require.NoError(t, err)

	require.NoError(t, outbox.CompleteCall(context.Background(), ports.CompleteOutboxCallCommand{
		EventID: claimed.Event.ID, DocumentID: claimed.Document.ID, AttemptID: att.ID,
		LeaseOwner: "owner-a", LeaseToken: claimed.Event.LeaseToken,
		AttemptStatus: ports.FiscalAttemptSucceeded, Definitive: true,
		NextDocumentState: domain.FiscalDocumentStateIssued, ProviderReference: "MOCK-ATOMIC", ProviderNumber: "FT 1",
	}))

	var docState, attemptStatus, eventStatus, providerRef string
	require.NoError(t, db.QueryRow(`SELECT state, provider_reference FROM fiscal_documents WHERE id=$1`, finalized.Document.ID).Scan(&docState, &providerRef))
	require.NoError(t, db.QueryRow(`SELECT status FROM fiscal_provider_attempts WHERE id=$1`, att.ID).Scan(&attemptStatus))
	require.NoError(t, db.QueryRow(`SELECT status FROM fiscal_outbox_events WHERE id=$1`, claimed.Event.ID).Scan(&eventStatus))
	assert.Equal(t, "issued", docState)
	assert.Equal(t, "MOCK-ATOMIC", providerRef)
	assert.Equal(t, "succeeded", attemptStatus)
	assert.Equal(t, "completed", eventStatus)

	var transitions int
	require.NoError(t, db.QueryRow(`SELECT count(*) FROM fiscal_state_transitions WHERE fiscal_document_id=$1`, finalized.Document.ID).Scan(&transitions))
	assert.GreaterOrEqual(t, transitions, 2)
}

func TestFiscalOutboxWorkerEndToEndIssueWithMock(t *testing.T) {
	finalized, outbox, db, _ := setupOutboxFixture(t)
	registry := fiscalmock.NewScenarioRegistry(map[string]fiscalmock.Scenario{
		finalized.Document.IssueOperationKey: fiscalmock.ScenarioSuccess,
	})
	provider, err := fiscalmock.NewProvider(fiscalmock.Options{
		AppEnv: "development", Registry: registry, Store: fiscalmock.NewMemoryOperationStore(),
	})
	require.NoError(t, err)

	worker := fiscalsvc.NewWorker(outbox, provider, fiscalsvc.WorkerConfig{Owner: "e2e", Lease: time.Minute})
	require.NoError(t, worker.ProcessOnce(context.Background()))

	var state, ref string
	require.NoError(t, db.QueryRow(`SELECT state, provider_reference FROM fiscal_documents WHERE id=$1`, finalized.Document.ID).Scan(&state, &ref))
	assert.Equal(t, "issued", state)
	assert.Contains(t, ref, "MOCK-")
}

func sha256Sum(b []byte) []byte {
	sum := sha256.Sum256(b)
	return sum[:]
}

type countingIssueProvider struct {
	onIssue func()
}

func (p *countingIssueProvider) VerifyConnection(context.Context, ports.FiscalVerifyConnectionRequest) (ports.FiscalVerifyConnectionResult, error) {
	return ports.FiscalVerifyConnectionResult{}, nil
}
func (p *countingIssueProvider) Issue(context.Context, ports.FiscalIssueRequest) (ports.FiscalIssueResult, error) {
	if p.onIssue != nil {
		p.onIssue()
	}
	return ports.FiscalIssueResult{}, nil
}
func (p *countingIssueProvider) Reconcile(context.Context, ports.FiscalReconcileRequest) (ports.FiscalReconcileResult, error) {
	return ports.FiscalReconcileResult{}, nil
}
func (p *countingIssueProvider) Void(context.Context, ports.FiscalVoidRequest) (ports.FiscalVoidResult, error) {
	return ports.FiscalVoidResult{}, nil
}
func (p *countingIssueProvider) FetchArtifact(context.Context, ports.FiscalFetchArtifactRequest) (ports.FiscalFetchArtifactResult, error) {
	return ports.FiscalFetchArtifactResult{}, nil
}
