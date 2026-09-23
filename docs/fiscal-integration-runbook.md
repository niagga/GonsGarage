# Fiscal integration runbook

Operational guide for the Cloudware fiscal foundation in GonsGarage. Legal production mutations stay **fail-closed** until parent tasks **13.2–13.4** record approved policy, storage, credentials, and gate evidence. This document does not authorize live Cloudware calls.

Related:

- Scenario matrix: [`docs/fiscal-integration-traceability.md`](fiscal-integration-traceability.md)
- Change design: `openspec/changes/cloudware-fiscal-integration-foundation/design.md`
- Deploy notes: [`deploy/README.md`](../deploy/README.md)

## Independent configuration switches

| Switch | Default | Meaning |
|--------|---------|---------|
| `FISCAL_FEATURE_ENABLED` | `false` | Additive fiscal routes, schema verification, projections |
| `FISCAL_FINALIZATION_ENABLED` | `false` | Legal freeze / retry / void enqueue (dark launch off) |
| `FISCAL_WORKER_ENABLED` | `false` | Outbox claim loop (`cmd/fiscal-worker`) |
| `FISCAL_PROVIDER` | empty → `mock` only outside production | Selected provider key |
| `FISCAL_CLOUDWARE_MUTATIONS` | `false` | Cloudware HTTP mutations (WU12; remain off) |
| `FISCAL_ARTIFACT_BACKEND` | `local` (dev) / `object` (prod example) | Artifact store |
| `FISCAL_CREDENTIAL_KEY_*` | unset | Dedicated AES-GCM keyring (never JWT secret) |

Production refuses: default JWT secrets, mock provider, local artifact backend, missing migration 011 schema, missing credential keyring.

## Rollout stages (dark launch)

1. **Schema dark launch** — Run migrate with feature/finalization/worker **off**. Verify migration version, triggers, indexes, zero outbox rows, existing invoices `legacy_unfiscalized`.
2. **Provider-neutral backend dark launch** — Enable `FISCAL_FEATURE_ENABLED=true` with finalization still off. Validate legacy invoice regressions and deletion protection.
3. **Mock validation (non-production only)** — `APP_ENV!=production`, `FISCAL_PROVIDER=mock`. Exercise issuance, ambiguity, reconcile, void, labeled PDFs.
4. **Production configuration** — Dedicated keyring, approved issuer/policy, private object storage readiness flags, retention/backup/access logging, monitoring. Keep Cloudware mutations and finalization off.
5. **Cloudware controlled validation** — Parent 13.2–13.4 + WU12 only. Collect gate evidence outside the repo.
6. **Explicit legal enablement** — Enable finalization and worker only after readiness passes. Monitor unknown outcomes, lease expiry, connection state, artifact recovery.

### Compose commands

```bash
# Dev schema + optional worker (profile fiscal); worker still requires FISCAL_WORKER_ENABLED=true
docker compose --profile fiscal up -d migrate
docker compose --profile fiscal up -d fiscal-worker

# Production schema one-shot
docker compose -f docker-compose.prod.yml --profile fiscal --env-file .env.prod run --rm gonsgarage-migrate
```

## Production prerequisites checklist

- [ ] `APP_ENV=production` with non-default `JWT_SECRET`
- [ ] Migration `011` applied; `VerifyFiscalSchema` passes
- [ ] `FISCAL_PROVIDER` is **not** `mock`
- [ ] `FISCAL_ARTIFACT_BACKEND=object` with private ACL, encryption, retention, backup/restore, and access-log evidence flags
- [ ] Dedicated `FISCAL_CREDENTIAL_KEY_VERSION` + `FISCAL_CREDENTIAL_KEY_B64`
- [ ] Approved fiscal policy / issuer / series recorded (parent 13.2)
- [ ] Every applicable `fiscal_enablement_gates` row approved (parent 13.4)
- [ ] `FISCAL_FINALIZATION_ENABLED` and `FISCAL_WORKER_ENABLED` remain `false` until the above are true

## Monitoring and alerts

Core API `/ready` is **PostgreSQL only**. Provider outages must not remove the API from rotation.

- Dependency status: `GET /fiscal/dependency-status` (feature/finalization/worker flags, provider health, schema, issues)
- Structured fiscal logs: allowlisted fields only (`correlation_id`, document/event/attempt IDs, provider key, operation, states, classification, lease owner, duration). No NIF, names, tokens, OAuth query, URLs, PDF bytes, or ciphertext.
- Alert on: oldest unknown outcome age, repeated expired leases, connection `action_required`, artifact recovery beyond policy, checksum compromise, production readiness regression.

## Credential rotation

1. Add new key version to the fiscal keyring (decrypt-old / encrypt-new).
2. Re-encrypt connection credential rows under optimistic locking + audit.
3. Retire old version only after all rows report the new version.
4. Never log plaintext tokens or ciphertext; never reuse `JWT_SECRET`.

## Backup, restore, and artifact retention

- Database: `pg_dump` / restore per [`deploy/README.md`](../deploy/README.md) before schema changes.
- Artifacts: private immutable object store with retention configuration and access logging; restore must preserve checksum identity.
- After any real fiscal use, prefer **forward-fix** migrations only. Do not delete frozen documents, attempts, transitions, provider references, or archived PDFs.

## Forward-fix rollback

1. Set `FISCAL_FINALIZATION_ENABLED=false` and `FISCAL_WORKER_ENABLED=false`.
2. Let leased work finish or expire into the conservative unknown path.
3. Keep read projections and authorized PDF downloads available.
4. Software rollback does **not** void a legal document — use the authorized void/correction workflow.
5. Invoice deletion guards and DB triggers must continue protecting retained fiscal history even if UI/API fiscal routes are disabled.

## Legal-history preservation

Frozen snapshots, outbox attempts, transitions, provider references, connection audit metadata, and archived artifacts are append-only evidence. Operators must not rewrite or physically delete them to “undo” an issuance.
