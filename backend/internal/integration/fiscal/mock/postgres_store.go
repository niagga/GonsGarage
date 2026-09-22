package mock

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// PostgresOperationStore persists mock results in fiscal_mock_operations.
type PostgresOperationStore struct {
	db *sql.DB
}

// NewPostgresOperationStore wraps the fiscal_mock_operations table.
func NewPostgresOperationStore(db *sql.DB) *PostgresOperationStore {
	return &PostgresOperationStore{db: db}
}

func (s *PostgresOperationStore) Get(ctx context.Context, operationKey string) (*OperationRecord, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT operation_key, scenario, canonical_sha256, provider_reference, result, pdf_sha256, voided, created_at, updated_at
FROM fiscal_mock_operations WHERE operation_key=$1`, operationKey)
	return scanOperation(row)
}

func (s *PostgresOperationStore) GetByProviderReference(ctx context.Context, providerReference string) (*OperationRecord, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT operation_key, scenario, canonical_sha256, provider_reference, result, pdf_sha256, voided, created_at, updated_at
FROM fiscal_mock_operations WHERE provider_reference=$1`, providerReference)
	return scanOperation(row)
}

func (s *PostgresOperationStore) PutIfAbsent(ctx context.Context, record *OperationRecord) (*OperationRecord, error) {
	var pdf any
	if record.PDFSHA256 != "" {
		pdf = record.PDFSHA256
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO fiscal_mock_operations(operation_key, scenario, canonical_sha256, provider_reference, result, pdf_sha256, voided)
VALUES ($1,$2,$3,$4,$5,$6,$7)
ON CONFLICT (operation_key) DO NOTHING`,
		record.OperationKey, record.Scenario, record.CanonicalSHA256, record.ProviderReference, []byte(record.Result), pdf, record.Voided)
	if err != nil {
		return nil, err
	}
	existing, err := s.Get(ctx, record.OperationKey)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, fmt.Errorf("mock operation %q missing after put", record.OperationKey)
	}
	// PDF bytes are derived; keep caller bytes when row was freshly inserted without byte column.
	if len(existing.PDFBytes) == 0 && len(record.PDFBytes) > 0 && existing.PDFSHA256 == record.PDFSHA256 {
		existing.PDFBytes = append([]byte(nil), record.PDFBytes...)
	}
	return existing, nil
}

func (s *PostgresOperationStore) MarkVoided(ctx context.Context, operationKey string) error {
	res, err := s.db.ExecContext(ctx, `
UPDATE fiscal_mock_operations SET voided=true, updated_at=now() WHERE operation_key=$1`, operationKey)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("mock operation %q not found", operationKey)
	}
	return nil
}

type scannable interface {
	Scan(dest ...any) error
}

func scanOperation(row scannable) (*OperationRecord, error) {
	var (
		rec       OperationRecord
		pdfSHA    sql.NullString
		resultRaw []byte
	)
	err := row.Scan(&rec.OperationKey, &rec.Scenario, &rec.CanonicalSHA256, &rec.ProviderReference, &resultRaw, &pdfSHA, &rec.Voided, &rec.CreatedAt, &rec.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	rec.Result = json.RawMessage(append([]byte(nil), resultRaw...))
	if pdfSHA.Valid {
		rec.PDFSHA256 = pdfSHA.String
	}
	if rec.CreatedAt.IsZero() {
		rec.CreatedAt = time.Now().UTC()
	}
	return &rec, nil
}
