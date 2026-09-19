# Design: Provider-neutral fiscal integration foundation

## Decision summary

GonsGarage will keep `Invoice` as the mutable operational/customer-workflow record and add a separate `FiscalDocument` aggregate. A fiscal draft is editable by workshop staff; finalization by a manager or admin freezes an exact canonical snapshot, fixes the provider and connection, and creates one PostgreSQL outbox event in the same transaction. Provider calls run outside HTTP requests through a lease-based worker. Any result that might have reached the provider becomes unknown and can only be reconciled with that same provider before another issue call is allowed.

The first executable provider is a deterministic development/test mock. The Cloudware adapter and OAuth boundary may be built, but production mutations remain fail-closed until all recorded enablement gates, fiscal-policy inputs, issuer configuration, credential protection, and artifact-storage readiness checks pass. No Cloudware behavior absent from the research artifact is assumed.

### Core decisions

| Area | Decision |
|---|---|
| Aggregate | `FiscalDocument` is the aggregate root; its snapshot and lines are editable only while state is `draft` and database-protected from update/delete after freezing. |
| Legacy compatibility | `Invoice` fields, paths, role semantics, pagination, timestamps, and client notes behavior remain unchanged. An internal `fiscal_eligibility` column distinguishes pre-deployment rows without changing the legacy DTO. |
| Arithmetic | API decimals are strings, Go uses exact decimal arithmetic, and PostgreSQL uses `NUMERIC(38,18)`. A versioned approved policy determines legal scale and rounding; the storage scale is capacity, not a fiscal rule. |
| Finalization | A single database transaction locks the invoice and draft, validates current configuration, writes canonical frozen bytes and hash, changes state to `pending`, and inserts one outbox event. |
| Dispatch | A PostgreSQL worker uses `FOR UPDATE SKIP LOCKED`, lease tokens, and pre-call attempt records. A crashed started call is treated as ambiguous, never blindly repeated. |
| Provider boundary | Gateway methods are `VerifyConnection`, `Issue`, `Reconcile`, `Void`, and `FetchArtifact`; capabilities and errors are normalized before leaving an adapter. |
| Recovery | Legal retries and reconciliation are explicit manager/admin actions in this release. No automatic Cloudware retry timing is invented. Artifact recovery may be automatic under separately configured bounded operational settings. |
| OAuth | Authorization starts under manager/admin authentication. State is one-time, hashed, actor-bound, and expiring. Tokens are server-side and envelope-encrypted with a dedicated versioned key. |
| PDF | Provider PDFs are copied to immutable private storage under deterministic keys, checksummed, and downloaded only through an authenticated ownership check. |
| Runtime | API and worker are separate commands using the same domain/services/repositories. Provider health is not part of core API readiness. |

## Scope and non-decisions

This design implements the four change specifications without enabling unknown fiscal policy or undocumented Cloudware behavior. The following are mandatory external configuration/evidence, not constants selected by this design:

- supported currency or currencies and currency minor-unit precision;
- quantity and unit-price scale limits;
- rounding mode and rounding point;
- discount order and legal document-level adjustment rules;
- approved IVA/tax codes, percentages, regions, and exemption-reason catalog;
- customer identity requirements for each supported case;
- issuer legal identity, tax regime, address, and approved Cloudware series mapping;
- Cloudware sandbox/controlled-validation access, OAuth details, safe idempotency/reconciliation lookup, rate limits/retry behavior, PDF semantics, AT/e-Fatura semantics, and credentialed error/void behavior;
- legal retention duration and production storage/backup/restore controls.

Migrations create no approved production policy, issuer profile, provider connection, or legal dispatch work. Finalization reports configuration errors and remains in `draft` until all applicable inputs are explicitly approved.

## Aggregate boundaries and invariants

### Operational invoice boundary

`domain.Invoice` remains the authority for the existing customer ID, binary-float compatibility amount, operational status, notes, and timestamps. Changing these fields after fiscal finalization does not alter fiscal content. Existing POST/PATCH behavior is preserved, including client notes-only updates.

The only database addition is `invoices.fiscal_eligibility`:

- migration default and backfill: `legacy_unfiscalized`;
- the upgraded `CreateInvoice` path explicitly writes `eligible` for newly created invoices;
- old application binaries or direct inserts omit the field and therefore fail safe as legacy;
- no status change, date, scan, payment, repair, or migration promotes a legacy row.

A legacy invoice requires a future separately approved workflow before it can host a draft. This change has no such endpoint.

### Fiscal document aggregate

The aggregate consists of:

- `FiscalDocument`: identity, source invoice, intent slot, lifecycle, optimistic version, fixed provider/connection after finalization, operation keys, provider result, and safe last-error summary;
- `FiscalSnapshot`: issuer/customer/address JSON, policy reference, exact totals, canonical frozen representation, and digest;
- `FiscalLine`: captured manual values and optional traceability reference;
- append-only `FiscalStateTransition` records;
- related attempts, outbox events, and artifacts, which are operational records around the aggregate rather than mutable legal input.

First release uses one intent slot, `primary_sale`, per invoice. At most one non-superseded document can occupy that slot. A rejected frozen intent may only be superseded by an explicit staff draft-creation request naming that rejected document. The service marks it superseded and creates the replacement atomically. `issued`, `voided`, unknown, pending, and recoverable documents cannot be superseded. This prevents hidden reissuance while retaining a clear path after a definite rejection.

Aggregate invariants:

1. Kind is exactly `FT` or `FR`.
2. A draft can exist only for an `eligible` invoice and must refer to that invoice's customer.
3. Source links are nullable traceability pairs (`repair` or `part`, UUID); they never drive recalculation after capture.
4. Only `draft` permits snapshot or line updates/deletes.
5. A non-draft document has non-null `frozen_at`, policy version, provider key, connection ID, intent key, issue operation key, canonical bytes, and canonical SHA-256.
6. Provider, connection, intent key, operation key, kind, snapshot, and lines never change after finalization.
7. Each provider request is built only from canonical frozen bytes plus adapter configuration; mutable invoice/user/repair/part records are not read for mapping.
8. Provider references are unique within a provider when known.
9. State changes occur only through the transition function and append an audit row in the same transaction.
10. Physical invoice deletion is rejected when any non-draft fiscal document exists. If only a draft exists, the legacy DELETE service may delete the draft aggregate first in the same transaction, preserving the historical behavior for invoices with no protected intent.

## Lifecycle state machine

### States

