'use client';

import React, { Suspense, useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { useAuth } from '@/stores';
import CarsContainer from '@/app/cars/components/CarsContainer';
import AppShell from '@/components/layout/AppShell';
import { AppLoading } from '@/components/ui/AppLoading';
import { useAuthHydrationReady } from '@/hooks/useAuthHydrationReady';
import styles from './cars.module.css';

export default function CarsPage() {
  const { user, logout } = useAuth();
  const router = useRouter();
  const authHydrated = useAuthHydrationReady();

  useEffect(() => {
    if (!authHydrated) return;
    if (!user) {
      router.replace('/auth/login');
    }
  }, [authHydrated, user, router]);

  if (!authHydrated || !user) {
    return (
      <div className="loadingScreen" aria-busy="true">
        <AppLoading size="lg" aria-busy={false} label="A sessão a carregar" />
      </div>
    );
  }

  return (
    <AppShell
      user={user}
      subtitle="Os meus automóveis"
      activeNav="cars"
      onLogout={logout}
      logoVariant="branded"
    >
      <Suspense
        fallback={
          <div className="loadingStack" aria-busy="true">
            <AppLoading size="md" aria-busy={false} />
            <span>A carregar…</span>
          </div>
        }
      >
        <CarsContainer
          headerTitle="Os meus automóveis"
          headerSubtitle="Gerir os seus automóveis registados"
          addButtonText="Adicionar automóvel"
          className={styles.carsSection}
        />
      </Suspense>
    </AppShell>
  );
}
