// ✅ Centralized services exports following Agent.md patterns
// Provides single entry point for all API services

export { authService, AuthService } from './auth.service';
export { carService, CarService } from './car.service';
export { supplierService, SupplierService } from './supplier.service';
export { receivedInvoiceService, ReceivedInvoiceService } from './received-invoice.service';
export { billingDocumentService, BillingDocumentService } from './billing-document.service';
export { issuedInvoiceService, IssuedInvoiceService } from './issued-invoice.service';
export { fiscalizationService, FiscalizationService } from './fiscalization.service';
export {
  fiscalIntegrationService,
  FiscalIntegrationService,
} from './fiscal-integration.service';
export type { FiscalConnectionSetupBody } from './fiscal-integration.service';

// ✅ Export types for convenience
export type { 
  ApiResponse, 
  ApiError, 
  RequestConfig,
  RequestInterceptor,
  ResponseInterceptor 
} from '../api-client';

// ✅ Export API client for direct usage when needed
export { apiClient, createApiClient, HTTP_STATUS } from '../api-client';

// ✅ Re-export auth types
export type {
  User,
  UserRole, 
  LoginRequest,
  LoginResponse,
  RegisterRequest
} from '@/types/auth';

// ✅ Re-export car types
export type {
  Car,
  CreateCarRequest,
  UpdateCarRequest,
  CarFilters,
  CarListResponse
} from './car.service';