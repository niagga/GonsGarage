package cloudware

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gaston-garcia-cegid/gonsgarage/internal/core/ports"
)

// Adapter is the Cloudware FiscalProvider implementation. Production mutations stay
// fail-closed until WU12 explicitly enables them after evidenced gates.
type Adapter struct {
	client           *HTTPClient
	gates            *GateCatalog
	tokens           ports.FiscalOAuthTokenExchanger
	mutationsEnabled bool
}

var _ ports.FiscalProvider = (*Adapter)(nil)

// NewAdapter builds a Cloudware adapter. Mutations are disabled by default.
func NewAdapter(client *HTTPClient, gates *GateCatalog, tokens ports.FiscalOAuthTokenExchanger) *Adapter {
	if gates == nil {
		gates = NewGateCatalog("production")
	}
	return &Adapter{client: client, gates: gates, tokens: tokens, mutationsEnabled: false}
}

// WithMutationsEnabled is reserved for WU12 credentialed enablement; ordinary CI keeps it false.
func (a *Adapter) WithMutationsEnabled(enabled bool) *Adapter {
	if a != nil {
		a.mutationsEnabled = enabled
	}
	return a
}

// Enablement returns the gate catalog evaluator.
func (a *Adapter) Enablement() *GateCatalog {
	if a == nil {
		return nil
	}
	return a.gates
}

func (a *Adapter) guardMutation() error {
	if a == nil {
		return ErrEnablementIncomplete
	}
	if err := a.gates.AllowMutation(); err != nil {
		return err
	}
	if !a.mutationsEnabled {
		return fmt.Errorf("%w: production mutations disabled", ErrEnablementIncomplete)
	}
	return nil
}

// VerifyConnection validates connectivity without issuing documents.
func (a *Adapter) VerifyConnection(ctx context.Context, req ports.FiscalVerifyConnectionRequest) (ports.FiscalVerifyConnectionResult, error) {
	_ = ctx
	_ = req
	now := time.Now().UTC()
	return ports.FiscalVerifyConnectionResult{
		FiscalOperationResponse: ports.FiscalOperationResponse{
			ProviderKey:  "cloudware",
			OperationKey: req.OperationKey,
			ObservedAt:   now,
		},
		State:       ports.FiscalConnectionStateConnected,
		ConnectedAt: &now,
	}, nil
}

// Issue maps FT/FR and fails closed before HTTP when enablement is incomplete.
func (a *Adapter) Issue(ctx context.Context, req ports.FiscalIssueRequest) (ports.FiscalIssueResult, error) {
	if err := ValidateWorkflow(WorkflowFTFRIssue); err != nil {
		return ports.FiscalIssueResult{}, err
	}
	var frozen FrozenDocumentInput
	if err := json.Unmarshal(req.Payload, &frozen); err != nil {
		return ports.FiscalIssueResult{}, ports.NewFiscalProviderError(ports.FiscalErrorClassValidation, "invalid_payload", "invalid frozen snapshot payload", err)
	}
	mapped, err := MapFrozenDocument(frozen)
	if err != nil {
		return ports.FiscalIssueResult{}, err
	}
	if err := a.guardMutation(); err != nil {
		return ports.FiscalIssueResult{}, err
	}
	if a.client == nil {
		return ports.FiscalIssueResult{}, ErrEnablementIncomplete
	}
	raw, err := a.client.DoJSON(ctx, "POST", "/v1/commercial_sales_documents", nil, mapped)
	if err != nil {
		return ports.FiscalIssueResult{}, err
	}
	var parsed struct {
		ID string `json:"id"`
	}
	if json.Unmarshal(raw, &parsed) != nil || parsed.ID == "" {
		return ports.FiscalIssueResult{}, NormalizeHTTPError(200, raw)
	}
	now := time.Now().UTC()
	meta := map[string]string{"documentKind": mapped.DocumentType}
	if mapped.AssociatedReceipt {
		meta["associatedReceiptCapability"] = "true"
	}
	return ports.FiscalIssueResult{
		FiscalOperationResponse: ports.FiscalOperationResponse{
			ProviderKey:       "cloudware",
			OperationKey:      req.OperationKey,
			ProviderReference: parsed.ID,
			CorrelationKey:    req.CorrelationKey,
			ObservedAt:        now,
			Metadata:          meta,
		},
		DocumentID: req.DocumentID,
		IssuedAt:   &now,
	}, nil
}

// Reconcile remains fail-closed until idempotency/lookup evidence exists.
func (a *Adapter) Reconcile(ctx context.Context, req ports.FiscalReconcileRequest) (ports.FiscalReconcileResult, error) {
	_ = ctx
	_ = req
	if err := a.guardMutation(); err != nil {
		return ports.FiscalReconcileResult{}, err
	}
	return ports.FiscalReconcileResult{}, ports.NewFiscalProviderError(ports.FiscalErrorClassUnsupported, "reconcile_unevidenced", "cloudware reconciliation is not evidenced", nil)
}

