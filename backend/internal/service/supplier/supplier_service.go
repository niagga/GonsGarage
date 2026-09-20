package supplier

import (
	"context"
	"fmt"

	"github.com/gaston-garcia-cegid/gonsgarage/internal/core/ports"
	"github.com/gaston-garcia-cegid/gonsgarage/internal/domain"
	"github.com/google/uuid"
)

// SupplierService implements ports.SupplierService (accounting staff-only CRUD for suppliers).
type SupplierService struct {
	repo     ports.SupplierRepository
	userRepo ports.UserRepository
}

func NewSupplierService(repo ports.SupplierRepository, userRepo ports.UserRepository) *SupplierService {
	return &SupplierService{repo: repo, userRepo: userRepo}
}

var _ ports.SupplierService = (*SupplierService)(nil)

func (s *SupplierService) requireAccountingAccess(ctx context.Context, requestingUserID uuid.UUID) (*domain.User, error) {
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

// Create persists a supplier; only manager/admin.
func (s *SupplierService) Create(ctx context.Context, row *domain.Supplier, requestingUserID uuid.UUID) (*domain.Supplier, error) {
	if _, err := s.requireAccountingAccess(ctx, requestingUserID); err != nil {
		return nil, err
	}
	if row == nil {
		return nil, fmt.Errorf("supplier is required")
	}
	if err := row.Validate(); err != nil {
		return nil, err
	}
	if row.ID == uuid.Nil {
		row.ID = uuid.New()
	}
	if err := s.repo.Create(ctx, row); err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, row.ID)
}

// Get returns a supplier by id; only manager/admin.
func (s *SupplierService) Get(ctx context.Context, id uuid.UUID, requestingUserID uuid.UUID) (*domain.Supplier, error) {
	if _, err := s.requireAccountingAccess(ctx, requestingUserID); err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, id)
}

// List returns paginated suppliers; only manager/admin.
func (s *SupplierService) List(ctx context.Context, requestingUserID uuid.UUID, limit, offset int) ([]*domain.Supplier, int64, error) {
	if _, err := s.requireAccountingAccess(ctx, requestingUserID); err != nil {
		return nil, 0, err
	}
	return s.repo.List(ctx, limit, offset)
}

// Update updates a supplier; only manager/admin.
func (s *SupplierService) Update(ctx context.Context, row *domain.Supplier, requestingUserID uuid.UUID) (*domain.Supplier, error) {
	if _, err := s.requireAccountingAccess(ctx, requestingUserID); err != nil {
		return nil, err
	}
	if row == nil {
		return nil, fmt.Errorf("supplier is required")
	}
	if err := row.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.Update(ctx, row); err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, row.ID)
}

// Delete soft-deletes a supplier; only manager/admin.
func (s *SupplierService) Delete(ctx context.Context, id uuid.UUID, requestingUserID uuid.UUID) error {
	if _, err := s.requireAccountingAccess(ctx, requestingUserID); err != nil {
		return err
	}
	return s.repo.Delete(ctx, id)
}
