package fiscal

import (
	"errors"
	"fmt"
	"strings"
)

var (
	// ErrUnsafeFiscalProductionConfig reports production composition that must fail closed.
	ErrUnsafeFiscalProductionConfig = errors.New("unsafe fiscal production configuration")
	// ErrFiscalWorkerDisabled reports the worker process was started without the enable switch.
	ErrFiscalWorkerDisabled = errors.New("fiscal worker is disabled")
	// ErrFiscalWorkerNotReady reports worker startup dependency failure.
	ErrFiscalWorkerNotReady = errors.New("fiscal worker is not ready")
)

// RuntimeConfig captures independent fiscal deployment switches.
type RuntimeConfig struct {
	AppEnv                    string
	FeatureEnabled            bool
	FinalizationEnabled       bool
	WorkerEnabled             bool
	ProviderKey               string
	CloudwareMutationsEnabled bool
	ArtifactBackend           string
	CredentialKeyConfigured   bool
	CredentialKeyVersion      string
	SchemaPresent             bool
	JWTSecretIsDefault        bool
}

// ParseRuntimeConfig loads fiscal switches from an env-style getter.
// Feature, finalization, worker, and Cloudware mutations default OFF.
func ParseRuntimeConfig(getenv func(string) string) RuntimeConfig {
	if getenv == nil {
		getenv = func(string) string { return "" }
	}
	appEnv := strings.TrimSpace(getenv("APP_ENV"))
	if appEnv == "" {
		appEnv = "development"
	}
	backend := strings.TrimSpace(getenv("FISCAL_ARTIFACT_BACKEND"))
	if backend == "" {
		backend = "local"
	}
	keyVersion := strings.TrimSpace(getenv("FISCAL_CREDENTIAL_KEY_VERSION"))
	if keyVersion == "" {
		keyVersion = "v1"
	}
	credConfigured := strings.TrimSpace(getenv("FISCAL_CREDENTIAL_KEY_B64")) != "" ||
		strings.TrimSpace(getenv("FISCAL_CREDENTIAL_KEY")) != ""
	jwt := strings.TrimSpace(getenv("JWT_SECRET"))
	return RuntimeConfig{
		AppEnv:                    appEnv,
		FeatureEnabled:            envTruthy(getenv("FISCAL_FEATURE_ENABLED")),
		FinalizationEnabled:       envTruthy(getenv("FISCAL_FINALIZATION_ENABLED")),
		WorkerEnabled:             envTruthy(getenv("FISCAL_WORKER_ENABLED")),
		ProviderKey:               strings.ToLower(strings.TrimSpace(getenv("FISCAL_PROVIDER"))),
		CloudwareMutationsEnabled: envTruthy(getenv("FISCAL_CLOUDWARE_MUTATIONS")),
		ArtifactBackend:           strings.ToLower(backend),
		CredentialKeyConfigured:   credConfigured,
		CredentialKeyVersion:      keyVersion,
		JWTSecretIsDefault:        jwt == "" || jwt == "your-super-secret-jwt-key" || strings.HasPrefix(jwt, "CHANGE_ME"),
	}
}

func envTruthy(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func (c RuntimeConfig) isProduction() bool {
	return strings.EqualFold(strings.TrimSpace(c.AppEnv), "production")
}

// ValidateProductionComposition fails closed for production unsafe defaults.
// Non-production environments return nil so mock/local development remains usable.
func (c RuntimeConfig) ValidateProductionComposition() error {
	if !c.isProduction() {
		return nil
	}
	if !c.FeatureEnabled {
		return nil
	}
	var issues []string
	if c.JWTSecretIsDefault {
		issues = append(issues, "default_jwt_secret")
	}
	if c.ProviderKey == "" || c.ProviderKey == "mock" {
		issues = append(issues, "mock_or_missing_provider")
	}
	if c.ArtifactBackend == "" || c.ArtifactBackend == "local" {
		issues = append(issues, "local_artifact_backend")
	}
	if !c.SchemaPresent {
		issues = append(issues, "missing_fiscal_schema")
	}
	if !c.CredentialKeyConfigured || strings.TrimSpace(c.CredentialKeyVersion) == "" {
		issues = append(issues, "missing_credential_keyring")
	}
	if len(issues) == 0 {
		return nil
	}
	return fmt.Errorf("%w: %s", ErrUnsafeFiscalProductionConfig, strings.Join(issues, ","))
}

// CoreAPIReady reports process readiness for /ready.
// fiscalProviderHealthy is accepted for call-site clarity but MUST NOT affect the result.
func CoreAPIReady(dbHealthy bool, fiscalProviderHealthy bool) bool {
	_ = fiscalProviderHealthy
	return dbHealthy
}

// WorkerReadinessInput is the worker process readiness probe input.
type WorkerReadinessInput struct {
	DBHealthy     bool
	SchemaPresent bool
	ConfigOK      bool
	WorkerEnabled bool
}

// EvaluateWorkerReadiness returns whether the worker may claim outbox work.
func EvaluateWorkerReadiness(in WorkerReadinessInput) (bool, []string) {
	var issues []string
	if !in.WorkerEnabled {
		issues = append(issues, "worker_disabled")
	}
	if !in.DBHealthy {
		issues = append(issues, "database_unavailable")
	}
	if !in.SchemaPresent {
		issues = append(issues, "schema_missing")
	}
	if !in.ConfigOK {
		issues = append(issues, "config_invalid")
	}
	return len(issues) == 0, issues
}

// DependencyStatusInput feeds the fiscal dependency status endpoint/metric.
type DependencyStatusInput struct {
	FeatureEnabled      bool
	FinalizationEnabled bool
	WorkerEnabled       bool
	ProviderKey         string
	ProviderHealthy     bool
	SchemaPresent       bool
}

// FiscalDependencyStatus is provider-neutral fiscal dependency health.
type FiscalDependencyStatus struct {
	FeatureEnabled      bool     `json:"featureEnabled"`
	FinalizationEnabled bool     `json:"finalizationEnabled"`
	WorkerEnabled       bool     `json:"workerEnabled"`
	ProviderKey         string   `json:"providerKey,omitempty"`
	ProviderHealthy     bool     `json:"providerHealthy"`
	SchemaPresent       bool     `json:"schemaPresent"`
	Issues              []string `json:"issues,omitempty"`
}

// BuildFiscalDependencyStatus builds a secret-free dependency report.
func BuildFiscalDependencyStatus(in DependencyStatusInput) FiscalDependencyStatus {
	issues := make([]string, 0, 4)
	if in.FeatureEnabled && !in.SchemaPresent {
		issues = append(issues, "schema_missing")
	}
	if in.FeatureEnabled && !in.ProviderHealthy {
		issues = append(issues, "provider_unhealthy")
	}
	if in.FeatureEnabled && strings.TrimSpace(in.ProviderKey) == "" {
		issues = append(issues, "provider_unconfigured")
	}
	return FiscalDependencyStatus{
		FeatureEnabled:      in.FeatureEnabled,
		FinalizationEnabled: in.FinalizationEnabled,
		WorkerEnabled:       in.WorkerEnabled,
		ProviderKey:         strings.TrimSpace(in.ProviderKey),
		ProviderHealthy:     in.ProviderHealthy,
		SchemaPresent:       in.SchemaPresent,
		Issues:              issues,
	}
}
