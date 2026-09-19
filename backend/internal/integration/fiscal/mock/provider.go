package mock

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/gaston-garcia-cegid/gonsgarage/internal/core/ports"
	"github.com/google/uuid"
)

type Provider struct{}

var _ ports.FiscalProvider = (*Provider)(nil)

func NewProvider() *Provider {
	return &Provider{}
}

type scenarioKind string

const (
	scenarioSuccess   scenarioKind = "success"
	scenarioTransient scenarioKind = "transient"
	scenarioPermanent scenarioKind = "permanent"
	scenarioAmbiguous scenarioKind = "ambiguous"
)

var deterministicBaseTime = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

func (p *Provider) VerifyConnection(ctx context.Context, req ports.FiscalVerifyConnectionRequest) (ports.FiscalVerifyConnectionResult, error) {
	_ = ctx
	scenario := resolveScenario(req.FiscalOperationRequest, "verify")
	seed := stableSeed("verify", req.ConnectionID.String(), req.ProviderKey, req.OperationKey, scenarioString(scenario))
	observedAt := stableTime(seed)
	metadata := baseMetadata(req.FiscalOperationRequest, scenario, seed)

	switch scenario {
	case scenarioTransient:
		return ports.FiscalVerifyConnectionResult{}, ports.NewFiscalProviderError(ports.FiscalErrorClassTransient, "verify_transient", "mock provider transient verification failure", nil)
	case scenarioPermanent:
		return ports.FiscalVerifyConnectionResult{}, ports.NewFiscalProviderError(ports.FiscalErrorClassPermanent, "verify_permanent", "mock provider permanent verification failure", nil)
	case scenarioAmbiguous:
		connectedAt := observedAt.Add(-15 * time.Minute)
		actionRequired := ports.FiscalVerifyConnectionResult{
			FiscalOperationResponse: ports.FiscalOperationResponse{
				ProviderKey:       req.ProviderKey,
				OperationKey:      req.OperationKey,
				ProviderReference: stableReference("verify", seed),
				CorrelationKey:    req.CorrelationKey,
				ObservedAt:        observedAt,
				Metadata:          metadata,
			},
			State:           ports.FiscalConnectionStateActionNeeded,
			GrantedScopes:   defaultScopes(),
			ConnectedAt:     &connectedAt,
			OrganizationRef: stableReference("verify-org", seed),
		}
		return actionRequired, ports.NewFiscalProviderError(ports.FiscalErrorClassConflict, "verify_ambiguous", "mock provider returned an ambiguous verification result", nil)
	default:
		expiresAt := observedAt.Add(30 * 24 * time.Hour)
		connectedAt := observedAt
		return ports.FiscalVerifyConnectionResult{
			FiscalOperationResponse: ports.FiscalOperationResponse{
				ProviderKey:       req.ProviderKey,
				OperationKey:      req.OperationKey,
				ProviderReference: stableReference("verify", seed),
				CorrelationKey:    req.CorrelationKey,
				ObservedAt:        observedAt,
				Metadata:          metadata,
			},
			State:           ports.FiscalConnectionStateConnected,
			GrantedScopes:   defaultScopes(),
			AccessExpiresAt: &expiresAt,
			ConnectedAt:     &connectedAt,
			OrganizationRef: stableReference("organization", seed),
		}, nil
	}
}

func (p *Provider) Issue(ctx context.Context, req ports.FiscalIssueRequest) (ports.FiscalIssueResult, error) {
	_ = ctx
	scenario := resolveScenario(req.FiscalOperationRequest, "issue")
	seed := stableSeed("issue", req.ConnectionID.String(), req.ProviderKey, req.OperationKey, req.DocumentID.String(), scenarioString(scenario))
	observedAt := stableTime(seed)
	metadata := baseMetadata(req.FiscalOperationRequest, scenario, seed)
	artifactID := uuid.NewSHA1(uuid.NameSpaceOID, []byte("artifact:"+hex.EncodeToString(seed[:])+":"+req.DocumentID.String()))
	providerNumber := stableReference("number", seed)

	result := ports.FiscalIssueResult{
		FiscalOperationResponse: ports.FiscalOperationResponse{
			ProviderKey:       req.ProviderKey,
			OperationKey:      req.OperationKey,
			ProviderReference: stableReference("issue", seed),
			CorrelationKey:    req.CorrelationKey,
			ObservedAt:        observedAt,
			Metadata:          metadata,
		},
		DocumentID:     req.DocumentID,
		ArtifactID:     artifactID,
		ProviderNumber: providerNumber,
	}
	if scenario == scenarioTransient {
		return ports.FiscalIssueResult{}, ports.NewFiscalProviderError(ports.FiscalErrorClassTransient, "issue_transient", "mock provider transient issue failure", nil)
	}
	if scenario == scenarioPermanent {
		return ports.FiscalIssueResult{}, ports.NewFiscalProviderError(ports.FiscalErrorClassPermanent, "issue_permanent", "mock provider permanent issue failure", nil)
	}
	issuedAt := observedAt
	result.IssuedAt = &issuedAt
	if scenario == scenarioAmbiguous {
		return result, ports.NewFiscalProviderError(ports.FiscalErrorClassConflict, "issue_ambiguous", "mock provider returned an ambiguous issue result", nil)
	}
	return result, nil
}

