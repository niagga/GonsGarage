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
		Owner: owner, Lease: lease, PollInterval: fiscalsvc.DefaultWorkerPollInterval,
	})

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
