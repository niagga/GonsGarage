package cloudware

import (
	"errors"
	"fmt"
	"strings"
)

var (
	// ErrUnsupportedDocumentKind rejects non FT/FR kinds.
	ErrUnsupportedDocumentKind = errors.New("unsupported cloudware document kind")
	// ErrUnsupportedWorkflow rejects receipt/FS/correction workflows.
	ErrUnsupportedWorkflow = errors.New("unsupported cloudware workflow")
	// ErrEnablementIncomplete blocks mutations when gates are incomplete.
	ErrEnablementIncomplete = errors.New("cloudware enablement incomplete")
	// ErrGateRationaleRequired requires rationale for not_applicable gates.
	ErrGateRationaleRequired = errors.New("gate not_applicable requires rationale")
	// ErrGateNotOptional rejects not_applicable on mandatory gates.
	ErrGateNotOptional = errors.New("gate cannot be marked not_applicable")
	// ErrBaseURLNotAllowed rejects non-allowlisted hosts.
	ErrBaseURLNotAllowed = errors.New("cloudware base URL is not allowlisted")
	// ErrResponseTooLarge rejects oversized provider bodies.
	ErrResponseTooLarge = errors.New("cloudware response exceeds limit")
)

// Workflow identifies first-release Cloudware workflows.
type Workflow string

const (
	WorkflowFTFRIssue           Workflow = "ft_fr_issue"
	WorkflowStandaloneReceipt   Workflow = "standalone_receipt"
	WorkflowPartialReceipt      Workflow = "partial_receipt"
	WorkflowAutomatedCorrection Workflow = "automated_correction"
)

// ValidateWorkflow accepts only evidenced first-release workflows.
func ValidateWorkflow(w Workflow) error {
	switch w {
	case WorkflowFTFRIssue:
		return nil
	case WorkflowStandaloneReceipt, WorkflowPartialReceipt, WorkflowAutomatedCorrection:
		return fmt.Errorf("%w: %s", ErrUnsupportedWorkflow, w)
	default:
		return fmt.Errorf("%w: %s", ErrUnsupportedWorkflow, w)
	}
}

// Gate keys seeded by migration 011 and owned by the readiness catalog.
const (
	GateSandboxTestCompany           = "sandbox_test_company"
	GateOAuthSecurity                = "oauth_security"
	GateIdempotencyLookup            = "idempotency_lookup"
	GateWebhooks                     = "webhooks"
	GateRateLimitsRetry              = "rate_limits_retry"
	GatePDFLifetimeAuthority         = "pdf_lifetime_authority"
	GateAutomaticATEFatura           = "automatic_at_efatura"
	GateCredentialedResponseBehavior = "credentialed_response_behavior"
)

// GateStatus mirrors fiscal_enablement_gates.status.
type GateStatus string

const (
	GatePending       GateStatus = "pending"
	GateSatisfied     GateStatus = "satisfied"
	GateNotApplicable GateStatus = "not_applicable"
)

// GateRecord is an enablement gate evaluation input.
type GateRecord struct {
	Key           string
	Status        GateStatus
	EvidenceRef   string
	DecisionRef   string
	AcceptanceRef string
	Rationale     string
}

// GateView is the safe readiness projection for a single gate.
type GateView struct {
	Key       string
	Status    GateStatus
	Guidance  string
	Rationale string
}

// ReadinessReport summarizes Cloudware production readiness without secrets.
type ReadinessReport struct {
	Ready                 bool
	Gates                 []GateView
	ATCommunicationStatus string
}

// GateCatalog evaluates Cloudware enablement gates.
type GateCatalog struct {
	environment string
	gates       map[string]GateRecord
}

// CloudwareEnablementEvaluator is the design-named gate evaluator.
type CloudwareEnablementEvaluator = GateCatalog

// MandatoryGateKeys returns gates that must be satisfied (not not_applicable).
func MandatoryGateKeys() []string {
	return []string{
		GateSandboxTestCompany,
		GateOAuthSecurity,
		GateIdempotencyLookup,
		GateRateLimitsRetry,
		GatePDFLifetimeAuthority,
		GateAutomaticATEFatura,
		GateCredentialedResponseBehavior,
	}
}

// OptionalGateKeys may be not_applicable with rationale.
func OptionalGateKeys() []string {
	return []string{GateWebhooks}
}

// NewGateCatalog builds a pending catalog for the environment.
func NewGateCatalog(environment string) *GateCatalog {
	c := &GateCatalog{
		environment: strings.TrimSpace(environment),
		gates:       make(map[string]GateRecord),
	}
	for _, key := range MandatoryGateKeys() {
		c.gates[key] = GateRecord{Key: key, Status: GatePending}
	}
	c.gates[GateWebhooks] = GateRecord{Key: GateWebhooks, Status: GatePending}
	return c
}

