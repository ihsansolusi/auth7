CREATE TABLE IF NOT EXISTS role_permissions (
    id                               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    role_id                          UUID NOT NULL,
    permission_id                    UUID NOT NULL,
    created_at                       TIMESTAMPTZ NOT NULL,
    CONSTRAINT fk_role_permissions_role_id FOREIGN KEY (role_id) REFERENCES roles(id),
    CONSTRAINT fk_role_permissions_permission_id FOREIGN KEY (permission_id) REFERENCES permissions(id),
    CONSTRAINT uq_role_permissions_role_id_permission_id UNIQUE (role_id, permission_id)
);

CREATE INDEX IF NOT EXISTS idx_role_permissions_role_id ON role_permissions(role_id);
CREATE INDEX IF NOT EXISTS idx_role_permissions_permission_id ON role_permissions(permission_id);