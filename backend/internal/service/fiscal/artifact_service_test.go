package fiscal_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gaston-garcia-cegid/gonsgarage/internal/core/ports"
	"github.com/gaston-garcia-cegid/gonsgarage/internal/domain"
	"github.com/gaston-garcia-cegid/gonsgarage/internal/platform/fiscalartifact"
	fiscalsvc "github.com/gaston-garcia-cegid/gonsgarage/internal/service/fiscal"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func artifactPDF(tag string) []byte {
	return []byte("%PDF-1.4\n%\xE2\xE3\xCF\xD3\ntrailer<<>>\n%%EOF\n%" + tag + "\n")
}

type memArtifactStore struct {
	mu    sync.Mutex
	blobs map[string]ports.StoredArtifact
	bytes map[string][]byte
}

func newMemArtifactStore() *memArtifactStore {
	return &memArtifactStore{blobs: map[string]ports.StoredArtifact{}, bytes: map[string][]byte{}}
}

func (s *memArtifactStore) PutImmutable(_ context.Context, key, mediaType string, body io.Reader) (ports.StoredArtifact, error) {
	data, err := io.ReadAll(body)
	if err != nil {
		return ports.StoredArtifact{}, err
	}
	if mediaType != "application/pdf" {
		return ports.StoredArtifact{}, ports.ErrFiscalArtifactMediaType
	}
	if !bytes.HasPrefix(data, []byte("%PDF")) {
		return ports.StoredArtifact{}, ports.ErrFiscalArtifactSignature
	}
	sum := sha256.Sum256(data)
	hexSum := hex.EncodeToString(sum[:])
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing, ok := s.blobs[key]; ok {
		if existing.SHA256 != hexSum || existing.ByteSize != int64(len(data)) {
			return ports.StoredArtifact{}, ports.ErrFiscalArtifactCollision
		}
		existing.Created = false
		return existing, nil
	}
	meta := ports.StoredArtifact{Key: key, MediaType: mediaType, ByteSize: int64(len(data)), SHA256: hexSum, Created: true}
	s.blobs[key] = meta
	s.bytes[key] = append([]byte(nil), data...)
	return meta, nil
}

func (s *memArtifactStore) Open(_ context.Context, key string) (io.ReadCloser, ports.StoredArtifact, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	meta, ok := s.blobs[key]
	if !ok {
		return nil, ports.StoredArtifact{}, ports.ErrFiscalArtifactNotFound
	}
	return io.NopCloser(bytes.NewReader(s.bytes[key])), meta, nil
}

func (s *memArtifactStore) Stat(_ context.Context, key string) (ports.StoredArtifact, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	meta, ok := s.blobs[key]
	if !ok {
		return ports.StoredArtifact{}, ports.ErrFiscalArtifactNotFound
	}
	return meta, nil
}

func (s *memArtifactStore) corrupt(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.bytes[key] = []byte("%PDF-CORRUPTED\n")
}

type memArtifactRepo struct {
	mu       sync.Mutex
	byID     map[uuid.UUID]*domain.FiscalArtifact
	byDoc    map[uuid.UUID]uuid.UUID
	access   []ports.FiscalArtifactAccessLog
	failSave error
}

func newMemArtifactRepo() *memArtifactRepo {
	return &memArtifactRepo{byID: map[uuid.UUID]*domain.FiscalArtifact{}, byDoc: map[uuid.UUID]uuid.UUID{}}
}

func (r *memArtifactRepo) Save(_ context.Context, art *domain.FiscalArtifact) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.failSave != nil {
		return r.failSave
	}
	cp := *art
	r.byID[art.ID] = &cp
	r.byDoc[art.FiscalDocumentID] = art.ID
	return nil
}

func (r *memArtifactRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.FiscalArtifact, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	art, ok := r.byID[id]
	if !ok {
		return nil, ports.ErrFiscalArtifactNotFound
	}
	cp := *art
	return &cp, nil
}

func (r *memArtifactRepo) GetByDocumentID(_ context.Context, documentID uuid.UUID) (*domain.FiscalArtifact, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	id, ok := r.byDoc[documentID]
	if !ok {
		return nil, ports.ErrFiscalArtifactNotFound
	}
	cp := *r.byID[id]
	return &cp, nil
}

func (r *memArtifactRepo) MarkCompromised(_ context.Context, id uuid.UUID, code, message string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	art, ok := r.byID[id]
	if !ok {
		return ports.ErrFiscalArtifactNotFound
	}
	art.Status = domain.FiscalArtifactStatusCompromised
	art.LastErrorCode = code
	art.LastErrorMessage = message
	return nil
}

