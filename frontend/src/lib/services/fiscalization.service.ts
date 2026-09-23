import { apiClient, type ApiResponse } from '../api-client';
import { getPublicApiOrigin } from '../api-public-origin';
import type {
  FiscalActionRequest,
  FiscalDraftRequest,
  FiscalizationProjection,
  FiscalizationSummariesResponse,
} from '@/types/fiscal';

export class FiscalizationService {
  private static instance: FiscalizationService;

  static getInstance(): FiscalizationService {
    if (!FiscalizationService.instance) {
      FiscalizationService.instance = new FiscalizationService();
    }
    return FiscalizationService.instance;
  }

  private constructor() {}

  async getProjection(invoiceId: string): Promise<ApiResponse<FiscalizationProjection>> {
    return apiClient.get<FiscalizationProjection>(`/invoices/${invoiceId}/fiscalization`);
  }

  async listSummaries(invoiceIds: string[]): Promise<ApiResponse<FiscalizationSummariesResponse>> {
    const csv = invoiceIds.map(encodeURIComponent).join('%2C');
    return apiClient.get<FiscalizationSummariesResponse>(
      `/invoices/fiscalization-summaries?invoiceIds=${csv}`,
    );
  }

  async upsertDraft(
    invoiceId: string,
    body: FiscalDraftRequest,
  ): Promise<ApiResponse<FiscalizationProjection>> {
    return apiClient.put<FiscalizationProjection>(
      `/invoices/${invoiceId}/fiscalization/draft`,
      body,
    );
  }

  async deleteDraft(invoiceId: string, expectedVersion: number): Promise<ApiResponse<unknown>> {
    return apiClient.delete(
      `/invoices/${invoiceId}/fiscalization/draft?expectedVersion=${expectedVersion}`,
    );
  }

  async finalize(
    invoiceId: string,
    body: FiscalActionRequest,
  ): Promise<ApiResponse<FiscalizationProjection>> {
    return apiClient.post<FiscalizationProjection>(
      `/invoices/${invoiceId}/fiscalization/finalize`,
      body,
    );
  }

  async retry(
    invoiceId: string,
    body: FiscalActionRequest,
  ): Promise<ApiResponse<FiscalizationProjection>> {
    return apiClient.post<FiscalizationProjection>(
      `/invoices/${invoiceId}/fiscalization/retry`,
      body,
    );
  }

  async reconcile(
    invoiceId: string,
    body: FiscalActionRequest,
  ): Promise<ApiResponse<FiscalizationProjection>> {
    return apiClient.post<FiscalizationProjection>(
      `/invoices/${invoiceId}/fiscalization/reconcile`,
      body,
    );
  }

  async voidDocument(
    invoiceId: string,
    body: FiscalActionRequest,
  ): Promise<ApiResponse<FiscalizationProjection>> {
    return apiClient.post<FiscalizationProjection>(
      `/invoices/${invoiceId}/fiscalization/void`,
      body,
    );
  }

  /**
   * Streams archived PDF bytes. Note: API ArtifactService may be nil until composed;
   * callers should handle unavailable/503 with provider-neutral guidance.
   */
  async downloadArtifact(
    invoiceId: string,
    artifactId: string,
  ): Promise<ApiResponse<Blob>> {
    const token = apiClient.getToken();
    const base = `${getPublicApiOrigin()}/api/v1`;
    const url = `${base}/invoices/${invoiceId}/fiscal-artifacts/${artifactId}`;
    try {
      const response = await fetch(url, {
        headers: token ? { Authorization: `Bearer ${token}` } : {},
      });
      if (!response.ok) {
        let message = 'PDF temporariamente indisponível.';
        try {
          const errBody = (await response.json()) as { error?: string };
          if (errBody.error) message = errBody.error;
        } catch {
          /* keep default */
        }
        return {
          success: false,
          error: { message, status: response.status },
        };
      }
      const blob = await response.blob();
      return { success: true, data: blob };
    } catch {
      return {
        success: false,
        error: { message: 'Não foi possível descarregar o PDF.', status: 0 },
      };
    }
  }
}

export const fiscalizationService = FiscalizationService.getInstance();
