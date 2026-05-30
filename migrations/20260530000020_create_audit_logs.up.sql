CREATE TABLE IF NOT EXISTS audit_logs (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id        UUID        NOT NULL REFERENCES organizations(id),
    actor_id      VARCHAR(36) NOT NULL DEFAULT '',
    actor_email   VARCHAR(255) NOT NULL DEFAULT '',
    action        VARCHAR(100) NOT NULL,
    resource_type VARCHAR(50)  NOT NULL DEFAULT '',
    resource_id   VARCHAR(255) NOT NULL DEFAULT '',
    old_value     JSONB,
    new_value     JSONB,
    ip_address    VARCHAR(50)  NOT NULL DEFAULT '',
    user_agent    VARCHAR(500) NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_audit_logs_org_id     ON audit_logs(org_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_actor_id   ON audit_logs(actor_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_action     ON audit_logs(action);
CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs(created_at DESC);
