import React from 'react';
import { Appointment } from '@/types/appointment';
import styles from '../appointments.module.css';
import { useRouter } from 'next/navigation';
import { useCarStore } from '@/stores/car.store';
import { useAppointmentStore } from '@/stores/appointment.store';

interface AppointmentCardProps {
  appointment: Appointment;
  onStatusChange?: (appointmentId: string, action: 'cancel' | 'confirm' | 'complete') => void;
  onEdit: () => void;
  onDelete: () => void;
}

// Individual appointment card component following Agent.md clean patterns
export default function AppointmentCard({
  appointment,
  onEdit,
  onDelete,
  onStatusChange,
}: AppointmentCardProps) {
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
    return new Date(dateString).toLocaleDateString('en-US', {
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
        <div className={styles.carIcon}>📅</div>
        <div className={styles.appointmentDate}>
          <span className={styles.dateLabel}>Scheduled</span>
          <span className={styles.dateValue}>
            {formatDate(appointment.date)}
          </span>
        </div>
        <span className={`${styles.statusBadge} ${getStatusClass(appointment.status)}`}>
          {appointment.status.replace('_', ' ').replace('-', ' ')}
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
        {/*<div className={styles.carContent}>
        <h3>{car.year} {car.make} {car.model}</h3>
        <div className={styles.carDetails}>
          <div className={styles.detail}>
            <span className={styles.label}>License Plate:</span>
            <span className={styles.value}>{car.licensePlate}</span>
          </div>
          <div className={styles.detail}>
            <span className={styles.label}>Color:</span>
            <span className={styles.value}>{car.color}</span>
          </div>
          {car.vin && (
            <div className={styles.detail}>
              <span className={styles.label}>VIN:</span>
              <span className={styles.value}>{car.vin}</span>
            </div>
          )}
          {car.mileage !== undefined && (
            <div className={styles.detail}>
              <span className={styles.label}>Mileage:</span>
              <span className={styles.value}>{car.mileage.toLocaleString()} miles</span>
            </div>
          )}
        </div>
      </div>*/}
      </div>
      
      {/* Footer with ID and actions */}
      <div className={styles.appointmentFooter}>
        <span className={styles.appointmentId}>
          ID: {appointment.id.slice(0, 8)}...
        </span>
        
        <div className={styles.appointmentActions}>
          {/* Actions based on appointment status */}
          {appointment.status === 'scheduled' && (
            <>
              <button 
                onClick={() => handleStatusChange('confirm')}
                className={styles.confirmButton}
                disabled={isUpdating}
              >
                {isUpdating ? 'Updating...' : 'Confirm'}
              </button>
              <button 
                onClick={onEdit}
                className={styles.editButton}
              >
                Edit
              </button>
              <button 
                onClick={() => handleStatusChange('cancel')}
                className={styles.cancelButton}
                disabled={isUpdating}
              >
                Cancel
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
                Complete
              </button>
              <button 
                onClick={() => handleStatusChange('cancel')}
                className={styles.cancelButton}
                disabled={isUpdating}
              >
                Cancel
              </button>
            </>
          )}
          
          {appointment.status === 'completed' && car && (
            <button 
              onClick={() => router.push(`/cars/${car.id}`)}
              className={styles.viewButton}
            >
              View Car
            </button>
          )}

          {appointment.status === 'cancelled' && (
            <button 
              onClick={onDelete}
              className={styles.rescheduleButton}
            >
              Reschedule
            </button>
          )}
        </div>
      </div>
    </div>
  );
}