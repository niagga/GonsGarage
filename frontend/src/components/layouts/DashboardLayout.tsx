// src/components/layouts/DashboardLayout.tsx
'use client';

import React from 'react';
import { useRouter } from 'next/navigation';
import { useAuth } from '@/stores';
import { BrandLogo } from '@/components/brand/BrandLogo';
import { Button } from '@/components/ui/button';
import { cn } from '@/lib/utils';
import styles from './DashboardLayout.module.css';

interface NavigationItem {
  key: string;
  label: string;
  href: string;
}

interface DashboardLayoutProps {
  children: React.ReactNode;
  title: string;
  subtitle: string;
  activeTab: string;
  navigationItems: NavigationItem[];
  onNavClick?: (tabKey: string) => void;  // ✅ Callback opcional
}

export default function DashboardLayout({
  children,
  title,
  subtitle,
  activeTab,
  navigationItems,
  onNavClick
}: DashboardLayoutProps) {
  const { user, logout } = useAuth();
  const router = useRouter();

  const handleNavigation = (item: NavigationItem) => {
    if (onNavClick) {
      // ✅ Se tem callback, usa (para SPA)
      onNavClick(item.key);
    } else {
      // ✅ Se não tem, usa router (para páginas separadas)
      router.push(item.href);
    }
  };

  return (
    <div className={styles.container}>
      {/* Header */}
      <header className={styles.header}>
        <div className={styles.headerContent}>
          <div 
            className={styles.logoSection} 
            onClick={() => router.push('/')} 
            style={{ cursor: 'pointer' }}
          >
            <div className={styles.logoIcon}>
              <BrandLogo alt="GonsGarage Logo" width={24} height={24} style={{ objectFit: 'contain' }} />
            </div>
            <div>
              <h1>GonsGarage</h1>
              <p>{subtitle}</p>
            </div>
          </div>
          
          <div className={styles.userSection}>
            <span>Welcome, {user?.firstName} {user?.lastName}</span>
            <Button type="button" variant="outline" size="sm" onClick={logout} className={styles.logoutButton}>
              Logout
            </Button>
          </div>
        </div>
      </header>

      {/* Navigation */}
      <nav className={styles.navigation}>
        {navigationItems.map((item) => (
          <Button
            key={item.key}
            type="button"
            variant="ghost"
            onClick={() => handleNavigation(item)}
            className={cn(styles.navButton, activeTab === item.key && styles.active)}
          >
            {item.label}
          </Button>
        ))}
      </nav>

      {/* Content */}
      <main className={styles.main}>
        <div className={styles.pageHeader}>
          <h2>{title}</h2>
        </div>
        {children}
      </main>
    </div>
  );
}