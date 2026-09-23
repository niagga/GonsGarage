import React from 'react';
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, waitFor, act } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { FiscalActions } from './FiscalActions';
import type { FiscalizationProjection } from '@/types/fiscal';
import { UserRole } from '@/types';

const finalizeMock = vi.fn();
const retryMock = vi.fn();
const reconcileMock = vi.fn();
const voidMock = vi.fn();
const getProjectionMock = vi.fn();
const downloadMock = vi.fn();

vi.mock('@/lib/services/fiscalization.service', () => ({
  fiscalizationService: {
    finalize: (...args: unknown[]) => finalizeMock(...args),
    retry: (...args: unknown[]) => retryMock(...args),
    reconcile: (...args: unknown[]) => reconcileMock(...args),
    voidDocument: (...args: unknown[]) => voidMock(...args),
    getProjection: (...args: unknown[]) => getProjectionMock(...args),
    downloadArtifact: (...args: unknown[]) => downloadMock(...args),
  },
}));

function projection(overrides: Partial<FiscalizationProjection> = {}): FiscalizationProjection {
  return {
    invoiceId: 'inv-1',
    documentId: 'doc-1',
    kind: 'FT',
    lifecycle: 'draft',
    status: 'draft',
    version: 5,
    currency: 'EUR',
    payableTotal: '12.34',
    allowedActions: ['view', 'edit', 'finalize'],
    ...overrides,
  };
}

