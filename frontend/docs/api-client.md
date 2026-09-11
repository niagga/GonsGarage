# API Client Documentation

## Overview

The centralized API client provides a unified interface for all HTTP communications in the GonsGarage frontend application. It follows Agent.md standards with TypeScript strong typing, comprehensive error handling, and interceptor support.

## Basic Usage

### Simple Requests

```typescript
import { apiClient } from '@/lib/api-client';

// GET request
const response = await apiClient.get<User[]>('/users');
if (response.success) {
  console.log('Users:', response.data);
} else {
  console.error('Error:', response.error?.message);
}

// POST request
const newUser = await apiClient.post<User>('/users', {
  name: 'John Doe',
  email: 'john@example.com'
});

// PUT request
const updated = await apiClient.put<User>('/users/1', {
  name: 'John Smith'
});

// DELETE request
const deleted = await apiClient.delete('/users/1');
```

### Authentication

```typescript
import { apiClient } from '@/lib/api-client';

// Set token (done automatically by authService)
apiClient.setToken('your-jwt-token');

// Clear token on logout
apiClient.clearToken();

// Skip authentication for public endpoints
const publicData = await apiClient.get('/public/status', {
  skipAuth: true
});
```

### Using Services

#### Auth Service

```typescript
import { authService } from '@/lib/services';

// Login
const loginResult = await authService.login({
  email: 'user@example.com',
  password: 'password123'
});

if (loginResult.success) {
  console.log('User:', loginResult.data?.user);
  console.log('Token:', loginResult.data?.token);
}

// Register
const registerResult = await authService.register({
  email: 'newuser@example.com',
  password: 'password123',
  firstName: 'Jane',
  lastName: 'Doe',
  role: UserRole.CLIENT
});

// Logout
await authService.logout();

// Validate current token
const userInfo = await authService.validateToken();
```

#### Vehicle Service

```typescript
import { vehicleService } from '@/lib/services';

// Get all vehicles with filters
const vehicles = await vehicleService.getVehicles({
  ownerId: 'user-123',
  make: 'Toyota',
  limit: 10
});

// Get single vehicle
const vehicle = await vehicleService.getVehicle('vehicle-id');

// Create vehicle
const newVehicle = await vehicleService.createVehicle({
  make: 'Honda',
  model: 'Civic',
  year: 2023,
  licensePlate: 'ABC-123',
  vin: '1HGBH41JXMN109186'
});

// Update vehicle
const updated = await vehicleService.updateVehicle('vehicle-id', {
  mileage: 15000
});

// Delete vehicle
await vehicleService.deleteVehicle('vehicle-id');
```

## Advanced Features

### Request Interceptors

```typescript
import { apiClient } from '@/lib/api-client';

// Add custom headers to all requests
apiClient.addRequestInterceptor({
  onRequest: (config) => ({
    ...config,
    headers: {
      ...config.headers,
      'X-Client-Version': '1.0.0',
      'X-Request-ID': generateRequestId()
    }
  })
});

// Handle request errors
apiClient.addRequestInterceptor({
  onRequestError: (error) => {
    console.error('Request setup failed:', error);
    return error;
  }
});
```

### Response Interceptors

```typescript
// Transform all responses
apiClient.addResponseInterceptor({
  onResponse: (response) => {
    // Log all successful responses
    if (response.success) {
      console.log('API Success:', response);
    }
    return response;
  },
  onResponseError: (error) => {
    // Global error handling
    if (error.status === 401) {
      // Redirect to login
      window.location.href = '/auth/login';
    }
    return error;
  }
});
```

### Configuration Options

```typescript
// Request with custom configuration
const response = await apiClient.get('/slow-endpoint', {
  timeout: 60000, // 60 seconds
  retry: 3,       // Retry 3 times on failure
  retryDelay: 2000, // 2 seconds between retries
  baseURL: 'https://different-api.com' // Override base URL
});
```

### Error Handling

