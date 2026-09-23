'use client';

import React, { useCallback, useEffect, useRef, useState } from 'react';
import { Button } from '@/components/ui/button';
import { fiscalizationService } from '@/lib/services/fiscalization.service';
import {
  canShowFiscalAction,
  DEFAULT_FISCAL_MAX_POLL_ATTEMPTS,
  DEFAULT_FISCAL_POLL_INTERVAL_MS,
  isTransientFiscalLifecycle,
  type FiscalizationProjection,
} from '@/types/fiscal';
import { FiscalStatusBadge } from '@/components/fiscal/FiscalStatusBadge';
import styles from '../../accounting.module.css';

export interface FiscalActionsProps {
  invoiceId: string;
  projection: FiscalizationProjection;
  role: string;
  onProjectionChange: (projection: FiscalizationProjection) => void;
  pollIntervalMs?: number;
  maxPollAttempts?: number;
  showMockLabels?: boolean;
}

export function FiscalActions({
  invoiceId,
  projection,
  role,
  onProjectionChange,
  pollIntervalMs = DEFAULT_FISCAL_POLL_INTERVAL_MS,
  maxPollAttempts = DEFAULT_FISCAL_MAX_POLL_ATTEMPTS,
  showMockLabels = process.env.NODE_ENV !== 'production',
}: Readonly<FiscalActionsProps>) {
  const [submitting, setSubmitting] = useState(false);
  const [progress, setProgress] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const pollCancelled = useRef(false);
  const pollTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const submittingRef = useRef(false);

  const stopPolling = useCallback(() => {
    pollCancelled.current = true;
    if (pollTimer.current) {
      clearTimeout(pollTimer.current);
      pollTimer.current = null;
    }
  }, []);

  useEffect(() => {
    pollCancelled.current = false;
    return () => {
      stopPolling();
    };
  }, [stopPolling]);

  const startPolling = useCallback(
    (from: FiscalizationProjection) => {
      pollCancelled.current = false;
      setProgress(true);
      let attempts = 0;

      const tick = async () => {
        if (pollCancelled.current) return;
        attempts += 1;
        const res = await fiscalizationService.getProjection(invoiceId);
        if (pollCancelled.current) return;
        if (res.success && res.data) {
          onProjectionChange(res.data);
          if (!isTransientFiscalLifecycle(res.data.lifecycle) || attempts >= maxPollAttempts) {
            setProgress(false);
            return;
          }
        } else if (attempts >= maxPollAttempts) {
          setProgress(false);
          return;
        }
        pollTimer.current = setTimeout(() => {
          void tick();
        }, pollIntervalMs);
      };

      if (isTransientFiscalLifecycle(from.lifecycle)) {
        pollTimer.current = setTimeout(() => {
          void tick();
        }, pollIntervalMs);
      } else {
        setProgress(false);
      }
    },
    [invoiceId, maxPollAttempts, onProjectionChange, pollIntervalMs],
  );

  const runAction = async (
    action: 'finalize' | 'retry' | 'reconcile' | 'void',
  ) => {
    if (submittingRef.current) return;
    submittingRef.current = true;
    setSubmitting(true);
    setError(null);
    const body = { expectedVersion: projection.version ?? 0 };
    let res;
    try {
      switch (action) {
        case 'finalize':
          res = await fiscalizationService.finalize(invoiceId, body);
          break;
        case 'retry':
          res = await fiscalizationService.retry(invoiceId, body);
          break;
        case 'reconcile':
          res = await fiscalizationService.reconcile(invoiceId, body);
          break;
        case 'void':
          res = await fiscalizationService.voidDocument(invoiceId, body);
          break;
      }
    } finally {
      submittingRef.current = false;
      setSubmitting(false);
    }
    if (res.success && res.data) {
      onProjectionChange(res.data);
      if (isTransientFiscalLifecycle(res.data.lifecycle)) {
        startPolling(res.data);
      }
      return;
    }
    if (res.error?.status === 409) {
      setError('Versão desatualizada ou conflito de estado. Recarregue e tente novamente.');
      return;
    }
    setError(res.error?.message ?? 'Ação fiscal falhou.');
  };

  async function onDownload() {
    if (!projection.artifactId) {
      setError(
        'PDF arquivado, mas o identificador ainda não está disponível nesta projeção. Tente mais tarde.',
      );
      return;
    }
    setSubmitting(true);
    setError(null);
    const res = await fiscalizationService.downloadArtifact(invoiceId, projection.artifactId);
    setSubmitting(false);
    if (!res.success || !res.data) {
      setError(
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

  const allowed = projection.allowedActions ?? [];
  const showFinalize = canShowFiscalAction(role, 'finalize', allowed);
  const showRetry = canShowFiscalAction(role, 'retry', allowed);
  const showReconcile = canShowFiscalAction(role, 'reconcile', allowed);
  const showVoid = canShowFiscalAction(role, 'void', allowed);
  const canDownload =
    projection.artifactStatus === 'available' || Boolean(projection.artifactId);

  return (
    <div className={styles.form}>
      <p>
        Estado: <strong><FiscalStatusBadge status={projection.status} /></strong>
        {projection.lifecycle ? ` (${projection.lifecycle})` : ''}
      </p>
      {progress ? (
        <p role="status">Em progresso — a atualizar o estado…</p>
      ) : null}
      {projection.status === 'unavailable' || projection.lifecycle === 'outcome_unknown' ? (
        <div className={styles.error} role="status">
          {projection.lastErrorMessage ||
            'Estado indeterminado ou indisponível. Utilize reconciliar quando autorizado.'}
        </div>
      ) : null}
      {showMockLabels && projection.artifactClassification === 'mock' ? (
        <div className={styles.error} role="note">
          SEM VALIDADE FISCAL — MOCK
        </div>
      ) : null}
      {error ? <div className={styles.error}>{error}</div> : null}
      <div className={styles.rowActions}>
        {showFinalize ? (
          <Button
            type="button"
            disabled={submitting}
            onClick={() => void runAction('finalize')}
          >
            Finalizar
          </Button>
        ) : null}
        {showRetry ? (
          <Button type="button" disabled={submitting} onClick={() => void runAction('retry')}>
            Repetir
          </Button>
        ) : null}
        {showReconcile ? (
          <Button
            type="button"
            disabled={submitting}
            onClick={() => void runAction('reconcile')}
          >
            Reconciliar
          </Button>
        ) : null}
        {showVoid ? (
          <Button
            type="button"
            variant="destructive"
            disabled={submitting}
            onClick={() => void runAction('void')}
          >
            Anular
          </Button>
        ) : null}
        {canDownload ? (
          <Button type="button" disabled={submitting} onClick={() => void onDownload()}>
            Descarregar PDF
          </Button>
        ) : null}
      </div>
    </div>
  );
}
