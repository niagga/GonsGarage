package appointment

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"strings"

	"github.com/gaston-garcia-cegid/gonsgarage/internal/core/ports"
	"github.com/gaston-garcia-cegid/gonsgarage/internal/domain"
	"github.com/google/uuid"
)

type AppointmentService struct {
	repo     ports.AppointmentRepository
	userRepo ports.UserRepository
	carRepo  ports.CarRepository
}

// NewAppointmentService wires appointment persistence, users, and cars (car ownership is validated on create/update).
func NewAppointmentService(
	repo ports.AppointmentRepository,
	userRepo ports.UserRepository,
	carRepo ports.CarRepository,
) *AppointmentService {
	return &AppointmentService{
		repo:     repo,
		userRepo: userRepo,
		carRepo:  carRepo,
	}
}

func canAccessAppointment(u *domain.User, appt *domain.Appointment, requestingUserID uuid.UUID) bool {
	if u == nil || appt == nil {
		return false
	}
	if u.IsClient() {
		return appt.CustomerID == requestingUserID
	}
	return u.IsEmployee()
}

// CreateAppointment creates a new appointment (client: own cars only; staff: on behalf of customerID).
func (s *AppointmentService) CreateAppointment(
	ctx context.Context,
	appointment *domain.Appointment,
	requestingUserID uuid.UUID,
) (*domain.Appointment, error) {
	queryCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	requestingUser, err := s.userRepo.GetByID(queryCtx, requestingUserID)
	if err != nil {
		log.Printf("failed to get requesting user: userID=%s, error=%v", requestingUserID, err)
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if requestingUser == nil {
		return nil, domain.ErrUserNotFound
	}

	car, err := s.carRepo.GetByID(queryCtx, appointment.CarID)
	if err != nil {
		if errors.Is(err, domain.ErrCarNotFound) {
			return nil, domain.ErrInvalidAppointmentData
		}
		return nil, fmt.Errorf("failed to get car: %w", err)
	}
	if car == nil {
		return nil, domain.ErrInvalidAppointmentData
	}

	// Resolve customer: clients always book as themselves; staff may omit customerID
	// and we derive it from the car owner (FE schedule-from-car often skips it).
	customerID := appointment.CustomerID
	if requestingUser.IsClient() {
		customerID = requestingUserID
		if car.OwnerID != customerID {
			return nil, domain.ErrUnauthorizedAccess
		}
	} else if requestingUser.IsEmployee() {
		if customerID == uuid.Nil {
			customerID = car.OwnerID
		} else if car.OwnerID != customerID {
			return nil, domain.ErrUnauthorizedAccess
		}
	} else {
		return nil, domain.ErrUnauthorizedAccess
	}

	if strings.TrimSpace(appointment.ServiceType) == "" {
		return nil, domain.ErrInvalidAppointmentData
	}
	if appointment.ScheduledAt.IsZero() {
		return nil, domain.ErrInvalidAppointmentData
	}
	if err := validateWorkshopClock(appointment.ScheduledAt); err != nil {
		return nil, err
	}
	dayStart, dayEnd := dayRangeUTC(appointment.ScheduledAt)
	nSameDay, err := s.repo.CountNonCancelledBetween(queryCtx, dayStart, dayEnd, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to count appointments for day: %w", err)
	}
	if nSameDay >= MaxAppointmentsPerDay {
		return nil, domain.ErrAppointmentDailyCapReached
	}

	appointment.CustomerID = customerID
	if appointment.Status == "" {
		appointment.Status = domain.AppointmentStatusScheduled
	}
	if !domain.ValidateAppointmentStatus(appointment.Status) {
		return nil, domain.ErrInvalidAppointmentData
	}

	appointment.ID = uuid.New()
	appointment.CreatedAt = time.Now()
	appointment.UpdatedAt = time.Now()

	if err := s.repo.Create(ctx, appointment); err != nil {
		return nil, err
	}

	return appointment, nil
}

// UpdateAppointment updates an existing appointment; clients may only change their own rows.
func (s *AppointmentService) UpdateAppointment(ctx context.Context, appointment *domain.Appointment, requestingUserID uuid.UUID) (*domain.Appointment, error) {
	requestingUser, err := s.userRepo.GetByID(ctx, requestingUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if requestingUser == nil {
		return nil, domain.ErrUserNotFound
	}

	existing, err := s.repo.GetByID(ctx, appointment.ID)
	if err != nil {
		if errors.Is(err, domain.ErrAppointmentNotFound) {
			return nil, domain.ErrAppointmentNotFound
		}
		return nil, err
	}
	if existing == nil {
		return nil, domain.ErrAppointmentNotFound
	}

	if !canAccessAppointment(requestingUser, existing, requestingUserID) {
		return nil, domain.ErrUnauthorizedAccess
	}

	merged := *existing
	if appointment.CarID != uuid.Nil {
		car, err := s.carRepo.GetByID(ctx, appointment.CarID)
		if err != nil {
			if errors.Is(err, domain.ErrCarNotFound) {
				return nil, domain.ErrInvalidAppointmentData
			}
			return nil, fmt.Errorf("failed to get car: %w", err)
		}
		if car == nil || car.OwnerID != merged.CustomerID {
			return nil, domain.ErrInvalidAppointmentData
		}
		merged.CarID = appointment.CarID
	}
	if !appointment.ScheduledAt.IsZero() {
		merged.ScheduledAt = appointment.ScheduledAt
	}
	if appointment.Status != "" {
		if !domain.ValidateAppointmentStatus(appointment.Status) {
			return nil, domain.ErrInvalidAppointmentData
		}
		merged.Status = appointment.Status
	}
	merged.Notes = appointment.Notes
	if appointment.ServiceType != "" {
		merged.ServiceType = appointment.ServiceType
	}
	merged.UpdatedAt = time.Now()

	if strings.TrimSpace(merged.ServiceType) == "" {
		return nil, domain.ErrInvalidAppointmentData
	}
	if err := validateWorkshopClock(merged.ScheduledAt); err != nil {
		return nil, err
	}
	uDay0, uDay1 := dayRangeUTC(merged.ScheduledAt)
	nSameDay, err := s.repo.CountNonCancelledBetween(ctx, uDay0, uDay1, &merged.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to count appointments for day: %w", err)
	}
	if nSameDay >= MaxAppointmentsPerDay {
		return nil, domain.ErrAppointmentDailyCapReached
	}

	if err := s.repo.Update(ctx, &merged); err != nil {
		return nil, err
	}
	return &merged, nil
}

// GetAppointment retrieves an appointment by ID with authorization.
func (s *AppointmentService) GetAppointment(ctx context.Context, appointmentID uuid.UUID, requestingUserID uuid.UUID) (*domain.Appointment, error) {
	requestingUser, err := s.userRepo.GetByID(ctx, requestingUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if requestingUser == nil {
		return nil, domain.ErrUserNotFound
	}

	appointment, err := s.repo.GetByID(ctx, appointmentID)
	if err != nil {
		if errors.Is(err, domain.ErrAppointmentNotFound) {
			return nil, domain.ErrAppointmentNotFound
		}
		return nil, err
	}
	if appointment == nil {
		return nil, domain.ErrAppointmentNotFound
	}

	if !canAccessAppointment(requestingUser, appointment, requestingUserID) {
		return nil, domain.ErrUnauthorizedAccess
	}

	return appointment, nil
}

// ListAppointments lists appointments: clients are scoped to their customer_id; staff may filter.
func (s *AppointmentService) ListAppointments(ctx context.Context, requestingUserID uuid.UUID, filters *ports.AppointmentFilters) ([]*domain.Appointment, int64, error) {
	requestingUser, err := s.userRepo.GetByID(ctx, requestingUserID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get user: %w", err)
	}
	if requestingUser == nil {
		return nil, 0, domain.ErrUserNotFound
	}

	if requestingUser.IsClient() {
		cid := requestingUserID
		if filters == nil {
			filters = &ports.AppointmentFilters{}
		}
		filters.CustomerID = &cid
	} else if !requestingUser.IsEmployee() {
		return nil, 0, domain.ErrUnauthorizedAccess
	}

	if filters == nil {
		filters = &ports.AppointmentFilters{
			Limit:     10,
			Offset:    0,
			SortBy:    "created_at",
			SortOrder: "DESC",
		}
	}

	if filters.Limit == 0 {
		filters.Limit = 10
	}

	if filters.SortBy == "" {
		filters.SortBy = "created_at"
	}

	if filters.SortOrder == "" {
		filters.SortOrder = "DESC"
	}

	appointments, total, err := s.repo.List(ctx, filters)
	if err != nil {
		return nil, 0, err
	}

	return appointments, total, nil
}

// DeleteAppointment deletes an appointment by ID with authorization.
func (s *AppointmentService) DeleteAppointment(ctx context.Context, appointmentID uuid.UUID, requestingUserID uuid.UUID) error {
	requestingUser, err := s.userRepo.GetByID(ctx, requestingUserID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}
	if requestingUser == nil {
		return domain.ErrUserNotFound
	}

	appt, err := s.repo.GetByID(ctx, appointmentID)
	if err != nil {
		if errors.Is(err, domain.ErrAppointmentNotFound) {
			return domain.ErrAppointmentNotFound
		}
		return err
	}
	if appt == nil {
		return domain.ErrAppointmentNotFound
	}

	if !canAccessAppointment(requestingUser, appt, requestingUserID) {
		return domain.ErrUnauthorizedAccess
	}

	return s.repo.Delete(ctx, appointmentID)
}
