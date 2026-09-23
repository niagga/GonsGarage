import React from 'react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import AdminUsersPage from './page';
import { UserRole } from '@/types';

const provisionUserMock = vi.fn();
const listUsersMock = vi.fn();

vi.mock('next/navigation', () => ({
  useRouter: () => ({
    push: vi.fn(),
    replace: vi.fn(),
  }),
}));

vi.mock('@/lib/api-client', () => ({
  apiClient: {
    provisionUser: (...args: unknown[]) => provisionUserMock(...args),
    listUsers: (...args: unknown[]) => listUsersMock(...args),
  },
}));

const adminUser = {
  id: 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa',
  email: 'admin@test.com',
  firstName: 'Ana',
  lastName: 'Admin',
  role: UserRole.ADMIN,
  createdAt: '2020-01-01T00:00:00.000Z',
  updatedAt: '2020-01-01T00:00:00.000Z',
};

vi.mock('@/stores', () => ({
  useAuth: () => ({
    user: adminUser,
    logout: vi.fn(),
  }),
}));

describe('AdminUsersPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    listUsersMock.mockResolvedValue({
      success: true,
      data: { items: [], total: 0 },
    });
    provisionUserMock.mockResolvedValue({
      success: true,
      data: {
        user: {
          id: 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb',
          email: 'new.user@example.com',
          firstName: 'N',
          lastName: 'U',
          role: 'client',
          createdAt: '2020-01-02T00:00:00.000Z',
          updatedAt: '2020-01-02T00:00:00.000Z',
        },
      },
    });
  });

  it('shows toolbar CTA and no dialog until opened', () => {
    render(<AdminUsersPage />);
    const main = screen.getByRole('main');
    expect(within(main).getByRole('heading', { name: 'Utilizadores', level: 1 })).toBeInTheDocument();
    expect(within(main).getByRole('button', { name: 'Novo utilizador' })).toBeInTheDocument();
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
  });

  it('opens create dialog from toolbar', async () => {
    const user = userEvent.setup();
    render(<AdminUsersPage />);
    await user.click(screen.getByRole('button', { name: 'Novo utilizador' }));
    expect(await screen.findByRole('dialog')).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'Novo utilizador' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Cancelar' })).toBeInTheDocument();
  });

  it('submits provision from modal and shows success on page', async () => {
    const user = userEvent.setup();
    render(<AdminUsersPage />);
    await user.click(screen.getByRole('button', { name: 'Novo utilizador' }));
    await screen.findByRole('dialog');

    await user.type(screen.getByLabelText(/E-mail/i), 'new.user@example.com');
    await user.type(screen.getByLabelText(/Palavra-passe inicial/i), 'secret12');
    await user.type(screen.getByLabelText(/^Nome$/i), 'Novo');
    await user.type(screen.getByLabelText(/Apelido/i), 'Utilizador');
    await user.click(screen.getByRole('button', { name: 'Criar utilizador' }));

    await waitFor(() => {
      expect(provisionUserMock).toHaveBeenCalledWith(
        expect.objectContaining({
          email: 'new.user@example.com',
          firstName: 'Novo',
          lastName: 'Utilizador',
          role: 'client',
        })
      );
    });

    await waitFor(() => {
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    });
    expect(screen.getByText(/Utilizador criado: new\.user@example\.com \(client\)/)).toBeInTheDocument();
  });

  it('does not call provision when canceling the dialog', async () => {
    const user = userEvent.setup();
    render(<AdminUsersPage />);
    await user.click(screen.getByRole('button', { name: 'Novo utilizador' }));
    await screen.findByRole('dialog');
    await user.click(screen.getByRole('button', { name: 'Cancelar' }));
    await waitFor(() => {
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    });
    expect(provisionUserMock).not.toHaveBeenCalled();
  });

  it('shows API error in modal and keeps dialog open on failure', async () => {
    provisionUserMock.mockResolvedValueOnce({
      success: false,
      error: { message: 'Combinação inválida', status: 403 },
    });
    const user = userEvent.setup();
    render(<AdminUsersPage />);
    await user.click(screen.getByRole('button', { name: 'Novo utilizador' }));
    await screen.findByRole('dialog');

    await user.type(screen.getByLabelText(/E-mail/i), 'x@y.com');
    await user.type(screen.getByLabelText(/Palavra-passe inicial/i), 'secret12');
    await user.type(screen.getByLabelText(/^Nome$/i), 'A');
    await user.type(screen.getByLabelText(/Apelido/i), 'B');
    await user.click(screen.getByRole('button', { name: 'Criar utilizador' }));

    expect(await screen.findByText('Combinação inválida')).toBeInTheDocument();
    expect(screen.getByRole('dialog')).toBeInTheDocument();
  });
});
