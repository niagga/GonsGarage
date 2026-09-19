# Fiscal Provider Foundation Specification

## Purpose

Define the provider-neutral, durable, and failure-safe boundary used to issue, reconcile, void, and retrieve evidence without coupling core invoice behavior to one provider.

## Requirements

### Requirement: Provider gateway is normalized and provider-bound

The system MUST expose provider-neutral operations for connection verification, issuance, reconciliation, voiding, and artifact retrieval. Provider-specific request models, responses, status names, and errors MUST be translated before crossing this boundary. The provider selected at finalization MUST remain fixed for issuance, retries, reconciliation, voiding, and artifact retrieval; automatic cross-provider failover MUST NOT occur.

#### Scenario: Provider becomes unavailable after finalization

- GIVEN a frozen intent selects provider A
- WHEN provider A becomes unavailable and provider B is healthy
- THEN the system MUST retain provider A for that intent
- AND it MUST expose a recoverable or unavailable state rather than dispatching to provider B

### Requirement: Finalization and initial dispatch are atomic

The frozen fiscal document, snapshot, lines, selected provider, stable intent key, and initial outbox event MUST be committed in one PostgreSQL transaction. A transaction failure MUST leave none of those finalization effects committed.

#### Scenario: Dispatch event persistence fails

- GIVEN an authorized finalization request
- WHEN the initial outbox event cannot be persisted
- THEN the finalization transaction MUST roll back
- AND no frozen intent may appear successfully queued

#### Scenario: Process stops after commit

- GIVEN finalization and its initial event commit successfully
- WHEN the API process stops before dispatch
- THEN durable work MUST remain claimable after recovery

### Requirement: Outbox claims are crash-safe and concurrency-safe

Provider work MUST use durable lease-based claims. At most one unexpired worker lease MAY actively process an event, expired leases MUST be recoverable, and an event MUST NOT be marked complete until its normalized result and audit transition are durable.

#### Scenario: Concurrent workers claim one event

- GIVEN multiple workers poll the same ready event
- WHEN claims race
- THEN only one worker MUST obtain the active lease
- AND other workers MUST NOT invoke the provider for that claim

#### Scenario: Worker crashes while leased

- GIVEN a worker crashes before durably recording an outcome
- WHEN its lease expires
- THEN another worker MAY recover the event using the same provider, operation key, and frozen input

### Requirement: Provider operations are idempotently correlated

Every legal operation MUST have a stable internal correlation and idempotency key. Repeated UI actions, API requests, recovered leases, and eligible retries MUST reuse the operation's key and MUST NOT create a second active fiscalization intent. Provider-native idempotency MAY be used only when its behavior is evidenced; internal safety MUST NOT assume it exists.

#### Scenario: User double-clicks finalization

- GIVEN the first finalization request has committed
- WHEN an equivalent request is repeated
- THEN the system MUST return the existing intent or an explicit conflict
- AND it MUST NOT enqueue a second legal issuance intent

### Requirement: Failures are classified without unsafe resubmission

Each attempt MUST be classified as validation, authorization, transient, rate-limit, permanent, or ambiguous outcome. Definite validation or permanent provider rejection MUST become `rejected`; authorization failure MUST become `connection_action_required`; an unambiguous transient or rate-limit failure MAY become `retryable_failure` and MAY receive bounded retry according to an approved policy; an ambiguous issuance result MUST become `outcome_unknown`. Bounded retries MUST reuse the original provider and operation key.

#### Scenario: Rate limit response is definite

- GIVEN the provider definitely reports a rate limit without accepting issuance
- WHEN the attempt is classified
- THEN the system MAY schedule bounded retry under the approved policy
- AND every retry MUST reuse the same frozen intent and provider

#### Scenario: Timeout may have followed issuance

- GIVEN the provider may have accepted the operation before a timeout
- WHEN no definitive result is available
- THEN the system MUST classify the result as ambiguous
- AND it MUST NOT automatically resubmit issuance

### Requirement: Ambiguous outcomes require reconciliation

Reconciliation MUST query the same provider using the stable provider reference, correlation data, or other evidenced lookup. Until reconciliation proves the legal result, issuance MUST remain blocked. If reconciliation proves that no document exists, the system MAY move the intent to `retryable_failure`, but resubmission MUST still require an eligible retry action or an approved bounded policy.

#### Scenario: Reconciliation finds an issued document

- GIVEN an issuance is `outcome_unknown`
- WHEN reconciliation finds the matching provider document
- THEN the system MUST record the provider reference and transition to `issued`
- AND it MUST NOT issue another document

#### Scenario: Reconciliation cannot establish a result

- GIVEN an ambiguous operation and available provider lookup
- WHEN lookup remains inconclusive
- THEN the system MUST remain in the corresponding unknown state
- AND it MUST present escalation guidance without blind retry

### Requirement: Attempts and transitions are durable and redacted

The system MUST retain each provider attempt with operation, timestamps, correlation identifiers, classification, normalized result, and redacted diagnostic data. Credentials, authorization codes, refresh tokens, access tokens, bearer URLs, and other secrets MUST NOT appear in attempts, outbox payloads, API responses, or unredacted logs.

#### Scenario: Provider returns a secret-bearing error

- GIVEN a provider response contains token or credential material
- WHEN the attempt is persisted or logged
- THEN secret material MUST be removed or irreversibly redacted
- AND non-secret diagnostic classification MUST remain available to authorized staff

### Requirement: Provider outages do not disable core invoice operations

Fiscal provider health MUST be isolated from ordinary invoice API and application availability. Provider-dependent actions MUST expose durable pending, degraded, or unavailable results without blocking unrelated invoice reads, notes, or allowed operational mutations.

#### Scenario: Provider is offline

- GIVEN the selected provider cannot be reached
- WHEN users perform ordinary invoice reads or client notes updates
- THEN those operations MUST continue under their existing contract
- AND provider-dependent work MUST remain durable for recovery or action

### Requirement: Non-production mock is deterministic and non-legal

A mock provider MUST be available only in development and automated-test environments. For the same operation key and frozen input it MUST return stable results and artifacts. It MUST support successful FT and FR issuance, stable repeated calls, validation rejection, expired connection, transient failure, ambiguous outcome followed by reconciliation, concurrent processing, permitted and refused voids, and deterministic PDF output. Mock output MUST be visibly identified as non-legal and MUST NOT be selectable for production legal issuance.

#### Scenario: Same mock issuance is repeated

- GIVEN identical frozen input and operation key
- WHEN the mock receives the issuance operation more than once
- THEN it MUST return the same normalized provider identity, result, and PDF bytes
- AND it MUST NOT simulate multiple legal documents

#### Scenario: Production selects mock

- GIVEN the runtime is configured as production
- WHEN mock selection or mock legal issuance is attempted
- THEN startup or the action MUST fail closed
- AND no artifact may be represented as legally issued
