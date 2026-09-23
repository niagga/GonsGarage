package fiscal

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gaston-garcia-cegid/gonsgarage/internal/platform/fiscalartifact"
	postgresrepo "github.com/gaston-garcia-cegid/gonsgarage/internal/repository/postgres"
)

// ArtifactComposition is the shared API/worker artifact wiring result.
type ArtifactComposition struct {
	Service     *ArtifactService
	Environment string
	Backend     string
}

// ComposeArtifactService builds the local or production-ready artifact path from env.
// Production object bytes remain readiness-gated until a cloud SDK is approved (WU12).
func ComposeArtifactService(appEnv string, db *sql.DB, getenv func(string) string, finalizationEnabled bool) (*ArtifactComposition, error) {
	if getenv == nil {
		getenv = os.Getenv
	}
	envName := getenv("FISCAL_ARTIFACT_ENV")
	if envName == "" {
		envName = appEnv
	}
	backend := getenv("FISCAL_ARTIFACT_BACKEND")
	if backend == "" {
		backend = fiscalartifact.BackendLocal
	}
	if appEnv == "production" {
		if backend == fiscalartifact.BackendLocal {
			return nil, fiscalartifact.AssertProductionIssuanceAllowed(fiscalartifact.ProductionReadiness{Backend: backend})
		}
		obj := fiscalartifact.NewObjectStore(
			getenv("FISCAL_ARTIFACT_OBJECT_ENDPOINT"),
			getenv("FISCAL_ARTIFACT_OBJECT_BUCKET"),
			fiscalartifact.ProductionReadiness{
				Backend:                backend,
				PrivateACL:             getenv("FISCAL_ARTIFACT_PRIVATE_ACL") == "true",
				EncryptionAtRest:       getenv("FISCAL_ARTIFACT_ENCRYPTION") == "true",
				RetentionConfigured:    getenv("FISCAL_ARTIFACT_RETENTION") == "true",
				BackupRestoreEvidenced: getenv("FISCAL_ARTIFACT_BACKUP_RESTORE") == "true",
				AccessLoggingEnabled:   getenv("FISCAL_ARTIFACT_ACCESS_LOG") == "true",
			},
		)
		if err := obj.EnsureReady(); err != nil {
			return nil, err
		}
		return nil, errors.New("production object store adapter bytes path is readiness-only until controlled enablement")
	}
	root := getenv("FISCAL_ARTIFACT_LOCAL_ROOT")
	if root == "" {
		root = filepath.Join(os.TempDir(), "gonsgarage-fiscal-artifacts")
	}
	store, err := fiscalartifact.NewLocalStore(root)
	if err != nil {
		return nil, err
	}
	svc := NewArtifactService(
		store,
		postgresrepo.NewFiscalArtifactRepository(db),
		postgresrepo.NewSQLInvoiceReader(db),
		WithDefaultEnvironment(envName),
		WithIssuanceEnabled(finalizationEnabled),
	)
	return &ArtifactComposition{Service: svc, Environment: envName, Backend: backend}, nil
}

// ResolveProviderKey picks the configured provider with production fail-closed defaults.
func ResolveProviderKey(cfg RuntimeConfig) (string, error) {
	key := cfg.ProviderKey
	if key == "" {
		if cfg.isProduction() {
			return "", fmt.Errorf("%w: provider_unconfigured", ErrUnsafeFiscalProductionConfig)
		}
		return "mock", nil
	}
	if cfg.isProduction() && key == "mock" {
		return "", fmt.Errorf("%w: mock_provider", ErrUnsafeFiscalProductionConfig)
	}
	return key, nil
}