func (r *memArtifactRepo) MarkUnavailable(_ context.Context, id uuid.UUID, code, message string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	art, ok := r.byID[id]
	if !ok {
		return ports.ErrFiscalArtifactNotFound
	}
	art.Status = domain.FiscalArtifactStatusUnavailable
	art.LastErrorCode = code
	art.LastErrorMessage = message
	return nil
}

func (r *memArtifactRepo) AppendAccessLog(_ context.Context, entry ports.FiscalArtifactAccessLog) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.access = append(r.access, entry)
	return nil
}

type artifactInvoiceRepo struct {
	byID map[uuid.UUID]*domain.Invoice
}

func (r *artifactInvoiceRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.Invoice, error) {
	inv, ok := r.byID[id]
	if !ok {
		return nil, domain.ErrInvoiceNotFound
	}
	cp := *inv
	return &cp, nil
}
func (r *artifactInvoiceRepo) Create(context.Context, *domain.Invoice) error { return nil }
func (r *artifactInvoiceRepo) Update(context.Context, *domain.Invoice) error { return nil }
func (r *artifactInvoiceRepo) Delete(context.Context, uuid.UUID) error       { return nil }
func (r *artifactInvoiceRepo) ListByCustomerID(context.Context, uuid.UUID, int, int) ([]*domain.Invoice, int64, error) {
	return nil, 0, nil
}
func (r *artifactInvoiceRepo) ListForStaff(context.Context, int, int) ([]*domain.Invoice, int64, error) {
	return nil, 0, nil
}

type fetchOnlyProvider struct {
	content   []byte
	mediaType string
	err       error
	fetches   int
	issues    int
}

func (p *fetchOnlyProvider) VerifyConnection(context.Context, ports.FiscalVerifyConnectionRequest) (ports.FiscalVerifyConnectionResult, error) {
	return ports.FiscalVerifyConnectionResult{}, nil
}
func (p *fetchOnlyProvider) Issue(context.Context, ports.FiscalIssueRequest) (ports.FiscalIssueResult, error) {
	p.issues++
	return ports.FiscalIssueResult{}, errors.New("Issue must not be called during artifact recovery")
}
func (p *fetchOnlyProvider) Reconcile(context.Context, ports.FiscalReconcileRequest) (ports.FiscalReconcileResult, error) {
	return ports.FiscalReconcileResult{}, nil
}
func (p *fetchOnlyProvider) Void(context.Context, ports.FiscalVoidRequest) (ports.FiscalVoidResult, error) {
	return ports.FiscalVoidResult{}, nil
}
func (p *fetchOnlyProvider) FetchArtifact(context.Context, ports.FiscalFetchArtifactRequest) (ports.FiscalFetchArtifactResult, error) {
	p.fetches++
	if p.err != nil {
		return ports.FiscalFetchArtifactResult{}, p.err
	}
	sum := sha256.Sum256(p.content)
	return ports.FiscalFetchArtifactResult{
		Content: p.content, MediaType: p.mediaType, Sha256: hex.EncodeToString(sum[:]),
	}, nil
}

func newArtifactServiceForTest(t *testing.T, opts ...fiscalsvc.ArtifactServiceOption) (*fiscalsvc.ArtifactService, *memArtifactStore, *memArtifactRepo, *artifactInvoiceRepo) {
	t.Helper()
	store := newMemArtifactStore()
	repo := newMemArtifactRepo()
	invoices := &artifactInvoiceRepo{byID: map[uuid.UUID]*domain.Invoice{}}
	svc := fiscalsvc.NewArtifactService(store, repo, invoices, opts...)
	return svc, store, repo, invoices
}

