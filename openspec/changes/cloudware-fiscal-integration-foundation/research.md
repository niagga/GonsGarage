{
  "schema": "gentle-ai.sdd-research/v1",
  "revision": 3,
  "changeName": "cloudware-fiscal-integration-foundation",
  "worktree": "D:/repos/gonsgarage",
  "outcome": "done",
  "retainedIntent": "establish official provider-contract facts needed for a provider-neutral/mock foundation and explicitly retain unknowns that require Cloudware credentials or support confirmation",
  "skill_resolution": "paths-injected",
  "questions": [
    {
      "id": 1,
      "question": "Credential acquisition and OAuth authorization/refresh flow.",
      "status": "answered",
      "answer": "An administrator enters the integrator name and email under Empresa > Dados API; the integrator receives a temporary credential link, and the authentication page specifies 72 hours. The documented OAuth flow uses GET <OAUTH_URL>/auth with client_id, redirect_uri, response_type=code, and scope=commercial. POST <OAUTH_URL>/token uses application/x-www-form-urlencoded, Accept: application/json, and HTTP Basic client_id:secret. Refresh uses grant_type=refresh_token and returns new access and refresh tokens. API requests require application/vnd.api+json, application/json, and Bearer access-token headers. The inspected official pages do not mention PKCE, OAuth state, revocation, or sandbox access, so those remain unconfirmed rather than assumed."
    },
    {
      "id": 2,
      "question": "v1 sales documents: FT/FS/FR, customer/NIF, series, lines, IVA/exemption, external_reference, finalize, void, and PDF.",
      "status": "answered",
      "answer": "POST /v1/commercial_sales_documents models document_type examples FT, FS, and FR; optional series id or prefix; customer id or NIF plus customer profile fields; external_reference; required lines with item, description, quantity, unit price, and unit references; VAT code/percentage/region and a document exemption-reason reference when a line is exempt. A document can be finalized during create/update or later through PATCH /v1/commercial_sales_documents/{id}/finalize. return_pdf requests PDF materialization. A preparation document can be deleted; after finalization it cannot be edited or deleted and may instead be voided through PATCH /{id}/void. The v1 OpenAPI 1.0 response schema exposes a url example, but URL lifetime, authentication, retention, and archival guarantees are not documented in the inspected sources."
    },
    {
      "id": 3,
      "question": "Receipt workflow and FT plus receipt versus FR.",
      "status": "answered",
      "answer": "The v1 sales-document page states that finalizing an FR automatically creates its associated receipt. Separately, POST /v1/commercial_sales_receipts creates a receipt whose lines identify Document or DocumentLine receivables and received_value; the page explicitly allows partial receipt and says the line identifies the FT or other document paid. Immediate-payment FR and FT-plus-separate-receipt are therefore distinct documented workflows."
    },
    {
      "id": 4,
      "question": "What official docs say and do not say about automatic AT/e-Fatura communication versus the v0 AT endpoint.",
      "status": "answered",
      "answer": "The official v0 page documents POST /send_document_at_webservice, AT username/password parameters, supported sales/shipment/purchase-shipment document categories, and communication status/code/message fields. The inspected v1 1.0 OpenAPI contains sales-document finalization but no statement or field establishing automatic AT/e-Fatura communication. Automatic v1 communication is therefore an explicit unknown requiring Cloudware credentials or support confirmation; absence in the inspected documentation is not evidence that the product does not communicate automatically."
    },
    {
      "id": 5,
      "question": "What official docs say and do not say about sandbox, idempotency, webhooks, rate limits, retries, and PDF URL lifetime.",
      "status": "answered",
      "answer": "The inspected official setup, authentication, request-characteristics, v1 sales-document, v1 OpenAPI 1.0, documentation-index, and POS pages provide no affirmative documentation for an API sandbox/test company, provider idempotency key, webhooks, rate limits, retry or ambiguous-outcome guidance, or PDF URL lifetime/authentication/retention. The POS page's 30-day trial is a product trial claim, not evidence of API sandbox entitlement. These remain explicit adapter unknowns pending credentialed testing or Cloudware support confirmation."
    },
    {
      "id": 6,
      "question": "Official product/licensing/integration claims.",
      "status": "answered",
      "answer": "Cloudware's request-characteristics page says the working company needs an active GC licence for non-GET requests, while GET requests remain authorized after licence expiry. The official POS page advertises online/cloud operation, an API for custom integrations, a 30-day trial, and pricing from 8 EUR/month. These are product and marketing facts; they do not establish API entitlement, sandbox access, or support terms."
    }
  ],
  "sourceRestriction": [
    "Cloudware GitBook original content",
    "Cloudware's GitBook-linked official OpenAPI",
    "cloudware.pt original content"
  ],
  "admission": {
    "outcome": "admitted",
    "reason": "The selected documentation class had the injected fetch_content capability, a matching active child-local tool, and the exact registered pi-web-access extension provenance. Open-web was explicitly unselected and was not used.",
    "classes": {
      "documentation": {
        "selected": true,
        "requestedTools": [
          "fetch_content"
        ],
        "observedActiveTools": [
          "fetch_content"
        ],
        "observedGrant": {
          "tools": [
            "fetch_content"
          ],
          "extensions": {
            "fetch_content": "C:\\Users\\gaston.garcia\\.pi\\agent\\npm\\node_modules\\pi-web-access\\index.ts"
          }
        },
        "admitted": true,
        "execution": "succeeded"
      },
      "open-web": {
        "selected": false,
        "requestedTools": [],
        "observedActiveTools": [],
        "observedGrant": {
          "tools": [],
          "extensions": {}
        },
        "admitted": false,
        "execution": "not-run",
        "denialReason": "Open-web was explicitly unselected by the user-approved narrowing and had no injected capability mapping for this run."
      }
    }
  },
  "artifactInputValidation": {
    "explore": {
      "path": "D:/repos/gonsgarage/openspec/changes/cloudware-fiscal-integration-foundation/exploration.md",
      "expectedRevision": 2,
      "expectedDigest": "sha256:26b1cb64a303abba5e600abc7510ebd5f950c58ca403ea3ebed6f78c4fff97e0",
      "read": true,
      "observedRevision": 2,
      "identityValidated": false,
      "reason": "The exact bounded JSON file was read completely and its revision, change name, and worktree matched, but no active approved tool exposed an independently observed SHA-256 digest."
    },
    "researchRevision2": {
      "path": "D:/repos/gonsgarage/openspec/changes/cloudware-fiscal-integration-foundation/research.md",
      "expectedRevision": 2,
      "expectedDigest": "sha256:760d4b160d36d0b99e25adb2bf73edad5602f547b3c375ecc03fd05245c499a5",
      "read": true,
      "observedRevision": 2,
      "identityValidated": false,
      "reason": "The bounded JSON was read completely and its revision, change name, and worktree matched, but no active approved tool exposed an independently observed SHA-256 digest."
    },
    "preproposalRevision2": {
      "path": "D:/repos/gonsgarage/openspec/changes/cloudware-fiscal-integration-foundation/preproposal.md",
      "expectedRevision": 2,
      "expectedDigest": "sha256:067e27efaf0a9e31c8e2df89c6e3fb4200f158b2fee1ffe2173fd44833812d7b",
      "read": true,
      "observedRevision": 2,
      "identityValidated": false,
      "reason": "The bounded JSON was read completely and its revision, change name, and worktree matched, but no active approved tool exposed an independently observed SHA-256 digest."
    }
  },
  "sources": [
    {
      "id": "CW-SETUP",
      "publisher": "Cloudware",
      "url": "https://cloudware.gitbook.io/documentacao-api/setup.md",
      "versionOrDate": "GitBook page; publication date not exposed",
      "tool": "fetch_content",
      "request": "Direct official-page fetch and targeted extraction for credential acquisition and official OpenAPI links; batch responseId mu88d9gqovle4u urlIndex 0; targeted answer response ID not exposed.",
      "retrievedAt": null,
      "retrievalTimeNote": "The callable tool did not expose a UTC retrieval timestamp.",
      "excerpt": "No produto ... através do menu Empresa > Dados API ... deverá introduzir o nome e o e-mail do integrador ... Este irá receber um e-mail com um link temporário ... Link para visualizar e descarregar OpenAPI: v0 ... APIv0/1.0; v1 ... APIv1/1.0."
    },
    {
      "id": "CW-AUTH",
      "publisher": "Cloudware",
      "url": "https://cloudware.gitbook.io/documentacao-api/autenticacao.md",
      "versionOrDate": "Common authentication page for v0/v1; publication date not exposed",
      "tool": "fetch_content",
      "request": "Direct official-page fetch and targeted extraction for OAuth authorization, token exchange, refresh, and API headers; batch responseId mu88d9gqovle4u urlIndex 1; targeted answer response ID not exposed.",
      "retrievedAt": null,
      "retrievalTimeNote": "The callable tool did not expose a UTC retrieval timestamp.",
      "excerpt": "link temporário, de 72h ... GET ... OAUTH_URL/auth ... client_id ... redirect_uri ... response_type=code ... scope=commercial ... Content-Type: application/x-www-form-urlencoded ... Authorization: Basic ... grant_type=refresh_token ... novo access_token e novo refresh_token ... Authorization: Bearer <access_token>."
    },
    {
      "id": "CW-REQUESTS",
      "publisher": "Cloudware",
      "url": "https://cloudware.gitbook.io/documentacao-api/carateristicas-dos-pedidos.md",
      "versionOrDate": "GitBook page; publication date not exposed",
      "tool": "fetch_content",
      "request": "Direct official-page fetch and targeted extraction for licensing and request characteristics; batch responseId mu88d9gqovle4u urlIndex 2; targeted answer response ID not exposed.",
      "retrievedAt": null,
      "retrievalTimeNote": "The callable tool did not expose a UTC retrieval timestamp.",
      "excerpt": "A empresa de trabalho deverá ter uma licença de GC activa, excepto para os GET, que serão autorizados mesmo quando a licença expira."
    },
    {
      "id": "CW-V1-SALES",
      "publisher": "Cloudware",
      "url": "https://cloudware.gitbook.io/documentacao-api/api-v1/documentos-de-venda.md",
      "versionOrDate": "API v1 page; publication date not exposed",
      "tool": "fetch_content",
      "request": "Direct official-page fetch and targeted extraction for sales-document fields, lifecycle, PDF flag, and FR receipt behavior; batch responseId mu88da7ezno62b urlIndex 0; targeted answer response ID not exposed.",
      "retrievedAt": null,
      "retrievalTimeNote": "The callable tool did not expose a UTC retrieval timestamp.",
      "excerpt": "document_type: FT|FS|FR ... customer_tax_registration_number ... external_reference ... tax_code: NOR|INT|RED|ISE ... return_pdf ... Após a finalização ... a sua eliminação — assim como a sua alteração — deixa de ser possível ... podendo apenas ser anulado ... Para as faturas-recibos (FR), o recibo associado é criado automaticamente quando a FR é finalizada."
    },
    {
      "id": "CW-V1-OPENAPI",
      "publisher": "Cloudware on SwaggerHub",
      "url": "https://api.swaggerhub.com/apis/cloudware-deploy/APIv1/1.0",
      "versionOrDate": "OpenAPI 3.0.1; Cloudware API v1 Documentation version 1.0",
      "tool": "fetch_content",
      "request": "Raw direct fetch and targeted answer fetch of the GitBook-linked SwaggerHub API version; response IDs were not exposed by the callable tool.",
      "retrievedAt": null,
      "retrievalTimeNote": "The callable tool did not expose a UTC retrieval timestamp.",
      "excerpt": "Paths include /v1/commercial_sales_documents, /v1/commercial_sales_documents/{id}/finalize, /v1/commercial_sales_documents/{id}/void, and /v1/commercial_sales_receipts; document-process-flags include finalize and return_pdf; sales-document responses contain url with example https://app.cloudware.pt/path_to_file."
    },
    {
      "id": "CW-V1-RECEIPTS",
      "publisher": "Cloudware",
      "url": "https://cloudware.gitbook.io/documentacao-api/api-v1/recibos.md",
      "versionOrDate": "API v1 page; publication date not exposed",
      "tool": "fetch_content",
      "request": "Direct official-page fetch and targeted extraction for receipt creation, receivable references, and partial settlement; batch responseId mu88da7ezno62b urlIndex 1; targeted answer response ID not exposed.",
      "retrievedAt": null,
      "retrievalTimeNote": "The callable tool did not expose a UTC retrieval timestamp.",
      "excerpt": "POST ... /v1/commercial_sales_receipts ... receivable_type: Document|DocumentLine ... received_value ... não é necessário receber a totalidade do documento, pode fazer-se um recebimento parcial ... É na linha do recibo que se indica qual o documento (FT, ou outro) que foi pago."
    },
    {
      "id": "CW-V0-AT",
      "publisher": "Cloudware",
      "url": "https://cloudware.gitbook.io/documentacao-api/api-v0/comunicacao-de-documentos-a-at.md",
      "versionOrDate": "API v0 page; publication date not exposed",
      "tool": "fetch_content",
      "request": "Direct official-page fetch and targeted extraction for AT communication endpoint, credentials, document categories, and status fields; batch responseId mu88da7ezno62b urlIndex 2; targeted answer response ID not exposed.",
      "retrievedAt": null,
      "retrievalTimeNote": "The callable tool did not expose a UTC retrieval timestamp.",
      "excerpt": "pedido POST ... send_document_at_webservice ... entity_password ... entity_username ... communication_status ... communication_code ... communication_message."
    },
    {
      "id": "CW-POS",
      "publisher": "Cloudware",
      "url": "https://www.cloudware.pt/pos/",
      "versionOrDate": "Product page; publication date not exposed",
      "tool": "fetch_content",
      "request": "Direct official-site fetch and targeted extraction for product, pricing, cloud, integration, and trial claims; batch responseId mu88dbkru4bh7s urlIndex 0; targeted answer response ID not exposed.",
      "retrievedAt": null,
      "retrievalTimeNote": "The callable tool did not expose a UTC retrieval timestamp.",
      "excerpt": "Fature em segundos desde 8€/mês ... programa de faturação online ... API disponível para integrações personalizadas ... experimente grátis durante 30 dias."
    },
    {
      "id": "CW-INDEX",
      "publisher": "Cloudware",
      "url": "https://cloudware.gitbook.io/documentacao-api/",
      "versionOrDate": "Documentation index; publication date not exposed",
      "tool": "fetch_content",
      "request": "Direct official-index fetch and targeted extraction for exposed documentation versions and absence checks; batch responseId mu88dbkru4bh7s urlIndex 1; targeted answer response ID not exposed.",
      "retrievedAt": null,
      "retrievalTimeNote": "The callable tool did not expose a UTC retrieval timestamp.",
      "excerpt": "The fetched navigation identifies Introdução and API-v1 / Introdução à API v1 and mentions API versions v1 and v0; the page itself contains no sandbox, idempotency, webhook, rate-limit, retry, automatic-v1-AT, or PDF-lifetime statement."
    }
  ],
  "validatedClaims": [
    {
      "id": "C1",
      "claim": "Cloudware officially documents administrator-initiated credential provisioning through Empresa > Dados API and a temporary credential link whose authentication page specifies 72 hours.",
      "sourceIds": [
        "CW-SETUP",
        "CW-AUTH"
      ]
    },
    {
      "id": "C2",
      "claim": "Cloudware officially documents an OAuth authorization-code flow with commercial scope, Basic client authentication at the token endpoint, refresh-token renewal, and Bearer authorization for API calls.",
      "sourceIds": [
        "CW-AUTH"
      ]
    },
    {
      "id": "C3",
      "claim": "Cloudware API v1 version 1.0 models FT, FS, and FR sales documents with customer, series, line, VAT/exemption, external-reference, finalize, void, and PDF-return concepts.",
      "sourceIds": [
        "CW-V1-SALES",
        "CW-V1-OPENAPI"
      ]
    },
    {
      "id": "C4",
      "claim": "Cloudware documents that finalizing an FR creates its receipt automatically, while a separate v1 receipt can settle an FT or another referenced receivable, including partially.",
      "sourceIds": [
        "CW-V1-SALES",
        "CW-V1-RECEIPTS",
        "CW-V1-OPENAPI"
      ]
    },
    {
      "id": "C5",
      "claim": "The inspected official documentation provides a dedicated AT communication endpoint under API v0 but does not establish whether v1 sales-document finalization communicates automatically to AT/e-Fatura.",
      "sourceIds": [
        "CW-V0-AT",
        "CW-V1-OPENAPI"
      ]
    },
    {
      "id": "C6",
      "claim": "The inspected official corpus does not affirmatively document an API sandbox/test company, idempotency, webhooks, rate limits, retry or ambiguous-outcome guidance, or PDF URL lifetime/authentication/retention; these remain unknown rather than disproven features.",
      "sourceIds": [
        "CW-AUTH",
        "CW-REQUESTS",
        "CW-V1-SALES",
        "CW-V1-OPENAPI",
        "CW-POS",
        "CW-INDEX"
      ]
    },
    {
      "id": "C7",
      "claim": "Cloudware's request documentation says non-GET API requests require an active GC licence, whereas GET requests remain authorized after licence expiry.",
      "sourceIds": [
        "CW-REQUESTS"
      ]
    },
    {
      "id": "C8",
      "claim": "Cloudware's official POS page advertises online operation, custom API integrations, pricing from 8 EUR per month, and a 30-day product trial, without establishing API sandbox entitlement or support terms.",
      "sourceIds": [
        "CW-POS"
      ]
    }
  ],
  "failedCalls": [],
  "unknownsRetained": [
    "PKCE, OAuth state, revocation, and any additional scopes are not established by the inspected official pages.",
    "API sandbox/test-company availability and credential provisioning for non-production use remain unconfirmed.",
    "Provider idempotency, correlation-key lookup, webhook support, rate limits, retry guidance, and ambiguous-outcome handling remain unconfirmed.",
    "PDF URL lifetime, access control, authenticity, retention, and archival authority remain unconfirmed.",
    "Automatic AT/e-Fatura communication behavior for v1 finalization remains unconfirmed.",
    "Credential-dependent response shapes, error taxonomy, and real account behavior remain untested.",
    "Exact UTC retrieval timestamps were not exposed by the callable fetch tool.",
    "Bounded locator digests could not be independently recomputed with the active approved tool set."
  ],
  "productDecisions": "pending",
  "proposal_ready": false,
  "proposalReadinessBlockers": [
    "The bounded input locator digests could not be independently verified with the active approved tool set.",
    "Cloudware credential- and support-dependent unknowns are retained for adapter design and real-provider enablement.",
    "Product decisions remain pending by parent instruction."
  ],
  "persistence": {
    "store": "openspec",
    "path": "D:/repos/gonsgarage/openspec/changes/cloudware-fiscal-integration-foundation/research.md",
    "writeTargetRevision": 3,
    "postWriteReadback": "required-before-reporting"
  }
}
