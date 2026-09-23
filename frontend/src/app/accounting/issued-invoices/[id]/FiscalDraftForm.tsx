'use client';

import React, { useState } from 'react';
import { Button } from '@/components/ui/button';
import { fiscalizationService } from '@/lib/services/fiscalization.service';
import type { FiscalDraftRequest, FiscalizationProjection } from '@/types/fiscal';
import styles from '../../accounting.module.css';

export interface FiscalDraftFormProps {
  invoiceId: string;
  projection: FiscalizationProjection | null;
  onSaved: (projection: FiscalizationProjection) => void;
}

function parseSourceRef(raw: string): { type: string; id: string } | undefined {
  const trimmed = raw.trim();
  if (!trimmed) return undefined;
  const idx = trimmed.indexOf(':');
  if (idx <= 0) return undefined;
  return { type: trimmed.slice(0, idx), id: trimmed.slice(idx + 1) };
}

export function FiscalDraftForm({ invoiceId, projection, onSaved }: Readonly<FiscalDraftFormProps>) {
  const [kind, setKind] = useState(projection?.kind === 'FR' ? 'FR' : 'FT');
  const [description, setDescription] = useState('Linha fiscal');
  const [quantity, setQuantity] = useState('1.000');
  const [unitPrice, setUnitPrice] = useState('0.00');
  const [taxTreatment, setTaxTreatment] = useState('IVA23');
  const [taxRate, setTaxRate] = useState('0.23');
  const [exemptionCode, setExemptionCode] = useState('');
  const [sourceRef, setSourceRef] = useState('');
  const [legalName, setLegalName] = useState('');
  const [taxId, setTaxId] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (taxTreatment === 'exempt' && !exemptionCode.trim()) {
      setError('Indique o código de isenção.');
      return;
    }
    const source = parseSourceRef(sourceRef);
    const body: FiscalDraftRequest = {
      version: projection?.version ?? 0,
      kind,
      policyKey: 'pt-sales',
      currency: projection?.currency || 'EUR',
      customer: {
        legalName: legalName.trim() || 'Cliente',
        taxIdentifier: taxId.trim() || '000000000',
        countryCode: 'PT',
      },
      billingAddress: {
        line1: '—',
        postalCode: '0000-000',
        city: '—',
        countryCode: 'PT',
      },
      lines: [
        {
          position: 1,
          description: description.trim() || 'Linha',
          quantity: quantity.trim(),
          unitCode: 'C62',
          unitPrice: unitPrice.trim(),
          discount: { kind: 'none', value: '0' },
          taxTreatmentCode: taxTreatment,
          taxRate: taxTreatment === 'exempt' ? '0' : taxRate.trim(),
          exemptionCode: taxTreatment === 'exempt' ? exemptionCode.trim() : null,
          ...(source ? { source } : {}),
        },
      ],
    };

    setSaving(true);
    setError(null);
    const res = await fiscalizationService.upsertDraft(invoiceId, body);
    setSaving(false);
    if (res.success && res.data) {
      onSaved(res.data);
      return;
    }
    if (res.error?.status === 409) {
      setError('Versão desatualizada. Recarregue e tente novamente.');
      return;
    }
    setError(res.error?.message ?? 'Erro ao guardar o rascunho.');
  }

  const readiness = projection?.readinessIssues ?? [];

  return (
    <form className={styles.form} onSubmit={(ev) => void onSubmit(ev)} aria-label="Rascunho fiscal">
      <div className={styles.field}>
        <label htmlFor="fiscal-kind">Tipo de documento</label>
        <select
          id="fiscal-kind"
          value={kind}
          onChange={(ev) => setKind(ev.target.value)}
        >
          <option value="FT">FT</option>
          <option value="FR">FR</option>
        </select>
      </div>
      <div className={styles.field}>
        <label htmlFor="fiscal-desc">Descrição</label>
        <input
          id="fiscal-desc"
          value={description}
          onChange={(ev) => setDescription(ev.target.value)}
        />
      </div>
      <div className={styles.field}>
        <label htmlFor="fiscal-qty">Quantidade</label>
        <input
          id="fiscal-qty"
          value={quantity}
          onChange={(ev) => setQuantity(ev.target.value)}
          inputMode="decimal"
          required
        />
      </div>
      <div className={styles.field}>
        <label htmlFor="fiscal-price">Preço unitário</label>
        <input
          id="fiscal-price"
          value={unitPrice}
          onChange={(ev) => setUnitPrice(ev.target.value)}
          inputMode="decimal"
          required
        />
      </div>
      <div className={styles.field}>
        <label htmlFor="fiscal-tax">Tratamento fiscal</label>
        <select
          id="fiscal-tax"
          value={taxTreatment}
          onChange={(ev) => setTaxTreatment(ev.target.value)}
        >
          <option value="IVA23">IVA 23%</option>
          <option value="IVA13">IVA 13%</option>
          <option value="IVA6">IVA 6%</option>
          <option value="exempt">Isento</option>
        </select>
      </div>
      {taxTreatment === 'exempt' ? (
        <div className={styles.field}>
          <label htmlFor="fiscal-exemption">Código de isenção</label>
          <input
            id="fiscal-exemption"
            value={exemptionCode}
            onChange={(ev) => setExemptionCode(ev.target.value)}
          />
        </div>
      ) : (
        <div className={styles.field}>
          <label htmlFor="fiscal-rate">Taxa</label>
          <input
            id="fiscal-rate"
            value={taxRate}
            onChange={(ev) => setTaxRate(ev.target.value)}
            inputMode="decimal"
          />
        </div>
      )}
      <div className={styles.field}>
        <label htmlFor="fiscal-source">Referência de origem</label>
        <input
          id="fiscal-source"
          value={sourceRef}
          onChange={(ev) => setSourceRef(ev.target.value)}
          placeholder="repair:uuid"
        />
      </div>
      <div className={styles.field}>
        <label htmlFor="fiscal-customer">Nome legal do cliente</label>
        <input
          id="fiscal-customer"
          value={legalName}
          onChange={(ev) => setLegalName(ev.target.value)}
        />
      </div>
      <div className={styles.field}>
        <label htmlFor="fiscal-nif">NIF</label>
        <input id="fiscal-nif" value={taxId} onChange={(ev) => setTaxId(ev.target.value)} />
      </div>

      <p className={styles.intro}>
        Totais calculados pelo servidor
        {projection?.payableTotal ? `: ${projection.payableTotal}` : ''}
        {projection?.currency ? ` ${projection.currency}` : ''}
      </p>
      {readiness.length > 0 ? (
        <div className={styles.error} role="status">
          <strong>Prontidão:</strong>{' '}
          {readiness.join(', ')}
        </div>
      ) : null}
      {error ? <div className={styles.error} role="alert">{error}</div> : null}

      <div className={styles.rowActions}>
        <Button type="submit" className={styles.submitButton} disabled={saving}>
          {saving ? 'A guardar…' : 'Guardar rascunho'}
        </Button>
      </div>
    </form>
  );
}
