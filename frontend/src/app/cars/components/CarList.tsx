import React from 'react';
import { Car } from '@/types/car';
import CarCard from './CarCard';
import styles from '../cars.module.css';

interface CarListProps {
  cars: Car[];
  onEdit: (car: Car) => void;
  onDelete: (id: string) => void;
  onViewDetails: (id: string) => void;
  onScheduleService: (id: string) => void;
}

// Car list component following Agent.md component conventions
export default function CarList({
  cars,
  onEdit,
  onDelete,
  onViewDetails,
  onScheduleService
}: CarListProps) {
  // Empty state - following Agent.md user experience
  if (cars.length === 0) {
    return (
      <div className={styles.emptyState}>
        <div className={styles.emptyIcon}>🚗</div>
        <h3>Ainda sem automóveis registados</h3>
        <p>Adicione o primeiro automóvel para começar a usar os nossos serviços</p>
      </div>
    );
  }

  return (
    <div className={styles.carsGrid}>
      {cars.map((car) => (
        <CarCard
          key={car.id}
          car={car}
          onEdit={() => onEdit(car)}
          onDelete={() => onDelete(car.id)}
          onViewDetails={() => onViewDetails(car.id)}
          onScheduleService={() => onScheduleService(car.id)}
        />
      ))}
    </div>
  );
}