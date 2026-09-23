package fiscal

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/gaston-garcia-cegid/gonsgarage/internal/core/ports"
	"github.com/gaston-garcia-cegid/gonsgarage/internal/domain"
	"github.com/gaston-garcia-cegid/gonsgarage/internal/platform/fiscalartifact"
	"github.com/google/uuid"
)

// InvoiceOwnerReader loads invoices for ownership checks.
type InvoiceOwnerReader interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Invoice, error)
}

// ArtifactService archives, recovers, and authorizes private fiscal PDFs.
type ArtifactService struct {
	store            ports.FiscalArtifactStore
	repo             ports.FiscalArtifactRepository
	invoices         InvoiceOwnerReader
	issuanceEnabled  bool
	environment      string
	verifyOnDownload bool
}

// ArtifactServiceOption configures ArtifactService.
type ArtifactServiceOption func(*ArtifactService)

// WithIssuanceEnabled controls whether issuance-gated archives are allowed.
func WithIssuanceEnabled(enabled bool) ArtifactServiceOption {
	return func(s *ArtifactService) { s.issuanceEnabled = enabled }
}

// WithDefaultEnvironment sets the default storage environment segment.
func WithDefaultEnvironment(env string) ArtifactServiceOption {
	return func(s *ArtifactService) { s.environment = env }
}

