// filepath: d:\Repos\GonsGarage\frontend\src\components\ui\Error\ErrorAlert.tsx
import React from 'react';
import styles from './Error.module.css';

interface ErrorAlertProps {
  message: string;
  onClose?: () => void;
}

// Error alert component following Agent.md error handling patterns
export default function ErrorAlert({ message, onClose }: ErrorAlertProps) {
  return (
    <div className={styles.errorAlert}>
      <div className={styles.errorIcon}>
        <svg fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
        </svg>
      </div>
      <span className={styles.errorMessage}>{message}</span>
      {onClose && (
        <button 
          onClick={onClose} 
          className={styles.closeButton}
          aria-label="Close error"
        >
          <svg fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      )}
    </div>
  );
}