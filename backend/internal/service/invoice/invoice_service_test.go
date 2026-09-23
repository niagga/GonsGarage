package invoice

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gaston-garcia-cegid/gonsgarage/internal/core/ports"
	"github.com/gaston-garcia-cegid/gonsgarage/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type invTestUserRepo struct {
	users map[uuid.UUID]*domain.User
}

func (r *invTestUserRepo) Create(ctx context.Context, user *domain.User) error { return nil }
func (r *invTestUserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	return nil, nil
}
func (r *invTestUserRepo) GetByRole(ctx context.Context, role string, limit, offset int) ([]*domain.User, error) {
	return nil, nil
}
func (r *invTestUserRepo) List(ctx context.Context, limit, offset int) ([]*domain.User, error) {
	return nil, nil
}
func (r *invTestUserRepo) Update(ctx context.Context, user *domain.User) error { return nil }
func (r *invTestUserRepo) Delete(ctx context.Context, id uuid.UUID) error      { return nil }
func (r *invTestUserRepo) UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string) error {
	return nil
}
func (r *invTestUserRepo) GetActiveUsers(ctx context.Context, limit, offset int) ([]*domain.User, error) {
	return nil, nil
}
func (r *invTestUserRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	u, ok := r.users[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return u, nil
}

type stubInvoiceRepo struct {
	byID      map[uuid.UUID]*domain.Invoice
	byCust    map[uuid.UUID][]*domain.Invoice
	listTotal int64
	updateErr error
}

func (s *stubInvoiceRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Invoice, error) {
	if s.byID == nil {
		return nil, nil
	}
	return s.byID[id], nil
}

func (s *stubInvoiceRepo) Update(ctx context.Context, invoice *domain.Invoice) error {
	if s.updateErr != nil {
		return s.updateErr
	}
	if s.byID != nil && invoice != nil {
		cp := *invoice
		s.byID[invoice.ID] = &cp
	}
	return nil
}

func (s *stubInvoiceRepo) ListByCustomerID(ctx context.Context, customerID uuid.UUID, limit, offset int) ([]*domain.Invoice, int64, error) {
	if s.byCust == nil {
		return nil, 0, nil
	}
	return s.byCust[customerID], s.listTotal, nil
}

func (s *stubInvoiceRepo) Create(ctx context.Context, invoice *domain.Invoice) error {
	if s.byID == nil {
		s.byID = make(map[uuid.UUID]*domain.Invoice)
	}
	if invoice == nil {
		return errors.New("nil invoice")
	}
	cp := *invoice
	s.byID[invoice.ID] = &cp
	return nil
}

func (s *stubInvoiceRepo) ListForStaff(ctx context.Context, limit, offset int) ([]*domain.Invoice, int64, error) {
	if s.byID == nil {
		return nil, 0, nil
	}
	out := make([]*domain.Invoice, 0, len(s.byID))
	for _, v := range s.byID {
		if v != nil {
			cp := *v
			out = append(out, &cp)
		}
	}
	return out, int64(len(out)), nil
}

func (s *stubInvoiceRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if s.byID != nil {
		delete(s.byID, id)
	}
	return nil
}

func TestInvoiceService_GetInvoice_ClientOwn(t *testing.T) {
	t.Parallel()
	cust := uuid.New()
	invID := uuid.New()
	u, err := domain.NewUser("c@x.com", "pw", "C", "L", domain.RoleClient)
	require.NoError(t, err)
	u.ID = cust

	inv := &domain.Invoice{
		ID:         invID,
		CustomerID: cust,
		Amount:     100,
		Status:     "open",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	svc := NewInvoiceService(
		&stubInvoiceRepo{byID: map[uuid.UUID]*domain.Invoice{invID: inv}},
		&invTestUserRepo{users: map[uuid.UUID]*domain.User{cust: u}},
	)
	out, err := svc.GetInvoice(context.Background(), invID, cust)
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, invID, out.ID)
}

func TestInvoiceService_GetInvoice_ClientOtherDenied(t *testing.T) {
	t.Parallel()
	cust := uuid.New()
	other := uuid.New()
	invID := uuid.New()
	u, err := domain.NewUser("c@x.com", "pw", "C", "L", domain.RoleClient)
	require.NoError(t, err)
	u.ID = cust

	inv := &domain.Invoice{ID: invID, CustomerID: other, Amount: 50, Status: "open", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	svc := NewInvoiceService(
		&stubInvoiceRepo{byID: map[uuid.UUID]*domain.Invoice{invID: inv}},
		&invTestUserRepo{users: map[uuid.UUID]*domain.User{cust: u}},
	)
	out, err := svc.GetInvoice(context.Background(), invID, cust)
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrUnauthorizedAccess)
	assert.Nil(t, out)
}

func TestInvoiceService_GetInvoice_EmployeeDenied(t *testing.T) {
	t.Parallel()
	employeeID := uuid.New()
	invID := uuid.New()
	employee, err := domain.NewUser("e-get@x.com", "pw", "E", "E", domain.RoleEmployee)
	require.NoError(t, err)
	employee.ID = employeeID
	inv := &domain.Invoice{ID: invID, CustomerID: uuid.New(), Amount: 50, Status: "open", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	svc := NewInvoiceService(
		&stubInvoiceRepo{byID: map[uuid.UUID]*domain.Invoice{invID: inv}},
		&invTestUserRepo{users: map[uuid.UUID]*domain.User{employeeID: employee}},
	)
	out, err := svc.GetInvoice(context.Background(), invID, employeeID)
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrUnauthorizedAccess)
	assert.Nil(t, out)
}

func TestInvoiceService_UpdateInvoice_ClientNotesOnly(t *testing.T) {
	t.Parallel()
	cust := uuid.New()
	invID := uuid.New()
	u, err := domain.NewUser("c@x.com", "pw", "C", "L", domain.RoleClient)
	require.NoError(t, err)
	u.ID = cust

	existing := &domain.Invoice{
		ID: invID, CustomerID: cust, Amount: 200, Status: "open", Notes: "old",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	repo := &stubInvoiceRepo{byID: map[uuid.UUID]*domain.Invoice{invID: existing}}
	svc := NewInvoiceService(repo, &invTestUserRepo{users: map[uuid.UUID]*domain.User{cust: u}})

	out, err := svc.UpdateInvoice(context.Background(), &domain.Invoice{ID: invID, Notes: "new note", Status: "paid"}, cust)
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, "new note", out.Notes)
	assert.Equal(t, "open", out.Status, "client cannot change status via this path")
	assert.Equal(t, 200.0, out.Amount)
}

func TestInvoiceService_UpdateInvoice_ClientOtherDenied(t *testing.T) {
	t.Parallel()
	cust := uuid.New()
	other := uuid.New()
	invID := uuid.New()
	u, err := domain.NewUser("c@x.com", "pw", "C", "L", domain.RoleClient)
	require.NoError(t, err)
	u.ID = cust

	existing := &domain.Invoice{ID: invID, CustomerID: other, Amount: 1, Status: "open", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	svc := NewInvoiceService(
		&stubInvoiceRepo{byID: map[uuid.UUID]*domain.Invoice{invID: existing}},
		&invTestUserRepo{users: map[uuid.UUID]*domain.User{cust: u}},
	)
	out, err := svc.UpdateInvoice(context.Background(), &domain.Invoice{ID: invID, Notes: "x"}, cust)
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrUnauthorizedAccess)
	assert.Nil(t, out)
}

func TestInvoiceService_UpdateInvoice_EmployeeDenied(t *testing.T) {
	t.Parallel()
	employeeID := uuid.New()
	invID := uuid.New()
	employee, err := domain.NewUser("e-upd@x.com", "pw", "E", "E", domain.RoleEmployee)
	require.NoError(t, err)
	employee.ID = employeeID

	existing := &domain.Invoice{ID: invID, CustomerID: uuid.New(), Amount: 1, Status: "open", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	svc := NewInvoiceService(
		&stubInvoiceRepo{byID: map[uuid.UUID]*domain.Invoice{invID: existing}},
		&invTestUserRepo{users: map[uuid.UUID]*domain.User{employeeID: employee}},
	)
	out, err := svc.UpdateInvoice(context.Background(), &domain.Invoice{ID: invID, Notes: "x"}, employeeID)
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrUnauthorizedAccess)
	assert.Nil(t, out)
}

func TestInvoiceService_ListMyInvoices_ClientOnly(t *testing.T) {
	t.Parallel()
	cust := uuid.New()
	u, err := domain.NewUser("c@x.com", "pw", "C", "L", domain.RoleClient)
	require.NoError(t, err)
	u.ID = cust

	emp, err := domain.NewUser("e@x.com", "pw", "E", "E", domain.RoleEmployee)
	require.NoError(t, err)
	empID := uuid.New()
	emp.ID = empID

	repo := &stubInvoiceRepo{byCust: map[uuid.UUID][]*domain.Invoice{
		cust: {{ID: uuid.New(), CustomerID: cust, Amount: 1, Status: "open", CreatedAt: time.Now(), UpdatedAt: time.Now()}},
	}, listTotal: 1}
	svc := NewInvoiceService(repo, &invTestUserRepo{users: map[uuid.UUID]*domain.User{cust: u, empID: emp}})
	list, total, err := svc.ListMyInvoices(context.Background(), cust, 10, 0)
	require.NoError(t, err)
	assert.Len(t, list, 1)
	assert.Equal(t, int64(1), total)

	_, _, err = svc.ListMyInvoices(context.Background(), empID, 10, 0)
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrUnauthorizedAccess)
}