func (p *Provider) Reconcile(ctx context.Context, req ports.FiscalReconcileRequest) (ports.FiscalReconcileResult, error) {
	_ = ctx
	scenario := resolveScenario(req.FiscalOperationRequest, "reconcile")
	seed := stableSeed("reconcile", req.ConnectionID.String(), req.ProviderKey, req.OperationKey, req.DocumentID.String(), req.ProviderReference, scenarioString(scenario))
	observedAt := stableTime(seed)
	metadata := baseMetadata(req.FiscalOperationRequest, scenario, seed)
	result := ports.FiscalReconcileResult{
		FiscalOperationResponse: ports.FiscalOperationResponse{
			ProviderKey:       req.ProviderKey,
			OperationKey:      req.OperationKey,
			ProviderReference: req.ProviderReference,
			CorrelationKey:    req.CorrelationKey,
			ObservedAt:        observedAt,
			Metadata:          metadata,
		},
		DocumentID: req.DocumentID,
		Matched:    true,
	}
	resolvedAt := observedAt
	result.ResolvedAt = &resolvedAt
	if scenario == scenarioTransient {
		return ports.FiscalReconcileResult{}, ports.NewFiscalProviderError(ports.FiscalErrorClassTransient, "reconcile_transient", "mock provider transient reconcile failure", nil)
	}
	if scenario == scenarioPermanent {
		return ports.FiscalReconcileResult{}, ports.NewFiscalProviderError(ports.FiscalErrorClassPermanent, "reconcile_permanent", "mock provider permanent reconcile failure", nil)
	}
	if scenario == scenarioAmbiguous {
		result.Matched = false
		return result, ports.NewFiscalProviderError(ports.FiscalErrorClassConflict, "reconcile_ambiguous", "mock provider returned an ambiguous reconciliation result", nil)
	}
	return result, nil
}

func (p *Provider) Void(ctx context.Context, req ports.FiscalVoidRequest) (ports.FiscalVoidResult, error) {
	_ = ctx
	scenario := resolveScenario(req.FiscalOperationRequest, "void")
	seed := stableSeed("void", req.ConnectionID.String(), req.ProviderKey, req.OperationKey, req.DocumentID.String(), req.ProviderReference, req.Reason, scenarioString(scenario))
	observedAt := stableTime(seed)
	metadata := baseMetadata(req.FiscalOperationRequest, scenario, seed)
	result := ports.FiscalVoidResult{
		FiscalOperationResponse: ports.FiscalOperationResponse{
			ProviderKey:       req.ProviderKey,
			OperationKey:      req.OperationKey,
			ProviderReference: req.ProviderReference,
			CorrelationKey:    req.CorrelationKey,
			ObservedAt:        observedAt,
			Metadata:          metadata,
		},
		DocumentID: req.DocumentID,
	}
	voidedAt := observedAt
	result.VoidedAt = &voidedAt
	if scenario == scenarioTransient {
		return ports.FiscalVoidResult{}, ports.NewFiscalProviderError(ports.FiscalErrorClassTransient, "void_transient", "mock provider transient void failure", nil)
	}
	if scenario == scenarioPermanent {
		return ports.FiscalVoidResult{}, ports.NewFiscalProviderError(ports.FiscalErrorClassPermanent, "void_permanent", "mock provider permanent void failure", nil)
	}
	if scenario == scenarioAmbiguous {
		return result, ports.NewFiscalProviderError(ports.FiscalErrorClassConflict, "void_ambiguous", "mock provider returned an ambiguous void result", nil)
	}
	return result, nil
}

