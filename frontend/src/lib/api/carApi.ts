import { Car, CreateCarRequest, UpdateCarRequest } from '@/types/car';
import { getPublicApiOrigin } from '@/lib/api-public-origin';

export interface ApiResponse<T> {
  data: T | null;
  error: {
    message: string;
    status?: number;
  } | null;
}

class CarApiClient {
  private baseUrl: string;

  constructor(baseUrl?: string) {
    const origin = (baseUrl?.trim() || getPublicApiOrigin()).replace(/\/+$/, '').replace(/\/api\/v1$/, '');
    this.baseUrl = `${origin}/api/v1`;
  }

  // ✅ Fixed: Safe token retrieval that works in both client and server
  private getAuthToken(): string | null {
    // Check if we're in the browser environment
    if (typeof window !== 'undefined') {
      // Try multiple storage locations following Agent.md security practices
      return localStorage.getItem('authToken') || 
             localStorage.getItem('token') || 
             sessionStorage.getItem('authToken') ||
             sessionStorage.getItem('token');
    }
    return null;
  }

  // ✅ Enhanced request method with better error handling
  private async request<T>(
    endpoint: string,
    options: RequestInit = {}
  ): Promise<ApiResponse<T>> {
    try {
      const token = this.getAuthToken();
      const url = `${this.baseUrl}${endpoint}`;
      
      // ✅ Better logging for debugging (following Agent.md logging practices)
      console.log('API Request:', {
        url,
        method: options.method || 'GET',
        hasToken: !!token,
        tokenPreview: token ? `${token.substring(0, 10)}...` : 'null'
      });

      const response = await fetch(url, {
        ...options,
        headers: {
          'Content-Type': 'application/json',
          // ✅ Fixed: Only add Authorization header if token exists
          ...(token && { Authorization: `Bearer ${token}` }),
          ...options.headers,
        },
      });

      // ✅ Enhanced error handling for different HTTP status codes
      if (!response.ok) {
        let errorMessage = `HTTP ${response.status}: ${response.statusText}`;
        
        try {
          const errorText = await response.text();
          if (errorText) {
            errorMessage = errorText;
          }
        } catch (e) {
          // If response body can't be read, use default message
        }

        // ✅ Special handling for authentication errors
        if (response.status === 401) {
          // Clear invalid token
          if (typeof window !== 'undefined') {
            localStorage.removeItem('authToken');
            localStorage.removeItem('token');
            sessionStorage.removeItem('authToken');
            sessionStorage.removeItem('token');
          }
          errorMessage = 'Authentication required. Please log in again.';
        }

        return {
          data: null,
          error: {
            message: errorMessage,
            status: response.status
          }
        };
      }

      // ✅ Handle empty responses (like DELETE operations)
      const contentType = response.headers.get('content-type');
      let data = null;
      
      if (contentType && contentType.includes('application/json')) {
        data = await response.json();
      } else if (response.status === 204) {
        // No content for successful DELETE
        data = { message: 'Operation completed successfully' } as T;
      }

      return { data, error: null };
    } catch (error) {
      console.error('API Request failed:', error);
      
      return {
        data: null,
        error: {
          message: error instanceof Error ? error.message : 'Network error occurred'
        }
      };
    }
  }

  // ✅ Method to set token (for integration with auth context)
  setAuthToken(token: string): void {
    if (typeof window !== 'undefined') {
      localStorage.setItem('authToken', token);
    }
  }

  // ✅ Method to clear token
  clearAuthToken(): void {
    if (typeof window !== 'undefined') {
      localStorage.removeItem('authToken');
      localStorage.removeItem('token');
      sessionStorage.removeItem('authToken');
      sessionStorage.removeItem('token');
    }
  }

  // Create car - following Agent.md error handling
  async createCar(carData: CreateCarRequest): Promise<ApiResponse<Car>> {
    return this.request<Car>('/cars', {
      method: 'POST',
      body: JSON.stringify(carData),
    });
  }

  // Read cars - following Agent.md naming conventions
  async getCars(): Promise<ApiResponse<Car[]>> {
    return this.request<Car[]>('/cars');
  }

  // Read single car
  async getCar(id: string): Promise<ApiResponse<Car>> {
    return this.request<Car>(`/cars/${id}`);
  }

  // Update car - following Agent.md clean architecture
  async updateCar(id: string, carData: Partial<CreateCarRequest>): Promise<ApiResponse<Car>> {
    return this.request<Car>(`/cars/${id}`, {
      method: 'PUT',
      body: JSON.stringify(carData),
    });
  }

  // Delete car - following Agent.md proper error handling
  async deleteCar(id: string): Promise<ApiResponse<{ message: string }>> {
    return this.request<{ message: string }>(`/cars/${id}`, {
      method: 'DELETE',
    });
  }

  // Car + repairs: Gin expone GET /repairs/car/:carId (no hay GET /cars/:id/repairs).
  async getCarWithRepairs(id: string): Promise<ApiResponse<Car>> {
    const car = await this.getCar(id);
    if (car.error || !car.data) {
      return car;
    }
    const repairs = await this.request<Car['repairs']>(`/repairs/car/${id}`);
    if (repairs.error) {
      return { data: { ...car.data, repairs: [] }, error: null };
    }
    return { data: { ...car.data, repairs: repairs.data ?? [] }, error: null };
  }
}

export const carApi = new CarApiClient();