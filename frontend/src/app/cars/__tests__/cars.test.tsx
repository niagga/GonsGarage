import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { useRouter } from 'next/navigation';
import { useAuth } from '@/stores';
import CarsPage from '../page';
import { apiClient } from '@/lib/api';

// Mock dependencies (next/navigation mocked globally in __tests__/setup.ts)
jest.mock('@/stores');
jest.mock('@/lib/api');

const mockRouter = {
  push: jest.fn(),
};

const mockUser = {
  id: '1',
  firstName: 'John',
  lastName: 'Doe',
  email: 'john@example.com',
  role: 'client'
};

const mockCars = [
  {
    id: '1',
    make: 'Toyota',
    model: 'Camry',
    year: 2023,
    licensePlate: 'ABC-123',
    color: 'Blue',
    vin: '1234567890',
    ownerId: '1',
    createdAt: '2024-01-01T00:00:00Z',
    updatedAt: '2024-01-01T00:00:00Z'
  }
];

describe('CarsPage', () => {
  beforeEach(() => {
    (useRouter as jest.Mock).mockReturnValue(mockRouter);
    (useAuth as jest.Mock).mockReturnValue({
      user: mockUser,
      logout: jest.fn()
    });
    (apiClient.getCars as jest.Mock).mockResolvedValue({
      data: mockCars,
      error: null
    });
  });

  afterEach(() => {
    jest.clearAllMocks();
  });

  describe('Read Cars', () => {
    it('should display list of cars', async () => {
      // Arrange & Act
      render(<CarsPage />);

      // Assert
      await waitFor(() => {
        expect(screen.getByText('2023 Toyota Camry')).toBeInTheDocument();
        expect(screen.getByText('ABC-123')).toBeInTheDocument();
      });
    });

    it('should display empty state when no cars exist', async () => {
      // Arrange
      (apiClient.getCars as jest.Mock).mockResolvedValue({
        data: [],
        error: null
      });

      // Act
      render(<CarsPage />);

      // Assert
      await waitFor(() => {
        expect(screen.getByText('No cars registered yet')).toBeInTheDocument();
        expect(screen.getByText('Add Your First Car')).toBeInTheDocument();
      });
    });
  });

  describe('Create Car', () => {
    it('should open create modal when add button clicked', async () => {
      // Arrange
      render(<CarsPage />);
      await waitFor(() => screen.getByText('Add Car'));

      // Act
      fireEvent.click(screen.getByText('Add Car'));

      // Assert
      expect(screen.getByText('Add New Car')).toBeInTheDocument();
      expect(screen.getByLabelText('Make')).toBeInTheDocument();
      expect(screen.getByLabelText('Model')).toBeInTheDocument();
    });

    it('should create car successfully', async () => {
      // Arrange
      (apiClient.createCar as jest.Mock).mockResolvedValue({
        data: { ...mockCars[0], id: '2' },
        error: null
      });

      render(<CarsPage />);
      await waitFor(() => screen.getByText('Add Car'));
      fireEvent.click(screen.getByText('Add Car'));

      // Act
      fireEvent.change(screen.getByLabelText('Make'), {
        target: { value: 'Honda' }
      });
      fireEvent.change(screen.getByLabelText('Model'), {
        target: { value: 'Civic' }
      });
      fireEvent.change(screen.getByLabelText('Year'), {
        target: { value: '2024' }
      });
      fireEvent.change(screen.getByLabelText('License Plate'), {
        target: { value: 'XYZ-789' }
      });
      fireEvent.change(screen.getByLabelText('Color'), {
        target: { value: 'Red' }
      });

      fireEvent.click(screen.getByText('Add Car'));

      // Assert
      await waitFor(() => {
        expect(apiClient.createCar).toHaveBeenCalledWith({
          make: 'Honda',
          model: 'Civic',
          year: 2024,
          licensePlate: 'XYZ-789',
          color: 'Red',
          vin: ''
        });
      });
    });

    it('should show validation errors for empty required fields', async () => {
      // Arrange
      render(<CarsPage />);
      await waitFor(() => screen.getByText('Add Car'));
      fireEvent.click(screen.getByText('Add Car'));

      // Act
      fireEvent.click(screen.getByText('Add Car'));

      // Assert
      await waitFor(() => {
        expect(screen.getByText('Make is required')).toBeInTheDocument();
        expect(screen.getByText('Model is required')).toBeInTheDocument();
        expect(screen.getByText('License plate is required')).toBeInTheDocument();
        expect(screen.getByText('Color is required')).toBeInTheDocument();
      });
    });
  });

  describe('Update Car', () => {
    it('should open edit modal when edit button clicked', async () => {
      // Arrange
      render(<CarsPage />);
      await waitFor(() => screen.getByTitle('Edit car'));

      // Act
      fireEvent.click(screen.getByTitle('Edit car'));

      // Assert
      expect(screen.getByText('Edit Car')).toBeInTheDocument();
      expect(screen.getByDisplayValue('Toyota')).toBeInTheDocument();
      expect(screen.getByDisplayValue('Camry')).toBeInTheDocument();
    });

    it('should update car successfully', async () => {
      // Arrange
      (apiClient.updateCar as jest.Mock).mockResolvedValue({
        data: { ...mockCars[0], color: 'Green' },
        error: null
      });

      render(<CarsPage />);
      await waitFor(() => screen.getByTitle('Edit car'));
      fireEvent.click(screen.getByTitle('Edit car'));

      // Act
      fireEvent.change(screen.getByLabelText('Color'), {
        target: { value: 'Green' }
      });
      fireEvent.click(screen.getByText('Update Car'));

      // Assert
      await waitFor(() => {
        expect(apiClient.updateCar).toHaveBeenCalledWith('1', {
          make: 'Toyota',
          model: 'Camry',
          year: 2023,
          licensePlate: 'ABC-123',
          color: 'Green',
          vin: '1234567890'
        });
      });
    });
  });

  describe('Delete Car', () => {
    it('should delete car when confirmed', async () => {
      // Arrange
      window.confirm = jest.fn(() => true);
      (apiClient.deleteCar as jest.Mock).mockResolvedValue({
        data: { message: 'Car deleted successfully' },
        error: null
      });

      render(<CarsPage />);
      await waitFor(() => screen.getByTitle('Delete car'));

      // Act
      fireEvent.click(screen.getByTitle('Delete car'));

      // Assert
      expect(window.confirm).toHaveBeenCalledWith('Are you sure you want to delete this car?');
      await waitFor(() => {
        expect(apiClient.deleteCar).toHaveBeenCalledWith('1');
      });
    });

    it('should not delete car when cancelled', async () => {
      // Arrange
      window.confirm = jest.fn(() => false);

      render(<CarsPage />);
      await waitFor(() => screen.getByTitle('Delete car'));

      // Act
      fireEvent.click(screen.getByTitle('Delete car'));

      // Assert
      expect(window.confirm).toHaveBeenCalled();
      expect(apiClient.deleteCar).not.toHaveBeenCalled();
    });
  });
});