| State | Meaning | Allowed user action |
|---|---|---|
| `draft` | Mutable fiscal preparation; no legal operation exists. | Staff edit/delete; manager/admin finalize. |
| `pending` | Frozen issuance intent durably queued. | Read only. |
| `dispatching` | Worker has recorded an issuance attempt before calling the provider. | Read only. |
| `issued` | Provider existence is definite; PDF may still be recovering. | Manager/admin request void when capability/policy allow. |
| `rejected` | Definite validation or permanent issuance rejection. | Staff may explicitly create a replacement draft that supersedes this intent. |
| `connection_action_required` | Definite authorization/connection problem occurred before an accepted issue result. | Manager/admin reconnect, verify, then retry. |
| `retryable_failure` | Definite non-acceptance due to transient or rate-limit failure. | Manager/admin retry. |
| `outcome_unknown` | Issuance may have happened. Issue retry is prohibited. | Manager/admin reconcile only. |
| `void_pending` | Authorized void is durably queued. | Read only. |
| `void_outcome_unknown` | Void may have happened. Another void is prohibited. | Manager/admin reconcile only. |
| `voided` | Provider definitively confirms void. | Read only. |

### Transition table

| From | To | Guard and operation |
|---|---|---|
| `draft` | `pending` | Manager/admin; complete approved policy and issuer; healthy allowed connection; finalization transaction succeeds. |
| `pending` | `dispatching` | Issue worker owns an unexpired lease and inserts a started attempt. |
| `dispatching` | `issued` | Definite provider confirmation or reconciliation finds the exact document. |
| `dispatching` | `rejected` | Definite validation/permanent rejection. |
| `dispatching` | `connection_action_required` | Definite authorization failure with no accepted operation. |
| `dispatching` | `retryable_failure` | Definite transient/rate-limit failure proving non-acceptance. |
| `dispatching` | `outcome_unknown` | Timeout, connection loss after call start, malformed success, or expired lease with an unfinished started attempt. |
| `retryable_failure` | `pending` | Manager/admin enqueues the same issue operation key. |
| `connection_action_required` | `pending` | Manager/admin action after the original connection verifies healthy. |
| `outcome_unknown` | `issued` | Same-provider reconciliation finds the exact operation/document. |
| `outcome_unknown` | `rejected` | Same-provider reconciliation proves a definite rejection. |
| `outcome_unknown` | `retryable_failure` | Evidenced lookup proves no document exists and an approved policy allows retry. |
| `outcome_unknown` | `outcome_unknown` | Reconciliation remains inconclusive or cannot authenticate; connection state may separately become action-required. |
| `issued` | `void_pending` | Manager/admin; provider capability and approved policy permit void; one void operation key is created. |
| `void_pending` | `voided` | Definite provider confirmation. |
| `void_pending` | `issued` | Definite refusal/non-acceptance; failed attempt remains. |
| `void_pending` | `void_outcome_unknown` | Void result may have reached provider or leased call is unfinished. |
| `void_outcome_unknown` | `voided` | Reconciliation proves void. |
| `void_outcome_unknown` | `issued` | Reconciliation proves the original remains issued and no void occurred. |
| `void_outcome_unknown` | `void_outcome_unknown` | Reconciliation is inconclusive. |

There is no transition back to `draft`. There is no transition from an unknown state to an issue/void pending state without a definitive reconciliation result. Artifact status is orthogonal and cannot move an `issued` document backward.

## Exact decimal and fiscal-policy seam

### Representation

- HTTP request/response decimal fields are canonical base-10 strings such as `"2.500"` or `"12.34"`; JSON numbers are rejected on new fiscal endpoints.
- Go uses one `domain.Decimal` value type backed by `github.com/shopspring/decimal`. Construction accepts strict non-exponent decimal strings, normalizes negative zero, rejects NaN/infinity, and never converts through `float64`.
- PostgreSQL stores quantities, rates, prices, discounts, and totals as `NUMERIC(38,18)`. Application validation applies the smaller approved policy limits before persistence/finalization.
- Frozen canonical JSON writes decimals as normalized strings and orders object keys/lines deterministically. Its bytes and SHA-256 are stored. Retries deserialize those bytes rather than rebuilding from current tables.
- Legacy `Invoice.Amount float64` is neither copied automatically nor used as an authoritative total. Staff enters fiscal lines; the UI may display the operational amount only as a comparison warning.

### Policy contract

`FiscalPolicyResolver.Resolve(policyKey, version)` returns an immutable `ArithmeticPolicy` containing:

- currency and currency output scale;
- maximum quantity and unit-price scales;
- supported tax treatments and exact rates;
- tax-region and exemption-reason requirements;
- discount type/order rules;
- rounding mode and whether rounding occurs per line or at document total;
- permitted rounding-adjustment range and sign;
- customer identity requirements by document kind;
- FT/FR and void permissions.

The calculator is a pure domain service. It computes gross, normalized discount, net, tax, line total, document subtotals, adjustment, and payable total. Caller-declared totals are comparison inputs only; a mismatch is a 422 error. Policy configuration is stored as schema-versioned JSON with a digest and approval metadata. Once approved or referenced by a frozen snapshot, it is immutable; correction requires a new version.

No production policy row is seeded. A development/test mock policy may be loaded only when `APP_ENV` is not production and is visibly classified `mock`. Missing or unapproved currency, IVA, exemption, precision, rounding, discount, adjustment, customer identity, issuer profile, or series input makes readiness false and finalization fail closed before outbox creation.

## PostgreSQL migration design

Create `backend/migrations/011_fiscal_integration_foundation.up.sql`. The migration runner already wraps each file in a transaction, so this file must not add its own `BEGIN/COMMIT`. Use PostgreSQL check constraints rather than database enum types so later additive states do not require enum rewrites.

### Existing-table change

```sql
ALTER TABLE invoices
  ADD COLUMN fiscal_eligibility varchar(32) NOT NULL DEFAULT 'legacy_unfiscalized',
  ADD CONSTRAINT invoices_fiscal_eligibility_ck
    CHECK (fiscal_eligibility IN ('legacy_unfiscalized','eligible'));
```

No UPDATE to `eligible` is run. New application writes explicitly use `eligible`. Existing invoice JSON omits this column.

### Configuration and connection tables

`fiscal_policy_versions`

- `id uuid PK`, `policy_key varchar(80)`, `version integer > 0`, `schema_version integer`, `classification varchar(16) CHECK legal|mock`;
- `status varchar(16) CHECK draft|approved|retired`, `config jsonb`, `config_sha256 char(64)`;
- `approved_by uuid NULL REFERENCES users(id) ON DELETE RESTRICT`, `approved_at timestamptz NULL`, timestamps;
- `UNIQUE(policy_key, version)`; partial unique index on `policy_key WHERE status='approved'`;
- trigger prevents update/delete after approval or after any snapshot references the row.

`fiscal_issuer_profiles`

- `id uuid PK`, `scope_key varchar(80) NOT NULL DEFAULT 'default'`, `version integer > 0`, `status draft|approved|retired`;
- `legal_identity jsonb`, `billing_address jsonb`, `tax_profile jsonb`, `series_config jsonb`, `config_sha256 char(64)`;
- approval actor/time and timestamps;
- `UNIQUE(scope_key, version)`; partial unique index on `scope_key WHERE status='approved'`;
- same immutability trigger as policy versions. No legal profile is seeded.

`fiscal_provider_connections`

