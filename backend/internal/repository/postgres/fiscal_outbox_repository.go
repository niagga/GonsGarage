package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/gaston-garcia-cegid/gonsgarage/internal/core/ports"
	"github.com/gaston-garcia-cegid/gonsgarage/internal/domain"
)

// FiscalOutboxRepository implements lease-fenced outbox claim/completion.
type FiscalOutboxRepository struct {
	db *sql.DB
}

// NewFiscalOutboxRepository builds the PostgreSQL outbox repository.
func NewFiscalOutboxRepository(db *sql.DB) *FiscalOutboxRepository {
	return &FiscalOutboxRepository{db: db}
}

// ClaimNext claims one ready or expired-leased event using FOR UPDATE SKIP LOCKED.
func (r *FiscalOutboxRepository) ClaimNext(ctx context.Context, owner string, leaseFor time.Duration) (*ports.ClaimedOutboxEvent, error) {
	if owner == "" {
		return nil, fmt.Errorf("lease owner is required")
	}
	if leaseFor <= 0 {
		leaseFor = 30 * time.Second
	}
	token := uuid.New()
	expires := time.Now().UTC().Add(leaseFor)

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	var (
		id, docID    uuid.UUID
		eventType    string
		operationKey string
		sequenceNo   int
		claimCount   int
		artifactRaw  sql.NullString
	)
	err = tx.QueryRowContext(ctx, `
WITH next_event AS (
  SELECT id FROM fiscal_outbox_events
  WHERE (status='ready' AND available_at <= now())
     OR (status='leased' AND lease_expires_at IS NOT NULL AND lease_expires_at < now())
  ORDER BY available_at ASC, created_at ASC
  FOR UPDATE SKIP LOCKED
  LIMIT 1
)
UPDATE fiscal_outbox_events e SET
  status='leased',
  lease_owner=$1,
  lease_token=$2,
  lease_expires_at=$3,
  claim_count=e.claim_count+1,
  updated_at=now()
FROM next_event
WHERE e.id=next_event.id
RETURNING e.id,e.fiscal_document_id,e.event_type,e.operation_key,e.sequence_no,e.claim_count,e.artifact_id::text`,
		owner, token, expires).Scan(&id, &docID, &eventType, &operationKey, &sequenceNo, &claimCount, &artifactRaw)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ports.ErrFiscalOutboxEmpty
	}
	if err != nil {
		return nil, fmt.Errorf("claim outbox: %w", err)
	}

	doc, err := lockDocument(ctx, tx, docID)
	if err != nil {
		return nil, err
	}
	var canonical []byte
	_ = tx.QueryRowContext(ctx, `SELECT COALESCE(canonical_bytes,'') FROM fiscal_snapshots WHERE fiscal_document_id=$1`, docID).Scan(&canonical)

	var startedID uuid.UUID
	hasStarted := false
	err = tx.QueryRowContext(ctx, `
SELECT id FROM fiscal_provider_attempts
WHERE outbox_event_id=$1 AND status='started'
ORDER BY attempt_no DESC LIMIT 1`, id).Scan(&startedID)
	if err == nil {
		hasStarted = true
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	var art *uuid.UUID
	if artifactRaw.Valid && artifactRaw.String != "" {
		parsed, perr := uuid.Parse(artifactRaw.String)
		if perr == nil {
			art = &parsed
		}
	}
	return &ports.ClaimedOutboxEvent{
		Event: ports.FiscalOutboxEvent{
			ID: id, FiscalDocumentID: docID, EventType: ports.FiscalOutboxEventType(eventType),
			OperationKey: operationKey, SequenceNo: sequenceNo, Status: ports.FiscalOutboxLeased,
			LeaseOwner: owner, LeaseToken: token, LeaseExpiresAt: &expires, ClaimCount: claimCount,
			ArtifactID: art,
		},
		Document:          *doc,
		CanonicalBytes:    append([]byte(nil), canonical...),
		HasStartedAttempt: hasStarted,
		StartedAttemptID:  startedID,
	}, nil
}

