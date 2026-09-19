package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

type fiscalTableRequirement struct {
	table, columns, constraints, indexes, triggers string
}

// This is the migration 011 contract. Keep it exhaustive so startup cannot
// enable fiscal behavior against a partially restored or manually altered schema.
var fiscalSchemaRequirements = []fiscalTableRequirement{
	{"invoices", "fiscal_eligibility", "invoices_fiscal_eligibility_ck", "", "invoices_protect_fiscal_history"},
	{"fiscal_policy_versions", "id,policy_key,version,schema_version,classification,status,config,config_sha256,approved_by,approved_at,created_at,updated_at", "fiscal_policy_versions_approved_by_fkey,fiscal_policy_versions_classification_check,fiscal_policy_versions_config_check,fiscal_policy_versions_pkey,fiscal_policy_versions_policy_key_version_key,fiscal_policy_versions_status_check,fiscal_policy_versions_version_check", "fiscal_policy_approved_uq", "fiscal_policy_immutable"},
	{"fiscal_issuer_profiles", "id,scope_key,version,status,legal_identity,billing_address,tax_profile,series_config,config_sha256,approved_by,approved_at,created_at,updated_at", "fiscal_issuer_profiles_approved_by_fkey,fiscal_issuer_profiles_billing_address_check,fiscal_issuer_profiles_legal_identity_check,fiscal_issuer_profiles_pkey,fiscal_issuer_profiles_scope_key_version_key,fiscal_issuer_profiles_series_config_check,fiscal_issuer_profiles_status_check,fiscal_issuer_profiles_tax_profile_check,fiscal_issuer_profiles_version_check", "fiscal_issuer_approved_uq", "fiscal_issuer_immutable"},
	{"fiscal_provider_connections", "id,scope_key,provider_key,state,provider_organization_ref,granted_scopes,access_expires_at,credential_ciphertext,credential_nonce,credential_key_version,credential_format_version,last_verified_at,connected_at,revoked_at,created_by,updated_by,created_at,updated_at,version", "fiscal_provider_connections_check,fiscal_provider_connections_created_by_fkey,fiscal_provider_connections_pkey,fiscal_provider_connections_scope_key_provider_key_key,fiscal_provider_connections_state_check,fiscal_provider_connections_updated_by_fkey", "fiscal_connections_state_idx", ""},
	{"fiscal_oauth_authorizations", "id,connection_id,actor_id,state_sha256,pkce_ciphertext,pkce_nonce,pkce_key_version,redirect_uri,expires_at,consumed_at,created_at", "fiscal_oauth_authorizations_actor_id_fkey,fiscal_oauth_authorizations_check,fiscal_oauth_authorizations_connection_id_fkey,fiscal_oauth_authorizations_pkey,fiscal_oauth_authorizations_state_sha256_key", "fiscal_oauth_pending_idx", ""},
	{"fiscal_enablement_gates", "id,provider_key,environment,gate_key,status,evidence_ref,decision_ref,acceptance_ref,rationale,approved_by,approved_at,created_at,updated_at", "fiscal_enablement_gates_approved_by_fkey,fiscal_enablement_gates_pkey,fiscal_enablement_gates_provider_key_environment_gate_key_key,fiscal_enablement_gates_status_check", "fiscal_gates_status_idx", ""},
	{"fiscal_documents", "id,source_invoice_id,intent_slot,kind,state,version,supersedes_document_id,superseded_at,provider_key,connection_id,intent_key,issue_operation_key,void_operation_key,provider_reference,provider_number,provider_confirmed_at,issued_at,voided_at,frozen_at,last_error_class,last_error_code,last_error_message,created_by,finalized_by,created_at,updated_at", "fiscal_documents_check,fiscal_documents_check1,fiscal_documents_check2,fiscal_documents_connection_id_fkey,fiscal_documents_created_by_fkey,fiscal_documents_finalized_by_fkey,fiscal_documents_kind_check,fiscal_documents_pkey,fiscal_documents_source_invoice_id_fkey,fiscal_documents_state_check,fiscal_documents_supersedes_document_id_fkey", "fiscal_documents_connection_idx,fiscal_documents_current_intent_uq,fiscal_documents_intent_key_uq,fiscal_documents_issue_key_uq,fiscal_documents_provider_ref_uq,fiscal_documents_source_idx,fiscal_documents_state_idx,fiscal_documents_void_key_uq", "fiscal_documents_frozen,fiscal_documents_transition"},
	{"fiscal_snapshots", "id,fiscal_document_id,schema_version,policy_version_id,issuer_profile_id,issuer,customer,billing_address,currency,gross_total,discount_total,net_total,tax_total,rounding_adjustment,payable_total,canonical_bytes,canonical_sha256,frozen_at,created_at,updated_at", "fiscal_snapshots_billing_address_check,fiscal_snapshots_check,fiscal_snapshots_customer_check,fiscal_snapshots_discount_total_check,fiscal_snapshots_fiscal_document_id_fkey,fiscal_snapshots_fiscal_document_id_key,fiscal_snapshots_gross_total_check,fiscal_snapshots_issuer_check,fiscal_snapshots_issuer_profile_id_fkey,fiscal_snapshots_net_total_check,fiscal_snapshots_payable_total_check,fiscal_snapshots_pkey,fiscal_snapshots_policy_version_id_fkey,fiscal_snapshots_tax_total_check", "", "fiscal_snapshots_immutable"},
	{"fiscal_document_lines", "id,snapshot_id,position,description,unit_code,quantity,unit_price,gross_amount,discount_kind,discount_value,discount_amount,net_amount,tax_rate,tax_amount,line_total,tax_treatment_code,exemption_code,exemption_reason,source_type,source_id,created_at,updated_at", "fiscal_document_lines_check,fiscal_document_lines_check1,fiscal_document_lines_discount_amount_check,fiscal_document_lines_discount_kind_check,fiscal_document_lines_discount_value_check,fiscal_document_lines_gross_amount_check,fiscal_document_lines_line_total_check,fiscal_document_lines_net_amount_check,fiscal_document_lines_pkey,fiscal_document_lines_position_check,fiscal_document_lines_quantity_check,fiscal_document_lines_snapshot_id_fkey,fiscal_document_lines_snapshot_id_position_key,fiscal_document_lines_source_type_check,fiscal_document_lines_tax_amount_check,fiscal_document_lines_tax_rate_check,fiscal_document_lines_unit_price_check", "fiscal_lines_source_idx", "fiscal_lines_immutable"},
	{"fiscal_state_transitions", "id,fiscal_document_id,from_state,to_state,operation,reason_class,reason_code,reason_detail,actor_type,actor_id,correlation_key,created_at", "fiscal_state_transitions_actor_type_check,fiscal_state_transitions_fiscal_document_id_fkey,fiscal_state_transitions_pkey", "fiscal_transitions_document_idx", "fiscal_transitions_append_only"},
	{"fiscal_outbox_events", "id,fiscal_document_id,event_type,operation_key,sequence_no,status,available_at,lease_owner,lease_token,lease_expires_at,claim_count,last_safe_error,created_at,updated_at,artifact_id", "fiscal_outbox_events_artifact_id_fkey,fiscal_outbox_events_check,fiscal_outbox_events_claim_count_check,fiscal_outbox_events_event_type_check,fiscal_outbox_events_fiscal_document_id_event_type_sequence_key,fiscal_outbox_events_fiscal_document_id_fkey,fiscal_outbox_events_pkey,fiscal_outbox_events_sequence_no_check,fiscal_outbox_events_status_check", "fiscal_outbox_active_uq,fiscal_outbox_claim_idx", ""},
	{"fiscal_provider_attempts", "id,fiscal_document_id,outbox_event_id,provider_key,operation,operation_key,attempt_no,status,classification,definitive,request_sha256,safe_provider_code,safe_provider_message,diagnostics,provider_reference,started_at,finished_at", "fiscal_provider_attempts_attempt_no_check,fiscal_provider_attempts_classification_check,fiscal_provider_attempts_diagnostics_check,fiscal_provider_attempts_fiscal_document_id_fkey,fiscal_provider_attempts_fiscal_document_id_operation_attem_key,fiscal_provider_attempts_operation_check,fiscal_provider_attempts_outbox_event_id_fkey,fiscal_provider_attempts_pkey,fiscal_provider_attempts_status_check", "fiscal_attempts_document_idx,fiscal_attempts_event_idx", "fiscal_attempts_append_only"},
	{"fiscal_artifacts", "id,fiscal_document_id,source_invoice_id,kind,status,classification,storage_key,media_type,byte_size,sha256,provider_reference,provider_version,created_at,available_at,last_verified_at,last_error_code,last_error_message", "fiscal_artifacts_check,fiscal_artifacts_classification_check,fiscal_artifacts_fiscal_document_id_fkey,fiscal_artifacts_fiscal_document_id_kind_key,fiscal_artifacts_kind_check,fiscal_artifacts_pkey,fiscal_artifacts_source_invoice_id_fkey,fiscal_artifacts_status_check,fiscal_artifacts_storage_key_key", "fiscal_artifacts_document_idx,fiscal_artifacts_invoice_idx", ""},
	{"fiscal_artifact_access_log", "id,artifact_id,fiscal_document_id,source_invoice_id,actor_id,actor_role,outcome,request_correlation_id,ip_hash,created_at", "fiscal_artifact_access_log_artifact_id_fkey,fiscal_artifact_access_log_fiscal_document_id_fkey,fiscal_artifact_access_log_outcome_check,fiscal_artifact_access_log_pkey,fiscal_artifact_access_log_source_invoice_id_fkey", "fiscal_artifact_access_idx", "fiscal_access_log_append_only"},
	{"fiscal_mock_operations", "operation_key,scenario,canonical_sha256,provider_reference,result,pdf_sha256,voided,created_at,updated_at", "fiscal_mock_operations_pkey,fiscal_mock_operations_provider_reference_key,fiscal_mock_operations_result_check", "", ""},
}

