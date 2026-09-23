package postgres

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequiredRepairsColumns_IncludesTimeline(t *testing.T) {
	t.Parallel()
	cols := RequiredRepairsColumns()
	require.Contains(t, cols, "started_at")
	require.Contains(t, cols, "completed_at")
	require.Contains(t, cols, "technician_id")
}

func TestMissingRepairsColumns_ReportsLegacyDrift(t *testing.T) {
	t.Parallel()
	// Shape of a DB created from migrations/004 (start_date/end_date, employee_id)
	// after only technician_id/service_job_id ensures ran — matches local repro.
	present := map[string]bool{
		"id": true, "car_id": true, "technician_id": true, "description": true,
		"status": true, "cost": true, "created_at": true, "updated_at": true,
		"deleted_at": true, "start_date": true, "end_date": true, "employee_id": true,
		"service_job_id": true,
	}
	missing := MissingRepairsColumns(present)
	assert.Equal(t, []string{"started_at", "completed_at"}, missing)
}

func TestMissingRepairsColumns_EmptyWhenAligned(t *testing.T) {
	t.Parallel()
	present := make(map[string]bool, len(requiredRepairsColumns))
	for _, c := range RequiredRepairsColumns() {
		present[c] = true
	}
	assert.Empty(t, MissingRepairsColumns(present))
}

func TestEnsureRepairsSchema_NilDB(t *testing.T) {
	t.Parallel()
	err := EnsureRepairsSchema(nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nil")
}
