'use client';

import React, { Suspense, useCallback, useEffect, useRef, useState } from 'react';
import Link from 'next/link';
import { useRouter, useSearchParams } from 'next/navigation';
import { useAuth } from '@/stores';
import AppShell from '@/components/layout/AppShell';
import { issuedInvoiceService } from '@/lib/services/issued-invoice.service';
import { fiscalizationService } from '@/lib/services/fiscalization.service';
import type { IssuedInvoice } from '@/types/accounting';
import type { FiscalizationSummary } from '@/types/fiscal';
import { FiscalStatusBadge } from '@/components/fiscal/FiscalStatusBadge';
import styles from '../accounting.module.css';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { IssuedInvoiceCreateForm } from './IssuedInvoiceCreateForm';

function IssuedInvoicesListContent() {
  const { user, logout } = useAuth();
  const router = useRouter();
  const searchParams = useSearchParams();
  const [items, setItems] = useState<IssuedInvoice[]>([]);
  const [summaries, setSummaries] = useState<Record<string, FiscalizationSummary>>({});
  const [error, setError] = useState<string | null>(null);
  const [createOpen, setCreateOpen] = useState(false);
  const [createDefaults, setCreateDefaults] = useState<{
    customerId?: string;
    carId?: string;
    repairId?: string;
  }>({});
  const openedFromCreateQuery = useRef(false);

  const load = useCallback(async () => {
    setError(null);
    const res = await issuedInvoiceService.listStaff();
    if (res.success && res.data) {
      const rows = res.data.items;
      setItems(rows);
      if (rows.length > 0) {
        const sumRes = await fiscalizationService.listSummaries(rows.map((r) => r.id));
        if (sumRes.success && sumRes.data?.items) {
          const map: Record<string, FiscalizationSummary> = {};
          for (const s of sumRes.data.items) {
            map[s.invoiceId] = s;
          }
          setSummaries(map);
        } else {
          setSummaries({});
        }
      } else {
        setSummaries({});
      }
    } else {
      setError(res.error?.message ?? 'Não foi possível carregar as faturas.');
    }
  }, []);

  useEffect(() => {
    let cancelled = false;
    queueMicrotask(() => {
      if (!cancelled) void load();
    });
    return () => {
      cancelled = true;
    };
  }, [load]);

  useEffect(() => {
    if (searchParams.get('create') !== '1' || openedFromCreateQuery.current) return;
    openedFromCreateQuery.current = true;
    setCreateDefaults({
      customerId: searchParams.get('customerId') ?? undefined,
      carId: searchParams.get('carId') ?? undefined,
      repairId: searchParams.get('repairId') ?? undefined,
    });
    setCreateOpen(true);
    router.replace('/accounting/issued-invoices');
  }, [searchParams, router]);

  if (!user) return null;

  const handleCreated = () => {
    setCreateOpen(false);
    setCreateDefaults({});
    void load();
  };

  return (
    <AppShell
      user={user}
      subtitle="Contabilidade — Faturas emitidas"
      activeNav="accounting"
      carsNavLabel="Viaturas"
      onLogout={logout}
      logoVariant="branded"
    >
      <div className={styles.toolbar}>
        <h1>Faturas emitidas (clientes)</h1>
        <Button type="button" onClick={() => setCreateOpen(true)}>
          Nova fatura
        </Button>
      </div>
      <p className={styles.intro}>
        <Link href="/accounting" className={styles.mutedLink}>
          ← Contabilidade
        </Link>
      </p>
      {error ? <div className={styles.error}>{error}</div> : null}
      <div className={styles.tableWrap}>
        <table className={styles.table}>
          <thead>
            <tr>
              <th>Cliente (ID)</th>
              <th>Valor</th>
              <th>Estado</th>
              <th>Fiscalização</th>
            </tr>
          </thead>
          <tbody>
            {items.map((inv) => {
              const fiscal = summaries[inv.id];
              return (
                <tr key={inv.id}>
                  <td>
                    <Link href={`/accounting/issued-invoices/${inv.id}`}>{inv.customerId}</Link>
                  </td>
                  <td>{inv.amount.toFixed(2)}</td>
                  <td>{inv.status}</td>
                  <td>{fiscal ? <FiscalStatusBadge status={fiscal.status} /> : '—'}</td>
                </tr>
              );
            })}
            {items.length === 0 && !error ? (
              <tr>
                <td colSpan={4}>
                  Sem faturas.{' '}
                  <button type="button" className={styles.inlineTextButton} onClick={() => setCreateOpen(true)}>
                    Criar a primeira
                  </button>
                </td>
              </tr>
            ) : null}
          </tbody>
        </table>
      </div>

      <Dialog open={createOpen} onOpenChange={setCreateOpen}>
        <DialogContent className="max-h-[90vh] overflow-hidden sm:max-w-lg" aria-describedby={undefined}>
          <DialogHeader>
            <DialogTitle>Nova fatura emitida</DialogTitle>
          </DialogHeader>
          <div className={styles.dialogFormBody}>
            <IssuedInvoiceCreateForm
              onSuccess={handleCreated}
              onCancel={() => {
                setCreateOpen(false);
                setCreateDefaults({});
              }}
              initialCustomerId={createDefaults.customerId}
              initialCarId={createDefaults.carId}
              initialRepairId={createDefaults.repairId}
            />
          </div>
        </DialogContent>
      </Dialog>
    </AppShell>
  );
}

export default function IssuedInvoicesStaffListPage() {
  return (
    <Suspense fallback={null}>
      <IssuedInvoicesListContent />
    </Suspense>
  );
}
