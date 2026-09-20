package billing_document

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

type bdTestUserRepo struct {
	users map[uuid.UUID]*domain.User
}

func (r *bdTestUserRepo) Create(ctx context.Context, user *domain.User) error { return nil }
func (r *bdTestUserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	return nil, nil
}
func (r *bdTestUserRepo) GetByRole(ctx context.Context, role string, limit, offset int) ([]*domain.User, error) {
	return nil, nil
}
func (r *bdTestUserRepo) List(ctx context.Context, limit, offset int) ([]*domain.User, error) {
	return nil, nil
}
func (r *bdTestUserRepo) Update(ctx context.Context, user *domain.User) error { return nil }
func (r *bdTestUserRepo) Delete(ctx context.Context, id uuid.UUID) error { return nil }
func (r *bdTestUserRepo) UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string) error {
	return nil
}
func (r *bdTestUserRepo) GetActiveUsers(ctx context.Context, limit, offset int) ([]*domain.User, error) {
	return nil, nil
}
func (r *bdTestUserRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	u, ok := r.users[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return u, nil
}

type stubBillingDocRepo struct {
	byID map[uuid.UUID]*domain.BillingDocument
}

func (s *stubBillingDocRepo) Create(ctx context.Context, doc *domain.BillingDocument) error {
	if s.byID == nil {
		s.byID = make(map[uuid.UUID]*domain.BillingDocument)
	}
	if doc == nil {
		return errors.New("nil")
	}
	cp := *doc
	s.byID[doc.ID] = &cp
	return nil
}

func (s *stubBillingDocRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.BillingDocument, error) {
	if s.byID == nil {
		return nil, domain.ErrBillingDocumentNotFound
	}
	v, ok := s.byID[id]
	if !ok {
		return nil, domain.ErrBillingDocumentNotFound
	}
	cp := *v
	return &cp, nil
}

func (s *stubBillingDocRepo) Update(ctx context.Context, doc *domain.BillingDocument) error {
	if s.byID == nil || doc == nil {
		return domain.ErrBillingDocumentNotFound
	}
	if _, ok := s.byID[doc.ID]; !ok {
		return domain.ErrBillingDocumentNotFound
	}
	cp := *doc
	s.byID[doc.ID] = &cp
	return nil
}

func (s *stubBillingDocRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if s.byID == nil {
		return domain.ErrBillingDocumentNotFound
	}
	if _, ok := s.byID[id]; !ok {
		return domain.ErrBillingDocumentNotFound
	}
	delete(s.byID, id)
	return nil
}

func (s *stubBillingDocRepo) List(ctx context.Context, limit, offset int) ([]*domain.BillingDocument, int64, error) {
	if s.byID == nil {
		return nil, 0, nil
	}
	out := make([]*domain.BillingDocument, 0, len(s.byID))
	for _, v := range s.byID {
		if v != nil {
			cp := *v
			out = append(out, &cp)
		}
	}
	return out, int64(len(out)), nil
}

func TestBillingDocumentService_Create_ManagerOK(t *testing.T) {
	t.Parallel()
	managerID := uuid.New()
	manager, err := domain.NewUser("bm@x.com", "pw", "M", "M", domain.RoleManager)
	require.NoError(t, err)
	manager.ID = managerID
	svc := NewBillingDocumentService(&stubBillingDocRepo{}, &bdTestUserRepo{users: map[uuid.UUID]*domain.User{managerID: manager}})
	doc := &domain.BillingDocument{
		Kind: domain.BillingDocumentKindIRS, Title: "Q1", Amount: 0,
	}
	out, err := svc.Create(context.Background(), doc, managerID)
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, domain.BillingDocumentKindIRS, out.Kind)
}

func TestBillingDocumentService_Create_ClientDenied(t *testing.T) {
	t.Parallel()
	custID := uuid.New()
	cust, err := domain.NewUser("bc@x.com", "pw", "C", "C", domain.RoleClient)
	require.NoError(t, err)
	cust.ID = custID
	svc := NewBillingDocumentService(&stubBillingDocRepo{}, &bdTestUserRepo{users: map[uuid.UUID]*domain.User{custID: cust}})
	_, err = svc.Create(context.Background(), &domain.BillingDocument{
		Kind: domain.BillingDocumentKindOther, Title: "t", Amount: 1,
	}, custID)
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrUnauthorizedAccess)
}

func TestBillingDocumentService_Create_EmployeeDenied(t *testing.T) {
	t.Parallel()
	employeeID := uuid.New()
	employee, err := domain.NewUser("be@x.com", "pw", "E", "E", domain.RoleEmployee)
	require.NoError(t, err)
	employee.ID = employeeID
	svc := NewBillingDocumentService(&stubBillingDocRepo{}, &bdTestUserRepo{users: map[uuid.UUID]*domain.User{employeeID: employee}})
	_, err = svc.Create(context.Background(), &domain.BillingDocument{
		Kind: domain.BillingDocumentKindOther, Title: "t", Amount: 1,
	}, employeeID)
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrUnauthorizedAccess)
}

func TestBillingDocumentService_Create_InvalidKind(t *testing.T) {
	t.Parallel()
	empID := uuid.New()
	emp, err := domain.NewUser("be2@x.com", "pw", "A", "A", domain.RoleAdmin)
	require.NoError(t, err)
	emp.ID = empID
	svc := NewBillingDocumentService(&stubBillingDocRepo{}, &bdTestUserRepo{users: map[uuid.UUID]*domain.User{empID: emp}})
	_, err = svc.Create(context.Background(), &domain.BillingDocument{
		Kind: domain.BillingDocumentKind("nope"), Title: "t", Amount: 0,
	}, empID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "kind")
}

func TestBillingDocumentService_Get_AccessByRole(t *testing.T) {
	t.Parallel()
	managerID, employeeID := uuid.New(), uuid.New()
	manager, err := domain.NewUser("bm3@x.com", "pw", "M", "M", domain.RoleManager)
	require.NoError(t, err)
	manager.ID = managerID
	employee, err := domain.NewUser("be3@x.com", "pw", "E", "E", domain.RoleEmployee)
	require.NoError(t, err)
	employee.ID = employeeID
	id := uuid.New()
	repo := &stubBillingDocRepo{byID: map[uuid.UUID]*domain.BillingDocument{
		id: {ID: id, Kind: domain.BillingDocumentKindPayroll, Title: "P", Amount: 0, CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}}
	svc := NewBillingDocumentService(repo, &bdTestUserRepo{users: map[uuid.UUID]*domain.User{managerID: manager, employeeID: employee}})
	got, err := svc.Get(context.Background(), id, managerID)
	require.NoError(t, err)
	assert.Equal(t, "P", got.Title)

	_, err = svc.Get(context.Background(), id, employeeID)
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrUnauthorizedAccess)
}
