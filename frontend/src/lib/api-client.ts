// ✅ Centralized API client following Agent.md standards
// Handles HTTP requests, error management, authentication, and interceptors

import type { ItemsTotal } from '@/types/accounting';
import type { User } from '@/types';
import type { PartItem, PartItemWriteBody } from '@/types/parts';

import { getPublicApiOrigin } from './api-public-origin';

// ✅ Base configuration constants
const API_VERSION = 'v1';
const DEFAULT_TIMEOUT = 30000; // 30 seconds

// ✅ Standard API response interface following Agent.md
export interface ApiResponse<T> {
  success: boolean;
  data?: T;
  error?: ApiError;
  message?: string;
}

// ✅ Enhanced error interface with detailed information
export interface ApiError {
  message: string;
  code?: string;
  status: number;
  details?: Record<string, unknown>;
}

export interface ClientLookupUser {
  id: string;
  email: string;
  firstName: string;
  lastName: string;
}

export interface ClientLookupResponse {
  items: ClientLookupUser[];
  total: number;
}

// ✅ Request configuration interface
export interface RequestConfig extends Omit<RequestInit, 'body'> {
  body?: unknown;
  timeout?: number;
  skipAuth?: boolean;
  baseURL?: string;
  retry?: number;
  retryDelay?: number;
}

// ✅ HTTP status code constants
export const HTTP_STATUS = {
  OK: 200,
  CREATED: 201,
  NO_CONTENT: 204,
  BAD_REQUEST: 400,
  UNAUTHORIZED: 401,
  FORBIDDEN: 403,
  NOT_FOUND: 404,
  CONFLICT: 409,
  UNPROCESSABLE_ENTITY: 422,
  INTERNAL_SERVER_ERROR: 500,
  SERVICE_UNAVAILABLE: 503,
} as const;

// ✅ Interceptor types for request/response middleware
export interface RequestInterceptor {
  onRequest?: (config: RequestConfig) => RequestConfig | Promise<RequestConfig>;
  onRequestError?: (error: Error) => Error | Promise<Error>;
}

export interface ResponseInterceptor {
  onResponse?: <T>(response: ApiResponse<T>) => ApiResponse<T> | Promise<ApiResponse<T>>;
  onResponseError?: (error: ApiError) => ApiError | Promise<ApiError>;
}

// ✅ Main API client class following Agent.md clean architecture
export class ApiClient {
  private baseURL: string;
  private token: string | null = null;
  private requestInterceptors: RequestInterceptor[] = [];
  private responseInterceptors: ResponseInterceptor[] = [];

  constructor(baseURL?: string) {
    this.baseURL = baseURL || `${getPublicApiOrigin()}/api/${API_VERSION}`;
    
    // ✅ Initialize token from localStorage (client-side only)
    if (typeof window !== 'undefined') {
      this.token = localStorage.getItem('auth_token');
    }
  }

  // ✅ Token management methods
  setToken(token: string): void {
    this.token = token;
    if (typeof window !== 'undefined') {
      localStorage.setItem('auth_token', token);
    }
  }

  clearToken(): void {
    this.token = null;
    if (typeof window !== 'undefined') {
      localStorage.removeItem('auth_token');
      localStorage.removeItem('auth_user');
    }
  }

  getToken(): string | null {
    return this.token;
  }

  // ✅ Interceptor management
  addRequestInterceptor(interceptor: RequestInterceptor): void {
    this.requestInterceptors.push(interceptor);
  }

  addResponseInterceptor(interceptor: ResponseInterceptor): void {
    this.responseInterceptors.push(interceptor);
  }

  // ✅ Enhanced error handling following Agent.md patterns
  private createApiError(
    message: string, 
    status: number, 
    code?: string,
    details?: Record<string, unknown>
  ): ApiError {
    return {
      message,
      status,
      code,
      details,
    };
  }

