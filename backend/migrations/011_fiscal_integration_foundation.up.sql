-- Provider-neutral fiscal foundation. The migration runner owns the transaction.
ALTER TABLE invoices ADD COLUMN fiscal_eligibility varchar(32) NOT NULL DEFAULT 'legacy_unfiscalized';
ALTER TABLE invoices ADD CONSTRAINT invoices_fiscal_eligibility_ck CHECK (fiscal_eligibility IN ('legacy_unfiscalized','eligible'));

CREATE TABLE fiscal_policy_versions (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(), policy_key varchar(80) NOT NULL, version integer NOT NULL CHECK (version>0),
 schema_version integer NOT NULL, classification varchar(16) NOT NULL CHECK (classification IN ('legal','mock')),
 status varchar(16) NOT NULL CHECK (status IN ('draft','approved','retired')), config jsonb NOT NULL CHECK (jsonb_typeof(config)='object'),
 config_sha256 char(64) NOT NULL, approved_by uuid REFERENCES users(id) ON DELETE RESTRICT, approved_at timestamptz,
 created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(), UNIQUE(policy_key,version)
);
CREATE UNIQUE INDEX fiscal_policy_approved_uq ON fiscal_policy_versions(policy_key) WHERE status='approved';

CREATE TABLE fiscal_issuer_profiles (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(), scope_key varchar(80) NOT NULL DEFAULT 'default', version integer NOT NULL CHECK (version>0),
 status varchar(16) NOT NULL CHECK (status IN ('draft','approved','retired')), legal_identity jsonb NOT NULL CHECK (jsonb_typeof(legal_identity)='object'),
 billing_address jsonb NOT NULL CHECK (jsonb_typeof(billing_address)='object'), tax_profile jsonb NOT NULL CHECK (jsonb_typeof(tax_profile)='object'),
 series_config jsonb NOT NULL CHECK (jsonb_typeof(series_config)='object'), config_sha256 char(64) NOT NULL,
 approved_by uuid REFERENCES users(id) ON DELETE RESTRICT, approved_at timestamptz, created_at timestamptz NOT NULL DEFAULT now(),
 updated_at timestamptz NOT NULL DEFAULT now(), UNIQUE(scope_key,version)
);
CREATE UNIQUE INDEX fiscal_issuer_approved_uq ON fiscal_issuer_profiles(scope_key) WHERE status='approved';

CREATE TABLE fiscal_provider_connections (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(), scope_key varchar(80) NOT NULL DEFAULT 'default', provider_key varchar(40) NOT NULL,
 state varchar(24) NOT NULL CHECK (state IN ('disconnected','authorizing','connected','action_required','revoked')),
 provider_organization_ref varchar(255), granted_scopes text[] NOT NULL DEFAULT '{}', access_expires_at timestamptz,
 credential_ciphertext bytea, credential_nonce bytea, credential_key_version varchar(40), credential_format_version integer,
 last_verified_at timestamptz, connected_at timestamptz, revoked_at timestamptz, created_by uuid REFERENCES users(id) ON DELETE RESTRICT,
 updated_by uuid REFERENCES users(id) ON DELETE RESTRICT, created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(),
 version integer NOT NULL DEFAULT 1, UNIQUE(scope_key,provider_key),
 CHECK ((credential_ciphertext IS NULL AND credential_nonce IS NULL AND credential_key_version IS NULL AND credential_format_version IS NULL) OR
        (credential_ciphertext IS NOT NULL AND credential_nonce IS NOT NULL AND credential_key_version IS NOT NULL AND credential_format_version IS NOT NULL))
);
CREATE INDEX fiscal_connections_state_idx ON fiscal_provider_connections(provider_key,state);

CREATE TABLE fiscal_oauth_authorizations (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(), connection_id uuid NOT NULL REFERENCES fiscal_provider_connections(id) ON DELETE CASCADE,
 actor_id uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT, state_sha256 char(64) NOT NULL UNIQUE,
 pkce_ciphertext bytea, pkce_nonce bytea, pkce_key_version varchar(40), redirect_uri text NOT NULL, expires_at timestamptz NOT NULL,
 consumed_at timestamptz, created_at timestamptz NOT NULL DEFAULT now(),
 CHECK ((pkce_ciphertext IS NULL AND pkce_nonce IS NULL AND pkce_key_version IS NULL) OR
        (pkce_ciphertext IS NOT NULL AND pkce_nonce IS NOT NULL AND pkce_key_version IS NOT NULL))
);
CREATE INDEX fiscal_oauth_pending_idx ON fiscal_oauth_authorizations(connection_id,expires_at) WHERE consumed_at IS NULL;

