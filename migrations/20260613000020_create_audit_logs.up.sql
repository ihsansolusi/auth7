CREATE TABLE IF NOT EXISTS audit_logs (
    id                               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id                           UUID NOT NULL,
    actor_id                         UUID NOT NULL,
    actor_email                      VARCHAR(255) NOT NULL DEFAULT '',
    action                           VARCHAR(100) NOT NULL DEFAULT '',
    resource_type                    VARCHAR(50) NOT NULL DEFAULT '',
    resource_id                      VARCHAR(255) NOT NULL DEFAULT '',
    old_value                        JSONB NOT NULL DEFAULT '{}',
    new_value                        JSONB NOT NULL DEFAULT '{}',
    ip_address                       VARCHAR(50) NOT NULL DEFAULT '',
    user_agent                       VARCHAR(500) NOT NULL DEFAULT '',
    created_at                       TIMESTAMPTZ NOT NULL,
    CONSTRAINT fk_audit_logs_org_id FOREIGN KEY (org_id) REFERENCES organizations(id)
);

CREATE INDEX IF NOT EXISTS idx_audit_logs_org_id ON audit_logs(org_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_actor_id ON audit_logs(actor_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_org_id_created_at
    ON audit_logs(org_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_action
    ON audit_logs(action);
CREATE INDEX IF NOT EXISTS idx_audit_logs_resource_type_resource_id
    ON audit_logs(resource_type, resource_id);