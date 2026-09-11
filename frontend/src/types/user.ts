// User management types following Agent.md unified domain model

import { UserRole } from './auth';

export interface User {
  id: string;
  email: string;
  firstName: string; // ✅ camelCase per Agent.md  
  lastName: string;  // ✅ camelCase per Agent.md
  role: UserRole;
  createdAt: string; // ✅ camelCase per Agent.md
  updatedAt: string; // ✅ camelCase per Agent.md
  deletedAt?: string; // ✅ camelCase per Agent.md
}

// ✅ Unified approach - Single User entity with roles (Agent.md compliant)
export interface CreateUserRequest {
  email: string;
  password: string;
  firstName: string; // ✅ camelCase per Agent.md
  lastName: string;  // ✅ camelCase per Agent.md
  role: UserRole;
}

export interface UpdateUserRequest {
  firstName?: string; // ✅ camelCase per Agent.md
  lastName?: string;  // ✅ camelCase per Agent.md
  email?: string;
  role?: UserRole;
}

export interface UserListResponse {
  users: User[];
  total: number;
  page: number;
  pageSize: number; // ✅ camelCase per Agent.md
}

export interface UserQueryParams {
  page?: number;
  pageSize?: number; // ✅ camelCase per Agent.md
  role?: UserRole;
  search?: string;
  sortBy?: 'firstName' | 'lastName' | 'email' | 'createdAt'; // ✅ camelCase per Agent.md
  sortOrder?: 'asc' | 'desc';
}

// Helper functions for role checking (Agent.md pattern)
export const isClient = (user: User): boolean => user.role === UserRole.CLIENT;
/** Personal do taller: employee, manager ou admin (navegação e API de visitas). */
export const isWorkshopStaff = (user: User): boolean => !isClient(user);
export const isEmployee = (user: User): boolean => user.role === UserRole.EMPLOYEE;
export const isAdmin = (user: User): boolean => user.role === UserRole.ADMIN;
export const canManageUsers = (user: User): boolean =>
  user.role === UserRole.ADMIN || user.role === UserRole.MANAGER;

// Profile management for any user type
export interface UserProfile {
  id: string;
  userId: string; // ✅ camelCase per Agent.md
  firstName: string; // ✅ camelCase per Agent.md
  lastName: string;  // ✅ camelCase per Agent.md
  email: string;
  phone?: string;
  address?: UserAddress;
  preferences?: UserPreferences;
  createdAt: string; // ✅ camelCase per Agent.md
  updatedAt: string; // ✅ camelCase per Agent.md
}

export interface UserAddress {
  street: string;
  city: string;
  state: string;
  zipCode: string; // ✅ camelCase per Agent.md
  country: string;
}

export interface UserPreferences {
  language: string;
  timezone: string;
  notifications: NotificationSettings;
}

export interface NotificationSettings {
  email: boolean;
  sms: boolean;
  push: boolean;
  appointmentReminders: boolean; // ✅ camelCase per Agent.md
  repairUpdates: boolean; // ✅ camelCase per Agent.md
}