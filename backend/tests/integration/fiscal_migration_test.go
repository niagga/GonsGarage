package integration

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"

	migrations "github.com/gaston-garcia-cegid/gonsgarage/cmd/migrate/migrations"
	postgresrepo "github.com/gaston-garcia-cegid/gonsgarage/internal/repository/postgres"
)

func fiscalTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("FISCAL_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("FISCAL_TEST_DATABASE_URL is required for PostgreSQL migration acceptance")
	}
	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	require.NoError(t, db.Ping())
	schema := "fiscal_" + uuid.NewString()[:8]
	_, err = db.Exec(`CREATE SCHEMA ` + schema)
	require.NoError(t, err)
	_, err = db.Exec(`SET search_path TO ` + schema + `,public`)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = db.Exec(`SET search_path TO public`)
		_, _ = db.Exec(`DROP SCHEMA ` + schema + ` CASCADE`)
		_ = db.Close()
	})
	return db
}

func migrationDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	source := filepath.Join(filepath.Dir(file), "..", "..", "migrations", "011_fiscal_integration_foundation.up.sql")
	content, err := os.ReadFile(source)
	require.NoError(t, err)
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, filepath.Base(source)), content, 0o600))
	return dir
}

func execFails(t *testing.T, db *sql.DB, query string, args ...any) {
	t.Helper()
	_, err := db.Exec(query, args...)
	require.Error(t, err)
}

func TestFiscalMigrationLegacyIsolationAndSchema(t *testing.T) {
	db := fiscalTestDB(t)
	_, err := db.Exec(`
CREATE TABLE users (id uuid PRIMARY KEY);
CREATE TABLE invoices (
 id uuid PRIMARY KEY, customer_id uuid NOT NULL REFERENCES users(id), amount double precision NOT NULL,
 status varchar(40) NOT NULL DEFAULT 'open', notes text, created_at timestamptz DEFAULT now(), updated_at timestamptz DEFAULT now()
);
INSERT INTO users(id) VALUES ('00000000-0000-0000-0000-000000000001');
INSERT INTO invoices(id,customer_id,amount) VALUES ('10000000-0000-0000-0000-000000000001','00000000-0000-0000-0000-000000000001',10);
`)
	require.NoError(t, err)
	require.Error(t, postgresrepo.VerifyFiscalSchema(context.Background(), db))

	require.NoError(t, migrations.Run(db, migrationDir(t)))
	require.NoError(t, postgresrepo.VerifyFiscalSchema(context.Background(), db))

	var legacy, documents, events, gates, policies, profiles int
	require.NoError(t, db.QueryRow(`SELECT count(*) FROM invoices WHERE fiscal_eligibility='legacy_unfiscalized'`).Scan(&legacy))
	require.NoError(t, db.QueryRow(`SELECT count(*) FROM fiscal_documents`).Scan(&documents))
	require.NoError(t, db.QueryRow(`SELECT count(*) FROM fiscal_outbox_events`).Scan(&events))
	require.NoError(t, db.QueryRow(`SELECT count(*) FROM fiscal_enablement_gates WHERE provider_key='cloudware' AND status='pending'`).Scan(&gates))
	require.NoError(t, db.QueryRow(`SELECT count(*) FROM fiscal_policy_versions`).Scan(&policies))
	require.NoError(t, db.QueryRow(`SELECT count(*) FROM fiscal_issuer_profiles`).Scan(&profiles))
	require.Equal(t, []int{1, 0, 0, 8, 0, 0}, []int{legacy, documents, events, gates, policies, profiles})

	_, err = db.Exec(`INSERT INTO invoices(id,customer_id,amount) VALUES ($1,$2,20)`, uuid.New(), "00000000-0000-0000-0000-000000000001")
	require.NoError(t, err)
	var eligibility string
	require.NoError(t, db.QueryRow(`SELECT fiscal_eligibility FROM invoices WHERE amount=20`).Scan(&eligibility))
	require.Equal(t, "legacy_unfiscalized", eligibility)
}

func createFiscalMigrationBase(t *testing.T, db *sql.DB) {
	t.Helper()
	_, err := db.Exec(`CREATE TABLE users (id uuid PRIMARY KEY); CREATE TABLE invoices (id uuid PRIMARY KEY, customer_id uuid NOT NULL, amount double precision NOT NULL);`)
	require.NoError(t, err)
	require.NoError(t, migrations.Run(db, migrationDir(t)))
}

