// Central types export following Agent.md conventions

// ✅ Authentication types
export type {
  User,
  LoginRequest,
  LoginResponse,
  RegisterRequest,
  RefreshTokenRequest,
  AuthContextType,
  JWTPayload,
  PasswordResetRequest,
  PasswordResetConfirm
} from './auth';

export { UserRole } from './auth';

export type {
  FiscalDraftRequest,
  FiscalizationProjection,
  FiscalizationSummary,
  FiscalDocumentKind,
  FiscalPresentationStatus,
} from './fiscal';

export {
  fiscalPresentationLabel,
  isTransientFiscalLifecycle,
  canShowFiscalAction,
} from './fiscal';

// ✅ User management types (Unified per Agent.md)
export type {
  CreateUserRequest,
  UpdateUserRequest,
  UserListResponse,
  UserQueryParams,
  UserProfile,
  UserAddress,
  UserPreferences,
  NotificationSettings
} from './user';

export {
  isClient,
  isEmployee,
  isAdmin,
  canManageUsers
} from './user';

// ✅ Employee types (Profile extension per Agent.md)
export type {
  Employee,
  CreateEmployeeRequest,
  UpdateEmployeeRequest,
  EmployeeListResponse,
  EmployeeQueryParams,
  WorkSchedule,
  Timesheet,
  EmployeePerformance,
  EmployeeDashboard
} from './employee';

// ✅ Car types (existing)
export type {
  Car,
  CreateCarRequest,
  UpdateCarRequest,
  CarFormData,
  CarValidationErrors,
  Repair
} from './car';

// ✅ Accounting / P1 billing types
export type {
  BillingDocumentKind,
  Supplier,
  ReceivedInvoice,
  BillingDocument,
  IssuedInvoice,
  ItemsTotal
} from './accounting';

// ✅ Parts inventory (manager/admin)
export type { PartItem, PartItemWriteBody, PartUOM } from './parts';

// ✅ API types per Agent.md
export type {
  ApiResponse,
  PaginatedResponse,
  ErrorResponse,
  ApiClientConfig,
  RequestOptions,
  ApiClient,
  FileUploadResponse,
  UploadOptions
} from './api';

export {
  HttpMethod,
  API_ENDPOINTS,
  HttpStatusCode
} from './api';

// ✅ Re-export common domain types for convenience
export type { User as UserType } from './auth';

// ✅ Type guards for runtime type checking
import { UserRole } from './auth';

export const isUserRole = (role: string): role is UserRole => {
  return Object.values(UserRole).includes(role as UserRole);
};

// ✅ Utility types per Agent.md
export type ID = string;
export type Timestamp = string;
export type Email = string;

// ✅ Common form types
export interface FormErrors {
  [key: string]: string;
}

export interface FormState<T> {
  data: T;
  errors: FormErrors;
  isSubmitting: boolean;
  isDirty: boolean;
}

// ✅ Common UI component props
export interface BaseComponentProps {
  className?: string;
  children?: React.ReactNode;
}

export interface LoadingProps {
  loading?: boolean;
  error?: string | null;
}

// ✅ Pagination types for UI components
export interface PaginationProps {
  currentPage: number;
  totalPages: number;
  pageSize: number;
  total: number;
  onPageChange: (page: number) => void;
  onPageSizeChange?: (pageSize: number) => void;
}