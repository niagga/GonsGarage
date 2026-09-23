import { apiClient, type ApiResponse } from '../api-client';
import type { IssuedInvoice, ItemsTotal } from '@/types/accounting';

export type { IssuedInvoice } from '@/types/accounting';

export class IssuedInvoiceService {
  private static instance: IssuedInvoiceService;
  static getInstance(): IssuedInvoiceService {
    if (!IssuedInvoiceService.instance) IssuedInvoiceService.instance = new IssuedInvoiceService();
    return IssuedInvoiceService.instance;
  }
  private constructor() {}

  /** Client: own invoices */
  async listMine(limit = 20, offset = 0): Promise<ApiResponse<ItemsTotal<IssuedInvoice>>> {
    return apiClient.get<ItemsTotal<IssuedInvoice>>(`/invoices/me?limit=${limit}&offset=${offset}`);
  }

  async get(id: string): Promise<ApiResponse<IssuedInvoice>> {
    return apiClient.get<IssuedInvoice>(`/invoices/${id}`);
  }

  async patchIssuedInvoice(
    id: string,
    body: { notes?: string; status?: string; amount?: number },
  ): Promise<ApiResponse<IssuedInvoice>> {
    return apiClient.patch<IssuedInvoice>(`/invoices/${id}`, body);
  }

  async patchNotes(id: string, notes: string): Promise<ApiResponse<IssuedInvoice>> {
    return this.patchIssuedInvoice(id, { notes });
  }

  /** Staff */
  async listStaff(limit = 20, offset = 0): Promise<ApiResponse<ItemsTotal<IssuedInvoice>>> {
    return apiClient.get<ItemsTotal<IssuedInvoice>>(`/invoices?limit=${limit}&offset=${offset}`);
  }

  /** Staff: invoice linked to a repair (empty items if none). */
  async getByRepair(repairId: string): Promise<ApiResponse<ItemsTotal<IssuedInvoice>>> {
    const q = new URLSearchParams({ repairId });
    return apiClient.get<ItemsTotal<IssuedInvoice>>(`/invoices?${q.toString()}`);
  }

  async createStaff(body: {
    customerId: string;
    amount: number;
    status?: string;
    notes?: string;
    repairId?: string;
  }): Promise<ApiResponse<IssuedInvoice>> {
    return apiClient.post<IssuedInvoice>('/invoices', body);
  }

  async removeStaff(id: string): Promise<ApiResponse<unknown>> {
    return apiClient.delete(`/invoices/${id}`);
  }
}

export const issuedInvoiceService = IssuedInvoiceService.getInstance();
