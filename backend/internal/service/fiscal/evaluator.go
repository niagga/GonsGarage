package fiscal

import (
	"github.com/gaston-garcia-cegid/gonsgarage/internal/domain"
)

// CloudwareEnablementEvaluator gates production mutations.
type CloudwareEnablementEvaluator struct {
	productionEnabled bool
}

// NewCloudwareEnablementEvaluator builds the gate evaluator.
func NewCloudwareEnablementEvaluator(productionEnabled bool) *CloudwareEnablementEvaluator {
	return &CloudwareEnablementEvaluator{productionEnabled: productionEnabled}
}

// Evaluate checks if production mutations are permitted for the given document.
func (e *CloudwareEnablementEvaluator) Evaluate(doc *domain.FiscalDocument) bool {
	// 1. Production must be explicitly enabled
	if !e.productionEnabled {
		return false
	}

	// 2. Additional gates (e.g., must have valid provider fixation)
	if !doc.ProviderFixed() || !doc.IntentFixed() {
		return false
	}

	return true
}