CREATE TABLE fiscal_enablement_gates (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(), provider_key varchar(40) NOT NULL, environment varchar(40) NOT NULL,
 gate_key varchar(80) NOT NULL, status varchar(24) NOT NULL CHECK (status IN ('pending','satisfied','not_applicable')),
 evidence_ref text, decision_ref text, acceptance_ref text, rationale text, approved_by uuid REFERENCES users(id) ON DELETE RESTRICT,
 approved_at timestamptz, created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(),
 UNIQUE(provider_key,environment,gate_key)
);
CREATE INDEX fiscal_gates_status_idx ON fiscal_enablement_gates(provider_key,environment,status);
INSERT INTO fiscal_enablement_gates(provider_key,environment,gate_key,status) VALUES
 ('cloudware','production','sandbox_test_company','pending'),('cloudware','production','oauth_security','pending'),
 ('cloudware','production','idempotency_lookup','pending'),('cloudware','production','webhooks','pending'),
 ('cloudware','production','rate_limits_retry','pending'),('cloudware','production','pdf_lifetime_authority','pending'),
 ('cloudware','production','automatic_at_efatura','pending'),('cloudware','production','credentialed_response_behavior','pending');

CREATE TABLE fiscal_documents (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(), source_invoice_id uuid NOT NULL REFERENCES invoices(id) ON DELETE RESTRICT,
 intent_slot varchar(40) NOT NULL DEFAULT 'primary_sale', kind varchar(2) NOT NULL CHECK (kind IN ('FT','FR')),
 state varchar(32) NOT NULL DEFAULT 'draft' CHECK (state IN ('draft','pending','dispatching','issued','rejected','connection_action_required','retryable_failure','outcome_unknown','void_pending','void_outcome_unknown','voided')),
 version bigint NOT NULL DEFAULT 1, supersedes_document_id uuid REFERENCES fiscal_documents(id) ON DELETE RESTRICT, superseded_at timestamptz,
 provider_key varchar(40), connection_id uuid REFERENCES fiscal_provider_connections(id) ON DELETE RESTRICT, intent_key uuid,
 issue_operation_key varchar(160), void_operation_key varchar(160), provider_reference varchar(255), provider_number varchar(255),
 provider_confirmed_at timestamptz, issued_at timestamptz, voided_at timestamptz, frozen_at timestamptz,
 last_error_class varchar(32), last_error_code varchar(80), last_error_message text,
 created_by uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT, finalized_by uuid REFERENCES users(id) ON DELETE RESTRICT,
 created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(),
 CHECK (state='draft' OR (provider_key IS NOT NULL AND connection_id IS NOT NULL AND intent_key IS NOT NULL AND issue_operation_key IS NOT NULL AND frozen_at IS NOT NULL)),
 CHECK ((superseded_at IS NULL) OR state='rejected'), CHECK ((state NOT IN ('void_pending','void_outcome_unknown','voided')) OR void_operation_key IS NOT NULL)
);
CREATE UNIQUE INDEX fiscal_documents_current_intent_uq ON fiscal_documents(source_invoice_id,intent_slot) WHERE superseded_at IS NULL;
CREATE UNIQUE INDEX fiscal_documents_intent_key_uq ON fiscal_documents(intent_key) WHERE intent_key IS NOT NULL;
CREATE UNIQUE INDEX fiscal_documents_issue_key_uq ON fiscal_documents(issue_operation_key) WHERE issue_operation_key IS NOT NULL;
CREATE UNIQUE INDEX fiscal_documents_void_key_uq ON fiscal_documents(void_operation_key) WHERE void_operation_key IS NOT NULL;
CREATE UNIQUE INDEX fiscal_documents_provider_ref_uq ON fiscal_documents(provider_key,provider_reference) WHERE provider_reference IS NOT NULL;
CREATE INDEX fiscal_documents_source_idx ON fiscal_documents(source_invoice_id,created_at DESC);
CREATE INDEX fiscal_documents_state_idx ON fiscal_documents(state,updated_at);
CREATE INDEX fiscal_documents_connection_idx ON fiscal_documents(connection_id,state);

