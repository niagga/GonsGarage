import React from 'react';
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import MyInvoiceDetailClient from './MyInvoiceDetailClient';
import { issuedInvoiceService } from '@/lib/services/issued-invoice.service';
import { fiscalizationService } from '@/lib/services/fiscalization.service';
import { UserRole } from '@/types';
import type { IssuedInvoice } from '@/types/accounting';
import type { FiscalizationProjection } from '@/types/fiscal';

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

const baseRow: IssuedInvoice = {
  id: 'inv-1',
  customerId: 'c1',
  amount: 42,
  status: 'issued',
  notes: 'versión servidor',
  createdAt: '2026-01-02T00:00:00.000Z',
  updatedAt: '2026-01-02T00:00:00.000Z',
};

const finalizedProjection: FiscalizationProjection = {
  invoiceId: 'inv-1',
  documentId: 'doc-1',
  kind: 'FT',
  lifecycle: 'issued',
  status: 'finalized',
  version: 3,
  allowedActions: ['view'],
  artifactStatus: 'available',
  artifactId: 'art-1',
  artifactClassification: 'legal',
};

describe('MyInvoiceDetailClient — useOptimistic notes save', () => {
  beforeEach(() => {
    vi.spyOn(issuedInvoiceService, 'get').mockResolvedValue({
      success: true,
      data: baseRow,
    });
    vi.spyOn(fiscalizationService, 'getProjection').mockResolvedValue({
      success: true,
      data: { ...finalizedProjection, artifactStatus: 'unavailable', artifactId: undefined },
    });
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('shows optimistic server notes snapshot while saving, then reverts on API error', async () => {
    const user = userEvent.setup();
    const patchSpy = vi.spyOn(issuedInvoiceService, 'patchNotes').mockImplementation(
      () =>
        new Promise((resolve) => {
          setTimeout(
            () =>
              resolve({
                success: false,
                error: { message: 'fallo', status: 500 },
              }),
            40,
          );
        }),
    );

    render(<MyInvoiceDetailClient invoiceId="inv-1" initialRow={baseRow} />);

    const textarea = await screen.findByRole('textbox', { name: /notas/i });
    await user.clear(textarea);
    await user.type(textarea, 'borrador cliente');
    expect(textarea).toHaveValue('borrador cliente');

    await user.click(screen.getByRole('button', { name: /Guardar notas/i }));

    await waitFor(() => expect(patchSpy).toHaveBeenCalled());
    expect(patchSpy.mock.calls[0]?.[1]).toBe('borrador cliente');

    await waitFor(() => {
      expect(screen.getByTestId('invoice-notes-optimistic')).toHaveTextContent('borrador cliente');
    });

    await waitFor(() => {
      expect(screen.getByTestId('invoice-notes-optimistic')).toHaveTextContent('versión servidor');
    });

    expect(screen.getByText(/fallo/i)).toBeInTheDocument();
  });

  it('reconciles optimistic state with server row after successful patch', async () => {
    const user = userEvent.setup();
    const updated: IssuedInvoice = { ...baseRow, notes: 'gardado no API' };
    vi.spyOn(issuedInvoiceService, 'patchNotes').mockResolvedValue({
      success: true,
      data: updated,
    });

    render(<MyInvoiceDetailClient invoiceId="inv-1" initialRow={baseRow} />);

    const textarea = await screen.findByRole('textbox', { name: /notas/i });
    await user.clear(textarea);
    await user.type(textarea, 'texto novo');
    await user.click(screen.getByRole('button', { name: /Guardar notas/i }));

    await waitFor(() => {
      expect(screen.getByTestId('invoice-notes-optimistic')).toHaveTextContent('gardado no API');
    });
  });
});

describe('MyInvoiceDetailClient — own-only fiscal status and PDF', () => {
  beforeEach(() => {
    vi.spyOn(issuedInvoiceService, 'get').mockResolvedValue({
      success: true,
      data: baseRow,
    });
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('shows simplified finalized status and authorized PDF download', async () => {
    const user = userEvent.setup();
    vi.spyOn(fiscalizationService, 'getProjection').mockResolvedValue({
      success: true,
      data: finalizedProjection,
    });
    const downloadSpy = vi.spyOn(fiscalizationService, 'downloadArtifact').mockResolvedValue({
      success: true,
      data: new Blob(['%PDF'], { type: 'application/pdf' }),
    });
    const createObjectURL = vi.fn(() => 'blob:fiscal-pdf');
    const revokeObjectURL = vi.fn();
    vi.stubGlobal('URL', { ...URL, createObjectURL, revokeObjectURL });
    const clickSpy = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => undefined);

    render(<MyInvoiceDetailClient invoiceId="inv-1" initialRow={baseRow} />);

    expect(await screen.findByText('Finalizado')).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /Finalizar|Reconciliar|Anular|Repetir/i })).not.toBeInTheDocument();
    expect(screen.queryByText(/credential|access_token|Cloudware|e-Fatura/i)).not.toBeInTheDocument();

    const pdfBtn = await screen.findByRole('button', { name: /Descarregar PDF/i });
    await user.click(pdfBtn);
    await waitFor(() => {
      expect(downloadSpy).toHaveBeenCalledWith('inv-1', 'art-1');
    });
    expect(clickSpy).toHaveBeenCalled();
    clickSpy.mockRestore();  });

  it('keeps notes editable while fiscal PDF is unavailable', async () => {
    vi.spyOn(fiscalizationService, 'getProjection').mockResolvedValue({
      success: true,
      data: {
        ...finalizedProjection,
        artifactStatus: 'unavailable',
        artifactId: undefined,
      },
    });

    render(<MyInvoiceDetailClient invoiceId="inv-1" initialRow={baseRow} />);

    expect(await screen.findByText('Finalizado')).toBeInTheDocument();
    expect(screen.getByText(/PDF temporariamente indisponível|indisponível/i)).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /Descarregar PDF/i })).not.toBeInTheDocument();
    expect(screen.getByRole('textbox', { name: /notas/i })).toBeEnabled();
  });

  it('denies non-owner projection without enumerating fiscal existence', async () => {
    vi.spyOn(fiscalizationService, 'getProjection').mockResolvedValue({
      success: false,
      error: { message: 'not found', status: 404 },
    });

    render(<MyInvoiceDetailClient invoiceId="inv-other" initialRow={null} />);

    await waitFor(() => {
      expect(screen.getByText(/não encontrada|não foi possível/i)).toBeInTheDocument();
    });
    expect(screen.queryByText(/Finalizado|Pendente|Anulado/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/documento fiscal|documentId|art-/i)).not.toBeInTheDocument();
  });

  it('refuses compromised artifacts with provider-neutral guidance', async () => {
    vi.spyOn(fiscalizationService, 'getProjection').mockResolvedValue({
      success: true,
      data: {
        ...finalizedProjection,
        artifactStatus: 'compromised',
        artifactId: 'art-bad',
      },
    });

    render(<MyInvoiceDetailClient invoiceId="inv-1" initialRow={baseRow} />);

    expect(await screen.findByText(/comprometido/i)).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /Descarregar PDF/i })).not.toBeInTheDocument();
  });
});
