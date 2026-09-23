import React from 'react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import IssuedInvoicesStaffListPage from './page';
import { UserRole } from '@/types';

const mockReplace = vi.fn();
let searchParamsString = '';

const { listMock, createMock, summariesMock, listClientsMock, getCarsByOwnerMock, getRepairsMock } =
  vi.hoisted(() => ({
    listMock: vi.fn(),
    createMock: vi.fn(),
    summariesMock: vi.fn(),
    listClientsMock: vi.fn(),
    getCarsByOwnerMock: vi.fn(),
    getRepairsMock: vi.fn(),
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

vi.mock('@/lib/services/issued-invoice.service', () => ({
  issuedInvoiceService: {
    listStaff: (...args: unknown[]) => listMock(...args),
    createStaff: (...args: unknown[]) => createMock(...args),
  },
}));

vi.mock('@/lib/api-client', () => ({
  apiClient: {
    listClientUsers: (...args: unknown[]) => listClientsMock(...args),
  },
}));

vi.mock('@/lib/api', () => ({
  apiClient: {
    getRepairs: (...args: unknown[]) => getRepairsMock(...args),
  },
}));

vi.mock('@/lib/services/car.service', () => ({
  carService: {
    getCarsByOwner: (...args: unknown[]) => getCarsByOwnerMock(...args),
    getCar: vi.fn(),
  },
}));

vi.mock('@/lib/services/fiscalization.service', () => ({
  fiscalizationService: {
    listSummaries: (...args: unknown[]) => summariesMock(...args),
  },
}));

const emptyList = { success: true, data: { items: [], total: 0 } };

const issuedRow = {
  id: 'ii-new-1',
  customerId: '22222222-2222-2222-2222-222222222222',
  amount: 250,
  status: 'open',
  notes: '',
  createdAt: '2020-01-01T00:00:00.000Z',
  updatedAt: '2020-01-01T00:00:00.000Z',
};

describe('IssuedInvoicesStaffListPage create modal', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    searchParamsString = '';
    listMock.mockResolvedValue(emptyList);
    createMock.mockResolvedValue({ success: true, data: issuedRow });
    summariesMock.mockResolvedValue({ success: true, data: { items: [] } });
    listClientsMock.mockResolvedValue({
      success: true,
      data: {
        items: [
          {
            id: '22222222-2222-2222-2222-222222222222',
            email: 'cli@test.com',
            firstName: 'Cliente',
            lastName: 'Teste',
          },
        ],
        total: 1,
      },
    });
    getCarsByOwnerMock.mockResolvedValue({
      success: true,
      data: [
        {
          id: 'cccccccc-cccc-cccc-cccc-cccccccccccc',
          make: 'VW',
          model: 'Golf',
          year: 2020,
          licensePlate: 'AA-00-BB',
          color: 'preto',
          ownerId: '22222222-2222-2222-2222-222222222222',
          createdAt: '2020-01-01T00:00:00.000Z',
          updatedAt: '2020-01-01T00:00:00.000Z',
        },
      ],
    });
    getRepairsMock.mockResolvedValue({
      data: [
        {
          id: 'rrrrrrrr-rrrr-rrrr-rrrr-rrrrrrrrrrrr',
          car_id: 'cccccccc-cccc-cccc-cccc-cccccccccccc',
          technician_id: '11111111-1111-1111-1111-111111111111',
          description: 'Revisão',
          status: 'completed',
          cost: 250,
          created_at: '2020-01-01T00:00:00.000Z',
          updated_at: '2020-01-01T00:00:00.000Z',
        },
      ],
      error: null,
    });
  });

  it('opens the create dialog from the toolbar button', async () => {
    const user = userEvent.setup();
    render(<IssuedInvoicesStaffListPage />);

    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Nova fatura' })).toBeInTheDocument();
    });

    await user.click(screen.getByRole('button', { name: 'Nova fatura' }));

    expect(screen.getByRole('dialog')).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'Nova fatura emitida' })).toBeInTheDocument();
  });

  it('opens the create dialog when URL has create=1', async () => {
    searchParamsString = 'create=1';
    render(<IssuedInvoicesStaffListPage />);

    await waitFor(() => {
      expect(screen.getByRole('heading', { name: 'Nova fatura emitida' })).toBeInTheDocument();
    });

    expect(mockReplace).toHaveBeenCalledWith('/accounting/issued-invoices');
  });

  it('opens the create dialog from the empty-state CTA', async () => {
    const user = userEvent.setup();
    render(<IssuedInvoicesStaffListPage />);

    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Criar a primeira' })).toBeInTheDocument();
    });

    await user.click(screen.getByRole('button', { name: 'Criar a primeira' }));

    expect(screen.getByRole('dialog')).toBeInTheDocument();
  });

  it('closes the dialog on Cancel without calling createStaff', async () => {
    const user = userEvent.setup();
    render(<IssuedInvoicesStaffListPage />);

    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Nova fatura' })).toBeInTheDocument();
    });
    await user.click(screen.getByRole('button', { name: 'Nova fatura' }));
    await user.click(screen.getByRole('button', { name: 'Cancelar' }));

    await waitFor(() => {
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    });
    expect(createMock).not.toHaveBeenCalled();
  });

  it('closes the dialog on Escape without calling createStaff', async () => {
    const user = userEvent.setup();
    render(<IssuedInvoicesStaffListPage />);

    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Nova fatura' })).toBeInTheDocument();
    });
    await user.click(screen.getByRole('button', { name: 'Nova fatura' }));
    await user.keyboard('{Escape}');

    await waitFor(() => {
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    });
    expect(createMock).not.toHaveBeenCalled();
  });

  it('submits createStaff, closes the dialog, and reloads the list', async () => {
    const user = userEvent.setup();
    render(<IssuedInvoicesStaffListPage />);

    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Nova fatura' })).toBeInTheDocument();
    });
    expect(listMock).toHaveBeenCalledTimes(1);

    await user.click(screen.getByRole('button', { name: 'Nova fatura' }));

    await waitFor(() => {
      expect(screen.getByLabelText('Cliente')).toBeInTheDocument();
    });

    await waitFor(() => {
      expect(
        screen.getByRole('option', { name: /Cliente Teste/i }),
      ).toBeInTheDocument();
    });

    await user.selectOptions(
      screen.getByLabelText('Cliente'),
      '22222222-2222-2222-2222-222222222222',
    );

    await waitFor(() => {
      expect(screen.getByRole('option', { name: /VW Golf/i })).toBeInTheDocument();
    });
    await user.selectOptions(
      screen.getByLabelText('Viatura'),
      'cccccccc-cccc-cccc-cccc-cccccccccccc',
    );

    await waitFor(() => {
      expect(screen.getByRole('option', { name: /Revisão/i })).toBeInTheDocument();
    });
    await user.selectOptions(
      screen.getByLabelText('Reparação concluída'),
      'rrrrrrrr-rrrr-rrrr-rrrr-rrrrrrrrrrrr',
    );

    await waitFor(() => {
      expect(screen.getByLabelText('Valor')).toHaveValue('250');
    });

    await user.click(screen.getByRole('button', { name: 'Criar fatura interna' }));

    await waitFor(() => {
      expect(createMock).toHaveBeenCalledTimes(1);
    });
    expect(createMock).toHaveBeenCalledWith(
      expect.objectContaining({
        customerId: '22222222-2222-2222-2222-222222222222',
        repairId: 'rrrrrrrr-rrrr-rrrr-rrrr-rrrrrrrrrrrr',
        amount: 250,
      }),
    );
    await waitFor(() => {
      expect(listMock).toHaveBeenCalledTimes(2);
    });
    await waitFor(() => {
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    });
  });
});

