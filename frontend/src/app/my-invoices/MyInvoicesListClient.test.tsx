import React from 'react';
import { describe, it, expect, vi, beforeEach, afterEach, type MockInstance } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import MyInvoicesListClient from './MyInvoicesListClient';
import { issuedInvoiceService } from '@/lib/services/issued-invoice.service';
import { fiscalizationService } from '@/lib/services/fiscalization.service';
import { UserRole } from '@/types';
import type { IssuedInvoice } from '@/types/accounting';

vi.mock('@/stores', () => ({
  useAuth: () => ({
    user: {
      id: 'u-1',
      email: 'client@test.com',
      firstName: 'C',
      lastName: 'L',
      role: UserRole.CLIENT,
      createdAt: '2026-01-01T00:00:00.000Z',
      updatedAt: '2026-01-01T00:00:00.000Z',
    },
    logout: vi.fn(),
  }),
}));

vi.mock('@/components/layout/AppShell', () => ({
  default: ({ children }: { children: React.ReactNode }) => <div data-testid="shell">{children}</div>,
}));

const sampleInvoices: IssuedInvoice[] = [
  {
    id: 'inv-1',
    customerId: 'cust-1',
    amount: 10,
    status: 'issued',
    notes: '',
    createdAt: '2026-01-02T00:00:00.000Z',
    updatedAt: '2026-01-02T00:00:00.000Z',
  },
  {
    id: 'inv-2',
    customerId: 'cust-1',
    amount: 20,
    status: 'issued',
    notes: '',
    createdAt: '2026-01-03T00:00:00.000Z',
    updatedAt: '2026-01-03T00:00:00.000Z',
  },
];

describe('MyInvoicesListClient', () => {
  let listMineSpy: MockInstance;
  let summariesSpy: MockInstance;

  beforeEach(() => {
    listMineSpy = vi.spyOn(issuedInvoiceService, 'listMine').mockResolvedValue({
      success: true,
      data: { items: [], total: 0 },
    });
    summariesSpy = vi.spyOn(fiscalizationService, 'listSummaries').mockResolvedValue({
      success: true,
      data: {
        items: [
          { invoiceId: 'inv-1', status: 'pending' },
          { invoiceId: 'inv-2', status: 'finalized' },
        ],
      },
    });
  });

  afterEach(() => {
    listMineSpy.mockRestore();
    summariesSpy.mockRestore();
  });

  it('does not call listMine on mount when server passed initial rows', async () => {
    render(<MyInvoicesListClient initialItems={sampleInvoices} />);
    await waitFor(() => {
      expect(listMineSpy).not.toHaveBeenCalled();
    });
  });

  it('calls listMine once when initial list is empty (client JWT path)', async () => {
    render(<MyInvoicesListClient initialItems={[]} />);
    await waitFor(() => {
      expect(listMineSpy).toHaveBeenCalledTimes(1);
    });
  });

  it('shows own-only simplified fiscal status badges for pending and finalized', async () => {
    render(<MyInvoicesListClient initialItems={sampleInvoices} />);

    await waitFor(() => {
      expect(summariesSpy).toHaveBeenCalledWith(['inv-1', 'inv-2']);
    });

    expect(await screen.findByText('Pendente')).toBeInTheDocument();
    expect(screen.getByText('Finalizado')).toBeInTheDocument();
  });

  it('does not expose privileged fiscal actions on the client list', async () => {
    render(<MyInvoicesListClient initialItems={sampleInvoices} />);
    await screen.findByText('Pendente');

    expect(screen.queryByRole('button', { name: /Finalizar/i })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /Reconciliar/i })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /Anular/i })).not.toBeInTheDocument();
    expect(screen.queryByText(/Cloudware|credential|token|e-Fatura|AT\b/i)).not.toBeInTheDocument();
  });
});

describe('MyInvoicesListClient — voided and unavailable badges', () => {
  beforeEach(() => {
    vi.spyOn(issuedInvoiceService, 'listMine').mockResolvedValue({
      success: true,
      data: { items: [], total: 0 },
    });
    vi.spyOn(fiscalizationService, 'listSummaries').mockResolvedValue({
      success: true,
      data: {
        items: [
          { invoiceId: 'inv-1', status: 'voided' },
          { invoiceId: 'inv-2', status: 'unavailable' },
        ],
      },
    });
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('shows voided and unavailable presentation labels', async () => {
    render(<MyInvoicesListClient initialItems={sampleInvoices} />);
    expect(await screen.findByText('Anulado')).toBeInTheDocument();
    expect(screen.getByText('Indisponível')).toBeInTheDocument();
  });
});
