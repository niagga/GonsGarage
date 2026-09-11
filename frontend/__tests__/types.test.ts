// Test for types compilation and import structure
import { 
  User, 
  UserRole, 
  Car, 
  Employee, 
  ApiResponse,
  isUserRole,
  canManageUsers
} from '../src/types';

describe('Types Structure', () => {
  it('should import and use User types correctly', () => {
    const user: User = {
      id: '123',
      email: 'test@example.com',
      firstName: 'John',
      lastName: 'Doe',
      role: UserRole.CLIENT,
      createdAt: '2025-01-01T00:00:00Z',
      updatedAt: '2025-01-01T00:00:00Z'
    };

    expect(user.firstName).toBe('John');
    expect(canManageUsers(user)).toBe(false);
    expect(isUserRole('client')).toBe(true);
    expect(isUserRole('invalid')).toBe(false);
  });

  it('should import and use Employee types correctly', () => {
    const employee: Employee = {
      id: '456',
      userId: '123',
      user: {
        id: '123',
        email: 'employee@example.com',
        firstName: 'Jane',
        lastName: 'Smith',
        role: UserRole.EMPLOYEE,
        createdAt: '2025-01-01T00:00:00Z',
        updatedAt: '2025-01-01T00:00:00Z'
      },
      position: 'Mechanic',
      hourlyRate: 25.50,
      hireDate: '2024-01-01T00:00:00Z',
      createdAt: '2025-01-01T00:00:00Z',
      updatedAt: '2025-01-01T00:00:00Z'
    };

    expect(employee.position).toBe('Mechanic');
    expect(employee.user.role).toBe(UserRole.EMPLOYEE);
  });

  it('should import and use Car types correctly', () => {
    const car: Car = {
      id: '789',
      make: 'Toyota',
      model: 'Camry',
      year: 2023,
      licensePlate: 'ABC123',
      color: 'Blue',
      ownerId: '123',
      createdAt: '2025-01-01T00:00:00Z',
      updatedAt: '2025-01-01T00:00:00Z'
    };

    expect(car.make).toBe('Toyota');
    expect(car.licensePlate).toBe('ABC123');
  });

  it('should import and use API types correctly', () => {
    const response: ApiResponse<User> = {
      success: true,
      data: {
        id: '123',
        email: 'test@example.com',
        firstName: 'John',
        lastName: 'Doe',
        role: UserRole.CLIENT,
        createdAt: '2025-01-01T00:00:00Z',
        updatedAt: '2025-01-01T00:00:00Z'
      },
      timestamp: '2025-01-01T00:00:00Z'
    };

    expect(response.success).toBe(true);
    expect(response.data?.firstName).toBe('John');
  });
});