# Proposal: Cloudware fiscal integration foundation

## Intent

Enable GonsGarage to prepare and explicitly finalize Portuguese fiscal sales documents while keeping the existing customer-invoice workflow compatible and keeping Cloudware operational details out of normal staff workflows.

The first product slice establishes a provider-neutral fiscal aggregate, immutable fiscal snapshots and lines, durable asynchronous dispatch, authorized PDF archiving, manager/admin controls, and a deterministic non-production mock provider. It prepares Cloudware as the first real provider, but production Cloudware issuance remains disabled until the provider unknowns listed under **Enablement gates** are resolved with credentialed evidence or Cloudware confirmation.

This change is needed because the current mutable `Invoice` record contains only a customer, binary floating-point amount, free-form status, notes, and timestamps. It cannot safely represent line-level tax data, legal identity snapshots, provider state, durable retries, or an immutable fiscal document.

## Product outcome

After this change:

1. An employee can prepare an FT or FR draft with manually entered fiscal lines and optional links to internal repair or part records.
2. A manager or admin can explicitly request finalization from GonsGarage.
3. Finalization freezes a provider-neutral fiscal snapshot and its lines, records the selected provider, and starts durable asynchronous processing without changing the meaning of the existing operational invoice status.
4. A manager or admin can retry eligible technical failures, reconcile ambiguous outcomes, or request a legally valid void when the document state and provider permit it.
5. Staff and the owning client can inspect provider-neutral fiscal status and download the archived PDF through authorized GonsGarage endpoints.
6. Normal operators do not need Cloudware accounts or Cloudware UI. A manager/admin may use the provider-hosted OAuth consent flow only while connecting or reconnecting the integration.
7. Existing invoices remain `legacy_unfiscalized` and are never queued, migrated, or selected for fiscalization automatically.

## Proposal question round

The pre-proposal phase served as the product question round and resolved the decisions that would otherwise make this proposal ambiguous:

| Product question | Confirmed decision |
|---|---|
| What initiates legal finalization? | An explicit staff action; no status-driven or background auto-finalization. |
| Which documents and source data belong in the first release? | FT and FR only; lines are entered manually and may optionally reference repairs or parts. |
| Who controls each lifecycle step? | Employees may prepare drafts. Only managers and admins may finalize, retry, reconcile, void, or manage the provider connection. |
| When does fiscal data become immutable? | The fiscal snapshot and lines freeze when finalization is requested. |
| How are legacy data, provider visibility, PDFs, and mocks handled? | Legacy invoices remain unfiscalized; provider details stay transparent to operators; immutable private PDFs are archived; mock output is development/test-only and never legal output. |

These decisions are treated as authoritative for the proposal. No unresolved provider behavior is silently converted into a product requirement. Corrections or a second product-question round can still be requested before the specification phase.

## Scope

### In scope

#### Fiscal drafts and immutable finalization

- Add a fiscal-document aggregate linked to, but distinct from, the existing operational `Invoice`.
- Support only document kinds **FT** and **FR** in the first release.
- Capture provider-neutral issuer, customer, billing address, currency, line, quantity, unit-price, tax, exemption, discount, rounding, total, document-kind, and source-reference data required by the supported workflow.
- Represent canonical fiscal money using decimal or integer minor-unit values; retain `float64` only at the legacy API compatibility boundary.
- Allow manual line creation with optional references to repair and part records. Those references provide traceability but do not make later changes to a repair or part mutate the captured fiscal line.
- Freeze the complete fiscal snapshot and all lines atomically when a manager/admin requests finalization.
- Preserve subsequent legal changes as explicit state transitions or related records. Do not rewrite or physically delete an issued fiscal record.

#### Authorization and lifecycle

- Preserve existing issued-invoice access: employees, managers, and admins retain operational invoice CRUD, while clients retain own-only list/detail access and notes editing.
- Allow employees to prepare fiscal drafts but deny employee access to finalization, retry, reconciliation, voiding, and connection management.
- Allow managers and admins to finalize, retry eligible technical failures, reconcile ambiguous outcomes, request a permitted void, and manage the Cloudware connection.
- Keep fiscal lifecycle state separate from the existing free-form `Invoice.Status`.
- Prevent automatic cross-provider failover. The provider selected at finalization remains fixed for that document and all of its retries and reconciliation.
- Prevent duplicate active fiscalization intents for the same source invoice and document intent.

