package mock

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gaston-garcia-cegid/gonsgarage/internal/core/ports"
	"github.com/google/uuid"
)

// Options configures the deterministic non-production mock provider.
type Options struct {
	AppEnv   string
	Registry ScenarioRegistry
	Store    OperationStore
}

// Provider is the deterministic mock FiscalProvider.
type Provider struct {
	registry ScenarioRegistry
	store    OperationStore
	appEnv   string
}

var _ ports.FiscalProvider = (*Provider)(nil)

var deterministicObservedAt = time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)

// NewProvider constructs a mock provider or fails closed in production.
func NewProvider(opts Options) (*Provider, error) {
	if err := AssertMockAllowed(opts.AppEnv); err != nil {
		return nil, err
	}
	if err := AssertMockStoreAllowed(opts.AppEnv); err != nil {
		return nil, err
	}
	if opts.Registry == nil {
		opts.Registry = NewScenarioRegistry(nil)
	}
	if opts.Store == nil {
		opts.Store = NewMemoryOperationStore()
	}
	return &Provider{registry: opts.Registry, store: opts.Store, appEnv: opts.AppEnv}, nil
}

// MustNewProvider panics when mock construction is forbidden.
func MustNewProvider(opts Options) *Provider {
	provider, err := NewProvider(opts)
	if err != nil {
		panic(err)
	}
	return provider
}

func (p *Provider) scenarioFor(operationKey string) Scenario {
	if scenario, ok := p.registry.Lookup(operationKey); ok {
		return scenario
	}
	return ScenarioSuccess
}

func (p *Provider) VerifyConnection(ctx context.Context, req ports.FiscalVerifyConnectionRequest) (ports.FiscalVerifyConnectionResult, error) {
	_ = ctx
	scenario := p.scenarioFor(req.OperationKey)
	observedAt := deterministicObservedAt
	base := ports.FiscalOperationResponse{
		ProviderKey:       req.ProviderKey,
		OperationKey:      req.OperationKey,
		ProviderReference: mockProviderReference(req.OperationKey, "verify"),
		CorrelationKey:    req.CorrelationKey,
		ObservedAt:        observedAt,
		Metadata:          map[string]string{"scenario": string(scenario), "provider": "mock"},
	}
	switch scenario {
	case ScenarioExpiredConnection:
		return ports.FiscalVerifyConnectionResult{}, ports.NewFiscalProviderError(
			ports.FiscalErrorClassUnauthorized, "connection_expired", "mock fiscal connection is expired", nil,
		)
	case ScenarioTransient:
		return ports.FiscalVerifyConnectionResult{}, ports.NewFiscalProviderError(
			ports.FiscalErrorClassTransient, "verify_transient", "mock provider transient verification failure", nil,
		)
	case ScenarioRateLimit:
		return ports.FiscalVerifyConnectionResult{}, ports.NewFiscalProviderError(
			ports.FiscalErrorClassRateLimit, "verify_rate_limit", "mock provider rate limited verification", nil,
		)
	default:
		expiresAt := observedAt.Add(30 * 24 * time.Hour)
		connectedAt := observedAt
		return ports.FiscalVerifyConnectionResult{
			FiscalOperationResponse: base,
			State:                   ports.FiscalConnectionStateConnected,
			GrantedScopes:           []string{"issue", "reconcile", "void", "fetch", "verify"},
			AccessExpiresAt:         &expiresAt,
			ConnectedAt:             &connectedAt,
			OrganizationRef:         "MOCK-ORG",
		}, nil
	}
}

