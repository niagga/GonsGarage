import { useState, useCallback } from 'react';
import { CarFormData, CarValidationErrors } from '@/types/car';

// Validation hook following Agent.md input validation principles
export function useCarValidation() {
  const [errors, setErrors] = useState<CarValidationErrors>({});

  // Validate car form data - following Agent.md validation rules
  const validateCar = useCallback((formData: CarFormData): boolean => {
    const newErrors: CarValidationErrors = {};

    // Required field validation
    if (!formData.make.trim()) {
      newErrors.make = 'A marca é obrigatória';
    }

    if (!formData.model.trim()) {
      newErrors.model = 'O modelo é obrigatório';
    }

    if (!formData.licensePlate.trim()) {
      newErrors.licensePlate = 'A matrícula é obrigatória';
    } else if (formData.licensePlate.length < 3) {
      newErrors.licensePlate = 'License plate must be at least 3 characters';
    }

    if (!formData.color.trim()) {
      newErrors.color = 'A cor é obrigatória';
    }

    // Year validation - following Agent.md business rules
    const currentYear = new Date().getFullYear();
    if (!formData.year || formData.year < 1900 || formData.year > currentYear + 2) {
      newErrors.year = `O ano deve estar entre 1900 e ${currentYear + 2}`;
    }

    // Mileage validation
    if (formData.mileage < 0) {
      newErrors.mileage = 'A quilometragem não pode ser negativa';
    }

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  }, []);

  // Clear specific field error
  const clearFieldError = useCallback((fieldName: keyof CarValidationErrors) => {
    setErrors(prev => {
      const newErrors = { ...prev };
      delete newErrors[fieldName];
      return newErrors;
    });
  }, []);

  // Clear all errors
  const clearErrors = useCallback(() => {
    setErrors({});
  }, []);

  const setGeneralError = useCallback((message: string) => {
    setErrors((prev) => ({ ...prev, general: message }));
  }, []);

  return {
    errors,
    validateCar,
    clearFieldError,
    clearErrors,
    setGeneralError,
  };
}