// NewArtifactService builds the fiscal artifact application service.
func NewArtifactService(store ports.FiscalArtifactStore, repo ports.FiscalArtifactRepository, invoices InvoiceOwnerReader, opts ...ArtifactServiceOption) *ArtifactService {
	s := &ArtifactService{
		store: store, repo: repo, invoices: invoices,
		issuanceEnabled: true, environment: "default", verifyOnDownload: true,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// ArchiveArtifactCommand archives PDF bytes for an issued document.
type ArchiveArtifactCommand struct {
	ArtifactID             uuid.UUID
	DocumentID             uuid.UUID
	SourceInvoiceID        uuid.UUID
	Environment            string
	Classification         domain.FiscalArtifactClassification
	ProviderReference      string
	ProviderVersion        string
	MediaType              string
	Body                   io.Reader
	RequireIssuanceEnabled bool
}

// RecoverArtifactCommand fetches provider bytes and archives without Issue.
type RecoverArtifactCommand struct {
	ArtifactID        uuid.UUID
	DocumentID        uuid.UUID
	SourceInvoiceID   uuid.UUID
	Environment       string
	Classification    domain.FiscalArtifactClassification
	ProviderReference string
	ProviderVersion   string
	Provider          ports.FiscalProvider
	ConnectionID      uuid.UUID
	ProviderKey       string
	OperationKey      string
	CorrelationKey    string
}

// DownloadArtifactCommand authorizes and opens archived bytes.
type DownloadArtifactCommand struct {
	ArtifactID    uuid.UUID
	ActorID       uuid.UUID
	ActorRole     domain.FiscalActorRole
	CorrelationID string
	IPHash        string
}

// ArtifactDownload is the safe streaming response (no storage keys/URLs).
type ArtifactDownload struct {
	Body       io.ReadCloser
	MediaType  string
	ByteSize   int64
	Filename   string
	Headers    map[string]string
	StorageKey string // intentionally empty in successful downloads; tests assert emptiness
}

// Archive stores immutable PDF bytes and available metadata.
func (s *ArtifactService) Archive(ctx context.Context, cmd ArchiveArtifactCommand) (*domain.FiscalArtifact, error) {
	if cmd.RequireIssuanceEnabled && !s.issuanceEnabled {
		return nil, ports.ErrFiscalIssuanceDisabled
	}
	if cmd.DocumentID == uuid.Nil || cmd.SourceInvoiceID == uuid.Nil || cmd.Body == nil {
		return nil, errors.New("archive command incomplete")
	}
	env := cmd.Environment
	if env == "" {
		env = s.environment
	}
	key := fiscalartifact.BuildStorageKey(env, cmd.DocumentID)

	stored, err := s.store.PutImmutable(ctx, key, cmd.MediaType, cmd.Body)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	artID := cmd.ArtifactID
	if artID == uuid.Nil {
		if existing, getErr := s.repo.GetByDocumentID(ctx, cmd.DocumentID); getErr == nil {
			artID = existing.ID
		} else if !errors.Is(getErr, ports.ErrFiscalArtifactNotFound) {
			return nil, getErr
		} else {
			artID = uuid.New()
		}
	}
	art := &domain.FiscalArtifact{
		ID: artID, FiscalDocumentID: cmd.DocumentID, SourceInvoiceID: cmd.SourceInvoiceID,
		Kind: domain.FiscalArtifactKindProviderPDF, Status: domain.FiscalArtifactStatusAvailable,
		Classification: cmd.Classification, StorageKey: key, MediaType: stored.MediaType,
		ByteSize: stored.ByteSize, SHA256: stored.SHA256, ProviderReference: cmd.ProviderReference,
		ProviderVersion: cmd.ProviderVersion, CreatedAt: now, AvailableAt: &now, LastVerifiedAt: &now,
	}
	if err := art.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.Save(ctx, art); err != nil {
		return nil, err
	}
	return art, nil
}

// RecoverFromProvider fetches PDF bytes via FetchArtifact (never Issue) and archives them.
func (s *ArtifactService) RecoverFromProvider(ctx context.Context, cmd RecoverArtifactCommand) (*domain.FiscalArtifact, error) {
	if cmd.Provider == nil {
		return nil, errors.New("provider is required for recovery")
	}
	res, err := cmd.Provider.FetchArtifact(ctx, ports.FiscalFetchArtifactRequest{
		FiscalOperationRequest: ports.FiscalOperationRequest{
			ConnectionID: cmd.ConnectionID, ProviderKey: cmd.ProviderKey,
			OperationKey: cmd.OperationKey, CorrelationKey: cmd.CorrelationKey,
		},
		ArtifactID: cmd.ArtifactID,
	})
	if err != nil {
		if cmd.ArtifactID != uuid.Nil {
			_ = s.repo.MarkUnavailable(ctx, cmd.ArtifactID, "fetch_failed", RedactSecrets(err.Error()))
		}
		return nil, err
	}
	media := res.MediaType
	if media == "" {
		media = "application/pdf"
	}
	return s.Archive(ctx, ArchiveArtifactCommand{
		ArtifactID: cmd.ArtifactID, DocumentID: cmd.DocumentID, SourceInvoiceID: cmd.SourceInvoiceID,
		Environment: cmd.Environment, Classification: cmd.Classification,
		ProviderReference: firstNonEmpty(cmd.ProviderReference, res.ProviderReference),
		ProviderVersion:   cmd.ProviderVersion, MediaType: media, Body: bytes.NewReader(res.Content),
	})
}

// OpenDownload authorizes ownership and streams private bytes with safe headers.
func (s *ArtifactService) OpenDownload(ctx context.Context, cmd DownloadArtifactCommand) (*ArtifactDownload, error) {
	art, err := s.repo.GetByID(ctx, cmd.ArtifactID)
	if err != nil {
		return nil, err
	}
	inv, err := s.invoices.GetByID(ctx, art.SourceInvoiceID)
	if err != nil {
		return nil, err
	}
	owns := inv.CustomerID == cmd.ActorID
	outcome := ports.FiscalArtifactAccessDenied
	defer func() {
		_ = s.repo.AppendAccessLog(ctx, ports.FiscalArtifactAccessLog{
			ArtifactID: art.ID, DocumentID: art.FiscalDocumentID, InvoiceID: art.SourceInvoiceID,
			ActorID: cmd.ActorID, ActorRole: string(cmd.ActorRole), Outcome: outcome,
			CorrelationID: cmd.CorrelationID, IPHash: cmd.IPHash,
		})
	}()

	if !art.Status.IsAvailable() {
		outcome = ports.FiscalArtifactAccessUnavailable
		if art.Status == domain.FiscalArtifactStatusCompromised {
			return nil, ports.ErrFiscalArtifactCompromised
		}
		return nil, ports.ErrFiscalArtifactUnavailable
	}
	if !art.CanAct(cmd.ActorRole, owns, domain.FiscalArtifactActionView) {
		outcome = ports.FiscalArtifactAccessDenied
		return nil, ports.ErrFiscalArtifactDenied
	}

	rc, meta, err := s.store.Open(ctx, art.StorageKey)
	if err != nil {
		outcome = ports.FiscalArtifactAccessUnavailable
		return nil, ports.ErrFiscalArtifactUnavailable
	}

	if s.verifyOnDownload {
		verified, vErr := verifyAndReopen(rc, meta, art)
		if vErr != nil {
			_ = rc.Close()
			_ = s.repo.MarkCompromised(ctx, art.ID, "checksum_mismatch", vErr.Error())
			outcome = ports.FiscalArtifactAccessUnavailable
			return nil, ports.ErrFiscalArtifactCompromised
		}
		rc = verified
	}

	filename := safeDownloadFilename(art.FiscalDocumentID)
	outcome = ports.FiscalArtifactAccessAllowed
	return &ArtifactDownload{
		Body: rc, MediaType: art.MediaType, ByteSize: art.ByteSize, Filename: filename,
		Headers: map[string]string{
			"Content-Type":           art.MediaType,
			"Content-Disposition":    fmt.Sprintf(`attachment; filename="%s"`, filename),
			"X-Content-Type-Options": "nosniff",
		},
		StorageKey: "",
	}, nil
}

func verifyAndReopen(rc io.ReadCloser, meta ports.StoredArtifact, art *domain.FiscalArtifact) (io.ReadCloser, error) {
	data, err := io.ReadAll(rc)
	_ = rc.Close()
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(data)
	hexSum := hex.EncodeToString(sum[:])
	if art.SHA256 != "" && !strings.EqualFold(art.SHA256, hexSum) {
		return nil, fmt.Errorf("stored checksum mismatch")
	}
	if meta.SHA256 != "" && !strings.EqualFold(meta.SHA256, hexSum) {
		return nil, fmt.Errorf("object checksum mismatch")
	}
	if art.ByteSize > 0 && int64(len(data)) != art.ByteSize {
		return nil, fmt.Errorf("stored size mismatch")
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}

func safeDownloadFilename(documentID uuid.UUID) string {
	id := documentID.String()
	if len(id) > 8 {
		id = id[:8]
	}
	return "fiscal-" + id + ".pdf"
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
