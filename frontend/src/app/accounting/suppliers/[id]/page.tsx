'use client';

import React, { useCallback, useEffect, useState } from 'react';
import Link from 'next/link';
import { useParams, useRouter } from 'next/navigation';
import { useAuth } from '@/stores';
import AppShell from '@/components/layout/AppShell';
import { supplierService } from '@/lib/services/supplier.service';
import type { Supplier } from '@/types/accounting';
import styles from '../../accounting.module.css';
import { AppLoading } from '@/components/ui/AppLoading';
import { Button } from '@/components/ui/button';

export default function SupplierDetailPage() {
  const { id } = useParams<{ id: string }>();
  const { user, logout } = useAuth();
  const router = useRouter();
  const [row, setRow] = useState<Supplier | null>(null);
  const [name, setName] = useState('');
  const [contactEmail, setContactEmail] = useState('');
  const [contactPhone, setContactPhone] = useState('');
  const [taxId, setTaxId] = useState('');
  const [notes, setNotes] = useState('');
  const [isActive, setIsActive] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);

  const load = useCallback(async () => {
    if (!id) return;
    setError(null);
    const res = await supplierService.get(id);
    if (res.success && res.data) {
      const s = res.data;
      setRow(s);
      setName(s.name);
      setContactEmail(s.contactEmail ?? '');
      setContactPhone(s.contactPhone ?? '');
      setTaxId(s.taxId ?? '');
      setNotes(s.notes ?? '');
      setIsActive(s.isActive);
    } else {
      setError(res.error?.message ?? 'Fornecedor não encontrado.');
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
    const res = await supplierService.update(id, {
      name: name.trim(),
      contactEmail: contactEmail.trim(),
      contactPhone: contactPhone.trim(),
      taxId: taxId.trim(),
      notes: notes.trim(),
      isActive,
    });
    setSaving(false);
    if (res.success && res.data) {
      setRow(res.data);
      return;
    }
    setError(res.error?.message ?? 'Erro ao guardar.');
  }

  async function onDelete() {
    if (!id || !window.confirm('Eliminar este fornecedor?')) return;
    const res = await supplierService.remove(id);
    if (res.success) {
      router.replace('/accounting/suppliers');
      return;
    }
    setError(res.error?.message ?? 'Erro ao eliminar.');
  }

  return (
    <AppShell
      user={user}
      subtitle={row?.name ?? 'Fornecedor'}
      activeNav="accounting"
      carsNavLabel="Viaturas"
      onLogout={logout}
      logoVariant="branded"
    >
      <p className={styles.intro}>
        <Link href="/accounting/suppliers" className={styles.mutedLink}>
          ← Fornecedores
        </Link>
      </p>
      {error ? <div className={styles.error}>{error}</div> : null}
      {row ? (
        <form className={styles.form} onSubmit={onSave}>
          <div className={styles.field}>
            <label htmlFor="name">Nome</label>
            <input id="name" value={name} onChange={(ev) => setName(ev.target.value)} required />
          </div>
          <div className={styles.field}>
            <label htmlFor="email">Email de contacto</label>
            <input id="email" type="email" value={contactEmail} onChange={(ev) => setContactEmail(ev.target.value)} />
          </div>
          <div className={styles.field}>
            <label htmlFor="phone">Telefone</label>
            <input id="phone" value={contactPhone} onChange={(ev) => setContactPhone(ev.target.value)} />
          </div>
          <div className={styles.field}>
            <label htmlFor="tax">NIF / Tax ID</label>
            <input id="tax" value={taxId} onChange={(ev) => setTaxId(ev.target.value)} />
          </div>
          <div className={styles.field}>
            <label htmlFor="notes">Notas</label>
            <textarea id="notes" value={notes} onChange={(ev) => setNotes(ev.target.value)} />
          </div>
          <div className={styles.field}>
            <label>
              <input type="checkbox" checked={isActive} onChange={(ev) => setIsActive(ev.target.checked)} /> Ativo
            </label>
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
