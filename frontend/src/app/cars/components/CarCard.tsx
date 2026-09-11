import React from 'react';
import { Car } from '@/types/car';
import styles from '../cars.module.css';

interface CarCardProps {
  car: Car;
  onEdit: () => void;
  onDelete: () => void;
  onViewDetails: () => void;
  onScheduleService: () => void;
}

// Individual car card component following Agent.md clean patterns
export default function CarCard({
  car,
  onEdit,
  onDelete,
  onViewDetails,
  onScheduleService
}: CarCardProps) {
  return (
    <div className={styles.carCard}>
      {/* Car Header */}
      <div className={styles.carHeader}>
        <div className={styles.carIcon}>🚗</div>
        <div className={styles.carActions}>
          <button 
            onClick={onEdit}
            className={styles.editButton}
            title="Editar automóvel"
            aria-label={`Editar ${car.year} ${car.make} ${car.model}`}
          >
            <svg fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
            </svg>
          </button>
          <button 
            onClick={onDelete}
            className={styles.deleteButton}
            title="Eliminar automóvel"
            aria-label={`Eliminar ${car.year} ${car.make} ${car.model}`}
          >
            <svg fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
            </svg>
          </button>
        </div>
      </div>

      {/* Car Content */}
      <div className={styles.carContent}>
        <h3>{car.year} {car.make} {car.model}</h3>
        <div className={styles.carDetails}>
          <div className={styles.detail}>
            <span className={styles.label}>Matrícula:</span>
            <span className={styles.value}>{car.licensePlate}</span>
          </div>
          <div className={styles.detail}>
            <span className={styles.label}>Cor:</span>
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
              <span className={styles.label}>Quilometragem:</span>
              <span className={styles.value}>{car.mileage.toLocaleString('pt-PT')} km</span>
            </div>
          )}
        </div>
      </div>

      {/* Car Footer */}
      <div className={styles.carFooter}>
        <button 
          onClick={onViewDetails}
          className={styles.viewDetailsButton}
        >
          Ver detalhes e reparações
        </button>
        <button 
          onClick={onScheduleService}
          className={styles.scheduleButton}
        >
          Marcar serviço
        </button>
      </div>
    </div>
  );
}