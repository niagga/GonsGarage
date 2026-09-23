import { apiClient, type ApiResponse } from '../api-client';
import type {
  FiscalConnectionStatus,
  FiscalReadiness,
} from '@/types/fiscal';

export type FiscalConnectionSetupBody = {
  credentialBase64: string;
  grantedScopes?: string[];
};

/**
 * Manager/admin fiscal connection + readiness client.
 * Keeps provider credentials server-side; responses must never carry tokens.
 */
export class FiscalIntegrationService {
  private static instance: FiscalIntegrationService;

  static getInstance(): FiscalIntegrationService {
    if (!FiscalIntegrationService.instance) {
      FiscalIntegrationService.instance = new FiscalIntegrationService();
    }
    return FiscalIntegrationService.instance;
  }

  private constructor() {}

  private connectionPath(scopeKey: string, providerKey: string): string {
    return `/fiscal/connections/${encodeURIComponent(scopeKey)}/${encodeURIComponent(providerKey)}`;
  }

  async getConnectionStatus(
    scopeKey: string,
    providerKey: string,
  ): Promise<ApiResponse<FiscalConnectionStatus>> {
    return apiClient.get<FiscalConnectionStatus>(this.connectionPath(scopeKey, providerKey));
  }

  async storeCredentials(
    scopeKey: string,
    providerKey: string,
    body: FiscalConnectionSetupBody,
  ): Promise<ApiResponse<FiscalConnectionStatus>> {
    return apiClient.put<FiscalConnectionStatus>(this.connectionPath(scopeKey, providerKey), body);
  }

  async verifyConnection(
    scopeKey: string,
    providerKey: string,
  ): Promise<ApiResponse<FiscalConnectionStatus>> {
    return apiClient.post<FiscalConnectionStatus>(
      `${this.connectionPath(scopeKey, providerKey)}/verify`,
      {},
    );
  }

  async revokeConnection(
    scopeKey: string,
    providerKey: string,
  ): Promise<ApiResponse<FiscalConnectionStatus>> {
    return apiClient.delete<FiscalConnectionStatus>(this.connectionPath(scopeKey, providerKey));
  }

  /**
   * Cloudware readiness gates (may be unavailable until enablement routes are composed).
   * Callers must treat missing readiness as fail-closed guidance, never as secrets.
   */
  async getReadiness(): Promise<ApiResponse<FiscalReadiness>> {
    return apiClient.get<FiscalReadiness>('/fiscal-integrations/cloudware/readiness');
  }
}

export const fiscalIntegrationService = FiscalIntegrationService.getInstance();
