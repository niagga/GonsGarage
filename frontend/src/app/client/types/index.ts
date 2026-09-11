// src/app/client/types/index.ts

import {Repair } from '@/shared/types';
import { Appointment } from '@/types/appointment';
import { Car } from '@/types/car';

// Types específicos do domínio cliente
export interface ClientDashboardProps {
  error: string | null;
  cars: Car[];
  recentRepairs: Repair[];
  upcomingAppointments: Appointment[];
  onNavigate: (tab: string) => void;
}

export interface ClientCarsProps {
  cars: Car[];
  onAddCar: () => void;
}

/** Props alinhados com `ClientAppointments` (dados vêm do store em `AppointmentsContainer`). */
export interface ClientAppointmentsProps {
  onAddAppointment?: (appointment: Appointment) => void;
  onUpdateAppointment?: (appointments: Appointment[]) => void;
  showAddButton?: boolean;
  maxAppointments?: number;
}