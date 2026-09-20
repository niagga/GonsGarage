package received_invoice

import (
	"context"
	"fmt"

	"github.com/gaston-garcia-cegid/gonsgarage/internal/core/ports"
	"github.com/gaston-garcia-cegid/gonsgarage/internal/domain"
	"github.com/google/uuid"
)

// ReceivedInvoiceService implements accounting CRUD for purchase-side received invoices.
type ReceivedInvoiceService struct {
	repo     ports.ReceivedInvoiceRepository
	userRepo ports.UserRepository
}

func NewReceivedInvoiceService(repo ports.ReceivedInvoiceRepository, userRepo ports.UserRepository) *ReceivedInvoiceService {
	return &ReceivedInvoiceService{repo: repo, userRepo: userRepo}
}

var _ ports.ReceivedInvoiceService = (*ReceivedInvoiceService)(nil)

func (s *ReceivedInvoiceService) requireAccountingAccess(ctx context.Context, requestingUserID uuid.UUID) (*domain.User, error) {
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

// Create persists a received invoice after domain validation; only manager/admin.
func (s *ReceivedInvoiceService) Create(ctx context.Context, inv *domain.ReceivedInvoice, requestingUserID uuid.UUID) (*domain.ReceivedInvoice, error) {
	if _, err := s.requireAccountingAccess(ctx, requestingUserID); err != nil {
		return nil, err
	}
	if inv == nil {
		return nil, fmt.Errorf("received invoice is required")
	}
	if err := inv.Validate(); err != nil {
		return nil, err
	}
	if inv.ID == uuid.Nil {
		inv.ID = uuid.New()
	}
	if err := s.repo.Create(ctx, inv); err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, inv.ID)
}

func (s *ReceivedInvoiceService) Get(ctx context.Context, id uuid.UUID, requestingUserID uuid.UUID) (*domain.ReceivedInvoice, error) {
	if _, err := s.requireAccountingAccess(ctx, requestingUserID); err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, id)
}

func (s *ReceivedInvoiceService) List(ctx context.Context, requestingUserID uuid.UUID, limit, offset int) ([]*domain.ReceivedInvoice, int64, error) {
	if _, err := s.requireAccountingAccess(ctx, requestingUserID); err != nil {
		return nil, 0, err
	}
	return s.repo.List(ctx, limit, offset)
}

func (s *ReceivedInvoiceService) Update(ctx context.Context, inv *domain.ReceivedInvoice, requestingUserID uuid.UUID) (*domain.ReceivedInvoice, error) {
	if _, err := s.requireAccountingAccess(ctx, requestingUserID); err != nil {
		return nil, err
	}
	if inv == nil {
		return nil, fmt.Errorf("received invoice is required")
	}
	if err := inv.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.Update(ctx, inv); err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, inv.ID)
}

func (s *ReceivedInvoiceService) Delete(ctx context.Context, id uuid.UUID, requestingUserID uuid.UUID) error {
	if _, err := s.requireAccountingAccess(ctx, requestingUserID); err != nil {
		return err
	}
	return s.repo.Delete(ctx, id)
}
