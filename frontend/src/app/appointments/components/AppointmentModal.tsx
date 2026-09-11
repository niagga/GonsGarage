'use client';

import React, { useEffect, useMemo, useState } from 'react';
import { Appointment, CreateAppointmentRequest, UpdateAppointmentRequest } from '@/types/appointment';
import { SERVICE_TYPES } from '@/shared/types';
import {
  combineLocalDateAndSlot,
  formatSlotLabel24h,
  getBookableSlotTimesForDay,
  isWeekdayLocalYmd,
  isWithinWorkshopHours,
  localYmdFromIso,
  todayLocalYmd,
  weekdayErrorMessage,
  workshopHoursErrorMessage,
  workshopSlotFromIso,
} from '@/lib/workshopAppointmentRules';
import { useCarStore } from '@/stores/car.store';
import styles from '../appointments.module.css';

interface AppointmentModalProps {
  appointment?: Appointment | null;
  onClose: () => void;
  onCreate: (data: CreateAppointmentRequest) => Promise<boolean>;
  onUpdate?: (id: string, data: Partial<UpdateAppointmentRequest>) => Promise<boolean>;
  preSelectedCarId?: string;
}

interface FormData {
  carId: string;
  service: string;
  date: string;
  time: string;
  notes: string;
}

export default function AppointmentModal({
  appointment,
  onClose,
  onCreate,
  onUpdate,
  preSelectedCarId,
}: AppointmentModalProps) {
  const { cars } = useCarStore();
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string>('');

  const [formData, setFormData] = useState<FormData>({
    carId: appointment?.carId || preSelectedCarId || '',
    service: appointment?.service || '',
    date: appointment?.date ? localYmdFromIso(appointment.date) : '',
    time: appointment?.date ? workshopSlotFromIso(appointment.date) : '',
    notes: appointment?.notes || '',
  });

  const bookableSlots = useMemo(() => getBookableSlotTimesForDay(formData.date), [formData.date]);

  useEffect(() => {
    if (!formData.time) return;
    if (bookableSlots.includes(formData.time)) return;
    let cancelled = false;
    queueMicrotask(() => {
      if (cancelled) return;
      setFormData((prev) => ({ ...prev, time: '' }));
    });
    return () => {
      cancelled = true;
    };
  }, [formData.date, bookableSlots, formData.time]);

  const selectedCar = cars.find(c => c.id === formData.carId);

  const handleChange = (
    e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement | HTMLTextAreaElement>
  ) => {
    const { name, value } = e.target;
    setFormData(prev => ({ ...prev, [name]: value }));
    setError('');
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    // Validation
    if (!formData.carId) {
      setError('Selecione um automóvel');
      return;
    }

    if (!formData.service) {
      setError('Selecione um serviço');
      return;
    }

    if (!formData.date || !formData.time) {
      setError('Selecione data e hora');
      return;
    }
    if (!isWeekdayLocalYmd(formData.date)) {
      setError(weekdayErrorMessage());
      return;
    }
    if (bookableSlots.length === 0) {
      setError('Não há horários disponíveis para esse dia.');
      return;
    }
    const scheduledLocal = combineLocalDateAndSlot(formData.date, formData.time);
    const scheduledDate = new Date(scheduledLocal);
    if (Number.isNaN(scheduledDate.getTime()) || scheduledDate <= new Date()) {
      setError('A marcação tem de ser no futuro');
      return;
    }
    if (!isWithinWorkshopHours(scheduledDate)) {
      setError(workshopHoursErrorMessage());
      return;
    }

    setIsLoading(true);

    try {
      const appointmentData: CreateAppointmentRequest = {
        clientName: selectedCar ? `${selectedCar.make} ${selectedCar.model}` : 'Desconhecido',
        carId: formData.carId,
        service: formData.service,
        date: scheduledLocal,
        notes: formData.notes || undefined,
        status: 'scheduled',
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString(),
        time: formData.time,
      };

      let success = false;

      if (appointment && onUpdate) {
        const appointmentUpdateData: UpdateAppointmentRequest = {
              carId: formData.carId,
              service: formData.service,
              date: scheduledLocal,
              notes: formData.notes || undefined,
              status: 'scheduled',
            };

        success = await onUpdate(appointment.id, appointmentUpdateData);
      } else {
        success = await onCreate(appointmentData);
      }

      if (success) {
        onClose();
      } else {
        setError('Não foi possível guardar a marcação. Tente novamente.');
      }
    } catch (err) {
      console.error('Error saving appointment:', err);
      setError('Ocorreu um erro. Tente novamente.');
    } finally {
      setIsLoading(false);
    }
  };

  const handleBackdropClick = (e: React.MouseEvent) => {
    if (e.target === e.currentTarget) {
      onClose();
    }
  };

  return (
    <div className={styles.modalOverlay} onClick={handleBackdropClick}>
      <div
        className={styles.modal}
        role="dialog"
        aria-modal="true"
        aria-labelledby="appointment-modal-title"
      >
        {/* Header */}
        <div className={styles.modalHeader}>
          <h3 id="appointment-modal-title">{appointment ? 'Editar marcação' : 'Marcar visita'}</h3>
          <button
            type="button"
            onClick={onClose}
            className={styles.closeButton}
            aria-label="Fechar"
          >
            <svg fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M6 18L18 6M6 6l12 12"
              />
            </svg>
          </button>
        </div>

        {/* Body */}
        <form onSubmit={handleSubmit}>
          <div className={styles.modalBody}>
            {error && (
              <div className={styles.errorAlert}>
                {error}
              </div>
            )}

            {/* Car Selection */}
            <div className={styles.section}>
              <h3>Selecionar viatura</h3>
              {selectedCar ? (
                <div className={styles.selectedCarInfo}>
                  <div className={styles.carIcon}>🚗</div>
                  <div className={styles.carDetails}>
                    <h4>{selectedCar.year} {selectedCar.make} {selectedCar.model}</h4>
                    <p>{selectedCar.licensePlate} • {selectedCar.color}</p>
                  </div>
                  {!appointment && (
                    <button
                      type="button"
                      onClick={() => setFormData(prev => ({ ...prev, carId: '' }))}
                      className={styles.changeCarButton}
                    >
                      Alterar
                    </button>
                  )}
                </div>
              ) : (
                <div className={styles.formGroup}>
                  <select
                    name="carId"
                    value={formData.carId}
                    onChange={handleChange}
                    required
                    disabled={isLoading}
                  >
                    <option value="">Selecione um automóvel…</option>
                    {cars.map(car => (
                      <option key={car.id} value={car.id}>
                        {car.year} {car.make} {car.model} - {car.licensePlate}
                      </option>
                    ))}
                  </select>
                </div>
              )}
            </div>

            {/* Service Selection */}
            <div className={styles.section}>
              <h3>Selecionar serviço</h3>
              <div className={styles.serviceGrid}>
                {SERVICE_TYPES.map(service => (
                  <label key={service.id} className={styles.serviceOption}>
                    <input
                      type="radio"
                      name="service"
                      value={service.id}
                      checked={formData.service === service.id}
                      onChange={handleChange}
                      disabled={isLoading}
                    />
                    <div className={styles.serviceCard}>
                      <h4>{service.name}</h4>
                      <p>{service.description}</p>
                    </div>
                  </label>
                ))}
              </div>
            </div>

            {/* Date & Time */}
            <div className={styles.section}>
              <h3>Data e hora</h3>
              <p className={styles.helpText}>
                Segunda a sexta, de 30 em 30 minutos: 9:30–12:30 e 14:00–17:30 (formato 24 h).
              </p>
              <div className={styles.formGrid}>
                <div className={styles.formGroup}>
                  <label htmlFor="date">Dia *</label>
                  <input
                    id="date"
                    name="date"
                    type="date"
                    value={formData.date}
                    onChange={handleChange}
                    min={todayLocalYmd()}
                    required
                    disabled={isLoading}
                  />
                </div>
                <div className={styles.formGroup}>
                  <label htmlFor="time">Hora *</label>
                  <select
                    id="time"
                    name="time"
                    value={formData.time}
                    onChange={handleChange}
                    required
                    disabled={isLoading || !formData.date || bookableSlots.length === 0}
                  >
                    <option value="">
                      {!formData.date
                        ? 'Escolha primeiro um dia…'
                        : bookableSlots.length === 0
                          ? 'Sem horários nesse dia'
                          : 'Escolha a hora…'}
                    </option>
                    {bookableSlots.map((hhmm) => (
                      <option key={hhmm} value={hhmm}>
                        {formatSlotLabel24h(hhmm)}
                      </option>
                    ))}
                  </select>
                </div>
              </div>
            </div>

            {/* Notes */}
            <div className={styles.section}>
              <h3>Notas adicionais (opcional)</h3>
              <div className={styles.formGroup}>
                <textarea
                  name="notes"
                  value={formData.notes}
                  onChange={handleChange}
                  placeholder="Observações ou pedidos específicos…"
                  rows={3}
                  disabled={isLoading}
                />
              </div>
            </div>

            {/* Summary */}
            {formData.carId && formData.service && formData.date && formData.time && (
              <div className={styles.appointmentSummary}>
                <h3>Resumo da marcação</h3>
                <div className={styles.summaryGrid}>
                  <div className={styles.summaryItem}>
                    <span className={styles.summaryLabel}>Viatura:</span>
                    <span className={styles.summaryValue}>
                      {selectedCar && `${selectedCar.year} ${selectedCar.make} ${selectedCar.model}`}
                    </span>
                  </div>
                  <div className={styles.summaryItem}>
                    <span className={styles.summaryLabel}>Serviço:</span>
                    <span className={styles.summaryValue}>
                      {SERVICE_TYPES.find(s => s.id === formData.service)?.name}
                    </span>
                  </div>
                  <div className={styles.summaryItem}>
                    <span className={styles.summaryLabel}>Dia:</span>
                    <span className={styles.summaryValue}>
                      {new Date(formData.date).toLocaleDateString('pt-PT', {
                        weekday: 'short',
                        year: 'numeric',
                        month: 'short',
                        day: 'numeric',
                      })}
                    </span>
                  </div>
                  <div className={styles.summaryItem}>
                    <span className={styles.summaryLabel}>Hora:</span>
                    <span className={styles.summaryValue}>{formatSlotLabel24h(formData.time)}</span>
                  </div>
                </div>
              </div>
            )}
          </div>

          {/* Footer Actions */}
          <div className={styles.modalActions}>
            <button
              type="button"
              onClick={onClose}
              className={styles.modalCancelButton}
              disabled={isLoading}
            >
              Cancelar
            </button>
            <button
              type="submit"
              className={styles.modalSubmitButton}
              disabled={isLoading}
            >
              {isLoading
                ? 'A guardar…'
                : appointment
                ? 'Atualizar marcação'
                : 'Confirmar marcação'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}