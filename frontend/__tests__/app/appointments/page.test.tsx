import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { useRouter } from 'next/navigation';
import AppointmentsPage from '@/app/appointments/page';
import { useAuthStore } from '@/stores/auth.store';
import { useAppointments } from '@/stores/appointment.store';
import { useCarStore } from '@/stores/car.store';

// Mock dependencies (next/navigation mocked globally in __tests__/setup.ts)
jest.mock('@/stores/auth.store');
jest.mock('@/stores/appointment.store');
jest.mock('@/stores/car.store');

const mockedUseRouter = useRouter as jest.MockedFunction<typeof useRouter>;
const mockedUseAuthStore = useAuthStore as jest.MockedFunction<typeof useAuthStore>;
const mockedUseAppointments = useAppointments as jest.MockedFunction<typeof useAppointments>;
const mockedUseCarStore = useCarStore as jest.MockedFunction<typeof useCarStore>;

describe('AppointmentsPage', () => {
  const mockPush = jest.fn();
  const mockFetchAppointments = jest.fn();
  const mockFetchCars = jest.fn();
  const mockCancelAppointment = jest.fn();

  beforeEach(() => {
    mockedUseRouter.mockReturnValue({
      push: mockPush,
      back: jest.fn(),
      forward: jest.fn(),
      refresh: jest.fn(),
      replace: jest.fn(),
      prefetch: jest.fn(),
    });

    mockedUseAuthStore.mockReturnValue({
      currentUser: {
        id: 'user1',
        firstName: 'John',
        lastName: 'Doe',
        email: 'john@example.com',
        role: 'client',
        createdAt: '2024-01-01T00:00:00Z',
        updatedAt: '2024-01-01T00:00:00Z',
      },
      isAuthenticated: true,
      login: jest.fn(),
      logout: jest.fn(),
      register: jest.fn(),
      refreshToken: jest.fn(),
      clearError: jest.fn(),
      reset: jest.fn(),
      token: 'mock-token',
      isLoading: false,
      error: null,
    });

    mockedUseAppointments.mockReturnValue({
      appointments: [
        {
          id: '1',
          customerId: 'user1',
          carId: 'car1',
          serviceType: 'Oil Change',
          scheduledAt: '2024-12-15T10:00:00Z',
          status: 'scheduled',
          notes: 'Regular maintenance',
          createdAt: '2024-12-01T10:00:00Z',
          updatedAt: '2024-12-01T10:00:00Z',
        },
      ],
      isLoading: false,
      error: null,
      fetchAppointments: mockFetchAppointments,
      cancelAppointment: mockCancelAppointment,
      confirmAppointment: jest.fn(),
      completeAppointment: jest.fn(),
      createAppointment: jest.fn(),
      updateAppointment: jest.fn(),
      deleteAppointment: jest.fn(),
      selectAppointment: jest.fn(),
      setFilters: jest.fn(),
      clearError: jest.fn(),
      reset: jest.fn(),
      selectedAppointment: null,
      filters: { status: 'all' },
      isCreating: false,
      isUpdating: false,
      isDeleting: false,
    });

    mockedUseCarStore.mockReturnValue({
      cars: [
        {
          id: 'car1',
          ownerId: 'user1',
          make: 'Toyota',
          model: 'Camry',
          year: 2020,
          licensePlate: 'ABC123',
          color: 'Blue',
          vin: '1234567890',
          mileage: 50000,
          createdAt: '2024-01-01T00:00:00Z',
          updatedAt: '2024-01-01T00:00:00Z',
        },
      ],
      isLoading: false,
      error: null,
      fetchCars: mockFetchCars,
      createCar: jest.fn(),
      updateCar: jest.fn(),
      deleteCar: jest.fn(),
      selectCar: jest.fn(),
      clearError: jest.fn(),
      reset: jest.fn(),
      selectedCar: null,
    });

    jest.clearAllMocks();
  });

  it('should render appointments page correctly', async () => {
    // Act
    render(<AppointmentsPage />);

    // Assert
    expect(screen.getByText('GonsGarage')).toBeInTheDocument();
    expect(screen.getByText('My Appointments (1)')).toBeInTheDocument();
    expect(screen.getByText('Oil Change')).toBeInTheDocument();
    expect(screen.getByText('2020 Toyota Camry')).toBeInTheDocument();
    expect(screen.getByText('ABC123')).toBeInTheDocument();
  });

  it('should fetch appointments and cars on mount', async () => {
    // Act
    render(<AppointmentsPage />);

    // Assert
    expect(mockFetchAppointments).toHaveBeenCalledTimes(1);
    expect(mockFetchCars).toHaveBeenCalledTimes(1);
  });

  it('should redirect to login if user is not authenticated', () => {
    // Arrange
    mockedUseAuthStore.mockReturnValue({
      currentUser: null,
      isAuthenticated: false,
      login: jest.fn(),
      logout: jest.fn(),
      register: jest.fn(),
      refreshToken: jest.fn(),
      clearError: jest.fn(),
      reset: jest.fn(),
      token: null,
      isLoading: false,
      error: null,
    });

    // Act
    render(<AppointmentsPage />);

    // Assert
    expect(mockPush).toHaveBeenCalledWith('/auth/login');
  });

  it('should filter appointments by status', () => {
    // Act
    render(<AppointmentsPage />);
    
    const statusFilter = screen.getByDisplayValue('All Status');
    fireEvent.change(statusFilter, { target: { value: 'scheduled' } });

    // Assert
    expect(screen.getByText('Oil Change')).toBeInTheDocument();
  });

  it('should navigate to new appointment page', () => {
    // Act
    render(<AppointmentsPage />);
    
    const scheduleButton = screen.getByText('➕ Schedule Service');
    fireEvent.click(scheduleButton);

    // Assert
    expect(mockPush).toHaveBeenCalledWith('/appointments/new');
  });

  it('should show empty state when no appointments', () => {
    // Arrange
    mockedUseAppointments.mockReturnValue({
      appointments: [],
      isLoading: false,
      error: null,
      fetchAppointments: mockFetchAppointments,
      cancelAppointment: mockCancelAppointment,
      confirmAppointment: jest.fn(),
      completeAppointment: jest.fn(),
      createAppointment: jest.fn(),
      updateAppointment: jest.fn(),
      deleteAppointment: jest.fn(),
      selectAppointment: jest.fn(),
      setFilters: jest.fn(),
      clearError: jest.fn(),
      reset: jest.fn(),
      selectedAppointment: null,
      filters: { status: 'all' },
      isCreating: false,
      isUpdating: false,
      isDeleting: false,
    });

    // Act
    render(<AppointmentsPage />);

    // Assert
    expect(screen.getByText('No appointments found')).toBeInTheDocument();
    expect(screen.getByText('Schedule your first service appointment')).toBeInTheDocument();
  });

  it('should handle cancel appointment', async () => {
    // Arrange
    mockCancelAppointment.mockResolvedValueOnce(true);

    // Act
    render(<AppointmentsPage />);
    
    const cancelButton = screen.getByText('Cancel');
    fireEvent.click(cancelButton);

    // Assert
    await waitFor(() => {
      expect(mockCancelAppointment).toHaveBeenCalledWith('1');
    });
  });

  it('should show loading state', () => {
    // Arrange
    mockedUseAppointments.mockReturnValue({
      appointments: [],
      isLoading: true,
      error: null,
      fetchAppointments: mockFetchAppointments,
      cancelAppointment: mockCancelAppointment,
      confirmAppointment: jest.fn(),
      completeAppointment: jest.fn(),
      createAppointment: jest.fn(),
      updateAppointment: jest.fn(),
      deleteAppointment: jest.fn(),
      selectAppointment: jest.fn(),
      setFilters: jest.fn(),
      clearError: jest.fn(),
      reset: jest.fn(),
      selectedAppointment: null,
      filters: { status: 'all' },
      isCreating: false,
      isUpdating: false,
      isDeleting: false,
    });

    // Act
    render(<AppointmentsPage />);

    // Assert
    expect(screen.getByText('Loading appointments...')).toBeInTheDocument();
  });
});