// src/hooks/useAuthGuard.ts
'use client';

import { useAuth } from '@/stores';

export function useAuthGuard(allowedRoles?: string | string[]) {
  const { user, isAuthenticated, isLoading } = useAuth();
  
  // Apenas verifica se está autorizado, SEM fazer redirecionamentos
  // O AuthStore já cuida dos redirecionamentos
  const isAuthorized = (() => {
    if (isLoading || !isAuthenticated || !user) return false;
    
    if (!allowedRoles) return true;
    
    const roles = Array.isArray(allowedRoles) ? allowedRoles : [allowedRoles];
    return roles.includes(user.role);
  })();

  return { 
    user, 
    isLoading, 
    isAuthenticated,
    isAuthorized 
  };
}