package mock

import (
	"errors"
	"strings"
)

var (
	// ErrMockForbiddenInProduction is returned when mock wiring is attempted in production.
	ErrMockForbiddenInProduction = errors.New("mock fiscal provider is forbidden in production")
	// ErrMockLegalClassificationForbidden is returned when mock evidence would be marked legal.
	ErrMockLegalClassificationForbidden = errors.New("mock fiscal artifacts cannot be classified as legal")
	// ErrMockStoreForbiddenInProduction is returned when fiscal_mock_operations is accessed in production.
	ErrMockStoreForbiddenInProduction = errors.New("mock fiscal operation store is forbidden in production")
	// ErrCanonicalConflict is returned when the same operation key is reused with different input.
	ErrCanonicalConflict = errors.New("mock operation key already bound to a different canonical input")
)

// MockPDFLabel is the visible non-legal watermark required on every mock PDF.
const MockPDFLabel = "SEM VALIDADE FISCAL — MOCK"

// AssertMockAllowed fails closed when APP_ENV is production.
func AssertMockAllowed(appEnv string) error {
	if isProductionEnv(appEnv) {
		return ErrMockForbiddenInProduction
	}
	return nil
}

// AssertMockStoreAllowed fails closed when mock repository access is requested in production.
func AssertMockStoreAllowed(appEnv string) error {
	if isProductionEnv(appEnv) {
		return ErrMockStoreForbiddenInProduction
	}
	return nil
}

// RejectLegalMockClassification fails closed when mock evidence is labeled legal.
func RejectLegalMockClassification(appEnv, classification string) error {
	_ = appEnv
	if strings.EqualFold(strings.TrimSpace(classification), "legal") {
		return ErrMockLegalClassificationForbidden
	}
	return nil
}

func isProductionEnv(appEnv string) bool {
	return strings.EqualFold(strings.TrimSpace(appEnv), "production")
}
