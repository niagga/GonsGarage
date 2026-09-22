package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gaston-garcia-cegid/gonsgarage/internal/core/ports"
	"github.com/gaston-garcia-cegid/gonsgarage/internal/domain"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

// FiscalRepository implements ports.FiscalRepository against PostgreSQL.
type FiscalRepository struct {
	db *sql.DB
}

var _ ports.FiscalRepository = (*FiscalRepository)(nil)

// NewFiscalRepository builds the SQL fiscal aggregate repository.
func NewFiscalRepository(db *sql.DB) *FiscalRepository {
	return &FiscalRepository{db: db}
}

func (r *FiscalRepository) HasProtectedFiscalHistory(ctx context.Context, invoiceID uuid.UUID) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx, `
SELECT EXISTS(
  SELECT 1 FROM fiscal_documents WHERE source_invoice_id=$1 AND state<>'draft'
)`, invoiceID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("has protected fiscal history: %w", err)
	}
	return exists, nil
}

func (r *FiscalRepository) CreateDraft(ctx context.Context, input ports.CreateDraftInput) (*ports.FiscalDocumentAggregate, error) {
	if input.SourceInvoiceID == uuid.Nil || input.CreatedBy == uuid.Nil {
		return nil, fmt.Errorf("source invoice and creator are required")
	}
	slot := strings.TrimSpace(input.IntentSlot)
	if slot == "" {
		slot = "primary_sale"
	}
	docID := input.ID
	if docID == uuid.Nil {
		docID = uuid.New()
	}
	snapID := input.Snapshot.ID
	if snapID == uuid.Nil {
		snapID = uuid.New()
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	var eligibility string
	err = tx.QueryRowContext(ctx, `SELECT fiscal_eligibility FROM invoices WHERE id=$1 FOR UPDATE`, input.SourceInvoiceID).Scan(&eligibility)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrInvoiceNotFound
	}
	if err != nil {
		return nil, err
	}
	if eligibility != string(domain.FiscalEligibilityEligible) {
		return nil, ports.ErrFiscalInvoiceNotEligible
	}

	_, err = tx.ExecContext(ctx, `
INSERT INTO fiscal_documents(id,source_invoice_id,intent_slot,kind,state,version,created_by)
VALUES ($1,$2,$3,$4,'draft',1,$5)`, docID, input.SourceInvoiceID, slot, string(input.Kind), input.CreatedBy)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ports.ErrFiscalIntentConflict
		}
		return nil, fmt.Errorf("insert fiscal document: %w", err)
	}
	if err := insertSnapshot(ctx, tx, snapID, docID, input.Snapshot); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return r.GetAggregate(ctx, docID)
}

