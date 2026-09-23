package ports

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/gaston-garcia-cegid/gonsgarage/internal/domain"
	"github.com/google/uuid"
)

var (
	// ErrFiscalArtifactNotFound reports a missing artifact or storage object.
	ErrFiscalArtifactNotFound = errors.New("fiscal artifact not found")
	// ErrFiscalArtifactCollision reports create-if-absent checksum/size mismatch.
	ErrFiscalArtifactCollision = errors.New("fiscal artifact storage collision")
	// ErrFiscalArtifactMediaType reports a rejected media type.
	ErrFiscalArtifactMediaType = errors.New("fiscal artifact media type rejected")
	// ErrFiscalArtifactSignature reports missing PDF signature bytes.
	ErrFiscalArtifactSignature = errors.New("fiscal artifact signature rejected")
	// ErrFiscalArtifactTooLarge reports an oversize payload.
	ErrFiscalArtifactTooLarge = errors.New("fiscal artifact exceeds size limit")
	// ErrFiscalArtifactDenied reports an ownership/role denial.
	ErrFiscalArtifactDenied = errors.New("fiscal artifact access denied")
	// ErrFiscalArtifactUnavailable reports temporarily missing evidence.
	ErrFiscalArtifactUnavailable = errors.New("fiscal artifact unavailable")
	// ErrFiscalArtifactCompromised reports integrity failure.
	ErrFiscalArtifactCompromised = errors.New("fiscal artifact compromised")
	// ErrFiscalIssuanceDisabled reports archive blocked while issuance is off.
	ErrFiscalIssuanceDisabled = errors.New("fiscal issuance is disabled")
	// ErrFiscalArtifactBackendNotReady reports production storage readiness failure.
	ErrFiscalArtifactBackendNotReady = errors.New("fiscal artifact backend not production-ready")
)

// StoredArtifact is the immutable binary metadata returned by the store.
type StoredArtifact struct {
	Key       string
	MediaType string
	ByteSize  int64
	SHA256    string
	Created   bool
}

// FiscalArtifactStore is the immutable private binary store contract.
type FiscalArtifactStore interface {
	PutImmutable(ctx context.Context, key string, mediaType string, body io.Reader) (StoredArtifact, error)
	Open(ctx context.Context, key string) (io.ReadCloser, StoredArtifact, error)
	Stat(ctx context.Context, key string) (StoredArtifact, error)
}

// FiscalArtifactAccessOutcome records download authorization results.
type FiscalArtifactAccessOutcome string

const (
	FiscalArtifactAccessAllowed     FiscalArtifactAccessOutcome = "allowed"
	FiscalArtifactAccessDenied      FiscalArtifactAccessOutcome = "denied"
	FiscalArtifactAccessUnavailable FiscalArtifactAccessOutcome = "unavailable"
)

// FiscalArtifactAccessLog is an append-only access decision row.
type FiscalArtifactAccessLog struct {
	ArtifactID    uuid.UUID
	DocumentID    uuid.UUID
	InvoiceID     uuid.UUID
	ActorID       uuid.UUID
	ActorRole     string
	Outcome       FiscalArtifactAccessOutcome
	CorrelationID string
	IPHash        string
	CreatedAt     time.Time
}

// FiscalArtifactRepository persists artifact metadata and access logs.
type FiscalArtifactRepository interface {
	Save(ctx context.Context, artifact *domain.FiscalArtifact) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.FiscalArtifact, error)
	GetByDocumentID(ctx context.Context, documentID uuid.UUID) (*domain.FiscalArtifact, error)
	MarkUnavailable(ctx context.Context, id uuid.UUID, code, message string) error
	MarkCompromised(ctx context.Context, id uuid.UUID, code, message string) error
	AppendAccessLog(ctx context.Context, entry FiscalArtifactAccessLog) error
}
