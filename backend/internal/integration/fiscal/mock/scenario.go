package mock

import "strings"

// Scenario selects deterministic mock behavior by operation key.
type Scenario string

const (
	ScenarioSuccess           Scenario = "success"
	ScenarioValidation        Scenario = "validation"
	ScenarioExpiredConnection Scenario = "expired_connection"
	ScenarioTransient         Scenario = "transient"
	ScenarioRateLimit         Scenario = "rate_limit"
	ScenarioAmbiguous         Scenario = "ambiguous"
	ScenarioVoidPermitted     Scenario = "void_permitted"
	ScenarioVoidRefused       Scenario = "void_refused"
)

// ScenarioRegistry resolves scenarios exclusively by operation key.
type ScenarioRegistry interface {
	Lookup(operationKey string) (Scenario, bool)
}

type mapRegistry map[string]Scenario

// NewScenarioRegistry builds an injected registry keyed only by operation key.
func NewScenarioRegistry(entries map[string]Scenario) ScenarioRegistry {
	cloned := make(mapRegistry, len(entries))
	for key, value := range entries {
		cloned[strings.TrimSpace(key)] = value
	}
	return cloned
}

func (r mapRegistry) Lookup(operationKey string) (Scenario, bool) {
	scenario, ok := r[strings.TrimSpace(operationKey)]
	return scenario, ok
}
