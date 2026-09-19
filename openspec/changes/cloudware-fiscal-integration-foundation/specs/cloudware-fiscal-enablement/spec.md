# Cloudware Fiscal Enablement Specification

## Purpose

Define Cloudware-specific connection and document behavior while keeping production mutations disabled until provider evidence supports safe operation.

## Requirements

### Requirement: Cloudware connections are scoped and privileged

The system MUST maintain an instance or workshop-scoped Cloudware connection with a stable internal identifier, provider key, connection state, provider organization reference, granted scopes, expiry information, and audit metadata. Only managers and admins MAY connect, reconnect, verify, or disconnect it. Employees and clients MUST NOT manage provider connections.

#### Scenario: Manager starts connection setup

- GIVEN an authenticated manager and an enabled non-production or approved production setup flow
- WHEN the manager starts Cloudware connection setup
- THEN the system MAY redirect that manager through the provider-hosted consent flow
- AND normal invoice operators MUST NOT need Cloudware accounts or Cloudware UI

#### Scenario: Employee calls connection endpoint

- GIVEN an authenticated employee
- WHEN the employee calls a connection-management action directly
- THEN the system MUST deny the action before exchanging or changing credentials

### Requirement: OAuth credentials remain server-side and protected

Cloudware authorization-code and refresh-token handling MUST occur server-side. Client secrets, authorization codes, access tokens, and refresh tokens MUST be encrypted at rest where persisted and MUST NOT enter frontend storage, ordinary API responses, outbox payloads, or unredacted logs. Production OAuth MUST remain disabled until redirect registration, request-state binding, PKCE support or non-use, scopes, expiry, refresh rotation, revocation, and reconnect behavior have evidence and an approved decision.

#### Scenario: OAuth callback succeeds

- GIVEN an authorized manager returns with a callback bound to the initiated session
- WHEN the server exchanges the authorization code
- THEN resulting credentials MUST be retained only in protected server-side storage
- AND the browser MUST receive only provider-neutral connection status

#### Scenario: Callback state is invalid

- GIVEN a callback is missing or does not match the initiated authorization context
- WHEN the callback is processed
- THEN the system MUST reject it without storing or exposing credentials

### Requirement: Cloudware mapping is limited to evidenced FT and FR concepts

The Cloudware adapter MUST map only the approved first-release FT and FR workflow and documented v1 concepts for customer identity, series, manual lines, quantity, unit price, IVA or exemption, external reference, finalization, voiding, and requested PDF materialization. It MUST NOT enable FS, credit notes, debit notes, standalone receipts, partial-receipt workflows, automated correction, or undocumented mappings in this release. Cloudware request and response models MUST NOT become generic fiscal API contracts.

#### Scenario: Frozen FR is mapped

- GIVEN an eligible frozen FR snapshot and production enablement gates are satisfied
- WHEN the adapter creates the Cloudware request
- THEN it MUST map only approved frozen values
- AND it MUST treat Cloudware's documented associated-receipt behavior as part of FR rather than creating a separate local receipt workflow

#### Scenario: Unsupported receipt workflow is requested

- GIVEN a caller requests a standalone or partial receipt
- WHEN the first-release fiscal API validates the request
- THEN it MUST reject the workflow before any Cloudware mutation

### Requirement: Production Cloudware mutation is evidence-gated

Real Cloudware issuance, retry, reconciliation, voiding, and artifact retrieval that depends on mutation behavior MUST remain disabled until every applicable gate has recorded evidence, an approved behavior, and passing acceptance coverage. The gates MUST cover test-company or controlled validation access; OAuth and reconnect security; idempotency and ambiguous-outcome lookup; webhook behavior if used; rate limits and retry guidance; PDF authentication, lifetime, authenticity, and retrieval; automatic AT/e-Fatura behavior; credentialed response and error behavior; FT/FR and void behavior; and active-GC-license requirements.

#### Scenario: One applicable gate lacks evidence

- GIVEN production configuration selects Cloudware and any applicable gate lacks evidence, decision, or acceptance coverage
- WHEN a mutation is attempted
- THEN the system MUST fail closed before sending the provider mutation
- AND it MUST expose provider-neutral setup guidance to authorized managers or admins

#### Scenario: A gate is not applicable

- GIVEN a feature such as webhooks is not used
- WHEN production readiness is evaluated
- THEN the gate record MUST explicitly state non-applicability and its rationale
- AND the system MUST NOT silently assume undocumented webhook behavior

### Requirement: Undocumented provider behavior is not assumed

The integration MUST NOT assume Cloudware provides native idempotency, correlation lookup, webhooks, sandbox entitlement, retry timing, rate-limit semantics, persistent PDF URLs, or automatic v1 AT/e-Fatura communication until the corresponding gate is evidenced. A product trial MUST NOT be treated as API sandbox entitlement. User-visible claims MUST distinguish local issuance state, provider confirmation, archived evidence, and any separately evidenced AT communication state.

#### Scenario: Provider times out before idempotency evidence exists

- GIVEN Cloudware production validation has not established a safe idempotency and lookup contract
- WHEN an issuance result is ambiguous
- THEN the system MUST block blind resubmission
- AND production enablement MUST remain incomplete until a safe reconciliation decision is approved

#### Scenario: UI presents issued status

- GIVEN Cloudware confirms a v1 document but automatic AT communication remains unverified
- WHEN status is shown to staff or a client
- THEN the system MUST NOT claim AT or e-Fatura communication occurred
- AND it MAY state only the evidenced provider-neutral issuance and artifact facts

### Requirement: Connection and provider errors are normalized

Cloudware-specific status codes and response bodies MUST be translated into the provider-neutral failure classifications and actionable connection states. Credential refresh or active-license failure MUST NOT expose secrets and MUST NOT make ordinary invoice routes unavailable.

#### Scenario: Refresh token fails

- GIVEN an otherwise valid frozen intent and a failed Cloudware refresh
- WHEN the adapter reports the failure
- THEN the connection MUST become action-required under the approved mapping
- AND the fiscal intent MUST become `connection_action_required` without switching provider

#### Scenario: Active GC license is absent

- GIVEN Cloudware refuses a non-GET operation because the working company lacks an active license
- WHEN the response is normalized
- THEN the system MUST present manager/admin remediation guidance
- AND it MUST retain the frozen intent without automatic failover or destructive mutation
