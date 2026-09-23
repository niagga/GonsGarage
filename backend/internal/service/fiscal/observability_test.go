package fiscal_test

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	fiscalsvc "github.com/gaston-garcia-cegid/gonsgarage/internal/service/fiscal"
)

func TestFiscalLogEvent_AllowlistsSafeFieldsOnly(t *testing.T) {
	docID := uuid.New()
	eventID := uuid.New()
	attemptID := uuid.New()
	fields := fiscalsvc.FiscalLogEvent{
		CorrelationID: "corr-1",
		DocumentID:    docID,
		OutboxEventID: eventID,
		AttemptID:     attemptID,
		ProviderKey:   "mock",
		Operation:     "issue",
		OldState:      "pending",
		NewState:      "issued",
		Classification: "success",
		LeaseOwner:    "worker-1",
		Duration:      125 * time.Millisecond,
		// Forbidden payloads must never appear even if callers set them.
		ForbiddenProbe: map[string]string{
			"nif":             "PT123",
			"authorization":   "Bearer secret-token",
			"canonical_json":  `{"customer":"leak"}`,
			"provider_url":    "https://provider.example/doc",
			"pdf_bytes":       "%PDF-1.4",
			"ciphertext":      "aabbcc",
		},
	}.AllowlistedFields()

	require.Equal(t, "corr-1", fields["correlation_id"])
	require.Equal(t, docID.String(), fields["fiscal_document_id"])
	require.Equal(t, eventID.String(), fields["outbox_event_id"])
	require.Equal(t, attemptID.String(), fields["attempt_id"])
	require.Equal(t, "mock", fields["provider_key"])
	require.Equal(t, "issue", fields["operation"])
	require.Equal(t, "pending", fields["old_state"])
	require.Equal(t, "issued", fields["new_state"])
	require.Equal(t, "success", fields["classification"])
	require.Equal(t, "worker-1", fields["lease_owner"])
	require.Equal(t, int64(125), fields["duration_ms"])

	for key := range fields {
		lower := strings.ToLower(key)
		require.NotContains(t, lower, "nif")
		require.NotContains(t, lower, "token")
		require.NotContains(t, lower, "canonical")
		require.NotContains(t, lower, "url")
		require.NotContains(t, lower, "pdf")
		require.NotContains(t, lower, "cipher")
		require.NotContains(t, lower, "authorization")
	}
	_, hasForbidden := fields["forbidden_probe"]
	require.False(t, hasForbidden)
}

func TestFiscalMetricsSnapshot_TracksOperationalSignals(t *testing.T) {
	snap := fiscalsvc.FiscalMetricsSnapshot{
		OutboxReadyAgeSeconds: 42,
		LeasedCount:           2,
		ExpiredLeaseCount:     1,
		UnknownStateAgeSeconds: 90,
		ArtifactUnavailable:   3,
		FinalizationConflicts: 1,
		ProviderKey:           "cloudware",
		ProviderResult:        "transient",
	}
	m := snap.MetricLabels()
	require.Equal(t, float64(42), m["outbox_ready_age_seconds"])
	require.Equal(t, float64(2), m["outbox_leased_count"])
	require.Equal(t, float64(1), m["outbox_expired_leases"])
	require.Equal(t, float64(90), m["unknown_state_age_seconds"])
	require.Equal(t, float64(3), m["artifact_unavailable_count"])
	require.Equal(t, float64(1), m["finalization_conflict_count"])
	require.Equal(t, "cloudware", m["provider_key"])
	require.Equal(t, "transient", m["provider_result"])
	require.NotContains(t, m, "access_token")
	require.NotContains(t, m, "nif")
}
