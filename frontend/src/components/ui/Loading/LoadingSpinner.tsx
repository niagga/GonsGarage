import React from 'react';
import styles from './LoadingSpinner.module.css';

interface LoadingSpinnerProps {
  size?: 'sm' | 'md' | 'lg';
  className?: string;
}

// Loading spinner component following Agent.md UI conventions
export default function LoadingSpinner({ 
  size = 'md', 
  className = '' 
}: LoadingSpinnerProps) {
  return (
    <div className={`${styles.spinner} ${styles[size]} ${className}`}>
      <div className={styles.circle}></div>
    </div>
  );
}