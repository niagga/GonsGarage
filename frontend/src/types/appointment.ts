// Appointment type definition

export interface Appointment {
  id: string;
  clientName: string;
  carId: string;
  service: string;
  date: string;
  time: string;
  status: 'scheduled' | 'completed' | 'cancelled' | 'in_progress' | 'confirmed';
  notes?: string;
  createdAt: string;
  updatedAt: string;
  deletedAt?: string;
}

export interface CreateAppointmentRequest {
  clientName: string;
  carId: string;
  service: string;
  date: string;
  time: string;
  status: 'scheduled' | 'completed' | 'cancelled' | 'in_progress' | 'confirmed';
  notes?: string;
  createdAt: string;
  updatedAt: string;
  deletedAt?: string;
}

export interface UpdateAppointmentRequest {
  carId?: string;
  service?: string;
  date?: string;
  time?: string;
  status?: 'scheduled' | 'completed' | 'cancelled' | 'in_progress' | 'confirmed';
  createdAt?: string;
  updatedAt?: string;
  deletedAt?: string;
  notes?: string;
}
