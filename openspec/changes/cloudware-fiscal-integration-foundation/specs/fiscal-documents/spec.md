# Fiscal Documents Specification

## Purpose

Define provider-neutral preparation, finalization, lifecycle, authorization, and presentation of Portuguese FT and FR documents linked to customer invoices.

## Requirements

### Requirement: Fiscal drafts are distinct from operational invoices

The system MUST represent a fiscal document as an aggregate linked to, but distinct from, the mutable operational customer invoice. The first release MUST accept only `FT` and `FR`, MUST use manually entered fiscal lines, and MAY attach a line to an existing repair or part solely for traceability. Authenticated employees, managers, and admins MUST be allowed to prepare eligible drafts; clients MUST NOT create, update, or list staff draft data.

#### Scenario: Employee prepares an FT draft

- GIVEN an authenticated employee and an eligible customer invoice
- WHEN the employee creates an FT draft with valid manual lines
- THEN the system MUST save a mutable fiscal draft linked to that invoice
- AND it MUST NOT request provider issuance

#### Scenario: Client attempts draft preparation

- GIVEN an authenticated client
- WHEN the client calls a fiscal draft creation or mutation action directly
- THEN the system MUST deny the action
- AND it MUST NOT expose another invoice's draft data

#### Scenario: Optional source record later changes

- GIVEN a draft line references a repair or part
- WHEN the referenced record changes or is deleted
- THEN the captured line values MUST remain unchanged
- AND the reference MUST remain traceability metadata rather than a source of automatic recalculation

#### Scenario: Unsupported document kind

- GIVEN a caller submits a document kind other than FT or FR
- WHEN the system validates the draft
- THEN it MUST reject the request without creating a fiscal finalization intent

### Requirement: Drafts contain complete provider-neutral fiscal data

A finalizable draft MUST contain the document kind, issuer identity, customer identity, billing address, currency, manual lines, quantities, unit prices, discounts, tax treatments, exemption reasons where applicable, rounding data, totals, and optional source references required by the applicable approved fiscal policy. Canonical quantities and money MUST use decimal values or integer minor units; binary floating-point MAY be accepted only at a legacy compatibility boundary and MUST NOT be the canonical fiscal value.

#### Scenario: Required fiscal identity is incomplete

- GIVEN a draft lacks data required by the approved policy for its document kind and customer
- WHEN finalization is requested
- THEN the system MUST reject finalization with provider-neutral field guidance
- AND the draft MUST remain editable

#### Scenario: Legacy floating-point amount is supplied

- GIVEN an existing invoice exposes an amount through its legacy floating-point contract
- WHEN a fiscal draft is prepared from user-entered fiscal data
- THEN the system MUST store canonical fiscal amounts independently as exact decimal or minor-unit values
- AND it MUST NOT treat the legacy floating-point amount as the immutable legal total

### Requirement: Fiscal arithmetic is deterministic and validated

The system MUST calculate each line and document under an approved, versioned fiscal arithmetic policy. That policy MUST explicitly define supported currencies, decimal precision limits, the tax and exemption catalog, discount order, rounding mode, rounding point, and permitted document-level adjustment; legal finalization MUST fail closed when no approved policy applies. For each line, gross amount MUST equal quantity multiplied by unit price, net amount MUST equal gross amount minus the normalized discount, tax MUST be derived from the captured tax treatment and taxable net amount using that policy's currency precision and rounding point, and line total MUST equal net amount plus tax. Document totals MUST equal the sums of frozen line amounts plus any explicit policy-permitted rounding adjustment. Finalization MUST reject negative quantities, non-positive sale quantities, discounts exceeding the gross amount, inconsistent totals, unsupported currencies or tax treatments, and missing exemption reasons when the selected tax treatment requires one.

#### Scenario: Arithmetic policy is unavailable

- GIVEN a draft has no approved arithmetic policy for its currency or tax treatment
- WHEN legal finalization is requested
- THEN the system MUST reject finalization before provider dispatch
- AND it MUST identify the missing policy to an authorized user

#### Scenario: Totals reconcile

- GIVEN valid lines and an available approved arithmetic policy
- WHEN the system calculates the draft
- THEN every displayed and frozen subtotal, tax total, rounding adjustment, and payable total MUST reconcile under the same policy version

#### Scenario: Submitted totals do not reconcile

- GIVEN caller-supplied totals differ from the system's deterministic calculation
- WHEN finalization is requested
- THEN the system MUST reject finalization
- AND it MUST identify the inconsistency without asking a provider to issue the document

### Requirement: Explicit finalization freezes one intent

Only an authenticated manager or admin MAY request finalization. A successful request MUST atomically freeze the complete snapshot and all lines, select one configured provider, assign a stable intent and correlation key, and create durable dispatch work. Repeated or concurrent requests for the same source invoice and document intent MUST resolve to the same fiscal intent or an explicit conflict and MUST NOT create duplicate active intents.

#### Scenario: Manager finalizes an eligible draft

- GIVEN an eligible mutable draft and an available allowed provider connection
- WHEN a manager explicitly requests finalization
- THEN the system MUST freeze the snapshot, lines, policy version, provider choice, and intent key atomically
- AND the resulting lifecycle state MUST be `pending`

#### Scenario: Employee attempts finalization

- GIVEN an authenticated employee has prepared a valid draft
- WHEN the employee calls the finalization action directly
- THEN the system MUST deny the action
- AND the draft MUST remain unfinalized

#### Scenario: Two finalization requests race

- GIVEN two authorized requests target the same eligible draft and intent
- WHEN they execute concurrently
- THEN at most one frozen intent and one initial dispatch event MUST exist
- AND both callers MUST receive either the same intent result or an explicit conflict