```typescript
import { HTTP_STATUS } from '@/lib/api-client';

const response = await apiClient.post('/users', userData);

if (!response.success) {
  const error = response.error;
  
  switch (error?.status) {
    case HTTP_STATUS.BAD_REQUEST:
      console.error('Validation error:', error.message);
      break;
    case HTTP_STATUS.UNAUTHORIZED:
      console.error('Authentication required');
      break;
    case HTTP_STATUS.FORBIDDEN:
      console.error('Access denied');
      break;
    case HTTP_STATUS.NOT_FOUND:
      console.error('Resource not found');
      break;
    default:
      console.error('Unexpected error:', error?.message);
  }
}
```

## Integration with Zustand Stores

### Using in Auth Store

```typescript
import { create } from 'zustand';
import { authService } from '@/lib/services';

interface AuthStore {
  user: User | null;
  login: (credentials: LoginRequest) => Promise<{ success: boolean; error?: string }>;
}

export const useAuthStore = create<AuthStore>((set) => ({
  user: null,
  
  login: async (credentials) => {
    const result = await authService.login(credentials);
    
    if (result.success) {
      set({ user: result.data?.user });
      return { success: true };
    } else {
      return { 
        success: false, 
        error: result.error?.message || 'Login failed' 
      };
    }
  }
}));
```

### Custom Service Example

```typescript
// services/appointment.service.ts
import { apiClient, type ApiResponse } from '@/lib/api-client';

export interface Appointment {
  id: string;
  clientId: string;
  vehicleId: string;
  serviceType: string;
  scheduledAt: string;
  status: 'pending' | 'confirmed' | 'completed' | 'cancelled';
  createdAt: string;
  updatedAt: string;
}

export class AppointmentService {
  async getAppointments(): Promise<ApiResponse<Appointment[]>> {
    return apiClient.get<Appointment[]>('/appointments');
  }

  async createAppointment(data: CreateAppointmentRequest): Promise<ApiResponse<Appointment>> {
    return apiClient.post<Appointment>('/appointments', data);
  }

  async updateAppointment(id: string, updates: Partial<Appointment>): Promise<ApiResponse<Appointment>> {
    return apiClient.put<Appointment>(`/appointments/${id}`, updates);
  }
}

export const appointmentService = new AppointmentService();
```

## Testing

### Mocking API Client

```typescript
// __tests__/services/auth.service.test.ts
import { apiClient } from '@/lib/api-client';
import { authService } from '@/lib/services';

jest.mock('@/lib/api-client');
const mockApiClient = apiClient as jest.Mocked<typeof apiClient>;

describe('AuthService', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('should login successfully', async () => {
    mockApiClient.post.mockResolvedValue({
      success: true,
      data: {
        user: mockUser,
        token: 'mock-token'
      }
    });

    const result = await authService.login(credentials);
    
    expect(result.success).toBe(true);
    expect(mockApiClient.setToken).toHaveBeenCalledWith('mock-token');
  });
});
```

## Migration from Legacy API

### Before (Legacy)
```typescript
// Old way - scattered fetch calls
const response = await fetch('/api/users', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
    'Authorization': `Bearer ${token}`
  },
  body: JSON.stringify(userData)
});

if (!response.ok) {
  throw new Error('Request failed');
}

const data = await response.json();
```

### After (New API Client)
```typescript
// New way - centralized client
const response = await apiClient.post<User>('/users', userData);

if (response.success) {
  console.log('User created:', response.data);
} else {
  console.error('Error:', response.error?.message);
}
```

## Best Practices

1. **Always check response.success** before accessing data
2. **Use TypeScript generics** to type response data
3. **Handle errors gracefully** with proper user feedback
4. **Use services** instead of direct API client calls in components
5. **Mock services in tests**, not the API client directly
6. **Configure interceptors** for cross-cutting concerns (logging, auth)
7. **Use proper HTTP status codes** for different error scenarios

## Performance Considerations

- **Request deduplication**: Implement at the service level for repeated requests
- **Caching**: Use Zustand stores or React Query for data caching
- **Retry logic**: Built-in exponential backoff for failed requests
- **Timeout configuration**: Adjust based on endpoint complexity
- **Bundle size**: Tree-shaking friendly exports minimize bundle impact

## Security

- **Token management**: Automatic token injection and cleanup
- **HTTPS only**: Ensure all production endpoints use HTTPS
- **Sanitization**: API client automatically handles JSON serialization
- **Error exposure**: Error messages are filtered to prevent information leakage