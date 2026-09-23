import { describe, expect, it } from 'vitest';
import {
  canDownloadFiscalArtifact,
  canShowFiscalAction,
  clientFiscalPresentationStatus,
  fiscalConnectionStateLabel,
  fiscalPresentationLabel,
  isTransientFiscalLifecycle,
} from '@/types/fiscal';

describe('fiscal presentation helpers', () => {
  it('maps presentation statuses to Portuguese provider-neutral labels', () => {
    expect(fiscalPresentationLabel('legacy_unfiscalized')).toMatch(/legado/i);
    expect(fiscalPresentationLabel('finalized')).toBe('Finalizado');
    expect(fiscalPresentationLabel('voided')).toBe('Anulado');
  });

  it('treats pending/dispatching/void_pending as transient for polling', () => {
    expect(isTransientFiscalLifecycle('pending')).toBe(true);
    expect(isTransientFiscalLifecycle('dispatching')).toBe(true);
    expect(isTransientFiscalLifecycle('void_pending')).toBe(true);
    expect(isTransientFiscalLifecycle('issued')).toBe(false);
    expect(isTransientFiscalLifecycle('outcome_unknown')).toBe(false);
  });

  it('hides privileged actions for employees and allows managers', () => {
    const allowed = ['view', 'finalize', 'retry', 'void', 'reconcile'];
    expect(canShowFiscalAction('employee', 'finalize', allowed)).toBe(false);
    expect(canShowFiscalAction('employee', 'edit', ['edit'])).toBe(true);
    expect(canShowFiscalAction('manager', 'finalize', allowed)).toBe(true);
    expect(canShowFiscalAction('admin', 'void', allowed)).toBe(true);
    expect(canShowFiscalAction('manager', 'finalize', ['view'])).toBe(false);
  });

  it('maps draft and legacy to unavailable for client simplified surfaces', () => {
    expect(clientFiscalPresentationStatus('draft')).toBe('unavailable');
    expect(clientFiscalPresentationStatus('legacy_unfiscalized')).toBe('unavailable');
    expect(clientFiscalPresentationStatus('pending')).toBe('pending');
    expect(clientFiscalPresentationStatus('finalized')).toBe('finalized');
  });

  it('labels connection states in Portuguese without provider branding', () => {
    expect(fiscalConnectionStateLabel('connected')).toBe('Ligado');
    expect(fiscalConnectionStateLabel('disconnected')).toBe('Desligado');
    expect(fiscalConnectionStateLabel('action_required')).toBe('Ação necessária');
  });

  it('allows PDF download only when artifact is available with an id', () => {
    expect(canDownloadFiscalArtifact({ artifactStatus: 'available', artifactId: 'a1' })).toBe(true);
    expect(canDownloadFiscalArtifact({ artifactStatus: 'available' })).toBe(false);
    expect(canDownloadFiscalArtifact({ artifactStatus: 'compromised', artifactId: 'a1' })).toBe(false);
    expect(canDownloadFiscalArtifact({ artifactStatus: 'unavailable', artifactId: 'a1' })).toBe(false);
  });
});
