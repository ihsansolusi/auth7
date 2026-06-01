DROP INDEX IF EXISTS idx_users_org_email;
DROP INDEX IF EXISTS idx_users_email_global;

CREATE INDEX IF NOT EXISTS idx_users_email ON users(org_id, email) WHERE deleted_at IS NULL;
ALTER TABLE users ADD CONSTRAINT uq_users_org_email UNIQUE (org_id, email);
