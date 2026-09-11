/**
 * Client-only: auth session + `apiClient` token sync. Single source for login/logout/check;
 * `AuthProvider` only calls `checkAuthStatus` on mount.
 */
// AuthStore using Zustand following Agent.md patterns

import { create } from 'zustand';
import { immer } from 'zustand/middleware/immer';
import { persist } from 'zustand/middleware';
import { User, UserRole, RegisterRequest } from '@/types';
import { apiClient } from '@/lib/api-client';
import { getPublicApiOrigin } from '@/lib/api-public-origin';

// ✅ Store state interface following Agent.md
interface AuthState {
  // Authentication state
  user: User | null;
  token: string | null;
  isLoading: boolean;
  error: string | null;
  
  // Derived state
  isAuthenticated: boolean;
}

// ✅ Store actions interface following Agent.md  
interface AuthActions {
  // Authentication methods (maintain same API as AuthContext)
  login: (email: string, password: string) => Promise<{ success: boolean; error?: string }>;
  register: (data: RegisterRequest) => Promise<{ success: boolean; error?: string }>;
  logout: () => void;
  
  // State management
  setUser: (user: User | null) => void;
  setToken: (token: string | null) => void;
  setLoading: (loading: boolean) => void;
  setError: (error: string | null) => void;
  clearError: () => void;
  
  // Initialization
  checkAuthStatus: () => Promise<void>;
}

// ✅ Complete store type
type AuthStore = AuthState & AuthActions;

function mapMeUser(raw: Record<string, unknown>): User {
  const u = raw as Record<string, unknown>;
  const idVal = u.id;
  const id = typeof idVal === 'string' ? idVal : idVal != null ? String(idVal) : '';
  return {
    id,
    email: String(u.email ?? ''),
    firstName: String(u.firstName ?? u.first_name ?? ''),
    lastName: String(u.lastName ?? u.last_name ?? ''),
    role: (u.role as UserRole) || UserRole.CLIENT,
    createdAt: String(u.createdAt ?? u.created_at ?? new Date().toISOString()),
    updatedAt: String(u.updatedAt ?? u.updated_at ?? new Date().toISOString()),
  };
}

async function fetchCurrentUser(token: string): Promise<User> {
  const response = await fetch(`${getPublicApiOrigin()}/api/v1/auth/me`, {
    headers: {
      Authorization: `Bearer ${token}`,
      'Content-Type': 'application/json',
    },
  });
  if (!response.ok) {
    throw new Error(`auth/me failed (${response.status})`);
  }
  const body = await response.json();
  const userPayload = (body as { user?: Record<string, unknown> }).user;
  if (!userPayload) {
    throw new Error('auth/me: missing user');
  }
  return mapMeUser(userPayload);
}

// ✅ Helper function for localStorage operations
const storage = {
  setAuthData: (token: string, user: User) => {
    if (typeof window !== 'undefined') {
      localStorage.setItem('auth_token', token);
      localStorage.setItem('auth_user', JSON.stringify(user));
    }
  },
  
  getAuthData: (): { token: string | null; user: User | null } => {
    if (typeof window === 'undefined') {
      return { token: null, user: null };
    }
    
    try {
      const token = localStorage.getItem('auth_token');
      const userData = localStorage.getItem('auth_user');
      
      if (token) {
        let user: User | null = null;
        if (userData) {
          try {
            user = JSON.parse(userData) as User;
          } catch {
            user = null;
          }
        }
        return { token, user };
      }
    } catch (error) {
      console.error('Failed to parse stored auth data:', error);
    }
    
    return { token: null, user: null };
  },
  
  clearAuthData: () => {
    if (typeof window !== 'undefined') {
      localStorage.removeItem('auth_token');
      localStorage.removeItem('auth_user');
    }
  }
};

// ✅ API functions following Agent.md
const authAPI = {
  login: async (email: string, password: string) => {
    const response = await fetch(`${getPublicApiOrigin()}/api/v1/auth/login`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ email, password }),
    });

    if (!response.ok) {
      const errorText = await response.text();
      let errorMessage = `Falha no início de sessão (${response.status})`;
      
      try {
        const errorData = JSON.parse(errorText);
        errorMessage = errorData.error || errorMessage;
      } catch {
        // Keep the default errorMessage
      }
      
      throw new Error(errorMessage);
    }

    return await response.json();
  },

  register: async (data: RegisterRequest) => {
    const response = await fetch(`${getPublicApiOrigin()}/api/v1/auth/register`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        email: data.email,
        password: data.password,
        firstName: data.firstName, // ✅ camelCase per Agent.md
        lastName: data.lastName,   // ✅ camelCase per Agent.md
        role: data.role || UserRole.CLIENT,
      }),
    });

    if (!response.ok) {
      const errorText = await response.text();
      let errorMessage = `Falha no registo (${response.status})`;
      
      try {
        const errorData = JSON.parse(errorText);
        errorMessage = errorData.error || errorData.message || errorMessage;
      } catch {
        // Keep the default errorMessage
      }
      
      throw new Error(errorMessage);
    }

    return await response.json();
  }
};