func (r *FiscalRepository) UpdateDraft(ctx context.Context, documentID uuid.UUID, expectedVersion int64, snapshot ports.FiscalSnapshotRecord) (*ports.FiscalDocumentAggregate, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	doc, err := lockDocument(ctx, tx, documentID)
	if err != nil {
		return nil, err
	}
	if doc.Version != expectedVersion {
		return nil, ports.ErrFiscalVersionConflict
	}
	if doc.State != domain.FiscalDocumentStateDraft {
		return nil, ports.ErrFiscalActionNotAllowed
	}
	var snapID uuid.UUID
	err = tx.QueryRowContext(ctx, `SELECT id FROM fiscal_snapshots WHERE fiscal_document_id=$1 FOR UPDATE`, documentID).Scan(&snapID)
	if err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM fiscal_document_lines WHERE snapshot_id=$1`, snapID); err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `
UPDATE fiscal_snapshots SET issuer=$2,customer=$3,billing_address=$4,currency=$5,
 gross_total=$6,discount_total=$7,net_total=$8,tax_total=$9,rounding_adjustment=$10,payable_total=$11,
 policy_version_id=$12,issuer_profile_id=$13,updated_at=now() WHERE id=$1`,
		snapID, jsonOrObject(snapshot.Issuer), jsonOrObject(snapshot.Customer), jsonOrObject(snapshot.BillingAddress),
		snapshot.Currency, decOrZero(snapshot.GrossTotal), decOrZero(snapshot.DiscountTotal), decOrZero(snapshot.NetTotal),
		decOrZero(snapshot.TaxTotal), decOrZero(snapshot.RoundingAdj), decOrZero(snapshot.PayableTotal),
		nullableUUID(snapshot.PolicyVersionID), nullableUUID(snapshot.IssuerProfileID)); err != nil {
		return nil, err
	}
	if err := insertLines(ctx, tx, snapID, snapshot.Lines); err != nil {
		return nil, err
	}
	res, err := tx.ExecContext(ctx, `UPDATE fiscal_documents SET version=version+1,updated_at=now() WHERE id=$1 AND version=$2`, documentID, expectedVersion)
	if err != nil {
		return nil, err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return nil, ports.ErrFiscalVersionConflict
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return r.GetAggregate(ctx, documentID)
}

func (r *FiscalRepository) GetAggregate(ctx context.Context, documentID uuid.UUID) (*ports.FiscalDocumentAggregate, error) {
	return loadAggregate(ctx, r.db, documentID)
}

func (r *FiscalRepository) GetCurrentIntent(ctx context.Context, invoiceID uuid.UUID, intentSlot string) (*ports.FiscalDocumentAggregate, error) {
	slot := strings.TrimSpace(intentSlot)
	if slot == "" {
		slot = "primary_sale"
	}
	var id uuid.UUID
	err := r.db.QueryRowContext(ctx, `
SELECT id FROM fiscal_documents WHERE source_invoice_id=$1 AND intent_slot=$2 AND superseded_at IS NULL`, invoiceID, slot).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ports.ErrFiscalDocumentNotFound
	}
	if err != nil {
		return nil, err
	}
	return r.GetAggregate(ctx, id)
}

func (r *FiscalRepository) GetNextPending(ctx context.Context) (*domain.FiscalDocument, error) {
	row := r.db.QueryRowContext(ctx, `
SELECT id,source_invoice_id,intent_slot,kind,state,version,supersedes_document_id,superseded_at,provider_key,connection_id,
 intent_key,issue_operation_key,void_operation_key,provider_reference,provider_number,provider_confirmed_at,issued_at,voided_at,
 frozen_at,last_error_class,last_error_code,last_error_message,created_by,finalized_by,created_at,updated_at
 FROM fiscal_documents 
 WHERE state IN ('pending', 'retryable_failure', 'void_pending')
 ORDER BY updated_at ASC LIMIT 1`)
	doc, err := scanDocument(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ports.ErrFiscalDocumentNotFound
	}
	return doc, err
}

func (r *FiscalRepository) DeleteDraft(ctx context.Context, documentID uuid.UUID, expectedVersion int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	doc, err := lockDocument(ctx, tx, documentID)
	if err != nil {
		return err
	}
	if doc.Version != expectedVersion {
		return ports.ErrFiscalVersionConflict
	}
	if doc.State != domain.FiscalDocumentStateDraft {
		return ports.ErrFiscalActionNotAllowed
	}
	var snapID uuid.UUID
	if err := tx.QueryRowContext(ctx, `SELECT id FROM fiscal_snapshots WHERE fiscal_document_id=$1`, documentID).Scan(&snapID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM fiscal_document_lines WHERE snapshot_id=$1`, snapID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM fiscal_snapshots WHERE id=$1`, snapID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM fiscal_documents WHERE id=$1`, documentID); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *FiscalRepository) FinalizeDraft(ctx context.Context, cmd ports.FinalizeDraftCommand) (*ports.FinalizeDraftResult, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	doc, err := lockDocument(ctx, tx, cmd.DocumentID)
	if err != nil {
		return nil, err
	}
	if doc.State == domain.FiscalDocumentStatePending && doc.IntentFixed() {
		outboxID, seq, qerr := latestIssueOutbox(ctx, tx, cmd.DocumentID)
		if qerr != nil {
			return nil, qerr
		}
		if err := tx.Commit(); err != nil {
			return nil, err
		}
		return &ports.FinalizeDraftResult{Document: doc.Frozen(), OutboxID: outboxID, Sequence: seq}, nil
	}
	if doc.Version != cmd.ExpectedVersion {
		return nil, ports.ErrFiscalVersionConflict
	}
	if doc.State != domain.FiscalDocumentStateDraft {
		return nil, ports.ErrFiscalActionNotAllowed
	}

	var invoiceID uuid.UUID
	var eligibility string
	if err := tx.QueryRowContext(ctx, `SELECT id,fiscal_eligibility FROM invoices WHERE id=$1 FOR UPDATE`, doc.SourceInvoiceID).Scan(&invoiceID, &eligibility); err != nil {
		return nil, err
	}
	if eligibility != string(domain.FiscalEligibilityEligible) {
		return nil, ports.ErrFiscalInvoiceNotEligible
	}

	var connState string
	if err := tx.QueryRowContext(ctx, `SELECT state FROM fiscal_provider_connections WHERE id=$1 FOR UPDATE`, cmd.ConnectionID).Scan(&connState); err != nil {
		return nil, fmt.Errorf("%w: connection", ports.ErrFiscalNotReady)
	}
	if connState != "connected" {
		return nil, fmt.Errorf("%w: connection state %s", ports.ErrFiscalNotReady, connState)
	}

	frozen, err := doc.Finalize(domain.FiscalActorRoleManager, domain.FiscalFreezeInput{
		ProviderKey: cmd.ProviderKey, ConnectionID: cmd.ConnectionID, IntentKey: cmd.IntentKey,
		IssueOperationKey: cmd.IssueOpKey, FinalizedBy: cmd.ActorID, FrozenAt: cmd.FrozenAt,
	})
	if err != nil {
		return nil, err
	}

	var snapID uuid.UUID
	if err := tx.QueryRowContext(ctx, `SELECT id FROM fiscal_snapshots WHERE fiscal_document_id=$1 FOR UPDATE`, cmd.DocumentID).Scan(&snapID); err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM fiscal_document_lines WHERE snapshot_id=$1`, snapID); err != nil {
		return nil, err
	}
	if err := insertLines(ctx, tx, snapID, cmd.Lines); err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `
UPDATE fiscal_snapshots SET policy_version_id=$2,issuer_profile_id=$3,issuer=$4,customer=$5,billing_address=$6,currency=$7,
 gross_total=$8,discount_total=$9,net_total=$10,tax_total=$11,rounding_adjustment=$12,payable_total=$13,
 canonical_bytes=$14,canonical_sha256=$15,frozen_at=$16,updated_at=now() WHERE id=$1`,
		snapID, nullableUUID(&cmd.PolicyVersionID), nullableUUID(&cmd.IssuerProfileID),
		jsonOrObject(cmd.Issuer), jsonOrObject(cmd.Customer), jsonOrObject(cmd.BillingAddress), cmd.Currency,
		decOrZero(cmd.GrossTotal), decOrZero(cmd.DiscountTotal), decOrZero(cmd.NetTotal), decOrZero(cmd.TaxTotal),
		decOrZero(cmd.RoundingAdj), decOrZero(cmd.PayableTotal), cmd.CanonicalBytes, cmd.CanonicalSHA256, cmd.FrozenAt); err != nil {
		return nil, err
	}

	res, err := tx.ExecContext(ctx, `
UPDATE fiscal_documents SET state='pending',version=version+1,provider_key=$2,connection_id=$3,intent_key=$4,
 issue_operation_key=$5,frozen_at=$6,finalized_by=$7,updated_at=now()
 WHERE id=$1 AND version=$8 AND state='draft'`,
		cmd.DocumentID, cmd.ProviderKey, cmd.ConnectionID, cmd.IntentKey, cmd.IssueOpKey, cmd.FrozenAt, cmd.ActorID, cmd.ExpectedVersion)
	if err != nil {
		return nil, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil, ports.ErrFiscalVersionConflict
	}
	if _, err := tx.ExecContext(ctx, `
INSERT INTO fiscal_state_transitions(fiscal_document_id,from_state,to_state,operation,actor_type,actor_id)
VALUES ($1,'draft','pending','finalize','user',$2)`, cmd.DocumentID, cmd.ActorID); err != nil {
		return nil, err
	}
	outboxID := uuid.New()
	if _, err := tx.ExecContext(ctx, `
INSERT INTO fiscal_outbox_events(id,fiscal_document_id,event_type,operation_key,sequence_no,status)
VALUES ($1,$2,'issue',$3,1,'ready')`, outboxID, cmd.DocumentID, cmd.IssueOpKey); err != nil {
		return nil, fmt.Errorf("insert outbox: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	frozen.Version = cmd.ExpectedVersion + 1
	frozen.State = domain.FiscalDocumentStatePending
	return &ports.FinalizeDraftResult{Document: frozen, OutboxID: outboxID, Sequence: 1}, nil
}

func (r *FiscalRepository) EnqueueAction(ctx context.Context, cmd ports.EnqueueActionCommand) (*domain.FiscalDocument, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	doc, err := lockDocument(ctx, tx, cmd.DocumentID)
	if err != nil {
		return nil, err
	}
	if doc.Version != cmd.ExpectedVersion {
		return nil, ports.ErrFiscalVersionConflict
	}
	from := doc.State
	if cmd.VoidOperationKey != "" {
		doc.VoidOperationKey = cmd.VoidOperationKey
	}
	if cmd.NextState != doc.State {
		if err := doc.TransitionTo(cmd.NextState, time.Now().UTC()); err != nil {
			return nil, err
		}
	}
	var seq int
	err = tx.QueryRowContext(ctx, `
SELECT COALESCE(MAX(sequence_no),0)+1 FROM fiscal_outbox_events WHERE fiscal_document_id=$1 AND event_type=$2`,
		cmd.DocumentID, cmd.EventType).Scan(&seq)
	if err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `
UPDATE fiscal_documents SET state=$2,version=version+1,void_operation_key=COALESCE(NULLIF($3,''),void_operation_key),updated_at=now()
 WHERE id=$1 AND version=$4`, cmd.DocumentID, string(cmd.NextState), cmd.VoidOperationKey, cmd.ExpectedVersion); err != nil {
		return nil, err
	}
	if from != cmd.NextState {
		if _, err := tx.ExecContext(ctx, `
INSERT INTO fiscal_state_transitions(fiscal_document_id,from_state,to_state,operation,actor_type,actor_id)
VALUES ($1,$2,$3,$4,'user',$5)`, cmd.DocumentID, string(from), string(cmd.NextState), cmd.EventType, cmd.ActorID); err != nil {
			return nil, err
		}
	}
	if _, err := tx.ExecContext(ctx, `
INSERT INTO fiscal_outbox_events(fiscal_document_id,event_type,operation_key,sequence_no,status)
VALUES ($1,$2,$3,$4,'ready')`, cmd.DocumentID, cmd.EventType, cmd.OperationKey, seq); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	doc.Version = cmd.ExpectedVersion + 1
	doc.State = cmd.NextState
	return doc, nil
}

func insertSnapshot(ctx context.Context, tx *sql.Tx, snapID, docID uuid.UUID, snapshot ports.FiscalSnapshotRecord) error {
	_, err := tx.ExecContext(ctx, `
INSERT INTO fiscal_snapshots(id,fiscal_document_id,policy_version_id,issuer_profile_id,issuer,customer,billing_address,currency,
 gross_total,discount_total,net_total,tax_total,rounding_adjustment,payable_total)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
		snapID, docID, nullableUUID(snapshot.PolicyVersionID), nullableUUID(snapshot.IssuerProfileID),
		jsonOrObject(snapshot.Issuer), jsonOrObject(snapshot.Customer), jsonOrObject(snapshot.BillingAddress), snapshot.Currency,
		decOrZero(snapshot.GrossTotal), decOrZero(snapshot.DiscountTotal), decOrZero(snapshot.NetTotal),
		decOrZero(snapshot.TaxTotal), decOrZero(snapshot.RoundingAdj), decOrZero(snapshot.PayableTotal))
	if err != nil {
		return fmt.Errorf("insert snapshot: %w", err)
	}
	return insertLines(ctx, tx, snapID, snapshot.Lines)
}