func TestInvoiceService_CreateInvoice_ManagerForClient(t *testing.T) {
	t.Parallel()
	custID := uuid.New()
	managerID := uuid.New()
	cust, err := domain.NewUser("c@x.com", "pw", "C", "L", domain.RoleClient)
	require.NoError(t, err)
	cust.ID = custID
	manager, err := domain.NewUser("m@x.com", "pw", "M", "M", domain.RoleManager)
	require.NoError(t, err)
	manager.ID = managerID

	repo := &stubInvoiceRepo{byID: map[uuid.UUID]*domain.Invoice{}}
	svc := NewInvoiceService(repo, &invTestUserRepo{users: map[uuid.UUID]*domain.User{custID: cust, managerID: manager}})

	in := &domain.Invoice{CustomerID: custID, Amount: 42, Notes: "svc"}
	out, err := svc.CreateInvoice(context.Background(), in, managerID)
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.NotEqual(t, uuid.Nil, out.ID)
	assert.Equal(t, custID, out.CustomerID)
	assert.Equal(t, 42.0, out.Amount)
	assert.Equal(t, "open", out.Status)
	assert.Equal(t, "svc", out.Notes)
}

func TestInvoiceService_CreateInvoice_ClientDenied(t *testing.T) {
	t.Parallel()
	custID := uuid.New()
	cust, err := domain.NewUser("c@x.com", "pw", "C", "L", domain.RoleClient)
	require.NoError(t, err)
	cust.ID = custID

	svc := NewInvoiceService(&stubInvoiceRepo{}, &invTestUserRepo{users: map[uuid.UUID]*domain.User{custID: cust}})
	out, err := svc.CreateInvoice(context.Background(), &domain.Invoice{CustomerID: custID, Amount: 1}, custID)
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrUnauthorizedAccess)
	assert.Nil(t, out)
}

