package received_invoice

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gaston-garcia-cegid/gonsgarage/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type riTestUserRepo struct {
	users map[uuid.UUID]*domain.User
}

func (r *riTestUserRepo) Create(ctx context.Context, user *domain.User) error { return nil }
func (r *riTestUserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	return nil, nil
}
func (r *riTestUserRepo) GetByRole(ctx context.Context, role string, limit, offset int) ([]*domain.User, error) {
	return nil, nil
}
func (r *riTestUserRepo) List(ctx context.Context, limit, offset int) ([]*domain.User, error) {
	return nil, nil
}
func (r *riTestUserRepo) Update(ctx context.Context, user *domain.User) error { return nil }
func (r *riTestUserRepo) Delete(ctx context.Context, id uuid.UUID) error { return nil }
func (r *riTestUserRepo) UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string) error {
	return nil
}
func (r *riTestUserRepo) GetActiveUsers(ctx context.Context, limit, offset int) ([]*domain.User, error) {
	return nil, nil
}
func (r *riTestUserRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	u, ok := r.users[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return u, nil
}

type stubReceivedInvoiceRepo struct {
	byID map[uuid.UUID]*domain.ReceivedInvoice
}

func (s *stubReceivedInvoiceRepo) Create(ctx context.Context, inv *domain.ReceivedInvoice) error {
	if s.byID == nil {
		s.byID = make(map[uuid.UUID]*domain.ReceivedInvoice)
	}
	if inv == nil {
		return errors.New("nil")
	}
	cp := *inv
	s.byID[inv.ID] = &cp
	return nil
}

func (s *stubReceivedInvoiceRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.ReceivedInvoice, error) {
	if s.byID == nil {
		return nil, domain.ErrReceivedInvoiceNotFound
	}
	v, ok := s.byID[id]
	if !ok {
		return nil, domain.ErrReceivedInvoiceNotFound
	}
	cp := *v
	return &cp, nil
}

func (s *stubReceivedInvoiceRepo) Update(ctx context.Context, inv *domain.ReceivedInvoice) error {
	if s.byID == nil || inv == nil {
		return domain.ErrReceivedInvoiceNotFound
	}
	if _, ok := s.byID[inv.ID]; !ok {
		return domain.ErrReceivedInvoiceNotFound
	}
	cp := *inv
	s.byID[inv.ID] = &cp
	return nil
}

func (s *stubReceivedInvoiceRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if s.byID == nil {
		return domain.ErrReceivedInvoiceNotFound
	}
	if _, ok := s.byID[id]; !ok {
		return domain.ErrReceivedInvoiceNotFound
	}
	delete(s.byID, id)
	return nil
}

func (s *stubReceivedInvoiceRepo) List(ctx context.Context, limit, offset int) ([]*domain.ReceivedInvoice, int64, error) {
	if s.byID == nil {
		return nil, 0, nil
	}
	out := make([]*domain.ReceivedInvoice, 0, len(s.byID))
	for _, v := range s.byID {
		if v != nil {
			cp := *v
			out = append(out, &cp)
		}
	}
	return out, int64(len(out)), nil
}

func sampleRI() *domain.ReceivedInvoice {
	return &domain.ReceivedInvoice{
		VendorName:  "V",
		Category:    "parts",
		Amount:      10,
		InvoiceDate: time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC),
	}
}

func TestReceivedInvoiceService_Create_ManagerOK(t *testing.T) {
	t.Parallel()
	managerID := uuid.New()
	manager, err := domain.NewUser("rim@x.com", "pw", "M", "M", domain.RoleManager)
	require.NoError(t, err)
	manager.ID = managerID
	svc := NewReceivedInvoiceService(&stubReceivedInvoiceRepo{}, &riTestUserRepo{users: map[uuid.UUID]*domain.User{managerID: manager}})
	out, err := svc.Create(context.Background(), sampleRI(), managerID)
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, 10.0, out.Amount)
}

func TestReceivedInvoiceService_Create_ClientDenied(t *testing.T) {
	t.Parallel()
	custID := uuid.New()
	cust, err := domain.NewUser("ric@x.com", "pw", "C", "C", domain.RoleClient)
	require.NoError(t, err)
	cust.ID = custID
	svc := NewReceivedInvoiceService(&stubReceivedInvoiceRepo{}, &riTestUserRepo{users: map[uuid.UUID]*domain.User{custID: cust}})
	_, err = svc.Create(context.Background(), sampleRI(), custID)
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrUnauthorizedAccess)
}

func TestReceivedInvoiceService_Create_EmployeeDenied(t *testing.T) {
	t.Parallel()
	employeeID := uuid.New()
	employee, err := domain.NewUser("rie@x.com", "pw", "E", "E", domain.RoleEmployee)
	require.NoError(t, err)
	employee.ID = employeeID
	svc := NewReceivedInvoiceService(&stubReceivedInvoiceRepo{}, &riTestUserRepo{users: map[uuid.UUID]*domain.User{employeeID: employee}})
	_, err = svc.Create(context.Background(), sampleRI(), employeeID)
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrUnauthorizedAccess)
}

func TestReceivedInvoiceService_List_AccessByRole(t *testing.T) {
	t.Parallel()
	managerID, employeeID, clientID := uuid.New(), uuid.New(), uuid.New()
	manager, err := domain.NewUser("rim2@x.com", "pw", "M", "M", domain.RoleManager)
	require.NoError(t, err)
	manager.ID = managerID
	employee, err := domain.NewUser("rie2-list@x.com", "pw", "E", "E", domain.RoleEmployee)
	require.NoError(t, err)
	employee.ID = employeeID
	client, err := domain.NewUser("ric2@x.com", "pw", "C", "C", domain.RoleClient)
	require.NoError(t, err)
	client.ID = clientID

	repo := &stubReceivedInvoiceRepo{byID: map[uuid.UUID]*domain.ReceivedInvoice{
		uuid.New(): sampleRI(),
	}}
	svc := NewReceivedInvoiceService(repo, &riTestUserRepo{users: map[uuid.UUID]*domain.User{managerID: manager, employeeID: employee, clientID: client}})

	list, total, err := svc.List(context.Background(), managerID, 10, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, list, 1)

	_, _, err = svc.List(context.Background(), employeeID, 10, 0)
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrUnauthorizedAccess)

	_, _, err = svc.List(context.Background(), clientID, 10, 0)
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrUnauthorizedAccess)
}

func TestReceivedInvoiceService_Create_InvalidAmount(t *testing.T) {
	t.Parallel()
	empID := uuid.New()
	emp, err := domain.NewUser("rie2@x.com", "pw", "M", "M", domain.RoleManager)
	require.NoError(t, err)
	emp.ID = empID
	svc := NewReceivedInvoiceService(&stubReceivedInvoiceRepo{}, &riTestUserRepo{users: map[uuid.UUID]*domain.User{empID: emp}})
	bad := sampleRI()
	bad.Amount = 0
	_, err = svc.Create(context.Background(), bad, empID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "amount")
}