func (p *Provider) FetchArtifact(ctx context.Context, req ports.FiscalFetchArtifactRequest) (ports.FiscalFetchArtifactResult, error) {
	_ = ctx
	scenario := resolveScenario(req.FiscalOperationRequest, "fetch")
	seed := stableSeed("fetch", req.ConnectionID.String(), req.ProviderKey, req.OperationKey, req.ArtifactID.String(), scenarioString(scenario))
	observedAt := stableTime(seed)
	metadata := baseMetadata(req.FiscalOperationRequest, scenario, seed)
	content := []byte(fmt.Sprintf("mock-fiscal-artifact:%s:%s:%s", req.ArtifactID.String(), req.OperationKey, hex.EncodeToString(seed[:8])))
	sum := sha256.Sum256(content)
	result := ports.FiscalFetchArtifactResult{
		FiscalOperationResponse: ports.FiscalOperationResponse{
			ProviderKey:       req.ProviderKey,
			OperationKey:      req.OperationKey,
			ProviderReference: stableReference("fetch", seed),
			CorrelationKey:    req.CorrelationKey,
			ObservedAt:        observedAt,
			Metadata:          metadata,
		},
		ArtifactID: req.ArtifactID,
		Content:    append([]byte(nil), content...),
		MediaType:  "application/octet-stream",
		Sha256:     hex.EncodeToString(sum[:]),
	}
	fetchedAt := observedAt
	result.FetchedAt = &fetchedAt
	if scenario == scenarioTransient {
		return ports.FiscalFetchArtifactResult{}, ports.NewFiscalProviderError(ports.FiscalErrorClassTransient, "fetch_transient", "mock provider transient fetch failure", nil)
	}
	if scenario == scenarioPermanent {
		return ports.FiscalFetchArtifactResult{}, ports.NewFiscalProviderError(ports.FiscalErrorClassPermanent, "fetch_permanent", "mock provider permanent fetch failure", nil)
	}
	if scenario == scenarioAmbiguous {
		return result, ports.NewFiscalProviderError(ports.FiscalErrorClassConflict, "fetch_ambiguous", "mock provider returned an ambiguous fetch result", nil)
	}
	return result, nil
}

func resolveScenario(req ports.FiscalOperationRequest, operation string) scenarioKind {
	candidates := []string{
		strings.TrimSpace(req.Metadata["scenario."+strings.ToLower(operation)]),
		strings.TrimSpace(req.Metadata["scenario"]),
		strings.TrimSpace(req.Metadata["outcome"]),
		strings.TrimSpace(req.Metadata["mode"]),
		strings.TrimSpace(req.Metadata["result"]),
		req.OperationKey,
	}
	for _, candidate := range candidates {
		if scenario := parseScenario(candidate); scenario != "" {
			return scenario
		}
	}
	return scenarioSuccess
}

func parseScenario(raw string) scenarioKind {
	raw = strings.ToLower(strings.TrimSpace(raw))
	if raw == "" {
		return ""
	}
	if strings.Contains(raw, ":") {
		parts := strings.Split(raw, ":")
		raw = strings.TrimSpace(parts[len(parts)-1])
	}
	switch raw {
	case "success", "ok", "pass", "normalized":
		return scenarioSuccess
	case "transient", "retryable", "temporary", "timeout":
		return scenarioTransient
	case "permanent", "fatal", "rejected", "failure":
		return scenarioPermanent
	case "ambiguous", "conflict", "partial", "indeterminate":
		return scenarioAmbiguous
	default:
		if strings.Contains(raw, "transient") {
			return scenarioTransient
		}
		if strings.Contains(raw, "permanent") || strings.Contains(raw, "fatal") {
			return scenarioPermanent
		}
		if strings.Contains(raw, "ambiguous") || strings.Contains(raw, "conflict") || strings.Contains(raw, "partial") {
			return scenarioAmbiguous
		}
		if strings.Contains(raw, "success") || strings.Contains(raw, "ok") || strings.Contains(raw, "pass") {
			return scenarioSuccess
		}
		return ""
	}
}

func scenarioString(s scenarioKind) string {
	if s == "" {
		return string(scenarioSuccess)
	}
	return string(s)
}

func stableSeed(parts ...string) []byte {
	hash := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return hash[:]
}

func stableTime(seed []byte) time.Time {
	if len(seed) < 8 {
		return deterministicBaseTime
	}
	seconds := binary.BigEndian.Uint64(seed[:8]) % 86400
	return deterministicBaseTime.Add(time.Duration(seconds) * time.Second)
}

func stableReference(prefix string, seed []byte) string {
	return fmt.Sprintf("%s-%s", prefix, hex.EncodeToString(seed[:6]))
}

func baseMetadata(req ports.FiscalOperationRequest, scenario scenarioKind, seed []byte) map[string]string {
	metadata := cloneMetadata(req.Metadata)
	metadata["scenario"] = scenarioString(scenario)
	metadata["operation"] = req.OperationKey
	metadata["connectionId"] = req.ConnectionID.String()
	metadata["providerKey"] = req.ProviderKey
	metadata["seed"] = hex.EncodeToString(seed[:8])
	return metadata
}

func cloneMetadata(values map[string]string) map[string]string {
	if values == nil {
		return map[string]string{}
	}
	out := make(map[string]string, len(values))
	for key, value := range values {
		out[key] = value
	}
	return out
}

func defaultScopes() []string {
	return []string{"issue", "reconcile", "void", "fetch", "verify"}
}