var requiredFiscalSchemaObjects = buildRequiredFiscalSchemaObjects()

func buildRequiredFiscalSchemaObjects() []string {
	objects := []string{"migration:011_fiscal_integration_foundation"}
	for _, requirement := range fiscalSchemaRequirements {
		if strings.HasPrefix(requirement.table, "fiscal_") {
			objects = append(objects, "table:"+requirement.table)
		}
		groups := []struct{ kind, names string }{
			{"column", requirement.columns}, {"constraint", requirement.constraints},
			{"index", requirement.indexes}, {"trigger", requirement.triggers},
		}
		for _, group := range groups {
			for _, name := range strings.Split(group.names, ",") {
				if name != "" {
					objects = append(objects, group.kind+":"+requirement.table+"."+name)
				}
			}
		}
	}
	return objects
}

// RequiredFiscalSchemaObjects returns the migration objects required before fiscal capability is enabled.
func RequiredFiscalSchemaObjects() []string {
	return append([]string(nil), requiredFiscalSchemaObjects...)
}

// MissingFiscalSchemaObjects compares an inspected object set with the required schema contract.
func MissingFiscalSchemaObjects(present map[string]bool) []string {
	missing := make([]string, 0)
	for _, object := range requiredFiscalSchemaObjects {
		if !present[object] {
			missing = append(missing, object)
		}
	}
	return missing
}

