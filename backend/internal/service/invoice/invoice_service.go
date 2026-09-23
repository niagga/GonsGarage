package invoice

import (
	"context"
	"errors"
	"fmt"

	"github.com/gaston-garcia-cegid/gonsgarage/internal/core/ports"
	"github.com/gaston-garcia-cegid/gonsgarage/internal/domain"
	"github.com/google/uuid"
)

type InvoiceService struct {
	invoiceRepo      ports.InvoiceRepository
	userRepo         ports.UserRepository
	repairRepo       ports.RepairRepository
	carRepo          ports.CarRepository
	fiscalProtection ports.FiscalProtectionReader
}

var _ ports.InvoiceService = (*InvoiceService)(nil)

func NewInvoiceService(
	invoiceRepo ports.InvoiceRepository,
	userRepo ports.UserRepository,
	repairRepo ports.RepairRepository,
	carRepo ports.CarRepository,
) *InvoiceService {
	return &InvoiceService{
		invoiceRepo: invoiceRepo,
		userRepo:    userRepo,
		repairRepo:  repairRepo,
		carRepo:     carRepo,
	}
}

// WithFiscalProtection attaches the narrow fiscal delete guard without changing legacy constructors.
func (s *InvoiceService) WithFiscalProtection(reader ports.FiscalProtectionReader) *InvoiceService {
	s.fiscalProtection = reader
	return s
}

