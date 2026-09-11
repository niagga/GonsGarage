// AuthStore tests following Agent.md TDD patterns

import { renderHook, act } from '@testing-library/react';
import { useAuthStore, useAuth } from '../src/stores/auth.store';
import { UserRole } from '../src/types';

// Mock fetch globally
const mockFetch = jest.fn();
global.fetch = mockFetch;

// Mock localStorage
const localStorageMock = {
  getItem: jest.fn(),
  setItem: jest.fn(),
  removeItem: jest.fn(),
  clear: jest.fn(),
};
Object.defineProperty(window, 'localStorage', {
  value: localStorageMock,
});

// Mock window.location  
const mockLocation = { href: '' };
delete (window as unknown as { location: unknown }).location;
(window as unknown as { location: typeof mockLocation }).location = mockLocation;

describe('AuthStore', () => {
  beforeEach(() => {
    // Reset all mocks
    mockFetch.mockClear();
    localStorageMock.getItem.mockClear();
    localStorageMock.setItem.mockClear();
    localStorageMock.removeItem.mockClear();
    mockLocation.href = '';

    // Reset store state
    useAuthStore.setState({
      user: null,
      token: null,
      isLoading: false,
      error: null,
      isAuthenticated: false,
    });
  });

  describe('Initial State', () => {
    it('should have correct initial state', () => {
      const { result } = renderHook(() => useAuth());

      expect(result.current.user).toBeNull();
      expect(result.current.token).toBeNull();
      expect(result.current.isLoading).toBe(false);
      expect(result.current.error).toBeNull();
      expect(result.current.isAuthenticated).toBe(false);
    });
  });

  describe('Login', () => {
    it('should login successfully', async () => {
      mockFetch
        .mockResolvedValueOnce({
          ok: true,
          json: () => Promise.resolve({ token: 'test-token' }),
        })
        .mockResolvedValueOnce({
          ok: true,
          json: () =>
            Promise.resolve({
              user: {
                id: '1',
                email: 'test@example.com',
                firstName: 'John',
                lastName: 'Doe',
                role: UserRole.CLIENT,
                createdAt: '2025-01-01T00:00:00Z',
                updatedAt: '2025-01-01T00:00:00Z',
              },
            }),
        });

      const { result } = renderHook(() => useAuth());

      let loginResult;
      await act(async () => {
        loginResult = await result.current.login('test@example.com', 'password123');
      });

      expect(loginResult).toEqual({ success: true });
      expect(result.current.isAuthenticated).toBe(true);
      expect(result.current.user?.email).toBe('test@example.com');
      expect(result.current.token).toBe('test-token');
      expect(result.current.error).toBeNull();
      
      // Check localStorage was called
      expect(localStorageMock.setItem).toHaveBeenCalledWith('auth_token', 'test-token');
      expect(localStorageMock.setItem).toHaveBeenCalledWith('auth_user', expect.any(String));
    });

    it('should handle login failure', async () => {
      // Mock failed login response
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 401,
        text: () => Promise.resolve('{"error":"Invalid credentials"}'),
      });

      const { result } = renderHook(() => useAuth());

      let loginResult;
      await act(async () => {
        loginResult = await result.current.login('test@example.com', 'wrongpassword');
      });

      expect(loginResult).toEqual({ success: false, error: 'Invalid credentials' });
      expect(result.current.isAuthenticated).toBe(false);
      expect(result.current.user).toBeNull();
      expect(result.current.token).toBeNull();
      expect(result.current.error).toBe('Invalid credentials');
    });

    it('should handle network error during login', async () => {
      // Mock network error
      mockFetch.mockRejectedValueOnce(new Error('Network error'));

      const { result } = renderHook(() => useAuth());

      let loginResult;
      await act(async () => {
        loginResult = await result.current.login('test@example.com', 'password123');
      });

      expect(loginResult).toEqual({ success: false, error: 'Network error' });
      expect(result.current.error).toBe('Network error');
    });
  });

  describe('Register', () => {
    it('should register successfully', async () => {
      // Mock successful registration response
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({ success: true }),
      });

      const { result } = renderHook(() => useAuth());

      let registerResult;
      await act(async () => {
        registerResult = await result.current.register({
          email: 'newuser@example.com',
          password: 'password123',
          firstName: 'Jane',
          lastName: 'Smith',
          role: UserRole.CLIENT,
        });
      });

      expect(registerResult).toEqual({ success: true });
      expect(result.current.error).toBeNull();
    });

    it('should handle registration failure', async () => {
      // Mock failed registration response
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 400,
        text: () => Promise.resolve('{"error":"Email already exists"}'),
      });

      const { result } = renderHook(() => useAuth());

      let registerResult;
      await act(async () => {
        registerResult = await result.current.register({
          email: 'existing@example.com',
          password: 'password123',
          firstName: 'Jane',
          lastName: 'Smith',
          role: UserRole.CLIENT,
        });
      });

      expect(registerResult).toEqual({ success: false, error: 'Email already exists' });
      expect(result.current.error).toBe('Email already exists');
    });
  });

  describe('Logout', () => {
    it('should logout successfully', async () => {
      // Set up logged-in state
      await act(async () => {
        useAuthStore.setState({
          user: {
            id: '1',
            email: 'test@example.com',
            firstName: 'John',
            lastName: 'Doe',
            role: UserRole.CLIENT,
            createdAt: '2025-01-01T00:00:00Z',
            updatedAt: '2025-01-01T00:00:00Z',
          },
          token: 'test-token',
          isAuthenticated: true,
          isLoading: false,
          error: null,
        });
      });

      const { result } = renderHook(() => useAuth());

      // Verify logged-in state
      expect(result.current.isAuthenticated).toBe(true);

      // Logout
      await act(async () => {
        result.current.logout();
      });

      expect(result.current.isAuthenticated).toBe(false);
      expect(result.current.user).toBeNull();
      expect(result.current.token).toBeNull();
      expect(result.current.error).toBeNull();

      // Check localStorage was cleared
      expect(localStorageMock.removeItem).toHaveBeenCalledWith('auth_token');
      expect(localStorageMock.removeItem).toHaveBeenCalledWith('auth_user');
    });
  });

  describe('Error Management', () => {
    it('should clear error', async () => {
      // Set error state
      await act(async () => {
        useAuthStore.setState({ error: 'Some error' });
      });

      const { result } = renderHook(() => useAuth());

      expect(result.current.error).toBe('Some error');

      // Clear error
      await act(async () => {
        result.current.clearError();
      });

      expect(result.current.error).toBeNull();
    });
  });

  describe('Auth Status Check', () => {
    it('should restore auth state from localStorage via auth/me', async () => {
      const mockUser = {
        id: '1',
        email: 'test@example.com',
        firstName: 'John',
        lastName: 'Doe',
        role: UserRole.CLIENT,
        createdAt: '2025-01-01T00:00:00Z',
        updatedAt: '2025-01-01T00:00:00Z',
      };

      localStorageMock.getItem.mockImplementation((key) => {
        if (key === 'auth_token') return 'stored-token';
        if (key === 'auth_user') return null;
        return null;
      });

      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({ user: mockUser }),
      });

      const { result } = renderHook(() => useAuthStore());

      await act(async () => {
        await result.current.checkAuthStatus();
      });

      expect(result.current.isAuthenticated).toBe(true);
      expect(result.current.user).toEqual(mockUser);
      expect(result.current.token).toBe('stored-token');
    });

    it('should clear session when auth/me rejects the token', async () => {
      localStorageMock.getItem.mockImplementation((key) => {
        if (key === 'auth_token') return 'stored-token';
        if (key === 'auth_user') return 'invalid-json';
        return null;
      });

      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 401,
        json: () => Promise.resolve({}),
      });

      const { result } = renderHook(() => useAuthStore());

      await act(async () => {
        await result.current.checkAuthStatus();
      });

      expect(result.current.isAuthenticated).toBe(false);
      expect(result.current.user).toBeNull();
      expect(result.current.token).toBeNull();
    });
  });

  describe('Convenience Hooks', () => {
    it('should provide user through useUser hook', async () => {
      const mockUser = {
        id: '1',
        email: 'test@example.com',
        firstName: 'John',
        lastName: 'Doe',
        role: UserRole.CLIENT,
        createdAt: '2025-01-01T00:00:00Z',
        updatedAt: '2025-01-01T00:00:00Z',
      };

      await act(async () => {
        useAuthStore.setState({ user: mockUser });
      });

      const { result } = renderHook(() => useAuthStore((state) => state.user));

      expect(result.current).toEqual(mockUser);
    });
  });
});