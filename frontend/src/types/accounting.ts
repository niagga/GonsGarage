/** P1 accounting / billing types (camelCase, aligned with Go JSON). */

export type BillingDocumentKind =
  | 'payroll'
  | 'irs'
  | 'other';

export interface Supplier {
  id: string;
  name: string;
  contactEmail: string;
  contactPhone: string;
  taxId: string;
  notes: string;
  isActive: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface ReceivedInvoice {
  id: string;
  supplierId?: string;
  vendorName: string;
  category: string;
  amount: number;
  invoiceDate: string;
  notes: string;
  createdAt: string;
  updatedAt: string;
}

export interface BillingDocument {
  id: string;
  kind: BillingDocumentKind;
  title: string;
  amount: number;
  customerId?: string;
  reference: string;
  notes: string;
  createdAt: string;
  updatedAt: string;
}

/** Customer-issued invoice (emitida al cliente). */
export interface IssuedInvoice {
  id: string;
  customerId: string;
  amount: number;
  status: string;
  notes: string;
  createdAt: string;
  updatedAt: string;
}

export interface ItemsTotal<T> {
  items: T[];
  total: number;
}
