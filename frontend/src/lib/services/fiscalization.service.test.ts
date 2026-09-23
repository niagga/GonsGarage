import { beforeEach, describe, expect, it, vi } from 'vitest';

vi.mock('../api-client', () => ({
  apiClient: {
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    delete: vi.fn(),
    getToken: vi.fn(() => 'test-token'),
  },
  HTTP_STATUS: {
    OK: 200,
    CREATED: 201,
    NO_CONTENT: 204,
    CONFLICT: 409,
    UNPROCESSABLE_ENTITY: 422,
    SERVICE_UNAVAILABLE: 503,
  },
}));

import { apiClient } from '../api-client';
import { fiscalizationService } from './fiscalization.service';
import type { FiscalDraftRequest } from '@/types/fiscal';

const mocked = vi.mocked(apiClient);

const sampleProjection = {
  invoiceId: 'inv-1',
  documentId: 'doc-1',
  kind: 'FT',
  lifecycle: 'draft',
  status: 'draft',
  version: 2,
  currency: 'EUR',
  payableTotal: '12.34',
  grossTotal: '10.00',
  taxTotal: '2.34',
  allowedActions: ['view', 'edit', 'delete', 'finalize'],
  readinessIssues: [],
};

const draftBody: FiscalDraftRequest = {
  version: 2,
  kind: 'FT',
  policyKey: 'pt-sales',
  currency: 'EUR',
  customer: { legalName: 'Cliente Lda', taxIdentifier: '123456789', countryCode: 'PT' },
  billingAddress: { line1: 'Rua A', postalCode: '1000-001', city: 'Lisboa', countryCode: 'PT' },
  lines: [
    {
      position: 1,
      description: 'Serviço',
      quantity: '1.000',
      unitCode: 'C62',
      unitPrice: '10.00',
      discount: { kind: 'none', value: '0' },
      taxTreatmentCode: 'IVA23',
      taxRate: '0.23',
      source: { type: 'repair', id: 'rep-1' },
    },
  ],
  declaredTotals: { payableTotal: '12.34' },
};

describe('fiscalizationService', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('getProjection GETs /invoices/:id/fiscalization', async () => {
    mocked.get.mockResolvedValueOnce({ success: true, data: sampleProjection });
    const res = await fiscalizationService.getProjection('inv-1');
    expect(mocked.get).toHaveBeenCalledWith('/invoices/inv-1/fiscalization');
    expect(res.success).toBe(true);
    expect(res.data?.payableTotal).toBe('12.34');
    expect(typeof res.data?.payableTotal).toBe('string');
  });

  it('listSummaries GETs fiscalization-summaries with CSV invoiceIds', async () => {
    mocked.get.mockResolvedValueOnce({
      success: true,
      data: { items: [{ invoiceId: 'a', status: 'legacy_unfiscalized', allowedActions: [] }] },
    });
    const res = await fiscalizationService.listSummaries(['a', 'b']);
    expect(mocked.get).toHaveBeenCalledWith('/invoices/fiscalization-summaries?invoiceIds=a%2Cb');
    expect(res.data?.items).toHaveLength(1);
    expect(res.data?.items[0].status).toBe('legacy_unfiscalized');
  });

  it('upsertDraft PUTs draft with decimal strings (not numbers)', async () => {
    mocked.put.mockResolvedValueOnce({ success: true, data: sampleProjection });
    await fiscalizationService.upsertDraft('inv-1', draftBody);
    expect(mocked.put).toHaveBeenCalledWith('/invoices/inv-1/fiscalization/draft', draftBody);
    const body = mocked.put.mock.calls[0][1] as FiscalDraftRequest;
    expect(body.lines[0].quantity).toBe('1.000');
    expect(body.lines[0].unitPrice).toBe('10.00');
    expect(typeof body.lines[0].quantity).toBe('string');
    expect(typeof body.lines[0].unitPrice).toBe('string');
  });

  it('finalize POSTs expectedVersion and returns projection (202 accepted as success)', async () => {
    mocked.post.mockResolvedValueOnce({
      success: true,
      data: { ...sampleProjection, lifecycle: 'pending', status: 'pending', allowedActions: ['view'] },
    });
    const res = await fiscalizationService.finalize('inv-1', { expectedVersion: 2 });
    expect(mocked.post).toHaveBeenCalledWith('/invoices/inv-1/fiscalization/finalize', {
      expectedVersion: 2,
    });
    expect(res.data?.lifecycle).toBe('pending');
  });

  it('retry, reconcile, and void post expectedVersion to action paths', async () => {
    mocked.post.mockResolvedValue({ success: true, data: sampleProjection });
    await fiscalizationService.retry('inv-1', { expectedVersion: 3 });
    await fiscalizationService.reconcile('inv-1', { expectedVersion: 3 });
    await fiscalizationService.voidDocument('inv-1', { expectedVersion: 3 });
    expect(mocked.post).toHaveBeenCalledWith('/invoices/inv-1/fiscalization/retry', {
      expectedVersion: 3,
    });
    expect(mocked.post).toHaveBeenCalledWith('/invoices/inv-1/fiscalization/reconcile', {
      expectedVersion: 3,
    });
    expect(mocked.post).toHaveBeenCalledWith('/invoices/inv-1/fiscalization/void', {
      expectedVersion: 3,
    });
  });

  it('deleteDraft DELETEs with expectedVersion query', async () => {
    mocked.delete.mockResolvedValueOnce({ success: true, data: null });
    await fiscalizationService.deleteDraft('inv-1', 4);
    expect(mocked.delete).toHaveBeenCalledWith(
      '/invoices/inv-1/fiscalization/draft?expectedVersion=4',
    );
  });

  it('downloadArtifact fetches PDF blob for artifact path', async () => {
    const blob = new Blob(['%PDF-mock'], { type: 'application/pdf' });
    const fetchSpy = vi.spyOn(globalThis, 'fetch').mockResolvedValueOnce(
      new Response(blob, {
        status: 200,
        headers: { 'Content-Type': 'application/pdf' },
      }),
    );
    const result = await fiscalizationService.downloadArtifact('inv-1', 'art-1');
    expect(fetchSpy).toHaveBeenCalled();
    const url = String(fetchSpy.mock.calls[0][0]);
    expect(url).toContain('/invoices/inv-1/fiscal-artifacts/art-1');
    expect(result.success).toBe(true);
    expect(result.data?.type).toBe('application/pdf');
    fetchSpy.mockRestore();
  });
});