CREATE TABLE fiscal_snapshots (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(), fiscal_document_id uuid NOT NULL UNIQUE REFERENCES fiscal_documents(id) ON DELETE RESTRICT,
 schema_version integer NOT NULL DEFAULT 1, policy_version_id uuid REFERENCES fiscal_policy_versions(id) ON DELETE RESTRICT,
 issuer_profile_id uuid REFERENCES fiscal_issuer_profiles(id) ON DELETE RESTRICT, issuer jsonb NOT NULL CHECK (jsonb_typeof(issuer)='object'),
 customer jsonb NOT NULL CHECK (jsonb_typeof(customer)='object'), billing_address jsonb NOT NULL CHECK (jsonb_typeof(billing_address)='object'),
 currency char(3) NOT NULL, gross_total numeric(38,18) NOT NULL DEFAULT 0 CHECK (gross_total>=0),
 discount_total numeric(38,18) NOT NULL DEFAULT 0 CHECK (discount_total>=0), net_total numeric(38,18) NOT NULL DEFAULT 0 CHECK (net_total>=0),
 tax_total numeric(38,18) NOT NULL DEFAULT 0 CHECK (tax_total>=0), rounding_adjustment numeric(38,18) NOT NULL DEFAULT 0,
 payable_total numeric(38,18) NOT NULL DEFAULT 0 CHECK (payable_total>=0), canonical_bytes bytea, canonical_sha256 char(64), frozen_at timestamptz,
 created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(),
 CHECK ((frozen_at IS NULL AND canonical_bytes IS NULL AND canonical_sha256 IS NULL) OR (frozen_at IS NOT NULL AND canonical_bytes IS NOT NULL AND canonical_sha256 IS NOT NULL))
);

CREATE TABLE fiscal_document_lines (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(), snapshot_id uuid NOT NULL REFERENCES fiscal_snapshots(id) ON DELETE RESTRICT,
 position integer NOT NULL CHECK (position>0), description text NOT NULL, unit_code varchar(40) NOT NULL DEFAULT 'unit',
 quantity numeric(38,18) NOT NULL CHECK (quantity>0), unit_price numeric(38,18) NOT NULL CHECK (unit_price>=0),
 gross_amount numeric(38,18) NOT NULL CHECK (gross_amount>=0), discount_kind varchar(16) NOT NULL DEFAULT 'none' CHECK (discount_kind IN ('none','amount','percent')),
 discount_value numeric(38,18) NOT NULL CHECK (discount_value>=0), discount_amount numeric(38,18) NOT NULL CHECK (discount_amount>=0),
 net_amount numeric(38,18) NOT NULL CHECK (net_amount>=0), tax_rate numeric(38,18) NOT NULL CHECK (tax_rate>=0),
 tax_amount numeric(38,18) NOT NULL CHECK (tax_amount>=0), line_total numeric(38,18) NOT NULL CHECK (line_total>=0),
 tax_treatment_code varchar(80) NOT NULL, exemption_code varchar(80), exemption_reason text, source_type varchar(16) CHECK (source_type IN ('repair','part')),
 source_id uuid, created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(),
 UNIQUE(snapshot_id,position), CHECK ((source_type IS NULL)=(source_id IS NULL)), CHECK (discount_amount<=gross_amount)
);
CREATE INDEX fiscal_lines_source_idx ON fiscal_document_lines(source_type,source_id) WHERE source_id IS NOT NULL;