describe('IssuedInvoicesStaffListPage fiscalization summaries', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    searchParamsString = '';
    createMock.mockResolvedValue({ success: true, data: issuedRow });
    listClientsMock.mockResolvedValue({ success: true, data: { items: [], total: 0 } });
    getCarsByOwnerMock.mockResolvedValue({ success: true, data: [] });
    getRepairsMock.mockResolvedValue({ data: [], error: null });
  });

  it('loads batch summaries and shows provider-neutral fiscal badges without changing columns', async () => {
    listMock.mockResolvedValue({
      success: true,
      data: { items: [issuedRow], total: 1 },
    });
    summariesMock.mockResolvedValue({
      success: true,
      data: {
        items: [
          {
            invoiceId: 'ii-new-1',
            status: 'draft',
            lifecycle: 'draft',
            allowedActions: ['view', 'edit'],
          },
        ],
      },
    });

    render(<IssuedInvoicesStaffListPage />);

    await waitFor(() => {
      expect(summariesMock).toHaveBeenCalledWith(['ii-new-1']);
    });
    await waitFor(() => {
      expect(screen.getByText(/rascunho/i)).toBeInTheDocument();
    });
    expect(screen.getByRole('columnheader', { name: 'Cliente (ID)' })).toBeInTheDocument();
    expect(screen.getByRole('columnheader', { name: 'Valor' })).toBeInTheDocument();
    expect(screen.getByRole('columnheader', { name: 'Estado' })).toBeInTheDocument();
    expect(screen.getByRole('columnheader', { name: /fiscalização/i })).toBeInTheDocument();
  });

  it('shows legacy_unfiscalized guidance for invoices without fiscal rows', async () => {
    listMock.mockResolvedValue({
      success: true,
      data: { items: [issuedRow], total: 1 },
    });
    summariesMock.mockResolvedValue({
      success: true,
      data: {
        items: [
          {
            invoiceId: 'ii-new-1',
            status: 'legacy_unfiscalized',
            allowedActions: [],
          },
        ],
      },
    });

    render(<IssuedInvoicesStaffListPage />);

    await waitFor(() => {
      expect(screen.getByText(/sem fiscalização|legado/i)).toBeInTheDocument();
    });
  });
});
