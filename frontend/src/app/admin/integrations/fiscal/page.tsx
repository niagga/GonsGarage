'use client';

import React, { Suspense, useCallback, useEffect, useState } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import { useAuth } from '@/stores';
import AppShell from '@/components/layout/AppShell';
import { Button } from '@/components/ui/button';
import { fiscalIntegrationService } from '@/lib/services/fiscal-integration.service';
import { canManageUsers } from '@/types/user';
import {
  fiscalConnectionStateLabel,
  type FiscalConnectionStatus,
  type FiscalReadiness,
} from '@/types/fiscal';
import styles from '../../../accounting/accounting.module.css';

const DEFAULT_SCOPE = 'default';
const DEFAULT_PROVIDER = 'mock';

const OAUTH_LEAK_PARAMS = [
  'code',
  'state',
  'access_token',
  'refresh_token',
  'id_token',
  'token',
  'credential',
];

function deriveOauthReturnNotice(params: URLSearchParams): string | null {
  const oauth = params.get('oauth');
  const hasLeak = OAUTH_LEAK_PARAMS.some((key) => params.has(key));
  if (oauth === 'success' || oauth === 'ok') {
    return 'Retorno OAuth: autorização concluída. Os códigos sensíveis foram removidos do URL.';
  }
  if (oauth === 'error' || oauth === 'failed') {
    return 'Retorno OAuth: a autorização falhou. Tente novamente sem partilhar o URL.';
  }
  if (hasLeak) {
    return 'Retorno OAuth processado. Parâmetros sensíveis foram removidos do URL.';
  }
  return null;
}

function FiscalIntegrationAdminContent() {
  const { user, logout } = useAuth();
  const router = useRouter();
  const searchParams = useSearchParams();
  const [connection, setConnection] = useState<FiscalConnectionStatus | null>(null);
  const [readiness, setReadiness] = useState<FiscalReadiness | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [oauthNotice, setOauthNotice] = useState<string | null>(null);
  const [seenOauthParams, setSeenOauthParams] = useState('');
  const [busy, setBusy] = useState(false);

  const oauthParamsKey = searchParams.toString();
  if (oauthParamsKey !== seenOauthParams) {
    setSeenOauthParams(oauthParamsKey);
    const derived = deriveOauthReturnNotice(searchParams);
    if (derived) {
      setOauthNotice(derived);
    }
  }

  const load = useCallback(async () => {
    setError(null);
    const [statusRes, readinessRes] = await Promise.all([
      fiscalIntegrationService.getConnectionStatus(DEFAULT_SCOPE, DEFAULT_PROVIDER),
      fiscalIntegrationService.getReadiness(),
    ]);
    if (statusRes.success && statusRes.data) {
      setConnection(statusRes.data);
    } else {
      setConnection(null);
      setError(statusRes.error?.message ?? 'Não foi possível carregar o estado da ligação.');
    }
    if (readinessRes.success && readinessRes.data) {
      setReadiness(readinessRes.data);
    } else {
      setReadiness({
        ready: false,
        gates: [
          {
            name: 'connection',
            status: 'pending',
            guidance:
              'A prontidão da integração ainda não está disponível. Confirme a ligação e as verificações no servidor antes de ativar a emissão legal.',
          },
        ],
      });
    }
  }, []);

  useEffect(() => {
    let cancelled = false;
    queueMicrotask(() => {
      if (!cancelled) void load();
    });
    return () => {
      cancelled = true;
    };
  }, [load]);

  useEffect(() => {
    const oauth = searchParams.get('oauth');
    const hasLeak = OAUTH_LEAK_PARAMS.some((key) => searchParams.has(key));
    if (!oauth && !hasLeak) return;
    router.replace('/admin/integrations/fiscal', { scroll: false });
  }, [searchParams, router]);

  if (!user) return null;

  if (!canManageUsers(user)) {
    return (
      <AppShell
        user={user}
        subtitle="Integração fiscal"
        activeNav="admin_fiscal"
        carsNavLabel="Viaturas"
        onLogout={logout}
        logoVariant="branded"
      >
        <div className={styles.error}>Apenas gestores e administradores podem gerir a integração fiscal.</div>
      </AppShell>
    );
  }

  async function onVerify() {
    setBusy(true);
    setError(null);
    const res = await fiscalIntegrationService.verifyConnection(DEFAULT_SCOPE, DEFAULT_PROVIDER);
    setBusy(false);
    if (res.success && res.data) {
      setConnection(res.data);
      return;
    }
    setError(res.error?.message ?? 'Falha ao verificar a ligação.');
  }

  async function onDisconnect() {
    setBusy(true);
    setError(null);
    const res = await fiscalIntegrationService.revokeConnection(DEFAULT_SCOPE, DEFAULT_PROVIDER);
    setBusy(false);
    if (res.success && res.data) {
      setConnection(res.data);
      return;
    }
    setError(res.error?.message ?? 'Falha ao desligar a ligação.');
  }

  return (
    <AppShell
      user={user}
      subtitle="Integração fiscal"
      activeNav="admin_fiscal"
      carsNavLabel="Viaturas"
      onLogout={logout}
      logoVariant="branded"
    >
      <h1>Integração fiscal</h1>
      <p className={styles.intro}>
        Estado da ligação ao fornecedor fiscal e orientação de prontidão. Credenciais e tokens
        permanecem no servidor.
      </p>
      {oauthNotice ? (
        <div className={styles.error} role="status" data-testid="oauth-return-status">
          {oauthNotice}
        </div>
      ) : null}
      {error ? <div className={styles.error}>{error}</div> : null}

      <section className={styles.readonlyBlock} aria-labelledby="fiscal-connection-heading">
        <h2 id="fiscal-connection-heading">Ligação</h2>
        <div>
          <strong>Estado:</strong> {fiscalConnectionStateLabel(connection?.state)}
        </div>
        <div>
          <strong>Credenciais armazenadas:</strong>{' '}
          {connection?.hasCredentials ? 'Sim (servidor)' : 'Não'}
        </div>
        <div className={styles.rowActions}>
          <Button type="button" disabled={busy} onClick={() => void onVerify()}>
            Verificar
          </Button>
          <Button type="button" variant="destructive" disabled={busy} onClick={() => void onDisconnect()}>
            Desligar
          </Button>
        </div>
      </section>

      <section className={styles.readonlyBlock} aria-labelledby="fiscal-readiness-heading">
        <h2 id="fiscal-readiness-heading">Prontidão</h2>
        <p>
          {readiness?.ready
            ? 'Todos os requisitos aplicáveis estão satisfeitos.'
            : 'A emissão legal permanece bloqueada até todos os requisitos aplicáveis estarem concluídos.'}
        </p>
        <ul>
          {(readiness?.gates ?? []).map((gate) => (
            <li key={gate.name}>
              <strong>{gate.name}</strong> ({gate.status})
              {gate.guidance ? ` — ${gate.guidance}` : ''}
            </li>
          ))}
        </ul>
        {readiness?.atCommunicationStatus ? (
          <p role="status">Comunicação fiscal evidenciada: {readiness.atCommunicationStatus}</p>
        ) : null}
      </section>
    </AppShell>
  );
}

export default function FiscalIntegrationAdminPage() {
  return (
    <Suspense fallback={null}>
      <FiscalIntegrationAdminContent />
    </Suspense>
  );
}