- `id uuid PK`, `scope_key varchar(80)`, `provider_key varchar(40)`, `state disconnected|authorizing|connected|action_required|revoked`;
- `provider_organization_ref varchar(255) NULL`, `granted_scopes text[] NOT NULL DEFAULT '{}'`, `access_expires_at timestamptz NULL`;
- `credential_ciphertext bytea NULL`, `credential_nonce bytea NULL`, `credential_key_version varchar(40) NULL`, `credential_format_version integer NULL`;
- `last_verified_at`, `connected_at`, `revoked_at`, `created_by`, `updated_by`, timestamps, integer `version`;
- `UNIQUE(scope_key, provider_key)`; consistency check requires all encryption metadata together or all null; index `(provider_key,state)`.

`fiscal_oauth_authorizations`

- `id uuid PK`, `connection_id FK ... ON DELETE CASCADE`, `actor_id FK users ON DELETE RESTRICT`;
- `state_sha256 char(64) UNIQUE`, `pkce_ciphertext/nonce/key_version NULL`, `redirect_uri text`, `expires_at`, `consumed_at NULL`, `created_at`;
- index `(connection_id, expires_at) WHERE consumed_at IS NULL`;
- only state hash is stored; raw state and authorization code are never persisted. PKCE fields remain null or populated according to the approved gate decision.

`fiscal_enablement_gates`

- `id uuid PK`, `provider_key`, `environment`, `gate_key`, `status pending|satisfied|not_applicable`;
- `evidence_ref text NULL`, `decision_ref text NULL`, `acceptance_ref text NULL`, `rationale text NULL`, `approved_by`, `approved_at`, timestamps;
- `UNIQUE(provider_key,environment,gate_key)`; index `(provider_key,environment,status)`;
- migration inserts pending Cloudware gate keys named in the proposal, never satisfied values. Readiness code owns the mandatory gate catalog and permits `not_applicable` only for explicitly optional features such as unused webhooks.

### Fiscal aggregate tables

`fiscal_documents`

- `id uuid PK`, `source_invoice_id uuid NOT NULL REFERENCES invoices(id) ON DELETE RESTRICT`;
- `intent_slot varchar(40) NOT NULL DEFAULT 'primary_sale'`, `kind varchar(2) CHECK kind IN ('FT','FR')`, lifecycle `state` with the eleven specified values;
- `version bigint NOT NULL DEFAULT 1`, `supersedes_document_id uuid NULL REFERENCES fiscal_documents(id)`, `superseded_at timestamptz NULL`;
- `provider_key varchar(40) NULL`, `connection_id uuid NULL REFERENCES fiscal_provider_connections(id) ON DELETE RESTRICT`;
- `intent_key uuid NULL`, `issue_operation_key varchar(160) NULL`, `void_operation_key varchar(160) NULL`;
- `provider_reference varchar(255) NULL`, `provider_number varchar(255) NULL`, `provider_confirmed_at timestamptz NULL`, `issued_at timestamptz NULL`, `voided_at timestamptz NULL`;
- safe `last_error_class`, `last_error_code`, `last_error_message`, timestamps and creator/finalizer UUIDs;
- partial unique index `(source_invoice_id,intent_slot) WHERE superseded_at IS NULL`;
- unique indexes on non-null `intent_key`, `issue_operation_key`, `void_operation_key`, and `(provider_key,provider_reference)`;
- indexes `(source_invoice_id,created_at DESC)`, `(state,updated_at)`, `(connection_id,state)`;
- checks enforce provider/frozen identity presence outside draft, void fields only on void states, and `superseded_at` only for `rejected` documents.

`fiscal_snapshots`

- `id uuid PK`, `fiscal_document_id uuid UNIQUE REFERENCES fiscal_documents(id) ON DELETE RESTRICT`;
- `schema_version integer`, `policy_version_id uuid NULL REFERENCES fiscal_policy_versions(id) ON DELETE RESTRICT`, `issuer_profile_id uuid NULL REFERENCES fiscal_issuer_profiles(id) ON DELETE RESTRICT`;
- `issuer jsonb`, `customer jsonb`, `billing_address jsonb`, `currency char(3)`;
- `gross_total`, `discount_total`, `net_total`, `tax_total`, `rounding_adjustment`, `payable_total NUMERIC(38,18)`;
- `canonical_bytes bytea NULL`, `canonical_sha256 char(64) NULL`, `frozen_at NULL`, timestamps;
- JSON object checks and non-negative total checks where universally valid; policy-specific limits stay in the calculator;
- immutable trigger rejects UPDATE/DELETE when the parent is not `draft` or `frozen_at` is set.

`fiscal_document_lines`

- `id uuid PK`, `snapshot_id uuid REFERENCES fiscal_snapshots(id) ON DELETE RESTRICT`, `position integer > 0`, `description text`, `unit_code varchar(40)`;
- `quantity`, `unit_price`, `gross_amount`, `discount_value`, `discount_amount`, `net_amount`, `tax_rate`, `tax_amount`, `line_total NUMERIC(38,18)`;
- `discount_kind varchar(16) CHECK none|amount|percent`, `tax_treatment_code varchar(80)`, `exemption_code varchar(80) NULL`, `exemption_reason text NULL`;
- `source_type varchar(16) NULL CHECK repair|part`, `source_id uuid NULL`, timestamps;
- `UNIQUE(snapshot_id,position)`; index `(source_type,source_id)` where source ID is present;
- pair checks require both source columns or neither, quantity > 0, unit price >= 0, discount >= 0, and the same parent-state immutability trigger. No foreign key is placed on the polymorphic source, allowing source deletion without deleting traceability.

`fiscal_state_transitions`

- `id bigserial PK`, `fiscal_document_id FK ON DELETE RESTRICT`, `from_state`, `to_state`, `operation`, `reason_class`, `reason_code`, safe `reason_detail`;
- `actor_type CHECK user|worker|system`, `actor_id uuid NULL`, `correlation_key varchar(160)`, `created_at`;
- index `(fiscal_document_id,id)`; UPDATE/DELETE are denied by trigger/role permissions.

A `BEFORE UPDATE OF state` trigger on `fiscal_documents` enforces the transition table above. Service transactions set a transaction-local audit context used by an `AFTER UPDATE` trigger, or explicitly insert the transition and call one repository transition function; startup schema verification must confirm whichever single mechanism is implemented. Direct un-audited state updates are not permitted.

### Durable work and evidence tables

`fiscal_outbox_events`

- `id uuid PK`, `fiscal_document_id uuid NOT NULL FK ON DELETE RESTRICT`, optional `artifact_id uuid` added after artifact table creation;
- `event_type CHECK issue|reconcile_issue|void|reconcile_void|recover_artifact`, `operation_key varchar(160)`, `sequence_no integer > 0`;
- `status CHECK ready|leased|completed|dead`, `available_at`, `lease_owner`, `lease_token uuid`, `lease_expires_at`, `claim_count integer`, `last_safe_error`, timestamps;
- payload contains only aggregate/artifact IDs; no snapshot, PII, token, URL, or provider body;
- `UNIQUE(fiscal_document_id,event_type,sequence_no)` and partial unique index on `fiscal_document_id WHERE status IN ('ready','leased')`;
- claim index `(status,available_at,lease_expires_at,created_at)`.

