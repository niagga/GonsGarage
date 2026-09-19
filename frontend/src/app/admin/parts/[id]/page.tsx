'use client';

import React, { useCallback, useEffect, useState } from 'react';
import Link from 'next/link';
import { useParams, useRouter } from 'next/navigation';
import { useAuth } from '@/stores';
import AppShell from '@/components/layout/AppShell';
import { apiClient } from '@/lib/api-client';
import type { PartItem, PartItemWriteBody, PartUOM } from '@/types/parts';
import styles from '../admin-parts.module.css';
import { AppLoading } from '@/components/ui/AppLoading';
import { Button } from '@/components/ui/button';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { PART_UOM_OPTIONS } from '../part-uom-options';

export default function AdminPartDetailPage() {
  const { id } = useParams<{ id: string }>();
  const { user, logout } = useAuth();
  const router = useRouter();
  const [row, setRow] = useState<PartItem | null>(null);
  const [reference, setReference] = useState('');
  const [brand, setBrand] = useState('');
  const [name, setName] = useState('');
  const [barcode, setBarcode] = useState('');
  const [quantity, setQuantity] = useState('0');
  const [uom, setUom] = useState<PartUOM>('unit');
  const [minimumQuantity, setMinimumQuantity] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);

  const load = useCallback(async () => {
    if (!id) return;
    setError(null);
    const res = await apiClient.getPart(id);
    if (res.success && res.data) {
      const p = res.data;
      setRow(p);
      setReference(p.reference);
      setBrand(p.brand);
      setName(p.name);
      setBarcode(p.barcode ?? '');
      setQuantity(String(p.quantity));
      setUom((p.uom === 'liter' ? 'liter' : 'unit') as PartUOM);
      setMinimumQuantity(
        p.minimumQuantity != null && Number.isFinite(p.minimumQuantity) ? String(p.minimumQuantity) : '',
      );
    } else {
      setError(res.error?.message ?? 'Peça não encontrada.');
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
    setSaving(true);
    setError(null);
    const qty = Number(quantity);
    const body: PartItemWriteBody = {
      reference: reference.trim(),
      brand: brand.trim(),
      name: name.trim(),
      barcode: barcode.trim(),
      quantity: Number.isFinite(qty) ? qty : 0,
      uom,
    };
    const min = minimumQuantity.trim();
    if (min !== '') {
      const m = parseFloat(min);
      body.minimumQuantity = Number.isFinite(m) ? m : null;
    } else {
      body.minimumQuantity = null;
    }
    const res = await apiClient.updatePart(id, body);
    setSaving(false);
    if (res.success && res.data) {
      setRow(res.data);
      return;
    }
    setError(res.error?.message ?? 'Erro ao guardar.');
  }

  async function onDelete() {
    if (!id || !window.confirm('Eliminar esta peça do inventário?')) return;
    const res = await apiClient.deletePart(id);
    if (res.success) {
      router.replace('/admin/parts');
      return;
    }
    setError(res.error?.message ?? 'Erro ao eliminar.');
  }

  return (
    <AppShell
      user={user}
      subtitle={row?.name ?? 'Peça'}
      activeNav="admin_parts"
      carsNavLabel="Viaturas"
      onLogout={logout}
      logoVariant="branded"
      toolbar={
        <>
          <h1>Editar peça</h1>
          <Button type="button" variant="outline" asChild>
            <Link href="/admin/parts">Voltar à lista</Link>
          </Button>
        </>
      }
    >
      <p className={styles.intro}>
        <Link href="/admin/parts" className={styles.mutedLink}>
          ← Peças (stock)
        </Link>
      </p>
      {error ? <div className={styles.error}>{error}</div> : null}
      {row ? (
        <form className={styles.form} onSubmit={(ev) => void onSave(ev)}>
          <div className={styles.field}>
            <label htmlFor="reference">Referência</label>
            <input id="reference" value={reference} onChange={(ev) => setReference(ev.target.value)} required />
          </div>
          <div className={styles.field}>
            <label htmlFor="brand">Marca</label>
            <input id="brand" value={brand} onChange={(ev) => setBrand(ev.target.value)} required />
          </div>
          <div className={styles.field}>
            <label htmlFor="name">Nome</label>
            <input id="name" value={name} onChange={(ev) => setName(ev.target.value)} required />
          </div>
          <div className={styles.field}>
            <label htmlFor="barcode">Código de barras</label>
            <input id="barcode" value={barcode} onChange={(ev) => setBarcode(ev.target.value)} />
          </div>
          <div className={styles.field}>
            <label htmlFor="quantity">Quantidade</label>
            <input
              id="quantity"
              type="number"
              min={0}
              step="any"
              value={quantity}
              onChange={(ev) => setQuantity(ev.target.value)}
              required
            />
          </div>
          <div className={styles.field}>
            <label htmlFor="uom">Unidade de medida</label>
            <Select value={uom} onValueChange={(value) => setUom(value as PartUOM)}>
              <SelectTrigger id="uom">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {PART_UOM_OPTIONS.map((o) => (
                  <SelectItem key={o.value} value={o.value}>
                    {o.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div className={styles.field}>
            <label htmlFor="minimumQuantity">Quantidade mínima (opcional)</label>
            <input
              id="minimumQuantity"
              type="number"
              min={0}
              step="any"
              value={minimumQuantity}
              onChange={(ev) => setMinimumQuantity(ev.target.value)}
            />
          </div>
          <div className={styles.rowActions}>
            <Button type="submit" disabled={saving}>
              {saving ? 'A guardar…' : 'Guardar'}
            </Button>
            <Button type="button" variant="destructive" onClick={() => void onDelete()}>
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