func TestInvoiceService_CreateInvoice_EmployeeDenied(t *testing.T) {
	t.Parallel()
	employeeID := uuid.New()
	employee, err := domain.NewUser("e-create@x.com", "pw", "E", "E", domain.RoleEmployee)
	require.NoError(t, err)
	employee.ID = employeeID

	svc := NewInvoiceService(&stubInvoiceRepo{}, &invTestUserRepo{users: map[uuid.UUID]*domain.User{employeeID: employee}})
	out, err := svc.CreateInvoice(context.Background(), &domain.Invoice{CustomerID: uuid.New(), Amount: 1}, employeeID)
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrUnauthorizedAccess)
	assert.Nil(t, out)
}

func TestInvoiceService_CreateInvoice_CustomerMustBeClient(t *testing.T) {
	t.Parallel()
	custID := uuid.New()
	empID := uuid.New()
	empAsCust, err := domain.NewUser("e2@x.com", "pw", "E", "E", domain.RoleEmployee)
	require.NoError(t, err)
	empAsCust.ID = custID
	actor, err := domain.NewUser("mgr@x.com", "pw", "M", "M", domain.RoleManager)
	require.NoError(t, err)
	actor.ID = empID

	svc := NewInvoiceService(&stubInvoiceRepo{}, &invTestUserRepo{users: map[uuid.UUID]*domain.User{custID: empAsCust, empID: actor}})
	out, err := svc.CreateInvoice(context.Background(), &domain.Invoice{CustomerID: custID, Amount: 10}, empID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "client")
	assert.Nil(t, out)
}

