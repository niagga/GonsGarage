package appointment

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

type apptTestUserRepo struct {
	users  map[uuid.UUID]*domain.User
	getErr error
}

func (r *apptTestUserRepo) Create(ctx context.Context, user *domain.User) error { return nil }
func (r *apptTestUserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	return nil, nil
}
func (r *apptTestUserRepo) GetByRole(ctx context.Context, role string, limit, offset int) ([]*domain.User, error) {
	return nil, nil
}
func (r *apptTestUserRepo) List(ctx context.Context, limit, offset int) ([]*domain.User, error) {
	return nil, nil
}
func (r *apptTestUserRepo) Update(ctx context.Context, user *domain.User) error { return nil }
func (r *apptTestUserRepo) Delete(ctx context.Context, id uuid.UUID) error      { return nil }
func (r *apptTestUserRepo) UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string) error {
	return nil
}
func (r *apptTestUserRepo) GetActiveUsers(ctx context.Context, limit, offset int) ([]*domain.User, error) {
	return nil, nil
}

func (r *apptTestUserRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	if r.getErr != nil {
		return nil, r.getErr
	}
	if r.users == nil {
		return nil, nil
	}
	u, ok := r.users[id]
	if !ok {
		return nil, nil
	}
	return u, nil
}

type stubApptRepo struct {
	created   []*domain.Appointment
	createErr error
	byID      map[uuid.UUID]*domain.Appointment
	lastList  *ports.AppointmentFilters
	listApps  []*domain.Appointment
	listTotal int64
	listErr   error
	updateErr error
	deleteErr error
	countN    int64
	countErr  error
}

func (s *stubApptRepo) Create(ctx context.Context, a *domain.Appointment) error {
	if s.createErr != nil {
		return s.createErr
	}
	s.created = append(s.created, a)
	return nil
}

func (s *stubApptRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Appointment, error) {
	if s.byID == nil {
		return nil, domain.ErrAppointmentNotFound
	}
	a, ok := s.byID[id]
	if !ok {
		return nil, domain.ErrAppointmentNotFound
	}
	return a, nil
}

func (s *stubApptRepo) Update(ctx context.Context, a *domain.Appointment) error {
	return s.updateErr
}

func (s *stubApptRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return s.deleteErr
}

func (s *stubApptRepo) List(ctx context.Context, filters *ports.AppointmentFilters) ([]*domain.Appointment, int64, error) {
	s.lastList = filters
	if s.listErr != nil {
		return nil, 0, s.listErr
	}
	return s.listApps, s.listTotal, nil
}

func (s *stubApptRepo) CountNonCancelledBetween(ctx context.Context, start, end time.Time, excludeID *uuid.UUID) (int64, error) {
	if s.countErr != nil {
		return 0, s.countErr
	}
	return s.countN, nil
}

type stubCarRepo struct {
	byID map[uuid.UUID]*domain.Car
}

func (s *stubCarRepo) Create(ctx context.Context, car *domain.Car) error { return nil }
func (s *stubCarRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Car, error) {
	if s.byID == nil {
		return nil, domain.ErrCarNotFound
	}
	c, ok := s.byID[id]
	if !ok {
		return nil, domain.ErrCarNotFound
	}
	return c, nil
}
func (s *stubCarRepo) GetByOwnerID(ctx context.Context, ownerID uuid.UUID) ([]*domain.Car, error) {
	return nil, nil
}
func (s *stubCarRepo) GetByLicensePlate(ctx context.Context, licensePlate string) (*domain.Car, error) {
	return nil, nil
}
func (s *stubCarRepo) List(ctx context.Context, limit, offset int) ([]*domain.Car, error) {
	return nil, nil
}
func (s *stubCarRepo) Update(ctx context.Context, car *domain.Car) error { return nil }
func (s *stubCarRepo) Delete(ctx context.Context, id uuid.UUID) error    { return nil }
func (s *stubCarRepo) GetWithRepairs(ctx context.Context, id uuid.UUID) (*domain.Car, error) {
	return nil, nil
}
func (s *stubCarRepo) GetDeletedByLicensePlate(ctx context.Context, licensePlate string) (*domain.Car, error) {
	return nil, nil
}
func (s *stubCarRepo) Restore(ctx context.Context, id uuid.UUID) error { return nil }