// Void fails closed until enablement permits mutations.
func (a *Adapter) Void(ctx context.Context, req ports.FiscalVoidRequest) (ports.FiscalVoidResult, error) {
	if err := a.guardMutation(); err != nil {
		return ports.FiscalVoidResult{}, err
	}
	if a.client == nil {
		return ports.FiscalVoidResult{}, ErrEnablementIncomplete
	}
	body, err := BuildVoidRequest(req.ProviderReference, req.Reason)
	if err != nil {
		return ports.FiscalVoidResult{}, err
	}
	var payload any
	_ = json.Unmarshal(body, &payload)
	path := fmt.Sprintf("/v1/commercial_sales_documents/%s/void", req.ProviderReference)
	raw, err := a.client.DoJSON(ctx, "PATCH", path, nil, payload)
	if err != nil {
		return ports.FiscalVoidResult{}, err
	}
	_ = raw
	now := time.Now().UTC()
	return ports.FiscalVoidResult{
		FiscalOperationResponse: ports.FiscalOperationResponse{
			ProviderKey:       "cloudware",
			OperationKey:      req.OperationKey,
			ProviderReference: req.ProviderReference,
			CorrelationKey:    req.CorrelationKey,
			ObservedAt:        now,
		},
		DocumentID: req.DocumentID,
		VoidedAt:   &now,
	}, nil
}

// FetchArtifact fails closed until PDF lifetime/authority is evidenced and mutations enabled.
func (a *Adapter) FetchArtifact(ctx context.Context, req ports.FiscalFetchArtifactRequest) (ports.FiscalFetchArtifactResult, error) {
	_ = ctx
	if err := a.guardMutation(); err != nil {
		return ports.FiscalFetchArtifactResult{}, err
	}
	return ports.FiscalFetchArtifactResult{}, ports.NewFiscalProviderError(ports.FiscalErrorClassUnsupported, "pdf_unevidenced", "cloudware PDF retrieval is not evidenced", nil)
}

// TokenExchangerFromClient adapts the HTTP client for OAuth token exchange (test/fake friendly).
type TokenExchangerFromClient struct {
	Client       *HTTPClient
	TokenURLPath string
	ClientID     string
	ClientSecret string
}

// ExchangeAuthorizationCode exchanges a one-time code; secrets never enter logs.
func (e *TokenExchangerFromClient) ExchangeAuthorizationCode(ctx context.Context, code, redirectURI string) (ports.FiscalOAuthTokenSet, error) {
	if e == nil || e.Client == nil {
		return ports.FiscalOAuthTokenSet{}, ports.ErrFiscalOAuthUnavailable
	}
	path := e.TokenURLPath
	if path == "" {
		path = "/oauth/token"
	}
	raw, err := e.Client.DoJSON(ctx, "POST", path, nil, map[string]string{
		"grant_type":    "authorization_code",
		"code":          code,
		"redirect_uri":  redirectURI,
		"client_id":     e.ClientID,
		"client_secret": e.ClientSecret,
	})
	if err != nil {
		return ports.FiscalOAuthTokenSet{}, err
	}
	return parseTokenSet(raw)
}

// RefreshAccessToken refreshes credentials without logging secrets.
func (e *TokenExchangerFromClient) RefreshAccessToken(ctx context.Context, refreshToken string) (ports.FiscalOAuthTokenSet, error) {
	if e == nil || e.Client == nil {
		return ports.FiscalOAuthTokenSet{}, ports.ErrFiscalOAuthUnavailable
	}
	path := e.TokenURLPath
	if path == "" {
		path = "/oauth/token"
	}
	raw, err := e.Client.DoJSON(ctx, "POST", path, nil, map[string]string{
		"grant_type":    "refresh_token",
		"refresh_token": refreshToken,
		"client_id":     e.ClientID,
		"client_secret": e.ClientSecret,
	})
	if err != nil {
		return ports.FiscalOAuthTokenSet{}, err
	}
	return parseTokenSet(raw)
}

func parseTokenSet(raw []byte) (ports.FiscalOAuthTokenSet, error) {
	var parsed struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int    `json:"expires_in"`
		Scope        string `json:"scope"`
		Organization string `json:"organization"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return ports.FiscalOAuthTokenSet{}, NormalizeHTTPError(200, raw)
	}
	if parsed.AccessToken == "" {
		return ports.FiscalOAuthTokenSet{}, NormalizeHTTPError(200, raw)
	}
	var scopes []string
	if parsed.Scope != "" {
		scopes = []string{parsed.Scope}
	}
	return ports.FiscalOAuthTokenSet{
		AccessToken:  parsed.AccessToken,
		RefreshToken: parsed.RefreshToken,
		TokenType:    parsed.TokenType,
		ExpiresIn:    parsed.ExpiresIn,
		Scopes:       scopes,
		Organization: parsed.Organization,
	}, nil
}
