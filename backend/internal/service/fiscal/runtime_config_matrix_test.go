package fiscal_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	fiscalsvc "github.com/gaston-garcia-cegid/gonsgarage/internal/service/fiscal"
)

func TestRuntimeConfig_FeatureOffKeepsProductionSafe(t *testing.T) {
	cfg := fiscalsvc.RuntimeConfig{
		AppEnv: "production", FeatureEnabled: false, ProviderKey: "mock",
		ArtifactBackend: "local", JWTSecretIsDefault: true, SchemaPresent: false,
	}
	require.NoError(t, cfg.ValidateProductionComposition(), "feature-off must not force composition failures")
}

func TestRuntimeConfig_CloudwareFailClosedWithoutMutations(t *testing.T) {
	env := map[string]string{
		"APP_ENV":                     "production",
		"FISCAL_FEATURE_ENABLED":      "true",
		"FISCAL_FINALIZATION_ENABLED": "false",
		"FISCAL_WORKER_ENABLED":       "false",
		"FISCAL_PROVIDER":             "cloudware",
		"FISCAL_CLOUDWARE_MUTATIONS":  "false",
		"FISCAL_ARTIFACT_BACKEND":     "object",
		"FISCAL_CREDENTIAL_KEY_B64":   "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
		"JWT_SECRET":                  "prod-not-default-secret-value-32b",
	}
	cfg := fiscalsvc.ParseRuntimeConfig(func(k string) string { return env[k] })
	cfg.SchemaPresent = true
	require.False(t, cfg.FinalizationEnabled)
	require.False(t, cfg.WorkerEnabled)
	require.False(t, cfg.CloudwareMutationsEnabled)
	require.NoError(t, cfg.ValidateProductionComposition())
}

func TestRuntimeConfig_ProviderOutageIsolatedFromCoreReady(t *testing.T) {
	status := fiscalsvc.BuildFiscalDependencyStatus(fiscalsvc.DependencyStatusInput{
		FeatureEnabled: true, ProviderKey: "cloudware", ProviderHealthy: false, SchemaPresent: true,
	})
	require.Contains(t, status.Issues, "provider_unhealthy")
	require.True(t, fiscalsvc.CoreAPIReady(true, status.ProviderHealthy))
}

func TestResolveProviderKey_DefaultsMockOnlyOutsideProduction(t *testing.T) {
	key, err := fiscalsvc.ResolveProviderKey(fiscalsvc.RuntimeConfig{AppEnv: "development"})
	require.NoError(t, err)
	require.Equal(t, "mock", key)

	_, err = fiscalsvc.ResolveProviderKey(fiscalsvc.RuntimeConfig{AppEnv: "production", ProviderKey: "mock"})
	require.Error(t, err)
	require.Contains(t, strings.ToLower(err.Error()), "mock")
}

func TestArtifactRecoveryConfig_IssuanceFollowsFinalizationSwitch(t *testing.T) {
	cfgOff := fiscalsvc.ParseRuntimeConfig(func(k string) string {
		if k == "FISCAL_FINALIZATION_ENABLED" {
			return "false"
		}
		return ""
	})
	require.False(t, cfgOff.FinalizationEnabled)

	cfgOn := fiscalsvc.ParseRuntimeConfig(func(k string) string {
		if k == "FISCAL_FINALIZATION_ENABLED" {
			return "true"
		}
		return ""
	})
	require.True(t, cfgOn.FinalizationEnabled)
}
