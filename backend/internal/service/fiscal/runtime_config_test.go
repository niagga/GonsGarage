package fiscal_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	fiscalsvc "github.com/gaston-garcia-cegid/gonsgarage/internal/service/fiscal"
)

func TestRuntimeConfig_IndependentSwitchesDefaultOff(t *testing.T) {
	cfg := fiscalsvc.ParseRuntimeConfig(func(string) string { return "" })

	require.False(t, cfg.FeatureEnabled, "feature must default off")
	require.False(t, cfg.FinalizationEnabled, "finalization must default off")
	require.False(t, cfg.WorkerEnabled, "worker must default off")
	require.False(t, cfg.CloudwareMutationsEnabled, "Cloudware mutations must default off")
	require.Equal(t, "development", cfg.AppEnv)
	require.Equal(t, "", cfg.ProviderKey)
	require.Equal(t, "local", cfg.ArtifactBackend)
}

func TestRuntimeConfig_IndependentSwitchesParseExplicitly(t *testing.T) {
	env := map[string]string{
		"APP_ENV":                       "staging",
		"FISCAL_FEATURE_ENABLED":        "true",
		"FISCAL_FINALIZATION_ENABLED":   "true",
		"FISCAL_WORKER_ENABLED":         "true",
		"FISCAL_PROVIDER":               "cloudware",
		"FISCAL_CLOUDWARE_MUTATIONS":    "false",
		"FISCAL_ARTIFACT_BACKEND":       "object",
		"FISCAL_CREDENTIAL_KEY_VERSION": "v2",
		"FISCAL_CREDENTIAL_KEY_B64":     "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
	}
	cfg := fiscalsvc.ParseRuntimeConfig(func(k string) string { return env[k] })

	require.True(t, cfg.FeatureEnabled)
	require.True(t, cfg.FinalizationEnabled)
	require.True(t, cfg.WorkerEnabled)
	require.False(t, cfg.CloudwareMutationsEnabled)
	require.Equal(t, "staging", cfg.AppEnv)
	require.Equal(t, "cloudware", cfg.ProviderKey)
	require.Equal(t, "object", cfg.ArtifactBackend)
	require.True(t, cfg.CredentialKeyConfigured)
	require.Equal(t, "v2", cfg.CredentialKeyVersion)
}

func TestRuntimeConfig_ProductionRejectsDefaultsMockLocalMissingSchema(t *testing.T) {
	cases := []struct {
		name string
		cfg  fiscalsvc.RuntimeConfig
		want string
	}{
		{
			name: "default jwt secret",
			cfg: fiscalsvc.RuntimeConfig{
				AppEnv: "production", FeatureEnabled: true, ProviderKey: "cloudware",
				ArtifactBackend: "object", CredentialKeyConfigured: true, CredentialKeyVersion: "v1",
				SchemaPresent: true, JWTSecretIsDefault: true,
			},
			want: "jwt",
		},
		{
			name: "mock provider",
			cfg: fiscalsvc.RuntimeConfig{
				AppEnv: "production", FeatureEnabled: true, ProviderKey: "mock",
				ArtifactBackend: "object", CredentialKeyConfigured: true, CredentialKeyVersion: "v1",
				SchemaPresent: true,
			},
			want: "mock",
		},
		{
			name: "local artifact backend",
			cfg: fiscalsvc.RuntimeConfig{
				AppEnv: "production", FeatureEnabled: true, ProviderKey: "cloudware",
				ArtifactBackend: "local", CredentialKeyConfigured: true, CredentialKeyVersion: "v1",
				SchemaPresent: true,
			},
			want: "artifact",
		},
		{
			name: "missing schema",
			cfg: fiscalsvc.RuntimeConfig{
				AppEnv: "production", FeatureEnabled: true, ProviderKey: "cloudware",
				ArtifactBackend: "object", CredentialKeyConfigured: true, CredentialKeyVersion: "v1",
				SchemaPresent: false,
			},
			want: "schema",
		},
		{
			name: "missing credential keyring",
			cfg: fiscalsvc.RuntimeConfig{
				AppEnv: "production", FeatureEnabled: true, ProviderKey: "cloudware",
				ArtifactBackend: "object", CredentialKeyConfigured: false, CredentialKeyVersion: "v1",
				SchemaPresent: true,
			},
			want: "credential",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.cfg.ValidateProductionComposition()
			require.Error(t, err)
			require.True(t, errors.Is(err, fiscalsvc.ErrUnsafeFiscalProductionConfig) ||
				strings.Contains(strings.ToLower(err.Error()), tc.want),
				"got %v want substring %q", err, tc.want)
		})
	}
}