`fiscal_provider_attempts`

- `id uuid PK`, `fiscal_document_id FK`, `outbox_event_id FK`, `provider_key`, `operation CHECK issue|reconcile_issue|void|reconcile_void|fetch_artifact`;
- `operation_key`, `attempt_no`, `status CHECK started|succeeded|failed|unknown`;
- `classification NULL CHECK validation|authorization|transient|rate_limit|permanent|ambiguous`, `definitive boolean`;
- request digest, safe provider code/message, allowlisted redacted diagnostics JSONB, provider reference, `started_at`, `finished_at`;
- `UNIQUE(fiscal_document_id,operation,attempt_no)`; indexes on `(fiscal_document_id,started_at)` and `(outbox_event_id)`;
- append-only after completion. Database permissions and application redaction reject secret-shaped diagnostic keys.

`fiscal_artifacts`

- `id uuid PK`, `fiscal_document_id FK`, denormalized `source_invoice_id FK`, `kind CHECK provider_pdf`, `status CHECK pending|available|unavailable|compromised`;
- `classification CHECK legal|mock`, private `storage_key text UNIQUE`, `media_type`, `byte_size bigint`, `sha256 char(64)`, `provider_reference`, `provider_version`, `created_at`, `available_at`, `last_verified_at`, safe failure fields;
- `UNIQUE(fiscal_document_id,kind)`; indexes `(source_invoice_id,status)` and `(fiscal_document_id,status)`;
- available-row check requires `application/pdf`, positive size, checksum, and storage key. Metadata is not deleted when issuance is disabled.

`fiscal_artifact_access_log`

- `id bigserial PK`, artifact/document/invoice IDs, actor ID and role, outcome `allowed|denied|unavailable`, request correlation ID, IP hash (not raw IP unless an accepted security policy requires it), timestamp;
- index `(artifact_id,created_at DESC)` and retention managed separately from legal artifacts.

`fiscal_mock_operations`

- development/test persistence for deterministic mock operation key, scenario, canonical request digest, stable provider reference, result JSON, PDF SHA-256, void state, timestamps;
- `operation_key` primary key and unique provider reference;
- repository refuses all access when `APP_ENV=production`. The table may exist in production but must remain empty.

### Database enforcement and startup verification

`backend/internal/repository/postgres/fiscal_schema.go` verifies all fiscal tables, required columns, unique/partial indexes, triggers, and migration version before fiscal routes or workers are enabled. Unlike the current best-effort `AutoMigrate` loop, missing fiscal schema is fatal for the fiscal capability. Existing unrelated AutoMigrate behavior can remain temporarily, but fiscal tables are never AutoMigrated.

Production packaging must build a migration command from `backend/cmd/migrate` (refactored from `backend/scripts/run_migrations.go`) and run it as a one-shot Compose service before API/worker startup. API startup never silently creates or repairs fiscal schema.

## Transaction and outbox behavior

### Finalization transaction

`FiscalRepository.FinalizeDraft` is one repository/unit-of-work operation, not a sequence of independently committed repositories:

1. `SELECT ... FOR UPDATE` the source invoice, current fiscal document, snapshot, lines, selected connection, policy, and issuer profile.
2. Verify role in the service and recheck invoice eligibility, draft state/version, current intent uniqueness, approved policy/profile, connection state, provider capability, and environment readiness.
3. Recalculate from exact persisted values. Compare any declared totals and canonicalize the frozen DTO.
4. Set policy/profile references, `frozen_at`, canonical bytes/hash, provider/connection, stable UUID intent key, issue operation key `fiscal:issue:<intent-key>`, state `pending`, and finalizer.
5. Insert the transition and exactly one ready `issue` event, sequence 1.
6. Commit. Any error rolls back all steps.

Concurrent finalization is resolved by row lock plus unique indexes. A repeat against the same now-frozen current draft returns its existing projection (202 while pending) rather than inserting work; a semantically different/superseded request returns 409.

### Claim and call protocol

Claim SQL uses a short transaction and a common-table expression around `FOR UPDATE SKIP LOCKED`. It selects `ready` events whose `available_at <= now()` or expired `leased` events, sets a random lease token/owner/expiry, and returns the row. Completion updates require matching event ID, token, owner, status `leased`, and unexpired lease; zero updated rows means the worker lost ownership and must not commit a result.

For a fresh legal call, the worker transaction also:

- validates event type against document state;
- changes `pending` to `dispatching` for issue (void remains `void_pending`);
- inserts a `started` attempt before any network call;
- commits before invoking the gateway.

If the process dies after that commit, lease recovery sees an unfinished started attempt. It does **not** call Issue/Void again. It atomically marks that attempt `unknown`, moves the document to the corresponding unknown state, completes the stale event, and waits for an explicit reconciliation request.

After a gateway call, one transaction checks the lease token, completes the attempt, applies the guarded state transition, stores safe provider facts, and completes the event. Definite transient/rate-limit results become `retryable_failure`; there is no automatic legal retry in this release. A privileged retry inserts a new sequence event but reuses the original operation key, provider, connection, and frozen bytes.

Reconciliation events leave the document in its unknown state while leased. They call only `Reconcile` using evidenced keys. Inconclusive responses complete the attempt/event but retain unknown state. Another manager/admin request can create the next reconciliation sequence. The service cannot enqueue issue from an unknown state.

## Provider contracts

Place provider-neutral contracts in `backend/internal/core/ports/fiscal_provider.go`. Representative shapes are:

```go
type FiscalProviderGateway interface {
    Key() string
    Capabilities(ctx context.Context, connection ProviderConnectionView) (ProviderCapabilities, error)
    VerifyConnection(ctx context.Context, connection ProviderConnectionView) (ConnectionVerification, error)
    Issue(ctx context.Context, op ProviderOperation, frozen FrozenFiscalDocument) (IssueResult, error)
    Reconcile(ctx context.Context, query ReconcileQuery) (ReconcileResult, error)
    Void(ctx context.Context, op ProviderOperation, issued IssuedFiscalDocument) (VoidResult, error)
    FetchArtifact(ctx context.Context, ref ArtifactReference) (ArtifactStream, error)
}
```

`ProviderCapabilities` explicitly states supported kinds, issue/finalize, reconciliation lookup modes, native idempotency evidence, void, PDF retrieval, associated FR receipt behavior, and evidence revision. A false or unknown capability blocks the operation; absence never means supported.

`ProviderOperation` carries internal correlation/operation keys, attempt ID, and deadline. It contains no credentials; the adapter resolves/decrypts the named connection internally. `FrozenFiscalDocument` comes from canonical bytes and contains no mutable domain pointers.

