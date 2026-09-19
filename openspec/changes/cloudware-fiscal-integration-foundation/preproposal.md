{
  "schema": "gentle-ai.sdd-preproposal/v1",
  "revision": 4,
  "changeName": "cloudware-fiscal-integration-foundation",
  "worktree": "D:/repos/gonsgarage",
  "explorationReference": {
    "store": "openspec",
    "path": "D:/repos/gonsgarage/openspec/changes/cloudware-fiscal-integration-foundation/exploration.md",
    "read": true,
    "observedRevision": 2,
    "expectedRevision": 2,
    "expectedDigest": "sha256:26b1cb64a303abba5e600abc7510ebd5f950c58ca403ea3ebed6f78c4fff97e0",
    "identityValidated": true,
    "reason": "Parent read the complete JSON artifact and independently verified its SHA-256 with sha256sum."
  },
  "researchRequest": {
    "retainedIntent": "establish official provider-contract facts needed for a provider-neutral/mock foundation and explicitly retain unknowns that require Cloudware credentials or support confirmation",
    "questions": [
      "Credential acquisition and OAuth authorization/refresh flow.",
      "v1 sales documents: FT/FS/FR, customer/NIF, series, lines, IVA/exemption, external_reference, finalize, void, PDF.",
      "Receipt workflow and FT+receipt versus FR.",
      "What official docs say and do not say about automatic AT/e-Fatura communication versus the v0 AT endpoint.",
      "What official docs say and do not say about sandbox, idempotency, webhooks, rate limits, retries, and PDF URL lifetime.",
      "Official product/licensing/integration claims."
    ],
    "classes": {
      "documentation": {
        "selected": true,
        "tools": ["fetch_content"],
        "extensions": {
          "fetch_content": "C:\\Users\\gaston.garcia\\.pi\\agent\\npm\\node_modules\\pi-web-access\\index.ts"
        }
      },
      "open-web": {
        "selected": false,
        "tools": [],
        "extensions": {}
      }
    },
    "officialSourcesOnly": true
  },
  "admission": {
    "outcome": "admitted",
    "reason": "Official-documentation research completed with source-backed answers for all retained questions. Open-web was explicitly unselected by the user."
  },
  "evidenceReferences": [
    "CW-SETUP",
    "CW-AUTH",
    "CW-REQUESTS",
    "CW-V1-SALES",
    "CW-V1-OPENAPI",
    "CW-V1-RECEIPTS",
    "CW-V0-AT",
    "CW-POS",
    "CW-INDEX"
  ],
  "researchReference": {
    "store": "openspec",
    "path": "D:/repos/gonsgarage/openspec/changes/cloudware-fiscal-integration-foundation/research.md",
    "revision": 3,
    "digest": "sha256:09e21c877218a760f5fddc39cdb98b0c726d6ced30bb6a6607ac2a0f1e7c0f4d",
    "identityValidated": true,
    "reason": "Parent read the complete revision 3 artifact and independently verified its SHA-256 with sha256sum."
  },
  "researchOutcome": "done",
  "productDecisions": {
    "status": "confirmed",
    "finalizationTrigger": "explicit staff action",
    "firstReleaseDocumentTypes": ["FT", "FR"],
    "lineSource": "manual lines with optional internal repair/part links",
    "issuanceRoles": ["manager", "admin"],
    "immutabilityBoundary": "freeze fiscal snapshot and lines when finalization is requested",
    "legacyPolicy": "existing invoices remain legacy_unfiscalized and are never selected automatically",
    "operatorExperience": "provider-transparent GonsGarage UI",
    "providerPolicy": "no automatic cross-provider failover",
    "pdfPolicy": "archive an immutable private copy and serve it through authorized GonsGarage endpoints",
    "oauthPolicy": "manager/admin connection setup; normal operators never use Cloudware UI",
    "mockPolicy": "mock provider allowed only for development/test and never represented as a legal fiscal document"
  },
  "proposal_ready": true,
  "blockers": [],
  "adapterEnablementUnknowns": [
    "Cloudware sandbox or test-company availability",
    "PKCE, OAuth state and revocation behavior",
    "native idempotency and reconciliation lookup",
    "webhooks, rate limits and retry guidance",
    "PDF URL lifetime and retention semantics",
    "automatic API v1 AT/e-Fatura communication"
  ],
  "persistence": {
    "store": "openspec",
    "path": "D:/repos/gonsgarage/openspec/changes/cloudware-fiscal-integration-foundation/preproposal.md",
    "writeTargetRevision": 4,
    "postWriteReadback": "completed-by-parent"
  }
}
