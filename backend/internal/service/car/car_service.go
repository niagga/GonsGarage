package car

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/gaston-garcia-cegid/gonsgarage/internal/core/ports"
	"github.com/gaston-garcia-cegid/gonsgarage/internal/domain"
	"github.com/google/uuid"
)

type CarService struct {
	carRepo   ports.CarRepository
	userRepo  ports.UserRepository
	logger    ports.Logger
	cacheRepo ports.CacheRepository
}

func NewCarService(
	carRepo ports.CarRepository,
	userRepo ports.UserRepository,
	cacheRepo ports.CacheRepository) *CarService {
	return &CarService{
		carRepo:   carRepo,
		userRepo:  userRepo,
		cacheRepo: cacheRepo,
	}
}

// CreateCar creates a new car for a client
func (uc *CarService) CreateCar(ctx context.Context, car *domain.Car, requestingUserID uuid.UUID) (*domain.Car, error) {
	// ✅ Get requesting user with timeout
	queryCtx, cancel := context.WithTimeout(ctx, 10*time.Second) // ✅ Increase timeout
	defer cancel()

	requestingUser, err := uc.userRepo.GetByID(queryCtx, requestingUserID)
	if err != nil {
		log.Printf("failed to get requesting user: userID=%s, error=%v", requestingUserID, err)
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// ✅ Check if license plate already exists (only active cars)
	existingCar, err := uc.carRepo.GetByLicensePlate(queryCtx, car.LicensePlate)
	if err != nil {
		log.Printf("error checking license plate: error=%v", err)
		return nil, fmt.Errorf("failed to check license plate: %w", err)
	}

	// ✅ If active car with same plate exists, reject
	if existingCar != nil {
		log.Printf("license plate already exists: plate=%s, existing_car_id=%s", car.LicensePlate, existingCar.ID)
		return nil, domain.ErrCarAlreadyExists
	}

	// ✅ License plate is available (either never used or previously deleted)
	// ✅ Explicit nil check
	if requestingUser == nil {
		log.Printf("user not found: userID=%s", requestingUserID)
		return nil, domain.ErrUserNotFound
	}

	// ✅ For clients, ALWAYS set owner to themselves
	if requestingUser.Role == domain.RoleClient {
		car.OwnerID = requestingUserID
	} else if requestingUser.Role == domain.RoleAdmin || requestingUser.Role == domain.RoleManager {
		if car.OwnerID == uuid.Nil {
			return nil, domain.ErrCarOwnerRequiredForStaff
		}
	} else {
		return nil, domain.ErrUnauthorizedAccess
	}

	// ✅ OPTIMIZATION: Reuse requestingUser if owner is same user
	var owner *domain.User
	if car.OwnerID == requestingUserID {
		owner = requestingUser // ✅ No second query needed!
	} else {
		// ✅ Only query if owner is different user
		owner, err = uc.userRepo.GetByID(queryCtx, car.OwnerID)
		if err != nil {
			log.Printf("failed to get car owner: owner_id=%s, error=%v", car.OwnerID, err)
			if errors.Is(err, domain.ErrUserNotFound) {
				return nil, domain.ErrCarOwnerNotFound
			}
			return nil, fmt.Errorf("failed to get car owner: %w", err)
		}
		if owner == nil {
			return nil, domain.ErrCarOwnerNotFound
		}
	}

	// ✅ Validate owner is a client
	if owner.Role != domain.RoleClient {
		return nil, domain.ErrCarOwnerMustBeClient
	}

	// ✅ Check if license plate already exists
	existingCar, err = uc.carRepo.GetByLicensePlate(queryCtx, car.LicensePlate)
	if err == nil && existingCar != nil {
		return nil, domain.ErrCarAlreadyExists
	}

	// ✅ Validate car data
	if err := car.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrInvalidCarData, err)
	}

	// ✅ Set metadata
	car.ID = uuid.New()
	car.CreatedAt = time.Now()
	car.UpdatedAt = time.Now()

	// ✅ Create the car
	if err := uc.carRepo.Create(ctx, car); err != nil {
		log.Printf("failed to create car: car=%+v, error=%v", car, err)
		return nil, fmt.Errorf("failed to create car: %w", err)
	}

	log.Printf("car created successfully: car_id=%s, owner_id=%s", car.ID, car.OwnerID)
	return car, nil
}

