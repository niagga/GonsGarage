import React from 'react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { FiscalDraftForm } from './FiscalDraftForm';
import type { FiscalizationProjection } from '@/types/fiscal';

const upsertMock = vi.fn();

vi.mock('@/lib/services/fiscalization.service', () => ({
  fiscalizationService: {
    upsertDraft: (...args: unknown[]) => upsertMock(...args),
  },
}));

const draftProjection: FiscalizationProjection = {
  invoiceId: 'inv-1',
  documentId: 'doc-1',
  kind: 'FT',
  lifecycle: 'draft',
  status: 'draft',
  version: 1,
  currency: 'EUR',
  payableTotal: '12.30',
  grossTotal: '10.00',
  taxTotal: '2.30',
  allowedActions: ['view', 'edit', 'delete'],
  readinessIssues: ['issuer_profile_missing'],
};

describe('FiscalDraftForm', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    upsertMock.mockResolvedValue({ success: true, data: draftProjection });
  });

  it('submits FT draft with decimal-string quantity and unit price', async () => {
    const user = userEvent.setup();
    const onSaved = vi.fn();
    render(
      <FiscalDraftForm invoiceId="inv-1" projection={draftProjection} onSaved={onSaved} />,
    );

    await user.clear(screen.getByLabelText(/quantidade/i));
    await user.type(screen.getByLabelText(/quantidade/i), '2.500');
    await user.clear(screen.getByLabelText(/preço unitário/i));
    await user.type(screen.getByLabelText(/preço unitário/i), '10.00');
    await user.click(screen.getByRole('button', { name: /guardar rascunho/i }));

    await waitFor(() => expect(upsertMock).toHaveBeenCalledTimes(1));
    const [, body] = upsertMock.mock.calls[0] as [string, { kind: string; lines: Array<{ quantity: string; unitPrice: string }> }];
    expect(body.kind).toBe('FT');
    expect(body.lines[0].quantity).toBe('2.500');
    expect(body.lines[0].unitPrice).toBe('10.00');
    expect(typeof body.lines[0].quantity).toBe('string');
  });

  it('allows switching document kind to FR', async () => {
    const user = userEvent.setup();
    render(<FiscalDraftForm invoiceId="inv-1" projection={draftProjection} onSaved={vi.fn()} />);
    await user.selectOptions(screen.getByLabelText(/tipo de documento/i), 'FR');
    await user.click(screen.getByRole('button', { name: /guardar rascunho/i }));
    await waitFor(() => expect(upsertMock).toHaveBeenCalled());
    expect(upsertMock.mock.calls[0][1].kind).toBe('FR');
  });

  it('shows server-calculated totals and readiness guidance from projection', () => {
    render(<FiscalDraftForm invoiceId="inv-1" projection={draftProjection} onSaved={vi.fn()} />);
    expect(screen.getByText(/12\.30/)).toBeInTheDocument();
    expect(screen.getByText(/totais calculados pelo servidor/i)).toBeInTheDocument();
    expect(screen.getByText(/issuer_profile_missing/i)).toBeInTheDocument();
  });

  it('shows source reference when present on a line', async () => {
    const user = userEvent.setup();
    render(<FiscalDraftForm invoiceId="inv-1" projection={draftProjection} onSaved={vi.fn()} />);
    await user.type(screen.getByLabelText(/referência de origem/i), 'repair:rep-99');
    await user.click(screen.getByRole('button', { name: /guardar rascunho/i }));
    await waitFor(() => expect(upsertMock).toHaveBeenCalled());
    const body = upsertMock.mock.calls[0][1] as {
      lines: Array<{ source?: { type: string; id: string } }>;
    };
    expect(body.lines[0].source).toEqual({ type: 'repair', id: 'rep-99' });
  });

  it('requires exemption code when tax treatment is exempt', async () => {
    const user = userEvent.setup();
    render(<FiscalDraftForm invoiceId="inv-1" projection={draftProjection} onSaved={vi.fn()} />);
    await user.selectOptions(screen.getByLabelText(/tratamento fiscal/i), 'exempt');
    await user.click(screen.getByRole('button', { name: /guardar rascunho/i }));
    expect(upsertMock).not.toHaveBeenCalled();
    expect(screen.getByRole('alert')).toHaveTextContent(/código de isenção/i);
  });
});