`ProviderError` has:

- class: `validation`, `authorization`, `transient`, `rate_limit`, `permanent`, or `ambiguous`;
- `DefinitiveNonAcceptance` boolean;
- safe code/message and field errors;
- optional retry-after only when provider evidence supports it;
- redacted diagnostics.

Transport timeout or lost response after call start defaults to ambiguous. Only an explicit evidenced response can set definitive non-acceptance. Unknown status/body mappings default to ambiguous for mutations and permanent/unavailable for non-mutating reads.

### Deterministic mock

`backend/internal/integration/fiscal/mock` implements the same interface. Scenarios are selected through an injected non-production `MockScenarioRegistry` keyed by operation key—not magic NIFs, descriptions, or legal fields. The adapter persists the first normalized result in `fiscal_mock_operations` and returns it for repeats.

- provider reference: stable SHA-256-derived `MOCK-...` value;
- PDF: deterministic bytes from a fixed renderer/template, canonical snapshot hash, and large visible `SEM VALIDADE FISCAL — MOCK` label;
- success covers FT and FR and records FR associated-receipt capability without creating a local standalone receipt;
- validation, expired connection, definite transient, rate-limit, ambiguous-then-reconcile, permitted/refused void, and concurrent repeat scenarios are explicit fixtures;
- startup fails if mock is selected in production or if a mock artifact could be classified `legal`.

### Cloudware adapter

`backend/internal/integration/fiscal/cloudware` owns Cloudware HTTP DTOs, OAuth/token refresh, FT/FR mapping, response parsing, and error translation. Generic domain/handler packages never import it.

The adapter maps only research-evidenced v1 concepts. Production mutation methods first call `CloudwareEnablementEvaluator`; if any applicable gate, issuer/series mapping, policy, credential, storage, or acceptance marker is incomplete, they return a local fail-closed error before sending HTTP. No webhook route is added. No sandbox, provider idempotency, lookup, retry, URL lifetime, or AT communication claim exists until its gate is updated with evidence and tests.

## OAuth and credential security

- `POST /api/v1/fiscal-integrations/cloudware/connect` is manager/admin-only and creates or reuses the scoped connection in `authorizing` state plus a 128-bit random state value. The database stores only its SHA-256, actor, redirect URI, and expiry.
- The raw state is returned only as part of the short-lived authorization URL. The callback route may be unauthenticated because the current frontend uses localStorage rather than an auth cookie; security rests on unguessable one-time state bound to the initiating actor and exact redirect URI. It consumes state atomically before code exchange.
- PKCE is used only according to the recorded gate decision; production stays disabled while support/non-use is unresolved. OAuth state protection itself is mandatory regardless of provider documentation.
- Authorization codes are held in memory for one exchange and never logged or stored. Client ID/secret come from a server secret provider, not frontend configuration.
- Access and refresh tokens are encoded in a versioned credential envelope and encrypted with AES-256-GCM. AAD includes connection ID, provider key, scope key, and format version. A dedicated keyring exposes an active key version and old decrypt-only versions for rotation. JWT secrets are never reused.
- Rotation re-encrypts rows under optimistic locking and audit, without exposing plaintext outside the credential component. Missing production keyring or default secrets makes startup fail.
- API responses include only connection ID, provider-neutral state, organization display/reference if approved, scopes, expiry, verification time, readiness, and safe guidance.
- Disconnect removes encrypted active credentials only after an approved revocation/local-disconnect decision, sets state `revoked`/`disconnected`, and preserves audit. Frozen documents retain connection ID and provider identity.

## Private PDF artifact flow

Introduce a dedicated `FiscalArtifactStore`, not the current metadata-only `FileStorage`:

```go
type FiscalArtifactStore interface {
    PutImmutable(ctx context.Context, key string, mediaType string, body io.Reader) (StoredArtifact, error)
    Open(ctx context.Context, key string) (io.ReadCloser, StoredArtifact, error)
    Stat(ctx context.Context, key string) (StoredArtifact, error)
}
```

Keys are deterministic and non-public: `fiscal/<environment>/<document-id>/provider.pdf`. `PutImmutable` uses create-if-absent semantics; an existing object is accepted only if checksum/size match. A local private-filesystem adapter is allowed for development/mock. Production requires an immutable/versioned private object backend, encryption at rest, no public ACL, backup/restore evidence, retention configuration, and access logging before readiness passes.

Provider confirmation is recorded as `issued` regardless of artifact success. The worker attempts immediate archive using returned bytes or `FetchArtifact(provider reference)`. It computes SHA-256 while streaming. Success stores available metadata; failure stores `unavailable` and enqueues `recover_artifact` without changing document state. Recovery uses only the provider reference and never calls `Issue`.

Before each download, `FiscalArtifactService` loads the persisted artifact, document, and source invoice; applies current user role/ownership; opens private bytes; optionally verifies size/checksum according to configured verification policy; and writes the access log. Checksum mismatch atomically marks `compromised` and returns no bytes. Storage keys, signed URLs, tokens, and provider URLs never enter the response.

## HTTP contracts

All routes are additive under `/api/v1`; existing invoice handlers and response fields keep their current meaning.

### Fiscal document routes

| Method and path | Roles | Result |
|---|---|---|
| `GET /invoices/:invoiceId/fiscalization` | staff; owning client | Provider-neutral projection. Client projection hides draft details and privileged actions. |
| `PUT /invoices/:invoiceId/fiscalization/draft` | employee/manager/admin | Create or replace editable draft; 201/200. Requires eligible invoice. |
| `DELETE /invoices/:invoiceId/fiscalization/draft` | employee/manager/admin | Delete only a current draft; 204. |
| `POST /invoices/:invoiceId/fiscalization/finalize` | manager/admin | Freeze and queue; 202 with current projection. |
| `POST /invoices/:invoiceId/fiscalization/retry` | manager/admin | Queue eligible same-operation retry; 202. |
| `POST /invoices/:invoiceId/fiscalization/reconcile` | manager/admin | Queue same-provider lookup; 202. |
| `POST /invoices/:invoiceId/fiscalization/void` | manager/admin | Queue permitted void; 202. |
| `GET /invoices/fiscalization-summaries?invoiceIds=<csv>` | authenticated | Maximum 100 IDs; staff can query workshop invoices, clients only an all-owned set. Used by existing list pages without changing invoice envelopes. |
| `GET /invoices/:invoiceId/fiscal-artifacts/:artifactId` | staff; owning client | Streams archived PDF with safe `Content-Disposition`; never redirects to storage/provider. |

Static `/invoices/fiscalization-summaries` is registered before `/:id`. Service authorization is repeated even where middleware exists. A non-owning client receives 404 for detail/artifact resources to avoid enumeration. Role failures are 403; missing resources 404; version/state conflicts 409; policy/readiness/field failures 422 with stable field codes; accepted actions 202; temporarily unavailable artifacts 503 with provider-neutral guidance.

