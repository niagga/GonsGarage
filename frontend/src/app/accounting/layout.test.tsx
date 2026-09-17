import React from 'react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import AccountingLayout from './layout';
import { UserRole } from '@/types';
import type { User } from '@/types/user';

const { authState, replaceMock } = vi.hoisted(() => ({
  authState: { user: null as User | null },
  replaceMock: vi.fn(),
}));

vi.mock('next/navigation', () => ({
  useRouter: () => ({
    replace: replaceMock,
    push: vi.fn(),
  }),
}));

vi.mock('@/hooks/useAuthHydrationReady', () => ({
  useAuthHydrationReady: () => true,
}));

vi.mock('@/stores', () => ({
  useAuth: () => ({
    user: authState.user,
    logout: vi.fn(),
  }),
}));

function userWithRole(role: UserRole): User {
  return {
    id: '11111111-1111-1111-1111-111111111111',
    email: `${role}@test.com`,
    firstName: 'Test',
    lastName: 'User',
    role,
    createdAt: '2020-01-01T00:00:00.000Z',
    updatedAt: '2020-01-01T00:00:00.000Z',
  };
}

describe('AccountingLayout staff gate', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    authState.user = null;
  });

  it('renders children for workshop staff', () => {
    authState.user = userWithRole(UserRole.EMPLOYEE);
    render(
      <AccountingLayout>
        <div>staff-only</div>
      </AccountingLayout>,
    );
    expect(screen.getByText('staff-only')).toBeInTheDocument();
    expect(replaceMock).not.toHaveBeenCalled();
  });

  it('redirects a client to /dashboard and does not render children', async () => {
    authState.user = userWithRole(UserRole.CLIENT);
    render(
      <AccountingLayout>
        <div>staff-only</div>
      </AccountingLayout>,
    );
    expect(screen.queryByText('staff-only')).not.toBeInTheDocument();
    await waitFor(() => {
      expect(replaceMock).toHaveBeenCalledWith('/dashboard');
    });
  });

  it('redirects an unauthenticated visitor to /auth/login', async () => {
    authState.user = null;
    render(
      <AccountingLayout>
        <div>staff-only</div>
      </AccountingLayout>,
    );
    expect(screen.queryByText('staff-only')).not.toBeInTheDocument();
    await waitFor(() => {
      expect(replaceMock).toHaveBeenCalledWith('/auth/login');
    });
  });
});
