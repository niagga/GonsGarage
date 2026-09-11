// ✅ Authentication service using centralized API client
// Provides auth-specific methods with proper error handling

import { apiClient, type ApiResponse } from '../api-client';
import type { 
  User, 
  LoginRequest, 
  LoginResponse, 
  RegisterRequest 
} from '@/types/auth';

// ✅ Authentication endpoints following Agent.md conventions
export class AuthService {
  private static instance: AuthService;

  // ✅ Singleton pattern for service consistency
  static getInstance(): AuthService {
    if (!AuthService.instance) {
      AuthService.instance = new AuthService();
    }
    return AuthService.instance;
  }

  private constructor() {
    // Private constructor for singleton
  }

  // ✅ Login method with enhanced error handling
  async login(credentials: LoginRequest): Promise<ApiResponse<LoginResponse>> {
    try {
      const response = await apiClient.post<LoginResponse>('/auth/login', credentials, {
        skipAuth: true, // Login doesn't require existing auth
      });

      // ✅ Set token in API client if login successful
      if (response.success && response.data?.token) {
        apiClient.setToken(response.data.token);
      }

      return response;
    } catch {
      return {
        success: false,
        error: {
          message: 'Login request failed',
          status: 0,
          code: 'LOGIN_ERROR'
        }
      };
    }
  }

  // ✅ Register method following Agent.md patterns
  async register(userData: RegisterRequest): Promise<ApiResponse<LoginResponse>> {
    try {
      const response = await apiClient.post<LoginResponse>('/auth/register', userData, {
        skipAuth: true, // Registration doesn't require existing auth
      });

      // ✅ Set token in API client if registration successful
      if (response.success && response.data?.token) {
        apiClient.setToken(response.data.token);
      }

      return response;
    } catch {
      return {
        success: false,
        error: {
          message: 'Registration request failed',
          status: 0,
          code: 'REGISTER_ERROR'
        }
      };
    }
  }

  // ✅ Logout method with token cleanup (no POST /auth/logout en el backend Gin actual)
  async logout(): Promise<ApiResponse<{ message: string }>> {
    apiClient.clearToken();
    return {
      success: true,
      data: { message: 'Logged out successfully' },
    };
  }

  // ✅ Refresh token method for token renewal
  async refreshToken(refreshToken: string): Promise<ApiResponse<LoginResponse>> {
    try {
      const response = await apiClient.post<LoginResponse>('/auth/refresh', {
        refreshToken
      }, {
        skipAuth: true, // Refresh uses refresh token, not access token
      });

      // ✅ Update token in API client if refresh successful
      if (response.success && response.data?.token) {
        apiClient.setToken(response.data.token);
      }

      return response;
    } catch {
      return {
        success: false,
        error: {
          message: 'Token refresh failed',
          status: 0,
          code: 'REFRESH_ERROR'
        }
      };
    }
  }

  // ✅ Validate token method for auth status checking
  async validateToken(): Promise<ApiResponse<User>> {
    try {
      const response = await apiClient.get<User>('/auth/me');
      return response;
    } catch {
      return {
        success: false,
        error: {
          message: 'Token validation failed',
          status: 0,
          code: 'VALIDATION_ERROR'
        }
      };
    }
  }

  // ✅ Password reset request
  async requestPasswordReset(email: string): Promise<ApiResponse<{ message: string }>> {
    try {
      const response = await apiClient.post<{ message: string }>('/auth/forgot-password', {
        email
      }, {
        skipAuth: true,
      });

      return response;
    } catch {
      return {
        success: false,
        error: {
          message: 'Password reset request failed',
          status: 0,
          code: 'RESET_REQUEST_ERROR'
        }
      };
    }
  }

  // ✅ Password reset confirmation
  async resetPassword(
    token: string, 
    newPassword: string
  ): Promise<ApiResponse<{ message: string }>> {
    try {
      const response = await apiClient.post<{ message: string }>('/auth/reset-password', {
        token,
        newPassword
      }, {
        skipAuth: true,
      });

      return response;
    } catch {
      return {
        success: false,
        error: {
          message: 'Password reset failed',
          status: 0,
          code: 'RESET_ERROR'
        }
      };
    }
  }

  // ✅ Change password for authenticated users
  async changePassword(
    currentPassword: string, 
    newPassword: string
  ): Promise<ApiResponse<{ message: string }>> {
    try {
      const response = await apiClient.put<{ message: string }>('/auth/change-password', {
        currentPassword,
        newPassword
      });

      return response;
    } catch {
      return {
        success: false,
        error: {
          message: 'Password change failed',
          status: 0,
          code: 'CHANGE_PASSWORD_ERROR'
        }
      };
    }
  }
}

// ✅ Export singleton instance for application use
export const authService = AuthService.getInstance();

// ✅ Default export for convenience
export default authService;