func TestAppointmentService_CreateAppointment_UserNotFound(t *testing.T) {
	t.Parallel()
	userID := uuid.New()
	svc := NewAppointmentService(&stubApptRepo{}, &apptTestUserRepo{users: map[uuid.UUID]*domain.User{}}, &stubCarRepo{})
	appt := sampleAppointment(userID, uuid.New())

	out, err := svc.CreateAppointment(context.Background(), appt, userID)
	require.ErrorIs(t, err, domain.ErrUserNotFound)
	assert.Nil(t, out)
}

func TestAppointmentService_CreateAppointment_GetUserError(t *testing.T) {
	t.Parallel()
	userID := uuid.New()
	repoErr := errors.New("db down")
	svc := NewAppointmentService(
		&stubApptRepo{},
		&apptTestUserRepo{getErr: repoErr},
		&stubCarRepo{},
	)
	appt := sampleAppointment(userID, uuid.New())

	out, err := svc.CreateAppointment(context.Background(), appt, userID)
	require.Error(t, err)
	assert.ErrorIs(t, err, repoErr)
	assert.Nil(t, out)
}

func TestAppointmentService_CreateAppointment_Success(t *testing.T) {
	t.Parallel()
	userID := uuid.New()
	carID := uuid.New()
	user, err := domain.NewUser("u@example.com", "pw", "U", "Ser", domain.RoleClient)
	require.NoError(t, err)
	user.ID = userID

	apptRepo := &stubApptRepo{}
	cars := &stubCarRepo{byID: map[uuid.UUID]*domain.Car{
		carID: {ID: carID, OwnerID: userID},
	}}
	svc := NewAppointmentService(
		apptRepo,
		&apptTestUserRepo{users: map[uuid.UUID]*domain.User{userID: user}},
		cars,
	)
	in := sampleAppointment(userID, carID)
	in.ID = uuid.Nil

	out, err := svc.CreateAppointment(context.Background(), in, userID)
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.NotEqual(t, uuid.Nil, out.ID)
	assert.False(t, out.CreatedAt.IsZero())
	assert.False(t, out.UpdatedAt.IsZero())
	require.Len(t, apptRepo.created, 1)
	assert.Equal(t, out.ID, apptRepo.created[0].ID)
}

func TestAppointmentService_CreateAppointment_RepoError(t *testing.T) {
	t.Parallel()
	userID := uuid.New()
	carID := uuid.New()
	user, err := domain.NewUser("u@example.com", "pw", "U", "Ser", domain.RoleClient)
	require.NoError(t, err)
	user.ID = userID

	createErr := errors.New("insert failed")
	cars := &stubCarRepo{byID: map[uuid.UUID]*domain.Car{carID: {ID: carID, OwnerID: userID}}}
	svc := NewAppointmentService(
		&stubApptRepo{createErr: createErr},
		&apptTestUserRepo{users: map[uuid.UUID]*domain.User{userID: user}},
		cars,
	)

	out, err := svc.CreateAppointment(context.Background(), sampleAppointment(userID, carID), userID)
	require.ErrorIs(t, err, createErr)
	assert.Nil(t, out)
}

func TestAppointmentService_ListAppointments_DefaultFilters(t *testing.T) {
	t.Parallel()
	userID := uuid.New()
	user, err := domain.NewUser("c@example.com", "pw", "C", "L", domain.RoleClient)
	require.NoError(t, err)
	user.ID = userID

	apptRepo := &stubApptRepo{listApps: []*domain.Appointment{}, listTotal: 0}
	svc := NewAppointmentService(apptRepo, &apptTestUserRepo{users: map[uuid.UUID]*domain.User{userID: user}}, &stubCarRepo{})

	apps, total, err := svc.ListAppointments(context.Background(), userID, nil)
	require.NoError(t, err)
	assert.Empty(t, apps)
	assert.Equal(t, int64(0), total)
	require.NotNil(t, apptRepo.lastList)
	assert.Equal(t, userID, *apptRepo.lastList.CustomerID)
	assert.Equal(t, 10, apptRepo.lastList.Limit)
	assert.Equal(t, 0, apptRepo.lastList.Offset)
	assert.Equal(t, "created_at", apptRepo.lastList.SortBy)
	assert.Equal(t, "DESC", apptRepo.lastList.SortOrder)
}