func TestInvoiceService_ListInvoicesForStaff_ManagerOk(t *testing.T) {
	t.Parallel()
	managerID := uuid.New()
	manager, err := domain.NewUser("m-list@x.com", "pw", "M", "M", domain.RoleManager)
	require.NoError(t, err)
	manager.ID = managerID
	inv := &domain.Invoice{ID: uuid.New(), CustomerID: uuid.New(), Amount: 9, Status: "open", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	repo := &stubInvoiceRepo{byID: map[uuid.UUID]*domain.Invoice{inv.ID: inv}}

	svc := NewInvoiceService(repo, &invTestUserRepo{users: map[uuid.UUID]*domain.User{managerID: manager}})
	list, total, err := svc.ListInvoicesForStaff(context.Background(), managerID, 10, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, list, 1)
	assert.Equal(t, inv.ID, list[0].ID)
}

func TestInvoiceService_ListInvoicesForStaff_ClientDenied(t *testing.T) {
	t.Parallel()
	custID := uuid.New()
	cust, err := domain.NewUser("c@x.com", "pw", "C", "L", domain.RoleClient)
	require.NoError(t, err)
	cust.ID = custID

	svc := NewInvoiceService(&stubInvoiceRepo{}, &invTestUserRepo{users: map[uuid.UUID]*domain.User{custID: cust}})
	_, _, err = svc.ListInvoicesForStaff(context.Background(), custID, 10, 0)
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrUnauthorizedAccess)
}

func TestInvoiceService_ListInvoicesForStaff_EmployeeDenied(t *testing.T) {
	t.Parallel()
	employeeID := uuid.New()
	employee, err := domain.NewUser("e-list@x.com", "pw", "E", "E", domain.RoleEmployee)
	require.NoError(t, err)
	employee.ID = employeeID

	svc := NewInvoiceService(&stubInvoiceRepo{}, &invTestUserRepo{users: map[uuid.UUID]*domain.User{employeeID: employee}})
	_, _, err = svc.ListInvoicesForStaff(context.Background(), employeeID, 10, 0)
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrUnauthorizedAccess)
}

