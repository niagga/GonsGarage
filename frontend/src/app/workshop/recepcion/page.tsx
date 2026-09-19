'use client';

import React, { useCallback, useEffect, useState } from 'react';
import Link from 'next/link';
import { useSearchParams } from 'next/navigation';
import { useAuth } from '@/stores';
import { useAuthHydrationReady } from '@/hooks/useAuthHydrationReady';
import AppShell from '@/components/layout/AppShell';
import { Button } from '@/components/ui/button';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { apiClient, type Car, type ServiceJob, type ServiceJobDetail } from '@/lib/api';
import styles from '../workshop.module.css';

export default function WorkshopRecepcionPage() {
  return (
    <React.Suspense fallback={<p className={styles.hint}>A carregar…</p>}>
      <WorkshopRecepcionInner />
    </React.Suspense>
  );
}

function WorkshopRecepcionInner() {
  const searchParams = useSearchParams();
  const jobId = (searchParams.get('job') ?? '').trim();
  const { user, logout } = useAuth();
  const authHydrated = useAuthHydrationReady();
  const [cars, setCars] = useState<Car[]>([]);
  const [carId, setCarId] = useState<string>('');
  const [jobs, setJobs] = useState<ServiceJob[]>([]);
  const [detail, setDetail] = useState<ServiceJobDetail | null>(null);
  const [rKm, setRKm] = useState('');
  const [rNotes, setRNotes] = useState('');
  const [err, setErr] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);

  const loadCars = useCallback(async () => {
    if (!user) return;
    const { data, error } = await apiClient.getCars();
    if (error) {
      setErr(error.message);
      return;
    }
    setCars(data ?? []);
  }, [user]);

  useEffect(() => {
    if (!authHydrated || !user) return;
    let cancelled = false;
    queueMicrotask(() => {
      if (!cancelled) void loadCars();
    });
    return () => {
      cancelled = true;
    };
  }, [authHydrated, user, loadCars]);

  useEffect(() => {
    if (cars.length === 0 || carId) return;
    let cancelled = false;
    queueMicrotask(() => {
      if (cancelled) return;
      setCarId(cars[0].id);
    });
    return () => {
      cancelled = true;
    };
  }, [cars, carId]);

  const loadJobs = useCallback(async () => {
    if (!carId) {
      setJobs([]);
      return;
    }
    setErr(null);
    const { data, error } = await apiClient.listServiceJobsByCar(carId);
    if (error) {
      setErr(error.message);
      setJobs([]);
      return;
    }
    setJobs(data ?? []);
  }, [carId]);

  useEffect(() => {
    if (!authHydrated || !carId || jobId) return;
    let cancelled = false;
    queueMicrotask(() => {
      if (!cancelled) void loadJobs();
    });
    return () => {
      cancelled = true;
    };
  }, [authHydrated, carId, jobId, loadJobs]);

  const loadDetail = useCallback(async () => {
    if (!jobId) {
      setDetail(null);
      return;
    }
    setErr(null);
    const { data, error } = await apiClient.getServiceJob(jobId);
    if (error) {
      setErr(error.message);
      setDetail(null);
      return;
    }
    setDetail(data ?? null);
  }, [jobId]);

  useEffect(() => {
    if (!authHydrated || !user || !jobId) return;
    let cancelled = false;
    queueMicrotask(() => {
      if (!cancelled) void loadDetail();
    });
    return () => {
      cancelled = true;
    };
  }, [authHydrated, user, jobId, loadDetail]);

  const onSaveReception = async () => {
    if (!jobId) return;
    const km = Number.parseInt(rKm, 10);
    if (Number.isNaN(km) || km < 0) {
      setErr('Indique quilometragem (km) válida.');
      return;
    }
    setErr(null);
    setSaving(true);
    const { error } = await apiClient.putServiceJobReception(jobId, {
      odometer_km: km,
      general_notes: rNotes.trim() || undefined,
    });
    setSaving(false);
    if (error) {
      setErr(error.message);
      return;
    }
    setRKm('');
    setRNotes('');
    await loadDetail();
  };

  if (!authHydrated || !user) return null;

  if (jobId && detail) {
    if (detail.job.status === 'closed' || detail.job.status === 'cancelled') {
      return (
        <AppShell
          user={user}
          subtitle="Taller — receção"
          activeNav="workshop"
          onLogout={logout}
          logoVariant="branded"
          toolbar={
            <>
              <h1>Receção</h1>
              <Button type="button" variant="outline" asChild>
                <Link href="/workshop/recepcion">Voltar</Link>
              </Button>
            </>
          }
        >
          <p className={styles.hint}>Esta visita já está fechada ou cancelada — use o detalhe completo se precisar de histórico.</p>
          <Button type="button" asChild>
            <Link href={`/workshop/${jobId}`}>Detalhe da visita</Link>
          </Button>
        </AppShell>
      );
    }

    return (
      <AppShell
        user={user}
        subtitle="Taller — receção rápida"
        activeNav="workshop"
        onLogout={logout}
        logoVariant="branded"
        toolbar={
          <>
            <h1>Receção</h1>
            <Button type="button" variant="outline" asChild>
              <Link href="/workshop/recepcion">Outra visita</Link>
            </Button>
            <Button type="button" variant="outline" asChild>
              <Link href={`/workshop/${jobId}`}>Detalhe completo</Link>
            </Button>
          </>
        }
      >
        {err ? <p className={styles.err}>{err}</p> : null}
        <p className={styles.hint}>Visita {jobId.slice(0, 8)}… (mesma API de recepção do detalhe)</p>
        {detail.reception ? (
          <p className={styles.hint}>
            Já registado: {detail.reception.odometer_km} km — {detail.reception.general_notes || '—'}
          </p>
        ) : null}
        <div className={styles.formGrid}>
          <label>
            Km
            <input
              type="number"
              min={0}
              value={rKm}
              onChange={e => setRKm(e.target.value)}
              placeholder="ex. 120000"
            />
          </label>
          <label>
            Notas
            <textarea value={rNotes} onChange={e => setRNotes(e.target.value)} rows={3} placeholder="Níveis, pneus, etc." />
          </label>
          <div className={styles.actions}>
            <Button type="button" disabled={saving} onClick={() => void onSaveReception()}>
              Gravar recepção
            </Button>
          </div>
        </div>
      </AppShell>
    );
  }

  if (jobId && !detail) {
    if (err) {
      return (
        <AppShell
          user={user}
          subtitle="Taller"
          activeNav="workshop"
          onLogout={logout}
          logoVariant="branded"
          toolbar={
            <>
              <h1>Receção</h1>
              <Button type="button" variant="outline" asChild>
                <Link href="/workshop/recepcion">Voltar</Link>
              </Button>
            </>
          }
        >
          <p className={styles.err}>{err}</p>
        </AppShell>
      );
    }
    return (
      <AppShell
        user={user}
        subtitle="Taller"
        activeNav="workshop"
        onLogout={logout}
        logoVariant="branded"
        toolbar={<h1>Receção</h1>}
      >
        <p>A carregar…</p>
      </AppShell>
    );
  }

  const openForCar = (jobs ?? []).filter(j => j.status === 'open' || j.status === 'in_progress');

  return (
    <AppShell
      user={user}
      subtitle="Taller — receção rápida"
      activeNav="workshop"
      onLogout={logout}
      logoVariant="branded"
      toolbar={
        <>
          <h1>Escolher visita</h1>
          <Button type="button" variant="outline" asChild>
            <Link href="/workshop">Lista taller</Link>
          </Button>
        </>
      }
    >
      {err ? <p className={styles.err}>{err}</p> : null}
      <p className={styles.hint}>
        Seleccione um veículo e abra a receção de uma visita aberta. Mesmo corpo de API (PUT /service-jobs/…/reception).
      </p>
      <div className={styles.select}>
        <label htmlFor="recepcion-car">Viatura</label>
        <Select value={carId || '__none__'} onValueChange={value => setCarId(value === '__none__' ? '' : value)}>
          <SelectTrigger id="recepcion-car">
            <SelectValue placeholder="—" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="__none__">—</SelectItem>
                {cars.map(c => (
              <SelectItem key={c.id} value={c.id}>
                {c.license_plate} — {c.make} {c.model}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
      {carId && openForCar.length === 0 ? (
        <p className={styles.muted}>Nenhuma visita aberta para este veículo. Crie uma em Lista taller.</p>
      ) : null}
      {openForCar.length > 0 ? (
        <ul className={styles.recepcionList}>
          {openForCar.map(j => (
            <li key={j.id}>
              <Button type="button" size="sm" asChild>
                <Link href={`/workshop/recepcion?job=${j.id}`}>Receção — {j.id.slice(0, 8)}…</Link>
              </Button>
            </li>
          ))}
        </ul>
      ) : null}
    </AppShell>
  );
}