// GetCar retrieves a car by ID with permission checks
func (uc *CarService) GetCar(ctx context.Context, carID uuid.UUID, requestingUserID uuid.UUID) (*domain.Car, error) {
	// Get the requesting user
	requestingUser, err := uc.userRepo.GetByID(ctx, requestingUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Get the car
	car, err := uc.carRepo.GetByID(ctx, carID)
	if err != nil {
		return nil, fmt.Errorf("failed to get car: %w", err)
	}

	if car == nil {
		return nil, domain.ErrCarNotFound
	}

	// Clients only see their own cars; workshop staff can open any car record.
	if requestingUser.IsClient() && !car.IsOwnedBy(requestingUserID) {
		return nil, domain.ErrUnauthorizedAccess
	}
	if !requestingUser.IsClient() && !requestingUser.IsEmployee() {
		return nil, domain.ErrUnauthorizedAccess
	}

	return car, nil
}

// GetCarsByOwner retrieves all cars owned by a specific user
func (uc *CarService) GetCarsByOwner(ctx context.Context, ownerID uuid.UUID, requestingUserID uuid.UUID) ([]*domain.Car, error) {
	// Get the requesting user
	requestingUser, err := uc.userRepo.GetByID(ctx, requestingUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if requestingUser.IsClient() && ownerID != requestingUserID {
		return nil, domain.ErrUnauthorizedAccess
	}

	// Workshop roles (employee / manager / admin) can list any client's garage.
	if !requestingUser.IsClient() && !requestingUser.IsEmployee() {
		return nil, domain.ErrUnauthorizedAccess
	}

	cars, err := uc.carRepo.GetByOwnerID(ctx, ownerID)
	if err != nil {
		log.Printf("failed to get cars by owner: owner_id=%s error=%v", ownerID, err)
		return nil, fmt.Errorf("failed to get cars: %w", err)
	}

	return cars, nil
}

// ListCars lists cars for a client (always their own) or for staff (optional owner filter or paginated inventory).
func (uc *CarService) ListCars(ctx context.Context, requestingUserID uuid.UUID, ownerID *uuid.UUID, limit, offset int) ([]*domain.Car, error) {
	requestingUser, err := uc.userRepo.GetByID(ctx, requestingUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if requestingUser == nil {
		return nil, domain.ErrUserNotFound
	}

	if requestingUser.IsClient() {
		if ownerID != nil && *ownerID != requestingUserID {
			return nil, domain.ErrUnauthorizedAccess
		}
		return uc.carRepo.GetByOwnerID(ctx, requestingUserID)
	}

	if !requestingUser.IsEmployee() {
		return nil, domain.ErrUnauthorizedAccess
	}

	if ownerID != nil {
		return uc.carRepo.GetByOwnerID(ctx, *ownerID)
	}

	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}

	return uc.carRepo.List(ctx, limit, offset)
}

// UpdateCar updates an existing car
func (uc *CarService) UpdateCar(ctx context.Context, car *domain.Car, requestingUserID uuid.UUID) (*domain.Car, error) {
	// Get the requesting user
	requestingUser, err := uc.userRepo.GetByID(ctx, requestingUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Get the existing car
	existingCar, err := uc.carRepo.GetByID(ctx, car.ID)
	if err != nil || existingCar == nil {
		return nil, domain.ErrCarNotFound
	}

	// Check permissions: clients can only update their own cars
	if requestingUser.IsClient() && !existingCar.IsOwnedBy(requestingUserID) {
		return nil, domain.ErrUnauthorizedAccess
	}

	// Admins and managers can update any car
	if !requestingUser.IsClient() && !requestingUser.CanManageUsers() {
		return nil, domain.ErrUnauthorizedAccess
	}

	// Validate car data
	if err := car.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrInvalidCarData, err)
	}

	// Preserve some fields
	car.ID = existingCar.ID
	car.OwnerID = existingCar.OwnerID
	car.CreatedAt = existingCar.CreatedAt
	car.UpdatedAt = time.Now()

	// Update the car
	if err := uc.carRepo.Update(ctx, car); err != nil {
		log.Printf("failed to update car: car_id=%s error=%v", car.ID, err)
		return nil, fmt.Errorf("failed to update car: %w", err)
	}

	log.Printf("car updated successfully: car_id=%s", car.ID)
	return car, nil
}

// DeleteCar deletes a car (soft delete)
func (uc *CarService) DeleteCar(ctx context.Context, carID uuid.UUID, requestingUserID uuid.UUID) error {
	// Get the requesting user
	requestingUser, err := uc.userRepo.GetByID(ctx, requestingUserID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	// Get the car
	car, err := uc.carRepo.GetByID(ctx, carID)
	if err != nil || car == nil {
		return domain.ErrCarNotFound
	}

	// Check permissions: clients can only delete their own cars
	if requestingUser.IsClient() && !car.IsOwnedBy(requestingUserID) {
		return domain.ErrUnauthorizedAccess
	}

	// Admins and managers can delete any car
	if !requestingUser.IsClient() && !requestingUser.CanManageUsers() {
		return domain.ErrUnauthorizedAccess
	}

	// Delete the car
	if err := uc.carRepo.Delete(ctx, carID); err != nil {
		log.Printf("failed to delete car: car_id=%s error=%v", carID, err)
		return fmt.Errorf("failed to delete car: %w", err)
	}

	log.Printf("car deleted successfully: car_id=%s", carID)
	return nil
}

// GetCarWithRepairs retrieves a car with its repair history
func (uc *CarService) GetCarWithRepairs(ctx context.Context, carID uuid.UUID, requestingUserID uuid.UUID) (*domain.Car, error) {
	// Get the requesting user
	requestingUser, err := uc.userRepo.GetByID(ctx, requestingUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Get the car with repairs
	car, err := uc.carRepo.GetWithRepairs(ctx, carID)
	if err != nil || car == nil {
		return nil, domain.ErrCarNotFound
	}

	// Clients only see their own cars; workshop staff can open any car record.
	if requestingUser.IsClient() && !car.IsOwnedBy(requestingUserID) {
		return nil, domain.ErrUnauthorizedAccess
	}
	if !requestingUser.IsClient() && !requestingUser.IsEmployee() {
		return nil, domain.ErrUnauthorizedAccess
	}

	return car, nil
}