  // ✅ Process request through interceptors
  private async processRequestInterceptors(config: RequestConfig): Promise<RequestConfig> {
    let processedConfig = config;
    
    for (const interceptor of this.requestInterceptors) {
      if (interceptor.onRequest) {
        try {
          processedConfig = await interceptor.onRequest(processedConfig);
        } catch (err) {
          if (interceptor.onRequestError) {
            throw await interceptor.onRequestError(err as Error);
          }
          throw err;
        }
      }
    }
    
    return processedConfig;
  }

  // ✅ Process response through interceptors
  private async processResponseInterceptors<T>(response: ApiResponse<T>): Promise<ApiResponse<T>> {
    let processedResponse = response;
    
    for (const interceptor of this.responseInterceptors) {
      if (response.error && interceptor.onResponseError) {
        try {
          const processedError = await interceptor.onResponseError(response.error);
          processedResponse = { ...processedResponse, error: processedError };
        } catch {
          // Continue with original error if interceptor fails
        }
      } else if (interceptor.onResponse) {
        try {
          processedResponse = await interceptor.onResponse(processedResponse);
        } catch {
          // Continue with original response if interceptor fails
        }
      }
    }
    
    return processedResponse;
  }

  // ✅ Core request method with comprehensive error handling
  async request<T = unknown>(
    endpoint: string,
    config: RequestConfig = {}
  ): Promise<ApiResponse<T>> {
    try {
      // ✅ Process request through interceptors
      const processedConfig = await this.processRequestInterceptors(config);
      
      // ✅ Extract configuration values
      const { body, timeout: configTimeout, skipAuth: configSkipAuth, baseURL: configBaseURL, retry: configRetry, retryDelay: configRetryDelay, ...restConfig } = processedConfig;

      // ✅ Keep Authorization in sync with Zustand-persisted session (singleton may be stale after login)
      if (typeof window !== 'undefined' && !configSkipAuth) {
        this.token = window.localStorage.getItem('auth_token');
      }
      
      // ✅ Build complete URL
      const baseURL = configBaseURL || this.baseURL;
      const url = `${baseURL}${endpoint}`;
      
      // ✅ Prepare headers with authentication
      const headers: Record<string, string> = {
        'Content-Type': 'application/json',
        ...((processedConfig.headers as Record<string, string>) || {}),
      };
      
      // ✅ Add authentication header if token exists and not skipped
      if (this.token && !configSkipAuth) {
        headers.Authorization = `Bearer ${this.token}`;
      }

      // ✅ Prepare fetch options
      const fetchOptions: RequestInit = {
        method: processedConfig.method || 'GET',
        headers,
        ...restConfig,
      };

      // ✅ Handle body serialization
      if (body && typeof body !== 'string') {
        fetchOptions.body = JSON.stringify(body);
      } else if (typeof body === 'string') {
        fetchOptions.body = body;
      }

      // ✅ Add timeout support
      const timeout = configTimeout || DEFAULT_TIMEOUT;
      const controller = new AbortController();
      const timeoutId = setTimeout(() => controller.abort(), timeout);
      
      fetchOptions.signal = controller.signal;

      // ✅ Execute request with retry logic
      let lastError: Error | null = null;
      const maxRetries = configRetry || 0;
      const retryDelay = configRetryDelay || 1000;

      for (let attempt = 0; attempt <= maxRetries; attempt++) {
        try {
          const response = await fetch(url, fetchOptions);
          clearTimeout(timeoutId);

          // ✅ Handle different response types
          let responseData: unknown;
          
          const contentType = response.headers.get('content-type');
          const isJson = contentType?.includes('application/json');

          if (response.status === HTTP_STATUS.NO_CONTENT) {
            responseData = null;
          } else if (isJson) {
            try {
              responseData = await response.json();
            } catch {
              // ✅ If JSON parsing fails, try to get text instead
              try {
                const text = await response.text();
                responseData = { message: text };
              } catch {
                responseData = { message: 'Failed to parse response' };
              }
            }
          } else {
            const text = await response.text();
            // ✅ Try to parse as JSON, fallback to text
            try {
              responseData = JSON.parse(text);
            } catch {
              responseData = { message: text };
            }
          }

          // ✅ Handle HTTP error status codes
          if (!response.ok) {
            const errorMessage = this.extractErrorMessage(responseData, response.status);
            const apiError = this.createApiError(
              errorMessage,
              response.status,
              this.extractErrorCode(responseData),
              this.extractErrorDetails(responseData)
            );

            const errorResponse: ApiResponse<T> = {
              success: false,
              error: apiError,
            };

            return await this.processResponseInterceptors(errorResponse);
          }

          // ✅ Success response
          const successResponse: ApiResponse<T> = {
            success: true,
            data: responseData as T,
            message: this.extractSuccessMessage(responseData),
          };

          return await this.processResponseInterceptors(successResponse);

        } catch (error) {
          clearTimeout(timeoutId);
          lastError = error as Error;

          // ✅ Don't retry on certain errors
          if (error instanceof Error) {
            if (error.name === 'AbortError') {
              break; // Don't retry timeout errors
            }
            if (attempt < maxRetries) {
              await new Promise(resolve => setTimeout(resolve, retryDelay * (attempt + 1)));
              continue;
            }
          }
          break;
        }
      }

      // ✅ Handle network/timeout errors
      const networkError = this.createApiError(
        lastError?.name === 'AbortError' 
          ? 'Request timeout - please try again'
          : 'Network error - please check your connection',
        0,
        lastError?.name || 'NETWORK_ERROR'
      );

      const errorResponse: ApiResponse<T> = {
        success: false,
        error: networkError,
      };

      return await this.processResponseInterceptors(errorResponse);

    } catch {
      // ✅ Fallback error handling
      const fallbackError = this.createApiError(
        'An unexpected error occurred',
        0,
        'UNKNOWN_ERROR'
      );

      return {
        success: false,
        error: fallbackError,
      };
    }
  }

