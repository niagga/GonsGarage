package ports

import (
	"context"
	"time"

	"github.com/gaston-garcia-cegid/gonsgarage/internal/domain"
	"github.com/google/uuid"
)

// UserRepository define os métodos para o repositório de usuários
type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	GetByRole(ctx context.Context, role string, limit, offset int) ([]*domain.User, error)
	List(ctx context.Context, limit, offset int) ([]*domain.User, error)
	Update(ctx context.Context, user *domain.User) error
	Delete(ctx context.Context, id uuid.UUID) error
	UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string) error
	GetActiveUsers(ctx context.Context, limit, offset int) ([]*domain.User, error)
}

// EmployeeRepository define os métodos para o repositório de funcionários
type EmployeeRepository interface {
	Create(ctx context.Context, employee *domain.Employee) error
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Employee, error)
	Update(ctx context.Context, employee *domain.Employee) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, filters *EmployeeFilters) ([]*domain.Employee, int64, error)
}

// EmployeeFilters representa os filtros para listagem de funcionários
type EmployeeFilters struct {
	Department *string
	IsActive   *bool
	Search     *string
	SortBy     string
	SortOrder  string
	Limit      int
	Offset     int
}

// CacheRepository define os métodos para o repositório de cache
type CacheRepository interface {
	Get(ctx context.Context, key string, dest interface{}) error
	Set(ctx context.Context, key string, value interface{}, ttl int) error
	Delete(ctx context.Context, key string) error
}

// SessionRepository define os métodos para o repositório de sessões
type SessionRepository interface {
	Create(ctx context.Context, session *domain.Session) error
	GetByID(ctx context.Context, id string) (*domain.Session, error)
	Update(ctx context.Context, session *domain.Session) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, limit, offset int) ([]*domain.Session, int64, error)
	SearchByName(ctx context.Context, name string, limit int) ([]*domain.Session, error)
}

// QueueRepository define os métodos para o repositório de filas
type QueueRepository interface {
	Enqueue(ctx context.Context, job *domain.Job) error
	Dequeue(ctx context.Context) (*domain.Job, error)
	Acknowledge(ctx context.Context, id string) error
	Reject(ctx context.Context, id string) error
	List(ctx context.Context, limit, offset int) ([]*domain.Job, int64, error)
	SearchByName(ctx context.Context, name string, limit int) ([]*domain.Job, error)
}

// WorkshopRepository defines the interface for the workshop repository
type WorkshopRepository interface {
	CreateWorkshop(ctx context.Context, workshop *domain.Workshop) error
	UpdateWorkshop(ctx context.Context, workshop *domain.Workshop) error
	DeleteWorkshop(ctx context.Context, id string) error
	GetWorkshopByID(ctx context.Context, id string) (*domain.Workshop, error)
	ListWorkshops(ctx context.Context, limit, offset int) ([]*domain.Workshop, int64, error)
}

// NotificationRepository defines the interface for the notification repository
type NotificationRepository interface {
	SendWorkshopCreatedNotification(ctx context.Context, workshop *domain.Workshop) error
	SendWorkshopUpdatedNotification(ctx context.Context, workshop *domain.Workshop) error
	SendWorkshopDeletedNotification(ctx context.Context, workshop *domain.Workshop) error
}

// AccountingRepository defines the interface for the accounting repository
type AccountingRepository interface {
	CreateAccountingEntry(ctx context.Context, entry *domain.AccountingEntry) error
	GetAccountingEntryByID(ctx context.Context, id string) (*domain.AccountingEntry, error)
	UpdateAccountingEntry(ctx context.Context, entry *domain.AccountingEntry) error
	DeleteAccountingEntry(ctx context.Context, id string) error
	ListAccountingEntries(ctx context.Context, limit, offset int) ([]*domain.AccountingEntry, int64, error)
}

// CarRepository defines the interface for the car repository
type CarRepository interface {
	Create(ctx context.Context, car *domain.Car) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Car, error)
	GetByOwnerID(ctx context.Context, ownerID uuid.UUID) ([]*domain.Car, error)
	GetByLicensePlate(ctx context.Context, licensePlate string) (*domain.Car, error)
	List(ctx context.Context, limit, offset int) ([]*domain.Car, error)
	Update(ctx context.Context, car *domain.Car) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetWithRepairs(ctx context.Context, id uuid.UUID) (*domain.Car, error)
	GetDeletedByLicensePlate(ctx context.Context, licensePlate string) (*domain.Car, error)
	Restore(ctx context.Context, id uuid.UUID) error
}

// RepairRepository defines the interface for the repair repository
type RepairRepository interface {
	Create(ctx context.Context, repair *domain.Repair) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Repair, error)
	Update(ctx context.Context, repair *domain.Repair) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetByCarID(ctx context.Context, carID uuid.UUID) ([]*domain.Repair, error)
	// ListIDsByServiceJobID returns repair row IDs linked to a visit; empty if none.
	ListIDsByServiceJobID(ctx context.Context, serviceJobID uuid.UUID) ([]uuid.UUID, error)
}

