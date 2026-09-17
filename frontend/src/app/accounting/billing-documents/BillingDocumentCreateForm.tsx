'use client';

import React, { useState } from 'react';
import { billingDocumentService } from '@/lib/services/billing-document.service';
import type { BillingDocumentKind } from '@/types/accounting';
import styles from '../accounting.module.css';
import { Button } from '@/components/ui/button';

const KINDS: { value: BillingDocumentKind; label: string }[] = [
  { value: 'payroll', label: 'Salários' },
  { value: 'irs', label: 'IRS' },
  { value: 'other', label: 'Outro' },
];

export interface BillingDocumentCreateFormProps {
  onSuccess: () => void;
  onCancel: () => void;
}

export function BillingDocumentCreateForm({ onSuccess, onCancel }: Readonly<BillingDocumentCreateFormProps>) {
  const [kind, setKind] = useState<BillingDocumentKind>('other');
  const [title, setTitle] = useState('');
  const [amount, setAmount] = useState('');
  const [customerId, setCustomerId] = useState('');
  const [reference, setReference] = useState('');
  const [notes, setNotes] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    const amt = Number.parseFloat(amount.replace(',', '.'));
    if (Number.isNaN(amt)) {
      setError('Indique um valor numérico válido.');
      return;
    }
    setSaving(true);
    setError(null);
    const cid = customerId.trim();
    const res = await billingDocumentService.create({
      kind,
      title: title.trim(),
      amount: amt,
      customerId: cid || undefined,
      reference: reference.trim(),
      notes: notes.trim(),
    });
    setSaving(false);
    if (res.success && res.data?.id) {
      onSuccess();
      return;
    }
    setError(res.error?.message ?? 'Erro ao criar.');
  }

  return (
    <div>
      {error ? <div className={styles.error}>{error}</div> : null}
      <form className={styles.form} onSubmit={onSubmit}>
        <div className={styles.field}>
          <label htmlFor="bd-kind">Tipo</label>
          <select id="bd-kind" value={kind} onChange={(ev) => setKind(ev.target.value as BillingDocumentKind)}>
            {KINDS.map((k) => (
              <option key={k.value} value={k.value}>
                {k.label}
              </option>
            ))}
          </select>
        </div>
        <div className={styles.field}>
          <label htmlFor="bd-title">Título</label>
          <input id="bd-title" value={title} onChange={(ev) => setTitle(ev.target.value)} required />
        </div>
        <div className={styles.field}>
          <label htmlFor="bd-amt">Valor</label>
          <input id="bd-amt" value={amount} onChange={(ev) => setAmount(ev.target.value)} required />
        </div>
        <div className={styles.field}>
          <label htmlFor="bd-cid">ID cliente (opcional, UUID)</label>
          <input id="bd-cid" value={customerId} onChange={(ev) => setCustomerId(ev.target.value)} />
        </div>
        <div className={styles.field}>
          <label htmlFor="bd-ref">Referência</label>
          <input id="bd-ref" value={reference} onChange={(ev) => setReference(ev.target.value)} required />
        </div>
        <div className={styles.field}>
          <label htmlFor="bd-notes">Notas</label>
          <textarea id="bd-notes" value={notes} onChange={(ev) => setNotes(ev.target.value)} />
        </div>
        <div className={styles.rowActions}>
          <Button type="button" variant="outline" onClick={onCancel} disabled={saving}>
            Cancelar
          </Button>
          <Button type="submit" className={styles.submitButton} disabled={saving}>
            {saving ? 'A guardar…' : 'Criar'}
          </Button>
        </div>
      </form>
    </div>
  );
}