#### Durable provider boundary

- Introduce a provider-neutral fiscal gateway with normalized operations for connection verification, issuance, reconciliation, voiding, and artifact retrieval.
- Persist the fiscal document, immutable snapshot, and initial outbox event in one PostgreSQL transaction.
- Process provider work through a lease-based transactional outbox that remains safe across crashes and concurrent workers.
- Use stable internal correlation/idempotency keys, persist redacted attempts, and classify validation, authorization, transient, rate-limit, permanent, and ambiguous-outcome failures.
- Reconcile an ambiguous result with the same provider before any resubmission; never blindly create another legal document after a timeout.
- Keep provider outages from making the ordinary invoice API or core application unavailable.

#### Mock provider foundation

- Provide a deterministic provider implementation for development and automated tests.
- Exercise successful FT/FR issuance, stable repeated calls, validation rejection, expired connection, transient failure, ambiguous outcome followed by reconciliation, concurrency, void behavior, and deterministic PDF generation.
- Enforce environment safeguards and visible labeling so mock documents cannot be presented or downloaded as legal fiscal documents in production.
- Model normalized application behavior rather than imitating undocumented Cloudware payloads.

#### Cloudware connection and adapter preparation

- Add instance/workshop-scoped provider connection records with a stable internal connection ID, provider key, connection state, provider organization reference, granted scopes, expiry, audit metadata, and encrypted server-side credentials.
- Support the documented Cloudware authorization-code and refresh-token concepts behind manager/admin-only setup. OAuth credentials and tokens must never enter frontend storage, API responses, outbox payloads, or unredacted logs.
- Isolate Cloudware-specific requests, responses, document mapping, and error translation inside its adapter.
- Map only documented v1 concepts for FT/FR, customer identity, series, lines, IVA/exemption, external reference, finalization, voiding, and requested PDF materialization.
- Keep real Cloudware dispatch disabled until every applicable enablement gate has recorded evidence and an approved behavior.

#### Private fiscal artifacts

- Archive an immutable private copy of the provider PDF as part of successful issuance processing.
- Persist artifact metadata including document ownership, storage key, media type, size, checksum, provider reference/version, and creation time.
- Serve PDFs only through authenticated GonsGarage endpoints. Staff access follows invoice staff policy; clients may access only artifacts linked to their own invoices.
- Avoid exposing provider bearer URLs, access credentials, or public storage paths.
- Include artifact storage, retention, backup, and access logging in production readiness.

#### Additive API and UI

- Add nested fiscal endpoints and projections rather than renaming or repurposing existing `/api/v1/invoices` contracts.
- Extend `/accounting/issued-invoices` with fiscal readiness, provider-neutral status, explicit finalization, retry, reconcile, void, and PDF actions when authorized and valid for the current state.
- Extend `/my-invoices` with a simple provider-neutral pending/finalized/voided/unavailable presentation and authorized PDF download.
- Add a separate manager/admin integration-settings surface for connection status and OAuth setup.
- Show durable progress after finalization and make repeated user actions safe against duplicate submission.

### Out of scope

- Automatic fiscalization based on `Invoice.Status`, timestamps, migrations, repair completion, payment, or background scans.
- Automatic selection or conversion of historical invoices.
- Fiscal document types other than FT and FR, including FS, credit notes, debit notes, standalone receipts, and partial-receipt workflows.
- Automated correction documents or hidden reissuance after rejection or voiding.
- Deriving fiscal lines automatically from repairs, parts, inventory, or service jobs.
- Automatic cross-provider failover or changing provider during retry/reconciliation.
- Multi-workshop or multi-tenant provider-account routing.
- Making the generic existing file abstraction responsible for fiscal PDFs without an explicit secure artifact contract.
- Claiming that Cloudware v1 automatically communicates with AT/e-Fatura without provider confirmation.
- Treating a Cloudware product trial as API sandbox entitlement.
- Representing mock-provider output as legally issued, submitted, certified, or fiscally valid.
- Changing existing invoice response fields, pagination, role semantics, client notes behavior, or legacy API routes.

## Business rules and invariants