func TestRuntimeConfig_NonProductionAllowsMockAndLocal(t *testing.T) {
	cfg := fiscalsvc.RuntimeConfig{
		AppEnv: "development", FeatureEnabled: true, ProviderKey: "mock",
		ArtifactBackend: "local", CredentialKeyConfigured: true, CredentialKeyVersion: "v1",
		SchemaPresent: true,
	}
	require.NoError(t, cfg.ValidateProductionComposition())
}

func TestCoreAPIReady_IgnoresProviderHealth(t *testing.T) {
	require.True(t, fiscalsvc.CoreAPIReady(true, false), "provider outage must not fail /ready")
	require.True(t, fiscalsvc.CoreAPIReady(true, true))
	require.False(t, fiscalsvc.CoreAPIReady(false, true), "DB failure must fail /ready")
	require.False(t, fiscalsvc.CoreAPIReady(false, false))
}

func TestWorkerReadiness_FailsOnSchemaConfigOrDB(t *testing.T) {
	ok, issues := fiscalsvc.EvaluateWorkerReadiness(fiscalsvc.WorkerReadinessInput{
		DBHealthy: true, SchemaPresent: true, ConfigOK: true, WorkerEnabled: true,
	})
	require.True(t, ok)
	require.Empty(t, issues)

	ok, issues = fiscalsvc.EvaluateWorkerReadiness(fiscalsvc.WorkerReadinessInput{
		DBHealthy: false, SchemaPresent: true, ConfigOK: true, WorkerEnabled: true,
	})
	require.False(t, ok)
	require.Contains(t, strings.Join(issues, ","), "database")

	ok, issues = fiscalsvc.EvaluateWorkerReadiness(fiscalsvc.WorkerReadinessInput{
		DBHealthy: true, SchemaPresent: false, ConfigOK: true, WorkerEnabled: true,
	})
	require.False(t, ok)
	require.Contains(t, strings.Join(issues, ","), "schema")

	ok, issues = fiscalsvc.EvaluateWorkerReadiness(fiscalsvc.WorkerReadinessInput{
		DBHealthy: true, SchemaPresent: true, ConfigOK: false, WorkerEnabled: true,
	})
	require.False(t, ok)
	require.Contains(t, strings.Join(issues, ","), "config")

	ok, issues = fiscalsvc.EvaluateWorkerReadiness(fiscalsvc.WorkerReadinessInput{
		DBHealthy: true, SchemaPresent: true, ConfigOK: true, WorkerEnabled: false,
	})
	require.False(t, ok)
	require.Contains(t, strings.Join(issues, ","), "worker")
}

func TestFiscalDependencyStatus_ReportsProviderWithoutAffectingCoreReady(t *testing.T) {
	status := fiscalsvc.BuildFiscalDependencyStatus(fiscalsvc.DependencyStatusInput{
		FeatureEnabled: true, FinalizationEnabled: false, WorkerEnabled: false,
		ProviderKey: "cloudware", ProviderHealthy: false, SchemaPresent: true,
	})
	require.False(t, status.ProviderHealthy)
	require.Contains(t, status.Issues, "provider_unhealthy")
	require.True(t, fiscalsvc.CoreAPIReady(true, status.ProviderHealthy))
}

func TestFinalizationDisabledByDefaultBlocksLegalActions(t *testing.T) {
	svc := fiscalsvc.NewFinalizationService(nil, nil, nil)
	require.False(t, svc.IsEnabled())

	enabled := fiscalsvc.NewFinalizationService(nil, nil, nil).WithEnabled(true)
	require.True(t, enabled.IsEnabled())
}