// ServiceJobRepository persists workshop visits (service jobs) and 1:1 reception/handover.
type ServiceJobRepository interface {
	Create(ctx context.Context, job *domain.ServiceJob) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.ServiceJob, error)
	Update(ctx context.Context, job *domain.ServiceJob) error
	ListByCarID(ctx context.Context, carID uuid.UUID) ([]*domain.ServiceJob, error)
	// ListByOpenedOn returns visits whose OpenedAt falls in [day 00:00 UTC, next day 00:00 UTC). Day is normalized to UTC date (year, month, day only).
	ListByOpenedOn(ctx context.Context, day time.Time) ([]*domain.ServiceJob, error)
	SaveReception(ctx context.Context, r *domain.ServiceJobReception) error
	GetReception(ctx context.Context, serviceJobID uuid.UUID) (*domain.ServiceJobReception, error)
	SaveHandover(ctx context.Context, h *domain.ServiceJobHandover) error
	GetHandover(ctx context.Context, serviceJobID uuid.UUID) (*domain.ServiceJobHandover, error)
}

// InvoiceRepository persists invoices (customer-scoped access enforced in InvoiceService).
// Delete protection for frozen fiscal history is provided by FiscalProtectionReader
// (see fiscal_repository.go) and wired through InvoiceService.WithFiscalProtection.
type InvoiceRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Invoice, error)
	// GetByRepairID returns the internal invoice linked to a repair, or ErrInvoiceNotFound.
	GetByRepairID(ctx context.Context, repairID uuid.UUID) (*domain.Invoice, error)
	Create(ctx context.Context, invoice *domain.Invoice) error
	Update(ctx context.Context, invoice *domain.Invoice) error
	Delete(ctx context.Context, id uuid.UUID) error
	ListByCustomerID(ctx context.Context, customerID uuid.UUID, limit, offset int) ([]*domain.Invoice, int64, error)
	// ListForStaff lists customer invoices for workshop staff (issued to clients).
	ListForStaff(ctx context.Context, limit, offset int) ([]*domain.Invoice, int64, error)
}

// ReceivedInvoiceRepository persists purchase-side invoices received by the workshop.
type ReceivedInvoiceRepository interface {
	Create(ctx context.Context, inv *domain.ReceivedInvoice) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.ReceivedInvoice, error)
	Update(ctx context.Context, inv *domain.ReceivedInvoice) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, limit, offset int) ([]*domain.ReceivedInvoice, int64, error)
}

// BillingDocumentRepository persists issued billing documents (payroll, IRS, client invoice, etc.).
type BillingDocumentRepository interface {
	Create(ctx context.Context, doc *domain.BillingDocument) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.BillingDocument, error)
	Update(ctx context.Context, doc *domain.BillingDocument) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, limit, offset int) ([]*domain.BillingDocument, int64, error)
}

// SupplierRepository persists suppliers.
type SupplierRepository interface {
	Create(ctx context.Context, s *domain.Supplier) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Supplier, error)
	Update(ctx context.Context, s *domain.Supplier) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, limit, offset int) ([]*domain.Supplier, int64, error)
}

// PartItemListFilters drives listing spare parts (barcode exact, or text search).
type PartItemListFilters struct {
	Barcode *string
	Search  *string
	Limit   int
	Offset  int
}

// PartItemRepository persists spare-part catalog rows (no auth rules here).
type PartItemRepository interface {
	Create(ctx context.Context, p *domain.PartItem) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.PartItem, error)
	GetByBarcode(ctx context.Context, barcode string) (*domain.PartItem, error)
	Update(ctx context.Context, p *domain.PartItem) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, f PartItemListFilters) ([]*domain.PartItem, int64, error)
}

// Logger defines the interface for logging
type Logger interface {
	Info(msg string, keysAndValues ...interface{})
	Error(msg string, keysAndValues ...interface{})
}

// AppointmentRepository defines the interface for the appointment repository
type AppointmentRepository interface {
	Create(ctx context.Context, appointment *domain.Appointment) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Appointment, error)
	Update(ctx context.Context, appointment *domain.Appointment) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, filters *AppointmentFilters) ([]*domain.Appointment, int64, error)
	// CountNonCancelledBetween counts appointments with scheduled_at in [start, end) (UTC), excluding cancelled, optionally excluding an id (e.g. current row on update).
	CountNonCancelledBetween(ctx context.Context, start, end time.Time, excludeID *uuid.UUID) (int64, error)
}

// AppointmentFilters represents filters for listing appointments
type AppointmentFilters struct {
	CustomerID  *uuid.UUID
	EmployeeID  *uuid.UUID
	CarID       *uuid.UUID
	ScheduledAt *time.Time
	Reason      *string
	Status      *string
	SortBy      string
	SortOrder   string
	Limit       int
	Offset      int
}
