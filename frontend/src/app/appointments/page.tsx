'use client';

import React, { Suspense, useCallback, useEffect, useRef, useState } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import { useAppointmentStore } from '@/stores/appointment.store';
import { useCarStore } from '@/stores/car.store';
import EmptyAppointmentState from '@/components/empty-states/EmptyAppointmentState';
import AppointmentCard from '@/components/appointments/AppointmentCard';
import NewAppointmentModal from '@/components/appointments/NewAppointmentModal';
import AppointmentModal from '@/app/appointments/components/AppointmentModal';
import type {
  Appointment,
  CreateAppointmentRequest,
  UpdateAppointmentRequest,
} from '@/types/appointment';
import { useAuth } from '@/stores';
import AppShell from '@/components/layout/AppShell';
import { useAuthHydrationReady } from '@/hooks/useAuthHydrationReady';
import styles from './appointments.module.css';
import emptyStyles from '@/components/empty-states/EmptyState.module.css';
import { AppLoading } from '@/components/ui/AppLoading';
import { Button } from '@/components/ui/button';

function AppointmentsPageContent() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const scheduleQueryHandledRef = useRef(false);
  const authHydrated = useAuthHydrationReady();
  const { user, logout, isAuthenticated } = useAuth();

  const [scheduleOpen, setScheduleOpen] = useState(false);
  const [scheduleCarId, setScheduleCarId] = useState<string | undefined>();
  const [editingAppointment, setEditingAppointment] = useState<Appointment | null>(null);

  const {
    appointments,
    isLoading: appointmentsLoading,
    error: appointmentsError,
    fetchAppointments,
    updateAppointment,
  } = useAppointmentStore();

  const { cars, isLoading: carsLoading, fetchCars } = useCarStore();

  useEffect(() => {
    if (!authHydrated) return;
    if (!user || !isAuthenticated) {
      router.replace('/auth/login');
    }
  }, [authHydrated, user, isAuthenticated, router]);

  useEffect(() => {
    if (!authHydrated || !user || !isAuthenticated) return;
    fetchAppointments();
    fetchCars();
  }, [authHydrated, user, isAuthenticated, fetchAppointments, fetchCars]);

  const scheduleFlag = searchParams.get('schedule');
  const carIdFromQuery = searchParams.get('carId');

  useEffect(() => {
    if (scheduleFlag !== '1') {
      scheduleQueryHandledRef.current = false;
      return;
    }
    if (scheduleQueryHandledRef.current) return;
    scheduleQueryHandledRef.current = true;
    let cancelled = false;
    queueMicrotask(() => {
      if (cancelled) {
        scheduleQueryHandledRef.current = false;
        return;
      }
      setScheduleOpen(true);
      if (carIdFromQuery) setScheduleCarId(carIdFromQuery);
      router.replace('/appointments', { scroll: false });
    });
    return () => {
      cancelled = true;
    };
  }, [scheduleFlag, carIdFromQuery, router]);

  useEffect(() => {
    if (!authHydrated || !user || !isAuthenticated) return;
    if (carsLoading) return;
    if (cars.length > 0) return;
    const t = setTimeout(() => {
      router.replace('/cars?addCar=1');
    }, 3200);
    return () => clearTimeout(t);
  }, [authHydrated, user, isAuthenticated, carsLoading, cars.length, router]);

  const openSchedule = useCallback((carId?: string) => {
    if (cars.length === 0) {
      router.replace('/cars?addCar=1');
      return;
    }
    setScheduleCarId(carId);
    setScheduleOpen(true);
  }, [cars.length, router]);

  const closeSchedule = useCallback(() => {
    setScheduleOpen(false);
    setScheduleCarId(undefined);
  }, []);

  const closeEditModal = useCallback(() => {
    setEditingAppointment(null);
  }, []);

  const handleCreateNoop = useCallback(async (_data: CreateAppointmentRequest): Promise<boolean> => {
    return false;
  }, []);

  const handleUpdateAppointmentFromModal = useCallback(
    async (id: string, appointmentData: Partial<UpdateAppointmentRequest>): Promise<boolean> => {
      try {
        const success = await updateAppointment(id, appointmentData as UpdateAppointmentRequest);
        if (success) {
          await fetchAppointments();
          setEditingAppointment(null);
          return true;
        }
        return false;
      } catch (error) {
        console.error('Failed to update appointment:', error);
        return false;
      }
    },
    [updateAppointment, fetchAppointments]
  );

  if (!authHydrated || !user || !isAuthenticated) {
    return (
      <div className="loadingScreen" aria-busy="true">
        <AppLoading size="lg" aria-busy={false} label="A sessão a carregar" />
      </div>
    );
  }

  // Only block the full page while the shell has no data yet — not when cars refresh in the background
  const isBootstrapping =
    (appointmentsLoading && appointments.length === 0 && !appointmentsError) ||
    (carsLoading && cars.length === 0);

  if (isBootstrapping) {
    return (
      <AppShell user={user} subtitle="Marcações" activeNav="appointments" logoVariant="branded" onLogout={logout}>
        <div className="loadingStack" aria-busy="true">
          <AppLoading size="md" aria-busy={false} />
          <span>A carregar marcações…</span>
        </div>
      </AppShell>
    );
  }

  if (appointmentsError) {
    return (
      <AppShell user={user} subtitle="Marcações" activeNav="appointments" logoVariant="branded" onLogout={logout}>
        <div className="alertError">
          <p>Erro ao carregar marcações: {appointmentsError}</p>
          <Button type="button" className="mt-3" onClick={() => fetchAppointments()}>
            Tentar novamente
          </Button>
        </div>
      </AppShell>
    );
  }

  if (cars.length === 0) {
    const toolbarNoCars = <h1>As minhas marcações</h1>;
    return (
      <AppShell
        user={user}
        subtitle="Marcações"
        activeNav="appointments"
        logoVariant="branded"
        onLogout={logout}
        toolbar={toolbarNoCars}
      >
        <div className={emptyStyles.emptyState}>
          <div className={emptyStyles.emptyStateIcon} aria-hidden>
            🚗
          </div>
          <h3 className={emptyStyles.emptyStateTitle}>Ainda não tem automóveis registados</h3>
          <p className={emptyStyles.emptyStateDescription}>
            Para marcar uma visita, registe primeiro pelo menos um automóvel em <strong>Os meus automóveis</strong>.
          </p>
          <p className={`${emptyStyles.emptyStateDescription} ${styles.noCarsRedirectHint}`}>
            A redirecionar para os seus automóveis dentro de momentos…
          </p>
          <Button type="button" className={emptyStyles.emptyStateButton} onClick={() => router.replace('/cars?addCar=1')}>
            Ir já para os meus automóveis
          </Button>
        </div>
      </AppShell>
    );
  }

  const toolbar = (
    <>
      <h1>As minhas marcações ({appointments.length})</h1>
      <Button type="button" onClick={() => openSchedule()}>
        Nova marcação
      </Button>
    </>
  );

  return (
    <>
      <AppShell
        user={user}
        subtitle="Marcações"
        activeNav="appointments"
        logoVariant="branded"
        onLogout={logout}
        toolbar={toolbar}
      >
        {appointments.length === 0 ? (
          <EmptyAppointmentState onSchedule={() => openSchedule()} />
        ) : (
          <div className={styles.list}>
            {appointments.map((appointment) => (
              <AppointmentCard
                key={appointment.id}
                appointment={appointment}
                onEdit={(apt) => setEditingAppointment(apt)}
                onReschedule={(carId) => openSchedule(carId)}
              />
            ))}
          </div>
        )}
      </AppShell>

      <NewAppointmentModal
        isOpen={scheduleOpen}
        onClose={closeSchedule}
        initialCarId={scheduleCarId}
        onCreated={() => {
          fetchAppointments().catch(() => {});
        }}
      />

      {editingAppointment && (
        <AppointmentModal
          appointment={editingAppointment}
          onClose={closeEditModal}
          onCreate={handleCreateNoop}
          onUpdate={handleUpdateAppointmentFromModal}
        />
      )}
    </>
  );
}

function AppointmentsFallback() {
  return (
    <div className="loadingScreen" aria-busy="true">
      <AppLoading size="lg" aria-busy={false} label="A carregar marcações" />
    </div>
  );
}

export default function AppointmentsPage() {
  return (
    <Suspense fallback={<AppointmentsFallback />}>
      <AppointmentsPageContent />
    </Suspense>
  );
}