// BeginLegalCall transitions the document when needed and inserts a started attempt under lease fencing.
func (r *FiscalOutboxRepository) BeginLegalCall(ctx context.Context, cmd ports.BeginLegalCallCommand) (*ports.FiscalProviderAttempt, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	if err := requireActiveLease(ctx, tx, cmd.EventID, cmd.LeaseOwner, cmd.LeaseToken); err != nil {
		return nil, err
	}
	doc, err := lockDocument(ctx, tx, cmd.DocumentID)
	if err != nil {
		return nil, err
	}
	from := doc.State
	if cmd.NextDocumentState != "" && cmd.NextDocumentState != doc.State {
		if err := doc.TransitionTo(cmd.NextDocumentState, time.Now().UTC()); err != nil {
			return nil, err
		}
		if _, err := tx.ExecContext(ctx, `
UPDATE fiscal_documents SET state=$2,version=version+1,updated_at=now() WHERE id=$1`,
			cmd.DocumentID, string(cmd.NextDocumentState)); err != nil {
			return nil, err
		}
		if _, err := tx.ExecContext(ctx, `
INSERT INTO fiscal_state_transitions(fiscal_document_id,from_state,to_state,operation,actor_type,correlation_key)
VALUES ($1,$2,$3,$4,'worker',$5)`,
			cmd.DocumentID, string(from), string(cmd.NextDocumentState), string(cmd.Operation), cmd.EventID.String()); err != nil {
			return nil, err
		}
	}

	var attemptNo int
	if err := tx.QueryRowContext(ctx, `
SELECT COALESCE(MAX(attempt_no),0)+1 FROM fiscal_provider_attempts
WHERE fiscal_document_id=$1 AND operation=$2`, cmd.DocumentID, string(cmd.Operation)).Scan(&attemptNo); err != nil {
		return nil, err
	}
	attemptID := uuid.New()
	now := time.Now().UTC()
	if _, err := tx.ExecContext(ctx, `
INSERT INTO fiscal_provider_attempts(
 id,fiscal_document_id,outbox_event_id,provider_key,operation,operation_key,attempt_no,status,request_sha256,started_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,'started',$8,$9)`,
		attemptID, cmd.DocumentID, cmd.EventID, cmd.ProviderKey, string(cmd.Operation), cmd.OperationKey,
		attemptNo, cmd.RequestSHA256, now); err != nil {
		return nil, fmt.Errorf("insert started attempt: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &ports.FiscalProviderAttempt{
		ID: attemptID, FiscalDocumentID: cmd.DocumentID, OutboxEventID: cmd.EventID,
		ProviderKey: cmd.ProviderKey, Operation: cmd.Operation, OperationKey: cmd.OperationKey,
		AttemptNo: attemptNo, Status: ports.FiscalAttemptStarted, RequestSHA256: cmd.RequestSHA256,
		StartedAt: now,
	}, nil
}

// RecoverStaleAttempt marks an unfinished started attempt unknown and completes the leased event.
func (r *FiscalOutboxRepository) RecoverStaleAttempt(ctx context.Context, cmd ports.RecoverStaleAttemptCommand) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if err := requireActiveLease(ctx, tx, cmd.EventID, cmd.LeaseOwner, cmd.LeaseToken); err != nil {
		return err
	}
	doc, err := lockDocument(ctx, tx, cmd.DocumentID)
	if err != nil {
		return err
	}
	from := doc.State
	now := time.Now().UTC()
	res, err := tx.ExecContext(ctx, `
UPDATE fiscal_provider_attempts SET status='unknown',classification='ambiguous',definitive=false,finished_at=$2,
 safe_provider_code='lease_expired',safe_provider_message=$3
WHERE id=$1 AND status='started'`, cmd.AttemptID, now, nullIfEmpty(cmd.SafeError))
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ports.ErrFiscalAttemptNotStarted
	}

	if cmd.UnknownState != "" && cmd.UnknownState != doc.State {
		if err := doc.TransitionTo(cmd.UnknownState, now); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `
UPDATE fiscal_documents SET state=$2,version=version+1,last_error_class='ambiguous',last_error_code='lease_expired',
 last_error_message=$3,updated_at=now() WHERE id=$1`,
			cmd.DocumentID, string(cmd.UnknownState), nullIfEmpty(cmd.SafeError)); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `
INSERT INTO fiscal_state_transitions(fiscal_document_id,from_state,to_state,operation,actor_type,correlation_key)
VALUES ($1,$2,$3,'recover_stale','worker',$4)`,
			cmd.DocumentID, string(from), string(cmd.UnknownState), cmd.EventID.String()); err != nil {
			return err
		}
	}
	if err := completeLeasedEvent(ctx, tx, cmd.EventID, cmd.LeaseOwner, cmd.LeaseToken, cmd.SafeError); err != nil {
		return err
	}
	return tx.Commit()
}

