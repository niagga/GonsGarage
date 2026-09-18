package auth

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

// stubUserRepo implements ports.UserRepository for auth unit tests (TDD-friendly minimal surface).
type stubUserRepo struct {
	byEmail   map[string]*domain.User
	createErr error
	created   []*domain.User
}

func newStubUserRepo() *stubUserRepo {
	return &stubUserRepo{byEmail: make(map[string]*domain.User)}
}

func (s *stubUserRepo) Create(ctx context.Context, user *domain.User) error {
	if s.createErr != nil {
		return s.createErr
	}
	s.byEmail[user.Email] = user
	s.created = append(s.created, user)
	return nil
}

func (s *stubUserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	u, ok := s.byEmail[email]
	if !ok {
		return nil, nil
	}
	return u, nil
}

func (s *stubUserRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	for _, u := range s.byEmail {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, errors.New("not found")
}

func (s *stubUserRepo) GetByRole(ctx context.Context, role string, limit, offset int) ([]*domain.User, error) {
	return nil, nil
}
func (s *stubUserRepo) List(ctx context.Context, limit, offset int) ([]*domain.User, error) {
	return nil, nil
}
func (s *stubUserRepo) Update(ctx context.Context, user *domain.User) error { return nil }
func (s *stubUserRepo) Delete(ctx context.Context, id uuid.UUID) error      { return nil }
func (s *stubUserRepo) UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string) error {
	return nil
}
func (s *stubUserRepo) GetActiveUsers(ctx context.Context, limit, offset int) ([]*domain.User, error) {
	return nil, nil
}

func TestAuthService_Register_NewClient(t *testing.T) {
	t.Parallel()
	repo := newStubUserRepo()
	svc := NewAuthService(repo, "unit-test-secret", 24)

	user, err := svc.Register(context.Background(), ports.RegisterRequest{
		Email:     "client@example.com",
		Password:  "secret123",
		FirstName: "Ada",
		LastName:  "Lovelace",
		Role:      domain.RoleClient,
	})
	require.NoError(t, err)
	require.NotNil(t, user)
	assert.Equal(t, "client@example.com", user.Email)
	assert.Empty(t, user.Password)
	assert.Len(t, repo.created, 1)
}

func TestAuthService_Register_DuplicateEmail(t *testing.T) {
	t.Parallel()
	repo := newStubUserRepo()
	existing, err := domain.NewUser("dup@example.com", "x", "A", "B", domain.RoleClient)
	require.NoError(t, err)
	repo.byEmail[existing.Email] = existing

	svc := NewAuthService(repo, "unit-test-secret", 24)
	_, err = svc.Register(context.Background(), ports.RegisterRequest{
		Email:     "dup@example.com",
		Password:  "other",
		FirstName: "C",
		LastName:  "D",
		Role:      domain.RoleClient,
	})
	assert.ErrorIs(t, err, domain.ErrUserAlreadyExists)
}

func TestAuthService_Register_InvalidRole(t *testing.T) {
	t.Parallel()
	repo := newStubUserRepo()
	svc := NewAuthService(repo, "unit-test-secret", 24)
	_, err := svc.Register(context.Background(), ports.RegisterRequest{
		Email:     "r@example.com",
		Password:  "secret123",
		FirstName: "A",
		LastName:  "B",
		Role:      "superuser",
	})
	require.ErrorIs(t, err, domain.ErrInvalidRole)
	assert.NotContains(t, err.Error(), domain.RoleAdmin)
	assert.NotContains(t, err.Error(), domain.RoleManager)
	assert.Empty(t, repo.created)
}

func TestAuthService_Register_RejectsAdmin(t *testing.T) {
	t.Parallel()
	repo := newStubUserRepo()
	svc := NewAuthService(repo, "unit-test-secret", 24)
	user, err := svc.Register(context.Background(), ports.RegisterRequest{
		Email:     "admin@example.com",
		Password:  "secret123",
		FirstName: "A",
		LastName:  "B",
		Role:      domain.RoleAdmin,
	})
	require.ErrorIs(t, err, domain.ErrInvalidRole)
	assert.Nil(t, user)
	assert.Empty(t, repo.created)
}

func TestAuthService_Register_RejectsManager(t *testing.T) {
	t.Parallel()
	repo := newStubUserRepo()
	svc := NewAuthService(repo, "unit-test-secret", 24)
	user, err := svc.Register(context.Background(), ports.RegisterRequest{
		Email:     "manager@example.com",
		Password:  "secret123",
		FirstName: "A",
		LastName:  "B",
		Role:      domain.RoleManager,
	})
	require.ErrorIs(t, err, domain.ErrInvalidRole)
	assert.Nil(t, user)
	assert.Empty(t, repo.created)
}

