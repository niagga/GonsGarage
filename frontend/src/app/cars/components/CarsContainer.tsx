'use client';

import React, { useState, useCallback, useEffect } from 'react';
import { Car, CreateCarRequest } from '@/types/car';
import { useCars, useCarStore } from '@/stores';
import CarList from '@/app/cars/components/CarList';
import CarModal from '@/app/cars/components/CarModal';
import LoadingSpinner from '@/components/ui/Loading/LoadingSpinner';
import ErrorAlert from '@/components/ui/Error/ErrorAlert';
import ConfirmModal from '@/components/ui/Modal/ConfirmModal';
import { Button } from '@/components/ui/button';
import EmptyCarState from '@/components/empty-states/EmptyCarState';
import { useRouter, useSearchParams } from 'next/navigation';
import styles from './CarsContainer.module.css'; // ✅ Import styles

interface CarsContainerProps {
  // Optional callbacks for parent components
  onAddCar?: (car: Car) => void;
  onUpdateCar?: (cars: Car[]) => void;
  onDeleteCar?: (carId: string) => void;
  
  // UI customization
  maxCars?: number;
  showHeader?: boolean;
  headerTitle?: string;
  headerSubtitle?: string;
  addButtonText?: string;
  className?: string;
  
  // Layout control
  renderHeader?: () => React.ReactNode;
  renderEmptyState?: () => React.ReactNode;
}

