import React from 'react';
import { useAuthToken } from '@/stores';
import { carApi } from '@/lib/api/carApi';

// ✅ Hook to integrate API client with authentication store (migrated from AuthContext)
export function useApiClient() {
  const token = useAuthToken();

  // ✅ Sync token with API client when auth token changes
  React.useEffect(() => {
    if (token) {
      carApi.setAuthToken(token);
    } else {
      carApi.clearAuthToken();
    }
  }, [token]);

  return carApi;
}