// src/components/client/ClientAppointments.tsx
'use client';

import { Appointment } from '@/types/appointment';
import AppointmentsContainer from '@/app/appointments/components/AppointmentContainer';

interface ClientAppointmentsProps {
  onAddAppointment?: (appointment: Appointment) => void;
  onUpdateAppointment?: (appointments: Appointment[]) => void;
  showAddButton?: boolean;
  maxAppointments?: number;
}

export default function ClientAppointments({
  onAddAppointment, 
  onUpdateAppointment,
  maxAppointments,
}: ClientAppointmentsProps) {
  return (
    <AppointmentsContainer
      onAddAppointment={onAddAppointment}
      onUpdateAppointment={onUpdateAppointment}
      maxAppointments={maxAppointments}
      headerTitle="Your Appointments"
      addButtonText="Add New Appointment"
    />
  );
}