'use client';

import React, { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { useAuth, useCars, useAppointments, UserRole } from '@/stores';
import { apiClient, Repair } from '@/lib/api';
import { useAuthHydrationReady } from '@/hooks/useAuthHydrationReady';
import AppShell from '@/components/layout/AppShell';
import { AppLoading } from '@/components/ui/AppLoading';
import { Button } from '@/components/ui/button';
import { StaffUsersDashboardCta } from './StaffUsersDashboardCta';
import { canManageUsers } from '@/types/user';
import styles from './dashboard.module.css';

function repairStatusPt(status: string): string {
  const map: Record<string, string> = {
    pending: 'Pendente',
    in_progress: 'Em curso',
    completed: 'Concluído',
    cancelled: 'Cancelado',
  };
  return map[status] ?? status.replace(/_/g, ' ');
}

export default function ClientDashboardPage() {
  const [recentRepairs, setRecentRepairs] = useState<Repair[]>([]);
  const [repairsLoading, setRepairsLoading] = useState(false);
  const [nowMs, setNowMs] = useState(() => Date.now());

  const { user, logout } = useAuth();
  const authHydrated = useAuthHydrationReady();
  const { cars, isLoading: carsLoading, error: carsError, fetchCars } = useCars();
  const { appointments, isLoading: appointmentsLoading, error: appointmentsError, fetchAppointments } = useAppointments();
  const router = useRouter();
  
  // Combined loading and error states
  const loading = carsLoading || appointmentsLoading;
  const error = carsError || appointmentsError;
  
  // Filter upcoming appointments
  const upcomingAppointments = appointments.filter((appointment) => {
    const raw = appointment.date;
    if (!raw) return false;
    const t = new Date(raw).getTime();
    return !Number.isNaN(t) && t > nowMs;
  });

  const isClientRole = user?.role === UserRole.CLIENT;

  useEffect(() => {
    const id = window.setInterval(() => setNowMs(Date.now()), 60_000);
    return () => window.clearInterval(id);
  }, []);

  useEffect(() => {
    if (!authHydrated) return;
    if (!user) {
      router.replace('/auth/login');
      return;
    }
    let cancelled = false;
    queueMicrotask(() => {
      if (cancelled) return;
      fetchCars();
      fetchAppointments();
    });
    return () => {
      cancelled = true;
    };
  }, [authHydrated, user, router, fetchCars, fetchAppointments]);

  useEffect(() => {
    if (!authHydrated || !user) return;
    if (cars.length === 0) {
      let cancelledEmpty = false;
      queueMicrotask(() => {
        if (!cancelledEmpty) setRecentRepairs([]);
      });
      return () => {
        cancelledEmpty = true;
      };
    }
    let cancelled = false;
    void (async () => {
      await Promise.resolve();
      if (cancelled) return;
      setRepairsLoading(true);
      try {
        const batches = await Promise.all(
          cars.map(async (car) => {
            const { data, error } = await apiClient.getRepairs(car.id);
            if (error || !data) return [] as Repair[];
            return Array.isArray(data) ? data : [];
          }),
        );
        if (cancelled) return;
        const merged = batches.flat();
        merged.sort(
          (a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime(),
        );
        setRecentRepairs(merged.slice(0, 8));
      } finally {
        if (!cancelled) setRepairsLoading(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [authHydrated, user, cars]);

  if (!authHydrated || !user) {
    return (
      <div className="loadingScreen" aria-busy="true">
        <AppLoading size="lg" aria-busy={false} label="A sessão a carregar" />
      </div>
    );
  }

  const shellSubtitle = isClientRole ? 'Painel do cliente' : 'Painel da oficina';
  const carsLabel = isClientRole ? 'Os meus automóveis' : 'Viaturas';

  if (loading) {
    return (
      <AppShell
        user={user}
        subtitle={shellSubtitle}
        activeNav="dashboard"
        carsNavLabel={carsLabel}
        logoVariant="branded"
        onLogout={logout}
      >
        <div className={styles.loadingContainer} aria-busy="true">
          <AppLoading size="md" aria-busy={false} />
          <span>A carregar o painel…</span>
        </div>
      </AppShell>
    );
  }

  return (
    <AppShell
      user={user}
      subtitle={shellSubtitle}
      activeNav="dashboard"
      carsNavLabel={carsLabel}
      logoVariant="branded"
      onLogout={logout}
    >
        {error && (
          <div className={styles.errorAlert}>
            <span>{error}</span>
          </div>
        )}

        {canManageUsers(user) ? (
          <StaffUsersDashboardCta onManageUsers={() => router.push('/admin/users')} />
        ) : null}

        {/* Stats Grid */}
        <div className={styles.statsGrid}>
          <div className={styles.statCard}>
            <div className={styles.statIcon}>
              <svg fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 21V5a2 2 0 00-2-2H7a2 2 0 00-2 2v16m14 0h2m-2 0h-2m-2 0H7m5 0v-5a2 2 0 012-2h2a2 2 0 012 2v5" />
              </svg>
            </div>
            <div>
              <h3>{isClientRole ? 'Os meus automóveis' : 'Viaturas'}</h3>
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
              <h3>Reparações ativas</h3>
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
              <h3>Próximas marcações</h3>
              <p className={styles.statNumber}>{upcomingAppointments.length}</p>
            </div>
          </div>
        </div>

        {/* Content Grid */}
        <div className={styles.contentGrid}>
          {/* Recent Cars */}
          <div className={styles.card}>
            <div className={styles.cardHeader}>
              <h3>{isClientRole ? 'Os meus automóveis' : 'Viaturas'}</h3>
              <Button type="button" variant="ghost" onClick={() => router.push('/cars')} className={styles.linkButton}>
                Ver tudo
              </Button>
            </div>
            <div className={styles.cardBody}>
              {cars.length === 0 ? (
                <div className={styles.emptyState}>
                  <p>{isClientRole ? 'Ainda sem automóveis registados' : 'Sem viaturas na lista'}</p>
                  <Button type="button" onClick={() => router.push('/cars')} className={styles.primaryButton}>
                    {isClientRole ? 'Adicionar o primeiro automóvel' : 'Abrir viaturas'}
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
                        onClick={() => router.push(`/cars/${car.id}`)}
                        className={styles.viewButton}
                      >
                        Ver
                      </Button>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>

          {/* Recent Repairs */}
          <div className={styles.card}>
            <div className={styles.cardHeader}>
              <h3>Reparações recentes</h3>
              <Button type="button" variant="ghost" onClick={() => router.push('/cars')} className={styles.linkButton}>
                Ver nos automóveis
              </Button>
            </div>
            <div className={styles.cardBody}>
              {repairsLoading ? (
                <div className={styles.emptyState}>
                  <p>A carregar reparações…</p>
                </div>
              ) : recentRepairs.length === 0 ? (
                <div className={styles.emptyState}>
                  <p>Ainda sem reparações</p>
                </div>
              ) : (
                <div className={styles.repairsList}>
                  {recentRepairs.map((repair) => (
                    <div key={repair.id} className={styles.repairItem}>
                      <div className={styles.repairStatus}>
                        <span className={`${styles.statusBadge} ${styles[repair.status]}`}>
                          {repairStatusPt(repair.status)}
                        </span>
                      </div>
                      <div className={styles.repairInfo}>
                        <h4>{repair.description}</h4>
                        <p>
                          {new Intl.NumberFormat('pt-PT', { style: 'currency', currency: 'EUR' }).format(repair.cost)}
                        </p>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>
        </div>
    </AppShell>
  );
}