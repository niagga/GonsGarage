'use client';

import React from 'react';
import styles from './EmptyState.module.css';

interface EmptyAppointmentStateProps {
  onSchedule?: () => void;
}

export default function EmptyAppointmentState({ onSchedule }: EmptyAppointmentStateProps) {
  return (
    <div className={styles.emptyState}>
      <div className={styles.emptyStateIcon}>
        📅
      </div>
      <h3 className={styles.emptyStateTitle}>Ainda sem marcações</h3>
      <p className={styles.emptyStateDescription}>
        Ainda não marcou nenhum serviço. Marque a primeira visita para manter o seu automóvel em boas condições.
      </p>
      {onSchedule && (
        <button onClick={onSchedule} className={styles.emptyStateButton}>
          <svg fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} 
              d="M12 6v6m0 0v6m0-6h6m-6 0H6" />
          </svg>
          Marcar primeira visita
        </button>
      )}
    </div>
  );
}
