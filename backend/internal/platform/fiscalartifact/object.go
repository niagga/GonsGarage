package fiscalartifact

import (
	"fmt"
	"strings"

	"github.com/gaston-garcia-cegid/gonsgarage/internal/core/ports"
)

const (
	// BackendLocal is the development private filesystem backend.
	BackendLocal = "local"
	// BackendObject is the production private object-store backend boundary.
	BackendObject = "object"
)

// ProductionReadiness captures fail-closed production storage checks.
type ProductionReadiness struct {
	Backend                string
	PrivateACL             bool
	EncryptionAtRest       bool
	RetentionConfigured    bool
	BackupRestoreEvidenced bool
	AccessLoggingEnabled   bool
}

// Missing reports readiness gaps in stable codes.
func (r ProductionReadiness) Missing() []string {
	var missing []string
	backend := strings.ToLower(strings.TrimSpace(r.Backend))
	if backend == "" || backend == BackendLocal {
		missing = append(missing, "backend_not_object")
	}
	if !r.PrivateACL {
		missing = append(missing, "private_acl")
	}
	if !r.EncryptionAtRest {
		missing = append(missing, "encryption_at_rest")
	}
	if !r.RetentionConfigured {
		missing = append(missing, "retention")
	}
	if !r.BackupRestoreEvidenced {
		missing = append(missing, "backup_restore")
	}
	if !r.AccessLoggingEnabled {
		missing = append(missing, "access_logging")
	}
	return missing
}

// Ready reports whether production issuance may use this backend.
func (r ProductionReadiness) Ready() bool {
	return len(r.Missing()) == 0
}

// AssertProductionIssuanceAllowed fails closed until the object backend passes readiness.
func AssertProductionIssuanceAllowed(readiness ProductionReadiness) error {
	missing := readiness.Missing()
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf("%w: %s", ports.ErrFiscalArtifactBackendNotReady, strings.Join(missing, ","))
}

// ObjectStore is the production adapter boundary.
// Concrete cloud SDKs are composed behind this type once readiness evidence exists.
type ObjectStore struct {
	// Endpoint is the private object API base (never returned to clients).
	Endpoint string
	Bucket   string
	// Readiness must pass before production issuance is enabled.
	Readiness ProductionReadiness
}

// NewObjectStore builds the production boundary with explicit readiness.
func NewObjectStore(endpoint, bucket string, readiness ProductionReadiness) *ObjectStore {
	readiness.Backend = BackendObject
	return &ObjectStore{Endpoint: endpoint, Bucket: bucket, Readiness: readiness}
}

// EnsureReady fails closed for production issuance composition.
func (s *ObjectStore) EnsureReady() error {
	if s == nil {
		return ports.ErrFiscalArtifactBackendNotReady
	}
	s.Readiness.Backend = BackendObject
	return AssertProductionIssuanceAllowed(s.Readiness)
}