func TestAppointmentService_ListAppointments_FillsEmptySortAndLimit(t *testing.T) {
	t.Parallel()
	userID := uuid.New()
	user, err := domain.NewUser("c@example.com", "pw", "C", "L", domain.RoleClient)
	require.NoError(t, err)
	user.ID = userID

	apptRepo := &stubApptRepo{}
	svc := NewAppointmentService(apptRepo, &apptTestUserRepo{users: map[uuid.UUID]*domain.User{userID: user}}, &stubCarRepo{})

	f := &ports.AppointmentFilters{Limit: 0, SortBy: "", SortOrder: ""}
	_, _, err = svc.ListAppointments(context.Background(), userID, f)
	require.NoError(t, err)
	require.NotNil(t, apptRepo.lastList)
	assert.Equal(t, userID, *apptRepo.lastList.CustomerID)
	assert.Equal(t, 10, apptRepo.lastList.Limit)
	assert.Equal(t, "created_at", apptRepo.lastList.SortBy)
	assert.Equal(t, "DESC", apptRepo.lastList.SortOrder)
}

func TestAppointmentService_GetAppointment(t *testing.T) {
	t.Parallel()
	id := uuid.New()
	userID := uuid.New()
	user, err := domain.NewUser("c@example.com", "pw", "C", "L", domain.RoleClient)
	require.NoError(t, err)
	user.ID = userID

	expected := &domain.Appointment{ID: id, CustomerID: userID, ServiceType: "oil"}
	svc := NewAppointmentService(
		&stubApptRepo{byID: map[uuid.UUID]*domain.Appointment{id: expected}},
		&apptTestUserRepo{users: map[uuid.UUID]*domain.User{userID: user}},
		&stubCarRepo{},
	)

	out, err := svc.GetAppointment(context.Background(), id, userID)
	require.NoError(t, err)
	assert.Equal(t, expected, out)
}

func TestAppointmentService_UpdateAppointment_RepoError(t *testing.T) {
	t.Parallel()
	apptID := uuid.New()
	userID := uuid.New()
	user, err := domain.NewUser("c@example.com", "pw", "C", "L", domain.RoleClient)
	require.NoError(t, err)
	user.ID = userID

	existing := &domain.Appointment{
		ID:          apptID,
		CustomerID:  userID,
		CarID:       uuid.New(),
		Status:      domain.AppointmentStatusScheduled,
		ServiceType: "svc",
		// Fixed local wall time inside validateWorkshopClock windows (09:30–12:30 / 14:00–17:30).
		// time.Now().UTC() is non-deterministic vs. local business hours and can skip repo.Update entirely.
		ScheduledAt: time.Date(2026, 6, 15, 10, 0, 0, 0, time.Local),
	}
	updErr := errors.New("update failed")
	svc := NewAppointmentService(
		&stubApptRepo{byID: map[uuid.UUID]*domain.Appointment{apptID: existing}, updateErr: updErr},
		&apptTestUserRepo{users: map[uuid.UUID]*domain.User{userID: user}},
		&stubCarRepo{},
	)
	patch := &domain.Appointment{ID: apptID, Notes: "n"}

	out, err := svc.UpdateAppointment(context.Background(), patch, userID)
	require.ErrorIs(t, err, updErr)
	assert.Nil(t, out)
}

func TestAppointmentService_DeleteAppointment(t *testing.T) {
	t.Parallel()
	apptID := uuid.New()
	userID := uuid.New()
	user, err := domain.NewUser("c@example.com", "pw", "C", "L", domain.RoleClient)
	require.NoError(t, err)
	user.ID = userID
	existing := &domain.Appointment{ID: apptID, CustomerID: userID}

	svc := NewAppointmentService(
		&stubApptRepo{byID: map[uuid.UUID]*domain.Appointment{apptID: existing}},
		&apptTestUserRepo{users: map[uuid.UUID]*domain.User{userID: user}},
		&stubCarRepo{},
	)
	err = svc.DeleteAppointment(context.Background(), apptID, userID)
	require.NoError(t, err)
}