describe('FiscalActions', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.useFakeTimers({ shouldAdvanceTime: true });
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('shows finalize for manager when allowedActions includes finalize', () => {
    render(
      <FiscalActions
        invoiceId="inv-1"
        projection={projection()}
        role={UserRole.MANAGER}
        onProjectionChange={vi.fn()}
      />,
    );
    expect(screen.getByRole('button', { name: /finalizar/i })).toBeInTheDocument();
  });

  it('hides privileged finalize/retry/void for employee even if somehow listed', () => {
    render(
      <FiscalActions
        invoiceId="inv-1"
        projection={projection({
          allowedActions: ['view', 'edit', 'finalize', 'retry', 'void'],
        })}
        role={UserRole.EMPLOYEE}
        onProjectionChange={vi.fn()}
      />,
    );
    expect(screen.queryByRole('button', { name: /finalizar/i })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /repetir/i })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /anular/i })).not.toBeInTheDocument();
  });

  it('treats 202 finalize as progress and starts polling until stable', async () => {
    const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime });
    const onChange = vi.fn();
    finalizeMock.mockResolvedValue({
      success: true,
      data: projection({ lifecycle: 'pending', status: 'pending', allowedActions: ['view'] }),
    });
    getProjectionMock
      .mockResolvedValueOnce({
        success: true,
        data: projection({ lifecycle: 'dispatching', status: 'pending', allowedActions: ['view'] }),
      })
      .mockResolvedValueOnce({
        success: true,
        data: projection({
          lifecycle: 'issued',
          status: 'finalized',
          allowedActions: ['view', 'void'],
          artifactStatus: 'available',
          artifactId: 'art-1',
        }),
      });

    render(
      <FiscalActions
        invoiceId="inv-1"
        projection={projection()}
        role={UserRole.ADMIN}
        onProjectionChange={onChange}
        pollIntervalMs={100}
        maxPollAttempts={5}
      />,
    );

    await user.click(screen.getByRole('button', { name: /finalizar/i }));
    await waitFor(() => expect(finalizeMock).toHaveBeenCalledTimes(1));
    expect(screen.getByText(/em progresso/i)).toBeInTheDocument();

    await act(async () => {
      await vi.advanceTimersByTimeAsync(120);
    });
    await waitFor(() => expect(getProjectionMock).toHaveBeenCalled());

    await act(async () => {
      await vi.advanceTimersByTimeAsync(120);
    });
    await waitFor(() =>
      expect(onChange).toHaveBeenCalledWith(
        expect.objectContaining({ status: 'finalized', lifecycle: 'issued' }),
      ),
    );
  });

  it('does not call finalize twice on double-click', async () => {
    const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime });
    let resolveFinalize: (v: unknown) => void = () => undefined;
    finalizeMock.mockImplementation(
      () =>
        new Promise((resolve) => {
          resolveFinalize = resolve;
        }),
    );

    render(
      <FiscalActions
        invoiceId="inv-1"
        projection={projection()}
        role={UserRole.MANAGER}
        onProjectionChange={vi.fn()}
      />,
    );

    const btn = screen.getByRole('button', { name: /finalizar/i });
    await user.click(btn);
    await user.click(btn);
    expect(finalizeMock).toHaveBeenCalledTimes(1);

    await act(async () => {
      resolveFinalize({
        success: true,
        data: projection({ lifecycle: 'pending', status: 'pending', allowedActions: ['view'] }),
      });
    });
  });

  it('stops capped polling and cancels on unmount', async () => {
    const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime });
    finalizeMock.mockResolvedValue({
      success: true,
      data: projection({ lifecycle: 'pending', status: 'pending', allowedActions: ['view'] }),
    });
    getProjectionMock.mockResolvedValue({
      success: true,
      data: projection({ lifecycle: 'pending', status: 'pending', allowedActions: ['view'] }),
    });

    const { unmount } = render(
      <FiscalActions
        invoiceId="inv-1"
        projection={projection()}
        role={UserRole.MANAGER}
        onProjectionChange={vi.fn()}
        pollIntervalMs={50}
        maxPollAttempts={3}
      />,
    );

    await user.click(screen.getByRole('button', { name: /finalizar/i }));
    await waitFor(() => expect(finalizeMock).toHaveBeenCalled());

    await act(async () => {
      await vi.advanceTimersByTimeAsync(60);
    });
    const callsBeforeUnmount = getProjectionMock.mock.calls.length;
    unmount();
    await act(async () => {
      await vi.advanceTimersByTimeAsync(500);
    });
    expect(getProjectionMock.mock.calls.length).toBe(callsBeforeUnmount);
  });

  it('shows unavailable/unknown guidance without privileged secrets', () => {
    render(
      <FiscalActions
        invoiceId="inv-1"
        projection={projection({
          lifecycle: 'outcome_unknown',
          status: 'unavailable',
          allowedActions: ['view', 'reconcile'],
          lastErrorMessage: 'Resultado ainda indeterminado',
        })}
        role={UserRole.MANAGER}
        onProjectionChange={vi.fn()}
      />,
    );
    expect(screen.getByRole('status')).toHaveTextContent(/indeterminado|indisponível/i);
    expect(screen.getByRole('button', { name: /reconciliar/i })).toBeInTheDocument();
    expect(screen.queryByText(/bearer|token|https:\/\//i)).not.toBeInTheDocument();
  });

  it('shows mock label outside production when classification is mock', () => {
    render(
      <FiscalActions
        invoiceId="inv-1"
        projection={projection({
          lifecycle: 'issued',
          status: 'finalized',
          allowedActions: ['view'],
          artifactStatus: 'available',
          artifactId: 'art-1',
          artifactClassification: 'mock',
        })}
        role={UserRole.MANAGER}
        onProjectionChange={vi.fn()}
        showMockLabels
      />,
    );
    expect(screen.getByText(/SEM VALIDADE FISCAL — MOCK/i)).toBeInTheDocument();
  });

  it('shows stale-version conflict message on 409', async () => {
    const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime });
    finalizeMock.mockResolvedValue({
      success: false,
      error: { message: 'version conflict', status: 409, code: 'conflict' },
    });
    render(
      <FiscalActions
        invoiceId="inv-1"
        projection={projection()}
        role={UserRole.MANAGER}
        onProjectionChange={vi.fn()}
      />,
    );
    await user.click(screen.getByRole('button', { name: /finalizar/i }));
    await waitFor(() => {
      expect(screen.getByText(/versão desatualizada|conflito/i)).toBeInTheDocument();
    });
  });
});
