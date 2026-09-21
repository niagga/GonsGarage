package car

import (
	"context"
	"testing"

	"github.com/gaston-garcia-cegid/gonsgarage/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type carTestUserRepo struct {
	users map[uuid.UUID]*domain.User
}

func (r *carTestUserRepo) Create(ctx context.Context, user *domain.User) error { return nil }
func (r *carTestUserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	return nil, nil
}
func (r *carTestUserRepo) GetByRole(ctx context.Context, role string, limit, offset int) ([]*domain.User, error) {
	return nil, nil
}
func (r *carTestUserRepo) List(ctx context.Context, limit, offset int) ([]*domain.User, error) {
	return nil, nil
}
func (r *carTestUserRepo) Update(ctx context.Context, user *domain.User) error { return nil }
func (r *carTestUserRepo) Delete(ctx context.Context, id uuid.UUID) error      { return nil }
func (r *carTestUserRepo) UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string) error {
	return nil
}
func (r *carTestUserRepo) GetActiveUsers(ctx context.Context, limit, offset int) ([]*domain.User, error) {
	return nil, nil
}

func (r *carTestUserRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	u, ok := r.users[id]
	if !ok {
		return nil, domain.ErrUserNotFound
	}
	return u, nil
}

type carTestCarRepo struct {
	byPlate map[string]*domain.Car
	byID    map[uuid.UUID]*domain.Car
	created []*domain.Car
	listOut []*domain.Car
}

func newCarTestCarRepo() *carTestCarRepo {
	return &carTestCarRepo{
		byPlate: make(map[string]*domain.Car),
		byID:    make(map[uuid.UUID]*domain.Car),
	}
}

func (r *carTestCarRepo) Create(ctx context.Context, car *domain.Car) error {
	r.created = append(r.created, car)
	return nil
}

func (r *carTestCarRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Car, error) {
	c, ok := r.byID[id]
	if !ok {
		return nil, nil
	}
	return c, nil
}

func (r *carTestCarRepo) GetByOwnerID(ctx context.Context, ownerID uuid.UUID) ([]*domain.Car, error) {
	return nil, nil
}

func (r *carTestCarRepo) GetByLicensePlate(ctx context.Context, licensePlate string) (*domain.Car, error) {
	c, ok := r.byPlate[licensePlate]
	if !ok {
		return nil, nil
	}
	return c, nil
}

func (r *carTestCarRepo) List(ctx context.Context, limit, offset int) ([]*domain.Car, error) {
	if r.listOut != nil {
		return r.listOut, nil
	}
	return nil, nil
}

func (r *carTestCarRepo) Update(ctx context.Context, car *domain.Car) error { return nil }

func (r *carTestCarRepo) Delete(ctx context.Context, id uuid.UUID) error { return nil }

func (r *carTestCarRepo) GetWithRepairs(ctx context.Context, id uuid.UUID) (*domain.Car, error) {
	return nil, nil
}

func (r *carTestCarRepo) GetDeletedByLicensePlate(ctx context.Context, licensePlate string) (*domain.Car, error) {
	return nil, nil
}

func (r *carTestCarRepo) Restore(ctx context.Context, id uuid.UUID) error { return nil }

type noopCache struct{}

func (noopCache) Get(ctx context.Context, key string, dest interface{}) error { return nil }

func (noopCache) Set(ctx context.Context, key string, value interface{}, ttl int) error { return nil }

func (noopCache) Delete(ctx context.Context, key string) error { return nil }

func TestCarService_CreateCar_ClientOwnsCar(t *testing.T) {
	t.Parallel()
	clientID := uuid.New()
	client, err := domain.NewUser("c@example.com", "pw", "C", "L", domain.RoleClient)
	require.NoError(t, err)
	client.ID = clientID

	userRepo := &carTestUserRepo{users: map[uuid.UUID]*domain.User{clientID: client}}
	carRepo := newCarTestCarRepo()
	svc := NewCarService(carRepo, userRepo, noopCache{})

	car := &domain.Car{
		Make:         "Toyota",
		Model:        "Corolla",
		Year:         2020,
		LicensePlate: "ABC-1234",
		Color:        "Blue",
		Mileage:      10000,
	}

	out, err := svc.CreateCar(context.Background(), car, clientID)
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, clientID, out.OwnerID)
	assert.NotEqual(t, uuid.Nil, out.ID)
	assert.Len(t, carRepo.created, 1)
}

