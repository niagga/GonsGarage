package main

import (
	"fmt"
	"strings"

	fiscalsvc "github.com/gaston-garcia-cegid/gonsgarage/internal/service/fiscal"
)

func validateWorkerStartup(cfg fiscalsvc.RuntimeConfig, dbHealthy bool) error {
	if err := cfg.ValidateProductionComposition(); err != nil {
		return err
	}
	if cfg.ProviderKey == "mock" && strings.EqualFold(cfg.AppEnv, "production") {
		return fmt.Errorf("%w: mock_provider", fiscalsvc.ErrUnsafeFiscalProductionConfig)
	}
	ok, issues := fiscalsvc.EvaluateWorkerReadiness(fiscalsvc.WorkerReadinessInput{
		DBHealthy:     dbHealthy,
		SchemaPresent: cfg.SchemaPresent,
		ConfigOK:      true,
		WorkerEnabled: cfg.WorkerEnabled,
	})
	if !ok {
		return fmt.Errorf("%w: %s", fiscalsvc.ErrFiscalWorkerNotReady, strings.Join(issues, ","))
	}
	return nil
}
