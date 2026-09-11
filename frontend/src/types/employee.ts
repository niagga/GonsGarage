// Employee types following Agent.md (Profile extension, not separate entity)

import { User } from './user';

// ✅ Employee Profile - Extension of User, not separate entity per Agent.md
export interface Employee {
  id: string;
  userId: string; // ✅ camelCase per Agent.md - Reference to User entity
  user: User;     // ✅ Unified User reference
  position: string;
  hourlyRate?: number; // ✅ camelCase per Agent.md
  hireDate: string;    // ✅ camelCase per Agent.md
  createdAt: string;   // ✅ camelCase per Agent.md
  updatedAt: string;   // ✅ camelCase per Agent.md
  deletedAt?: string;  // ✅ camelCase per Agent.md
}

export interface CreateEmployeeRequest {
  userId: string; // ✅ camelCase per Agent.md - Must be existing User with employee role
  position: string;
  hourlyRate?: number; // ✅ camelCase per Agent.md
  hireDate: string;    // ✅ camelCase per Agent.md
}

export interface UpdateEmployeeRequest {
  position?: string;
  hourlyRate?: number; // ✅ camelCase per Agent.md
  hireDate?: string;   // ✅ camelCase per Agent.md
}

export interface EmployeeListResponse {
  employees: Employee[];
  total: number;
  page: number;
  pageSize: number; // ✅ camelCase per Agent.md
}

export interface EmployeeQueryParams {
  page?: number;
  pageSize?: number; // ✅ camelCase per Agent.md
  position?: string;
  search?: string;
  sortBy?: 'position' | 'hireDate' | 'hourlyRate' | 'createdAt'; // ✅ camelCase per Agent.md
  sortOrder?: 'asc' | 'desc';
}

// Employee-specific business logic types
export interface WorkSchedule {
  id: string;
  employeeId: string; // ✅ camelCase per Agent.md
  dayOfWeek: number;  // ✅ camelCase per Agent.md (0 = Sunday, 6 = Saturday)
  startTime: string;  // ✅ camelCase per Agent.md (HH:MM format)
  endTime: string;    // ✅ camelCase per Agent.md (HH:MM format)
  isActive: boolean;  // ✅ camelCase per Agent.md
}

export interface Timesheet {
  id: string;
  employeeId: string; // ✅ camelCase per Agent.md
  date: string;
  clockIn: string;    // ✅ camelCase per Agent.md (ISO datetime)
  clockOut?: string;  // ✅ camelCase per Agent.md (ISO datetime)
  breakMinutes?: number; // ✅ camelCase per Agent.md
  totalMinutes?: number; // ✅ camelCase per Agent.md
  overtimeMinutes?: number; // ✅ camelCase per Agent.md
  notes?: string;
  createdAt: string;  // ✅ camelCase per Agent.md
  updatedAt: string;  // ✅ camelCase per Agent.md
}

export interface EmployeePerformance {
  id: string;
  employeeId: string;        // ✅ camelCase per Agent.md
  reviewPeriod: string;      // ✅ camelCase per Agent.md (e.g., "2024-Q1")
  overallRating: number;     // ✅ camelCase per Agent.md (1-5 scale)
  punctualityRating: number; // ✅ camelCase per Agent.md
  qualityRating: number;     // ✅ camelCase per Agent.md
  customerServiceRating: number; // ✅ camelCase per Agent.md
  comments?: string;
  reviewedBy: string;        // ✅ camelCase per Agent.md (User ID)
  reviewDate: string;        // ✅ camelCase per Agent.md
  createdAt: string;         // ✅ camelCase per Agent.md
  updatedAt: string;         // ✅ camelCase per Agent.md
}

// Employee dashboard data
export interface EmployeeDashboard {
  employee: Employee;
  todaySchedule: WorkSchedule[];
  recentTimesheets: Timesheet[];
  upcomingAppointments: number;
  monthlyHours: number;
  monthlyEarnings: number;
}