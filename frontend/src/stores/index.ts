/**
 * Stores index — central exports following Agent.md.
 * Client-only: import from `"use client"` modules, hooks, or event handlers only
 * (Zustand + `persist`); do not import from Server Components.
 */

// ✅ Authentication store
export {
  useAuthStore,
  useAuth,
  useUser,
  useAuthToken,
  useAuthLoading,
  useAuthError,
  useIsAuthenticated
} from './auth.store';

// ✅ Car store
export {
  useCarStore,
  useCars,
  useCarList,
  useCarDetails,
  useCarMutations
} from './car.store';

// ✅ Appointment store
export {
  useAppointmentStore,
  useAppointments,
} from './appointment.store';

export type {
  Appointment,
  CreateAppointmentRequest,
  UpdateAppointmentRequest,
} from '@/types/appointment';

// ✅ Store types available for advanced usage
// Store types are internal to their implementations

// ✅ Re-export types for convenience
export type {
  User,
  RegisterRequest,
  LoginRequest,
  LoginResponse
} from '@/types';

export { UserRole } from '@/types';