1. **Explicit intent:** only an authenticated manager/admin action can turn an eligible fiscal draft into a finalization request.
2. **Immutable legal input:** snapshot and line data cannot change after finalization is requested, regardless of later changes to customer, issuer, repair, part, or operational invoice records.
3. **Legacy isolation:** every pre-existing invoice starts and remains `legacy_unfiscalized` unless a future explicit, separately approved workflow is introduced. This change provides no such workflow.
4. **One intent, one provider:** a finalized intent has one stable internal key and one selected provider. Retry and reconciliation reuse both.
5. **No blind retry:** ambiguous outcomes enter an unknown/reconciliation state rather than automatically issuing again.
6. **No hidden mutation:** rejection, issuance, reconciliation, and voiding are auditable transitions. Issued records and archived artifacts are retained.
7. **Role separation:** employees may prepare; managers/admins control legal and integration actions; clients remain own-only readers of fiscal status/artifacts.
8. **Provider transparency:** user-facing states and errors explain required action without exposing credentials or making normal operators use Cloudware.
9. **Private evidence:** archived PDFs are immutable, checksummed, non-public, and authorized through the source invoice relationship.
10. **Mock isolation:** mock mode is unavailable for legal production issuance and is clearly distinguishable in data, UI, and artifacts.

## Enablement gates

The provider-neutral foundation and mock flow may be completed while these items remain unknown. **Production Cloudware issuance must remain disabled** until each applicable gate has evidence, an explicit implementation decision, and acceptance coverage.

| Gate | Evidence required before production enablement |
|---|---|
| Sandbox/test company | Confirm how non-production credentials and a test company are provisioned, or define an approved controlled validation process if no sandbox exists. |
| OAuth security details | Verify redirect registration, OAuth state handling, PKCE support/requirement, revocation, expiry behavior, scopes, and reconnect behavior with real credentials or Cloudware support. |
| Idempotency and lookup | Establish whether Cloudware accepts an idempotency/correlation key and which lookup can safely resolve an ambiguous issuance outcome; approve a no-duplicate strategy based on observed behavior. |
| Webhooks | Confirm availability, signatures, ordering, and retries before using webhooks. Absence of webhook evidence must not be replaced by an invented contract. |
| Rate limits and retry guidance | Obtain limits and provider retry guidance, then configure bounded backoff and operator messaging from evidence rather than guesses. |
| PDF lifetime and authority | Verify PDF URL authentication, lifetime, authenticity, and retrieval behavior so the immutable archive can be captured reliably. |
| Automatic AT/e-Fatura communication | Confirm whether v1 finalization communicates automatically and what evidence/status proves that outcome. The documented v0 AT endpoint does not prove v1 behavior. |
| Credentialed response behavior | Validate real response bodies, status transitions, error taxonomy, FT/FR behavior, void behavior, and active-GC-license requirements before enabling live mutations. |

## Compatibility contract

The following existing behavior remains unchanged:

- `GET /api/v1/invoices` remains the staff paginated collection.
- `POST /api/v1/invoices` retains its current request and response contract.
- `GET /api/v1/invoices/me` remains the client's own-only paginated collection.
- `GET /api/v1/invoices/:id` remains available to staff and to the owning client.
- `PATCH /api/v1/invoices/:id` retains staff mutation and client notes-only behavior.
- The current response fields and RFC 3339 timestamps remain available with their existing meanings.
- Existing pagination defaults, maximums, envelopes, and role semantics remain unchanged.
- Operational deletion behavior remains available for records with no protected fiscal intent. A linked pending, issued, or retained fiscal record must not be physically deleted through the legacy endpoint; the API should return an explicit conflict and direct authorized users to the applicable fiscal action.

Fiscal APIs and response projections are additive. They do not reinterpret the current invoice `status` field as provider or legal state.

## Proposed approach

```text
mutable operational Invoice
        |
        | employee/manager/admin prepares FT or FR draft
        | manager/admin explicitly finalizes
        v
FiscalDocument + immutable FiscalSnapshot/Lines
        |
        | same PostgreSQL transaction
        v
FiscalOutboxEvent ---- leased worker ----> FiscalProviderGateway
        |                                      |
        |                                      +--> deterministic mock (dev/test)
        |                                      +--> Cloudware adapter (gated)
        v                                      |
FiscalAttempt/audit <---- reconcile/void ------+
        |
        +--> FiscalArtifact metadata --> private immutable storage
        |
        +--> additive API/UI projections for staff and owning client
```

