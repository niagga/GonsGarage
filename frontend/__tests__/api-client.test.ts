// ✅ API client tests following Agent.md testing patterns
// Comprehensive test coverage for centralized HTTP client

import { ApiClient, HTTP_STATUS } from '@/lib/api-client';

// ✅ Mock fetch for testing
const mockFetch = jest.fn();
global.fetch = mockFetch;

// ✅ Mock localStorage
const localStorageMock = {
  getItem: jest.fn(),
  setItem: jest.fn(),
  removeItem: jest.fn(),
  clear: jest.fn(),
};
Object.defineProperty(window, 'localStorage', {
  value: localStorageMock,
});

describe('ApiClient', () => {
  let apiClient: ApiClient;

  beforeEach(() => {
    apiClient = new ApiClient('http://localhost:8080/api/v1');
    mockFetch.mockClear();
    localStorageMock.getItem.mockClear();
    localStorageMock.setItem.mockClear();
    localStorageMock.removeItem.mockClear();
  });

  describe('Token Management', () => {
    it('should set and store token correctly', () => {
      const token = 'test-token';
      
      apiClient.setToken(token);
      
      expect(apiClient.getToken()).toBe(token);
      expect(localStorageMock.setItem).toHaveBeenCalledWith('auth_token', token);
    });

    it('should clear token and localStorage', () => {
      apiClient.setToken('test-token');
      
      apiClient.clearToken();
      
      expect(apiClient.getToken()).toBeNull();
      expect(localStorageMock.removeItem).toHaveBeenCalledWith('auth_token');
      expect(localStorageMock.removeItem).toHaveBeenCalledWith('auth_user');
    });
  });

  describe('HTTP Methods', () => {
    beforeEach(() => {
      mockFetch.mockResolvedValue({
        ok: true,
        status: HTTP_STATUS.OK,
        headers: {
          get: jest.fn().mockReturnValue('application/json'),
        },
        json: jest.fn().mockResolvedValue({ success: true, data: { id: '1' } }),
      });
    });

    it('should make GET request correctly', async () => {
      const response = await apiClient.get('/test');

      expect(mockFetch).toHaveBeenCalledWith(
        'http://localhost:8080/api/v1/test',
        expect.objectContaining({
          method: 'GET',
          headers: expect.objectContaining({
            'Content-Type': 'application/json',
          }),
        })
      );
      expect(response.success).toBe(true);
    });

    it('should make POST request with body', async () => {
      const testData = { name: 'test' };
      
      await apiClient.post('/test', testData);

      expect(mockFetch).toHaveBeenCalledWith(
        'http://localhost:8080/api/v1/test',
        expect.objectContaining({
          method: 'POST',
          headers: expect.objectContaining({
            'Content-Type': 'application/json',
          }),
          body: JSON.stringify(testData),
        })
      );
    });

    it('should make PUT request with body', async () => {
      const testData = { name: 'updated' };
      
      await apiClient.put('/test/1', testData);

      expect(mockFetch).toHaveBeenCalledWith(
        'http://localhost:8080/api/v1/test/1',
        expect.objectContaining({
          method: 'PUT',
          body: JSON.stringify(testData),
        })
      );
    });

    it('should make DELETE request correctly', async () => {
      await apiClient.delete('/test/1');

      expect(mockFetch).toHaveBeenCalledWith(
        'http://localhost:8080/api/v1/test/1',
        expect.objectContaining({
          method: 'DELETE',
        })
      );
    });
  });

  describe('Authentication', () => {
    it('should include Authorization header when token is set', async () => {
      const token = 'test-auth-token';
      apiClient.setToken(token);

      mockFetch.mockResolvedValueOnce({
        ok: true,
        status: HTTP_STATUS.OK,
        headers: { get: jest.fn().mockReturnValue('application/json') },
        json: jest.fn().mockResolvedValue({}),
      });

      await apiClient.get('/protected');

      expect(mockFetch).toHaveBeenCalledWith(
        expect.any(String),
        expect.objectContaining({
          headers: expect.objectContaining({
            Authorization: `Bearer ${token}`,
          }),
        })
      );
    });

    it('should skip Authorization header when skipAuth is true', async () => {
      apiClient.setToken('test-token');

      mockFetch.mockResolvedValueOnce({
        ok: true,
        status: HTTP_STATUS.OK,
        headers: { get: jest.fn().mockReturnValue('application/json') },
        json: jest.fn().mockResolvedValue({}),
      });

      await apiClient.get('/public', { skipAuth: true });

      expect(mockFetch).toHaveBeenCalledWith(
        expect.any(String),
        expect.objectContaining({
          headers: expect.not.objectContaining({
            Authorization: expect.any(String),
          }),
        })
      );
    });
  });

  describe('Error Handling', () => {
    it('should handle HTTP error responses correctly', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: HTTP_STATUS.BAD_REQUEST,
        headers: { get: jest.fn().mockReturnValue('application/json') },
        json: jest.fn().mockResolvedValue({ error: 'Validation failed' }),
      });

      const response = await apiClient.get('/invalid');

      expect(response.success).toBe(false);
      expect(response.error).toEqual({
        message: 'Validation failed',
        status: HTTP_STATUS.BAD_REQUEST,
      });
    });

    it('should handle network errors correctly', async () => {
      mockFetch.mockRejectedValueOnce(new Error('Network error'));

      const response = await apiClient.get('/test');

      expect(response.success).toBe(false);
      expect(response.error?.message).toBe('Network error - please check your connection');
    });

    it('should handle timeout errors correctly', async () => {
      // Mock AbortError for timeout
      const abortError = new Error('Request timeout');
      abortError.name = 'AbortError';
      mockFetch.mockRejectedValueOnce(abortError);

      const response = await apiClient.get('/test', { timeout: 1000 });

      expect(response.success).toBe(false);
      expect(response.error?.message).toBe('Request timeout - please try again');
    });

    it('should handle JSON parsing errors gracefully', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: HTTP_STATUS.INTERNAL_SERVER_ERROR,
        headers: { get: jest.fn().mockReturnValue('application/json') },
        json: jest.fn().mockRejectedValue(new Error('Invalid JSON')),
        text: jest.fn().mockResolvedValue('Server Error'),
      });

      const response = await apiClient.get('/test');

      expect(response.success).toBe(false);
      expect(response.error?.status).toBe(HTTP_STATUS.INTERNAL_SERVER_ERROR);
    });
  });

  describe('Response Processing', () => {
    it('should handle 204 No Content responses', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        status: HTTP_STATUS.NO_CONTENT,
        headers: { get: jest.fn().mockReturnValue('application/json') },
      });

      const response = await apiClient.delete('/test');

      expect(response.success).toBe(true);
      expect(response.data).toBeNull();
    });

    it('should handle non-JSON responses', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        status: HTTP_STATUS.OK,
        headers: { get: jest.fn().mockReturnValue('text/plain') },
        text: jest.fn().mockResolvedValue('Plain text response'),
      });

      const response = await apiClient.get('/text');

      expect(response.success).toBe(true);
      expect(response.data).toEqual({ message: 'Plain text response' });
    });
  });

  describe('Configuration', () => {
    it('should use custom base URL when provided', async () => {
      const customClient = new ApiClient('https://custom-api.com/v2');
      
      mockFetch.mockResolvedValueOnce({
        ok: true,
        status: HTTP_STATUS.OK,
        headers: { get: jest.fn().mockReturnValue('application/json') },
        json: jest.fn().mockResolvedValue({}),
      });

      await customClient.get('/test');

      expect(mockFetch).toHaveBeenCalledWith(
        'https://custom-api.com/v2/test',
        expect.any(Object)
      );
    });

    it('should override base URL with config baseURL', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        status: HTTP_STATUS.OK,
        headers: { get: jest.fn().mockReturnValue('application/json') },
        json: jest.fn().mockResolvedValue({}),
      });

      await apiClient.get('/test', { baseURL: 'https://override.com/api' });

      expect(mockFetch).toHaveBeenCalledWith(
        'https://override.com/api/test',
        expect.any(Object)
      );
    });
  });

  describe('Interceptors', () => {
    it('should apply request interceptors', async () => {
      const requestInterceptor = {
        onRequest: jest.fn().mockImplementation((config) => ({
          ...config,
          headers: { ...config.headers, 'X-Custom': 'test' }
        }))
      };

      apiClient.addRequestInterceptor(requestInterceptor);

      mockFetch.mockResolvedValueOnce({
        ok: true,
        status: HTTP_STATUS.OK,
        headers: { get: jest.fn().mockReturnValue('application/json') },
        json: jest.fn().mockResolvedValue({}),
      });

      await apiClient.get('/test');

      expect(requestInterceptor.onRequest).toHaveBeenCalled();
      expect(mockFetch).toHaveBeenCalledWith(
        expect.any(String),
        expect.objectContaining({
          headers: expect.objectContaining({
            'X-Custom': 'test',
          }),
        })
      );
    });

    it('should apply response interceptors', async () => {
      const responseInterceptor = {
        onResponse: jest.fn().mockImplementation((response) => ({
          ...response,
          data: { ...response.data, intercepted: true }
        }))
      };

      apiClient.addResponseInterceptor(responseInterceptor);

      mockFetch.mockResolvedValueOnce({
        ok: true,
        status: HTTP_STATUS.OK,
        headers: { get: jest.fn().mockReturnValue('application/json') },
        json: jest.fn().mockResolvedValue({ original: true }),
      });

      const response = await apiClient.get('/test');

      expect(responseInterceptor.onResponse).toHaveBeenCalled();
      expect(response.data).toEqual({ original: true, intercepted: true });
    });
  });
});