export default function CarsContainer({
  onAddCar,
  onUpdateCar,
  onDeleteCar,
  maxCars,
  showHeader = true,
  headerTitle = 'Os seus automóveis',
  headerSubtitle,
  addButtonText = 'Novo automóvel',
  className = '',
  renderHeader,
  renderEmptyState,
}: CarsContainerProps) {
  const router = useRouter();
  const searchParams = useSearchParams();
  const { cars, isLoading, error, createCar, updateCar, deleteCar, fetchCars } = useCars();

  // Modal states
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [editingCar, setEditingCar] = useState<Car | null>(null);
  const [isCreating, setIsCreating] = useState(false);
  
  // Delete confirmation state
  const [deleteConfirmation, setDeleteConfirmation] = useState<{
    isOpen: boolean;
    carId: string | null;
    carName: string;
  }>({
    isOpen: false,
    carId: null,
    carName: '',
  });

  // ✅ Create car handler
  const handleCreateCar = useCallback(async (carData: CreateCarRequest): Promise<boolean> => {
    if (maxCars && cars.length >= maxCars) {
      alert(`Só pode registar até ${maxCars} automóveis.`);
      return false;
    }

    setIsCreating(true);

    try {
      const success = await createCar(carData);
      
      if (success) {
        await fetchCars();

        const freshCars = useCarStore.getState().cars;
        const newCar = freshCars.find(
          (car) =>
            car.make === carData.make &&
            car.model === carData.model &&
            car.licensePlate === carData.licensePlate,
        );

        if (onAddCar && newCar) {
          onAddCar(newCar);
        }

        if (onUpdateCar) {
          onUpdateCar(freshCars);
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
  }, [cars.length, createCar, fetchCars, maxCars, onAddCar, onUpdateCar]);

  // ✅ Update car handler
  const handleUpdateCar = useCallback(async (id: string, carData: Partial<CreateCarRequest>): Promise<boolean> => {
    try {
      const success = await updateCar(id, carData);
      
      if (success) {
        await fetchCars();

        if (onUpdateCar) {
          onUpdateCar(useCarStore.getState().cars);
        }

        setEditingCar(null);
        return true;
      }

      return false;
    } catch (error) {
      console.error('Failed to update car:', error);
      return false;
    }
  }, [updateCar, fetchCars, onUpdateCar]);

  // ✅ Delete car handler - opens confirmation modal
  const handleDeleteCar = useCallback((id: string) => {
    const car = cars.find(c => c.id === id);
    if (!car) return;

    setDeleteConfirmation({
      isOpen: true,
      carId: id,
      carName: `${car.make} ${car.model} (${car.licensePlate})`,
    });
  }, [cars]);

  // ✅ Confirm deletion
  const confirmDelete = useCallback(async () => {
    const { carId } = deleteConfirmation;
    if (!carId) return;

    try {
      const success = await deleteCar(carId);
      
      if (success) {
        if (onDeleteCar) {
          onDeleteCar(carId);
        }

        if (onUpdateCar) {
          onUpdateCar(useCarStore.getState().cars.filter((car) => car.id !== carId));
        }
      } else {
        alert('Não foi possível eliminar o automóvel. Tente novamente.');
      }
    } catch (error) {
      console.error('Failed to delete car:', error);
      alert('Ocorreu um erro ao eliminar o automóvel.');
    } finally {
      setDeleteConfirmation({ isOpen: false, carId: null, carName: '' });
    }
  }, [deleteConfirmation, deleteCar, onDeleteCar, onUpdateCar]);

  // ✅ Cancel deletion
  const cancelDelete = useCallback(() => {
    setDeleteConfirmation({ isOpen: false, carId: null, carName: '' });
  }, []);

  // ✅ Edit car handler
  const handleEditCar = useCallback((car: Car) => {
    setEditingCar(car);
  }, []);

  // ✅ Close modal
  const handleModalClose = useCallback(() => {
    setShowCreateModal(false);
    setEditingCar(null);
  }, []);

  // ✅ Open create modal
  const handleAddCarClick = useCallback(() => {
    if (maxCars && cars.length >= maxCars) {
      alert(`You have reached the maximum limit of ${maxCars} cars.`);
      return;
    }
    setShowCreateModal(true);
  }, [cars.length, maxCars]);

  useEffect(() => {
    if (isLoading) return;
    if (searchParams.get('addCar') !== '1') return;
    if (maxCars && cars.length >= maxCars) {
      router.replace('/cars', { scroll: false });
      return;
    }
    let cancelled = false;
    queueMicrotask(() => {
      if (cancelled) return;
      setShowCreateModal(true);
      router.replace('/cars', { scroll: false });
    });
    return () => {
      cancelled = true;
    };
  }, [isLoading, searchParams, router, maxCars, cars.length]);

  // ✅ Loading state
  if (isLoading) {
    return (
      <div className={styles.loadingContainer}>
        <LoadingSpinner />
        <span>A carregar automóveis…</span>
      </div>
    );
  }

  // ✅ Empty state (still render modals so “Add car” / ?addCar=1 works)
  if (!isLoading && cars.length === 0) {
    return (
      <div className={styles.emptyStateWrapper}>
        {renderEmptyState ? renderEmptyState() : <EmptyCarState onAddCar={handleAddCarClick} />}
        {(showCreateModal || editingCar) && (
          <CarModal
            car={editingCar}
            onClose={handleModalClose}
            onCreate={handleCreateCar}
            onUpdate={handleUpdateCar}
          />
        )}
        <ConfirmModal
          isOpen={deleteConfirmation.isOpen}
          title="Eliminar automóvel"
          message={`Tem a certeza de que pretende eliminar ${deleteConfirmation.carName}? Esta ação não pode ser anulada.`}
          confirmText="Eliminar"
          cancelText="Cancelar"
          variant="danger"
          onConfirm={confirmDelete}
          onCancel={cancelDelete}
        />
      </div>
    );
  }

  const canAddCars = !maxCars || cars.length < maxCars;
  const computedSubtitle = headerSubtitle || (
    maxCars 
      ? `Gerir os seus automóveis (${cars.length}/${maxCars})`
      : 'Gerir os seus automóveis registados'
  );

  return (
    <div className={`${styles.container} ${className}`}>
      {/* ✅ Custom header or default */}
      {renderHeader ? (
        renderHeader()
      ) : showHeader ? (
        <div className={styles.header}>
          <div className={styles.headerContent}>
            <h2>{headerTitle}</h2>
            <p>{computedSubtitle}</p>
          </div>
          <Button
            type="button"
            onClick={handleAddCarClick}
            className={styles.addButton}
            disabled={isCreating || !canAddCars}
          >
            <svg fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden>
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 6v6m0 0v6m0-6h6m-6 0H6" />
            </svg>
            {isCreating ? 'A adicionar…' : addButtonText}
          </Button>
        </div>
      ) : null}

      {/* ✅ Max cars warning */}
      {maxCars && cars.length >= maxCars && (
        <div className={styles.maxCarsMessage}>
          <span>⚠️ Limite máximo de automóveis atingido ({cars.length}/{maxCars})</span>
        </div>
      )}

      {/* ✅ Error display */}
      {error && <ErrorAlert message={error} />}

      {/* ✅ Car list */}
      <CarList
        cars={cars}
        onEdit={handleEditCar}
        onDelete={handleDeleteCar}
        onViewDetails={(id) => router.push(`/cars/${id}`)}
        onScheduleService={(id) =>
          router.push(`/appointments?schedule=1&carId=${encodeURIComponent(id)}`)
        }
      />

      {/* ✅ Car Modal */}
      {(showCreateModal || editingCar) && (
        <CarModal
          car={editingCar}
          onClose={handleModalClose}
          onCreate={handleCreateCar}
          onUpdate={handleUpdateCar}
        />
      )}

      {/* ✅ Confirmation Modal */}
      <ConfirmModal
        isOpen={deleteConfirmation.isOpen}
        title="Eliminar automóvel"
        message={`Tem a certeza de que pretende eliminar ${deleteConfirmation.carName}? Esta ação não pode ser anulada.`}
        confirmText="Eliminar"
        cancelText="Cancelar"
        variant="danger"
        onConfirm={confirmDelete}
        onCancel={cancelDelete}
      />
    </div>
  );
}