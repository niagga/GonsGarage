import { act, renderHook } from '@testing-library/react';
import { useCarStore } from '@/stores/car.store';
import { carService } from '@/lib/services/car.service';

jest.mock('@/lib/services/car.service', () => ({
  carService: {
    getCars: jest.fn(),
    getCar: jest.fn(),
    createCar: jest.fn(),
    updateCar: jest.fn(),
    deleteCar: jest.fn(),
  },
}));

const mockedCarService = carService as jest.Mocked<typeof carService>;

describe('useCarStore', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    useCarStore.setState({
      cars: [],
      selectedCar: null,
      isLoading: false,
      isCreating: false,
      isUpdating: false,
      isDeleting: false,
      error: null,
      totalCars: 0,
      currentPage: 1,
      pageSize: 10,
      filters: {},
    });
  });

  it('fetchCars maps API cars into store state', async () => {
    mockedCarService.getCars.mockResolvedValueOnce({
      success: true,
      data: {
        cars: [
          {
            id: '1',
            make: 'Toyota',
            model: 'Corolla',
            year: 2020,
            licensePlate: 'ABC-1',
            vin: 'VIN',
            color: 'Blue',
            mileage: 1000,
            ownerId: 'o1',
            createdAt: '2025-01-01T00:00:00Z',
            updatedAt: '2025-01-01T00:00:00Z',
          },
        ],
        total: 1,
        limit: 10,
        offset: 0,
      },
    });

    const { result } = renderHook(() => useCarStore());

    await act(async () => {
      await result.current.fetchCars();
    });

    expect(mockedCarService.getCars).toHaveBeenCalled();
    expect(result.current.cars).toHaveLength(1);
    expect(result.current.cars[0].licensePlate).toBe('ABC-1');
    expect(result.current.totalCars).toBe(1);
    expect(result.current.isLoading).toBe(false);
  });

  it('fetchCars sets error when service returns failure', async () => {
    mockedCarService.getCars.mockResolvedValueOnce({
      success: false,
      error: { message: 'unauthorized', status: 401, code: 'x' },
    });

    const { result } = renderHook(() => useCarStore());

    await act(async () => {
      await result.current.fetchCars();
    });

    expect(result.current.cars).toEqual([]);
    expect(result.current.error).toBe('unauthorized');
  });
});
