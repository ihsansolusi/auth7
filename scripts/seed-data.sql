-- Core7 Seed Data for E2E Testing
-- Run after migrations: psql -h postgres -U core7 -d auth7 -f seed-data.sql

-- ──────────────────────────────────────────────
-- 1. Organization
-- ──────────────────────────────────────────────
INSERT INTO organizations (id, code, name, domain, status, settings)
VALUES (
    '00000000-0000-0000-0000-000000000001',
    'BANK-DEMO',
    'Bank Demo Indonesia',
    'bank-demo.co.id',
    'active',
    '{"mfa_required": true, "max_sessions": 3}'::jsonb
)
ON CONFLICT (id) DO NOTHING;

-- ──────────────────────────────────────────────
-- 2. Branch Types
-- ──────────────────────────────────────────────
INSERT INTO branch_types (id, org_id, code, label, short_code, level, is_operational, can_have_children, sort_order)
VALUES
    ('00000000-0000-0000-0000-000000000101', '00000000-0000-0000-0000-000000000001', 'KC', 'Kantor Cabang', 'KC', 1, true, true, 1),
    ('00000000-0000-0000-0000-000000000102', '00000000-0000-0000-0000-000000000001', 'KCP', 'Kantor Cabang Pembantu', 'KCP', 2, true, false, 2),
    ('00000000-0000-0000-0000-000000000103', '00000000-0000-0000-0000-000000000001', 'KAS', 'Kantor Kas', 'KAS', 3, true, false, 3)
ON CONFLICT (id) DO NOTHING;

-- ──────────────────────────────────────────────
-- 3. Branches
-- ──────────────────────────────────────────────
INSERT INTO branches (id, org_id, branch_type_id, code, name, status)
VALUES
    ('00000000-0000-0000-0000-000000000201', '00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000101', 'KC-BDG-001', 'Kantor Cabang Bandung', 'active'),
    ('00000000-0000-0000-0000-000000000202', '00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000101', 'KC-JKT-001', 'Kantor Cabang Jakarta', 'active'),
    ('00000000-0000-0000-0000-000000000203', '00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000102', 'KCP-DGO-001', 'KCP Dago', 'active'),
    ('00000000-0000-0000-0000-000000000204', '00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000103', 'KAS-CIM-001', 'Kantor Kas Cimahi', 'active')
ON CONFLICT (id) DO NOTHING;

-- ──────────────────────────────────────────────
-- 4. Branch Hierarchies
-- ──────────────────────────────────────────────
INSERT INTO branch_hierarchies (id, org_id, parent_id, child_id, path, depth)
VALUES
    ('00000000-0000-0000-0000-000000000301', '00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000201', '00000000-0000-0000-0000-000000000203', '/KC-BDG-001/KCP-DGO-001', 1),
    ('00000000-0000-0000-0000-000000000302', '00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000202', '00000000-0000-0000-0000-000000000204', '/KC-JKT-001/KAS-CIM-001', 1)
ON CONFLICT (id) DO NOTHING;

-- ──────────────────────────────────────────────
-- 5. Users (password: "Password123!" for all test users)
-- ──────────────────────────────────────────────
-- Argon2id hash for "Password123!"
-- Generated with: argon2id, m=65536, t=3, p=4
INSERT INTO users (id, org_id, email, username, first_name, last_name, status, email_verified, mfa_enabled, created_at)
VALUES
    ('00000000-0000-0000-0000-000000000401', '00000000-0000-0000-0000-000000000001', 'admin@bank-demo.co.id', 'admin', 'Super', 'Admin', 'active', true, true, NOW()),
    ('00000000-0000-0000-0000-000000000402', '00000000-0000-0000-0000-000000000001', 'john@bank-demo.co.id', 'john.doe', 'John', 'Doe', 'active', true, false, NOW()),
    ('00000000-0000-0000-0000-000000000403', '00000000-0000-0000-0000-000000000001', 'jane@bank-demo.co.id', 'jane.smith', 'Jane', 'Smith', 'active', true, true, NOW()),
    ('00000000-0000-0000-0000-000000000404', '00000000-0000-0000-0000-000000000001', 'teller@bank-demo.co.id', 'teller01', 'Teller', 'One', 'active', true, false, NOW())
ON CONFLICT (id) DO NOTHING;