func TestAuthService_Register_NewEmployee(t *testing.T) {
	t.Parallel()
	repo := newStubUserRepo()
	svc := NewAuthService(repo, "unit-test-secret", 24)

	user, err := svc.Register(context.Background(), ports.RegisterRequest{
		Email:     "employee@example.com",
		Password:  "secret123",
		FirstName: "Col",
		LastName:  "Aborador",
		Role:      domain.RoleEmployee,
	})
	require.NoError(t, err)
	require.NotNil(t, user)
	assert.Equal(t, "employee@example.com", user.Email)
	assert.Equal(t, domain.RoleEmployee, user.Role)
	assert.Empty(t, user.Password)
	assert.Len(t, repo.created, 1)
}

func TestAuthService_Login_Success(t *testing.T) {
	t.Setenv("JWT_SECRET", "jwt-test-secret")
	repo := newStubUserRepo()
	u, err := domain.NewUser("login@example.com", "correct-password", "L", "N", domain.RoleClient)
	require.NoError(t, err)
	repo.byEmail[u.Email] = u

	svc := NewAuthService(repo, "ignored-for-generate", 1)
	token, err := svc.Login(context.Background(), "login@example.com", "correct-password")
	require.NoError(t, err)
	assert.NotEmpty(t, token)
}

func TestAuthService_Login_WrongPassword(t *testing.T) {
	repo := newStubUserRepo()
	u, err := domain.NewUser("login2@example.com", "good", "L", "N", domain.RoleClient)
	require.NoError(t, err)
	repo.byEmail[u.Email] = u

	svc := NewAuthService(repo, "unit-test-secret", 1)
	_, err = svc.Login(context.Background(), "login2@example.com", "bad")
	require.Error(t, err)
	assert.Equal(t, "invalid credentials", err.Error())
}

func TestAuthService_Login_InactiveUser(t *testing.T) {
	repo := newStubUserRepo()
	u, err := domain.NewUser("inactive@example.com", "pw", "I", "N", domain.RoleClient)
	require.NoError(t, err)
	u.IsActive = false
	repo.byEmail[u.Email] = u

	svc := NewAuthService(repo, "unit-test-secret", 1)
	_, err = svc.Login(context.Background(), "inactive@example.com", "pw")
	require.Error(t, err)
	assert.Equal(t, "user account is deactivated", err.Error())
}

func TestAuthService_CurrentUser(t *testing.T) {
	t.Parallel()
	repo := newStubUserRepo()
	u, err := domain.NewUser("me@example.com", "pw", "M", "E", domain.RoleClient)
	require.NoError(t, err)
	repo.byEmail[u.Email] = u

	svc := NewAuthService(repo, "secret", 1)
	out, err := svc.CurrentUser(context.Background(), u.ID)
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, u.Email, out.Email)
	assert.Empty(t, out.Password)
}

func TestAuthService_CurrentUser_NotFound(t *testing.T) {
	t.Parallel()
	repo := newStubUserRepo()
	svc := NewAuthService(repo, "secret", 1)
	_, err := svc.CurrentUser(context.Background(), uuid.New())
	assert.ErrorIs(t, err, domain.ErrUserNotFound)
}

func TestAuthService_Register_DefaultRoleClient(t *testing.T) {
	t.Parallel()
	repo := newStubUserRepo()
	svc := NewAuthService(repo, "unit-test-secret", 24)
	user, err := svc.Register(context.Background(), ports.RegisterRequest{
		Email:     "noreqrole@example.com",
		Password:  "secret123",
		FirstName: "A",
		LastName:  "B",
		Role:      "",
	})
	require.NoError(t, err)
	require.NotNil(t, user)
	assert.Equal(t, domain.RoleClient, user.Role)
}

func TestAuthService_GenerateToken_UsesExpireTimeFromConstructor(t *testing.T) {
	t.Setenv("JWT_SECRET", "jwt-consistency-secret")
	repo := newStubUserRepo()
	u, err := domain.NewUser("tok@example.com", "pw", "T", "K", domain.RoleClient)
	require.NoError(t, err)

	svc := NewAuthService(repo, "constructor-secret", 2).(*AuthService)
	token, exp, err := svc.GenerateToken(u)
	require.NoError(t, err)
	assert.NotEmpty(t, token)
	assert.WithinDuration(t, time.Now().Add(2*time.Hour), exp, 5*time.Second)
}