  // ✅ Helper methods for error extraction following Agent.md patterns
  private extractErrorMessage(data: unknown, status: number): string {
    if (typeof data === 'object' && data !== null) {
      const errorObj = data as Record<string, unknown>;
      
      // ✅ Try different common error message fields
      const message = errorObj.error || errorObj.message || errorObj.detail;
      
      if (typeof message === 'string') {
        return message;
      }
    }
    
    // ✅ Fallback to status-based messages
    switch (status) {
      case HTTP_STATUS.BAD_REQUEST:
        return 'Invalid request data';
      case HTTP_STATUS.UNAUTHORIZED:
        return 'Authentication required';
      case HTTP_STATUS.FORBIDDEN:
        return 'Access denied';
      case HTTP_STATUS.NOT_FOUND:
        return 'Resource not found';
      case HTTP_STATUS.CONFLICT:
        return 'Resource conflict';
      case HTTP_STATUS.UNPROCESSABLE_ENTITY:
        return 'Validation failed';
      case HTTP_STATUS.INTERNAL_SERVER_ERROR:
        return 'Internal server error';
      case HTTP_STATUS.SERVICE_UNAVAILABLE:
        return 'Service temporarily unavailable';
      default:
        return `Request failed (${status})`;
    }
  }

  private extractErrorCode(data: unknown): string | undefined {
    if (typeof data === 'object' && data !== null) {
      const errorObj = data as Record<string, unknown>;
      return typeof errorObj.code === 'string' ? errorObj.code : undefined;
    }
    return undefined;
  }

  private extractErrorDetails(data: unknown): Record<string, unknown> | undefined {
    if (typeof data === 'object' && data !== null) {
      const errorObj = data as Record<string, unknown>;
      return errorObj.details as Record<string, unknown> || undefined;
    }
    return undefined;
  }

  private extractSuccessMessage(data: unknown): string | undefined {
    if (typeof data === 'object' && data !== null) {
      const successObj = data as Record<string, unknown>;
      return typeof successObj.message === 'string' ? successObj.message : undefined;
    }
    return undefined;
  }

