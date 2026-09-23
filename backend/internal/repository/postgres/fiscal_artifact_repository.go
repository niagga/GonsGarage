package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/gaston-garcia-cegid/gonsgarage/internal/core/ports"
	"github.com/gaston-garcia-cegid/gonsgarage/internal/domain"
	"github.com/google/uuid"
)

// FiscalArtifactRepository persists fiscal artifact metadata and access logs.
type FiscalArtifactRepository struct {
	db *sql.DB
}

// NewFiscalArtifactRepository builds the PostgreSQL artifact repository.
func NewFiscalArtifactRepository(db *sql.DB) *FiscalArtifactRepository {
	return &FiscalArtifactRepository{db: db}
}

// Save upserts artifact metadata by id (insert or update non-immutable fields).
func (r *FiscalArtifactRepository) Save(ctx context.Context, artifact *domain.FiscalArtifact) error {
	if artifact == nil {
		return errors.New("fiscal artifact is nil")
	}
	if artifact.ID == uuid.Nil {
		artifact.ID = uuid.New()
	}
	if artifact.CreatedAt.IsZero() {
		artifact.CreatedAt = time.Now().UTC()
	}
	_, err := r.db.ExecContext(ctx, `
INSERT INTO fiscal_artifacts (
 id, fiscal_document_id, source_invoice_id, kind, status, classification, storage_key, media_type,
 byte_size, sha256, provider_reference, provider_version, created_at, available_at, last_verified_at,
 last_error_code, last_error_message
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)
ON CONFLICT (id) DO UPDATE SET
 status=EXCLUDED.status, storage_key=EXCLUDED.storage_key, media_type=EXCLUDED.media_type,
 byte_size=EXCLUDED.byte_size, sha256=EXCLUDED.sha256, provider_reference=EXCLUDED.provider_reference,
 provider_version=EXCLUDED.provider_version, available_at=EXCLUDED.available_at,
 last_verified_at=EXCLUDED.last_verified_at, last_error_code=EXCLUDED.last_error_code,
 last_error_message=EXCLUDED.last_error_message`,
		artifact.ID, artifact.FiscalDocumentID, artifact.SourceInvoiceID, string(artifact.Kind),
		string(artifact.Status), string(artifact.Classification), nullIfEmpty(artifact.StorageKey),
		nullIfEmpty(artifact.MediaType), nullIfZero(artifact.ByteSize), nullIfEmpty(artifact.SHA256),
		nullIfEmpty(artifact.ProviderReference), nullIfEmpty(artifact.ProviderVersion), artifact.CreatedAt,
		artifact.AvailableAt, artifact.LastVerifiedAt, nullIfEmpty(artifact.LastErrorCode), nullIfEmpty(artifact.LastErrorMessage),
	)
	return err
}

// GetByID loads an artifact by primary key.
func (r *FiscalArtifactRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.FiscalArtifact, error) {
	return r.scanOne(ctx, `SELECT id,fiscal_document_id,source_invoice_id,kind,status,classification,storage_key,media_type,
byte_size,sha256,provider_reference,provider_version,created_at,available_at,last_verified_at,last_error_code,last_error_message
FROM fiscal_artifacts WHERE id=$1`, id)
}

// GetByDocumentID loads the provider_pdf artifact for a document.
func (r *FiscalArtifactRepository) GetByDocumentID(ctx context.Context, documentID uuid.UUID) (*domain.FiscalArtifact, error) {
	return r.scanOne(ctx, `SELECT id,fiscal_document_id,source_invoice_id,kind,status,classification,storage_key,media_type,
byte_size,sha256,provider_reference,provider_version,created_at,available_at,last_verified_at,last_error_code,last_error_message
FROM fiscal_artifacts WHERE fiscal_document_id=$1 AND kind='provider_pdf'`, documentID)
}

// MarkUnavailable marks metadata as unavailable without deleting storage evidence.
func (r *FiscalArtifactRepository) MarkUnavailable(ctx context.Context, id uuid.UUID, code, message string) error {
	res, err := r.db.ExecContext(ctx, `
UPDATE fiscal_artifacts SET status='unavailable', last_error_code=$2, last_error_message=$3 WHERE id=$1`,
		id, truncate(code, 80), message)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ports.ErrFiscalArtifactNotFound
	}
	return nil
}