Recommended lifecycle vocabulary is provider-neutral and distinct from operational invoice status, for example: `draft`, `pending`, `dispatching`, `issued`, `rejected`, `connection_action_required`, `outcome_unknown`, and `voided`. The specification and design phases will define exact transition guards, terminal states, conflict responses, and operator messages.

## Affected areas

| Area | Expected impact |
|---|---|
| `backend/internal/domain` | New fiscal document, snapshot/line, connection, attempt, outbox, and artifact concepts; existing `Invoice` remains the operational aggregate. |
| `backend/internal/core/ports` | Provider gateway, fiscal repositories, unit-of-work/outbox boundary, and private fiscal artifact storage contracts. |
| `backend/internal/service` | Draft preparation, finalization, retry, reconciliation, void, connection, artifact authorization, and worker orchestration. |
| `backend/internal/repository/postgres` and `backend/migrations` | Explicit additive tables, constraints, leases, indexes, encrypted credential metadata, audit data, and fail-fast schema validation. |
| `backend/internal/handler` and `backend/cmd/api` | Additive fiscal, artifact, and integration routes with manager/admin and own-invoice authorization. |
| Runtime/configuration | Provider selection, production mock prohibition, credential encryption key, worker controls, private artifact backend, retention, and readiness signals. |
| `frontend/src/types` and `frontend/src/lib/services` | Additive provider-neutral fiscal DTOs and service calls without breaking current invoice types. |
| `/accounting/issued-invoices` | Fiscal draft lines, readiness, status, privileged actions, error guidance, and artifact download. |
| `/my-invoices` | Own-only fiscal status and protected PDF download while preserving notes behavior. |
| Manager/admin settings UI | Connection setup, status, reconnect/action-required presentation, and provider-independent wording. |
| Backend/frontend tests | Strict-TDD coverage for role gates, immutability, idempotency, concurrency, compatibility, failures, reconciliation, mock isolation, and artifact authorization. |
| OpenSpec/API documentation | New fiscal capability, lifecycle, role matrix additions, provider gates, and additive endpoint documentation. |

## Risks and mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| Duplicate legal documents after timeout, crash, or double-click | Critical fiscal and customer impact | Stable intent key, database uniqueness, transactional outbox, concurrency-safe claims, durable pending UI, and reconciliation before any resubmission. |
| Undocumented Cloudware behavior is mistaken for a guarantee | Incorrect legal operation or production outage | Keep every listed unknown as an explicit enablement gate; require credentialed/support evidence and acceptance coverage. |
| Mutable operational data changes legal content | Audit and compliance failure | Copy canonical data into an immutable snapshot at finalization and generate every provider request/retry from that snapshot only. |
| Historical invoices are fiscalized accidentally | Unexpected legal documents and support burden | Default all existing rows to `legacy_unfiscalized`; create no migration events; require explicit eligible draft plus manager/admin finalization. |
| Floating-point or rounding differences | Rejected or inconsistent totals | Use decimal/minor-unit canonical values and explicit line/tax/rounding rules; isolate legacy `float64` at its API boundary. |
| OAuth secrets leak through logs, browser state, or events | Account compromise | Encrypt server-side, use a dedicated key and rotation plan, redact telemetry, never expose tokens to clients, and restrict setup to manager/admin. |
| Provider outage blocks workshop operations | Operational disruption | Dispatch asynchronously, separate provider health from core readiness, cap retries, and expose actionable pending/degraded states. |
| PDF becomes unavailable at the provider | Loss of legal/customer evidence | Retrieve and archive an immutable private copy promptly, checksum it, back it up, and gate production on verified PDF behavior. |
| Archived PDF is exposed to the wrong customer | Privacy and compliance incident | Authorize every download through invoice ownership/staff policy; keep storage private and log access. |
| Mock output is confused with a legal document | False compliance representation | Prohibit mock in production legal mode and mark mock state/artifacts unmistakably. |
| Legacy delete/update behavior conflicts with fiscal retention | Data loss or confusing API behavior | Preserve normal behavior before fiscal intent; return explicit conflict for protected records; use append-only fiscal transitions and voiding. |
| Scope is too broad for effective review | Defects across fiscal, security, data, API, and UI boundaries | Keep implementation internally staged with strict TDD and review checkpoints. The confirmed single-PR plan creates no agent commit/PR and still requires an explicit scope or `size:exception` decision if the forecast exceeds the 600-line review budget. |

## Rollback plan

