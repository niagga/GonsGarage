import React from 'react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import IssuedInvoiceStaffDetailPage from './page';
import { UserRole } from '@/types';

const getMock = vi.fn();
const patchMock = vi.fn();
const removeMock = vi.fn();
const projectionMock = vi.fn();

vi.mock('next/navigation', () => ({
  useParams: () => ({ id: 'inv-detail-1' }),
  useRouter: () => ({ replace: vi.fn(), push: vi.fn() }),
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
  default: ({ children }: { children: React.ReactNode }) => <div data-testid="shell">{children}</div>,
}));

vi.mock('@/lib/services/issued-invoice.service', () => ({
  issuedInvoiceService: {
    get: (...args: unknown[]) => getMock(...args),
    patchIssuedInvoice: (...args: unknown[]) => patchMock(...args),
    removeStaff: (...args: unknown[]) => removeMock(...args),
  },
}));

vi.mock('@/lib/services/fiscalization.service', () => ({
  fiscalizationService: {
    getProjection: (...args: unknown[]) => projectionMock(...args),
    upsertDraft: vi.fn(),
    finalize: vi.fn(),
    retry: vi.fn(),
    reconcile: vi.fn(),
    voidDocument: vi.fn(),
    downloadArtifact: vi.fn(),
    deleteDraft: vi.fn(),
    listSummaries: vi.fn(),
  },
}));

const invoice = {
  id: 'inv-detail-1',
  customerId: '22222222-2222-2222-2222-222222222222',
  amount: 100,
  status: 'open',
  notes: 'nota operacional',
  createdAt: '2020-01-01T00:00:00.000Z',
  updatedAt: '2020-01-01T00:00:00.000Z',
};

describe('IssuedInvoiceStaffDetailPage fiscal panel', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    getMock.mockResolvedValue({ success: true, data: invoice });
    projectionMock.mockResolvedValue({
      success: true,
      data: {
        invoiceId: 'inv-detail-1',
        kind: 'FT',
        lifecycle: 'draft',
        status: 'draft',
        version: 1,
        currency: 'EUR',
        payableTotal: '100.00',
        allowedActions: ['view', 'edit', 'finalize'],
        readinessIssues: [],
      },
    });
  });

  it('keeps operational edit controls enabled alongside fiscal panel', async () => {
    render(<IssuedInvoiceStaffDetailPage />);
    await waitFor(() => {
      expect(screen.getByLabelText('Valor')).toBeInTheDocument();
    });
    expect(screen.getByLabelText('Valor')).not.toBeDisabled();
    expect(screen.getByLabelText('Estado')).not.toBeDisabled();
    expect(screen.getByLabelText('Notas')).not.toBeDisabled();
    expect(screen.getByRole('button', { name: 'Guardar' })).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: /fiscalização/i })).toBeInTheDocument();
  });

  it('loads fiscal projection for the invoice detail', async () => {
    render(<IssuedInvoiceStaffDetailPage />);
    await waitFor(() => {
      expect(projectionMock).toHaveBeenCalledWith('inv-detail-1');
    });
    await waitFor(() => {
      expect(screen.getByRole('heading', { name: /fiscalização/i })).toBeInTheDocument();
    });
    expect(screen.getByText(/estado fiscal:/i)).toHaveTextContent(/rascunho/i);
  });

  it('still patches operational fields without touching fiscalization finalize', async () => {
    const user = userEvent.setup();
    patchMock.mockResolvedValue({ success: true, data: { ...invoice, notes: 'atualizado' } });
    render(<IssuedInvoiceStaffDetailPage />);
    await waitFor(() => expect(screen.getByLabelText('Notas')).toBeInTheDocument());
    await user.clear(screen.getByLabelText('Notas'));
    await user.type(screen.getByLabelText('Notas'), 'atualizado');
    await user.click(screen.getByRole('button', { name: 'Guardar' }));
    await waitFor(() => expect(patchMock).toHaveBeenCalled());
    expect(patchMock.mock.calls[0][1]).toMatchObject({ notes: 'atualizado' });
  });
});