// VerifyFiscalSchema fails closed when migration 011 or a required PostgreSQL object is absent.
func VerifyFiscalSchema(ctx context.Context, db *sql.DB) error {
	if db == nil {
		return fmt.Errorf("verify fiscal schema: database is nil")
	}
	present := make(map[string]bool, len(requiredFiscalSchemaObjects))
	var migrationPresent bool
	if err := db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version='011_fiscal_integration_foundation')`).Scan(&migrationPresent); err != nil {
		return fmt.Errorf("verify fiscal schema migration: %w", err)
	}
	present["migration:011_fiscal_integration_foundation"] = migrationPresent

	catalogQueries := []struct{ kind, query string }{
		{"table", `SELECT c.relname,'' FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname=current_schema() AND c.relkind='r'`},
		{"column", `SELECT table_name,column_name FROM information_schema.columns WHERE table_schema=current_schema()`},
		{"constraint", `SELECT c.relname,x.conname FROM pg_constraint x JOIN pg_class c ON c.oid=x.conrelid JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname=current_schema()`},
		{"index", `SELECT c.relname,i.relname FROM pg_index x JOIN pg_class i ON i.oid=x.indexrelid JOIN pg_class c ON c.oid=x.indrelid JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname=current_schema()`},
		{"trigger", `SELECT c.relname,x.tgname FROM pg_trigger x JOIN pg_class c ON c.oid=x.tgrelid JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname=current_schema() AND NOT x.tgisinternal`},
	}
	for _, catalog := range catalogQueries {
		rows, err := db.QueryContext(ctx, catalog.query)
		if err != nil {
			return fmt.Errorf("verify fiscal schema %s catalog: %w", catalog.kind, err)
		}
		for rows.Next() {
			var table, name string
			if err := rows.Scan(&table, &name); err != nil {
				rows.Close()
				return fmt.Errorf("verify fiscal schema %s catalog row: %w", catalog.kind, err)
			}
			identity := table
			if name != "" {
				identity += "." + name
			}
			present[catalog.kind+":"+identity] = true
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return fmt.Errorf("verify fiscal schema %s catalog rows: %w", catalog.kind, err)
		}
		if err := rows.Close(); err != nil {
			return fmt.Errorf("verify fiscal schema %s catalog close: %w", catalog.kind, err)
		}
	}
	if missing := MissingFiscalSchemaObjects(present); len(missing) != 0 {
		return fmt.Errorf("fiscal schema is incomplete: missing %s", strings.Join(missing, ", "))
	}
	return nil
}
