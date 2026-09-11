'use client';

import React, { useState, useCallback } from 'react';
import { Appointment, CreateAppointmentRequest, UpdateAppointmentRequest } from '@/types/appointment';
import AppointmentList from '@/app/appointments/components/AppointmentList';
import AppointmentModal from './AppointmentModal';
import LoadingSpinner from '@/components/ui/Loading/LoadingSpinner';
import ErrorAlert from '@/components/ui/Error/ErrorAlert';
import ConfirmModal from '@/components/ui/Modal/ConfirmModal';
import styles from './AppointmentContainer.module.css';
import { useAppointments } from '@/stores';
import EmptyAppointmentState from '@/components/empty-states/EmptyAppointmentState';

interface AppointmentsContainerProps {
  // Optional callbacks for parent components
  onAddAppointment?: (appointment: Appointment) => void;
  onUpdateAppointment?: (appointments: Appointment[]) => void;
  onDeleteAppointment?: (appointmentId: string) => void;
  
  // UI customization
  maxAppointments?: number;
  showHeader?: boolean;
  headerTitle?: string;
  headerSubtitle?: string;
  addButtonText?: string;
  className?: string;
  
  // Layout control
  renderHeader?: () => React.ReactNode;
  renderEmptyState?: () => React.ReactNode;
}

