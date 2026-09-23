'use client';

import React from 'react';
import { useRouter } from 'next/navigation';
import { BrandLogo } from '@/components/brand/BrandLogo';
import type { User } from '@/types';
import { canManageUsers, isClient, isWorkshopStaff } from '@/types/user';
import { Button } from '@/components/ui/button';
import { cn } from '@/lib/utils';
import styles from './AppShell.module.css';

export type AppShellNavId =
  | 'dashboard'
  | 'cars'
  | 'appointments'
  | 'admin_users'
  | 'admin_parts'
  | 'admin_fiscal'
  | 'accounting'
  | 'my_invoices'
  | 'workshop';

export interface AppShellProps {
  user: User;
  subtitle: string;
  activeNav: AppShellNavId;
  carsNavLabel?: string;
  onLogout: () => void;
  children: React.ReactNode;
  /** Use workshop logo on home link (same as cars page). */
  logoVariant?: 'svg' | 'branded';
  /** Optional row below nav (e.g. page title + primary action). */
  toolbar?: React.ReactNode;
}

export default function AppShell({
  user,
  subtitle,
  activeNav,
  carsNavLabel = 'Os meus automóveis',
  onLogout,
  children,
  logoVariant = 'svg',
  toolbar,
}: Readonly<AppShellProps>) {
  const router = useRouter();

  return (
    <div className={styles.container}>
      <header className={styles.header}>
        <div className={styles.headerContent}>
          <Button
            type="button"
            variant="ghost"
            className={cn(styles.logoSection, 'h-auto gap-0 px-2 py-2')}
            onClick={() => router.push('/')}
            aria-label="Ir para a página inicial"
          >
            <div className={styles.logoIcon}>
              {logoVariant === 'branded' ? (
                <BrandLogo
                  alt=""
                  width={32}
                  height={32}
                  style={{ objectFit: 'cover', width: '100%', height: '100%' }}
                />
              ) : (
                <svg fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden>
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M19 21V5a2 2 0 00-2-2H7a2 2 0 00-2 2v16m14 0h2m-2 0h-2m-2 0H7m5 0v-5a2 2 0 012-2h2a2 2 0 012 2v5"
                  />
                </svg>
              )}
            </div>
            <div className={styles.logoText}>
              <h1>GonsGarage</h1>
              <p>{subtitle}</p>
            </div>
          </Button>
          <div className={styles.userSection}>
            <span>
              Olá, {user.firstName} {user.lastName}
            </span>
            <Button type="button" variant="outline" size="sm" onClick={onLogout} className={styles.logoutButton}>
              Terminar sessão
            </Button>
          </div>
        </div>
      </header>

      <nav className={styles.navigation} aria-label="Principal">
        <Button
          type="button"
          variant="ghost"
          onClick={() => router.push('/dashboard')}
          className={cn(styles.navButton, activeNav === 'dashboard' && styles.active)}
        >
          Painel
        </Button>
        <Button
          type="button"
          variant="ghost"
          onClick={() => router.push('/cars')}
          className={cn(styles.navButton, activeNav === 'cars' && styles.active)}
        >
          {carsNavLabel}
        </Button>
        <Button
          type="button"
          variant="ghost"
          onClick={() => router.push('/appointments')}
          className={cn(styles.navButton, activeNav === 'appointments' && styles.active)}
        >
          Marcações
        </Button>
        {isWorkshopStaff(user) ? (
          <Button
            type="button"
            variant="ghost"
            onClick={() => router.push('/workshop')}
            className={cn(styles.navButton, activeNav === 'workshop' && styles.active)}
          >
            Taller
          </Button>
        ) : null}
        {canManageUsers(user) ? (
          <>
            <Button
              type="button"
              variant="ghost"
              onClick={() => router.push('/admin/users')}
              className={cn(styles.navButton, activeNav === 'admin_users' && styles.active)}
            >
              Utilizadores
            </Button>
            <Button
              type="button"
              variant="ghost"
              onClick={() => router.push('/admin/parts')}
              className={cn(styles.navButton, activeNav === 'admin_parts' && styles.active)}
            >
              Peças (stock)
            </Button>
            <Button
              type="button"
              variant="ghost"
              onClick={() => router.push('/admin/integrations/fiscal')}
              className={cn(styles.navButton, activeNav === 'admin_fiscal' && styles.active)}
            >
              Integração fiscal
            </Button>
          </>
        ) : null}
        {isClient(user) ? (
          <Button
            type="button"
            variant="ghost"
            onClick={() => router.push('/my-invoices')}
            className={cn(styles.navButton, activeNav === 'my_invoices' && styles.active)}
          >
            As minhas faturas
          </Button>
        ) : canManageUsers(user) ? (
          <Button
            type="button"
            variant="ghost"
            onClick={() => router.push('/accounting')}
            className={cn(styles.navButton, activeNav === 'accounting' && styles.active)}
          >
            Contabilidade
          </Button>
        ) : null}
      </nav>

      <main className={styles.main}>
        {toolbar ? <div className={styles.pageToolbar}>{toolbar}</div> : null}
        {children}
      </main>
    </div>
  );
}