// MarkCompromised marks metadata as compromised after integrity failure.
func (r *FiscalArtifactRepository) MarkCompromised(ctx context.Context, id uuid.UUID, code, message string) error {
	res, err := r.db.ExecContext(ctx, `
UPDATE fiscal_artifacts SET status='compromised', last_error_code=$2, last_error_message=$3 WHERE id=$1`,
		id, truncate(code, 80), message)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ports.ErrFiscalArtifactNotFound
	}
	return nil
}

// AppendAccessLog inserts an append-only access decision.
func (r *FiscalArtifactRepository) AppendAccessLog(ctx context.Context, entry ports.FiscalArtifactAccessLog) error {
	_, err := r.db.ExecContext(ctx, `
INSERT INTO fiscal_artifact_access_log (
 artifact_id, fiscal_document_id, source_invoice_id, actor_id, actor_role, outcome, request_correlation_id, ip_hash
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		entry.ArtifactID, entry.DocumentID, entry.InvoiceID, entry.ActorID, entry.ActorRole,
		string(entry.Outcome), nullIfEmpty(entry.CorrelationID), nullIfEmpty(entry.IPHash),
	)
	return err
}

func (r *FiscalArtifactRepository) scanOne(ctx context.Context, query string, arg any) (*domain.FiscalArtifact, error) {
	row := r.db.QueryRowContext(ctx, query, arg)
	var art domain.FiscalArtifact
	var kind, status, classification string
	var storageKey, mediaType, sha, providerRef, providerVer, errCode, errMsg sql.NullString
	var byteSize sql.NullInt64
	var availableAt, lastVerified sql.NullTime
	err := row.Scan(
		&art.ID, &art.FiscalDocumentID, &art.SourceInvoiceID, &kind, &status, &classification,
		&storageKey, &mediaType, &byteSize, &sha, &providerRef, &providerVer, &art.CreatedAt,
		&availableAt, &lastVerified, &errCode, &errMsg,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ports.ErrFiscalArtifactNotFound
	}
	if err != nil {
		return nil, err
	}
	art.Kind = domain.FiscalArtifactKind(kind)
	art.Status = domain.FiscalArtifactStatus(status)
	art.Classification = domain.FiscalArtifactClassification(classification)
	art.StorageKey = storageKey.String
	art.MediaType = mediaType.String
	art.ByteSize = byteSize.Int64
	art.SHA256 = sha.String
	art.ProviderReference = providerRef.String
	art.ProviderVersion = providerVer.String
	art.LastErrorCode = errCode.String
	art.LastErrorMessage = errMsg.String
	if availableAt.Valid {
		t := availableAt.Time.UTC()
		art.AvailableAt = &t
	}
	if lastVerified.Valid {
		t := lastVerified.Time.UTC()
		art.LastVerifiedAt = &t
	}
	return &art, nil
}

// SQLInvoiceReader loads invoice ownership rows for artifact authorization.
type SQLInvoiceReader struct {
	db *sql.DB
}

// NewSQLInvoiceReader builds a narrow invoice ownership reader.
func NewSQLInvoiceReader(db *sql.DB) *SQLInvoiceReader {
	return &SQLInvoiceReader{db: db}
}

// GetByID returns id + customer_id for ownership checks.
func (r *SQLInvoiceReader) GetByID(ctx context.Context, id uuid.UUID) (*domain.Invoice, error) {
	var inv domain.Invoice
	err := r.db.QueryRowContext(ctx, `SELECT id, customer_id, COALESCE(amount,0), COALESCE(status,''), COALESCE(notes,'') FROM invoices WHERE id=$1`, id).
		Scan(&inv.ID, &inv.CustomerID, &inv.Amount, &inv.Status, &inv.Notes)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrInvoiceNotFound
	}
	if err != nil {
		return nil, err
	}
	return &inv, nil
}

func nullIfZero(v int64) any {
	if v == 0 {
		return nil
	}
	return v
}

func truncate(v string, n int) string {
	if len(v) <= n {
		return v
	}
	return v[:n]
}
