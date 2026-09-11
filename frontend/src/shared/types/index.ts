// src/shared/types/index.ts
// export interface Car {
//   id: string;
//   make: string;
//   model: string;
//   year: number;
//   licensePlate: string;
//   vin?: string;
//   createdAt: string;
//   updatedAt: string;
// }

export interface Repair {
  id: string;
  car_id: string;
  description: string;
  status: 'pending' | 'in_progress' | 'completed' | 'cancelled';
  cost: number;
  created_at: string;
  completed_at?: string;
}

// export interface Appointment {
//   id: string;
//   customerId: string;
//   carId: string;
//   serviceType: string;
//   scheduledAt: string;
//   status: 'scheduled' | 'confirmed' | 'in_progress' | 'completed' | 'cancelled';
//   notes?: string;
//   createdAt: string;
//   updatedAt: string;
// }

// export interface CreateAppointmentRequest {
//   carId: string;         // ✅ camelCase
//   serviceType: string;   // ✅ camelCase
//   scheduledAt: string;   // ✅ camelCase
//   notes?: string;
// }

// export interface UpdateAppointmentRequest {
//   serviceType?: string;  // ✅ camelCase
//   scheduledAt?: string;  // ✅ camelCase
//   status?: 'scheduled' | 'in-progress' | 'completed' | 'cancelled';
//   notes?: string;
// }

export const SERVICE_TYPES = [
  { id: 'oil_change', name: 'Mudança de óleo', description: 'Mudança periódica de óleo e filtro' },
  { id: 'brake_service', name: 'Travões', description: 'Pastilhas, discos e fluido de travões' },
  { id: 'tire_service', name: 'Pneus', description: 'Rotação, alinhamento e substituição de pneus' },
  { id: 'engine_diagnostic', name: 'Diagnóstico de motor', description: 'Luz de avaria e diagnóstico' },
  { id: 'transmission_service', name: 'Transmissão', description: 'Fluido e inspeção da transmissão' },
  { id: 'air_conditioning', name: 'Climatização', description: 'Reparação e manutenção do A/C' },
  { id: 'battery_service', name: 'Bateria', description: 'Teste e substituição de bateria' },
  { id: 'general_maintenance', name: 'Manutenção geral', description: 'Inspeção multiponto' },
  { id: 'other', name: 'Outro', description: 'Serviço personalizado' },
] as const;