func (p *Provider) Issue(ctx context.Context, req ports.FiscalIssueRequest) (ports.FiscalIssueResult, error) {
	scenario := p.scenarioFor(req.OperationKey)
	canonical := canonicalSHA256(req.Payload)
	documentKind := documentKindFrom(req)

	switch scenario {
	case ScenarioValidation:
		return ports.FiscalIssueResult{}, ports.NewFiscalProviderError(
			ports.FiscalErrorClassValidation, "issue_validation", "mock provider rejected invalid fiscal input", nil,
		)
	case ScenarioExpiredConnection:
		return ports.FiscalIssueResult{}, ports.NewFiscalProviderError(
			ports.FiscalErrorClassUnauthorized, "connection_expired", "mock fiscal connection is expired", nil,
		)
	case ScenarioTransient:
		return ports.FiscalIssueResult{}, ports.NewFiscalProviderError(
			ports.FiscalErrorClassTransient, "issue_transient", "mock provider transient issue failure", nil,
		)
	case ScenarioRateLimit:
		return ports.FiscalIssueResult{}, ports.NewFiscalProviderError(
			ports.FiscalErrorClassRateLimit, "issue_rate_limit", "mock provider rate limited issuance", nil,
		)
	}

	existing, err := p.store.Get(ctx, req.OperationKey)
	if err != nil {
		return ports.FiscalIssueResult{}, err
	}
	if existing != nil {
		if existing.CanonicalSHA256 != canonical {
			return ports.FiscalIssueResult{}, ports.NewFiscalProviderError(
				ports.FiscalErrorClassConflict, "canonical_conflict", ErrCanonicalConflict.Error(), ErrCanonicalConflict,
			)
		}
		return issueResultFromRecord(existing, req)
	}

	ref := mockProviderReference(req.OperationKey, canonical)
	artifactID := mockArtifactID(req.OperationKey, canonical, req.DocumentID)
	number := mockProviderNumber(ref)
	pdfBytes, pdfSHA := RenderMockPDF(req.OperationKey, canonical, documentKind)
	observedAt := deterministicObservedAt
	payload := IssueResultPayload{
		DocumentID:     req.DocumentID,
		ArtifactID:     artifactID,
		ProviderNumber: number,
		DocumentKind:   documentKind,
		Ambiguous:      scenario == ScenarioAmbiguous,
		ObservedAtUnix: observedAt.Unix(),
		CorrelationKey: req.CorrelationKey,
		ProviderKey:    req.ProviderKey,
		ConnectionID:   req.ConnectionID,
	}
	if scenario != ScenarioAmbiguous {
		payload.IssuedAtUnix = observedAt.Unix()
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return ports.FiscalIssueResult{}, err
	}
	record := &OperationRecord{
		OperationKey:      req.OperationKey,
		Scenario:          string(scenario),
		CanonicalSHA256:   canonical,
		ProviderReference: ref,
		Result:            raw,
		PDFSHA256:         pdfSHA,
		PDFBytes:          pdfBytes,
	}
	stored, err := p.store.PutIfAbsent(ctx, record)
	if err != nil {
		return ports.FiscalIssueResult{}, err
	}
	if stored.CanonicalSHA256 != canonical {
		return ports.FiscalIssueResult{}, ports.NewFiscalProviderError(
			ports.FiscalErrorClassConflict, "canonical_conflict", ErrCanonicalConflict.Error(), ErrCanonicalConflict,
		)
	}
	result, err := issueResultFromRecord(stored, req)
	if err != nil {
		return ports.FiscalIssueResult{}, err
	}
	if scenario == ScenarioAmbiguous {
		ambErr := ports.NewFiscalProviderError(
			ports.FiscalErrorClassAmbiguous, "issue_ambiguous", "mock provider returned an ambiguous issue result", nil,
		)
		ambErr.DefinitiveNonAcceptance = false
		ambErr.Retryable = false
		return result, ambErr
	}
	return result, nil
}

