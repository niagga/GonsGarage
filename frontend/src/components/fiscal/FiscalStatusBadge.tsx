'use client';

import React from 'react';
import {
  clientFiscalPresentationStatus,
  fiscalPresentationLabel,
} from '@/types/fiscal';

export type FiscalStatusBadgeProps = Readonly<{
  status: string | undefined;
  /** When true, map draft/legacy to unavailable for client surfaces. */
  clientSimplified?: boolean;
}>;

/** Provider-neutral fiscal presentation badge shared by staff and client UIs. */
export function FiscalStatusBadge({ status, clientSimplified = false }: FiscalStatusBadgeProps) {
  const resolved = clientSimplified ? clientFiscalPresentationStatus(status) : (status ?? '—');
  return <span data-testid="fiscal-status-badge">{fiscalPresentationLabel(resolved)}</span>;
}