func TestAuthService_ProvisionUser_AdminCreatesManager(t *testing.T) {
	t.Parallel()
	repo := newStubUserRepo()
	svc := NewAuthService(repo, "secret", 24)
	caller := uuid.New()

	user, err := svc.ProvisionUser(context.Background(), caller, domain.RoleAdmin, ports.ProvisionUserRequest{
		Email: "mgr@example.com", Password: "secret12", FirstName: "M", LastName: "R", Role: domain.RoleManager,
	})
	require.NoError(t, err)
	require.NotNil(t, user)
	assert.Equal(t, domain.RoleManager, user.Role)
	assert.Empty(t, user.Password)
}

func TestAuthService_ProvisionUser_ManagerCannotCreateManager(t *testing.T) {
	t.Parallel()
	repo := newStubUserRepo()
	svc := NewAuthService(repo, "secret", 24)
	caller := uuid.New()

	_, err := svc.ProvisionUser(context.Background(), caller, domain.RoleManager, ports.ProvisionUserRequest{
		Email: "x@example.com", Password: "secret12", FirstName: "A", LastName: "B", Role: domain.RoleManager,
	})
	assert.ErrorIs(t, err, domain.ErrPermissionDenied)
}

func TestAuthService_ProvisionUser_TargetAdminRejected(t *testing.T) {
	t.Parallel()
	repo := newStubUserRepo()
	svc := NewAuthService(repo, "secret", 24)

	_, err := svc.ProvisionUser(context.Background(), uuid.New(), domain.RoleAdmin, ports.ProvisionUserRequest{
		Email: "root@example.com", Password: "secret12", FirstName: "A", LastName: "B", Role: domain.RoleAdmin,
	})
	assert.ErrorIs(t, err, domain.ErrInvalidRole)
}

func TestAuthService_ProvisionUser_UnknownTargetRole(t *testing.T) {
	t.Parallel()
	repo := newStubUserRepo()
	svc := NewAuthService(repo, "secret", 24)

	_, err := svc.ProvisionUser(context.Background(), uuid.New(), domain.RoleAdmin, ports.ProvisionUserRequest{
		Email: "u@example.com", Password: "secret12", FirstName: "A", LastName: "B", Role: "superuser",
	})
	assert.ErrorIs(t, err, domain.ErrInvalidRole)
}

func TestAuthService_ProvisionUser_CallerEmployeeRejected(t *testing.T) {
	t.Parallel()
	repo := newStubUserRepo()
	svc := NewAuthService(repo, "secret", 24)

	_, err := svc.ProvisionUser(context.Background(), uuid.New(), domain.RoleEmployee, ports.ProvisionUserRequest{
		Email: "u@example.com", Password: "secret12", FirstName: "A", LastName: "B", Role: domain.RoleClient,
	})
	assert.ErrorIs(t, err, domain.ErrPermissionDenied)
}

func TestAuthService_ProvisionUser_DuplicateEmail(t *testing.T) {
	t.Parallel()
	repo := newStubUserRepo()
	existing, err := domain.NewUser("dupprov@example.com", "x", "E", "X", domain.RoleClient)
	require.NoError(t, err)
	repo.byEmail[existing.Email] = existing

	svc := NewAuthService(repo, "secret", 24)
	_, err = svc.ProvisionUser(context.Background(), uuid.New(), domain.RoleAdmin, ports.ProvisionUserRequest{
		Email: "dupprov@example.com", Password: "secret12", FirstName: "A", LastName: "B", Role: domain.RoleClient,
	})
	assert.ErrorIs(t, err, domain.ErrUserAlreadyExists)
}

func TestAuthService_ProvisionUser_TrimsTargetRoleWhitespace(t *testing.T) {
	t.Parallel()
	repo := newStubUserRepo()
	svc := NewAuthService(repo, "secret", 24)

	user, err := svc.ProvisionUser(context.Background(), uuid.New(), domain.RoleAdmin, ports.ProvisionUserRequest{
		Email: "trim@example.com", Password: "secret12", FirstName: "A", LastName: "B",
		Role: "  " + domain.RoleClient + "  ",
	})
	require.NoError(t, err)
	require.NotNil(t, user)
	assert.Equal(t, domain.RoleClient, user.Role)
}
