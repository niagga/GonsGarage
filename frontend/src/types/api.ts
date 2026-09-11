// API response types following Agent.md conventions

// ✅ Standard API Response wrapper per Agent.md
export interface ApiResponse<T = unknown> {
  success: boolean;
  data?: T;
  message?: string;
  error?: string;
  errors?: Record<string, string>; // Validation errors
  timestamp: string;
}

// ✅ Paginated API Response per Agent.md
export interface PaginatedResponse<T = unknown> {
  success: boolean;
  data: T[];
  pagination: {
    page: number;
    pageSize: number; // ✅ camelCase per Agent.md
    total: number;
    totalPages: number; // ✅ camelCase per Agent.md
    hasNext: boolean;   // ✅ camelCase per Agent.md
    hasPrev: boolean;   // ✅ camelCase per Agent.md
  };
  message?: string;
  timestamp: string;
}

// ✅ Error Response per Agent.md
export interface ErrorResponse {
  success: false;
  error: string;
  message: string;
  details?: Record<string, string>;
  statusCode: number; // ✅ camelCase per Agent.md
  timestamp: string;
}

// ✅ API Client Configuration per Agent.md
export interface ApiClientConfig {
  baseUrl: string;    // ✅ camelCase per Agent.md
  timeout?: number;
  retries?: number;
  headers?: Record<string, string>;
}

// ✅ HTTP Methods per Agent.md
export enum HttpMethod {
  GET = 'GET',
  POST = 'POST',
  PUT = 'PUT',
  PATCH = 'PATCH',
  DELETE = 'DELETE'
}

// ✅ Request Options per Agent.md
export interface RequestOptions {
  method: HttpMethod;
  headers?: Record<string, string>;
  body?: unknown;
  params?: Record<string, string | number>;
  signal?: AbortSignal;
}

// ✅ API Endpoints per Agent.md
export const API_ENDPOINTS = {
  // Authentication
  AUTH: {
    LOGIN: '/auth/login',
    REGISTER: '/auth/register',
    LOGOUT: '/auth/logout',
    REFRESH: '/auth/refresh',
    PROFILE: '/auth/profile',
    RESET_PASSWORD: '/auth/reset-password',
    CONFIRM_RESET: '/auth/confirm-reset'
  },
  
  // Users (Unified per Agent.md)
  USERS: {
    LIST: '/users',
    CREATE: '/users',
    GET: (id: string) => `/users/${id}`,
    UPDATE: (id: string) => `/users/${id}`,
    DELETE: (id: string) => `/users/${id}`
  },
  
  // Employees (Profile extension per Agent.md)
  EMPLOYEES: {
    LIST: '/employees',
    CREATE: '/employees',
    GET: (id: string) => `/employees/${id}`,
    UPDATE: (id: string) => `/employees/${id}`,
    DELETE: (id: string) => `/employees/${id}`,
    SCHEDULE: (id: string) => `/employees/${id}/schedule`,
    TIMESHEET: (id: string) => `/employees/${id}/timesheet`,
    PERFORMANCE: (id: string) => `/employees/${id}/performance`
  },
  
  // Cars
  CARS: {
    LIST: '/cars',
    CREATE: '/cars',
    GET: (id: string) => `/cars/${id}`,
    UPDATE: (id: string) => `/cars/${id}`,
    DELETE: (id: string) => `/cars/${id}`,
    /** Listado de reparaciones por coche (ver GET /api/v1/repairs/car/{carId}) */
    REPAIRS_BY_CAR: (carId: string) => `/repairs/car/${carId}`
  },
  
  // Appointments
  APPOINTMENTS: {
    LIST: '/appointments',
    CREATE: '/appointments',
    GET: (id: string) => `/appointments/${id}`,
    UPDATE: (id: string) => `/appointments/${id}`,
    DELETE: (id: string) => `/appointments/${id}`
  },
  
  // Repairs: listado por coche + CRUD por id (staff en servicio)
  REPAIRS: {
    BY_CAR: (carId: string) => `/repairs/car/${carId}`,
    CREATE: '/repairs',
    GET: (id: string) => `/repairs/${id}`,
    UPDATE: (id: string) => `/repairs/${id}`,
    DELETE: (id: string) => `/repairs/${id}`
  }
} as const;

// ✅ Status codes per Agent.md
export enum HttpStatusCode {
  OK = 200,
  CREATED = 201,
  NO_CONTENT = 204,
  BAD_REQUEST = 400,
  UNAUTHORIZED = 401,
  FORBIDDEN = 403,
  NOT_FOUND = 404,
  CONFLICT = 409,
  INTERNAL_SERVER_ERROR = 500,
  BAD_GATEWAY = 502,
  SERVICE_UNAVAILABLE = 503
}

// ✅ Generic API Client interface per Agent.md
export interface ApiClient {
  get<T>(url: string, options?: RequestOptions): Promise<ApiResponse<T>>;
  post<T>(url: string, data?: unknown, options?: RequestOptions): Promise<ApiResponse<T>>;
  put<T>(url: string, data?: unknown, options?: RequestOptions): Promise<ApiResponse<T>>;
  patch<T>(url: string, data?: unknown, options?: RequestOptions): Promise<ApiResponse<T>>;
  delete<T>(url: string, options?: RequestOptions): Promise<ApiResponse<T>>;
}

// ✅ Upload types for file handling
export interface FileUploadResponse {
  success: boolean;
  fileUrl: string;    // ✅ camelCase per Agent.md
  fileName: string;   // ✅ camelCase per Agent.md
  fileSize: number;   // ✅ camelCase per Agent.md
  contentType: string; // ✅ camelCase per Agent.md
}

export interface UploadOptions {
  maxSize?: number;      // ✅ camelCase per Agent.md - in bytes
  allowedTypes?: string[]; // ✅ camelCase per Agent.md - MIME types
  folder?: string;
}