-- ──────────────────────────────────────────────
-- 6. User Credentials
-- ──────────────────────────────────────────────
INSERT INTO user_credentials (id, user_id, credential_type, credential_value, is_primary, created_at)
VALUES
    ('00000000-0000-0000-0000-000000000501', '00000000-0000-0000-0000-000000000401', 'password', '$argon2id$v=19$m=65536,t=3,p=4$c29tZXNhbHQ$RdescudvJCsgt3ub+b+daw', true, NOW()),
    ('00000000-0000-0000-0000-000000000502', '00000000-0000-0000-0000-000000000402', 'password', '$argon2id$v=19$m=65536,t=3,p=4$c29tZXNhbHQ$RdescudvJCsgt3ub+b+daw', true, NOW()),
    ('00000000-0000-0000-0000-000000000503', '00000000-0000-0000-0000-000000000403', 'password', '$argon2id$v=19$m=65536,t=3,p=4$c29tZXNhbHQ$RdescudvJCsgt3ub+b+daw', true, NOW()),
    ('00000000-0000-0000-0000-000000000504', '00000000-0000-0000-0000-000000000404', 'password', '$argon2id$v=19$m=65536,t=3,p=4$c29tZXNhbHQ$RdescudvJCsgt3ub+b+daw', true, NOW())
ON CONFLICT (id) DO NOTHING;

-- ──────────────────────────────────────────────
-- 7. Roles & Permissions
-- ──────────────────────────────────────────────
INSERT INTO roles (id, org_id, name, description, is_default)
VALUES
    ('00000000-0000-0000-0000-000000000601', '00000000-0000-0000-0000-000000000001', 'super_admin', 'Super Administrator', false),
    ('00000000-0000-0000-0000-000000000602', '00000000-0000-0000-0000-000000000001', 'branch_manager', 'Branch Manager', false),
    ('00000000-0000-0000-0000-000000000603', '00000000-0000-0000-0000-000000000001', 'supervisor', 'Supervisor', false),
    ('00000000-0000-0000-0000-000000000604', '00000000-0000-0000-0000-000000000001', 'teller', 'Teller', true)
ON CONFLICT (id) DO NOTHING;

INSERT INTO permissions (id, code, name, description, category)
VALUES
    ('00000000-0000-0000-0000-000000000701', 'user:read', 'Read Users', 'View user information', 'user'),
    ('00000000-0000-0000-0000-000000000702', 'user:write', 'Write Users', 'Create/update users', 'user'),
    ('00000000-0000-0000-0000-000000000703', 'user:delete', 'Delete Users', 'Delete users', 'user'),
    ('00000000-0000-0000-0000-000000000704', 'transaction:read', 'Read Transactions', 'View transactions', 'transaction'),
    ('00000000-0000-0000-0000-000000000705', 'transaction:write', 'Write Transactions', 'Create transactions', 'transaction'),
    ('00000000-0000-0000-0000-000000000706', 'transaction:approve', 'Approve Transactions', 'Approve transactions', 'transaction'),
    ('00000000-0000-0000-0000-000000000707', 'report:read', 'Read Reports', 'View reports', 'report'),
    ('00000000-0000-0000-0000-000000000708', 'admin:access', 'Admin Access', 'Access admin panel', 'admin')
ON CONFLICT (id) DO NOTHING;

-- Role-Permission assignments
INSERT INTO role_permissions (role_id, permission_id)
VALUES
    ('00000000-0000-0000-0000-000000000601', '00000000-0000-0000-0000-000000000701'),
    ('00000000-0000-0000-0000-000000000601', '00000000-0000-0000-0000-000000000702'),
    ('00000000-0000-0000-0000-000000000601', '00000000-0000-0000-0000-000000000703'),
    ('00000000-0000-0000-0000-000000000601', '00000000-0000-0000-0000-000000000704'),
    ('00000000-0000-0000-0000-000000000601', '00000000-0000-0000-0000-000000000705'),
    ('00000000-0000-0000-0000-000000000601', '00000000-0000-0000-0000-000000000706'),
    ('00000000-0000-0000-0000-000000000601', '00000000-0000-0000-0000-000000000707'),
    ('00000000-0000-0000-0000-000000000601', '00000000-0000-0000-0000-000000000708'),
    ('00000000-0000-0000-0000-000000000602', '00000000-0000-0000-0000-000000000701'),
    ('00000000-0000-0000-0000-000000000602', '00000000-0000-0000-0000-000000000704'),
    ('00000000-0000-0000-0000-000000000602', '00000000-0000-0000-0000-000000000705'),
    ('00000000-0000-0000-0000-000000000602', '00000000-0000-0000-0000-000000000706'),
    ('00000000-0000-0000-0000-000000000602', '00000000-0000-0000-0000-000000000707'),
    ('00000000-0000-0000-0000-000000000603', '00000000-0000-0000-0000-000000000701'),
    ('00000000-0000-0000-0000-000000000603', '00000000-0000-0000-0000-000000000704'),
    ('00000000-0000-0000-0000-000000000603', '00000000-0000-0000-0000-000000000705'),
    ('00000000-0000-0000-0000-000000000603', '00000000-0000-0000-0000-000000000706'),
    ('00000000-0000-0000-0000-000000000604', '00000000-0000-0000-0000-000000000701'),
    ('00000000-0000-0000-0000-000000000604', '00000000-0000-0000-0000-000000000704'),
    ('00000000-0000-0000-0000-000000000604', '00000000-0000-0000-0000-000000000705')
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- ──────────────────────────────────────────────
-- 8. User-Role Assignments
-- ──────────────────────────────────────────────
INSERT INTO user_roles (id, user_id, role_id, branch_id, org_id, granted_by, granted_at)
VALUES
    ('00000000-0000-0000-0000-000000000801', '00000000-0000-0000-0000-000000000401', '00000000-0000-0000-0000-000000000601', '00000000-0000-0000-0000-000000000201', '00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000401', NOW()),
    ('00000000-0000-0000-0000-000000000802', '00000000-0000-0000-0000-000000000402', '00000000-0000-0000-0000-000000000602', '00000000-0000-0000-0000-000000000201', '00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000401', NOW()),
    ('00000000-0000-0000-0000-000000000803', '00000000-0000-0000-0000-000000000403', '00000000-0000-0000-0000-000000000603', '00000000-0000-0000-0000-000000000201', '00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000401', NOW()),
    ('00000000-0000-0000-0000-000000000804', '00000000-0000-0000-0000-000000000404', '00000000-0000-0000-0000-000000000604', '00000000-0000-0000-0000-000000000201', '00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000401', NOW())