func (p *Provider) Reconcile(ctx context.Context, req ports.FiscalReconcileRequest) (ports.FiscalReconcileResult, error) {
	scenario := p.scenarioFor(req.OperationKey)
	if scenario == ScenarioTransient {
		return ports.FiscalReconcileResult{}, ports.NewFiscalProviderError(
			ports.FiscalErrorClassTransient, "reconcile_transient", "mock provider transient reconcile failure", nil,
		)
	}
	record, err := p.store.Get(ctx, req.OperationKey)
	if err != nil {
		return ports.FiscalReconcileResult{}, err
	}
	if record == nil && req.ProviderReference != "" {
		record, err = p.store.GetByProviderReference(ctx, req.ProviderReference)
		if err != nil {
			return ports.FiscalReconcileResult{}, err
		}
	}
	observedAt := deterministicObservedAt
	base := ports.FiscalOperationResponse{
		ProviderKey:    req.ProviderKey,
		OperationKey:   req.OperationKey,
		CorrelationKey: req.CorrelationKey,
		ObservedAt:     observedAt,
		Metadata:       map[string]string{"scenario": string(scenario), "provider": "mock"},
	}
	if record == nil {
		base.ProviderReference = req.ProviderReference
		return ports.FiscalReconcileResult{FiscalOperationResponse: base, DocumentID: req.DocumentID, Matched: false}, nil
	}
	base.ProviderReference = record.ProviderReference
	resolvedAt := observedAt
	return ports.FiscalReconcileResult{
		FiscalOperationResponse: base,
		DocumentID:              req.DocumentID,
		Matched:                 true,
		ResolvedAt:              &resolvedAt,
	}, nil
}

func (p *Provider) Void(ctx context.Context, req ports.FiscalVoidRequest) (ports.FiscalVoidResult, error) {
	scenario := p.scenarioFor(req.OperationKey)
	switch scenario {
	case ScenarioVoidRefused, ScenarioValidation:
		return ports.FiscalVoidResult{}, ports.NewFiscalProviderError(
			ports.FiscalErrorClassPermanent, "void_refused", "mock provider refused void", nil,
		)
	case ScenarioTransient:
		return ports.FiscalVoidResult{}, ports.NewFiscalProviderError(
			ports.FiscalErrorClassTransient, "void_transient", "mock provider transient void failure", nil,
		)
	case ScenarioRateLimit:
		return ports.FiscalVoidResult{}, ports.NewFiscalProviderError(
			ports.FiscalErrorClassRateLimit, "void_rate_limit", "mock provider rate limited void", nil,
		)
	}

	record, err := p.store.GetByProviderReference(ctx, req.ProviderReference)
	if err != nil {
		return ports.FiscalVoidResult{}, err
	}
	if record == nil {
		return ports.FiscalVoidResult{}, ports.NewFiscalProviderError(
			ports.FiscalErrorClassNotFound, "void_not_found", "mock issued document not found for void", nil,
		)
	}
	if scenario == ScenarioVoidPermitted || scenario == ScenarioSuccess {
		if err := p.store.MarkVoided(ctx, record.OperationKey); err != nil {
			return ports.FiscalVoidResult{}, err
		}
	} else {
		return ports.FiscalVoidResult{}, ports.NewFiscalProviderError(
			ports.FiscalErrorClassPermanent, "void_refused", "mock provider refused void", nil,
		)
	}
	voidedAt := deterministicObservedAt
	return ports.FiscalVoidResult{
		FiscalOperationResponse: ports.FiscalOperationResponse{
			ProviderKey:       req.ProviderKey,
			OperationKey:      req.OperationKey,
			ProviderReference: record.ProviderReference,
			CorrelationKey:    req.CorrelationKey,
			ObservedAt:        voidedAt,
			Metadata:          map[string]string{"scenario": string(scenario), "provider": "mock"},
		},
		DocumentID: req.DocumentID,
		VoidedAt:   &voidedAt,
	}, nil
}

