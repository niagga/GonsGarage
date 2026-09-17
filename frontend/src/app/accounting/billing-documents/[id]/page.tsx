'use client';

import React, { useCallback, useEffect, useState } from 'react';
import Link from 'next/link';
import { useParams, useRouter } from 'next/navigation';
import { useAuth } from '@/stores';
import AppShell from '@/components/layout/AppShell';
import { billingDocumentService } from '@/lib/services/billing-document.service';
import type { BillingDocument, BillingDocumentKind } from '@/types/accounting';
import styles from '../../accounting.module.css';
import { AppLoading } from '@/components/ui/AppLoading';
import { Button } from '@/components/ui/button';

const KINDS: { value: BillingDocumentKind; label: string }[] = [
  { value: 'payroll', label: 'Salários' },
  { value: 'irs', label: 'IRS' },
  { value: 'other', label: 'Outro' },
];

export default function BillingDocumentDetailPage() {
  const { id } = useParams<{ id: string }>();
  const { user, logout } = useAuth();
  const router = useRouter();
  const [row, setRow] = useState<BillingDocument | null>(null);
  const [kind, setKind] = useState<BillingDocumentKind>('other');
  const [title, setTitle] = useState('');
  const [amount, setAmount] = useState('');
  const [customerId, setCustomerId] = useState('');
  const [reference, setReference] = useState('');
  const [notes, setNotes] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);

  const load = useCallback(async () => {
    if (!id) return;
    setError(null);
    const res = await billingDocumentService.get(id);
    if (res.success && res.data) {
      const d = res.data;
      setRow(d);
      setKind(d.kind);
      setTitle(d.title);
      setAmount(String(d.amount));
      setCustomerId(d.customerId ?? '');
      setReference(d.reference);
      setNotes(d.notes ?? '');
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
    const cid = customerId.trim();
    const res = await billingDocumentService.update(id, {
      kind,
      title: title.trim(),
      amount: amt,
      customerId: cid ? cid : undefined,
      reference: reference.trim(),
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
    if (!id || !window.confirm('Eliminar este documento?')) return;
    const res = await billingDocumentService.remove(id);
    if (res.success) {
      router.replace('/accounting/billing-documents');
      return;
    }
    setError(res.error?.message ?? 'Erro ao eliminar.');
  }

  return (
    <AppShell
      user={user}
      subtitle={row?.title ?? 'Documento'}
      activeNav="accounting"
      carsNavLabel="Viaturas"
      onLogout={logout}
      logoVariant="branded"
    >
      <p className={styles.intro}>
        <Link href="/accounting/billing-documents" className={styles.mutedLink}>
          ← Documentos
        </Link>
      </p>
      {error ? <div className={styles.error}>{error}</div> : null}
      {row ? (
        <form className={styles.form} onSubmit={onSave}>
          <div className={styles.field}>
            <label htmlFor="kind">Tipo</label>
            <select id="kind" value={kind} onChange={(ev) => setKind(ev.target.value as BillingDocumentKind)}>
              {KINDS.map((k) => (
                <option key={k.value} value={k.value}>
                  {k.label}
                </option>
              ))}
            </select>
          </div>
          <div className={styles.field}>
            <label htmlFor="title">Título</label>
            <input id="title" value={title} onChange={(ev) => setTitle(ev.target.value)} required />
          </div>
          <div className={styles.field}>
            <label htmlFor="amt">Valor</label>
            <input id="amt" value={amount} onChange={(ev) => setAmount(ev.target.value)} required />
          </div>
          <div className={styles.field}>
            <label htmlFor="cid">ID cliente (opcional)</label>
            <input id="cid" value={customerId} onChange={(ev) => setCustomerId(ev.target.value)} />
          </div>
          <div className={styles.field}>
            <label htmlFor="ref">Referência</label>
            <input id="ref" value={reference} onChange={(ev) => setReference(ev.target.value)} required />
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
