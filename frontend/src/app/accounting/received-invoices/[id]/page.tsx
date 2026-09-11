'use client';

import React, { useCallback, useEffect, useState } from 'react';
import Link from 'next/link';
import { useParams, useRouter } from 'next/navigation';
import { useAuth } from '@/stores';
import AppShell from '@/components/layout/AppShell';
import { receivedInvoiceService } from '@/lib/services/received-invoice.service';
import type { ReceivedInvoice } from '@/types/accounting';
import styles from '../../accounting.module.css';
import { AppLoading } from '@/components/ui/AppLoading';
import { Button } from '@/components/ui/button';

export default function ReceivedInvoiceDetailPage() {
  const { id } = useParams<{ id: string }>();
  const { user, logout } = useAuth();
  const router = useRouter();
  const [row, setRow] = useState<ReceivedInvoice | null>(null);
  const [supplierId, setSupplierId] = useState('');
  const [vendorName, setVendorName] = useState('');
  const [category, setCategory] = useState('');
  const [amount, setAmount] = useState('');
  const [invoiceDate, setInvoiceDate] = useState('');
  const [notes, setNotes] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);

  const load = useCallback(async () => {
    if (!id) return;
    setError(null);
    const res = await receivedInvoiceService.get(id);
    if (res.success && res.data) {
      const r = res.data;
      setRow(r);
      setSupplierId(r.supplierId ?? '');
      setVendorName(r.vendorName);
      setCategory(r.category);
      setAmount(String(r.amount));
      setInvoiceDate(r.invoiceDate?.slice(0, 10) ?? '');
      setNotes(r.notes ?? '');
    } else {
      setError(res.error?.message ?? 'Registo não encontrado.');
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
    const sid = supplierId.trim();
    const res = await receivedInvoiceService.update(id, {
      supplierId: sid ? sid : undefined,
      vendorName: vendorName.trim(),
      category: category.trim(),
      amount: amt,
      invoiceDate: invoiceDate.trim(),
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
    if (!id || !window.confirm('Eliminar esta fatura recebida?')) return;
    const res = await receivedInvoiceService.remove(id);
    if (res.success) {
      router.replace('/accounting/received-invoices');
      return;
    }
    setError(res.error?.message ?? 'Erro ao eliminar.');
  }

  return (
    <AppShell
      user={user}
      subtitle={row?.vendorName ?? 'Fatura recebida'}
      activeNav="accounting"
      carsNavLabel="Viaturas"
      onLogout={logout}
      logoVariant="branded"
    >
      <p className={styles.intro}>
        <Link href="/accounting/received-invoices" className={styles.mutedLink}>
          ← Faturas recebidas
        </Link>
      </p>
      {error ? <div className={styles.error}>{error}</div> : null}
      {row ? (
        <form className={styles.form} onSubmit={onSave}>
          <div className={styles.field}>
            <label htmlFor="sid">ID fornecedor (opcional)</label>
            <input id="sid" value={supplierId} onChange={(ev) => setSupplierId(ev.target.value)} />
          </div>
          <div className={styles.field}>
            <label htmlFor="vendor">Nome do fornecedor</label>
            <input id="vendor" value={vendorName} onChange={(ev) => setVendorName(ev.target.value)} required />
          </div>
          <div className={styles.field}>
            <label htmlFor="cat">Categoria</label>
            <input id="cat" value={category} onChange={(ev) => setCategory(ev.target.value)} required />
          </div>
          <div className={styles.field}>
            <label htmlFor="amt">Valor</label>
            <input id="amt" value={amount} onChange={(ev) => setAmount(ev.target.value)} required />
          </div>
          <div className={styles.field}>
            <label htmlFor="d">Data da fatura</label>
            <input id="d" type="date" value={invoiceDate} onChange={(ev) => setInvoiceDate(ev.target.value)} required />
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
      ) : !error ? (
        <div className="loadingStack" aria-busy="true">
          <AppLoading size="md" aria-busy={false} />
          <span>A carregar…</span>
        </div>
      ) : null}
    </AppShell>
  );
}
