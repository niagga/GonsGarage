-- ✅ Ensure user.id is properly indexed
BEGIN;

CREATE INDEX IF NOT EXISTS idx_users_id ON users(id) WHERE deleted_at IS NULL;

COMMIT;