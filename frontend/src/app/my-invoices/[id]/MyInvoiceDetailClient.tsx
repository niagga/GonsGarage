'use client';

import React, { useCallback, useEffect, useOptimistic, useState } from 'react';
import Link from 'next/link';
import { useAuth } from '@/stores';
import AppShell from '@/components/layout/AppShell';
import { FiscalStatusBadge } from '@/components/fiscal/FiscalStatusBadge';
import { Button } from '@/components/ui/button';
import { AppLoading } from '@/components/ui/AppLoading';
import { issuedInvoiceService } from '@/lib/services/issued-invoice.service';
import { fiscalizationService } from '@/lib/services/fiscalization.service';
import type { IssuedInvoice } from '@/types/accounting';
import {
  canDownloadFiscalArtifact,
  type FiscalizationProjection,
} from '@/types/fiscal';
import styles from '../../accounting/accounting.module.css';

export type MyInvoiceDetailClientProps = Readonly<{
  invoiceId: string;
  /** When set (cookie-based server fetch), skip the first client GET for the same id. */
  initialRow: IssuedInvoice | null;
}>;

export default function MyInvoiceDetailClient({ invoiceId, initialRow }: MyInvoiceDetailClientProps) {
  const { user, logout } = useAuth();
  const [row, setRow] = useState<IssuedInvoice | null>(initialRow);
  const [notes, setNotes] = useState(initialRow?.notes ?? '');
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const [projection, setProjection] = useState<FiscalizationProjection | null>(null);
  const [fiscalMessage, setFiscalMessage] = useState<string | null>(null);
  const [pdfBusy, setPdfBusy] = useState(false);

  const [optimisticRow, mergeOptimisticNotes] = useOptimistic(
    row,
    (current, nextNotes: string) => {
      if (!current) return current;
      return { ...current, notes: nextNotes };
    },
  );

  const shownRow = optimisticRow;

  const loadProjection = useCallback(async () => {
    if (!invoiceId) return;
    setFiscalMessage(null);
    const res = await fiscalizationService.getProjection(invoiceId);
    if (res.success && res.data) {
      setProjection(res.data);
      if (res.data.artifactStatus === 'unavailable' || res.data.artifactStatus === 'compromised') {
        setFiscalMessage(
          res.data.artifactStatus === 'compromised'
            ? 'O PDF fiscal está comprometido e não pode ser descarregado.'
            : 'PDF temporariamente indisponível.',
        );
      }
      return;
    }
    setProjection(null);
    if (res.error?.status === 404) {
      setError('Fatura não encontrada.');
      return;
    }
    if (res.error?.status === 403) {
      setError('Não foi possível aceder a esta fatura.');
    }
  }, [invoiceId]);

  const load = useCallback(async () => {
    if (!invoiceId) return;
    setError(null);
    const res = await issuedInvoiceService.get(invoiceId);
    if (res.success && res.data) {
      setRow(res.data);
      setNotes(res.data.notes ?? '');
      await loadProjection();
    } else {
      setError(res.error?.message ?? 'Fatura não encontrada.');
      setRow(null);
      setProjection(null);
    }
  }, [invoiceId, loadProjection]);

  useEffect(() => {
    let cancelled = false;
    queueMicrotask(() => {
      if (cancelled) return;
      if (initialRow) {
        setRow(initialRow);
        setNotes(initialRow.notes ?? '');
        void loadProjection();
        return;
      }
      void load();
    });
    return () => {
      cancelled = true;
    };
  }, [initialRow, load, loadProjection]);

  if (!user) return null;

  async function saveNotesAction(formData: FormData) {
    if (!invoiceId) return;
    const snapshot = row;
    if (!snapshot) return;
    const notesToSave = String(formData.get('notes') ?? notes);
    setError(null);
    mergeOptimisticNotes(notesToSave);
    setSaving(true);
    try {
      const res = await issuedInvoiceService.patchNotes(invoiceId, notesToSave);
      if (res.success && res.data) {
        setRow(res.data);
        setNotes(res.data.notes ?? '');
        return;
      }
      setError(res.error?.message ?? 'Erro ao guardar as notas.');
      setRow(snapshot);
    } finally {
      setSaving(false);
    }
  }

  async function onDownloadPdf() {
    if (!projection || !canDownloadFiscalArtifact(projection) || !projection.artifactId) {
      setFiscalMessage('PDF temporariamente indisponível.');
      return;
    }
    setPdfBusy(true);
    setFiscalMessage(null);
    const res = await fiscalizationService.downloadArtifact(invoiceId, projection.artifactId);
    setPdfBusy(false);
    if (!res.success || !res.data) {
      setFiscalMessage(
        res.error?.status === 503
          ? 'PDF temporariamente indisponível.'
          : (res.error?.message ?? 'Não foi possível descarregar o PDF.'),
      );
      return;
    }
    const url = URL.createObjectURL(res.data);
    const a = document.createElement('a');
    a.href = url;
    a.download = `fiscal-${invoiceId}.pdf`;
    a.click();
    URL.revokeObjectURL(url);
  }

  const showPdfButton = projection ? canDownloadFiscalArtifact(projection) : false;

  return (
    <AppShell
      user={user}
      subtitle="Detalhe da fatura"
      activeNav="my_invoices"
      onLogout={logout}
      logoVariant="branded"
    >
      <p className={styles.intro}>
        <Link href="/my-invoices" className={styles.mutedLink}>
          ← As minhas faturas
        </Link>
      </p>
      {error ? <div className={styles.error}>{error}</div> : null}
      {shownRow ? (
        <>
          <span className="sr-only" data-testid="invoice-notes-optimistic">
            {shownRow.notes}
          </span>
          <div className={styles.readonlyBlock}>
            <div>
              <strong>Valor:</strong> {shownRow.amount.toFixed(2)} €
            </div>
            <div>
              <strong>Estado:</strong> {shownRow.status}
            </div>
            <div>
              <strong>Fiscalização:</strong>{' '}
              {projection ? (
                <FiscalStatusBadge status={projection.status} clientSimplified />
              ) : (
                '—'
              )}
            </div>
            <div>
              <strong>Criada:</strong> {shownRow.createdAt?.slice(0, 19).replace('T', ' ') ?? '—'}
            </div>
          </div>
          {fiscalMessage ? (
            <div className={styles.error} role="status">
              {fiscalMessage}
            </div>
          ) : null}
          {showPdfButton ? (
            <div className={styles.rowActions}>
              <Button type="button" disabled={pdfBusy} onClick={() => void onDownloadPdf()}>
                Descarregar PDF
              </Button>
            </div>
          ) : null}
          <form className={styles.form} action={saveNotesAction}>
            <div className={styles.field}>
              <label htmlFor="notes">As suas notas (editável)</label>
              <textarea id="notes" name="notes" value={notes} onChange={(ev) => setNotes(ev.target.value)} />
            </div>
            <div className={styles.rowActions}>
              <button type="submit" className={styles.submitButton} disabled={saving}>
                {saving ? 'A guardar…' : 'Guardar notas'}
              </button>
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