Draft request decimals are strings:

```json
{
  "version": 3,
  "kind": "FT",
  "issuerProfileId": "uuid",
  "policyKey": "pt-sales",
  "currency": "EUR",
  "customer": { "legalName": "...", "taxIdentifier": "...", "countryCode": "PT" },
  "billingAddress": { "line1": "...", "postalCode": "...", "city": "...", "countryCode": "PT" },
  "lines": [{
    "position": 1,
    "description": "...",
    "quantity": "1.000",
    "unitCode": "...",
    "unitPrice": "10.00",
    "discount": { "kind": "none", "value": "0" },
    "taxTreatmentCode": "<approved-policy-code>",
    "taxRate": "<approved-policy-rate>",
    "exemptionCode": null,
    "source": { "type": "repair", "id": "uuid" }
  }],
  "declaredTotals": { "payableTotal": "<optional comparison>" }
}
```

Values shown are structural examples, not approved policy constants. The server resolves the requested approved policy and returns calculated decimal strings and readiness errors. Finalize body contains `expectedVersion` and optional selected provider key; the server never trusts client totals.

`FiscalizationProjection` includes document ID, kind, lifecycle, simplified presentation status, version, calculated/frozen totals as strings, readiness issues, safe last error, artifact status, and server-derived allowed actions. It excludes credentials, raw provider payloads, storage keys, provider URLs, and diagnostic bodies.

Presentation mapping is stable and provider-neutral:

- no fiscal row + legacy invoice: `legacy_unfiscalized`;
- staff-visible draft: `draft` (clients receive `unavailable` with no draft fields);
- pending/dispatching/void-pending: `pending`;
- issued: `finalized` even when artifact is temporarily unavailable;
- voided: `voided`;
- rejected, connection-action-required, retryable, or unknown: `unavailable` plus role-appropriate guidance.

### Integration routes

- `GET /api/v1/fiscal-integrations/cloudware/connection`
- `POST /api/v1/fiscal-integrations/cloudware/connect`
- `GET /api/v1/fiscal-integrations/cloudware/oauth/callback` (state-protected callback)
- `POST /api/v1/fiscal-integrations/cloudware/verify`
- `POST /api/v1/fiscal-integrations/cloudware/disconnect`
- `GET /api/v1/fiscal-integrations/cloudware/readiness`

All except the callback require manager/admin middleware and service checks. Readiness returns gate names/status and safe missing-action guidance, not secret evidence content unless separately authorized.

## Required sequence diagrams

### Finalization and issuance

```mermaid
sequenceDiagram
    actor M as Manager/Admin
    participant UI as Next.js UI
    participant API as Fiscal Handler/Service
    participant DB as PostgreSQL
    participant W as Fiscal Worker
    participant G as Provider Gateway
    participant S as Private Artifact Store

    M->>UI: Finalize explicit draft
    UI->>API: POST finalize(expectedVersion)
    API->>DB: BEGIN; lock invoice/draft/config
    API->>API: exact calculation + readiness + canonicalize
    API->>DB: freeze snapshot/lines, state=pending,
insert transition + issue outbox
    DB-->>API: COMMIT
    API-->>UI: 202 pending + allowedActions
    W->>DB: claim issue lease (SKIP LOCKED)
    W->>DB: state=dispatching + started attempt; COMMIT
    W->>G: Issue(same operation key, canonical bytes)
    G-->>W: definite issued + provider reference
    W->>S: PutImmutable deterministic PDF key
    alt archive succeeds
        S-->>W: checksum and size
        W->>DB: issued + attempt/event complete + artifact available
    else archive fails
        W->>DB: issued + attempt/event complete + artifact unavailable + recovery event
    end
    UI->>API: GET fiscalization
    API-->>UI: finalized; PDF available or recovering
```

### Ambiguous timeout and reconciliation

```mermaid
sequenceDiagram
    participant W as Fiscal Worker
    participant DB as PostgreSQL
    participant G as Same Provider
    actor M as Manager/Admin
    participant API as Fiscal API

    W->>DB: lease event; insert started attempt; state=dispatching
    W->>G: Issue(operation key)
    G--xW: timeout/lost response
    W->>DB: attempt=unknown; state=outcome_unknown; complete issue event
    Note over DB: No issue retry can be enqueued
    M->>API: POST reconcile
    API->>DB: authorize state; enqueue reconcile_issue
    W->>DB: claim reconciliation lease
    W->>G: Reconcile(evidenced lookup, same provider/key)
    alt exact document found
        G-->>W: issued + provider reference
        W->>DB: state=issued; complete attempt/event
    else definitely no document
        G-->>W: absent under approved lookup
        W->>DB: state=retryable_failure; complete attempt/event
        Note over M,API: A later explicit retry reuses original issue key
    else inconclusive or auth unavailable
        G-->>W: inconclusive
        W->>DB: retain outcome_unknown; complete attempt/event
    end
```

### PDF recovery without reissuance

```mermaid
sequenceDiagram
    participant W as Worker
    participant DB as PostgreSQL
    participant G as Provider Gateway
    participant S as Private Store
    actor U as Authorized Staff/Owner
    participant API as Artifact API

    W->>DB: load issued document + provider reference
    W->>G: FetchArtifact(reference)
    G--xW: temporary retrieval failure
    W->>DB: artifact=unavailable; enqueue bounded recover_artifact
    Note over W,G: Issue is never called
    W->>DB: later claim recovery lease
    W->>G: FetchArtifact(same reference)
    G-->>W: PDF stream
    W->>S: PutImmutable(deterministic key)
    S-->>W: size + SHA-256
    W->>DB: artifact=available
    U->>API: GET artifact
    API->>DB: authorize through source invoice ownership
    API->>S: Open(private key)
    S-->>API: PDF stream + metadata
    API-->>U: application/pdf; safe filename
```

### Provider connection

```mermaid
sequenceDiagram
    actor A as Manager/Admin
    participant UI as Integration Settings
    participant API as Connection Service
    participant DB as PostgreSQL
    participant C as Cloudware OAuth
    participant K as Credential Keyring

    A->>UI: Connect Cloudware
    UI->>API: POST connect (Bearer)
    API->>DB: store actor-bound state hash + expiry
    API-->>UI: short-lived authorization URL
    UI->>C: browser authorization
    C->>API: callback(code, raw state)
    API->>DB: atomically consume matching unexpired state hash
    alt invalid/missing/reused state
        API-->>UI: reject; store no credential
    else valid state and gates permit exchange
        API->>C: server-side code exchange
        C-->>API: tokens/scopes/expiry
        API->>K: AES-GCM encrypt credential envelope
        K-->>API: ciphertext + nonce + key version
        API->>DB: connected state + encrypted fields + audit
        API-->>UI: redirect to provider-neutral connected status
    end
```

## Package and file placement

### Backend

