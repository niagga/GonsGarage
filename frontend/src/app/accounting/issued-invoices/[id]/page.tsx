'use client';

import React, { useCallback, useEffect, useState } from 'react';
import Link from 'next/link';
import { useParams, useRouter } from 'next/navigation';
import { useAuth } from '@/stores';
import AppShell from '@/components/layout/AppShell';
import { issuedInvoiceService } from '@/lib/services/issued-invoice.service';
import type { IssuedInvoice } from '@/types/accounting';
import styles from '../../accounting.module.css';
import { AppLoading } from '@/components/ui/AppLoading';
import { Button } from '@/components/ui/button';

export default function IssuedInvoiceStaffDetailPage() {
  const { id } = useParams<{ id: string }>();
  const { user, logout } = useAuth();
  const router = useRouter();
  const [row, setRow] = useState<IssuedInvoice | null>(null);
  const [amount, setAmount] = useState('');
  const [status, setStatus] = useState('');
  const [notes, setNotes] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);

  const load = useCallback(async () => {
    if (!id) return;
    setError(null);
    const res = await issuedInvoiceService.get(id);
    if (res.success && res.data) {
      const inv = res.data;
      setRow(inv);
      setAmount(String(inv.amount));
      setStatus(inv.status);
      setNotes(inv.notes ?? '');
    } else {
      setError(res.error?.message ?? 'Fatura não encontrada.');
    }
  }, [id]);

  useEffect(() => {
    let cancelled = false;
    queueMicrotask(() => {
      if (!cancelled) void load();
    });
    return () => {
      cancelled = true;
    };
  }, [load]);

  if (!user) return null;

  async function onSave(e: React.FormEvent) {
    e.preventDefault();
    if (!id) return;
    const amt = Number.parseFloat(amount.replace(',', '.'));
    if (Number.isNaN(amt)) {
      setError('Indique um valor numérico válido.');
      return;
    }
    setSaving(true);
    setError(null);
    const res = await issuedInvoiceService.patchIssuedInvoice(id, {
      amount: amt,
      status: status.trim(),
      notes: notes.trim(),
    });
    setSaving(false);
    if (res.success && res.data) {
      setRow(res.data);
      return;
    }
    setError(res.error?.message ?? 'Erro ao guardar.');
  }

  async function onDelete() {
    if (!id || !window.confirm('Eliminar esta fatura?')) return;
    const res = await issuedInvoiceService.removeStaff(id);
    if (res.success) {
      router.replace('/accounting/issued-invoices');
      return;
    }
    setError(res.error?.message ?? 'Erro ao eliminar.');
  }

  return (
    <AppShell
      user={user}
      subtitle={row ? `Fatura ${row.id.slice(0, 8)}…` : 'Fatura emitida'}
      activeNav="accounting"
      carsNavLabel="Viaturas"
      onLogout={logout}
      logoVariant="branded"
    >
      <p className={styles.intro}>
        <Link href="/accounting/issued-invoices" className={styles.mutedLink}>
          ← Faturas emitidas
        </Link>
      </p>
      {error ? <div className={styles.error}>{error}</div> : null}
      {row ? (
        <>
          <div className={styles.readonlyBlock}>
            <strong>Cliente:</strong> {row.customerId}
          </div>
          <form className={styles.form} onSubmit={onSave}>
            <div className={styles.field}>
              <label htmlFor="amt">Valor</label>
              <input id="amt" value={amount} onChange={(ev) => setAmount(ev.target.value)} required />
            </div>
            <div className={styles.field}>
              <label htmlFor="st">Estado</label>
              <input id="st" value={status} onChange={(ev) => setStatus(ev.target.value)} required />
            </div>
            <div className={styles.field}>
              <label htmlFor="notes">Notas</label>
              <textarea id="notes" value={notes} onChange={(ev) => setNotes(ev.target.value)} />
            </div>
            <div className={styles.rowActions}>
              <Button type="submit" className={styles.submitButton} disabled={saving}>
                {saving ? 'A guardar…' : 'Guardar'}
              </Button>
              <Button type="button" variant="destructive" className={styles.dangerButton} onClick={() => void onDelete()}>
                Eliminar
              </Button>
            </div>
          </form>
        </>
      ) : !error ? (
        <div className="loadingStack" aria-busy="true">
          <AppLoading size="md" aria-busy={false} />
          <span>A carregar…</span>
        </div>
      ) : null}
    </AppShell>
  );
}