ON CONFLICT (id) DO NOTHING;

-- ──────────────────────────────────────────────
-- 9. OAuth2 Clients
-- ──────────────────────────────────────────────
INSERT INTO oauth2_clients (id, client_id, org_id, name, description, client_type, token_endpoint_auth_method, allowed_scopes, allowed_redirect_uris, is_active)
VALUES
    ('00000000-0000-0000-0000-000000000901', 'bos7-portal', '00000000-0000-0000-0000-000000000001', 'BOS7 Portal', 'Main banking portal', 'web', 'client_secret_basic', '{openid,profile,email,roles}', '{http://localhost:3000/callback,https://bos7.bank-demo.co.id/callback}', true),
    ('00000000-0000-0000-0000-000000000902', 'workflow7-web', '00000000-0000-0000-0000-000000000001', 'Workflow7 Web', 'Workflow management UI', 'web', 'client_secret_basic', '{openid,profile,email,roles}', '{http://localhost:3001/callback,https://workflow7.bank-demo.co.id/callback}', true),
    ('00000000-0000-0000-0000-000000000903', 'core7-api', '00000000-0000-0000-0000-000000000001', 'Core7 API', 'M2M service client', 'machine', 'client_secret_basic', '{openid,profile}', '{}', true),
    ('00000000-0000-0000-0000-000000000904', 'mobile-app', '00000000-0000-0000-0000-000000000001', 'Mobile Banking App', 'iOS/Android app', 'native', 'none', '{openid,profile,email}', '{bankdemo://callback}', true),
    ('00000000-0000-0000-0000-000000000905', 'workflow7-svc', '00000000-0000-0000-0000-000000000001', 'Workflow7 Service', 'M2M service client for workflow7', 'machine', 'client_secret_basic', '{openid,profile}', '{}', true),
    ('00000000-0000-0000-0000-000000000906', 'notif7-svc', '00000000-0000-0000-0000-000000000001', 'Notif7 Service', 'M2M service client for notif7', 'machine', 'client_secret_basic', '{openid,profile}', '{}', true)
ON CONFLICT (id) DO NOTHING;

-- ──────────────────────────────────────────────
-- 10. MFA Configs (for users with MFA enabled)
-- ──────────────────────────────────────────────
INSERT INTO mfa_configs (id, user_id, is_totp_enabled, is_email_otp_enabled, is_backup_codes_enabled, mfa_enabled_at)
VALUES
    ('00000000-0000-0000-0000-000000001001', '00000000-0000-0000-0000-000000000401', true, true, true, NOW()),
    ('00000000-0000-0000-0000-000000001003', '00000000-0000-0000-0000-000000000403', true, false, false, NOW())
ON CONFLICT (id) DO NOTHING;

-- ──────────────────────────────────────────────
-- Summary
-- ──────────────────────────────────────────────
-- Users:
--   - admin@bank-demo.co.id / Password123! (super_admin, all permissions)
--   - john@bank-demo.co.id / Password123! (branch_manager)
--   - jane@bank-demo.co.id / Password123! (supervisor, MFA enabled)
--   - teller@bank-demo.co.id / Password123! (teller)
--
-- OAuth2 Clients:
--   - bos7-portal (web, client_secret_basic)
--   - workflow7-web (web, client_secret_basic)
--   - core7-api (machine, client_secret_basic)
--   - mobile-app (native, public)
--
-- Branches:
--   - KC-BDG-001 (Kantor Cabang Bandung)
--   - KC-JKT-001 (Kantor Cabang Jakarta)
--   - KCP-DGO-001 (KCP Dago) - child of KC-BDG-001
--   - KAS-CIM-001 (Kantor Kas Cimahi) - child of KC-JKT-001