export default function AppointmentsContainer({
  onAddAppointment,
  onUpdateAppointment,
  onDeleteAppointment,
  maxAppointments,
  showHeader = true,
  headerTitle = 'As suas marcações',
  headerSubtitle,
  addButtonText = 'Nova marcação',
  className = '',
  renderHeader,
  renderEmptyState,
}: AppointmentsContainerProps) {
  const { appointments, isLoading, error, createAppointment, updateAppointment, deleteAppointment, fetchAppointments } = useAppointments();

  // Modal states
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [editingAppointment, setEditingAppointment] = useState<Appointment | null>(null);
  const [isCreating, setIsCreating] = useState(false);
  
  // Delete confirmation state
  const [deleteConfirmation, setDeleteConfirmation] = useState<{
    appointmentId: string | null;
    clientName: string;
    isOpen: boolean;
    carId: string | null;
    carName: string;
  }>({
    appointmentId: null,
    clientName: '',
    isOpen: false,
    carId: null,
    carName: '',
  });

  // ✅ Create appointment handler
  const handleCreateAppointment = useCallback(async (appointmentData: CreateAppointmentRequest): Promise<boolean> => {
    if (maxAppointments && appointments.length >= maxAppointments) {
      alert(`Só pode registar até ${maxAppointments} marcações.`);
      return false;
    }

    setIsCreating(true);

    try {
      const success = await createAppointment(appointmentData);
      
      if (success) {
        await fetchAppointments();
        
        const newAppointment = appointments.find(appointment => 
          appointment.date === appointmentData.date &&
          appointment.time === appointmentData.time &&
          appointment.carId === appointmentData.carId
        );

        if (onAddAppointment && newAppointment) {
          onAddAppointment(newAppointment);
        }

        if (onUpdateAppointment) {
          onUpdateAppointment(appointments);
        }

        setShowCreateModal(false);
        return true;
      }

      return false;
    } catch (error) {
      console.error('Failed to create car:', error);
      return false;
    } finally {
      setIsCreating(false);
    }
  }, [createAppointment, fetchAppointments, appointments, maxAppointments, onAddAppointment, onUpdateAppointment]);

  // ✅ Update appointment handler
  const handleUpdateAppointment = useCallback(async (id: string, appointmentData: Partial<UpdateAppointmentRequest>): Promise<boolean> => {
    try {
      const success = await updateAppointment(id, appointmentData);
      
      if (success) {
        await fetchAppointments();
        
        if (onUpdateAppointment) {
          onUpdateAppointment(appointments);
        }

        setEditingAppointment(null);
        return true;
      }

      return false;
    } catch (error) {
      console.error('Failed to update car:', error);
      return false;
    }
  }, [updateAppointment, fetchAppointments, appointments, onUpdateAppointment]);

  // ✅ Delete appointment handler - opens confirmation modal
  const handleDeleteAppointment = useCallback((id: string) => {
    const appointment = appointments.find(a => a.id === id);
    if (!appointment) return;

    setDeleteConfirmation({
      appointmentId: id,
      clientName: appointment.clientName,
      isOpen: true,
      carId: id,
      carName: `${appointment.clientName} (${appointment.carId})`,
    });
  }, [appointments]);

  // ✅ Confirm deletion
  const confirmDelete = useCallback(async () => {
    const { appointmentId } = deleteConfirmation;
    if (!appointmentId) return;

    try {
      const success = await deleteAppointment(appointmentId);
      
      if (success) {
        if (onDeleteAppointment) {
          onDeleteAppointment(appointmentId);
        }

        if (onUpdateAppointment) {
          onUpdateAppointment(appointments.filter(appointment => appointment.id !== appointmentId));
        }
      } else {
        alert('Não foi possível eliminar a marcação. Tente novamente.');
      }
    } catch (error) {
      console.error('Failed to delete car:', error);
      alert('Ocorreu um erro ao eliminar a marcação.');
    } finally {
      setDeleteConfirmation({ isOpen: false, appointmentId: null, clientName: '', carId: null, carName: '' });
    }
  }, [deleteConfirmation, deleteAppointment, appointments, onDeleteAppointment, onUpdateAppointment]);

  // ✅ Cancel deletion
  const cancelDelete = useCallback(() => {
    setDeleteConfirmation({ isOpen: false, appointmentId: null, clientName: '', carId: null, carName: ''});
  }, []);

  // ✅ Edit appointment handler
  const handleEditAppointment = useCallback((appointment: Appointment) => {
    setEditingAppointment(appointment);
  }, []);

  // ✅ Close modal
  const handleModalClose = useCallback(() => {
    setShowCreateModal(false);
    setEditingAppointment(null);
  }, []);

  // ✅ Open create modal
  const handleAddAppointmentClick = useCallback(() => {
    if (maxAppointments && appointments.length >= maxAppointments) {
      alert(`Atingiu o limite máximo de ${maxAppointments} marcações.`);
      return;
    }
    setShowCreateModal(true);
  }, [appointments.length, maxAppointments]);

  // Loading state
  if (isLoading) {
    return (
      <div className="loading-container">
        <LoadingSpinner />
        <span>A carregar marcações…</span>
      </div>
    );
  }

  // Empty state
  if (!isLoading && appointments.length === 0) {
    return (
      <div className={`${styles.container} ${className}`}>
        {/* Show header even when empty */}
        {showHeader && (
          <div className={styles.header}>
            <div className={styles.headerContent}>
              <h2>{headerTitle}</h2>
              <p>Comece por marcar a manutenção do seu automóvel</p>
            </div>
            <button 
              onClick={handleAddAppointmentClick}
              className={styles.addButton}
            >
              <svg fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} 
                  d="M12 6v6m0 0v6m0-6h6m-6 0H6" />
              </svg>
              {addButtonText}
            </button>
          </div>
        )}
        
        {/* Empty state */}
        {renderEmptyState ? (
          renderEmptyState()
        ) : (
          <EmptyAppointmentState onSchedule={handleAddAppointmentClick} />
        )}

        {/* Modal still accessible from empty state */}
        {showCreateModal && (
          <AppointmentModal
            onClose={handleModalClose}
            onCreate={handleCreateAppointment}
          />
        )}
      </div>
    );
  }

  const canAddAppointments = !maxAppointments || appointments.length < maxAppointments;
  const computedSubtitle = headerSubtitle || (
    maxAppointments 
      ? `Gerir as suas marcações (${appointments.length}/${maxAppointments})`
      : 'Gerir as suas marcações'
  );

  return (
    <div className={`${styles.container} ${className}`}>
      {/* Custom header or default */}
      {renderHeader ? (
        renderHeader()
      ) : showHeader ? (
        <div className={styles.header}>
          <div className={styles.headerContent}>
            <h2>{headerTitle}</h2>
            <p>{computedSubtitle}</p>
          </div>
          <button 
            onClick={handleAddAppointmentClick}
            className={styles.addButton}
            disabled={isCreating || !canAddAppointments}
          >
            <svg fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} 
                d="M12 6v6m0 0v6m0-6h6m-6 0H6" />
            </svg>
            {isCreating ? 'A marcar…' : addButtonText}
          </button>
        </div>
      ) : null}

      {/* Max appointments warning */}
      {maxAppointments && appointments.length >= maxAppointments && (
        <div className={styles.maxAppointmentsMessage}>
          <span>⚠️ Limite máximo de marcações atingido ({appointments.length}/{maxAppointments})</span>
        </div>
      )}

      {/* Error display */}
      {error && <ErrorAlert message={error} />}

      {/* Appointment list */}
      <AppointmentList
        appointments={appointments}
        onEdit={handleEditAppointment}
        onDelete={handleDeleteAppointment}
      />

      {/* Appointment Modal */}
      {(showCreateModal || editingAppointment) && (
        <AppointmentModal
          appointment={editingAppointment}
          onClose={handleModalClose}
          onCreate={handleCreateAppointment}
          onUpdate={handleUpdateAppointment}
        />
      )}

      {/* Confirmation Modal */}
      <ConfirmModal
        isOpen={deleteConfirmation.isOpen}
        title="Eliminar marcação"
        message={`Tem a certeza de que pretende eliminar a marcação de ${deleteConfirmation.clientName}? Esta ação não pode ser anulada.`}
        confirmText="Eliminar"
        cancelText="Cancelar"
        variant="danger"
        onConfirm={confirmDelete}
        onCancel={cancelDelete}
      />
    </div>
  );
}