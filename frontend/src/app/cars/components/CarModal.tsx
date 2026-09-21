import React, { useEffect, useState } from 'react';
import { Car, CreateCarRequest, CarFormData } from '@/types/car';
import { useCarValidation } from '@/hooks/useCarValidation';
import { useAuth, useCarStore } from '@/stores';
import { canManageUsers } from '@/types/user';
import { apiClient, type ClientLookupUser } from '@/lib/api-client';
import styles from '../cars.module.css';

type OwnerMode = 'existing' | 'new';

interface NewClientFormData {
  firstName: string;
  lastName: string;
  email: string;
  password: string;
}

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
  const { user } = useAuth();
  const canAssignOwner = !car && !!user && canManageUsers(user);

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
  const [ownerMode, setOwnerMode] = useState<OwnerMode>('existing');
  const [clients, setClients] = useState<ClientLookupUser[]>([]);
  const [selectedClientId, setSelectedClientId] = useState('');
  const [clientsLoading, setClientsLoading] = useState(false);
  const [newClientData, setNewClientData] = useState<NewClientFormData>({
    firstName: '',
    lastName: '',
    email: '',
    password: '',
  });

  const { errors, validateCar, clearFieldError, setGeneralError } = useCarValidation();

  useEffect(() => {
    if (!canAssignOwner) return;

    let active = true;
    const loadClients = async () => {
      setClientsLoading(true);
      const response = await apiClient.listClientUsers({ limit: 200, offset: 0 });
      if (!active) return;

      if (response.success && response.data?.items) {
        setClients(response.data.items);
      } else {
        setGeneralError(response.error?.message || 'Não foi possível carregar os clientes para associar a viatura.');
      }
      setClientsLoading(false);
    };

    void loadClients();

    return () => {
      active = false;
    };
  }, [canAssignOwner, setGeneralError]);

  // Handle form field changes - following Agent.md form handling
  const handleChange = (e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>) => {
    const { name, value } = e.target;

    setFormData(prev => ({
      ...prev,
      [name]: name === 'year' || name === 'mileage' ? parseInt(value, 10) || 0 : value
    }));

    // Clear field error when user starts typing
    if (errors[name as keyof typeof errors]) {
      clearFieldError(name as keyof typeof errors);
    }
  };

  const validateOwnerInput = (): boolean => {
    if (!canAssignOwner) return true;

    if (ownerMode === 'existing') {
      if (!selectedClientId) {
        setGeneralError('Selecionar um cliente existente é obrigatório para criar o automóvel.');
        return false;
      }
      return true;
    }

    const firstName = newClientData.firstName.trim();
    const lastName = newClientData.lastName.trim();
    const email = newClientData.email.trim();

    if (!firstName || !lastName || !email || !newClientData.password) {
      setGeneralError('Preenche nome, apelido, email e password para criar o novo cliente.');
      return false;
    }

    if (newClientData.password.length < 6) {
      setGeneralError('A password inicial do cliente tem de ter pelo menos 6 caracteres.');
      return false;
    }

    return true;
  };

  // Handle form submission - following Agent.md error handling
  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    clearFieldError('general');

    if (!validateCar(formData)) {
      return;
    }

    if (!validateOwnerInput()) {
      return;
    }

    setIsLoading(true);

    try {
      let ownerID: string | undefined;

      if (!car && canAssignOwner) {
        if (ownerMode === 'existing') {
          ownerID = selectedClientId;
        } else {
          const provisionResult = await apiClient.provisionUser({
            email: newClientData.email.trim(),
            password: newClientData.password,
            firstName: newClientData.firstName.trim(),
            lastName: newClientData.lastName.trim(),
            role: 'client',
          });

          if (!provisionResult.success || !provisionResult.data?.user) {
            setGeneralError(provisionResult.error?.message || 'Não foi possível criar o novo cliente.');
            return;
          }

          ownerID = provisionResult.data.user.id;
        }
      }

      const carData: CreateCarRequest = {
        make: formData.make,
        model: formData.model,
        year: formData.year,
        licensePlate: formData.licensePlate,
        vin: formData.vin || undefined,
        color: formData.color,
        mileage: formData.mileage || undefined,
        ownerID,
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
        const storeError = useCarStore.getState().error;
        setGeneralError(storeError || 'Não foi possível guardar o automóvel. Tente novamente.');
      }
    } catch (err) {
      setGeneralError(err instanceof Error ? err.message : 'Ocorreu um erro. Tente novamente.');
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
                backgroundColor: 'var(--chip-danger-bg)',
                border: '1px solid var(--chip-danger-border)',
                color: 'var(--chip-danger-fg)',
                padding: 'var(--space-3)',
                borderRadius: 'var(--radius)',
                marginBottom: 'var(--space-4)',
                fontSize: '0.875rem',
              }}>
                {errors.general}
              </div>
            )}

          {canAssignOwner && (
            <>
              <div className={styles.formGroup}>
                <label>Cliente proprietário *</label>
                <select
                  value={ownerMode}
                  onChange={(e) => setOwnerMode(e.target.value as OwnerMode)}
                  disabled={isLoading}
                >
                  <option value="existing">Associar cliente existente</option>
                  <option value="new">Criar novo cliente</option>
                </select>
              </div>

              {ownerMode === 'existing' ? (
                <div className={styles.formGroup}>
                  <label htmlFor="ownerID">Cliente existente *</label>
                  <select
                    id="ownerID"
                    value={selectedClientId}
                    onChange={(e) => setSelectedClientId(e.target.value)}
                    disabled={isLoading || clientsLoading}
                  >
                    <option value="">Selecionar cliente…</option>
                    {clients.map((client) => (
                      <option key={client.id} value={client.id}>
                        {client.firstName} {client.lastName} — {client.email}
                      </option>
                    ))}
                  </select>
                  {clientsLoading && <span className={styles.errorText}>A carregar clientes…</span>}
                  {!clientsLoading && clients.length === 0 && (
                    <span className={styles.errorText}>Não há clientes disponíveis. Cria um cliente novo.</span>
                  )}
                </div>
              ) : (
                <div className={styles.formGrid}>
                  <div className={styles.formGroup}>
                    <label htmlFor="newClientFirstName">Nome cliente *</label>
                    <input
                      id="newClientFirstName"
                      type="text"
                      value={newClientData.firstName}
                      onChange={(e) => setNewClientData((prev) => ({ ...prev, firstName: e.target.value }))}
                      disabled={isLoading}
                    />
                  </div>
                  <div className={styles.formGroup}>
                    <label htmlFor="newClientLastName">Apelido cliente *</label>
                    <input
                      id="newClientLastName"
                      type="text"
                      value={newClientData.lastName}
                      onChange={(e) => setNewClientData((prev) => ({ ...prev, lastName: e.target.value }))}
                      disabled={isLoading}
                    />
                  </div>
                  <div className={styles.formGroup}>
                    <label htmlFor="newClientEmail">Email cliente *</label>
                    <input
                      id="newClientEmail"
                      type="email"
                      value={newClientData.email}
                      onChange={(e) => setNewClientData((prev) => ({ ...prev, email: e.target.value }))}
                      disabled={isLoading}
                    />
                  </div>
                  <div className={styles.formGroup}>
                    <label htmlFor="newClientPassword">Password inicial *</label>
                    <input
                      id="newClientPassword"
                      type="password"
                      value={newClientData.password}
                      onChange={(e) => setNewClientData((prev) => ({ ...prev, password: e.target.value }))}
                      minLength={6}
                      disabled={isLoading}
                    />
                  </div>
                </div>
              )}
            </>
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