// Set updates a gate. not_applicable requires rationale and is only valid for optional gates.
func (c *GateCatalog) Set(key string, status GateStatus, evidence, decision, acceptance, rationale string) error {
	if c == nil {
		return ErrEnablementIncomplete
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return fmt.Errorf("gate key is required")
	}
	if status == GateNotApplicable {
		if strings.TrimSpace(rationale) == "" {
			return ErrGateRationaleRequired
		}
		if !isOptionalGate(key) {
			return ErrGateNotOptional
		}
	}
	c.gates[key] = GateRecord{
		Key:           key,
		Status:        status,
		EvidenceRef:   strings.TrimSpace(evidence),
		DecisionRef:   strings.TrimSpace(decision),
		AcceptanceRef: strings.TrimSpace(acceptance),
		Rationale:     strings.TrimSpace(rationale),
	}
	return nil
}

func isOptionalGate(key string) bool {
	for _, optional := range OptionalGateKeys() {
		if optional == key {
			return true
		}
	}
	return false
}

// IncompleteMandatory returns mandatory gates that are not satisfied.
func (c *GateCatalog) IncompleteMandatory() []string {
	if c == nil {
		return append([]string(nil), MandatoryGateKeys()...)
	}
	incomplete := make([]string, 0)
	for _, key := range MandatoryGateKeys() {
		rec, ok := c.gates[key]
		if !ok || rec.Status != GateSatisfied {
			incomplete = append(incomplete, key)
			continue
		}
		if rec.EvidenceRef == "" || rec.DecisionRef == "" || rec.AcceptanceRef == "" {
			incomplete = append(incomplete, key)
		}
	}
	if rec, ok := c.gates[GateWebhooks]; ok {
		switch rec.Status {
		case GateSatisfied:
			if rec.EvidenceRef == "" || rec.DecisionRef == "" || rec.AcceptanceRef == "" {
				// optional but if claimed satisfied still needs evidence
				incomplete = append(incomplete, GateWebhooks)
			}
		case GateNotApplicable:
			if strings.TrimSpace(rec.Rationale) == "" {
				incomplete = append(incomplete, GateWebhooks)
			}
		case GatePending:
			incomplete = append(incomplete, GateWebhooks)
		}
	} else {
		incomplete = append(incomplete, GateWebhooks)
	}
	return incomplete
}

// AllowMutation returns nil only when every applicable gate is complete.
func (c *GateCatalog) AllowMutation() error {
	incomplete := c.IncompleteMandatory()
	if len(incomplete) == 0 {
		return nil
	}
	return fmt.Errorf("%w: %s", ErrEnablementIncomplete, strings.Join(incomplete, ","))
}

// Readiness builds a secret-free readiness report.
func (c *GateCatalog) Readiness() ReadinessReport {
	report := ReadinessReport{Ready: false}
	if c == nil {
		return report
	}
	keys := append(MandatoryGateKeys(), OptionalGateKeys()...)
	seen := make(map[string]bool, len(keys))
	for _, key := range keys {
		if seen[key] {
			continue
		}
		seen[key] = true
		rec := c.gates[key]
		view := GateView{
			Key:       key,
			Status:    rec.Status,
			Rationale: rec.Rationale,
			Guidance:  gateGuidance(key, rec),
		}
		if view.Status == "" {
			view.Status = GatePending
			view.Guidance = gateGuidance(key, GateRecord{Key: key, Status: GatePending})
		}
		report.Gates = append(report.Gates, view)
	}
	report.Ready = len(c.IncompleteMandatory()) == 0
	if at, ok := c.gates[GateAutomaticATEFatura]; ok && at.Status == GateSatisfied {
		report.ATCommunicationStatus = "evidenced"
	}
	return report
}

func gateGuidance(key string, rec GateRecord) string {
	if rec.Status == GateSatisfied {
		return "Gate satisfied with recorded evidence."
	}
	if rec.Status == GateNotApplicable {
		return "Gate marked not applicable."
	}
	switch key {
	case GateSandboxTestCompany:
		return "Provision an approved Cloudware test company or controlled validation account."
	case GateOAuthSecurity:
		return "Complete OAuth redirect, state, PKCE decision, and reconnect evidence."
	case GateIdempotencyLookup:
		return "Approve a safe idempotency and ambiguous-outcome lookup strategy."
	case GateWebhooks:
		return "Record webhook non-use rationale or evidenced webhook contract."
	case GateRateLimitsRetry:
		return "Obtain provider rate-limit and retry guidance before enabling mutations."
	case GatePDFLifetimeAuthority:
		return "Verify PDF authentication, lifetime, and archival retrieval behavior."
	case GateAutomaticATEFatura:
		return "Do not claim AT/e-Fatura until v1 communication is evidenced."
	case GateCredentialedResponseBehavior:
		return "Validate FT/FR, void, and active-license behavior with credentials."
	default:
		return "Complete enablement evidence before production mutations."
	}
}