Rollback must preserve issued fiscal evidence rather than attempting to erase legal history.

1. Disable new fiscal finalization and the Cloudware adapter through configuration while leaving existing invoice routes operational.
2. Stop new outbox claims after allowing in-flight work to reach a known state; retain unprocessed events for diagnosis or controlled replay.
3. Disconnect/revoke provider credentials where supported, mark the local connection unavailable, and remove no audit history.
4. Continue serving already archived PDFs and read-only fiscal status to authorized users where operationally possible.
5. Retain fiscal documents, immutable snapshots, attempts, provider references, and artifacts. An application rollback must not delete or mutate issued records.
6. If a legal document itself must be reversed, use the authorized provider/domain void or future corrective-document workflow; software rollback is not a fiscal cancellation mechanism.
7. Revert additive UI/API exposure if necessary while keeping the compatibility contract intact.
8. Reverse additive database migrations only before any real fiscal document, credential, attempt, or artifact has been stored. After production use, use forward-fix migrations instead.

## Dependencies

- A persisted issuer/workshop fiscal profile and validated customer fiscal identity fields.
- Explicit tax, exemption, currency, decimal, discount, and rounding rules to be made normative in the specification.
- PostgreSQL transactional and locking support for fiscal persistence and outbox claims.
- A production-grade private artifact backend with backup and retention policy.
- A dedicated credential-encryption key and rotation/operations procedure.
- Active Cloudware licensing for non-GET requests, production credentials, and completion of every applicable enablement gate.
- Product/legal confirmation of the required retention period and of the accepted FT/FR and void workflows before live use.

## Success criteria

- [ ] Existing invoice API, UI, pagination, role behavior, client notes flow, and response fields continue to pass regression tests unchanged.
- [ ] Existing invoices remain `legacy_unfiscalized`, and migration/startup creates no fiscal document or outbox event for them.
- [ ] Employees can prepare FT/FR drafts but cannot finalize, retry, reconcile, void, manage connections, or bypass those restrictions through direct HTTP calls.
- [ ] Managers and admins can explicitly finalize an eligible FT/FR draft exactly once, and concurrent or repeated requests cannot create duplicate fiscal intents.
- [ ] Finalization atomically freezes provider-neutral issuer/customer data, manual lines, optional repair/part references, taxes, totals, and provider selection.
- [ ] Changes to operational invoices, customers, repairs, or parts after finalization do not change the frozen provider request or archived fiscal evidence.
- [ ] A committed finalization request survives process failure through the PostgreSQL outbox and can be claimed safely by only one active worker.
- [ ] Ambiguous provider outcomes enter reconciliation without blind reissuance; retries retain the original provider and stable intent key.
- [ ] The deterministic mock covers success, stable repeated calls, rejection, expired authorization, transient failure, ambiguous reconciliation, concurrent claims, voiding, and PDF generation in automated tests.
- [ ] Mock mode cannot produce a production artifact represented as legally fiscal.
- [ ] Successful issuance archives a checksummed immutable private PDF, and only staff or the owning client can download it through GonsGarage.
- [ ] Staff/client UI presents provider-neutral status and actionable guidance without exposing credentials or requiring normal operators to use Cloudware.
- [ ] Cloudware-specific models remain inside the adapter, and no automatic provider failover occurs.
- [ ] Production Cloudware mutation remains disabled until all applicable enablement gates have documented evidence, approved decisions, and passing acceptance coverage.
- [ ] Issued fiscal data remains readable and retained when finalization is disabled or the provider is unavailable.

## Evidence basis

This proposal is based on:

- `exploration.md` revision 2 for the current invoice architecture, compatibility constraints, data gaps, provider-neutral boundary, and delivery risks.
- `research.md` revision 3 for official Cloudware OAuth, v1 FT/FS/FR, finalization, void, receipt, PDF, licensing, and documented-unknown evidence.
- `preproposal.md` revision 4 for the confirmed product decisions and `proposal_ready: true` handoff.

Official evidence supports Cloudware authorization-code/refresh concepts; v1 FT/FR data and lifecycle concepts; requested PDF materialization; voiding finalized documents; and active-license requirements for non-GET requests. It does not establish sandbox access, native idempotency, reconciliation lookup, webhooks, rate limits, retry guidance, PDF lifetime, or automatic v1 AT/e-Fatura communication, so this proposal intentionally leaves those as production enablement gates.
