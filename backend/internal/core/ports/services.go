package ports

import (
	"context"
	"time"

	"github.com/gaston-garcia-cegid/gonsgarage/internal/domain"
	"github.com/google/uuid"
)

// AuthService define os métodos do serviço de autenticação
type AuthService interface {
	Login(ctx context.Context, email, password string) (string, error)
	Register(ctx context.Context, req RegisterRequest) (*domain.User, error)
	CurrentUser(ctx context.Context, userID uuid.UUID) (*domain.User, error)
	ValidateToken(token string) (*domain.User, error)
	RefreshToken(ctx context.Context, token string) (string, error)
	// ProvisionUser creates a user with manager/employee/client roles only (staff flow; caller must be admin or manager per service rules).
	ProvisionUser(ctx context.Context, callerUserID uuid.UUID, callerRole string, req ProvisionUserRequest) (*domain.User, error)
}

// ProvisionUserRequest is the body for POST /api/v1/admin/users (staff provisioning).
type ProvisionUserRequest struct {
	Email     string `json:"email" binding:"required,email"`
	Password  string `json:"password" binding:"required,min=6"`
	FirstName string `json:"firstName" binding:"required"`
	LastName  string `json:"lastName" binding:"required"`
	Role      string `json:"role" binding:"required"`
}

// RegisterRequest representa os dados para registro
type RegisterRequest struct {
	Email     string `json:"email" binding:"required,email"`
	Password  string `json:"password" binding:"required,min=6"`
	FirstName string `json:"firstName" binding:"required"`
	LastName  string `json:"lastName" binding:"required"`
	Role      string `json:"role"`
}

// LoginResponse representa a resposta do login
type LoginResponse struct {
	Token string       `json:"token"`
	User  *domain.User `json:"user"`
}

// RegisterResponse representa a resposta do registro
type RegisterResponse struct {
	User    *domain.User `json:"user"`
	Message string       `json:"message"`
}

// EmployeeService define os métodos do serviço de funcionários
type EmployeeService interface {
	CreateEmployee(ctx context.Context, req CreateEmployeeRequest) (*domain.Employee, error)
	GetEmployee(ctx context.Context, id uuid.UUID) (*domain.Employee, error)
	ListEmployees(ctx context.Context, filters *EmployeeFilters) ([]*domain.Employee, int64, error)
	UpdateEmployee(ctx context.Context, id uuid.UUID, req UpdateEmployeeRequest) (*domain.Employee, error)
	DeleteEmployee(ctx context.Context, id uuid.UUID) error
}

