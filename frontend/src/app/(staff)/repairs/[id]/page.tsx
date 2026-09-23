'use client';

import React, { useCallback, useEffect, useState } from 'react';
import Link from 'next/link';
import { useParams, useRouter } from 'next/navigation';
import { useAuth } from '@/stores';
import { apiClient, type Repair } from '@/lib/api';
import { issuedInvoiceService, type IssuedInvoice } from '@/lib/services/issued-invoice.service';
import { canManageUsers, isClient } from '@/types/user';
import AppShell from '@/components/layout/AppShell';
import { AppLoading } from '@/components/ui/AppLoading';
import { Button } from '@/components/ui/button';
import { useAuthHydrationReady } from '@/hooks/useAuthHydrationReady';
import styles from './repair-detail.module.css';

function repairStatusPt(status: string): string {
  const map: Record<string, string> = {
    pending: 'Pendente',
    in_progress: 'Em curso',
    completed: 'Concluída',
    cancelled: 'Cancelada',
  };
  return map[status] ?? status.replace(/_/g, ' ');
}

function formatCurrency(amount: number): string {
  return new Intl.NumberFormat('pt-PT', { style: 'currency', currency: 'EUR' }).format(amount);
}

function formatDate(iso?: string): string {
  if (!iso) return '—';
  return new Date(iso).toLocaleString('pt-PT', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });
}

export default function RepairDetailPage() {
  const params = useParams();
  const repairId = params.id as string;
  const router = useRouter();
  const { user, logout } = useAuth();
  const authHydrated = useAuthHydrationReady();

  const [repair, setRepair] = useState<Repair | null>(null);
  const [invoice, setInvoice] = useState<IssuedInvoice | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const staffAccounting = Boolean(user && canManageUsers(user));

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const { data, error: apiErr } = await apiClient.getRepair(repairId);
      if (apiErr || !data) {
        setError(apiErr?.message ?? 'Não foi possível carregar a reparação.');
        setRepair(null);
        return;
      }
      setRepair(data);

      if (staffAccounting) {
        const invRes = await issuedInvoiceService.getByRepair(repairId);
        if (invRes.success && invRes.data?.items?.[0]) {
          setInvoice(invRes.data.items[0]);
        } else {
          setInvoice(null);
        }
      } else {
        setInvoice(null);
      }
    } finally {
      setLoading(false);
    }
  }, [repairId, staffAccounting]);

  useEffect(() => {
    if (!authHydrated) return;
    if (!user) {
      router.replace('/auth/login');
      return;
    }
    if (isClient(user)) {
      router.replace('/dashboard');
      return;
    }
    let cancelled = false;
    queueMicrotask(() => {
      if (!cancelled) void load();
    });
    return () => {
      cancelled = true;
    };
  }, [authHydrated, user, router, load]);

  if (!authHydrated || !user) {
    return (
      <div className="loadingScreen" aria-busy="true">
        <AppLoading size="lg" aria-busy={false} label="A sessão a carregar" />
      </div>
    );
  }

  const createInvoiceHref =
    repair && repair.status === 'completed'
      ? `/accounting/issued-invoices?create=1&repairId=${encodeURIComponent(repair.id)}&carId=${encodeURIComponent(repair.car_id)}`
      : null;

  return (
    <AppShell
      user={user}
      subtitle="Reparação"
      activeNav="cars"
      carsNavLabel="Viaturas"
      onLogout={logout}
      logoVariant="branded"
      toolbar={
        <>
          <h1>Detalhe da reparação</h1>
          {repair ? (
            <Button type="button" variant="outline" onClick={() => router.push(`/repairs/${repair.id}/edit`)}>
              Editar
            </Button>
          ) : null}
        </>
      }
    >
      <p className={styles.back}>
        <Link href="/dashboard">← Painel</Link>
        {repair ? (
          <>
            {' · '}
            <Link href={`/cars/${repair.car_id}`}>Ver viatura</Link>
          </>
        ) : null}
      </p>

      {loading ? (
        <div className={styles.loading} aria-busy="true">
          <AppLoading size="sm" aria-busy={false} />
          <span>A carregar…</span>
        </div>
      ) : null}

      {error ? <div className={styles.error}>{error}</div> : null}

      {!loading && repair ? (
        <div className={styles.panel}>
          <div className={styles.row}>
            <span className={`${styles.badge} ${styles[repair.status]}`}>{repairStatusPt(repair.status)}</span>
            <strong className={styles.cost}>{formatCurrency(repair.cost)}</strong>
          </div>
          <h2 className={styles.title}>{repair.description}</h2>
          <dl className={styles.meta}>
            <div>
              <dt>Criada</dt>
              <dd>{formatDate(repair.created_at)}</dd>
            </div>
            <div>
              <dt>Início</dt>
              <dd>{formatDate(repair.started_at)}</dd>
            </div>
            <div>
              <dt>Conclusão</dt>
              <dd>{formatDate(repair.completed_at)}</dd>
            </div>
          </dl>

          {staffAccounting ? (
            <section className={styles.billing} aria-labelledby="billing-heading">
              <h3 id="billing-heading">Faturação interna</h3>
              <p className={styles.billingHint}>
                Fatura do sistema (registo operativo). Sem valor fiscal — independente do Cloudware.
              </p>
              {invoice ? (
                <div className={styles.billingActions}>
                  <Button type="button" onClick={() => router.push(`/accounting/issued-invoices/${invoice.id}`)}>
                    Ver fatura interna
                  </Button>
                </div>
              ) : repair.status === 'completed' && createInvoiceHref ? (
                <div className={styles.billingActions}>
                  <Button type="button" onClick={() => router.push(createInvoiceHref)}>
                    Criar fatura interna
                  </Button>
                </div>
              ) : (
                <p className={styles.billingMuted}>
                  Só é possível criar fatura quando a reparação está concluída.
                </p>
              )}
            </section>
          ) : null}
        </div>
      ) : null}
    </AppShell>
  );
}
