// ✅ Car service using centralized API client
// Provides car management methods following Agent.md patterns

import { apiClient, type ApiResponse } from '../api-client';

// ✅ Car interfaces following Agent.md camelCase conventions
export interface Car {
  id: string;
  make: string;
  model: string;
  year: number;
  licensePlate: string;
  vin?: string;
  color?: string;
  mileage?: number;
  ownerId: string;
  createdAt: string;
  updatedAt: string;
}

export interface CreateCarRequest {
  make: string;
  model: string;
  year: number;
  licensePlate: string;
  vin?: string;
  color?: string;
  mileage?: number;
  ownerID?: string;
}

export interface UpdateCarRequest extends Partial<CreateCarRequest> {
  id: string;
}

export interface CarFilters {
  ownerId?: string;
  make?: string;
  model?: string;
  year?: number;
  limit?: number;
  offset?: number;
}

export interface CarListResponse {
  cars: Car[];
  total: number;
  limit: number;
  offset: number;
}

// ✅ Car service class following Agent.md clean architecture
export class CarService {
  private static instance: CarService;

  // ✅ Singleton pattern for service consistency
  static getInstance(): CarService {
    if (!CarService.instance) {
      CarService.instance = new CarService();
    }
    return CarService.instance;
  }

  private constructor() {
    // Private constructor for singleton
  }

  // ✅ Get all cars with optional filters
  async getCars(filters: CarFilters = {}): Promise<ApiResponse<CarListResponse>> {
    try {
      const queryParams = new URLSearchParams();
      
      Object.entries(filters).forEach(([key, value]) => {
        if (value !== undefined && value !== null) {
          queryParams.append(key, String(value));
        }
      });

      const queryString = queryParams.toString();
      const endpoint = `/cars${queryString ? `?${queryString}` : ''}`;

      const response = await apiClient.get<CarListResponse>(endpoint); // ✅ Use 'any' to handle both formats
      
      if (response.success && response.data) {
        // ✅ Handle both response formats
        let carListResponse: CarListResponse;
        
        if (Array.isArray(response.data)) {
          // ✅ Backend returns direct array
          carListResponse = {
            cars: response.data,
            total: response.data.length,
            limit: filters.limit || 10,
            offset: filters.offset || 0,
          };
        } else if (response.data.cars && Array.isArray(response.data.cars)) {
          // ✅ Backend returns wrapped object
          carListResponse = {
            cars: response.data.cars,
            total: response.data.total || response.data.cars.length,
            limit: response.data.limit || filters.limit || 10,
            offset: response.data.offset || filters.offset || 0,
          };
        } else {
          // ✅ Handle empty/invalid response
          carListResponse = {
            cars: [],
            total: 0,
            limit: filters.limit || 10,
            offset: filters.offset || 0,
          };
        }
        
        console.log('✅ Cars fetched successfully:', carListResponse);
        
        return {
          success: true,
          data: carListResponse
        };
      } else {
        console.error('❌ Cars API Error:', response.error);
        return response;
      }
    } catch (error) {
      console.error('❌ Cars Service Error:', error);
      return {
        success: false,
        error: {
          message: 'Failed to fetch cars',
          status: 0,
          code: 'FETCH_CARS_ERROR'
        }
      };
    }
  }

  // ✅ Get single car by ID
  async getCar(id: string): Promise<ApiResponse<Car>> {
    try {
      return await apiClient.get<Car>(`/cars/${id}`);
    } catch {
      return {
        success: false,
        error: {
          message: 'Failed to fetch car',
          status: 0,
          code: 'FETCH_CAR_ERROR'
        }
      };
    }
  }

  // ✅ Create new car
  async createCar(carData: CreateCarRequest): Promise<ApiResponse<Car>> {
    try {
      return await apiClient.post<Car>('/cars', carData);
    } catch {
      return {
        success: false,
        error: {
          message: 'Failed to create car',
          status: 0,
          code: 'CREATE_CAR_ERROR'
        }
      };
    }
  }

  // ✅ Update existing car
  async updateCar(id: string, updates: Partial<CreateCarRequest>): Promise<ApiResponse<Car>> {
    try {
      return await apiClient.put<Car>(`/cars/${id}`, updates);
    } catch {
      return {
        success: false,
        error: {
          message: 'Failed to update car',
          status: 0,
          code: 'UPDATE_CAR_ERROR'
        }
      };
    }
  }

  // ✅ Delete car
  async deleteCar(id: string): Promise<ApiResponse<{ message: string }>> {
    try {
      return await apiClient.delete<{ message: string }>(`/cars/${id}`);
    } catch {
      return {
        success: false,
        error: {
          message: 'Failed to delete car',
          status: 0,
          code: 'DELETE_CAR_ERROR'
        }
      };
    }
  }

  // ✅ Get cars by owner ID (for client dashboard)
  async getCarsByOwner(ownerId: string): Promise<ApiResponse<Car[]>> {
    try {
      const q = new URLSearchParams({ ownerId, limit: '100', offset: '0' });
      return await apiClient.get<Car[]>(`/cars?${q.toString()}`);
    } catch {
      return {
        success: false,
        error: {
          message: 'Failed to fetch owner cars',
          status: 0,
          code: 'FETCH_OWNER_CARS_ERROR'
        }
      };
    }
  }
}

// ✅ Export singleton instance for application use
export const carService = CarService.getInstance();

// ✅ Default export for convenience
export default carService;