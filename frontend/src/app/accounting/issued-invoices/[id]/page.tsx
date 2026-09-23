'use client';

import React, { useCallback, useEffect, useState } from 'react';
import Link from 'next/link';
import { useParams, useRouter } from 'next/navigation';
import { useAuth } from '@/stores';
import AppShell from '@/components/layout/AppShell';
import { issuedInvoiceService } from '@/lib/services/issued-invoice.service';
import { fiscalizationService } from '@/lib/services/fiscalization.service';
import type { IssuedInvoice } from '@/types/accounting';
import type { FiscalizationProjection } from '@/types/fiscal';
import { fiscalPresentationLabel } from '@/types/fiscal';
import styles from '../../accounting.module.css';
import { AppLoading } from '@/components/ui/AppLoading';
import { Button } from '@/components/ui/button';
import { FiscalDraftForm } from './FiscalDraftForm';
import { FiscalActions } from './FiscalActions';

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
  const [fiscal, setFiscal] = useState<FiscalizationProjection | null>(null);

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
      return;
    }
    const proj = await fiscalizationService.getProjection(id);
    if (proj.success && proj.data) {
      setFiscal(proj.data);
    } else if (proj.error?.status !== 404) {
      // Keep operational form usable when fiscal feature is off or unavailable.
      setFiscal(null);
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

  const showDraft =
    !fiscal ||
    fiscal.status === 'draft' ||
    fiscal.status === 'legacy_unfiscalized' ||
    (fiscal.allowedActions ?? []).includes('edit');

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

          <section className={styles.form} aria-labelledby="fiscal-heading">
            <h2 id="fiscal-heading">Fiscalização</h2>
            {fiscal ? (
              <p>
                Estado fiscal: <strong>{fiscalPresentationLabel(fiscal.status)}</strong>
              </p>
            ) : (
              <p>Sem projeção fiscal carregada.</p>
            )}
            {id && showDraft ? (
              <FiscalDraftForm
                invoiceId={id}
                projection={fiscal}
                onSaved={(p) => setFiscal(p)}
              />
            ) : null}
            {id && fiscal ? (
              <FiscalActions
                invoiceId={id}
                projection={fiscal}
                role={user.role}
                onProjectionChange={setFiscal}
              />
            ) : null}
          </section>
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