func insertLines(ctx context.Context, tx *sql.Tx, snapID uuid.UUID, lines []ports.FiscalLineRecord) error {
	for _, line := range lines {
		id := line.ID
		if id == uuid.Nil {
			id = uuid.New()
		}
		unit := line.UnitCode
		if unit == "" {
			unit = "unit"
		}
		discountKind := line.DiscountKind
		if discountKind == "" {
			discountKind = "none"
		}
		_, err := tx.ExecContext(ctx, `
INSERT INTO fiscal_document_lines(id,snapshot_id,position,description,unit_code,quantity,unit_price,gross_amount,
 discount_kind,discount_value,discount_amount,net_amount,tax_rate,tax_amount,line_total,tax_treatment_code,
 exemption_code,exemption_reason,source_type,source_id)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20)`,
			id, snapID, line.Position, line.Description, unit, decOrZero(line.Quantity), decOrZero(line.UnitPrice),
			decOrZero(line.GrossAmount), discountKind, decOrZero(line.DiscountValue), decOrZero(line.DiscountAmount),
			decOrZero(line.NetAmount), decOrZero(line.TaxRate), decOrZero(line.TaxAmount), decOrZero(line.LineTotal),
			line.TaxTreatmentCode, nullIfEmpty(line.ExemptionCode), nullIfEmpty(line.ExemptionReason),
			nullIfEmpty(line.SourceType), line.SourceID)
		if err != nil {
			return fmt.Errorf("insert line %d: %w", line.Position, err)
		}
	}
	return nil
}

