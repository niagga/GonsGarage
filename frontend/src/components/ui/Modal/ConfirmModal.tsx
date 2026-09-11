'use client';

import React from 'react';
import styles from './ConfirmModal.module.css';

interface ConfirmModalProps {
  isOpen: boolean;
  title: string;
  message: string;
  confirmText?: string;
  cancelText?: string;
  variant?: 'danger' | 'warning' | 'info';
  onConfirm: () => void;
  onCancel: () => void;
}

export default function ConfirmModal({
  isOpen,
  title,
  message,
  confirmText = 'Confirm',
  cancelText = 'Cancel',
  variant = 'warning',
  onConfirm,
  onCancel,
}: ConfirmModalProps) {
  if (!isOpen) return null;

  return (
    <div className={styles.overlay} onClick={onCancel}>
      <div 
        className={styles.modal} 
        onClick={(e) => e.stopPropagation()}
      >
        {/* Header */}
        <div className={`${styles.header} ${styles[variant]}`}>
          <div className={styles.iconWrapper}>
            {variant === 'danger' && (
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} 
                  d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
              </svg>
            )}
            {variant === 'warning' && (
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} 
                  d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
            )}
            {variant === 'info' && (
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} 
                  d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
            )}
          </div>
          <h3>{title}</h3>
        </div>

        {/* Body */}
        <div className={styles.body}>
          <p>{message}</p>
        </div>

        {/* Footer */}
        <div className={styles.footer}>
          <button 
            onClick={onCancel}
            className={styles.cancelButton}
          >
            {cancelText}
          </button>
          <button 
            onClick={onConfirm}
            className={`${styles.confirmButton} ${styles[variant]}`}
          >
            {confirmText}
          </button>
        </div>
      </div>
    </div>
  );
}