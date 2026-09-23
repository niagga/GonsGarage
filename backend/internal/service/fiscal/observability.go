package fiscal

import (
	"time"

	"github.com/google/uuid"
)

// FiscalLogEvent is the allowlisted structured fiscal log payload.
// ForbiddenProbe exists only so tests can prove non-allowlisted maps are dropped.
type FiscalLogEvent struct {
	CorrelationID  string
	DocumentID     uuid.UUID
	OutboxEventID  uuid.UUID
	AttemptID      uuid.UUID
	ProviderKey    string
	Operation      string
	OldState       string
	NewState       string
	Classification string
	LeaseOwner     string
	Duration       time.Duration
	ForbiddenProbe map[string]string
}

// AllowlistedFields returns only operational identifiers — never PII, secrets, or payloads.
func (e FiscalLogEvent) AllowlistedFields() map[string]any {
	fields := map[string]any{
		"correlation_id": e.CorrelationID,
		"provider_key":   e.ProviderKey,
		"operation":      e.Operation,
		"old_state":      e.OldState,
		"new_state":      e.NewState,
		"classification": e.Classification,
		"lease_owner":    e.LeaseOwner,
		"duration_ms":    e.Duration.Milliseconds(),
	}
	if e.DocumentID != uuid.Nil {
		fields["fiscal_document_id"] = e.DocumentID.String()
	}
	if e.OutboxEventID != uuid.Nil {
		fields["outbox_event_id"] = e.OutboxEventID.String()
	}
	if e.AttemptID != uuid.Nil {
		fields["attempt_id"] = e.AttemptID.String()
	}
	return fields
}

// FiscalMetricsSnapshot is a secret-free operational metrics view.
type FiscalMetricsSnapshot struct {
	OutboxReadyAgeSeconds  float64
	LeasedCount            float64
	ExpiredLeaseCount      float64
	UnknownStateAgeSeconds float64
	ArtifactUnavailable    float64
	FinalizationConflicts  float64
	ProviderKey            string
	ProviderResult         string
}

// MetricLabels returns allowlisted metric labels/values.
func (s FiscalMetricsSnapshot) MetricLabels() map[string]any {
	return map[string]any{
		"outbox_ready_age_seconds":    s.OutboxReadyAgeSeconds,
		"outbox_leased_count":         s.LeasedCount,
		"outbox_expired_leases":       s.ExpiredLeaseCount,
		"unknown_state_age_seconds":   s.UnknownStateAgeSeconds,
		"artifact_unavailable_count":  s.ArtifactUnavailable,
		"finalization_conflict_count": s.FinalizationConflicts,
		"provider_key":                s.ProviderKey,
		"provider_result":             s.ProviderResult,
	}
}