func TestArtifactService_ArchiveCreateIfAbsentAndOwnershipDownload(t *testing.T) {
	svc, store, repo, invoices := newArtifactServiceForTest(t)
	owner := uuid.New()
	other := uuid.New()
	invoiceID := uuid.New()
	docID := uuid.New()
	invoices.byID[invoiceID] = &domain.Invoice{ID: invoiceID, CustomerID: owner}

	pdf := artifactPDF("ok")
	sum := sha256.Sum256(pdf)
	archived, err := svc.Archive(context.Background(), fiscalsvc.ArchiveArtifactCommand{
		DocumentID: docID, SourceInvoiceID: invoiceID, Environment: "dev",
		Classification:    domain.FiscalArtifactClassificationLegal,
		ProviderReference: "REF-1", MediaType: "application/pdf", Body: bytes.NewReader(pdf),
	})
	require.NoError(t, err)
	require.Equal(t, domain.FiscalArtifactStatusAvailable, archived.Status)
	require.Equal(t, hex.EncodeToString(sum[:]), archived.SHA256)
	require.Equal(t, fiscalartifact.BuildStorageKey("dev", docID), archived.StorageKey)

	repeat, err := svc.Archive(context.Background(), fiscalsvc.ArchiveArtifactCommand{
		DocumentID: docID, SourceInvoiceID: invoiceID, Environment: "dev",
		Classification:    domain.FiscalArtifactClassificationLegal,
		ProviderReference: "REF-1", MediaType: "application/pdf", Body: bytes.NewReader(pdf),
		ArtifactID: archived.ID,
	})
	require.NoError(t, err)
	assert.Equal(t, archived.ID, repeat.ID)
	assert.Equal(t, archived.SHA256, repeat.SHA256)

	dl, err := svc.OpenDownload(context.Background(), fiscalsvc.DownloadArtifactCommand{
		ArtifactID: archived.ID, ActorID: owner, ActorRole: domain.FiscalActorRoleClient,
	})
	require.NoError(t, err)
	defer dl.Body.Close()
	got, err := io.ReadAll(dl.Body)
	require.NoError(t, err)
	assert.Equal(t, pdf, got)
	assert.Equal(t, "application/pdf", dl.MediaType)
	assert.Equal(t, "nosniff", dl.Headers["X-Content-Type-Options"])
	assert.Contains(t, dl.Headers["Content-Disposition"], "attachment")
	assert.NotContains(t, dl.Filename, "/")
	assert.NotContains(t, dl.Filename, "\\")
	assert.Empty(t, dl.StorageKey)
	assert.NotContains(t, strings.ToLower(dl.Filename+dl.Headers["Content-Disposition"]), "http")
	require.Len(t, repo.access, 1)
	assert.Equal(t, ports.FiscalArtifactAccessAllowed, repo.access[0].Outcome)

	_, err = svc.OpenDownload(context.Background(), fiscalsvc.DownloadArtifactCommand{
		ArtifactID: archived.ID, ActorID: other, ActorRole: domain.FiscalActorRoleClient,
	})
	require.ErrorIs(t, err, ports.ErrFiscalArtifactDenied)
	require.GreaterOrEqual(t, len(repo.access), 2)
	assert.Equal(t, ports.FiscalArtifactAccessDenied, repo.access[len(repo.access)-1].Outcome)

	_, ok := store.blobs[archived.StorageKey]
	assert.True(t, ok)
}

func TestArtifactService_FailedArchiveRetainsIssuedAndRecoverySkipsIssue(t *testing.T) {
	store := newMemArtifactStore()
	repo := newMemArtifactRepo()
	repo.failSave = errors.New("metadata write failed")
	invoices := &artifactInvoiceRepo{byID: map[uuid.UUID]*domain.Invoice{}}
	svc := fiscalsvc.NewArtifactService(store, repo, invoices)

	docID := uuid.New()
	invoiceID := uuid.New()
	pdf := artifactPDF("same")
	_, err := svc.Archive(context.Background(), fiscalsvc.ArchiveArtifactCommand{
		DocumentID: docID, SourceInvoiceID: invoiceID, Environment: "dev",
		Classification:    domain.FiscalArtifactClassificationLegal,
		ProviderReference: "REF-FAIL", MediaType: "application/pdf", Body: bytes.NewReader(pdf),
	})
	require.Error(t, err)

	// Document lifecycle is owned by caller; Archive must not invent Issue calls.
	// Recovery reuses the same provider PDF bytes (create-if-absent checksum match).
	provider := &fetchOnlyProvider{content: pdf, mediaType: "application/pdf"}
	repo.failSave = nil
	recovered, err := svc.RecoverFromProvider(context.Background(), fiscalsvc.RecoverArtifactCommand{
		DocumentID: docID, SourceInvoiceID: invoiceID, Environment: "dev",
		Classification:    domain.FiscalArtifactClassificationLegal,
		ProviderReference: "REF-FAIL", Provider: provider,
		ConnectionID: uuid.New(), ProviderKey: "mock", OperationKey: "op-1",
	})
	require.NoError(t, err)
	assert.Equal(t, domain.FiscalArtifactStatusAvailable, recovered.Status)
	assert.Equal(t, 1, provider.fetches)
	assert.Equal(t, 0, provider.issues)
}

