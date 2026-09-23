package main

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/lib/pq"

	fiscalmock "github.com/gaston-garcia-cegid/gonsgarage/internal/integration/fiscal/mock"
	postgresrepo "github.com/gaston-garcia-cegid/gonsgarage/internal/repository/postgres"
	fiscalsvc "github.com/gaston-garcia-cegid/gonsgarage/internal/service/fiscal"
)

func main() {
	cfg := fiscalsvc.ParseRuntimeConfig(os.Getenv)
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://admindb:gonsgarage123@localhost:5432/gonsgarage?sslmode=disable"
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	dbHealthy := db.Ping() == nil
	cfg.SchemaPresent = false
	if dbHealthy {
		cfg.SchemaPresent = postgresrepo.VerifyFiscalSchema(context.Background(), db) == nil
	}
	if err := validateWorkerStartup(cfg, dbHealthy); err != nil {
		log.Fatal(err)
	}

	providerKey, err := fiscalsvc.ResolveProviderKey(cfg)
	if err != nil {
		log.Fatalf("provider: %v", err)
	}
	if providerKey != "mock" {
		log.Fatalf("fiscal worker provider %q is not executable in this work unit; use mock in non-production or wait for WU12 Cloudware enablement", providerKey)
	}
	provider, err := fiscalmock.NewProvider(fiscalmock.Options{
		AppEnv:   cfg.AppEnv,
		Registry: fiscalmock.NewScenarioRegistry(nil),
		Store:    fiscalmock.NewPostgresOperationStore(db),
	})
	if err != nil {
		log.Fatalf("provider: %v", err)
	}

	outbox := postgresrepo.NewFiscalOutboxRepository(db)
	artifactComp, err := fiscalsvc.ComposeArtifactService(cfg.AppEnv, db, os.Getenv, cfg.FinalizationEnabled)
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
		Owner: owner, Lease: lease, PollInterval: fiscalsvc.DefaultWorkerPollInterval, Environment: artifactComp.Environment,
	}).WithArtifactService(artifactComp.Service)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	fields := fiscalsvc.FiscalLogEvent{
		CorrelationID:  owner,
		ProviderKey:    providerKey,
		Operation:      "worker_start",
		LeaseOwner:     owner,
		Classification: "info",
	}.AllowlistedFields()
	log.Printf("fiscal worker starting fields=%v lease=%s finalization_enabled=%v", fields, lease, cfg.FinalizationEnabled)
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