// CreateEmployeeRequest representa os dados para criar um funcionário
type CreateEmployeeRequest struct {
	UserID       uuid.UUID `json:"userId" gorm:"type:uuid;not null"`
	FirstName    string    `json:"firstName" gorm:"not null"`
	LastName     string    `json:"lastName" gorm:"not null"`
	Email        string    `json:"email" binding:"required,email"`
	Position     string    `json:"position" binding:"required"`
	HourlyRate   float64   `json:"hourlyRate" gorm:"not null"`
	HoursWorked  float64   `json:"hoursWorked" gorm:"default:0"`
	IsActive     bool      `json:"isActive" gorm:"default:true"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
	EmployeeCode string    `json:"employeeCode" gorm:"uniqueIndex;not null"`
	Department   string    `json:"department"`
	HireDate     time.Time `json:"hireDate"`
	Salary       float64   `json:"salary"`
	HoursPerWeek int       `json:"hoursPerWeek"`
	PhoneNumber  string    `json:"phoneNumber"`
	Role         string    `json:"role"`
}

// UpdateEmployeeRequest representa os dados para atualizar um funcionário
type UpdateEmployeeRequest struct {
	Name        *string `json:"name"`
	Email       *string `json:"email"`
	Position    *string `json:"position"`
	Department  *string `json:"department"`
	PhoneNumber *string `json:"phone_number"`
}

// WorkshopService defines the interface for the workshop service
type WorkshopService interface {
	CreateWorkshop(ctx context.Context, workshop *domain.Workshop) error
	UpdateWorkshop(ctx context.Context, workshop *domain.Workshop) error
	DeleteWorkshop(ctx context.Context, id string) error
	GetWorkshopByID(ctx context.Context, id string) (*domain.Workshop, error)
	ListWorkshops(ctx context.Context, limit, offset int) ([]*domain.Workshop, int64, error)
}

// NotificationService defines the interface for the notification service
type NotificationService interface {
	SendWorkshopCreatedNotification(ctx context.Context, workshop *domain.Workshop) error
	SendWorkshopUpdatedNotification(ctx context.Context, workshop *domain.Workshop) error
	SendWorkshopDeletedNotification(ctx context.Context, workshop *domain.Workshop) error
}

// AccountingService defines the interface for the accounting service
type AccountingService interface {
	CreateAccountingEntry(ctx context.Context, entry *domain.AccountingEntry) error
	UpdateAccountingEntry(ctx context.Context, entry *domain.AccountingEntry) error
	DeleteAccountingEntry(ctx context.Context, id string) error
	ListAccountingEntries(ctx context.Context, limit, offset int) ([]*domain.AccountingEntry, int64, error)
}

// CarService defines the contract for car business operations
type CarService interface {
	// CreateCar creates a new car with proper authorization checks
	CreateCar(ctx context.Context, car *domain.Car, requestingUserID uuid.UUID) (*domain.Car, error)

	// GetCar retrieves a car by ID with authorization checks
	GetCar(ctx context.Context, carID uuid.UUID, requestingUserID uuid.UUID) (*domain.Car, error)

	// GetCarsByOwner retrieves all cars for a specific owner with authorization
	GetCarsByOwner(ctx context.Context, ownerID uuid.UUID, requestingUserID uuid.UUID) ([]*domain.Car, error)

	// ListCars returns the caller's cars (client) or workshop inventory (staff): optional owner filter and pagination when listing all.
	ListCars(ctx context.Context, requestingUserID uuid.UUID, ownerID *uuid.UUID, limit, offset int) ([]*domain.Car, error)

	// UpdateCar modifies an existing car with authorization checks
	UpdateCar(ctx context.Context, car *domain.Car, requestingUserID uuid.UUID) (*domain.Car, error)

	// DeleteCar removes a car with authorization checks
	DeleteCar(ctx context.Context, carID uuid.UUID, requestingUserID uuid.UUID) error

	// GetCarWithRepairs retrieves a car with its repair history
	GetCarWithRepairs(ctx context.Context, carID uuid.UUID, requestingUserID uuid.UUID) (*domain.Car, error)
}

// AppointmentService defines the contract for appointment business operations
type AppointmentService interface {
	// CreateAppointment schedules a new appointment with authorization checks
	CreateAppointment(ctx context.Context, appointment *domain.Appointment, requestingUserID uuid.UUID) (*domain.Appointment, error)
	// GetAppointment retrieves an appointment by ID with authorization checks
	GetAppointment(ctx context.Context, appointmentID uuid.UUID, requestingUserID uuid.UUID) (*domain.Appointment, error)
	// UpdateAppointment modifies an existing appointment with authorization checks
	UpdateAppointment(ctx context.Context, appointment *domain.Appointment, requestingUserID uuid.UUID) (*domain.Appointment, error)
	// DeleteAppointment removes an appointment with authorization checks
	DeleteAppointment(ctx context.Context, appointmentID uuid.UUID, requestingUserID uuid.UUID) error
	// ListAppointments lists appointments with optional filters and authorization checks
	ListAppointments(ctx context.Context, requestingUserID uuid.UUID, filters *AppointmentFilters) ([]*domain.Appointment, int64, error)
}

	// InvoiceService customer invoices (client: own invoices only for read/update notes).
type InvoiceService interface {
	GetInvoice(ctx context.Context, invoiceID uuid.UUID, requestingUserID uuid.UUID) (*domain.Invoice, error)
	GetInvoiceByRepairID(ctx context.Context, repairID uuid.UUID, requestingUserID uuid.UUID) (*domain.Invoice, error)
	UpdateInvoice(ctx context.Context, invoice *domain.Invoice, requestingUserID uuid.UUID) (*domain.Invoice, error)
	ListMyInvoices(ctx context.Context, requestingUserID uuid.UUID, limit, offset int) ([]*domain.Invoice, int64, error)
	CreateInvoice(ctx context.Context, invoice *domain.Invoice, requestingUserID uuid.UUID) (*domain.Invoice, error)
	ListInvoicesForStaff(ctx context.Context, requestingUserID uuid.UUID, limit, offset int) ([]*domain.Invoice, int64, error)
	DeleteInvoice(ctx context.Context, invoiceID uuid.UUID, requestingUserID uuid.UUID) error
}

// ReceivedInvoiceService manages received (purchase-side) invoices.
type ReceivedInvoiceService interface {
	Create(ctx context.Context, inv *domain.ReceivedInvoice, requestingUserID uuid.UUID) (*domain.ReceivedInvoice, error)
	Get(ctx context.Context, id uuid.UUID, requestingUserID uuid.UUID) (*domain.ReceivedInvoice, error)
	List(ctx context.Context, requestingUserID uuid.UUID, limit, offset int) ([]*domain.ReceivedInvoice, int64, error)
	Update(ctx context.Context, inv *domain.ReceivedInvoice, requestingUserID uuid.UUID) (*domain.ReceivedInvoice, error)
	Delete(ctx context.Context, id uuid.UUID, requestingUserID uuid.UUID) error
}

// BillingDocumentService manages issued billing documents.
type BillingDocumentService interface {
	Create(ctx context.Context, doc *domain.BillingDocument, requestingUserID uuid.UUID) (*domain.BillingDocument, error)
	Get(ctx context.Context, id uuid.UUID, requestingUserID uuid.UUID) (*domain.BillingDocument, error)
	List(ctx context.Context, requestingUserID uuid.UUID, limit, offset int) ([]*domain.BillingDocument, int64, error)
	Update(ctx context.Context, doc *domain.BillingDocument, requestingUserID uuid.UUID) (*domain.BillingDocument, error)
	Delete(ctx context.Context, id uuid.UUID, requestingUserID uuid.UUID) error
}

// SupplierService manages supplier master data.
type SupplierService interface {
	Create(ctx context.Context, s *domain.Supplier, requestingUserID uuid.UUID) (*domain.Supplier, error)
	Get(ctx context.Context, id uuid.UUID, requestingUserID uuid.UUID) (*domain.Supplier, error)
	List(ctx context.Context, requestingUserID uuid.UUID, limit, offset int) ([]*domain.Supplier, int64, error)
	Update(ctx context.Context, s *domain.Supplier, requestingUserID uuid.UUID) (*domain.Supplier, error)
	Delete(ctx context.Context, id uuid.UUID, requestingUserID uuid.UUID) error
}

// PartService manages spare-parts inventory (authorization in HTTP layer).
type PartService interface {
	Create(ctx context.Context, item *domain.PartItem, requestingUserID uuid.UUID) (*domain.PartItem, error)
	Get(ctx context.Context, id uuid.UUID, requestingUserID uuid.UUID) (*domain.PartItem, error)
	List(ctx context.Context, filters PartItemListFilters, requestingUserID uuid.UUID) ([]*domain.PartItem, int64, error)
	Update(ctx context.Context, item *domain.PartItem, requestingUserID uuid.UUID) (*domain.PartItem, error)
	Delete(ctx context.Context, id uuid.UUID, requestingUserID uuid.UUID) error
}
