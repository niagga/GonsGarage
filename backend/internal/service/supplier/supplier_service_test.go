package supplier

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

type supTestUserRepo struct {
	users map[uuid.UUID]*domain.User
}

func (r *supTestUserRepo) Create(ctx context.Context, user *domain.User) error { return nil }
func (r *supTestUserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	return nil, nil
}
func (r *supTestUserRepo) GetByRole(ctx context.Context, role string, limit, offset int) ([]*domain.User, error) {
	return nil, nil
}
func (r *supTestUserRepo) List(ctx context.Context, limit, offset int) ([]*domain.User, error) {
	return nil, nil
}
func (r *supTestUserRepo) Update(ctx context.Context, user *domain.User) error { return nil }
func (r *supTestUserRepo) Delete(ctx context.Context, id uuid.UUID) error { return nil }
func (r *supTestUserRepo) UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string) error {
	return nil
}
func (r *supTestUserRepo) GetActiveUsers(ctx context.Context, limit, offset int) ([]*domain.User, error) {
	return nil, nil
}
func (r *supTestUserRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	u, ok := r.users[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return u, nil
}

type stubSupplierRepo struct {
	byID map[uuid.UUID]*domain.Supplier
}

func (s *stubSupplierRepo) Create(ctx context.Context, row *domain.Supplier) error {
	if s.byID == nil {
		s.byID = make(map[uuid.UUID]*domain.Supplier)
	}
	if row == nil {
		return errors.New("nil")
	}
	cp := *row
	s.byID[row.ID] = &cp
	return nil
}

func (s *stubSupplierRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Supplier, error) {
	if s.byID == nil {
		return nil, domain.ErrSupplierNotFound
	}
	v, ok := s.byID[id]
	if !ok {
		return nil, domain.ErrSupplierNotFound
	}
	cp := *v
	return &cp, nil
}

func (s *stubSupplierRepo) Update(ctx context.Context, row *domain.Supplier) error {
	if s.byID == nil || row == nil {
		return domain.ErrSupplierNotFound
	}
	if _, ok := s.byID[row.ID]; !ok {
		return domain.ErrSupplierNotFound
	}
	cp := *row
	s.byID[row.ID] = &cp
	return nil
}

func (s *stubSupplierRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if s.byID == nil {
		return domain.ErrSupplierNotFound
	}
	if _, ok := s.byID[id]; !ok {
		return domain.ErrSupplierNotFound
	}
	delete(s.byID, id)
	return nil
}

func (s *stubSupplierRepo) List(ctx context.Context, limit, offset int) ([]*domain.Supplier, int64, error) {
	if s.byID == nil {
		return nil, 0, nil
	}
	out := make([]*domain.Supplier, 0, len(s.byID))
	for _, v := range s.byID {
		if v != nil {
			cp := *v
			out = append(out, &cp)
		}
	}
	return out, int64(len(out)), nil
}

func TestSupplierService_Create_ManagerOK(t *testing.T) {
	t.Parallel()
	managerID := uuid.New()
	manager, err := domain.NewUser("m@x.com", "pw", "M", "M", domain.RoleManager)
	require.NoError(t, err)
	manager.ID = managerID
	svc := NewSupplierService(&stubSupplierRepo{}, &supTestUserRepo{users: map[uuid.UUID]*domain.User{managerID: manager}})
	row := &domain.Supplier{Name: "Parts Inc", IsActive: true}
	out, err := svc.Create(context.Background(), row, managerID)
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.NotEqual(t, uuid.Nil, out.ID)
	assert.Equal(t, "Parts Inc", out.Name)
}

func TestSupplierService_Create_ClientDenied(t *testing.T) {
	t.Parallel()
	custID := uuid.New()
	cust, err := domain.NewUser("c@x.com", "pw", "C", "C", domain.RoleClient)
	require.NoError(t, err)
	cust.ID = custID
	svc := NewSupplierService(&stubSupplierRepo{}, &supTestUserRepo{users: map[uuid.UUID]*domain.User{custID: cust}})
	_, err = svc.Create(context.Background(), &domain.Supplier{Name: "X", IsActive: true}, custID)
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrUnauthorizedAccess)
}

func TestSupplierService_Create_EmployeeDenied(t *testing.T) {
	t.Parallel()
	employeeID := uuid.New()
	employee, err := domain.NewUser("e-denied@x.com", "pw", "E", "D", domain.RoleEmployee)
	require.NoError(t, err)
	employee.ID = employeeID
	svc := NewSupplierService(&stubSupplierRepo{}, &supTestUserRepo{users: map[uuid.UUID]*domain.User{employeeID: employee}})
	_, err = svc.Create(context.Background(), &domain.Supplier{Name: "X", IsActive: true}, employeeID)
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrUnauthorizedAccess)
}

func TestSupplierService_Create_InvalidName(t *testing.T) {
	t.Parallel()
	empID := uuid.New()
	emp, err := domain.NewUser("e2@x.com", "pw", "E", "E", domain.RoleManager)
	require.NoError(t, err)
	emp.ID = empID
	svc := NewSupplierService(&stubSupplierRepo{}, &supTestUserRepo{users: map[uuid.UUID]*domain.User{empID: emp}})
	_, err = svc.Create(context.Background(), &domain.Supplier{Name: "   ", IsActive: true}, empID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "name")
}

func TestSupplierService_List_AccessByRole(t *testing.T) {
	t.Parallel()
	managerID, employeeID, custID := uuid.New(), uuid.New(), uuid.New()
	manager, err := domain.NewUser("m3@x.com", "pw", "M", "M", domain.RoleManager)
	require.NoError(t, err)
	manager.ID = managerID
	employee, err := domain.NewUser("e3@x.com", "pw", "E", "E", domain.RoleEmployee)
	require.NoError(t, err)
	employee.ID = employeeID
	cust, err := domain.NewUser("c2@x.com", "pw", "C", "C", domain.RoleClient)
	require.NoError(t, err)
	cust.ID = custID
	id := uuid.New()
	repo := &stubSupplierRepo{byID: map[uuid.UUID]*domain.Supplier{
		id: {ID: id, Name: "A", IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}}
	svc := NewSupplierService(repo, &supTestUserRepo{users: map[uuid.UUID]*domain.User{managerID: manager, employeeID: employee, custID: cust}})
	list, total, err := svc.List(context.Background(), managerID, 10, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, list, 1)

	_, _, err = svc.List(context.Background(), employeeID, 10, 0)
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrUnauthorizedAccess)

	_, _, err = svc.List(context.Background(), custID, 10, 0)
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrUnauthorizedAccess)
}
