import React, { useState } from 'react';
import { Car, CreateCarRequest, CarFormData } from '@/types/car';
import { useCarValidation } from '@/hooks/useCarValidation';
import styles from '../cars.module.css';

interface CarModalProps {
  car: Car | null;
  onClose: () => void;
  onCreate: (carData: CreateCarRequest) => Promise<boolean>;
  onUpdate: (id: string, carData: Partial<CreateCarRequest>) => Promise<boolean>;
}

// Car modal component following Agent.md modal patterns
export default function CarModal({ 
  car, 
  onClose, 
  onCreate, 
  onUpdate 
}: CarModalProps) {
  const [formData, setFormData] = useState<CarFormData>({
    make: car?.make || '',
    model: car?.model || '',
    year: car?.year || new Date().getFullYear(),
    licensePlate: car?.licensePlate || '',
    vin: car?.vin || '',
    color: car?.color || '',
    mileage: car?.mileage || 0,
  });
  const [isLoading, setIsLoading] = useState(false);

  const { errors, validateCar, clearFieldError, setGeneralError } = useCarValidation();

  // Handle form field changes - following Agent.md form handling
  const handleChange = (e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>) => {
    const { name, value } = e.target;
    
    setFormData(prev => ({
      ...prev,
      [name]: name === 'year' || name === 'mileage' ? parseInt(value) || 0 : value
    }));

    // Clear field error when user starts typing
    if (errors[name as keyof typeof errors]) {
      clearFieldError(name as keyof typeof errors);
    }
  };

  // Handle form submission - following Agent.md error handling
  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!validateCar(formData)) {
      return;
    }

    setIsLoading(true);

    try {
      const carData: CreateCarRequest = {
        make: formData.make,
        model: formData.model,
        year: formData.year,
        licensePlate: formData.licensePlate,
        vin: formData.vin || undefined,
        color: formData.color,
        mileage: formData.mileage || undefined,
      };

      let success = false;

      if (car) {
        // Update existing car
        success = await onUpdate(car.id, carData);
      } else {
        // Create new car
        success = await onCreate(carData);
      }

      if (success) {
        onClose();
      } else {
        setGeneralError('Não foi possível guardar o automóvel. Tente novamente.');
      }
    } catch (err) {
      console.log(err);
      alert('Ocorreu um erro. Tente novamente.');
    } finally {
      setIsLoading(false);
    }
  };

  // Handle modal backdrop click
  const handleBackdropClick = (e: React.MouseEvent) => {
    if (e.target === e.currentTarget) {
      onClose();
    }
  };

  return (
    <div className={styles.modalOverlay} onClick={handleBackdropClick}>
      <div className={styles.modal}>
        <div className={styles.modalHeader}>
          <h3>{car ? 'Editar automóvel' : 'Novo automóvel'}</h3>
          <button 
            onClick={onClose} 
            className={styles.closeButton}
            aria-label="Fechar"
          >
            <svg fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        <form onSubmit={handleSubmit} className={styles.modalForm}>
            {errors.general && (
              <div style={{
                backgroundColor: '#fef2f2',
                border: '1px solid #fecaca',
                color: '#dc2626',
                padding: 'var(--space-3)',
                borderRadius: 'var(--radius)',
                marginBottom: 'var(--space-4)',
                fontSize: '0.875rem',
              }}>
                {errors.general}
              </div>
            )}
          <div className={styles.formGrid}>
            {/* Make Field */}
            <div className={styles.formGroup}>
              <label htmlFor="make">Marca *</label>
              <input
                id="make"
                name="make"
                type="text"
                value={formData.make}
                onChange={handleChange}
                placeholder="ex.: Toyota"
                className={errors.make ? styles.inputError : ''}
                disabled={isLoading}
              />
              {errors.make && <span className={styles.errorText}>{errors.make}</span>}
            </div>

            {/* Model Field */}
            <div className={styles.formGroup}>
              <label htmlFor="model">Modelo *</label>
              <input
                id="model"
                name="model"
                type="text"
                value={formData.model}
                onChange={handleChange}
                placeholder="ex.: Corolla"
                className={errors.model ? styles.inputError : ''}
                disabled={isLoading}
              />
              {errors.model && <span className={styles.errorText}>{errors.model}</span>}
            </div>

            {/* Year Field */}
            <div className={styles.formGroup}>
              <label htmlFor="year">Ano *</label>
              <input
                id="year"
                name="year"
                type="number"
                value={formData.year}
                onChange={handleChange}
                min="1900"
                max={new Date().getFullYear() + 2}
                className={errors.year ? styles.inputError : ''}
                disabled={isLoading}
              />
              {errors.year && <span className={styles.errorText}>{errors.year}</span>}
            </div>

            {/* Color Field */}
            <div className={styles.formGroup}>
              <label htmlFor="color">Cor *</label>
              <input
                id="color"
                name="color"
                type="text"
                value={formData.color}
                onChange={handleChange}
                placeholder="ex.: Azul"
                className={errors.color ? styles.inputError : ''}
                disabled={isLoading}
              />
              {errors.color && <span className={styles.errorText}>{errors.color}</span>}
            </div>
          </div>

          {/* License Plate Field */}
          <div className={styles.formGroup}>
            <label htmlFor="licensePlate">Matrícula *</label>
            <input
              id="licensePlate"
              name="licensePlate"
              type="text"
              value={formData.licensePlate}
              onChange={handleChange}
              placeholder="ex.: AA-12-BB"
              className={errors.licensePlate ? styles.inputError : ''}
              disabled={isLoading}
            />
            {errors.licensePlate && <span className={styles.errorText}>{errors.licensePlate}</span>}
          </div>

          {/* VIN Field */}
          <div className={styles.formGroup}>
            <label htmlFor="vin">VIN (opcional)</label>
            <input
              id="vin"
              name="vin"
              type="text"
              value={formData.vin}
              onChange={handleChange}
              placeholder="VIN com 17 caracteres"
              maxLength={17}
              disabled={isLoading}
            />
          </div>

          {/* Mileage Field */}
          <div className={styles.formGroup}>
            <label htmlFor="mileage">Quilometragem (opcional)</label>
            <input
              id="mileage"
              name="mileage"
              type="number"
              value={formData.mileage}
              onChange={handleChange}
              min="0"
              placeholder="Quilometragem atual"
              className={errors.mileage ? styles.inputError : ''}
              disabled={isLoading}
            />
            {errors.mileage && <span className={styles.errorText}>{errors.mileage}</span>}
          </div>

          {/* Modal Actions */}
          <div className={styles.modalActions}>
            <button
              type="button"
              onClick={onClose}
              className={styles.cancelButton}
              disabled={isLoading}
            >
              Cancelar
            </button>
            <button
              type="submit"
              disabled={isLoading}
              className={styles.submitButton}
            >
              {isLoading ? 'A guardar…' : (car ? 'Atualizar' : 'Adicionar')}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}