-- Promote email to global-unique identity (mirror username pattern).
-- Both username and email are now unique across all orgs so login can derive
-- org_id from the identifier alone (no org_id input required at login).

ALTER TABLE users DROP CONSTRAINT IF EXISTS uq_users_org_email;
DROP INDEX IF EXISTS idx_users_email;

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email_global
    ON users(email) WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_users_org_email
    ON users(org_id, email) WHERE deleted_at IS NULL;