| Path | Responsibility |
|---|---|
| `internal/domain/fiscal_document.go` | Aggregate, states, transitions, guards, frozen DTO. |
| `internal/domain/fiscal_decimal.go` | Strict exact decimal wrapper and canonical string encoding. |
| `internal/domain/fiscal_policy.go` | Policy model and pure calculator. |
| `internal/domain/fiscal_connection.go` | Provider-neutral connection/capability types. |
| `internal/domain/fiscal_artifact.go` | Artifact metadata/status. |
| `internal/core/ports/fiscal_repository.go` | Atomic aggregate/finalization/action/query contracts. |
| `internal/core/ports/fiscal_provider.go` | Gateway, capabilities, operation/results, normalized errors. |
| `internal/core/ports/fiscal_artifact_store.go` | Immutable private binary store. |
| `internal/core/ports/credential_cipher.go` | Versioned encrypt/decrypt contract. |
| `internal/service/fiscal/draft_service.go` | Draft validation, calculation, authorization. |
| `internal/service/fiscal/finalization_service.go` | Readiness and atomic finalization/retry/reconcile/void commands. |
| `internal/service/fiscal/worker.go` | Lease processing and crash/ambiguity protocol. |
| `internal/service/fiscal/artifact_service.go` | Recovery and ownership-authorized downloads. |
| `internal/service/fiscal/connection_service.go` | OAuth state, verification, refresh/reconnect/disconnect. |
| `internal/repository/postgres/fiscal_repository.go` | Aggregate/query transactions. |
| `internal/repository/postgres/fiscal_outbox_repository.go` | Claim/lease/complete SQL. |
| `internal/repository/postgres/fiscal_connection_repository.go` | Connection/gate/OAuth persistence. |
| `internal/repository/postgres/fiscal_artifact_repository.go` | Artifact and access-log persistence. |
| `internal/repository/postgres/fiscal_schema.go` | Fail-fast migration/index/trigger validation. |
| `internal/integration/fiscal/mock/*` | Deterministic non-production provider and persisted scenarios. |
| `internal/integration/fiscal/cloudware/*` | Cloudware-only DTOs, OAuth client, mapping, error translation, gate evaluator. |
| `internal/platform/crypto/fiscal_credentials.go` | AES-GCM keyring and rotation support. |
| `internal/platform/fiscalartifact/local.go` | Development private filesystem store. |
| `internal/platform/fiscalartifact/object.go` | Production private object-store adapter. |
| `internal/handler/fiscal_handler.go` | Nested document/action/summary/artifact DTOs. |
| `internal/handler/fiscal_integration_handler.go` | Connection and callback endpoints. |
| `cmd/fiscal-worker/main.go` | Worker composition and graceful lease shutdown. |
| `cmd/migrate/main.go` | Production migration executable derived from current script. |
| `cmd/api/main.go` | Wire fiscal services/routes only; do not run provider work inline. |
| `migrations/011_fiscal_integration_foundation.up.sql` | Additive schema, constraints, indexes, triggers, pending gates. |

Existing `invoice_service.go` gains only eligibility-on-create and delete protection through a narrow `FiscalProtectionReader`; current DTO fields and update semantics remain. Existing root `ports` style is followed rather than introducing a second architecture hierarchy.

### Frontend

| Path | Responsibility |
|---|---|
| `src/types/fiscal.ts` | Decimal-string DTOs, lifecycle, projection, connection/readiness types. |
| `src/lib/services/fiscalization.service.ts` | Draft/actions/summaries/artifact calls. |
| `src/lib/services/fiscal-integration.service.ts` | Manager/admin connection and readiness calls. |
| `src/app/accounting/issued-invoices/page.tsx` | Batch provider-neutral badges/readiness while preserving current invoice columns/actions. |
| `src/app/accounting/issued-invoices/[id]/page.tsx` | Fiscal panel beside existing operational edit form; frozen state does not disable notes/status/amount compatibility edits. |
| `src/app/accounting/issued-invoices/[id]/FiscalDraftForm.tsx` | Manual lines, exact string inputs, calculated totals, source links, readiness. |
| `src/app/accounting/issued-invoices/[id]/FiscalActions.tsx` | Server-derived finalize/retry/reconcile/void/PDF actions with double-submit protection. |
| `src/app/my-invoices/MyInvoicesListClient.tsx` | Simple pending/finalized/voided/unavailable badge. |
| `src/app/my-invoices/[id]/MyInvoiceDetailClient.tsx` | Provider-neutral status and authorized PDF; current notes form unchanged. |
| `src/app/admin/integrations/fiscal/page.tsx` | Manager/admin connection state, connect/reconnect/verify, and readiness. |
| `src/components/layout/AppShell.tsx` | Manager/admin-only integration-settings navigation. |

The UI treats a 202 response as durable progress, disables duplicate buttons while submitting, then polls the projection with capped client-side polling while the page is open. Server uniqueness remains authoritative. It never renders raw provider errors, tokens, storage locations, or claims about AT/e-Fatura absent an evidenced projection field.

## Observability and operations

### Structured logs

Every fiscal log carries request/worker correlation ID, fiscal document ID, outbox event ID, attempt ID, provider key, operation, old/new state, classification, lease owner, and duration. It excludes canonical snapshot content, names, NIFs, addresses, OAuth query strings/codes/state, tokens, authorization headers, provider URLs, PDF bytes, and ciphertext. A central allowlist redactor is applied before attempt diagnostics and logs.

### Metrics

- outbox ready age, leased count, expired leases, and dead events;
- operations by provider/type/result/classification;
- documents by lifecycle, especially unknown-state age;
- connection state and token-expiry horizon without token values;
- artifact unavailable/compromised count and recovery age;
- finalization conflicts/readiness failures;
- PDF authorization denials and checksum failures.

Alerts target oldest unknown outcome, repeated expired leases, action-required connection, artifact recovery beyond policy, checksum compromise, and production readiness regression. Provider failures appear in a fiscal dependency status endpoint/metric but do not make `/ready` fail while PostgreSQL and core API are healthy. Worker readiness may separately report DB/schema/config availability.

## Security controls

- Defense in depth: route middleware plus service-level role/ownership checks.
- Manager/admin-only legal and connection actions; employee draft-only; clients own-only projections/artifacts.
- Optimistic `version` and database locks prevent stale draft/finalization writes.
- Request body limits, strict JSON decoding, bounded line count from approved operational configuration, and canonical decimal parsing prevent resource abuse.
- OAuth state is random, one-time, actor/redirect-bound, expiring, and stored hashed.
- Credentials use dedicated versioned encryption keys and never enter outbox, frontend, or normal diagnostics.
- Provider HTTP clients enforce TLS, allowlisted base URLs, bounded response sizes, deadlines, and no automatic mutation retries at the transport layer.
- PDF retrieval enforces PDF media/type/signature checks, size limits, streaming checksum, private immutable storage, safe filenames, `nosniff`, and authenticated access.
- Database application roles should deny UPDATE/DELETE on transition/attempt history and frozen snapshot data except through narrowly permitted functions/repository transactions.
- Mock selection, mock storage classification, and mock PDF labeling fail closed in production.