func TestAppointmentService_DeleteAppointment_RepoError(t *testing.T) {
	t.Parallel()
	apptID := uuid.New()
	userID := uuid.New()
	user, err := domain.NewUser("c@example.com", "pw", "C", "L", domain.RoleClient)
	require.NoError(t, err)
	user.ID = userID
	existing := &domain.Appointment{ID: apptID, CustomerID: userID}

	delErr := errors.New("delete failed")
	svc := NewAppointmentService(
		&stubApptRepo{byID: map[uuid.UUID]*domain.Appointment{apptID: existing}, deleteErr: delErr},
		&apptTestUserRepo{users: map[uuid.UUID]*domain.User{userID: user}},
		&stubCarRepo{},
	)

	err = svc.DeleteAppointment(context.Background(), apptID, userID)
	require.ErrorIs(t, err, delErr)
}

// Client role: list must never use another user's customer_id, even if passed in filters (tenant isolation).
func TestAppointmentService_ListAppointments_ClientAlwaysOwnCustomerID(t *testing.T) {
	t.Parallel()
	clientID := uuid.New()
	otherID := uuid.New()
	user, err := domain.NewUser("c@example.com", "pw", "C", "L", domain.RoleClient)
	require.NoError(t, err)
	user.ID = clientID

	apptRepo := &stubApptRepo{listApps: []*domain.Appointment{}, listTotal: 0}
	svc := NewAppointmentService(
		apptRepo,
		&apptTestUserRepo{users: map[uuid.UUID]*domain.User{clientID: user}},
		&stubCarRepo{},
	)
	malicious := otherID
	filters := &ports.AppointmentFilters{CustomerID: &malicious, Limit: 10, Offset: 0}

	_, _, err = svc.ListAppointments(context.Background(), clientID, filters)
	require.NoError(t, err)
	require.NotNil(t, apptRepo.lastList)
	require.NotNil(t, apptRepo.lastList.CustomerID)
	assert.Equal(t, clientID, *apptRepo.lastList.CustomerID, "client list must be scoped to the authenticated user")
}

func TestAppointmentService_CreateAppointment_StaffDerivesCustomerFromCarOwner(t *testing.T) {
	t.Parallel()
	staffID := uuid.New()
	ownerID := uuid.New()
	carID := uuid.New()

	staff, err := domain.NewUser("staff@example.com", "pw", "S", "Taff", domain.RoleEmployee)
	require.NoError(t, err)
	staff.ID = staffID

	apptRepo := &stubApptRepo{}
	cars := &stubCarRepo{byID: map[uuid.UUID]*domain.Car{
		carID: {ID: carID, OwnerID: ownerID},
	}}
	svc := NewAppointmentService(
		apptRepo,
		&apptTestUserRepo{users: map[uuid.UUID]*domain.User{staffID: staff}},
		cars,
	)

	// Staff omits customerID (uuid.Nil) — common when FE schedules from a car detail page.
	in := sampleAppointment(uuid.Nil, carID)
	in.ID = uuid.Nil

	out, err := svc.CreateAppointment(context.Background(), in, staffID)
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, ownerID, out.CustomerID)
	require.Len(t, apptRepo.created, 1)
	assert.Equal(t, ownerID, apptRepo.created[0].CustomerID)
}

func TestAppointmentService_CreateAppointment_StaffWrongCustomerForbidden(t *testing.T) {
	t.Parallel()
	staffID := uuid.New()
	ownerID := uuid.New()
	otherID := uuid.New()
	carID := uuid.New()

	staff, err := domain.NewUser("staff2@example.com", "pw", "S", "Taff", domain.RoleEmployee)
	require.NoError(t, err)
	staff.ID = staffID

	svc := NewAppointmentService(
		&stubApptRepo{},
		&apptTestUserRepo{users: map[uuid.UUID]*domain.User{staffID: staff}},
		&stubCarRepo{byID: map[uuid.UUID]*domain.Car{
			carID: {ID: carID, OwnerID: ownerID},
		}},
	)

	out, err := svc.CreateAppointment(context.Background(), sampleAppointment(otherID, carID), staffID)
	require.ErrorIs(t, err, domain.ErrUnauthorizedAccess)
	assert.Nil(t, out)
}

func sampleAppointment(customerID, carID uuid.UUID) *domain.Appointment {
	return &domain.Appointment{
		CustomerID:  customerID,
		CarID:       carID,
		ServiceType: "inspection",
		Status:      domain.AppointmentStatusScheduled,
		// Fixed local time inside workshop morning window (9:30–12:30)
		ScheduledAt: time.Date(2030, 6, 15, 10, 30, 0, 0, time.Local),
	}
}
