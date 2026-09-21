// Car domain types following Agent.md camelCase conventions

export interface Car {
  id: string;
  make: string;
  model: string;
  year: number;
  licensePlate: string; // ✅ camelCase per Agent.md
  vin?: string;
  color: string;
  mileage?: number;
  ownerId: string; // ✅ camelCase per Agent.md
  createdAt: string; // ✅ camelCase per Agent.md
  updatedAt: string; // ✅ camelCase per Agent.md
  repairs?: Repair[];
}

export interface CreateCarRequest {
  make: string;
  model: string;
  year: number;
  licensePlate: string; // ✅ camelCase per Agent.md
  vin?: string;
  color: string;
  mileage?: number;
  ownerID?: string; // backend create payload expects ownerID
}

export interface UpdateCarRequest extends Partial<CreateCarRequest> {
  id: string;
}

export interface CarFormData {
  make: string;
  model: string;
  year: number;
  licensePlate: string; // ✅ camelCase per Agent.md
  vin: string;
  color: string;
  mileage: number;
}

export interface CarValidationErrors {
  make?: string;
  model?: string;
  year?: string;
  licensePlate?: string; // ✅ camelCase per Agent.md
  color?: string;
  mileage?: string;
  general?: string;
}

export interface Repair {
  id: string;
  carId: string; // ✅ camelCase per Agent.md
  description: string;
  status: 'pending' | 'inProgress' | 'completed' | 'cancelled'; // ✅ camelCase per Agent.md
  cost: number;
  startedAt?: string; // ✅ camelCase per Agent.md
  completedAt?: string; // ✅ camelCase per Agent.md
  createdAt: string; // ✅ camelCase per Agent.md
  updatedAt: string; // ✅ camelCase per Agent.md
}