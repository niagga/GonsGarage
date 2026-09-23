package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	fiscalsvc "github.com/gaston-garcia-cegid/gonsgarage/internal/service/fiscal"
)

func TestWorkerStartupGate_RequiresExplicitEnableAndHealthyDeps(t *testing.T) {
	err := validateWorkerStartup(fiscalsvc.RuntimeConfig{
		AppEnv: "development", FeatureEnabled: true, WorkerEnabled: false,
		ProviderKey: "mock", ArtifactBackend: "local", SchemaPresent: true,
		CredentialKeyConfigured: true, CredentialKeyVersion: "v1",
	}, true)
	require.Error(t, err)
	require.Contains(t, strings.ToLower(err.Error()), "worker")

	err = validateWorkerStartup(fiscalsvc.RuntimeConfig{
		AppEnv: "development", FeatureEnabled: true, WorkerEnabled: true,
		ProviderKey: "mock", ArtifactBackend: "local", SchemaPresent: false,
		CredentialKeyConfigured: true, CredentialKeyVersion: "v1",
	}, true)
	require.Error(t, err)
	require.Contains(t, strings.ToLower(err.Error()), "schema")

	err = validateWorkerStartup(fiscalsvc.RuntimeConfig{
		AppEnv: "development", FeatureEnabled: true, WorkerEnabled: true,
		ProviderKey: "mock", ArtifactBackend: "local", SchemaPresent: true,
		CredentialKeyConfigured: true, CredentialKeyVersion: "v1",
	}, false)
	require.Error(t, err)
	require.Contains(t, strings.ToLower(err.Error()), "database")

	err = validateWorkerStartup(fiscalsvc.RuntimeConfig{
		AppEnv: "development", FeatureEnabled: true, WorkerEnabled: true,
		ProviderKey: "mock", ArtifactBackend: "local", SchemaPresent: true,
		CredentialKeyConfigured: true, CredentialKeyVersion: "v1",
	}, true)
	require.NoError(t, err)
}

func TestWorkerStartupGate_ProductionRejectsMockProvider(t *testing.T) {
	err := validateWorkerStartup(fiscalsvc.RuntimeConfig{
		AppEnv: "production", FeatureEnabled: true, WorkerEnabled: true,
		ProviderKey: "mock", ArtifactBackend: "object", SchemaPresent: true,
		CredentialKeyConfigured: true, CredentialKeyVersion: "v1",
	}, true)
	require.Error(t, err)
	require.Contains(t, strings.ToLower(err.Error()), "mock")
}
