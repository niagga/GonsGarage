'use client';

import React, { Suspense, useCallback, useEffect, useRef, useState } from 'react';
import Link from 'next/link';
import { useRouter, useSearchParams } from 'next/navigation';
import { useAuth } from '@/stores';
import AppShell from '@/components/layout/AppShell';
import { billingDocumentService } from '@/lib/services/billing-document.service';
import type { BillingDocument } from '@/types/accounting';
import styles from '../accounting.module.css';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { BillingDocumentCreateForm } from './BillingDocumentCreateForm';

const kindLabel: Record<string, string> = {
  payroll: 'Salários',
  irs: 'IRS',
  other: 'Outro',
};

function BillingDocumentsListContent() {
  const { user, logout } = useAuth();
  const router = useRouter();
  const searchParams = useSearchParams();
  const [items, setItems] = useState<BillingDocument[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [createOpen, setCreateOpen] = useState(false);
  const openedFromCreateQuery = useRef(false);

  const load = useCallback(async () => {
    setError(null);
    const res = await billingDocumentService.list();
    if (res.success && res.data) setItems(res.data.items);
    else setError(res.error?.message ?? 'Não foi possível carregar os documentos.');
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
    setCreateOpen(true);
    router.replace('/accounting/billing-documents');
  }, [searchParams, router]);

  if (!user) return null;

  const handleCreated = () => {
    setCreateOpen(false);
    void load();
  };

  return (
    <AppShell
      user={user}
      subtitle="Contabilidade — Documentos"
      activeNav="accounting"
      carsNavLabel="Viaturas"
      onLogout={logout}
      logoVariant="branded"
    >
      <div className={styles.toolbar}>
        <h1>Documentos de faturação</h1>
        <Button type="button" onClick={() => setCreateOpen(true)}>
          Novo documento
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
              <th>Tipo</th>
              <th>Título</th>
              <th>Valor</th>
            </tr>
          </thead>
          <tbody>
            {items.map((d) => (
              <tr key={d.id}>
                <td>{kindLabel[d.kind] ?? d.kind}</td>
                <td>
                  <Link href={`/accounting/billing-documents/${d.id}`}>{d.title}</Link>
                </td>
                <td>{d.amount.toFixed(2)}</td>
              </tr>
            ))}
            {items.length === 0 && !error ? (
              <tr>
                <td colSpan={3}>
                  Sem registos.{' '}
                  <button type="button" className={styles.inlineTextButton} onClick={() => setCreateOpen(true)}>
                    Criar o primeiro
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
            <DialogTitle>Novo documento</DialogTitle>
          </DialogHeader>
          <div className={styles.dialogFormBody}>
            <BillingDocumentCreateForm onSuccess={handleCreated} onCancel={() => setCreateOpen(false)} />
          </div>
        </DialogContent>
      </Dialog>
    </AppShell>
  );
}

export default function BillingDocumentsListPage() {
  return (
    <Suspense fallback={null}>
      <BillingDocumentsListContent />
    </Suspense>
  );
}
