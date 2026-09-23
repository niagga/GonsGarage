package main

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	_ "github.com/lib/pq"

	fiscalmock "github.com/gaston-garcia-cegid/gonsgarage/internal/integration/fiscal/mock"
	"github.com/gaston-garcia-cegid/gonsgarage/internal/platform/fiscalartifact"
	postgresrepo "github.com/gaston-garcia-cegid/gonsgarage/internal/repository/postgres"
	fiscalsvc "github.com/gaston-garcia-cegid/gonsgarage/internal/service/fiscal"
)

func main() {
	if os.Getenv("FISCAL_WORKER_ENABLED") == "false" {
		log.Fatal("FISCAL_WORKER_ENABLED=false")
	}
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://admindb:gonsgarage123@localhost:5432/gonsgarage?sslmode=disable"
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	if err = db.Ping(); err != nil {
		log.Fatal(err)
	}
	if err = postgresrepo.VerifyFiscalSchema(context.Background(), db); err != nil {
		log.Fatalf("fiscal schema: %v", err)
	}

	appEnv := os.Getenv("APP_ENV")
	if appEnv == "" {
		appEnv = "development"
	}
	provider, err := fiscalmock.NewProvider(fiscalmock.Options{
		AppEnv:   appEnv,
		Registry: fiscalmock.NewScenarioRegistry(nil),
		Store:    fiscalmock.NewPostgresOperationStore(db),
	})
	if err != nil {
		log.Fatalf("provider: %v", err)
	}

	outbox := postgresrepo.NewFiscalOutboxRepository(db)
	artifactSvc, envName, err := composeArtifactService(appEnv, db)
	if err != nil {
		log.Fatalf("artifact store: %v", err)
	}
	owner := os.Getenv("FISCAL_WORKER_OWNER")
	if owner == "" {
		owner = "fiscal-worker-" + hostname()
	}
	lease := fiscalsvc.DefaultWorkerLease
	if v := os.Getenv("FISCAL_WORKER_LEASE"); v != "" {
		if d, perr := time.ParseDuration(v); perr == nil {
			lease = d
		}
	}
	worker := fiscalsvc.NewWorker(outbox, provider, fiscalsvc.WorkerConfig{
		Owner: owner, Lease: lease, PollInterval: fiscalsvc.DefaultWorkerPollInterval, Environment: envName,
	}).WithArtifactService(artifactSvc)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	log.Printf("fiscal worker starting owner=%s lease=%s", owner, lease)
	if err := worker.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		log.Fatal(err)
	}
	log.Print("fiscal worker stopped gracefully")
}

func hostname() string {
	h, err := os.Hostname()
	if err != nil || h == "" {
		return "unknown"
	}
	return h
}

func composeArtifactService(appEnv string, db *sql.DB) (*fiscalsvc.ArtifactService, string, error) {
	envName := os.Getenv("FISCAL_ARTIFACT_ENV")
	if envName == "" {
		envName = appEnv
	}
	backend := os.Getenv("FISCAL_ARTIFACT_BACKEND")
	if backend == "" {
		backend = fiscalartifact.BackendLocal
	}
	if appEnv == "production" {
		if backend == fiscalartifact.BackendLocal {
			return nil, "", fiscalartifact.AssertProductionIssuanceAllowed(fiscalartifact.ProductionReadiness{Backend: backend})
		}
		obj := fiscalartifact.NewObjectStore(
			os.Getenv("FISCAL_ARTIFACT_OBJECT_ENDPOINT"),
			os.Getenv("FISCAL_ARTIFACT_OBJECT_BUCKET"),
			fiscalartifact.ProductionReadiness{
				Backend:                backend,
				PrivateACL:             os.Getenv("FISCAL_ARTIFACT_PRIVATE_ACL") == "true",
				EncryptionAtRest:       os.Getenv("FISCAL_ARTIFACT_ENCRYPTION") == "true",
				RetentionConfigured:    os.Getenv("FISCAL_ARTIFACT_RETENTION") == "true",
				BackupRestoreEvidenced: os.Getenv("FISCAL_ARTIFACT_BACKUP_RESTORE") == "true",
				AccessLoggingEnabled:   os.Getenv("FISCAL_ARTIFACT_ACCESS_LOG") == "true",
			},
		)
		if err := obj.EnsureReady(); err != nil {
			return nil, "", err
		}
		return nil, "", errors.New("production object store adapter bytes path is configured for readiness only in WU6; wire cloud SDK in a later enablement unit")
	}
	root := os.Getenv("FISCAL_ARTIFACT_LOCAL_ROOT")
	if root == "" {
		root = filepath.Join(os.TempDir(), "gonsgarage-fiscal-artifacts")
	}
	store, err := fiscalartifact.NewLocalStore(root)
	if err != nil {
		return nil, "", err
	}
	svc := fiscalsvc.NewArtifactService(
		store,
		postgresrepo.NewFiscalArtifactRepository(db),
		postgresrepo.NewSQLInvoiceReader(db),
		fiscalsvc.WithDefaultEnvironment(envName),
		fiscalsvc.WithIssuanceEnabled(os.Getenv("FISCAL_FINALIZATION_ENABLED") != "false"),
	)
	return svc, envName, nil
}