CREATE TABLE fiscal_state_transitions (
 id bigserial PRIMARY KEY, fiscal_document_id uuid NOT NULL REFERENCES fiscal_documents(id) ON DELETE RESTRICT,
 from_state varchar(32) NOT NULL, to_state varchar(32) NOT NULL, operation varchar(80) NOT NULL,
 reason_class varchar(32), reason_code varchar(80), reason_detail text, actor_type varchar(16) NOT NULL CHECK (actor_type IN ('user','worker','system')),
 actor_id uuid, correlation_key varchar(160), created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX fiscal_transitions_document_idx ON fiscal_state_transitions(fiscal_document_id,id);

CREATE TABLE fiscal_outbox_events (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(), fiscal_document_id uuid NOT NULL REFERENCES fiscal_documents(id) ON DELETE RESTRICT,
 event_type varchar(24) NOT NULL CHECK (event_type IN ('issue','reconcile_issue','void','reconcile_void','recover_artifact')),
 operation_key varchar(160) NOT NULL, sequence_no integer NOT NULL CHECK (sequence_no>0), status varchar(16) NOT NULL DEFAULT 'ready' CHECK (status IN ('ready','leased','completed','dead')),
 available_at timestamptz NOT NULL DEFAULT now(), lease_owner varchar(160), lease_token uuid, lease_expires_at timestamptz,
 claim_count integer NOT NULL DEFAULT 0 CHECK (claim_count>=0), last_safe_error text, created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(),
 UNIQUE(fiscal_document_id,event_type,sequence_no), CHECK ((status='leased' AND lease_owner IS NOT NULL AND lease_token IS NOT NULL AND lease_expires_at IS NOT NULL) OR
 (status<>'leased' AND lease_owner IS NULL AND lease_token IS NULL AND lease_expires_at IS NULL))
);
CREATE UNIQUE INDEX fiscal_outbox_active_uq ON fiscal_outbox_events(fiscal_document_id) WHERE status IN ('ready','leased');
CREATE INDEX fiscal_outbox_claim_idx ON fiscal_outbox_events(status,available_at,lease_expires_at,created_at);

CREATE TABLE fiscal_provider_attempts (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(), fiscal_document_id uuid NOT NULL REFERENCES fiscal_documents(id) ON DELETE RESTRICT,
 outbox_event_id uuid NOT NULL REFERENCES fiscal_outbox_events(id) ON DELETE RESTRICT, provider_key varchar(40) NOT NULL,
 operation varchar(24) NOT NULL CHECK (operation IN ('issue','reconcile_issue','void','reconcile_void','fetch_artifact')),
 operation_key varchar(160) NOT NULL, attempt_no integer NOT NULL CHECK (attempt_no>0), status varchar(16) NOT NULL CHECK (status IN ('started','succeeded','failed','unknown')),
 classification varchar(24) CHECK (classification IN ('validation','authorization','transient','rate_limit','permanent','ambiguous')),
 definitive boolean NOT NULL DEFAULT false, request_sha256 char(64) NOT NULL, safe_provider_code varchar(80), safe_provider_message text,
 diagnostics jsonb NOT NULL DEFAULT '{}' CHECK (jsonb_typeof(diagnostics)='object'), provider_reference varchar(255),
 started_at timestamptz NOT NULL DEFAULT now(), finished_at timestamptz, UNIQUE(fiscal_document_id,operation,attempt_no)
);
CREATE INDEX fiscal_attempts_document_idx ON fiscal_provider_attempts(fiscal_document_id,started_at);
CREATE INDEX fiscal_attempts_event_idx ON fiscal_provider_attempts(outbox_event_id);

CREATE TABLE fiscal_artifacts (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(), fiscal_document_id uuid NOT NULL REFERENCES fiscal_documents(id) ON DELETE RESTRICT,
 source_invoice_id uuid NOT NULL REFERENCES invoices(id) ON DELETE RESTRICT, kind varchar(24) NOT NULL DEFAULT 'provider_pdf' CHECK (kind='provider_pdf'),
 status varchar(16) NOT NULL CHECK (status IN ('pending','available','unavailable','compromised')),
 classification varchar(16) NOT NULL CHECK (classification IN ('legal','mock')), storage_key text UNIQUE, media_type varchar(100), byte_size bigint,
 sha256 char(64), provider_reference varchar(255), provider_version varchar(80), created_at timestamptz NOT NULL DEFAULT now(),
 available_at timestamptz, last_verified_at timestamptz, last_error_code varchar(80), last_error_message text, UNIQUE(fiscal_document_id,kind),
 CHECK (status<>'available' OR (storage_key IS NOT NULL AND media_type='application/pdf' AND byte_size>0 AND sha256 IS NOT NULL AND available_at IS NOT NULL))
);
ALTER TABLE fiscal_outbox_events ADD COLUMN artifact_id uuid REFERENCES fiscal_artifacts(id) ON DELETE RESTRICT;
CREATE INDEX fiscal_artifacts_invoice_idx ON fiscal_artifacts(source_invoice_id,status);
CREATE INDEX fiscal_artifacts_document_idx ON fiscal_artifacts(fiscal_document_id,status);

CREATE TABLE fiscal_artifact_access_log (
 id bigserial PRIMARY KEY, artifact_id uuid NOT NULL REFERENCES fiscal_artifacts(id) ON DELETE RESTRICT,
 fiscal_document_id uuid NOT NULL REFERENCES fiscal_documents(id) ON DELETE RESTRICT, source_invoice_id uuid NOT NULL REFERENCES invoices(id) ON DELETE RESTRICT,
 actor_id uuid NOT NULL, actor_role varchar(40) NOT NULL, outcome varchar(16) NOT NULL CHECK (outcome IN ('allowed','denied','unavailable')),
 request_correlation_id varchar(160), ip_hash char(64), created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX fiscal_artifact_access_idx ON fiscal_artifact_access_log(artifact_id,created_at DESC);

CREATE TABLE fiscal_mock_operations (
 operation_key varchar(160) PRIMARY KEY, scenario varchar(80) NOT NULL, canonical_sha256 char(64) NOT NULL,
 provider_reference varchar(255) NOT NULL UNIQUE, result jsonb NOT NULL CHECK (jsonb_typeof(result)='object'), pdf_sha256 char(64),
 voided boolean NOT NULL DEFAULT false, created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE FUNCTION fiscal_append_only() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION '% is append-only',TG_TABLE_NAME USING ERRCODE='55000'; END $$;
CREATE TRIGGER fiscal_transitions_append_only BEFORE UPDATE OR DELETE ON fiscal_state_transitions FOR EACH ROW EXECUTE FUNCTION fiscal_append_only();
CREATE TRIGGER fiscal_access_log_append_only BEFORE UPDATE OR DELETE ON fiscal_artifact_access_log FOR EACH ROW EXECUTE FUNCTION fiscal_append_only();
CREATE FUNCTION fiscal_attempt_history() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN
 IF TG_OP='DELETE' OR OLD.status<>'started' THEN RAISE EXCEPTION 'completed fiscal attempts are append-only' USING ERRCODE='55000'; END IF; RETURN NEW; END $$;
CREATE TRIGGER fiscal_attempts_append_only BEFORE UPDATE OR DELETE ON fiscal_provider_attempts FOR EACH ROW EXECUTE FUNCTION fiscal_attempt_history();

CREATE FUNCTION fiscal_document_immutable() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN
 IF OLD.state<>'draft' AND (TG_OP='DELETE' OR ROW(NEW.kind,NEW.provider_key,NEW.connection_id,NEW.intent_key,NEW.issue_operation_key) IS DISTINCT FROM ROW(OLD.kind,OLD.provider_key,OLD.connection_id,OLD.intent_key,OLD.issue_operation_key)) THEN
  RAISE EXCEPTION 'frozen fiscal document identity is immutable' USING ERRCODE='55000'; END IF; RETURN CASE WHEN TG_OP='DELETE' THEN OLD ELSE NEW END; END $$;
CREATE TRIGGER fiscal_documents_frozen BEFORE UPDATE OR DELETE ON fiscal_documents FOR EACH ROW EXECUTE FUNCTION fiscal_document_immutable();

CREATE FUNCTION fiscal_snapshot_immutable() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE document_id uuid; document_state varchar(32); snapshot_frozen timestamptz; snapshot_id uuid;
BEGIN
 IF TG_TABLE_NAME='fiscal_snapshots' THEN
  document_id:=OLD.fiscal_document_id; snapshot_frozen:=OLD.frozen_at;
 ELSE
  snapshot_id:=CASE WHEN TG_OP='INSERT' THEN NEW.snapshot_id ELSE OLD.snapshot_id END;
  SELECT s.fiscal_document_id,s.frozen_at INTO document_id,snapshot_frozen FROM fiscal_snapshots s WHERE s.id=snapshot_id;
 END IF;
 SELECT state INTO document_state FROM fiscal_documents WHERE id=document_id;
 IF document_state<>'draft' OR snapshot_frozen IS NOT NULL THEN RAISE EXCEPTION 'frozen fiscal input is immutable' USING ERRCODE='55000'; END IF;
 RETURN CASE WHEN TG_OP='DELETE' THEN OLD ELSE NEW END;
END $$;
CREATE TRIGGER fiscal_snapshots_immutable BEFORE UPDATE OR DELETE ON fiscal_snapshots FOR EACH ROW EXECUTE FUNCTION fiscal_snapshot_immutable();
CREATE TRIGGER fiscal_lines_immutable BEFORE INSERT OR UPDATE OR DELETE ON fiscal_document_lines FOR EACH ROW EXECUTE FUNCTION fiscal_snapshot_immutable();

CREATE FUNCTION fiscal_config_immutable() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE referenced boolean;
BEGIN
 IF TG_TABLE_NAME='fiscal_policy_versions' THEN SELECT EXISTS(SELECT 1 FROM fiscal_snapshots WHERE policy_version_id=OLD.id) INTO referenced;
 ELSE SELECT EXISTS(SELECT 1 FROM fiscal_snapshots WHERE issuer_profile_id=OLD.id) INTO referenced; END IF;
 IF OLD.status='approved' OR referenced THEN RAISE EXCEPTION 'approved or referenced fiscal configuration is immutable' USING ERRCODE='55000'; END IF;
 RETURN CASE WHEN TG_OP='DELETE' THEN OLD ELSE NEW END;
END $$;
CREATE TRIGGER fiscal_policy_immutable BEFORE UPDATE OR DELETE ON fiscal_policy_versions FOR EACH ROW EXECUTE FUNCTION fiscal_config_immutable();
CREATE TRIGGER fiscal_issuer_immutable BEFORE UPDATE OR DELETE ON fiscal_issuer_profiles FOR EACH ROW EXECUTE FUNCTION fiscal_config_immutable();

CREATE FUNCTION fiscal_guard_transition() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF NEW.state='pending' AND NOT EXISTS(SELECT 1 FROM fiscal_snapshots s JOIN fiscal_policy_versions p ON p.id=s.policy_version_id JOIN fiscal_issuer_profiles i ON i.id=s.issuer_profile_id
  WHERE s.fiscal_document_id=OLD.id AND s.frozen_at IS NOT NULL AND s.canonical_bytes IS NOT NULL AND s.canonical_sha256 IS NOT NULL AND p.status='approved' AND i.status='approved') THEN
  RAISE EXCEPTION 'pending fiscal document requires an approved frozen snapshot' USING ERRCODE='23514';
 END IF;
 IF NOT ((OLD.state='draft' AND NEW.state='pending') OR (OLD.state='pending' AND NEW.state='dispatching') OR
  (OLD.state='dispatching' AND NEW.state IN ('issued','rejected','connection_action_required','retryable_failure','outcome_unknown')) OR
  (OLD.state IN ('retryable_failure','connection_action_required') AND NEW.state='pending') OR
  (OLD.state='outcome_unknown' AND NEW.state IN ('issued','rejected','retryable_failure','outcome_unknown')) OR
  (OLD.state='issued' AND NEW.state='void_pending') OR
  (OLD.state='void_pending' AND NEW.state IN ('voided','issued','void_outcome_unknown')) OR
  (OLD.state='void_outcome_unknown' AND NEW.state IN ('voided','issued','void_outcome_unknown'))) THEN
  RAISE EXCEPTION 'forbidden fiscal transition % -> %',OLD.state,NEW.state USING ERRCODE='55000';
 END IF;
 INSERT INTO fiscal_state_transitions(fiscal_document_id,from_state,to_state,operation,actor_type)
 VALUES(OLD.id,OLD.state,NEW.state,coalesce(nullif(current_setting('fiscal.operation',true),''),'database_transition'),
        coalesce(nullif(current_setting('fiscal.actor_type',true),''),'system'));
 RETURN NEW;
END $$;
CREATE TRIGGER fiscal_documents_transition BEFORE UPDATE OF state ON fiscal_documents FOR EACH ROW WHEN (OLD.state IS DISTINCT FROM NEW.state) EXECUTE FUNCTION fiscal_guard_transition();

CREATE FUNCTION fiscal_protect_invoice_delete() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF EXISTS(SELECT 1 FROM fiscal_documents WHERE source_invoice_id=OLD.id AND state<>'draft') THEN
  RAISE EXCEPTION 'invoice has retained fiscal history' USING ERRCODE='23503';
 END IF;
 DELETE FROM fiscal_document_lines WHERE snapshot_id IN (SELECT id FROM fiscal_snapshots WHERE fiscal_document_id IN (SELECT id FROM fiscal_documents WHERE source_invoice_id=OLD.id));
 DELETE FROM fiscal_snapshots WHERE fiscal_document_id IN (SELECT id FROM fiscal_documents WHERE source_invoice_id=OLD.id);
 DELETE FROM fiscal_documents WHERE source_invoice_id=OLD.id;
 RETURN OLD;
END $$;
CREATE TRIGGER invoices_protect_fiscal_history BEFORE DELETE ON invoices FOR EACH ROW EXECUTE FUNCTION fiscal_protect_invoice_delete();
