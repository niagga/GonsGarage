'use client';

import React from 'react';
import { useRouter } from 'next/navigation';
import { Appointment } from '@/types/appointment';
import { useCarStore } from '@/stores/car.store';
import { useAppointmentStore } from '@/stores/appointment.store';
import styles from './AppointmentCard.module.css';

function appointmentStatusLabel(status: string): string {
  const map: Record<string, string> = {
    scheduled: 'Agendado',
    confirmed: 'Confirmado',
    in_progress: 'Em curso',
    'in-progress': 'Em curso',
    completed: 'Concluído',
    cancelled: 'Cancelado',
  };
  return map[status] ?? status.replace(/_/g, ' ').replace(/-/g, ' ');
}

interface AppointmentCardProps {
  appointment: Appointment;
  onStatusChange?: (appointmentId: string, action: 'cancel' | 'confirm' | 'complete') => void;
  /** Opens edit UI (e.g. modal). If omitted, no Editar action is shown — there is no dedicated /edit route. */
  onEdit?: (appointment: Appointment) => void;
  /** Opens in-app scheduling (e.g. modal on /appointments). Falls back to navigation if omitted. */
  onReschedule?: (carId: string) => void;
}

export default function AppointmentCard({ appointment, onStatusChange, onEdit, onReschedule }: AppointmentCardProps) {
  const router = useRouter();
  const { cars } = useCarStore();
  const { 
    cancelAppointment, 
    confirmAppointment, 
    completeAppointment,
    isUpdating 
  } = useAppointmentStore();

  // Find the car associated with this appointment
  const car = cars.find(c => c.id === appointment.carId);

  // Format the appointment date
  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleDateString('pt-PT', {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit'
    });
  };

  // Handle status change actions
  const handleStatusChange = async (action: 'cancel' | 'confirm' | 'complete') => {
    try {
      let success = false;
      
      switch (action) {
        case 'cancel':
          success = await cancelAppointment(appointment.id);
          break;
        case 'confirm':
          success = await confirmAppointment(appointment.id);
          break;
        case 'complete':
          success = await completeAppointment(appointment.id);
          break;
      }

      if (success && onStatusChange) {
        onStatusChange(appointment.id, action);
      }
    } catch (error) {
      console.error(`Failed to ${action} appointment:`, error);
    }
  };

  // Get status badge class
  const getStatusClass = (status: string) => {
    switch (status) {
      case 'scheduled':
        return styles.scheduled;
      case 'confirmed':
        return styles.confirmed;
      case 'in-progress':
        return styles.inProgress;
      case 'completed':
        return styles.completed;
      case 'cancelled':
        return styles.cancelled;
      default:
        return '';
    }
  };

  return (
    <div className={styles.appointmentCard}>
      {/* Header with date and status */}
      <div className={styles.appointmentHeader}>
        <div className={styles.appointmentDate}>
          <span className={styles.dateLabel}>Marcado para</span>
          <span className={styles.dateValue}>
            {formatDate(appointment.date)}
          </span>
        </div>
        <span className={`${styles.statusBadge} ${getStatusClass(appointment.status)}`}>
          {appointmentStatusLabel(appointment.status)}
        </span>
      </div>
      
      {/* Body with service info and car details */}
      <div className={styles.appointmentBody}>
        <div className={styles.serviceInfo}>
          <h3 className={styles.serviceType}>{appointment.service}</h3>
          {appointment.notes && (
            <p className={styles.notes}>{appointment.notes}</p>
          )}
        </div>

        {/* Car information */}
        {car && (
          <div className={styles.carInfo}>
            <div className={styles.carIcon}>🚗</div>
            <div className={styles.carDetails}>
              <h4>{car.year} {car.make} {car.model}</h4>
              <p>{car.licensePlate}</p>
            </div>
          </div>
        )}
      </div>
      
      {/* Footer: acções (o ID interno não é mostrado ao utilizador) */}
      <div className={styles.appointmentFooter}>
        <div className={styles.appointmentActions}>
          {/* Actions based on appointment status */}
          {appointment.status === 'scheduled' && (
            <>
              <button 
                onClick={() => handleStatusChange('confirm')}
                className={styles.confirmButton}
                disabled={isUpdating}
              >
                {isUpdating ? 'A atualizar…' : 'Confirmar'}
              </button>
              {onEdit && (
                <button
                  type="button"
                  onClick={() => onEdit(appointment)}
                  className={styles.editButton}
                >
                  Editar
                </button>
              )}
              <button 
                onClick={() => handleStatusChange('cancel')}
                className={styles.cancelButton}
                disabled={isUpdating}
              >
                Cancelar
              </button>
            </>
          )}
          
          {appointment.status === 'confirmed' && (
            <>
              <button 
                onClick={() => handleStatusChange('complete')}
                className={styles.completeButton}
                disabled={isUpdating}
              >
                {isUpdating ? 'A atualizar…' : 'Concluir'}
              </button>
              <button 
                onClick={() => handleStatusChange('cancel')}
                className={styles.cancelButton}
                disabled={isUpdating}
              >
                Cancelar
              </button>
            </>
          )}
          
          {appointment.status === 'completed' && car && (
            <button 
              onClick={() => router.push(`/cars/${car.id}`)}
              className={styles.viewButton}
            >
              Ver automóvel
            </button>
          )}

          {appointment.status === 'cancelled' && (
            <button
              type="button"
              onClick={() =>
                onReschedule
                  ? onReschedule(appointment.carId)
                  : router.push(`/appointments?schedule=1&carId=${encodeURIComponent(appointment.carId)}`)
              }
              className={styles.rescheduleButton}
            >
              Remarcar
            </button>
          )}
        </div>
      </div>
    </div>
  );
}