### Requirement: Frozen fiscal input is immutable

After finalization is requested, the system MUST NOT modify or physically delete the frozen issuer, customer, address, currency, document kind, lines, source references, tax data, totals, arithmetic policy, provider choice, or intent key. Every provider issuance or retry MUST be derived from that frozen snapshot, not from current operational records.

#### Scenario: Customer or invoice changes after finalization

- GIVEN a fiscal document is no longer `draft`
- WHEN its customer, invoice, repair, part, or issuer source record changes
- THEN the frozen fiscal snapshot and every subsequent provider request MUST remain byte-for-byte equivalent in fiscal meaning

#### Scenario: Staff attempts to edit frozen lines

- GIVEN a finalization intent exists
- WHEN any role attempts to alter or delete a frozen line
- THEN the system MUST reject the mutation with an explicit conflict

### Requirement: Lifecycle transitions are guarded and auditable

The provider-neutral lifecycle MUST use `draft`, `pending`, `dispatching`, `issued`, `rejected`, `connection_action_required`, `retryable_failure`, `outcome_unknown`, `void_pending`, `void_outcome_unknown`, and `voided`. Only `draft` MAY return to editable draft behavior. `pending` MAY become `dispatching`; issuance processing MAY end in `issued`, `rejected`, `connection_action_required`, `retryable_failure`, or `outcome_unknown`. Only an eligible `issued` document MAY enter `void_pending`; void processing MUST resolve to `voided`, return to `issued` after a definite failed void, or enter `void_outcome_unknown`. Every transition MUST retain actor, time, operation, prior state, next state, and normalized reason.

#### Scenario: Definite validation rejection

- GIVEN the provider definitely rejects the frozen issuance request as invalid
- WHEN the result is recorded
- THEN the document MUST enter `rejected`
- AND the system MUST NOT silently edit or reissue the frozen intent

#### Scenario: Ambiguous issuance result

- GIVEN dispatch times out without proof that issuance failed
- WHEN the attempt is recorded
- THEN the document MUST enter `outcome_unknown`
- AND no issuance retry MUST be permitted before reconciliation establishes a safe result

#### Scenario: Definite void failure

- GIVEN an issued document enters `void_pending`
- WHEN the provider definitely refuses or fails the void without changing legal state
- THEN the document MUST return to `issued`
- AND the failed void attempt MUST remain auditable

### Requirement: Privileged recovery and void actions preserve legal history

Only managers and admins MAY retry an eligible `retryable_failure` or `connection_action_required` intent, reconcile `outcome_unknown` or `void_outcome_unknown`, or request voiding. Retry and reconciliation MUST reuse the frozen provider and stable intent key. Voiding MUST be allowed only for an issued document when the provider and approved fiscal policy permit it, and MUST retain the issued record and evidence.

#### Scenario: Manager retries after connection recovery

- GIVEN a document is `connection_action_required` and its original provider connection is healthy again
- WHEN a manager requests retry
- THEN the system MUST queue the same frozen intent for the same provider
- AND it MUST NOT create a replacement fiscal document

#### Scenario: Employee requests reconciliation or void

- GIVEN an employee can view an invoice
- WHEN the employee directly requests reconciliation or voiding
- THEN the system MUST deny the action regardless of current fiscal state

#### Scenario: Void succeeds

- GIVEN a manager requests a permitted void of an issued document
- WHEN the provider confirms the void
- THEN the system MUST mark the document `voided`
- AND it MUST retain the original snapshot, provider reference, attempts, and artifacts

### Requirement: Legacy invoices remain isolated

Every invoice that predates this capability MUST be presented as `legacy_unfiscalized`. Migrations, startup, scheduled work, invoice status changes, repair completion, payment, and background scans MUST NOT create drafts, intents, or dispatch events for legacy invoices.

#### Scenario: Upgrade with historical invoices

- GIVEN the database contains invoices before the fiscal feature is deployed
- WHEN migrations and application startup complete
- THEN those invoices MUST remain `legacy_unfiscalized`
- AND no fiscal document or outbox event MUST be created for them

### Requirement: Fiscal APIs and user interfaces are additive and provider-neutral

The system MUST expose fiscal resources and actions through additive nested contracts without renaming, removing, or reinterpreting existing invoice routes or fields. `/accounting/issued-invoices` MUST show authorized draft preparation, readiness, lifecycle, actions, durable progress, and actionable provider-neutral errors. `/my-invoices` MUST show the owning client a simple pending, finalized, voided, or unavailable presentation. Normal operators MUST NOT need a provider account or provider UI.

#### Scenario: Existing invoice contract remains stable

- GIVEN an existing consumer uses invoice list, create, own-list, detail, update, pagination, timestamps, or client notes behavior
- WHEN the fiscal capability is introduced
- THEN the existing route, field, role, and notes semantics MUST remain unchanged
- AND fiscal projections MUST be additive

#### Scenario: Owning client reads fiscal status

- GIVEN an authenticated client owns the source invoice
- WHEN the client requests its detail
- THEN the client MAY receive the provider-neutral fiscal projection
- AND MUST NOT receive provider credentials, raw provider payloads, or privileged lifecycle actions

#### Scenario: Protected invoice deletion

- GIVEN an invoice has a linked pending, issued, voided, or otherwise retained fiscal intent
- WHEN deletion is requested through the legacy invoice endpoint
- THEN the system MUST reject physical deletion with an explicit conflict
- AND it MUST direct an authorized user to an applicable fiscal action when one exists