func TestInvoiceService_DeleteInvoice_ManagerOk(t *testing.T) {
	t.Parallel()
	managerID := uuid.New()
	invID := uuid.New()
	manager, err := domain.NewUser("m-del@x.com", "pw", "M", "M", domain.RoleManager)
	require.NoError(t, err)
	manager.ID = managerID
	inv := &domain.Invoice{ID: invID, CustomerID: uuid.New(), Amount: 1, Status: "open", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	repo := &stubInvoiceRepo{byID: map[uuid.UUID]*domain.Invoice{invID: inv}}

	svc := NewInvoiceService(repo, &invTestUserRepo{users: map[uuid.UUID]*domain.User{managerID: manager}})
	err = svc.DeleteInvoice(context.Background(), invID, managerID)
	require.NoError(t, err)
	got, gerr := repo.GetByID(context.Background(), invID)
	require.NoError(t, gerr)
	assert.Nil(t, got)
}

func TestInvoiceService_DeleteInvoice_ClientDenied(t *testing.T) {
	t.Parallel()
	custID := uuid.New()
	invID := uuid.New()
	cust, err := domain.NewUser("c@x.com", "pw", "C", "L", domain.RoleClient)
	require.NoError(t, err)
	cust.ID = custID
	inv := &domain.Invoice{ID: invID, CustomerID: custID, Amount: 1, Status: "open", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	repo := &stubInvoiceRepo{byID: map[uuid.UUID]*domain.Invoice{invID: inv}}

	svc := NewInvoiceService(repo, &invTestUserRepo{users: map[uuid.UUID]*domain.User{custID: cust}})
	err = svc.DeleteInvoice(context.Background(), invID, custID)
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrUnauthorizedAccess)
}

func TestInvoiceService_DeleteInvoice_EmployeeDenied(t *testing.T) {
	t.Parallel()
	employeeID := uuid.New()
	invID := uuid.New()
	employee, err := domain.NewUser("e-del@x.com", "pw", "E", "E", domain.RoleEmployee)
	require.NoError(t, err)
	employee.ID = employeeID
	inv := &domain.Invoice{ID: invID, CustomerID: uuid.New(), Amount: 1, Status: "open", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	repo := &stubInvoiceRepo{byID: map[uuid.UUID]*domain.Invoice{invID: inv}}

	svc := NewInvoiceService(repo, &invTestUserRepo{users: map[uuid.UUID]*domain.User{employeeID: employee}})
	err = svc.DeleteInvoice(context.Background(), invID, employeeID)
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrUnauthorizedAccess)
}

type protectFiscalReader struct {
	protected bool
}

func (p protectFiscalReader) HasProtectedFiscalHistory(context.Context, uuid.UUID) (bool, error) {
	return p.protected, nil
}

func TestInvoiceService_DeleteInvoice_FrozenHistoryConflict(t *testing.T) {
	t.Parallel()
	managerID := uuid.New()
	invID := uuid.New()
	manager, err := domain.NewUser("m-protect@x.com", "pw", "M", "M", domain.RoleManager)
	require.NoError(t, err)
	manager.ID = managerID
	inv := &domain.Invoice{ID: invID, CustomerID: uuid.New(), Amount: 10, Status: "open", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	repo := &stubInvoiceRepo{byID: map[uuid.UUID]*domain.Invoice{invID: inv}}
	svc := NewInvoiceService(repo, &invTestUserRepo{users: map[uuid.UUID]*domain.User{managerID: manager}}).
		WithFiscalProtection(protectFiscalReader{protected: true})

	err = svc.DeleteInvoice(context.Background(), invID, managerID)
	require.Error(t, err)
	assert.ErrorIs(t, err, ports.ErrFiscalHistoryProtected)
	assert.Contains(t, repo.byID, invID)
}

func TestInvoiceService_DeleteInvoice_DraftHistoryAllowed(t *testing.T) {
	t.Parallel()
	managerID := uuid.New()
	invID := uuid.New()
	manager, err := domain.NewUser("m-draft@x.com", "pw", "M", "M", domain.RoleManager)
	require.NoError(t, err)
	manager.ID = managerID
	inv := &domain.Invoice{ID: invID, CustomerID: uuid.New(), Amount: 10, Status: "open", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	repo := &stubInvoiceRepo{byID: map[uuid.UUID]*domain.Invoice{invID: inv}}
	svc := NewInvoiceService(repo, &invTestUserRepo{users: map[uuid.UUID]*domain.User{managerID: manager}}).
		WithFiscalProtection(protectFiscalReader{protected: false})

	err = svc.DeleteInvoice(context.Background(), invID, managerID)
	require.NoError(t, err)
	assert.NotContains(t, repo.byID, invID)
}
