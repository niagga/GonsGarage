# Fiscal Artifacts Specification

## Purpose

Define immutable, private, and authorized preservation of fiscal PDF evidence independently of temporary provider links.

## Requirements

### Requirement: Issuance archives immutable PDF evidence

For each provider-confirmed issuance, the system MUST obtain and archive a private immutable PDF copy and MUST persist the source invoice and fiscal document ownership, private storage key, media type, byte size, checksum, provider reference or version, environment or legal classification, and creation time. Provider bearer URLs and public storage paths MUST NOT be exposed as the customer download contract.

#### Scenario: Issuance returns a PDF

- GIVEN the provider confirms issuance and supplies PDF material
- WHEN issuance processing handles the result
- THEN the system MUST store immutable PDF bytes in private storage
- AND the checksum and ownership metadata MUST match those bytes and the fiscal document

#### Scenario: Archived bytes are altered

- GIVEN an artifact has been archived
- WHEN later integrity verification computes a different checksum
- THEN the system MUST mark the artifact unavailable or compromised
- AND it MUST NOT serve the altered bytes as valid fiscal evidence

### Requirement: Artifact failure cannot cause duplicate issuance

A provider-confirmed legal issuance MUST remain `issued` even if PDF retrieval or archival fails. The system MUST expose the artifact as temporarily unavailable, retain durable recovery work, and MUST NOT reissue the fiscal document merely to obtain its PDF.

#### Scenario: PDF retrieval fails after confirmed issuance

- GIVEN the provider has confirmed the fiscal document exists
- WHEN the PDF cannot be retrieved or archived
- THEN the fiscal document MUST remain `issued`
- AND artifact recovery MUST use the same provider reference without issuing again

### Requirement: Downloads are authenticated and ownership-authorized

Fiscal PDFs MUST be served only through authenticated GonsGarage endpoints. Employees, managers, and admins MAY access artifacts according to the existing staff invoice policy; a client MAY access an artifact only when the linked source invoice belongs to that client. Every download decision MUST be based on current authenticated authorization and the persisted ownership relationship.

#### Scenario: Owning client downloads PDF

- GIVEN an authenticated client owns the source invoice and its archived legal PDF is available
- WHEN the client requests the artifact through GonsGarage
- THEN the system MUST authorize and return the archived PDF
- AND it MUST NOT reveal storage credentials or a provider bearer URL

#### Scenario: Different client requests PDF

- GIVEN an authenticated client does not own the source invoice
- WHEN that client requests the artifact directly by identifier
- THEN the system MUST deny access without returning artifact bytes or sensitive metadata

### Requirement: Legal and mock artifacts are unmistakable

The system MUST distinguish legal provider artifacts from mock artifacts in persisted metadata and user presentation. Mock PDFs MUST contain visible non-legal labeling and MUST NOT be served from a production legal-document flow.

#### Scenario: Developer downloads mock PDF

- GIVEN a non-production mock document exists
- WHEN its PDF is rendered or downloaded
- THEN the PDF and surrounding presentation MUST identify it as mock and not legally valid

### Requirement: Production artifact readiness is explicit

Production fiscal issuance MUST NOT be enabled unless private storage, checksum verification, retention, backup, restore, and access logging policies are configured and accepted. Disabling new issuance or a provider MUST NOT remove authorized read access to already archived evidence.

#### Scenario: Artifact backend is not production-ready

- GIVEN required storage, retention, backup, restore, or access logging readiness is absent
- WHEN production issuance is enabled
- THEN enablement MUST fail closed

#### Scenario: New issuance is disabled

- GIVEN legal PDFs were archived before issuance was disabled
- WHEN an authorized user requests one of those PDFs
- THEN the system SHOULD continue to serve the retained artifact when the application and storage are available