func (s *InvoiceService) GetInvoice(ctx context.Context, invoiceID uuid.UUID, requestingUserID uuid.UUID) (*domain.Invoice, error) {
	u, err := s.userRepo.GetByID(ctx, requestingUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if u == nil {
		return nil, domain.ErrUserNotFound
	}

	inv, err := s.invoiceRepo.GetByID(ctx, invoiceID)
	if err != nil {
		return nil, err
	}
	if inv == nil {
		return nil, domain.ErrInvoiceNotFound
	}

	if u.IsClient() && inv.CustomerID != requestingUserID {
		return nil, domain.ErrUnauthorizedAccess
	}
	if !u.IsClient() && !u.CanManageUsers() {
		return nil, domain.ErrUnauthorizedAccess
	}
	return inv, nil
}

// GetInvoiceByRepairID returns the internal invoice linked to a repair (staff/manager only).
func (s *InvoiceService) GetInvoiceByRepairID(ctx context.Context, repairID uuid.UUID, requestingUserID uuid.UUID) (*domain.Invoice, error) {
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
	inv, err := s.invoiceRepo.GetByRepairID(ctx, repairID)
	if err != nil {
		return nil, err
	}
	if inv == nil {
		return nil, domain.ErrInvoiceNotFound
	}
	return inv, nil
}

// UpdateInvoice merges updates. Clients may only update invoices they own and may only change Notes.
func (s *InvoiceService) UpdateInvoice(ctx context.Context, invoice *domain.Invoice, requestingUserID uuid.UUID) (*domain.Invoice, error) {
	if invoice == nil {
		return nil, fmt.Errorf("invoice is required")
	}
	u, err := s.userRepo.GetByID(ctx, requestingUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if u == nil {
		return nil, domain.ErrUserNotFound
	}

	existing, err := s.invoiceRepo.GetByID(ctx, invoice.ID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, domain.ErrInvoiceNotFound
	}

	if u.IsClient() {
		if existing.CustomerID != requestingUserID {
			return nil, domain.ErrUnauthorizedAccess
		}
		merged := *existing
		merged.Notes = invoice.Notes
		if err := s.invoiceRepo.Update(ctx, &merged); err != nil {
			return nil, err
		}
		return s.invoiceRepo.GetByID(ctx, merged.ID)
	}

	if !u.CanManageUsers() {
		return nil, domain.ErrUnauthorizedAccess
	}

	merged := *existing
	merged.Notes = invoice.Notes
	if invoice.Status != "" {
		merged.Status = invoice.Status
	}
	if invoice.Amount != 0 {
		merged.Amount = invoice.Amount
	}
	if err := s.invoiceRepo.Update(ctx, &merged); err != nil {
		return nil, err
	}
	return s.invoiceRepo.GetByID(ctx, merged.ID)
}

func (s *InvoiceService) ListMyInvoices(ctx context.Context, requestingUserID uuid.UUID, limit, offset int) ([]*domain.Invoice, int64, error) {
	u, err := s.userRepo.GetByID(ctx, requestingUserID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get user: %w", err)
	}
	if u == nil {
		return nil, 0, domain.ErrUserNotFound
	}
	if !u.IsClient() {
		return nil, 0, domain.ErrUnauthorizedAccess
	}
	limit, offset = clampInvoiceListParams(limit, offset)
	return s.invoiceRepo.ListByCustomerID(ctx, requestingUserID, limit, offset)
}

func clampInvoiceListParams(limit, offset int) (int, int) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}

	// CreateInvoice persists a customer invoice. Only manager/admin may create.
// When RepairID is set, the repair must be completed, customer must own the car,
// and at most one invoice may exist per repair (internal billing link).
func (s *InvoiceService) CreateInvoice(ctx context.Context, invoice *domain.Invoice, requestingUserID uuid.UUID) (*domain.Invoice, error) {
	if invoice == nil {
		return nil, fmt.Errorf("invoice is required")
	}
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

	toSave := *invoice
	if toSave.RepairID != nil && *toSave.RepairID != uuid.Nil {
		if err := s.applyRepairLink(ctx, &toSave); err != nil {
			return nil, err
		}
	}

	if toSave.CustomerID == uuid.Nil {
		return nil, fmt.Errorf("customer id is required")
	}
	if toSave.Amount <= 0 {
		return nil, fmt.Errorf("amount must be positive")
	}
	cust, err := s.userRepo.GetByID(ctx, toSave.CustomerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get customer: %w", err)
	}
	if cust == nil {
		return nil, domain.ErrUserNotFound
	}
	if !cust.IsClient() {
		return nil, fmt.Errorf("invoice customer must be a client user")
	}

	if toSave.ID == uuid.Nil {
		toSave.ID = uuid.New()
	}
	if toSave.Status == "" {
		toSave.Status = "open"
	}
	if toSave.FiscalEligibility == "" {
		toSave.FiscalEligibility = domain.FiscalEligibilityEligible
	}
	if err := s.invoiceRepo.Create(ctx, &toSave); err != nil {
		return nil, err
	}
	return s.invoiceRepo.GetByID(ctx, toSave.ID)
}

func (s *InvoiceService) applyRepairLink(ctx context.Context, inv *domain.Invoice) error {
	if s.repairRepo == nil || s.carRepo == nil {
		return fmt.Errorf("repair linking is not configured")
	}
	repairID := *inv.RepairID
	existing, err := s.invoiceRepo.GetByRepairID(ctx, repairID)
	if err != nil && !errors.Is(err, domain.ErrInvoiceNotFound) {
		return err
	}
	if existing != nil {
		return domain.ErrInvoiceAlreadyExistsForRepair
	}

	repair, err := s.repairRepo.GetByID(ctx, repairID)
	if err != nil {
		if errors.Is(err, domain.ErrRepairNotFound) {
			return domain.ErrRepairNotFound
		}
		return fmt.Errorf("failed to get repair: %w", err)
	}
	if repair == nil {
		return domain.ErrRepairNotFound
	}
	if repair.Status != domain.RepairStatusCompleted {
		return domain.ErrRepairNotCompleted
	}

	car, err := s.carRepo.GetByID(ctx, repair.CarID)
	if err != nil {
		if errors.Is(err, domain.ErrCarNotFound) {
			return domain.ErrCarNotFound
		}
		return fmt.Errorf("failed to get car: %w", err)
	}
	if car == nil {
		return domain.ErrCarNotFound
	}

	if inv.CustomerID == uuid.Nil {
		inv.CustomerID = car.OwnerID
	} else if inv.CustomerID != car.OwnerID {
		return domain.ErrUnauthorizedAccess
	}

	carID := repair.CarID
	inv.CarID = &carID
	if inv.Amount <= 0 {
		inv.Amount = repair.Cost
	}
	return nil
}

// ListInvoicesForStaff lists issued customer invoices for manager/admin.
func (s *InvoiceService) ListInvoicesForStaff(ctx context.Context, requestingUserID uuid.UUID, limit, offset int) ([]*domain.Invoice, int64, error) {
	u, err := s.userRepo.GetByID(ctx, requestingUserID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get user: %w", err)
	}
	if u == nil {
		return nil, 0, domain.ErrUserNotFound
	}
	if !u.CanManageUsers() {
		return nil, 0, domain.ErrUnauthorizedAccess
	}
	limit, offset = clampInvoiceListParams(limit, offset)
	return s.invoiceRepo.ListForStaff(ctx, limit, offset)
}

// DeleteInvoice removes a customer invoice. Only manager/admin may delete.
func (s *InvoiceService) DeleteInvoice(ctx context.Context, invoiceID uuid.UUID, requestingUserID uuid.UUID) error {
	u, err := s.userRepo.GetByID(ctx, requestingUserID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}
	if u == nil {
		return domain.ErrUserNotFound
	}
	if !u.CanManageUsers() {
		return domain.ErrUnauthorizedAccess
	}
	existing, err := s.invoiceRepo.GetByID(ctx, invoiceID)
	if err != nil {
		return err
	}
	if existing == nil {
		return domain.ErrInvoiceNotFound
	}
	if s.fiscalProtection != nil {
		protected, err := s.fiscalProtection.HasProtectedFiscalHistory(ctx, invoiceID)
		if err != nil {
			return err
		}
		if protected {
			return ports.ErrFiscalHistoryProtected
		}
	}
	return s.invoiceRepo.Delete(ctx, invoiceID)
}