func lockDocument(ctx context.Context, tx *sql.Tx, documentID uuid.UUID) (*domain.FiscalDocument, error) {
	row := tx.QueryRowContext(ctx, `
SELECT id,source_invoice_id,intent_slot,kind,state,version,supersedes_document_id,superseded_at,provider_key,connection_id,
 intent_key,issue_operation_key,void_operation_key,provider_reference,provider_number,provider_confirmed_at,issued_at,voided_at,
 frozen_at,last_error_class,last_error_code,last_error_message,created_by,finalized_by,created_at,updated_at
 FROM fiscal_documents WHERE id=$1 FOR UPDATE`, documentID)
	doc, err := scanDocument(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ports.ErrFiscalDocumentNotFound
	}
	return doc, err
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanDocument(row rowScanner) (*domain.FiscalDocument, error) {
	var doc domain.FiscalDocument
	var supersedes, connection, intent, finalized sql.NullString
	var supersededAt, providerConfirmed, issuedAt, voidedAt, frozenAt sql.NullTime
	var providerKey, issueKey, voidKey, providerRef, providerNum, errClass, errCode, errMsg sql.NullString
	err := row.Scan(
		&doc.ID, &doc.SourceInvoiceID, &doc.IntentSlot, &doc.Kind, &doc.State, &doc.Version,
		&supersedes, &supersededAt, &providerKey, &connection, &intent, &issueKey, &voidKey,
		&providerRef, &providerNum, &providerConfirmed, &issuedAt, &voidedAt, &frozenAt,
		&errClass, &errCode, &errMsg, &doc.CreatedBy, &finalized, &doc.CreatedAt, &doc.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	doc.SupersedesDocumentID = parseUUIDPtr(supersedes)
	doc.ConnectionID = parseUUIDPtr(connection)
	doc.IntentKey = parseUUIDPtr(intent)
	doc.FinalizedBy = parseUUIDPtr(finalized)
	doc.ProviderKey = providerKey.String
	doc.IssueOperationKey = issueKey.String
	doc.VoidOperationKey = voidKey.String
	doc.ProviderReference = providerRef.String
	doc.ProviderNumber = providerNum.String
	doc.LastErrorClass = errClass.String
	doc.LastErrorCode = errCode.String
	doc.LastErrorMessage = errMsg.String
	if supersededAt.Valid {
		doc.SupersededAt = &supersededAt.Time
	}
	if providerConfirmed.Valid {
		doc.ProviderConfirmedAt = &providerConfirmed.Time
	}
	if issuedAt.Valid {
		doc.IssuedAt = &issuedAt.Time
	}
	if voidedAt.Valid {
		doc.VoidedAt = &voidedAt.Time
	}
	if frozenAt.Valid {
		doc.FrozenAt = &frozenAt.Time
	}
	return &doc, nil
}

func loadAggregate(ctx context.Context, db *sql.DB, documentID uuid.UUID) (*ports.FiscalDocumentAggregate, error) {
	row := db.QueryRowContext(ctx, `
SELECT id,source_invoice_id,intent_slot,kind,state,version,supersedes_document_id,superseded_at,provider_key,connection_id,
 intent_key,issue_operation_key,void_operation_key,provider_reference,provider_number,provider_confirmed_at,issued_at,voided_at,
 frozen_at,last_error_class,last_error_code,last_error_message,created_by,finalized_by,created_at,updated_at
 FROM fiscal_documents WHERE id=$1`, documentID)
	doc, err := scanDocument(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ports.ErrFiscalDocumentNotFound
	}
	if err != nil {
		return nil, err
	}
	var snap ports.FiscalSnapshotRecord
	var policyID, issuerID sql.NullString
	var canonical []byte
	var sha sql.NullString
	var frozen sql.NullTime
	err = db.QueryRowContext(ctx, `
SELECT id,policy_version_id,issuer_profile_id,issuer,customer,billing_address,currency,
 gross_total::text,discount_total::text,net_total::text,tax_total::text,rounding_adjustment::text,payable_total::text,
 canonical_bytes,canonical_sha256,frozen_at
 FROM fiscal_snapshots WHERE fiscal_document_id=$1`, documentID).Scan(
		&snap.ID, &policyID, &issuerID, &snap.Issuer, &snap.Customer, &snap.BillingAddress, &snap.Currency,
		&snap.GrossTotal, &snap.DiscountTotal, &snap.NetTotal, &snap.TaxTotal, &snap.RoundingAdj, &snap.PayableTotal,
		&canonical, &sha, &frozen,
	)
	if err != nil {
		return nil, err
	}
	snap.PolicyVersionID = parseUUIDPtr(policyID)
	snap.IssuerProfileID = parseUUIDPtr(issuerID)
	snap.CanonicalBytes = canonical
	snap.CanonicalSHA256 = sha.String
	if frozen.Valid {
		snap.FrozenAt = &frozen.Time
	}
	rows, err := db.QueryContext(ctx, `
SELECT id,position,description,unit_code,quantity::text,unit_price::text,gross_amount::text,discount_kind,
 discount_value::text,discount_amount::text,net_amount::text,tax_rate::text,tax_amount::text,line_total::text,
 tax_treatment_code,COALESCE(exemption_code,''),COALESCE(exemption_reason,''),COALESCE(source_type,''),source_id
 FROM fiscal_document_lines WHERE snapshot_id=$1 ORDER BY position`, snap.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var line ports.FiscalLineRecord
		var sourceID sql.NullString
		if err := rows.Scan(&line.ID, &line.Position, &line.Description, &line.UnitCode, &line.Quantity, &line.UnitPrice,
			&line.GrossAmount, &line.DiscountKind, &line.DiscountValue, &line.DiscountAmount, &line.NetAmount,
			&line.TaxRate, &line.TaxAmount, &line.LineTotal, &line.TaxTreatmentCode, &line.ExemptionCode,
			&line.ExemptionReason, &line.SourceType, &sourceID); err != nil {
			return nil, err
		}
		line.SourceID = parseUUIDPtr(sourceID)
		snap.Lines = append(snap.Lines, line)
	}
	return &ports.FiscalDocumentAggregate{Document: *doc, Snapshot: snap}, rows.Err()
}

func latestIssueOutbox(ctx context.Context, tx *sql.Tx, documentID uuid.UUID) (uuid.UUID, int, error) {
	var id uuid.UUID
	var seq int
	err := tx.QueryRowContext(ctx, `
SELECT id,sequence_no FROM fiscal_outbox_events WHERE fiscal_document_id=$1 AND event_type='issue'
 ORDER BY sequence_no ASC LIMIT 1`, documentID).Scan(&id, &seq)
	if err != nil {
		return uuid.Nil, 0, err
	}
	return id, seq, nil
}

func jsonOrObject(raw json.RawMessage) []byte {
	if len(raw) == 0 {
		return []byte(`{}`)
	}
	return raw
}

func decOrZero(v string) string {
	if strings.TrimSpace(v) == "" {
		return "0"
	}
	return v
}

func nullIfEmpty(v string) any {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	return v
}

func nullableUUID(id *uuid.UUID) any {
	if id == nil || *id == uuid.Nil {
		return nil
	}
	return *id
}

func parseUUIDPtr(v sql.NullString) *uuid.UUID {
	if !v.Valid || v.String == "" {
		return nil
	}
	id, err := uuid.Parse(v.String)
	if err != nil {
		return nil
	}
	return &id
}

func isUniqueViolation(err error) bool {
	var pqErr *pq.Error
	return errors.As(err, &pqErr) && pqErr.Code == "23505"
}