// CompleteCall applies the normalized outcome under lease fencing in one transaction.
func (r *FiscalOutboxRepository) CompleteCall(ctx context.Context, cmd ports.CompleteOutboxCallCommand) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if err := requireActiveLease(ctx, tx, cmd.EventID, cmd.LeaseOwner, cmd.LeaseToken); err != nil {
		return err
	}
	doc, err := lockDocument(ctx, tx, cmd.DocumentID)
	if err != nil {
		return err
	}
	from := doc.State
	now := time.Now().UTC()
	diag := []byte("{}")
	if cmd.Diagnostics != nil {
		encoded, merr := json.Marshal(cmd.Diagnostics)
		if merr != nil {
			return merr
		}
		if len(encoded) > 0 && string(encoded) != "null" {
			diag = encoded
		}
	}
	class := nullIfEmpty(string(cmd.Classification))
	res, err := tx.ExecContext(ctx, `
UPDATE fiscal_provider_attempts SET status=$2,classification=$3,definitive=$4,safe_provider_code=$5,safe_provider_message=$6,
 diagnostics=$7::jsonb,provider_reference=COALESCE(NULLIF($8,''),provider_reference),finished_at=$9
WHERE id=$1 AND status='started'`,
		cmd.AttemptID, string(cmd.AttemptStatus), class, cmd.Definitive,
		nullIfEmpty(cmd.SafeProviderCode), nullIfEmpty(cmd.SafeProviderMsg), string(diag),
		cmd.ProviderReference, now)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ports.ErrFiscalAttemptNotStarted
	}

	if cmd.NextDocumentState != "" && cmd.NextDocumentState != doc.State {
		if err := doc.TransitionTo(cmd.NextDocumentState, now); err != nil {
			return err
		}
	}
	issuedAt := sql.NullTime{}
	voidedAt := sql.NullTime{}
	if cmd.IssuedAt != nil {
		issuedAt = sql.NullTime{Time: *cmd.IssuedAt, Valid: true}
	} else if cmd.NextDocumentState == domain.FiscalDocumentStateIssued {
		issuedAt = sql.NullTime{Time: now, Valid: true}
	}
	if cmd.VoidedAt != nil {
		voidedAt = sql.NullTime{Time: *cmd.VoidedAt, Valid: true}
	} else if cmd.NextDocumentState == domain.FiscalDocumentStateVoided {
		voidedAt = sql.NullTime{Time: now, Valid: true}
	}

	confirmIssued := cmd.NextDocumentState == domain.FiscalDocumentStateIssued
	if _, err := tx.ExecContext(ctx, `
UPDATE fiscal_documents SET
 state=$2,
 version=version+1,
 provider_reference=COALESCE(NULLIF($3::text,''),provider_reference),
 provider_number=COALESCE(NULLIF($4::text,''),provider_number),
 issued_at=COALESCE($5::timestamptz,issued_at),
 voided_at=COALESCE($6::timestamptz,voided_at),
 provider_confirmed_at=CASE WHEN $10::boolean THEN COALESCE(provider_confirmed_at,now()) ELSE provider_confirmed_at END,
 last_error_class=$7,
 last_error_code=$8,
 last_error_message=$9,
 updated_at=now()
WHERE id=$1`,
		cmd.DocumentID, string(cmd.NextDocumentState), cmd.ProviderReference, cmd.ProviderNumber,
		issuedAt, voidedAt, class, nullIfEmpty(cmd.SafeProviderCode), nullIfEmpty(cmd.SafeProviderMsg),
		confirmIssued); err != nil {
		return err
	}
	if from != cmd.NextDocumentState {
		if _, err := tx.ExecContext(ctx, `
INSERT INTO fiscal_state_transitions(fiscal_document_id,from_state,to_state,operation,actor_type,correlation_key)
VALUES ($1,$2,$3,'complete','worker',$4)`,
			cmd.DocumentID, string(from), string(cmd.NextDocumentState), cmd.EventID.String()); err != nil {
			return err
		}
	}
	if err := completeLeasedEvent(ctx, tx, cmd.EventID, cmd.LeaseOwner, cmd.LeaseToken, cmd.LastSafeError); err != nil {
		return err
	}
	return tx.Commit()
}

func requireActiveLease(ctx context.Context, tx *sql.Tx, eventID uuid.UUID, owner string, token uuid.UUID) error {
	var ok bool
	err := tx.QueryRowContext(ctx, `
SELECT true FROM fiscal_outbox_events
WHERE id=$1 AND status='leased' AND lease_owner=$2 AND lease_token=$3
  AND lease_expires_at IS NOT NULL AND lease_expires_at > now()`,
		eventID, owner, token).Scan(&ok)
	if errors.Is(err, sql.ErrNoRows) {
		return ports.ErrFiscalLeaseLost
	}
	return err
}

func completeLeasedEvent(ctx context.Context, tx *sql.Tx, eventID uuid.UUID, owner string, token uuid.UUID, safeError string) error {
	res, err := tx.ExecContext(ctx, `
UPDATE fiscal_outbox_events SET status='completed',lease_owner=NULL,lease_token=NULL,lease_expires_at=NULL,
 last_safe_error=COALESCE(NULLIF($4,''),last_safe_error),updated_at=now()
WHERE id=$1 AND status='leased' AND lease_owner=$2 AND lease_token=$3
  AND lease_expires_at IS NOT NULL AND lease_expires_at > now()`,
		eventID, owner, token, safeError)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ports.ErrFiscalLeaseLost
	}
	return nil
}
