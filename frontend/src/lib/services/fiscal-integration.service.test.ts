import { beforeEach, describe, expect, it, vi } from 'vitest';

vi.mock('../api-client', () => ({
  apiClient: {
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    delete: vi.fn(),
  },
}));

import { apiClient } from '../api-client';
import { fiscalIntegrationService } from './fiscal-integration.service';

const mocked = vi.mocked(apiClient);

describe('fiscalIntegrationService', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('getConnectionStatus GETs /fiscal/connections/:scope/:provider', async () => {
    mocked.get.mockResolvedValueOnce({
      success: true,
      data: {
        scopeKey: 'default',
        providerKey: 'mock',
        state: 'connected',
        hasCredentials: true,
      },
    });
    const res = await fiscalIntegrationService.getConnectionStatus('default', 'mock');
    expect(mocked.get).toHaveBeenCalledWith('/fiscal/connections/default/mock');
    expect(res.data?.state).toBe('connected');
    expect(JSON.stringify(res.data)).not.toMatch(/token|ciphertext|nonce/i);
  });

  it('verifyConnection POSTs verify without leaking credentials in the path', async () => {
    mocked.post.mockResolvedValueOnce({
      success: true,
      data: { scopeKey: 'default', providerKey: 'mock', state: 'connected', hasCredentials: true },
    });
    await fiscalIntegrationService.verifyConnection('default', 'mock');
    expect(mocked.post).toHaveBeenCalledWith('/fiscal/connections/default/mock/verify', {});
  });

  it('revokeConnection DELETEs the connection resource', async () => {
    mocked.delete.mockResolvedValueOnce({
      success: true,
      data: { scopeKey: 'default', providerKey: 'mock', state: 'revoked', hasCredentials: false },
    });
    await fiscalIntegrationService.revokeConnection('default', 'mock');
    expect(mocked.delete).toHaveBeenCalledWith('/fiscal/connections/default/mock');
  });

  it('getReadiness GETs cloudware readiness without query tokens', async () => {
    mocked.get.mockResolvedValueOnce({
      success: true,
      data: { ready: false, gates: [] },
    });
    await fiscalIntegrationService.getReadiness();
    expect(mocked.get).toHaveBeenCalledWith('/fiscal-integrations/cloudware/readiness');
  });
});
