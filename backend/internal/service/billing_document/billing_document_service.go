package billing_document

import (
	"context"
	"fmt"

	"github.com/gaston-garcia-cegid/gonsgarage/internal/core/ports"
	"github.com/gaston-garcia-cegid/gonsgarage/internal/domain"
	"github.com/google/uuid"
)

// BillingDocumentService implements accounting CRUD for issued billing documents.
type BillingDocumentService struct {
	repo     ports.BillingDocumentRepository
	userRepo ports.UserRepository
}

func NewBillingDocumentService(repo ports.BillingDocumentRepository, userRepo ports.UserRepository) *BillingDocumentService {
	return &BillingDocumentService{repo: repo, userRepo: userRepo}
}

var _ ports.BillingDocumentService = (*BillingDocumentService)(nil)

func (s *BillingDocumentService) requireAccountingAccess(ctx context.Context, requestingUserID uuid.UUID) (*domain.User, error) {
	u, err := s.userRepo.GetByID(ctx, requestingUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if u == nil {
		return nil, domain.ErrUserNotFound
	}
	if !u.CanManageUsers() {
		return nil, domain.ErrUnauthorizedAccess
	}
	return u, nil
}

// Create persists a billing document after domain validation; only manager/admin.
func (s *BillingDocumentService) Create(ctx context.Context, doc *domain.BillingDocument, requestingUserID uuid.UUID) (*domain.BillingDocument, error) {
	if _, err := s.requireAccountingAccess(ctx, requestingUserID); err != nil {
		return nil, err
	}
	if doc == nil {
		return nil, fmt.Errorf("billing document is required")
	}
	if err := doc.Validate(); err != nil {
		return nil, err
	}
	if doc.ID == uuid.Nil {
		doc.ID = uuid.New()
	}
	if err := s.repo.Create(ctx, doc); err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, doc.ID)
}

func (s *BillingDocumentService) Get(ctx context.Context, id uuid.UUID, requestingUserID uuid.UUID) (*domain.BillingDocument, error) {
	if _, err := s.requireAccountingAccess(ctx, requestingUserID); err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, id)
}

func (s *BillingDocumentService) List(ctx context.Context, requestingUserID uuid.UUID, limit, offset int) ([]*domain.BillingDocument, int64, error) {
	if _, err := s.requireAccountingAccess(ctx, requestingUserID); err != nil {
		return nil, 0, err
	}
	return s.repo.List(ctx, limit, offset)
}

func (s *BillingDocumentService) Update(ctx context.Context, doc *domain.BillingDocument, requestingUserID uuid.UUID) (*domain.BillingDocument, error) {
	if _, err := s.requireAccountingAccess(ctx, requestingUserID); err != nil {
		return nil, err
	}
	if doc == nil {
		return nil, fmt.Errorf("billing document is required")
	}
	if err := doc.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.Update(ctx, doc); err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, doc.ID)
}

func (s *BillingDocumentService) Delete(ctx context.Context, id uuid.UUID, requestingUserID uuid.UUID) error {
	if _, err := s.requireAccountingAccess(ctx, requestingUserID); err != nil {
		return err
	}
	return s.repo.Delete(ctx, id)
}