  // ✅ Convenience methods for common HTTP verbs
  async get<T>(endpoint: string, config?: RequestConfig): Promise<ApiResponse<T>> {
    return this.request<T>(endpoint, { ...config, method: 'GET' });
  }

  async post<T>(endpoint: string, body?: unknown, config?: RequestConfig): Promise<ApiResponse<T>> {
    return this.request<T>(endpoint, { ...config, method: 'POST', body });
  }

  /** Staff-only: POST /api/v1/admin/users (JWT required; backend enforces admin/manager + role matrix). */
  async provisionUser(body: {
    email: string;
    password: string;
    firstName: string;
    lastName: string;
    role: 'manager' | 'employee' | 'client';
  }): Promise<ApiResponse<{ user: User }>> {
    return this.post<{ user: User }>('/admin/users', body);
  }

  /** Staff-only: GET /api/v1/admin/users/clients for car owner association. */
  async listClientUsers(params?: {
    q?: string;
    limit?: number;
    offset?: number;
  }): Promise<ApiResponse<ClientLookupResponse>> {
    const q = new URLSearchParams();
    if (params?.q) q.set('q', params.q);
    if (params?.limit != null) q.set('limit', String(params.limit));
    if (params?.offset != null) q.set('offset', String(params.offset));
    const qs = q.toString();
    return this.get<ClientLookupResponse>(`/admin/users/clients${qs ? `?${qs}` : ''}`);
  }

  /** Manager/admin: GET /api/v1/parts (`barcode`, `search`, pagination). */
  async listParts(params?: {
    barcode?: string;
    search?: string;
    limit?: number;
    offset?: number;
  }): Promise<ApiResponse<ItemsTotal<PartItem>>> {
    const q = new URLSearchParams();
    if (params?.barcode) q.set('barcode', params.barcode);
    if (params?.search) q.set('search', params.search);
    if (params?.limit != null) q.set('limit', String(params.limit));
    if (params?.offset != null) q.set('offset', String(params.offset));
    const qs = q.toString();
    return this.get<ItemsTotal<PartItem>>(`/parts${qs ? `?${qs}` : ''}`);
  }

  /** Manager/admin: GET /api/v1/parts/:id */
  async getPart(id: string): Promise<ApiResponse<PartItem>> {
    return this.get<PartItem>(`/parts/${encodeURIComponent(id)}`);
  }

  /** Manager/admin: POST /api/v1/parts */
  async createPart(body: PartItemWriteBody): Promise<ApiResponse<PartItem>> {
    return this.post<PartItem>('/parts', body);
  }

  /** Manager/admin: PATCH /api/v1/parts/:id */
  async updatePart(id: string, body: PartItemWriteBody): Promise<ApiResponse<PartItem>> {
    return this.patch<PartItem>(`/parts/${encodeURIComponent(id)}`, body);
  }

  /** Manager/admin: DELETE /api/v1/parts/:id (soft delete). */
  async deletePart(id: string): Promise<ApiResponse<unknown>> {
    return this.delete(`/parts/${encodeURIComponent(id)}`);
  }

  async put<T>(endpoint: string, body?: unknown, config?: RequestConfig): Promise<ApiResponse<T>> {
    return this.request<T>(endpoint, { ...config, method: 'PUT', body });
  }

  async patch<T>(endpoint: string, body?: unknown, config?: RequestConfig): Promise<ApiResponse<T>> {
    return this.request<T>(endpoint, { ...config, method: 'PATCH', body });
  }

  async delete<T>(endpoint: string, config?: RequestConfig): Promise<ApiResponse<T>> {
    return this.request<T>(endpoint, { ...config, method: 'DELETE' });
  }
}

// ✅ Export singleton instance for application use
export const apiClient = new ApiClient();

// ✅ Export factory for creating custom instances
export const createApiClient = (baseURL?: string): ApiClient => {
  return new ApiClient(baseURL);
};

// ✅ Default export for convenience
export default apiClient;