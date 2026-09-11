'use client';

import React from 'react';
import Link from 'next/link';

interface EmptyCarStateProps {
  onAddCar?: () => void;
}

export default function EmptyCarState({ onAddCar }: EmptyCarStateProps) {
  return (
    <div className="empty-state">
      <div className="empty-state-content">
        <div className="empty-state-icon">
          🚗
        </div>
        <h3>Sem automóveis registados</h3>
        <p>
          Ainda não adicionou nenhum automóvel à sua conta.
          Adicione o primeiro para poder marcar visitas.
        </p>
        <div className="empty-state-actions">
          {onAddCar ? (
            <button 
              onClick={onAddCar}
              className="btn-primary"
            >
              Adicionar o primeiro automóvel
            </button>
          ) : (
            <Link href="/cars?addCar=1" className="btn-primary">
              Adicionar o primeiro automóvel
            </Link>
          )}
        </div>
      </div>
    </div>
  );
}
