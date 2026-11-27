'use client';

import React from 'react';
import { BrandLogo } from '@/components/brand/BrandLogo';
import ThemeSwitcher from '@/components/theme/ThemeSwitcher';
import styles from './AuthShell.module.css';

export type AuthShellBanner = {
  variant: 'success' | 'error';
  message: string;
};

export type AuthShellProps = {
  title: string;
  subtitle?: string;
  banner?: AuthShellBanner | null;
  children: React.ReactNode;
};

export function AuthShell({ title, subtitle, banner, children }: Readonly<AuthShellProps>) {
  return (
    <div className={styles.pageOuter}>
      <div className={styles.column}>
        <div className={styles.themeRow}>
          <ThemeSwitcher />
        </div>
        <div className={styles.card}>
          <header className={styles.header}>
            <div className={styles.logoWrap}>
              <BrandLogo
                alt="Logótipo GonsGarage"
                width={48}
                height={48}
                className={styles.logoImage}
                priority
                dataTestId="brand-logo"
              />
            </div>
            <h1 className={styles.title}>{title}</h1>
            {subtitle ? <p className={styles.subtitle}>{subtitle}</p> : null}
          </header>

          {banner ? (
            <div
              className={banner.variant === 'success' ? styles.bannerSuccess : styles.bannerError}
              role={banner.variant === 'error' ? 'alert' : 'status'}
            >
              {banner.message}
            </div>
          ) : null}

          <div className={styles.body}>{children}</div>
        </div>
      </div>
    </div>
  );
}

export function AuthShellFooter({ children }: Readonly<{ children: React.ReactNode }>) {
  return <div className={styles.footer}>{children}</div>;
}