// ✅ Zustand store with Immer and Persist middleware
export const useAuthStore = create<AuthStore>()(
  persist(
    immer((set) => ({
      // ✅ Initial state
      user: null,
      token: null,
      isLoading: false,
      error: null,
      isAuthenticated: false,

      // ✅ Authentication methods (maintain AuthContext API)
      login: async (email: string, password: string) => {
        try {
          set((state) => {
            state.isLoading = true;
            state.error = null;
          });

          const data = await authAPI.login(email, password);

          if (data.token) {
            try {
              const userData = await fetchCurrentUser(data.token);

              console.log('Setting user data:', userData);

              set((state) => {
                state.token = data.token;
                state.user = userData;
                state.isAuthenticated = true;
                state.isLoading = false;
              });

              storage.setAuthData(data.token, userData);
              apiClient.setToken(data.token);

              return { success: true };
            } catch (meErr) {
              const msg =
                meErr instanceof Error ? meErr.message : 'Não foi possível validar a sessão com o servidor.';
              set((state) => {
                state.error = msg;
                state.isLoading = false;
                state.isAuthenticated = false;
              });
              return { success: false, error: msg };
            }
          } else {
            set((state) => {
              state.error = data.error || 'Falha no início de sessão';
              state.isLoading = false;
            });
            return { success: false, error: data.error || 'Falha no início de sessão' };
          }
        } catch (error) {
          const errorMessage = error instanceof Error ? error.message : 'Erro de rede. Tente novamente.';
          set((state) => {
            state.error = errorMessage;
            state.isLoading = false;
          });
          return { success: false, error: errorMessage };
        }
      },

      register: async (data: RegisterRequest) => {
        try {
          set((state) => {
            state.isLoading = true;
            state.error = null;
          });

          await authAPI.register(data);

          set((state) => {
            state.isLoading = false;
          });

          return { success: true };
        } catch (error) {
          const errorMessage = error instanceof Error ? error.message : 'Erro de rede. Tente novamente.';
          set((state) => {
            state.error = errorMessage;
            state.isLoading = false;
          });
          return { success: false, error: errorMessage };
        }
      },

      logout: () => {
        set((state) => {
          state.user = null;
          state.token = null;
          state.isAuthenticated = false;
          state.error = null;
        });

        // Clear localStorage
        storage.clearAuthData();
        apiClient.clearToken();

        // Redirect to landing page
        if (typeof window !== 'undefined') {
          window.location.href = '/';
        }
      },

      // ✅ State management methods
      setUser: (user: User | null) => {
        set((state) => {
          state.user = user;
          state.isAuthenticated = !!user && !!state.token;
        });
      },

      setToken: (token: string | null) => {
        set((state) => {
          state.token = token;
          state.isAuthenticated = !!state.user && !!token;
        });
        if (typeof token === 'string' && token.length > 0) {
          apiClient.setToken(token);
        }
      },

      setLoading: (loading: boolean) => {
        set((state) => {
          state.isLoading = loading;
        });
      },

      setError: (error: string | null) => {
        set((state) => {
          state.error = error;
        });
      },

      clearError: () => {
        set((state) => {
          state.error = null;
        });
      },

      // ✅ Initialize auth status from localStorage
      checkAuthStatus: async () => {
        try {
          set((state) => {
            state.isLoading = true;
          });

          const { token } = storage.getAuthData();

          if (token) {
            try {
              const user = await fetchCurrentUser(token);
              set((state) => {
                state.token = token;
                state.user = user;
                state.isAuthenticated = true;
              });
              storage.setAuthData(token, user);
              apiClient.setToken(token);
            } catch {
              storage.clearAuthData();
              set((state) => {
                state.user = null;
                state.token = null;
                state.isAuthenticated = false;
              });
            }
          } else {
            set((state) => {
              state.isAuthenticated = false;
            });
          }
        } catch (error) {
          console.error('Auth check failed:', error);
          storage.clearAuthData();
          set((state) => {
            state.user = null;
            state.token = null;
            state.isAuthenticated = false;
            state.error = 'Authentication check failed';
          });
        } finally {
          set((state) => {
            state.isLoading = false;
          });
        }
      },
    })),
    {
      name: 'gons-garage-auth', // ✅ Persistent storage key
      partialize: (state) => ({
        user: state.user,
        token: state.token,
        // Don't persist loading states or errors
      }),
      merge: (persistedState, currentState) => {
        const next = {
          ...currentState,
          ...(persistedState as object),
        };
        return {
          ...next,
          // `isAuthenticated` is not partialed; derive it so /appointments does not treat a valid session as logged out
          isAuthenticated: Boolean(next.user && next.token),
        };
      },
    }
  )
);

// ✅ Convenience hooks following Agent.md patterns
export const useAuth = () => {
  const store = useAuthStore();
  return {
    // State
    user: store.user,
    token: store.token,
    isLoading: store.isLoading,
    error: store.error,
    isAuthenticated: store.isAuthenticated,
    
    // Actions (maintain AuthContext API)
    login: store.login,
    register: store.register,
    logout: store.logout,
    clearError: store.clearError,
  };
};

// ✅ Additional convenience hooks
export const useUser = () => useAuthStore((state) => state.user);
export const useAuthToken = () => useAuthStore((state) => state.token);
export const useAuthLoading = () => useAuthStore((state) => state.isLoading);
export const useAuthError = () => useAuthStore((state) => state.error);
export const useIsAuthenticated = () => useAuthStore((state) => state.isAuthenticated);