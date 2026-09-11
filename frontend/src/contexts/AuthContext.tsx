'use client';

import { type ReactNode, useEffect } from 'react';
import { useAuthStore } from '@/stores/auth.store';

/**
 * Mount-only bootstrap: session validation and `apiClient` token sync live in
 * `auth.store` (`checkAuthStatus`). No duplicate localStorage or fetch login here.
 */
export function AuthProvider({ children }: Readonly<{ children: ReactNode }>) {
  useEffect(() => {
    void useAuthStore.getState().checkAuthStatus();
  }, []);

  return <>{children}</>;
}
