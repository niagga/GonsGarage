'use client';

import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { useAuth } from '@/stores';
import { UserRole } from '@/types';
import { canManageUsers } from '@/types/user';
import { apiClient } from '@/lib/api-client';
import AppShell from '@/components/layout/AppShell';
import { Button } from '@/components/ui/button';
import styles from './admin-users.module.css';
import ProvisionUserModal, { type ProvisionRole, type RoleOption } from './ProvisionUserModal';

type ListedUser = {
  id: string;
  email: string;
  firstName: string;
  lastName: string;
  role: string;
  isActive: boolean;
  createdAt: string;
};

function roleLabel(role: string): string {
  switch (role) {
    case 'admin':
      return 'Admin';
    case 'manager':
      return 'Gestor';
    case 'employee':
      return 'Funcionário';
    case 'client':
      return 'Cliente';
    default:
      return role;
  }
}

export default function AdminUsersPage() {
  const { user, logout } = useAuth();
  const [createOpen, setCreateOpen] = useState(false);
  const [message, setMessage] = useState<string | null>(null);
  const [items, setItems] = useState<ListedUser[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [roleFilter, setRoleFilter] = useState('');
  const [query, setQuery] = useState('');
  const [loading, setLoading] = useState(false);

  const roleOptions = useMemo((): RoleOption[] => {
    if (!user || !canManageUsers(user)) return [];
    if (user.role === UserRole.ADMIN) {
      return [
        { value: 'manager', label: 'Gestor (manager)' },
        { value: 'employee', label: 'Funcionário (employee)' },
        { value: 'client', label: 'Cliente (client)' },
      ];
    }
    return [
      { value: 'employee', label: 'Funcionário (employee)' },
      { value: 'client', label: 'Cliente (client)' },
    ];
  }, [user]);

  const defaultRole: ProvisionRole = user?.role === UserRole.ADMIN ? 'client' : 'employee';

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    const res = await apiClient.listUsers({
      role: roleFilter || undefined,
      q: query.trim() || undefined,
      limit: 200,
    });
    setLoading(false);
    if (res?.success && res.data?.items) {
      setItems(res.data.items);
    } else {
      setItems([]);
      setError(res?.error?.message ?? 'Não foi possível carregar os utilizadores.');
    }
  }, [roleFilter, query]);

  useEffect(() => {
    let cancelled = false;
    const t = window.setTimeout(() => {
      if (!cancelled) void load();
    }, 200);
    return () => {
      cancelled = true;
      window.clearTimeout(t);
    };
  }, [load]);

  if (!user) return null;

  const toolbar = (
    <>
      <h1>Utilizadores</h1>
      <Button type="button" onClick={() => setCreateOpen(true)}>
        Novo utilizador
      </Button>
    </>
  );

  return (
    <AppShell
      user={user}
      subtitle="Utilizadores"
      activeNav="admin_users"
      carsNavLabel="Viaturas"
      onLogout={logout}
      logoVariant="branded"
      toolbar={toolbar}
    >
      <p className={styles.hint}>
        Lista de contas do sistema. Apenas administradores e gestores podem criar utilizadores.
      </p>
      {message ? <p className={styles.ok}>{message}</p> : null}
      {error ? <p className={styles.error}>{error}</p> : null}

      <div className={styles.filters}>
        <label className={styles.fieldInline}>
          <span>Papel</span>
          <select value={roleFilter} onChange={(e) => setRoleFilter(e.target.value)}>
            <option value="">Todos</option>
            <option value="client">Cliente</option>
            <option value="employee">Funcionário</option>
            <option value="manager">Gestor</option>
            <option value="admin">Admin</option>
          </select>
        </label>
        <label className={styles.fieldInline}>
          <span>Pesquisar</span>
          <input
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Nome ou email…"
          />
        </label>
      </div>

      <div className={styles.tableWrap}>
        <table className={styles.table}>
          <thead>
            <tr>
              <th>Nome</th>
              <th>Email</th>
              <th>Nível de acesso</th>
              <th>Ativo</th>
              <th>Criado</th>
            </tr>
          </thead>
          <tbody>
            {loading ? (
              <tr>
                <td colSpan={5}>A carregar…</td>
              </tr>
            ) : null}
            {!loading &&
              items.map((u) => (
                <tr key={u.id}>
                  <td>
                    {u.firstName} {u.lastName}
                  </td>
                  <td>{u.email}</td>
                  <td>{roleLabel(u.role)}</td>
                  <td>{u.isActive ? 'Sim' : 'Não'}</td>
                  <td>{u.createdAt?.slice(0, 10) ?? '—'}</td>
                </tr>
              ))}
            {!loading && items.length === 0 && !error ? (
              <tr>
                <td colSpan={5}>Sem utilizadores para estes filtros.</td>
              </tr>
            ) : null}
          </tbody>
        </table>
      </div>

      <ProvisionUserModal
        open={createOpen}
        onOpenChange={setCreateOpen}
        roleOptions={roleOptions}
        defaultRole={defaultRole}
        callerRole={user.role}
        onProvisioned={({ email, role }) => {
          setMessage(`Utilizador criado: ${email} (${role})`);
          void load();
        }}
      />
    </AppShell>
  );
}