func (p *Provider) FetchArtifact(ctx context.Context, req ports.FiscalFetchArtifactRequest) (ports.FiscalFetchArtifactResult, error) {
	record, err := p.store.Get(ctx, req.OperationKey)
	if err != nil {
		return ports.FiscalFetchArtifactResult{}, err
	}
	if record == nil {
		return ports.FiscalFetchArtifactResult{}, ports.NewFiscalProviderError(
			ports.FiscalErrorClassNotFound, "artifact_not_found", "mock artifact operation not found", nil,
		)
	}
	var payload IssueResultPayload
	if err := json.Unmarshal(record.Result, &payload); err != nil {
		return ports.FiscalFetchArtifactResult{}, err
	}
	if payload.ArtifactID != req.ArtifactID && req.ArtifactID != uuid.Nil {
		return ports.FiscalFetchArtifactResult{}, ports.NewFiscalProviderError(
			ports.FiscalErrorClassNotFound, "artifact_mismatch", "mock artifact id does not match stored issue", nil,
		)
	}
	content := record.PDFBytes
	sha := record.PDFSHA256
	if len(content) == 0 {
		content, sha = RenderMockPDF(record.OperationKey, record.CanonicalSHA256, payload.DocumentKind)
	}
	if record.PDFSHA256 != "" && sha != record.PDFSHA256 {
		return ports.FiscalFetchArtifactResult{}, fmt.Errorf("mock pdf checksum mismatch")
	}
	fetchedAt := deterministicObservedAt
	return ports.FiscalFetchArtifactResult{
		FiscalOperationResponse: ports.FiscalOperationResponse{
			ProviderKey:       req.ProviderKey,
			OperationKey:      req.OperationKey,
			ProviderReference: record.ProviderReference,
			CorrelationKey:    req.CorrelationKey,
			ObservedAt:        fetchedAt,
			Metadata:          map[string]string{"classification": "mock", "label": MockPDFLabel},
		},
		ArtifactID: payload.ArtifactID,
		Content:    append([]byte(nil), content...),
		MediaType:  "application/pdf",
		Sha256:     sha,
		FetchedAt:  &fetchedAt,
	}, nil
}

func documentKindFrom(req ports.FiscalIssueRequest) string {
	if kind := strings.TrimSpace(req.Metadata["documentKind"]); kind != "" {
		return strings.ToUpper(kind)
	}
	var body struct {
		Kind string `json:"kind"`
	}
	_ = json.Unmarshal(req.Payload, &body)
	if kind := strings.TrimSpace(body.Kind); kind != "" {
		return strings.ToUpper(kind)
	}
	return "FT"
}

func issueResultFromRecord(record *OperationRecord, req ports.FiscalIssueRequest) (ports.FiscalIssueResult, error) {
	var payload IssueResultPayload
	if err := json.Unmarshal(record.Result, &payload); err != nil {
		return ports.FiscalIssueResult{}, err
	}
	observedAt := time.Unix(payload.ObservedAtUnix, 0).UTC()
	if payload.ObservedAtUnix == 0 {
		observedAt = deterministicObservedAt
	}
	meta := map[string]string{
		"scenario":       record.Scenario,
		"provider":       "mock",
		"documentKind":   payload.DocumentKind,
		"classification": "mock",
	}
	if payload.DocumentKind == "FR" {
		meta["associatedReceiptCapability"] = "true"
	}
	result := ports.FiscalIssueResult{
		FiscalOperationResponse: ports.FiscalOperationResponse{
			ProviderKey:       firstNonEmpty(payload.ProviderKey, req.ProviderKey),
			OperationKey:      record.OperationKey,
			ProviderReference: record.ProviderReference,
			CorrelationKey:    firstNonEmpty(payload.CorrelationKey, req.CorrelationKey),
			ObservedAt:        observedAt,
			Metadata:          meta,
		},
		DocumentID:     payload.DocumentID,
		ArtifactID:     payload.ArtifactID,
		ProviderNumber: payload.ProviderNumber,
	}
	if payload.IssuedAtUnix > 0 && !payload.Ambiguous {
		issuedAt := time.Unix(payload.IssuedAtUnix, 0).UTC()
		result.IssuedAt = &issuedAt
	}
	return result, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
