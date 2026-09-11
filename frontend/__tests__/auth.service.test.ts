// ✅ Auth service tests following Agent.md patterns  
// Tests authentication service functionality with mocked API client

import { authService } from '@/lib/services/auth.service';
import { apiClient } from '@/lib/api-client';
import type { LoginRequest, RegisterRequest } from '@/types/auth';
import { UserRole } from '@/types/auth';

// ✅ Mock the API client
jest.mock('@/lib/api-client', () => ({
  apiClient: {
    post: jest.fn(),
    get: jest.fn(),
    put: jest.fn(),
    setToken: jest.fn(),
    clearToken: jest.fn(),
  },
}));

const mockApiClient = apiClient as jest.Mocked<typeof apiClient>;

describe('AuthService', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('Singleton Pattern', () => {
    it('should return the same instance', () => {
      const instance1 = authService;
      const instance2 = authService;
      
      expect(instance1).toBe(instance2);
    });
  });

  describe('Login', () => {
    const loginCredentials: LoginRequest = {
      email: 'test@example.com',
      password: 'password123'
    };

    it('should login successfully and set token', async () => {
      const mockLoginResponse = {
        success: true,
        data: {
          user: {
            id: '1',
            email: 'test@example.com',
            firstName: 'John',
            lastName: 'Doe',
            role: 'client',
            createdAt: '2025-01-01T00:00:00Z',
            updatedAt: '2025-01-01T00:00:00Z'
          },
          token: 'test-token',
          refreshToken: 'refresh-token'
        }
      };

      mockApiClient.post.mockResolvedValueOnce(mockLoginResponse);

      const result = await authService.login(loginCredentials);

      expect(mockApiClient.post).toHaveBeenCalledWith(
        '/auth/login', 
        loginCredentials, 
        { skipAuth: true }
      );
      expect(mockApiClient.setToken).toHaveBeenCalledWith('test-token');
      expect(result).toEqual(mockLoginResponse);
    });

    it('should handle login failure', async () => {
      const mockErrorResponse = {
        success: false,
        error: {
          message: 'Invalid credentials',
          status: 401,
          code: 'INVALID_CREDENTIALS'
        }
      };

      mockApiClient.post.mockResolvedValueOnce(mockErrorResponse);

      const result = await authService.login(loginCredentials);

      expect(result).toEqual(mockErrorResponse);
      expect(mockApiClient.setToken).not.toHaveBeenCalled();
    });

    it('should handle network errors during login', async () => {
      mockApiClient.post.mockRejectedValueOnce(new Error('Network error'));

      const result = await authService.login(loginCredentials);

      expect(result.success).toBe(false);
      expect(result.error?.code).toBe('LOGIN_ERROR');
      expect(mockApiClient.setToken).not.toHaveBeenCalled();
    });
  });

  describe('Register', () => {
    const registerData: RegisterRequest = {
      email: 'newuser@example.com',
      password: 'password123',
      firstName: 'Jane',
      lastName: 'Smith',
      role: UserRole.CLIENT
    };

    it('should register successfully and set token', async () => {
      const mockRegisterResponse = {
        success: true,
        data: {
          user: {
            id: '2',
            email: 'newuser@example.com',
            firstName: 'Jane',
            lastName: 'Smith',
            role: 'client',
            createdAt: '2025-01-01T00:00:00Z',
            updatedAt: '2025-01-01T00:00:00Z'
          },
          token: 'new-user-token',
          refreshToken: 'new-refresh-token'
        }
      };

      mockApiClient.post.mockResolvedValueOnce(mockRegisterResponse);

      const result = await authService.register(registerData);

      expect(mockApiClient.post).toHaveBeenCalledWith(
        '/auth/register', 
        registerData, 
        { skipAuth: true }
      );
      expect(mockApiClient.setToken).toHaveBeenCalledWith('new-user-token');
      expect(result).toEqual(mockRegisterResponse);
    });

    it('should handle registration failure', async () => {
      const mockErrorResponse = {
        success: false,
        error: {
          message: 'Email already exists',
          status: 409,
          code: 'EMAIL_EXISTS'
        }
      };

      mockApiClient.post.mockResolvedValueOnce(mockErrorResponse);

      const result = await authService.register(registerData);

      expect(result).toEqual(mockErrorResponse);
      expect(mockApiClient.setToken).not.toHaveBeenCalled();
    });

    it('should handle network errors during registration', async () => {
      mockApiClient.post.mockRejectedValueOnce(new Error('Network error'));

      const result = await authService.register(registerData);

      expect(result.success).toBe(false);
      expect(result.error?.code).toBe('REGISTER_ERROR');
    });
  });

  describe('Logout', () => {
    it('should clear token locally without calling /auth/logout', async () => {
      const result = await authService.logout();

      expect(mockApiClient.post).not.toHaveBeenCalled();
      expect(mockApiClient.clearToken).toHaveBeenCalled();
      expect(result.success).toBe(true);
      expect(result.data?.message).toBe('Logged out successfully');
    });
  });

  describe('Token Validation', () => {
    it('should validate token successfully', async () => {
      const mockUserResponse = {
        success: true,
        data: {
          id: '1',
          email: 'test@example.com',
          firstName: 'John',
          lastName: 'Doe',
          role: 'client',
          createdAt: '2025-01-01T00:00:00Z',
          updatedAt: '2025-01-01T00:00:00Z'
        }
      };

      mockApiClient.get.mockResolvedValueOnce(mockUserResponse);

      const result = await authService.validateToken();

      expect(mockApiClient.get).toHaveBeenCalledWith('/auth/me');
      expect(result).toEqual(mockUserResponse);
    });

    it('should handle token validation failure', async () => {
      const mockErrorResponse = {
        success: false,
        error: {
          message: 'Invalid token',
          status: 401,
          code: 'INVALID_TOKEN'
        }
      };

      mockApiClient.get.mockResolvedValueOnce(mockErrorResponse);

      const result = await authService.validateToken();

      expect(result).toEqual(mockErrorResponse);
    });

    it('should handle network errors during validation', async () => {
      mockApiClient.get.mockRejectedValueOnce(new Error('Network error'));

      const result = await authService.validateToken();

      expect(result.success).toBe(false);
      expect(result.error?.code).toBe('VALIDATION_ERROR');
    });
  });

  describe('Token Refresh', () => {
    it('should refresh token successfully', async () => {
      const refreshToken = 'old-refresh-token';
      const mockRefreshResponse = {
        success: true,
        data: {
          user: {
            id: '1',
            email: 'test@example.com',
            firstName: 'John',
            lastName: 'Doe',
            role: 'client',
            createdAt: '2025-01-01T00:00:00Z',
            updatedAt: '2025-01-01T00:00:00Z'
          },
          token: 'new-access-token',
          refreshToken: 'new-refresh-token'
        }
      };

      mockApiClient.post.mockResolvedValueOnce(mockRefreshResponse);

      const result = await authService.refreshToken(refreshToken);

      expect(mockApiClient.post).toHaveBeenCalledWith(
        '/auth/refresh', 
        { refreshToken }, 
        { skipAuth: true }
      );
      expect(mockApiClient.setToken).toHaveBeenCalledWith('new-access-token');
      expect(result).toEqual(mockRefreshResponse);
    });

    it('should handle refresh token failure', async () => {
      mockApiClient.post.mockRejectedValueOnce(new Error('Network error'));

      const result = await authService.refreshToken('invalid-token');

      expect(result.success).toBe(false);
      expect(result.error?.code).toBe('REFRESH_ERROR');
    });
  });

  describe('Password Management', () => {
    it('should request password reset successfully', async () => {
      const email = 'test@example.com';
      const mockResetResponse = {
        success: true,
        data: { message: 'Password reset email sent' }
      };

      mockApiClient.post.mockResolvedValueOnce(mockResetResponse);

      const result = await authService.requestPasswordReset(email);

      expect(mockApiClient.post).toHaveBeenCalledWith(
        '/auth/forgot-password', 
        { email }, 
        { skipAuth: true }
      );
      expect(result).toEqual(mockResetResponse);
    });

    it('should reset password successfully', async () => {
      const token = 'reset-token';
      const newPassword = 'newpassword123';
      const mockResetResponse = {
        success: true,
        data: { message: 'Password reset successful' }
      };

      mockApiClient.post.mockResolvedValueOnce(mockResetResponse);

      const result = await authService.resetPassword(token, newPassword);

      expect(mockApiClient.post).toHaveBeenCalledWith(
        '/auth/reset-password', 
        { token, newPassword }, 
        { skipAuth: true }
      );
      expect(result).toEqual(mockResetResponse);
    });

    it('should change password successfully', async () => {
      const currentPassword = 'oldpassword';
      const newPassword = 'newpassword123';
      const mockChangeResponse = {
        success: true,
        data: { message: 'Password changed successfully' }
      };

      mockApiClient.put.mockResolvedValueOnce(mockChangeResponse);

      const result = await authService.changePassword(currentPassword, newPassword);

      expect(mockApiClient.put).toHaveBeenCalledWith(
        '/auth/change-password', 
        { currentPassword, newPassword }
      );
      expect(result).toEqual(mockChangeResponse);
    });
  });
});