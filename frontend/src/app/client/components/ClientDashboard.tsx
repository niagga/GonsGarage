// src/app/client/components/ClientDashboard.tsx
'use client';

import React from 'react';
import { Button } from '@/components/ui/button';
import { cn } from '@/lib/utils';
import { ClientDashboardProps } from '../types';
import styles from '../client.module.css';

export default function ClientDashboard({ 
  error, 
  cars, 
  recentRepairs, 
  upcomingAppointments,
  onNavigate 
}: ClientDashboardProps) {
  return (
    <div>
      {/* Error Alert */}
      {error && (
        <div className={styles.errorAlert}>
          <span>{error}</span>
        </div>
      )}

      {/* Stats Grid */}
      <div className={styles.statsGrid}>
        <div className={styles.statCard}>
          <div className={styles.statIcon}>
            <svg fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 21V5a2 2 0 00-2-2H7a2 2 0 00-2 2v16m14 0h2m-2 0h-2m-2 0H7m5 0v-5a2 2 0 012-2h2a2 2 0 012 2v5" />
            </svg>
          </div>
          <div>
            <h3>My Cars</h3>
            <p className={styles.statNumber}>{cars.length}</p>
          </div>
        </div>

        <div className={styles.statCard}>
          <div className={styles.statIcon}>
            <svg fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
            </svg>
          </div>
          <div>
            <h3>Active Repairs</h3>
            <p className={styles.statNumber}>
              {recentRepairs.filter(r => r.status === 'in_progress').length}
            </p>
          </div>
        </div>

        <div className={styles.statCard}>
          <div className={styles.statIcon}>
            <svg fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
            </svg>
          </div>
          <div>
            <h3>Upcoming Appointments</h3>
            <p className={styles.statNumber}>{upcomingAppointments.length}</p>
          </div>
        </div>
      </div>

      {/* Content Grid */}
      <div className={styles.contentGrid}>
        {/* My Cars Section */}
        <div className={styles.card}>
          <div className={styles.cardHeader}>
            <h3>My Cars</h3>
            <Button
              type="button"
              variant="link"
              className={cn('h-auto p-0 font-medium', styles.linkButton)}
              onClick={() => onNavigate('cars')}
            >
              View All
            </Button>
          </div>
          <div className={styles.cardBody}>
            {cars.length === 0 ? (
              <div className={styles.emptyState}>
                <p>No cars registered yet</p>
                <Button type="button" onClick={() => onNavigate('cars')}>
                  Add Your First Car
                </Button>
              </div>
            ) : (
              <div className={styles.carsList}>
                {cars.slice(0, 3).map((car) => (
                  <div key={car.id} className={styles.carItem}>
                    <div className={styles.carIcon}>
                      🚗
                    </div>
                    <div className={styles.carInfo}>
                      <h4>{car.year} {car.make} {car.model}</h4>
                      <p>{car.licensePlate}</p>
                    </div>
                    <Button
                      type="button"
                      variant="outline"
                      size="sm"
                      className={styles.viewButton}
                      onClick={() => onNavigate('cars')}
                    >
                      View
                    </Button>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>

        {/* Recent Repairs Section */}
        <div className={styles.card}>
          <div className={styles.cardHeader}>
            <h3>Recent Repairs</h3>
            <Button
              type="button"
              variant="link"
              className={cn('h-auto p-0 font-medium', styles.linkButton)}
              onClick={() => onNavigate('appointments')}
            >
              View All
            </Button>
          </div>
          <div className={styles.cardBody}>
            {recentRepairs.length === 0 ? (
              <div className={styles.emptyState}>
                <p>No repairs yet</p>
              </div>
            ) : (
              <div className={styles.repairsList}>
                {recentRepairs.slice(0, 3).map((repair) => (
                  <div key={repair.id} className={styles.repairItem}>
                    <div className={styles.repairInfo}>
                      <h4>{repair.description}</h4>
                      <p>${repair.cost.toFixed(2)}</p>
                    </div>
                    <div className={styles.repairStatus}>
                      <span className={`${styles.statusBadge} ${styles[repair.status]}`}>
                        {repair.status.replace('_', ' ')}
                      </span>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>
      </div>

      {/* Recent Activity */}
      <div className={styles.card}>
        <div className={styles.cardHeader}>
          <h3>Recent Activity</h3>
        </div>
        <div className={styles.cardBody}>
          <p>Your recent activities will appear here...</p>
        </div>
      </div>
    </div>
  );
}