func TestCarService_CreateCar_DuplicatePlate(t *testing.T) {
	t.Parallel()
	clientID := uuid.New()
	client, err := domain.NewUser("c2@example.com", "pw", "C", "L", domain.RoleClient)
	require.NoError(t, err)
	client.ID = clientID

	userRepo := &carTestUserRepo{users: map[uuid.UUID]*domain.User{clientID: client}}
	carRepo := newCarTestCarRepo()
	carRepo.byPlate["DUP-1"] = &domain.Car{ID: uuid.New(), LicensePlate: "DUP-1"}

	svc := NewCarService(carRepo, userRepo, noopCache{})
	_, err = svc.CreateCar(context.Background(), &domain.Car{
		Make: "VW", Model: "Golf", Year: 2019, LicensePlate: "DUP-1", Color: "Red", Mileage: 1,
	}, clientID)
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrCarAlreadyExists)
}

func TestCarService_CreateCar_ManagerRequiresClientOwner(t *testing.T) {
	t.Parallel()
	managerID := uuid.New()
	manager, err := domain.NewUser("m@example.com", "pw", "M", "G", domain.RoleManager)
	require.NoError(t, err)
	manager.ID = managerID

	userRepo := &carTestUserRepo{users: map[uuid.UUID]*domain.User{managerID: manager}}
	svc := NewCarService(newCarTestCarRepo(), userRepo, noopCache{})

	_, err = svc.CreateCar(context.Background(), &domain.Car{
		Make: "Ford", Model: "Focus", Year: 2020, LicensePlate: "MGR-1", Color: "Gray", Mileage: 100,
	}, managerID)
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrCarOwnerRequiredForStaff)
}

func TestCarService_CreateCar_ManagerOwnerMustBeClient(t *testing.T) {
	t.Parallel()
	managerID := uuid.New()
	employeeID := uuid.New()

	manager, err := domain.NewUser("m2@example.com", "pw", "M", "G", domain.RoleManager)
	require.NoError(t, err)
	manager.ID = managerID

	employee, err := domain.NewUser("e2@example.com", "pw", "E", "E", domain.RoleEmployee)
	require.NoError(t, err)
	employee.ID = employeeID

	userRepo := &carTestUserRepo{users: map[uuid.UUID]*domain.User{
		managerID:  manager,
		employeeID: employee,
	}}
	svc := NewCarService(newCarTestCarRepo(), userRepo, noopCache{})

	_, err = svc.CreateCar(context.Background(), &domain.Car{
		OwnerID: employeeID,
		Make: "Ford", Model: "Focus", Year: 2020, LicensePlate: "MGR-2", Color: "Gray", Mileage: 100,
	}, managerID)
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrCarOwnerMustBeClient)
}

func TestCarService_CreateCar_EmployeeDenied(t *testing.T) {
	t.Parallel()
	employeeID := uuid.New()
	clientID := uuid.New()

	employee, err := domain.NewUser("emp-create@example.com", "pw", "E", "M", domain.RoleEmployee)
	require.NoError(t, err)
	employee.ID = employeeID

	client, err := domain.NewUser("owner@example.com", "pw", "C", "L", domain.RoleClient)
	require.NoError(t, err)
	client.ID = clientID

	userRepo := &carTestUserRepo{users: map[uuid.UUID]*domain.User{
		employeeID: employee,
		clientID:   client,
	}}
	svc := NewCarService(newCarTestCarRepo(), userRepo, noopCache{})

	_, err = svc.CreateCar(context.Background(), &domain.Car{
		OwnerID: clientID,
		Make: "Seat", Model: "Ibiza", Year: 2022, LicensePlate: "EMP-1", Color: "White", Mileage: 15,
	}, employeeID)
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrUnauthorizedAccess)
}

