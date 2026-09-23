'use client';

import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { issuedInvoiceService } from '@/lib/services/issued-invoice.service';
import { apiClient as legacyApi, type Repair } from '@/lib/api';
import { apiClient } from '@/lib/api-client';
import { carService, type Car } from '@/lib/services/car.service';
import styles from '../accounting.module.css';
import { Button } from '@/components/ui/button';

type ClientOption = {
  id: string;
  email: string;
  firstName: string;
  lastName: string;
};

export interface IssuedInvoiceCreateFormProps {
  onSuccess: () => void;
  onCancel: () => void;
  initialCustomerId?: string;
  initialCarId?: string;
  initialRepairId?: string;
}

function clientLabel(c: ClientOption): string {
  const name = `${c.firstName} ${c.lastName}`.trim();
  return name ? `${name} · ${c.email}` : c.email;
}

function repairLabel(r: Repair): string {
  const st = r.status.replace(/_/g, ' ');
  return `${r.description} (${st}) — ${r.cost.toFixed(2)} €`;
}

export function IssuedInvoiceCreateForm({
  onSuccess,
  onCancel,
  initialCustomerId = '',
  initialCarId = '',
  initialRepairId = '',
}: Readonly<IssuedInvoiceCreateFormProps>) {
  const [clients, setClients] = useState<ClientOption[]>([]);
  const [clientQuery, setClientQuery] = useState('');
  const [customerId, setCustomerId] = useState(initialCustomerId);
  const [cars, setCars] = useState<Car[]>([]);
  const [carId, setCarId] = useState(initialCarId);
  const [repairs, setRepairs] = useState<Repair[]>([]);
  const [repairId, setRepairId] = useState(initialRepairId);
  const [amount, setAmount] = useState('');
  const [status, setStatus] = useState('open');
  const [notes, setNotes] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const [loadingClients, setLoadingClients] = useState(false);

  const loadClients = useCallback(async (q: string) => {
    setLoadingClients(true);
    try {
      const res = await apiClient.listClientUsers({ q: q.trim() || undefined, limit: 100 });
      if (res.success && res.data?.items) {
        setClients(res.data.items);
      } else {
        setClients([]);
      }
    } finally {
      setLoadingClients(false);
    }
  }, []);

  useEffect(() => {
    if (customerId || !initialCarId) return;
    let cancelled = false;
    void (async () => {
      const res = await carService.getCar(initialCarId);
      if (cancelled || !res.success || !res.data?.ownerId) return;
      setCustomerId(res.data.ownerId);
      setCarId(initialCarId);
    })();
    return () => {
      cancelled = true;
    };
  }, [customerId, initialCarId]);

  useEffect(() => {
    let cancelled = false;
    const t = window.setTimeout(() => {
      if (!cancelled) void loadClients(clientQuery);
    }, 200);
    return () => {
      cancelled = true;
      window.clearTimeout(t);
    };
  }, [clientQuery, loadClients]);

  useEffect(() => {
    if (!customerId) {
      setCars([]);
      setCarId('');
      return;
    }
    let cancelled = false;
    void (async () => {
      const res = await carService.getCarsByOwner(customerId);
      if (cancelled) return;
      if (res.success && res.data) {
        setCars(res.data);
        if (initialCarId && res.data.some((c) => c.id === initialCarId)) {
          setCarId(initialCarId);
        }
      } else {
        setCars([]);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [customerId, initialCarId]);

  useEffect(() => {
    if (!carId) {
      setRepairs([]);
      setRepairId('');
      return;
    }
    let cancelled = false;
    void (async () => {
      const { data, error: err } = await legacyApi.getRepairs(carId);
      if (cancelled) return;
      if (!err && data) {
        const list = Array.isArray(data) ? data : [];
        setRepairs(list);
        if (initialRepairId && list.some((r) => r.id === initialRepairId)) {
          setRepairId(initialRepairId);
        }
      } else {
        setRepairs([]);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [carId, initialRepairId]);

  useEffect(() => {
    if (!repairId) return;
    const r = repairs.find((x) => x.id === repairId);
    if (r && r.cost > 0) {
      setAmount(String(r.cost));
    }
  }, [repairId, repairs]);

  const completedRepairs = useMemo(
    () => repairs.filter((r) => r.status === 'completed'),
    [repairs],
  );

  const selectedRepair = repairs.find((r) => r.id === repairId);

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!customerId.trim()) {
      setError('Selecione um cliente.');
      return;
    }
    if (!repairId.trim()) {
      setError('Selecione uma reparação concluída.');
      return;
    }
    if (selectedRepair && selectedRepair.status !== 'completed') {
      setError('Só é possível faturar reparações concluídas.');
      return;
    }
    const amt = Number.parseFloat(amount.replace(',', '.'));
    if (Number.isNaN(amt) || amt <= 0) {
      setError('Indique um valor numérico válido.');
      return;
    }
    setSaving(true);
    setError(null);
    const res = await issuedInvoiceService.createStaff({
      customerId: customerId.trim(),
      amount: amt,
      status: status.trim() || 'open',
      notes: notes.trim(),
      repairId: repairId.trim(),
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
        <p className={styles.intro} style={{ marginBottom: '0.75rem' }}>
          Fatura interna (sem valor fiscal). Escolha cliente → viatura → reparação concluída.
        </p>

        <div className={styles.field}>
          <label htmlFor="ii-client-q">Filtrar clientes</label>
          <input
            id="ii-client-q"
            value={clientQuery}
            onChange={(ev) => setClientQuery(ev.target.value)}
            placeholder="Nome ou email…"
          />
        </div>

        <div className={styles.field}>
          <label htmlFor="ii-client">Cliente</label>
          <select
            id="ii-client"
            value={customerId}
            onChange={(ev) => {
              setCustomerId(ev.target.value);
              setCarId('');
              setRepairId('');
            }}
            required
          >
            <option value="">{loadingClients ? 'A carregar…' : 'Selecione…'}</option>
            {clients.map((c) => (
              <option key={c.id} value={c.id}>
                {clientLabel(c)}
              </option>
            ))}
          </select>
        </div>

        <div className={styles.field}>
          <label htmlFor="ii-car">Viatura</label>
          <select
            id="ii-car"
            value={carId}
            onChange={(ev) => {
              setCarId(ev.target.value);
              setRepairId('');
            }}
            required
            disabled={!customerId}
          >
            <option value="">{customerId ? 'Selecione…' : 'Escolha o cliente primeiro'}</option>
            {cars.map((c) => (
              <option key={c.id} value={c.id}>
                {c.make} {c.model} · {c.licensePlate}
              </option>
            ))}
          </select>
        </div>

        <div className={styles.field}>
          <label htmlFor="ii-repair">Reparação concluída</label>
          <select
            id="ii-repair"
            value={repairId}
            onChange={(ev) => setRepairId(ev.target.value)}
            required
            disabled={!carId}
          >
            <option value="">{carId ? 'Selecione…' : 'Escolha a viatura primeiro'}</option>
            {completedRepairs.map((r) => (
              <option key={r.id} value={r.id}>
                {repairLabel(r)}
              </option>
            ))}
          </select>
          {carId && completedRepairs.length === 0 ? (
            <span className={styles.mutedLink} style={{ marginTop: '0.35rem', display: 'block' }}>
              Não há reparações concluídas nesta viatura.
            </span>
          ) : null}
        </div>

        <div className={styles.field}>
          <label htmlFor="ii-amt">Valor</label>
          <input id="ii-amt" value={amount} onChange={(ev) => setAmount(ev.target.value)} required />
        </div>
        <div className={styles.field}>
          <label htmlFor="ii-st">Estado</label>
          <input id="ii-st" value={status} onChange={(ev) => setStatus(ev.target.value)} placeholder="open" />
        </div>
        <div className={styles.field}>
          <label htmlFor="ii-notes">Notas</label>
          <textarea id="ii-notes" value={notes} onChange={(ev) => setNotes(ev.target.value)} />
        </div>
        <div className={styles.rowActions}>
          <Button type="button" variant="outline" onClick={onCancel} disabled={saving}>
            Cancelar
          </Button>
          <Button type="submit" className={styles.submitButton} disabled={saving || !repairId}>
            {saving ? 'A guardar…' : 'Criar fatura interna'}
          </Button>
        </div>
      </form>
    </div>
  );
}