func TestArtifactService_UnavailableCompromisedAndStaffRoles(t *testing.T) {
	svc, store, repo, invoices := newArtifactServiceForTest(t)
	owner := uuid.New()
	invoiceID := uuid.New()
	docID := uuid.New()
	invoices.byID[invoiceID] = &domain.Invoice{ID: invoiceID, CustomerID: owner}

	pdf := artifactPDF("role")
	archived, err := svc.Archive(context.Background(), fiscalsvc.ArchiveArtifactCommand{
		DocumentID: docID, SourceInvoiceID: invoiceID, Environment: "dev",
		Classification:    domain.FiscalArtifactClassificationLegal,
		ProviderReference: "REF-2", MediaType: "application/pdf", Body: bytes.NewReader(pdf),
	})
	require.NoError(t, err)

	for _, role := range []domain.FiscalActorRole{
		domain.FiscalActorRoleEmployee, domain.FiscalActorRoleManager, domain.FiscalActorRoleAdmin,
	} {
		dl, err := svc.OpenDownload(context.Background(), fiscalsvc.DownloadArtifactCommand{
			ArtifactID: archived.ID, ActorID: uuid.New(), ActorRole: role,
		})
		require.NoError(t, err, role)
		_ = dl.Body.Close()
	}

	require.NoError(t, repo.MarkUnavailable(context.Background(), archived.ID, "fetch_failed", "temporary"))
	_, err = svc.OpenDownload(context.Background(), fiscalsvc.DownloadArtifactCommand{
		ArtifactID: archived.ID, ActorID: owner, ActorRole: domain.FiscalActorRoleClient,
	})
	require.ErrorIs(t, err, ports.ErrFiscalArtifactUnavailable)

	// Restore available metadata then corrupt bytes → compromised refusal.
	now := time.Now().UTC()
	archived.Status = domain.FiscalArtifactStatusAvailable
	archived.AvailableAt = &now
	require.NoError(t, repo.Save(context.Background(), archived))
	store.corrupt(archived.StorageKey)
	_, err = svc.OpenDownload(context.Background(), fiscalsvc.DownloadArtifactCommand{
		ArtifactID: archived.ID, ActorID: owner, ActorRole: domain.FiscalActorRoleClient,
	})
	require.ErrorIs(t, err, ports.ErrFiscalArtifactCompromised)
	got, err := repo.GetByID(context.Background(), archived.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.FiscalArtifactStatusCompromised, got.Status)
}

func TestArtifactService_ReadWhileIssuanceDisabled(t *testing.T) {
	svc, store, repo, invoices := newArtifactServiceForTest(t, fiscalsvc.WithIssuanceEnabled(false))
	owner := uuid.New()
	invoiceID := uuid.New()
	docID := uuid.New()
	invoices.byID[invoiceID] = &domain.Invoice{ID: invoiceID, CustomerID: owner}

	_, err := svc.Archive(context.Background(), fiscalsvc.ArchiveArtifactCommand{
		DocumentID: docID, SourceInvoiceID: invoiceID, Environment: "dev",
		Classification:    domain.FiscalArtifactClassificationLegal,
		ProviderReference: "REF-3", MediaType: "application/pdf", Body: bytes.NewReader(artifactPDF("disabled")),
		RequireIssuanceEnabled: true,
	})
	require.ErrorIs(t, err, ports.ErrFiscalIssuanceDisabled)

	pdf := artifactPDF("kept")
	sum := sha256.Sum256(pdf)
	key := fiscalartifact.BuildStorageKey("dev", docID)
	_, err = store.PutImmutable(context.Background(), key, "application/pdf", bytes.NewReader(pdf))
	require.NoError(t, err)
	now := time.Now().UTC()
	art := &domain.FiscalArtifact{
		ID: uuid.New(), FiscalDocumentID: docID, SourceInvoiceID: invoiceID,
		Kind: domain.FiscalArtifactKindProviderPDF, Status: domain.FiscalArtifactStatusAvailable,
		Classification: domain.FiscalArtifactClassificationLegal, StorageKey: key,
		MediaType: "application/pdf", ByteSize: int64(len(pdf)), SHA256: hex.EncodeToString(sum[:]),
		ProviderReference: "REF-3", AvailableAt: &now, CreatedAt: now,
	}
	require.NoError(t, repo.Save(context.Background(), art))

	dl, err := svc.OpenDownload(context.Background(), fiscalsvc.DownloadArtifactCommand{
		ArtifactID: art.ID, ActorID: owner, ActorRole: domain.FiscalActorRoleClient,
	})
	require.NoError(t, err)
	defer dl.Body.Close()
	got, err := io.ReadAll(dl.Body)
	require.NoError(t, err)
	assert.Equal(t, pdf, got)
}