func TestCarService_GetCar_EmployeeCanViewAnyCar(t *testing.T) {
	t.Parallel()
	empID := uuid.New()
	emp, err := domain.NewUser("e@example.com", "pw", "E", "E", domain.RoleEmployee)
	require.NoError(t, err)
	emp.ID = empID

	ownerID := uuid.New()
	carID := uuid.New()
	owned := &domain.Car{ID: carID, OwnerID: ownerID, Make: "X", Model: "Y", Year: 2020, LicensePlate: "E-1", Color: "Black", Mileage: 0}

	userRepo := &carTestUserRepo{users: map[uuid.UUID]*domain.User{empID: emp}}
	carRepo := newCarTestCarRepo()
	carRepo.byID[carID] = owned

	svc := NewCarService(carRepo, userRepo, noopCache{})
	out, err := svc.GetCar(context.Background(), carID, empID)
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, carID, out.ID)
}

func TestCarService_ListCars_AdminUsesInventoryList(t *testing.T) {
	t.Parallel()
	adminID := uuid.New()
	admin, err := domain.NewUser("a@example.com", "pw", "A", "D", domain.RoleAdmin)
	require.NoError(t, err)
	admin.ID = adminID

	userRepo := &carTestUserRepo{users: map[uuid.UUID]*domain.User{adminID: admin}}
	carRepo := newCarTestCarRepo()
	carRepo.listOut = []*domain.Car{
		{ID: uuid.New(), OwnerID: uuid.New(), Make: "VW", Model: "Golf", Year: 2019, LicensePlate: "L-99", Color: "Red", Mileage: 2},
	}

	svc := NewCarService(carRepo, userRepo, noopCache{})
	out, err := svc.ListCars(context.Background(), adminID, nil, 10, 0)
	require.NoError(t, err)
	require.Len(t, out, 1)
	assert.Equal(t, "L-99", out[0].LicensePlate)
}

func TestCarService_GetCar_ClientOtherOwnerDenied(t *testing.T) {
	t.Parallel()
	clientID := uuid.New()
	otherID := uuid.New()
	client, err := domain.NewUser("me@example.com", "pw", "M", "E", domain.RoleClient)
	require.NoError(t, err)
	client.ID = clientID

	carID := uuid.New()
	owned := &domain.Car{ID: carID, OwnerID: otherID, Make: "X", Model: "Y", Year: 2020, LicensePlate: "Z-1", Color: "Black", Mileage: 0}

	userRepo := &carTestUserRepo{users: map[uuid.UUID]*domain.User{clientID: client}}
	carRepo := newCarTestCarRepo()
	carRepo.byID[carID] = owned

	svc := NewCarService(carRepo, userRepo, noopCache{})
	_, err = svc.GetCar(context.Background(), carID, clientID)
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrUnauthorizedAccess)
}

func TestCarService_GetCar_ClientOwnCar(t *testing.T) {
	t.Parallel()
	clientID := uuid.New()
	carID := uuid.New()
	client, err := domain.NewUser("own@example.com", "pw", "O", "W", domain.RoleClient)
	require.NoError(t, err)
	client.ID = clientID

	owned := &domain.Car{ID: carID, OwnerID: clientID, Make: "X", Model: "Y", Year: 2020, LicensePlate: "OWN-1", Color: "Black", Mileage: 0}
	userRepo := &carTestUserRepo{users: map[uuid.UUID]*domain.User{clientID: client}}
	carRepo := newCarTestCarRepo()
	carRepo.byID[carID] = owned

	svc := NewCarService(carRepo, userRepo, noopCache{})
	out, err := svc.GetCar(context.Background(), carID, clientID)
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, carID, out.ID)
	assert.Equal(t, clientID, out.OwnerID)
}

func TestCarService_ListCars_ClientRejectsForeignOwnerFilter(t *testing.T) {
	t.Parallel()
	clientID := uuid.New()
	otherID := uuid.New()
	client, err := domain.NewUser("c@example.com", "pw", "C", "L", domain.RoleClient)
	require.NoError(t, err)
	client.ID = clientID

	userRepo := &carTestUserRepo{users: map[uuid.UUID]*domain.User{clientID: client}}
	svc := NewCarService(newCarTestCarRepo(), userRepo, noopCache{})

	_, err = svc.ListCars(context.Background(), clientID, &otherID, 10, 0)
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrUnauthorizedAccess)
}