func TestFiscalMigrationSchemaVerificationRejectsMissingObjectsIndependently(t *testing.T) {
	cases := []struct {
		name, sabotage, missing string
	}{
		{"intent index", `DROP INDEX fiscal_documents_intent_key_uq`, "index:fiscal_documents.fiscal_documents_intent_key_uq"},
		{"void index", `DROP INDEX fiscal_documents_void_key_uq`, "index:fiscal_documents.fiscal_documents_void_key_uq"},
		{"access log trigger", `DROP TRIGGER fiscal_access_log_append_only ON fiscal_artifact_access_log`, "trigger:fiscal_artifact_access_log.fiscal_access_log_append_only"},
		{"policy trigger", `DROP TRIGGER fiscal_policy_immutable ON fiscal_policy_versions`, "trigger:fiscal_policy_versions.fiscal_policy_immutable"},
		{"issuer trigger", `DROP TRIGGER fiscal_issuer_immutable ON fiscal_issuer_profiles`, "trigger:fiscal_issuer_profiles.fiscal_issuer_immutable"},
		{"required column", `ALTER TABLE fiscal_documents DROP COLUMN provider_number`, "column:fiscal_documents.provider_number"},
		{"required constraint", `ALTER TABLE fiscal_documents DROP CONSTRAINT fiscal_documents_kind_check`, "constraint:fiscal_documents.fiscal_documents_kind_check"},
		{"required table", `DROP TABLE fiscal_mock_operations`, "table:fiscal_mock_operations"},
		{"migration marker", `DELETE FROM schema_migrations WHERE version='011_fiscal_integration_foundation'`, "migration:011_fiscal_integration_foundation"},
		{"table-scoped trigger", `DROP TRIGGER fiscal_lines_immutable ON fiscal_document_lines; CREATE TABLE trigger_decoy(id integer); CREATE TRIGGER fiscal_lines_immutable BEFORE UPDATE ON trigger_decoy FOR EACH ROW EXECUTE FUNCTION fiscal_append_only()`, "trigger:fiscal_document_lines.fiscal_lines_immutable"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db := fiscalTestDB(t)
			createFiscalMigrationBase(t, db)
			_, err := db.Exec(tc.sabotage)
			require.NoError(t, err)
			err = postgresrepo.VerifyFiscalSchema(context.Background(), db)
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.missing)
		})
	}
}

