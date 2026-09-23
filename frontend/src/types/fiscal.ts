/** Provider-neutral fiscalization DTOs (decimal fields are canonical strings). */

export type FiscalDocumentKind = 'FT' | 'FR';

export type FiscalLifecycle =
  | 'draft'
  | 'pending'
  | 'dispatching'
  | 'issued'
  | 'rejected'
  | 'connection_action_required'
  | 'retryable_failure'
  | 'outcome_unknown'
  | 'void_pending'
  | 'void_outcome_unknown'
  | 'voided';

export type FiscalPresentationStatus =
  | 'legacy_unfiscalized'
  | 'draft'
  | 'pending'
  | 'finalized'
  | 'voided'
  | 'unavailable';

export type FiscalDocumentAction =
  | 'view'
  | 'edit'
  | 'delete'
  | 'finalize'
  | 'retry'
  | 'reconcile'
  | 'void'
  | 'supersede';

export interface FiscalParty {
  legalName: string;
  taxIdentifier: string;
  countryCode: string;
}

export interface FiscalBillingAddress {
  line1: string;
  postalCode: string;
  city: string;
  countryCode: string;
}

export interface FiscalLineSource {
  type: string;
  id: string;
}

export interface FiscalDraftLine {
  position: number;
  description: string;
  quantity: string;
  unitCode: string;
  unitPrice: string;
  discount?: { kind: string; value: string };
  taxTreatmentCode: string;
  taxRate: string;
  exemptionCode?: string | null;
  source?: FiscalLineSource;
}

export interface FiscalDraftRequest {
  version: number;
  kind: FiscalDocumentKind | string;
  intentSlot?: string;
  issuerProfileId?: string;
  policyKey: string;
  currency: string;
  customer: FiscalParty;
  billingAddress: FiscalBillingAddress;
  lines: FiscalDraftLine[];
  declaredTotals?: { payableTotal: string };
}

export interface FiscalActionRequest {
  expectedVersion: number;
  providerKey?: string;
  connectionId?: string;
  policyVersionId?: string;
  issuerProfileId?: string;
}

export interface FiscalizationProjection {
  invoiceId: string;
  documentId?: string | null;
  kind?: string;
  lifecycle?: string;
  status: string;
  version?: number;
  currency?: string;
  payableTotal?: string;
  grossTotal?: string;
  taxTotal?: string;
  allowedActions: string[];
  artifactStatus?: string;
  /** Optional until API projection exposes artifact id for downloads. */
  artifactId?: string;
  artifactClassification?: 'mock' | 'legal' | string;
  lastErrorCode?: string;
  lastErrorMessage?: string;
  readinessIssues?: string[];
}

export interface FiscalizationSummary {
  invoiceId: string;
  status: string;
  lifecycle?: string;
  allowedActions?: string[];
}

export interface FiscalizationSummariesResponse {
  items: FiscalizationSummary[];
}

/** Client-side polling stops when lifecycle is not in-flight. */
export const FISCAL_TRANSIENT_LIFECYCLES = new Set([
  'pending',
  'dispatching',
  'void_pending',
]);

export const DEFAULT_FISCAL_POLL_INTERVAL_MS = 2000;
export const DEFAULT_FISCAL_MAX_POLL_ATTEMPTS = 30;

export function isTransientFiscalLifecycle(lifecycle: string | undefined): boolean {
  if (!lifecycle) return false;
  return FISCAL_TRANSIENT_LIFECYCLES.has(lifecycle);
}

export function fiscalPresentationLabel(status: string): string {
  switch (status) {
    case 'legacy_unfiscalized':
      return 'Sem fiscalização (legado)';
    case 'draft':
      return 'Rascunho';
    case 'pending':
      return 'Pendente';
    case 'finalized':
      return 'Finalizado';
    case 'voided':
      return 'Anulado';
    case 'unavailable':
      return 'Indisponível';
    default:
      return status || '—';
  }
}

/** Privileged legal actions require manager/admin on the client even if listed. */
export const FISCAL_PRIVILEGED_ACTIONS = new Set([
  'finalize',
  'retry',
  'reconcile',
  'void',
]);

export function isManagerOrAdminRole(role: string): boolean {
  return role === 'manager' || role === 'admin';
}

export function canShowFiscalAction(role: string, action: string, allowed: string[]): boolean {
  if (!allowed.includes(action)) return false;
  if (FISCAL_PRIVILEGED_ACTIONS.has(action) && !isManagerOrAdminRole(role)) {
    return false;
  }
  return true;
}
