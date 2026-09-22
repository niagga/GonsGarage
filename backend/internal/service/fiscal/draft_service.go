package fiscal

import (
	"context"
	"fmt"
	"strings"

	"github.com/gaston-garcia-cegid/gonsgarage/internal/core/ports"
	"github.com/gaston-garcia-cegid/gonsgarage/internal/domain"
	"github.com/google/uuid"
)

// DraftService validates and persists mutable fiscal drafts.
type DraftService struct {
	repo     ports.FiscalRepository
	invoices ports.InvoiceRepository
	users    ports.UserRepository
}

// NewDraftService builds the staff draft application service.
func NewDraftService(repo ports.FiscalRepository, invoices ports.InvoiceRepository, users ports.UserRepository) *DraftService {
	return &DraftService{repo: repo, invoices: invoices, users: users}
}

// CreateDraftRequest creates a mutable FT/FR draft for an eligible invoice.
type CreateDraftRequest struct {
	SourceInvoiceID uuid.UUID
	IntentSlot      string
	Kind            domain.DocumentKind
	Snapshot        ports.FiscalSnapshotRecord
}

// CreateDraft stores a new current-intent draft for staff callers.
func (s *DraftService) CreateDraft(ctx context.Context, actorID uuid.UUID, req CreateDraftRequest) (*ports.FiscalDocumentAggregate, error) {
	role, err := s.requireStaff(ctx, actorID)
	if err != nil {
		return nil, err
	}
	_ = role
	inv, err := s.invoices.GetByID(ctx, req.SourceInvoiceID)
	if err != nil {
		return nil, err
	}
	if inv.FiscalEligibility != domain.FiscalEligibilityEligible {
		return nil, ports.ErrFiscalInvoiceNotEligible
	}
	if !req.Kind.IsValid() {
		return nil, fmt.Errorf("%w: unsupported kind", ports.ErrFiscalActionNotAllowed)
	}
	slot := strings.TrimSpace(req.IntentSlot)
	if slot == "" {
		slot = "primary_sale"
	}
	return s.repo.CreateDraft(ctx, ports.CreateDraftInput{
		SourceInvoiceID: req.SourceInvoiceID,
		IntentSlot:      slot,
		Kind:            req.Kind,
		CreatedBy:       actorID,
		Snapshot:        req.Snapshot,
	})
}

// UpdateDraft applies an optimistic draft snapshot update.
func (s *DraftService) UpdateDraft(ctx context.Context, actorID, documentID uuid.UUID, expectedVersion int64, snapshot ports.FiscalSnapshotRecord) (*ports.FiscalDocumentAggregate, error) {
	if _, err := s.requireStaff(ctx, actorID); err != nil {
		return nil, err
	}
	return s.repo.UpdateDraft(ctx, documentID, expectedVersion, snapshot)
}

// DeleteDraft removes a mutable draft under optimistic concurrency.
func (s *DraftService) DeleteDraft(ctx context.Context, actorID, documentID uuid.UUID, expectedVersion int64) error {
	if _, err := s.requireStaff(ctx, actorID); err != nil {
		return err
	}
	return s.repo.DeleteDraft(ctx, documentID, expectedVersion)
}

// GetDraft loads a draft/document aggregate for staff.
func (s *DraftService) GetDraft(ctx context.Context, actorID, documentID uuid.UUID) (*ports.FiscalDocumentAggregate, error) {
	if _, err := s.requireStaff(ctx, actorID); err != nil {
		return nil, err
	}
	return s.repo.GetAggregate(ctx, documentID)
}

func (s *DraftService) requireStaff(ctx context.Context, actorID uuid.UUID) (domain.FiscalActorRole, error) {
	u, err := s.users.GetByID(ctx, actorID)
	if err != nil {
		return "", err
	}
	if u == nil {
		return "", domain.ErrUserNotFound
	}
	switch u.Role {
	case domain.RoleEmployee:
		return domain.FiscalActorRoleEmployee, nil
	case domain.RoleManager:
		return domain.FiscalActorRoleManager, nil
	case domain.RoleAdmin:
		return domain.FiscalActorRoleAdmin, nil
	default:
		return "", domain.ErrUnauthorizedAccess
	}
}