func TestFiscalMigrationConstraintsAndImmutability(t *testing.T) {
	db := fiscalTestDB(t)
	_, err := db.Exec(`CREATE TABLE users (id uuid PRIMARY KEY); CREATE TABLE invoices (id uuid PRIMARY KEY, customer_id uuid NOT NULL, amount double precision NOT NULL); INSERT INTO users VALUES ('00000000-0000-0000-0000-000000000001'); INSERT INTO invoices VALUES ('10000000-0000-0000-0000-000000000001','00000000-0000-0000-0000-000000000001',10),('10000000-0000-0000-0000-000000000002','00000000-0000-0000-0000-000000000001',20);`)
	require.NoError(t, err)
	require.NoError(t, migrations.Run(db, migrationDir(t)))

	doc1, doc2, snapshot, connection, policy, issuer := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	_, err = db.Exec(`WITH connection AS (INSERT INTO fiscal_provider_connections(id,provider_key,state) VALUES ($1,'mock','connected')),
policy AS (INSERT INTO fiscal_policy_versions(id,policy_key,version,schema_version,classification,status,config,config_sha256) VALUES ($2,'test',1,1,'mock','approved','{}',repeat('a',64)))
INSERT INTO fiscal_issuer_profiles(id,version,status,legal_identity,billing_address,tax_profile,series_config,config_sha256) VALUES ($3,1,'approved','{}','{}','{}','{}',repeat('b',64))`, connection, policy, issuer)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO fiscal_documents(id,source_invoice_id,kind,created_by) VALUES ($1,$2,'FT',$3)`, doc1, "10000000-0000-0000-0000-000000000001", "00000000-0000-0000-0000-000000000001")
	require.NoError(t, err)
	execFails(t, db, `INSERT INTO fiscal_documents(id,source_invoice_id,kind,created_by) VALUES ($1,$2,'FR',$3)`, doc2, "10000000-0000-0000-0000-000000000001", "00000000-0000-0000-0000-000000000001")

	_, err = db.Exec(`INSERT INTO fiscal_snapshots(id,fiscal_document_id,policy_version_id,issuer_profile_id,issuer,customer,billing_address,currency) VALUES ($1,$2,$3,$4,'{}','{}','{}','EUR')`, snapshot, doc1, policy, issuer)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO fiscal_document_lines(snapshot_id,position,description,quantity,unit_price,gross_amount,discount_value,discount_amount,net_amount,tax_rate,tax_amount,line_total,tax_treatment_code) VALUES ($1,1,'labor',1,10,10,0,0,10,23,2.3,12.3,'normal')`, snapshot)
	require.NoError(t, err)
	_, err = db.Exec(`UPDATE fiscal_snapshots SET frozen_at=now(),canonical_bytes='frozen',canonical_sha256=repeat('c',64) WHERE id=$1`, snapshot)
	require.NoError(t, err)
	// Snapshot freezing is authoritative even before the document leaves draft.
	execFails(t, db, `INSERT INTO fiscal_document_lines(snapshot_id,position,description,quantity,unit_price,gross_amount,discount_value,discount_amount,net_amount,tax_rate,tax_amount,line_total,tax_treatment_code) VALUES ($1,2,'part',1,5,5,0,0,5,23,1.15,6.15,'normal')`, snapshot)
	execFails(t, db, `UPDATE fiscal_document_lines SET description='changed' WHERE snapshot_id=$1`, snapshot)
	execFails(t, db, `DELETE FROM fiscal_document_lines WHERE snapshot_id=$1`, snapshot)
	_, err = db.Exec(`UPDATE fiscal_documents SET state='pending',provider_key='mock',connection_id=$2,intent_key=$3,issue_operation_key='fiscal:issue:one',provider_reference='MOCK-ONE',frozen_at=now(),finalized_by=created_by WHERE id=$1`, doc1, connection, uuid.New())
	require.NoError(t, err)
	execFails(t, db, `INSERT INTO fiscal_documents(id,source_invoice_id,kind,state,provider_key,connection_id,intent_key,issue_operation_key,frozen_at,created_by) VALUES ($1,'10000000-0000-0000-0000-000000000002','FT','pending','mock',$2,$3,'fiscal:issue:one',now(),'00000000-0000-0000-0000-000000000001')`, uuid.New(), connection, uuid.New())
	execFails(t, db, `INSERT INTO fiscal_documents(id,source_invoice_id,kind,state,provider_key,connection_id,intent_key,issue_operation_key,provider_reference,frozen_at,created_by) VALUES ($1,'10000000-0000-0000-0000-000000000002','FT','pending','mock',$2,$3,'fiscal:issue:two','MOCK-ONE',now(),'00000000-0000-0000-0000-000000000001')`, uuid.New(), connection, uuid.New())
	execFails(t, db, `UPDATE fiscal_documents SET provider_key='other' WHERE id=$1`, doc1)
	execFails(t, db, `UPDATE fiscal_snapshots SET customer='{"changed":true}' WHERE id=$1`, snapshot)
	execFails(t, db, `DELETE FROM fiscal_document_lines WHERE snapshot_id=$1`, snapshot)
	execFails(t, db, `DELETE FROM invoices WHERE id='10000000-0000-0000-0000-000000000001'`)

	_, err = db.Exec(`INSERT INTO fiscal_state_transitions(fiscal_document_id,from_state,to_state,operation,actor_type) VALUES ($1,'pending','dispatching','test','system')`, doc1)
	require.NoError(t, err)
	execFails(t, db, `UPDATE fiscal_state_transitions SET operation='rewritten' WHERE fiscal_document_id=$1`, doc1)

	_, err = db.Exec(`INSERT INTO fiscal_documents(id,source_invoice_id,kind,created_by) VALUES ($1,'10000000-0000-0000-0000-000000000002','FR','00000000-0000-0000-0000-000000000001')`, doc2)
	require.NoError(t, err)
	_, err = db.Exec(`DELETE FROM invoices WHERE id='10000000-0000-0000-0000-000000000002'`)
	require.NoError(t, err)

	execFails(t, db, `INSERT INTO fiscal_artifacts(id,fiscal_document_id,source_invoice_id,status,classification,storage_key,media_type) VALUES ($1,$2,'10000000-0000-0000-0000-000000000001','available','legal','x','text/plain')`, uuid.New(), doc1)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	var found int
	require.NoError(t, db.QueryRowContext(ctx, `SELECT count(*) FROM fiscal_documents WHERE id=$1`, doc2).Scan(&found))
	require.Zero(t, found)
}