## Strict-TDD verification plan

Implementation follows RED → GREEN → TRIANGULATE → REFACTOR for each slice. PostgreSQL-specific behavior must not be accepted from SQLite tests alone.

### Domain unit tests

- every allowed and forbidden lifecycle transition;
- role/action matrix and server-derived allowed actions;
- exact decimal parsing, normalization, arithmetic, mismatch reporting, and canonical hash stability;
- fail-closed missing policy, tax/exemption, rounding, currency, issuer, series, and void rules;
- snapshot immutability and replacement-after-rejection rules;
- provider error normalization defaults ambiguous when mutation acceptance is uncertain.

### PostgreSQL integration tests

Add a PostgreSQL service to CI or an equivalent isolated integration target and test:

- migration leaves all prior invoices `legacy_unfiscalized` and creates no fiscal/outbox rows;
- new create explicitly writes `eligible` while legacy response JSON is unchanged;
- partial unique current-intent and operation/provider-reference constraints;
- concurrent finalization yields one frozen intent/event;
- rollback when outbox insert fails;
- snapshot/line/config immutability triggers;
- `SKIP LOCKED` claims, one active lease, lease-token fencing, expiry recovery;
- crashed started attempt becomes unknown without a second gateway call;
- invoice deletion succeeds with no protected intent, handles draft as designed, and returns conflict for frozen history;
- append-only attempts/transitions and artifact metadata constraints.

### Service/provider tests

- employee draft success and privileged-action denial;
- client own-only projection/artifact and cross-client denial;
- retries reuse provider, connection, operation key, and canonical digest;
- unknown states reject issue/void retries until definitive reconciliation;
- deterministic mock repeats return identical reference/result/PDF, including concurrent calls and process-repository reload;
- all mock scenarios from the provider-foundation spec;
- issued state survives artifact failure and recovery never calls Issue;
- Cloudware gate evaluator blocks before HTTP when any gate is incomplete;
- OAuth invalid/expired/reused state, encryption round trip, key rotation, refresh failure, and redaction tests;
- contract fixtures for only evidenced FT/FR Cloudware mapping, with no live production call in ordinary CI.

### HTTP and frontend tests

- regression tests keep existing invoice routes, fields, paging, RFC 3339 timestamps, notes-only client patch, and role behavior;
- route ordering for summary versus `/:id`;
- 403/404/409/422/202 mappings and repeated action safety;
- artifact response headers and no URL/key leakage;
- staff list/detail draft/readiness/actions; employees cannot see privileged controls;
- client list/detail simplified status and PDF ownership behavior while notes remain editable;
- integration settings manager/admin-only;
- polling stops on stable state/unmount and double-click cannot produce duplicate API action;
- mock labels are visible and production configuration rejects mock.

Required commands remain:

```text
cd backend && go test ./... -count=1 -race -timeout=2m
cd backend && go vet ./...
cd frontend && pnpm test -- --passWithNoTests
cd frontend && pnpm lint
cd frontend && pnpm typecheck
cd frontend && pnpm build
```

CI must additionally exercise the numbered migration and PostgreSQL concurrency suite.

## Rollout, migration, and rollback

### Rollout stages

1. **Schema dark launch:** deploy migration runner and `011` with fiscal feature/worker off. Verify migration version, triggers, indexes, zero outbox rows, and all existing invoices legacy.
2. **Provider-neutral backend dark launch:** deploy domain/repositories/API reads with finalization off. Configure no production policy by default. Validate legacy regressions and deletion protection.
3. **Development/test mock vertical slice:** enable mock only outside production; run full issuance, ambiguity, reconciliation, void, and PDF tests.
4. **Production configuration:** provision dedicated credential keyring, approved issuer/policy, private object storage, retention/backup/restore/access logging, worker settings, and monitoring. Keep Cloudware mutations off.
5. **Cloudware controlled validation:** collect each gate's evidence and acceptance reference with credentials in an approved environment. Enable only the operations whose capabilities are evidenced.
6. **Explicit legal enablement:** enable finalization/worker for authorized users only after all readiness checks pass. Monitor unknown outcomes, lease expiry, connection state, and artifact recovery.

Configuration switches are separate: `FISCAL_FEATURE_ENABLED`, `FISCAL_FINALIZATION_ENABLED`, `FISCAL_WORKER_ENABLED`, selected provider, Cloudware mutation enablement, credential keyring, and artifact backend. Production refuses default secrets, local/mock artifact backends, mock provider, missing schema, or incomplete readiness.

### Rollback

- Turn off new finalization and stop new worker claims; let already leased work finish or expire into the conservative recovery path.
- Keep read projections and artifact downloads available.
- Never delete or rewrite frozen documents, attempts, transitions, provider references, connections' audit metadata, or archived artifacts.
- A software rollback does not void a legal document. Use the authorized fiscal void/correction workflow.
- Before any fiscal draft/credential/artifact/attempt exists, a reviewed down migration may remove additive tables/column. After any real use, use forward-fix migrations only; `fiscal_eligibility` and all evidence tables remain.
- If the UI/API fiscal routes are rolled back, foreign keys and deletion checks must still prevent old binaries from physically deleting invoices with retained fiscal records. A database trigger should guard invoice deletion for non-draft fiscal rows independently of application code.

## Delivery and review-budget risk

The implementation spans migrations, domain arithmetic, repositories, worker concurrency, crypto/OAuth, provider adapters, artifact storage, HTTP contracts, two frontend experiences, settings UI, deployment, and PostgreSQL CI. It will materially exceed the confirmed 600 changed-line review budget as one implementation diff.

The session labels delivery as `single-pr` planning only and authorizes no commits or PRs. It does **not** constitute acceptance of `size:exception`. Before apply, the orchestrator must pause under the configured risk policy and obtain either an explicit `size:exception` acceptance or permission to change delivery strategy. This design does not invent a chain strategy.

## Design acceptance checklist

- [ ] Exact lifecycle and transition guards are implemented in domain and PostgreSQL protection.
- [ ] No production fiscal constant is inferred for precision, rounding, IVA, exemptions, issuer, or series.
- [ ] Finalization and initial outbox creation are one transaction.
- [ ] A crashed started mutation becomes unknown and cannot be blindly replayed.
- [ ] Gateway capabilities/errors are provider-neutral; Cloudware types remain adapter-local.
- [ ] OAuth secrets remain encrypted server-side and production gates fail closed.
- [ ] Issued state is independent of artifact availability; PDF recovery never reissues.
- [ ] Downloads reauthorize through current invoice ownership and private storage.
- [ ] Existing invoice HTTP/UI contracts remain operational and fiscal projections are additive.
- [ ] PostgreSQL concurrency, migration, immutability, role, compatibility, and deterministic-mock tests pass before enablement.
