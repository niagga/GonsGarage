import React from 'react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import FiscalIntegrationAdminPage from './page';
import { UserRole } from '@/types';

const mockReplace = vi.fn();
let searchParamsString = '';

const { statusMock, verifyMock, revokeMock, readinessMock } = vi.hoisted(() => ({
  statusMock: vi.fn(),
  verifyMock: vi.fn(),
  revokeMock: vi.fn(),
  readinessMock: vi.fn(),
}));

vi.mock('next/navigation', () => ({
  useRouter: () => ({
    replace: mockReplace,
    push: vi.fn(),
  }),
  useSearchParams: () => new URLSearchParams(searchParamsString),
}));

vi.mock('@/stores', () => ({
  useAuth: () => ({
    user: {
      id: '11111111-1111-1111-1111-111111111111',
      email: 'mgr@test.com',
      firstName: 'Maria',
      lastName: 'Gestora',
      role: UserRole.MANAGER,
      createdAt: '2020-01-01T00:00:00.000Z',
      updatedAt: '2020-01-01T00:00:00.000Z',
    },
    logout: vi.fn(),
  }),
}));

vi.mock('@/components/layout/AppShell', () => ({
  default: ({ children }: { children: React.ReactNode }) => (
    <div data-testid="shell">{children}</div>
  ),
}));

vi.mock('@/lib/services/fiscal-integration.service', () => ({
  fiscalIntegrationService: {
    getConnectionStatus: (...args: unknown[]) => statusMock(...args),
    verifyConnection: (...args: unknown[]) => verifyMock(...args),
    revokeConnection: (...args: unknown[]) => revokeMock(...args),
    getReadiness: (...args: unknown[]) => readinessMock(...args),
  },
}));

describe('FiscalIntegrationAdminPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    searchParamsString = '';
    statusMock.mockResolvedValue({
      success: true,
      data: {
        id: 'conn-1',
        scopeKey: 'default',
        providerKey: 'mock',
        state: 'connected',
        hasCredentials: true,
        version: 1,
      },
    });
    readinessMock.mockResolvedValue({
      success: true,
      data: {
        ready: false,
        gates: [
          {
            name: 'oauth_security',
            status: 'pending',
            guidance: 'Complete a configuração OAuth no servidor antes de ativar.',
          },
          {
            name: 'pdf_retrieval',
            status: 'pending',
            guidance: 'Confirme o arquivo privado de PDF antes da emissão legal.',
          },
        ],
      },
    });
    verifyMock.mockResolvedValue({
      success: true,
      data: {
        scopeKey: 'default',
        providerKey: 'mock',
        state: 'connected',
        hasCredentials: true,
        version: 2,
      },
    });
    revokeMock.mockResolvedValue({
      success: true,
      data: {
        scopeKey: 'default',
        providerKey: 'mock',
        state: 'revoked',
        hasCredentials: false,
        version: 3,
      },
    });
  });

  it('shows connection state and safe readiness guidance without secrets', async () => {
    render(<FiscalIntegrationAdminPage />);

    expect(await screen.findByText(/ligado|connected|Ligado/i)).toBeInTheDocument();
    expect(screen.getByText(/Complete a configuração OAuth/i)).toBeInTheDocument();
    expect(screen.queryByText(/access_token|refresh_token|credentialBase64|ciphertext/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/e-Fatura|Autoridade Tributária|AT\/e-Fatura/i)).not.toBeInTheDocument();
  });

  it('allows verify and disconnect actions for managers', async () => {
    const user = userEvent.setup();
    render(<FiscalIntegrationAdminPage />);

    await screen.findByText(/ligado|Ligado/i);
    await user.click(screen.getByRole('button', { name: /Verificar/i }));
    await waitFor(() => expect(verifyMock).toHaveBeenCalled());

    await user.click(screen.getByRole('button', { name: /Desligar|Desconectar/i }));
    await waitFor(() => expect(revokeMock).toHaveBeenCalled());
  });

  it('strips OAuth token/query leakage on return and shows safe status', async () => {
    searchParamsString = 'oauth=success&code=AUTH_CODE_LEAK&state=STATE_LEAK&access_token=TOK';
    render(<FiscalIntegrationAdminPage />);

    await waitFor(() => {
      expect(mockReplace).toHaveBeenCalledWith('/admin/integrations/fiscal', expect.anything());
    });

    expect(await screen.findByTestId('oauth-return-status')).toHaveTextContent(
      /autorização concluída|códigos sensíveis foram removidos/i,
    );
    expect(screen.queryByText(/AUTH_CODE_LEAK|STATE_LEAK|TOK/)).not.toBeInTheDocument();
  });

  it('shows AT/e-Fatura only when readiness exposes an evidenced field', async () => {
    readinessMock.mockResolvedValue({
      success: true,
      data: {
        ready: false,
        gates: [{ name: 'oauth_security', status: 'pending', guidance: 'Complete OAuth.' }],
        atCommunicationStatus: 'evidenced_local_only',
      },
    });
    render(<FiscalIntegrationAdminPage />);

    expect(await screen.findByText(/Comunicação fiscal evidenciada/i)).toBeInTheDocument();
    expect(screen.getByText(/evidenced_local_only/)).toBeInTheDocument();
  });

  it('does not invent AT/e-Fatura claims when the evidenced field is absent', async () => {
    render(<FiscalIntegrationAdminPage />);
    await screen.findByText(/Ligado/i);
    expect(screen.queryByText(/Comunicação fiscal evidenciada|e-Fatura|Autoridade Tributária/i)).not.toBeInTheDocument();
  });
});
