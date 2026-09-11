// Authentication types following Agent.md camelCase conventions

export interface User {
  id: string;
  email: string;
  firstName: string; // ✅ camelCase per Agent.md
  lastName: string;  // ✅ camelCase per Agent.md
  role: UserRole;
  createdAt: string; // ✅ camelCase per Agent.md
  updatedAt: string; // ✅ camelCase per Agent.md
}

export enum UserRole {
  CLIENT = 'client',
  EMPLOYEE = 'employee',
  MANAGER = 'manager',
  ADMIN = 'admin'
}

export interface LoginRequest {
  email: string;
  password: string;
}

export interface LoginResponse {
  user: User;
  token: string;
  refreshToken?: string;
}

export interface RegisterRequest {
  email: string;
  password: string;
  firstName: string; // ✅ camelCase per Agent.md
  lastName: string;  // ✅ camelCase per Agent.md
  role?: UserRole;
}

export interface RefreshTokenRequest {
  refreshToken: string;
}

export interface AuthContextType {
  user: User | null;
  token: string | null;
  loading: boolean;
  error: string | null;
  login: (email: string, password: string) => Promise<void>;
  register: (userData: RegisterRequest) => Promise<void>;
  logout: () => void;
  clearError: () => void;
}

// JWT Token payload interface
export interface JWTPayload {
  userId: string;
  email: string;
  role: UserRole;
  iat: number;
  exp: number;
}

export interface PasswordResetRequest {
  email: string;
}

export interface PasswordResetConfirm {
  token: string;
  newPassword: string;
}