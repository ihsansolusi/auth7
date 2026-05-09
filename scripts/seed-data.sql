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
INSERT INTO users (id, org_id, email, username, full_name, status, email_verified, mfa_enabled)
VALUES
    ('00000000-0000-0000-0000-000000000401', '00000000-0000-0000-0000-000000000001', 'admin@bank-demo.co.id', 'admin', 'Super Admin', 'active', true, true),
    ('00000000-0000-0000-0000-000000000402', '00000000-0000-0000-0000-000000000001', 'john@bank-demo.co.id', 'john.doe', 'John Doe', 'active', true, false),
    ('00000000-0000-0000-0000-000000000403', '00000000-0000-0000-0000-000000000001', 'jane@bank-demo.co.id', 'jane.smith', 'Jane Smith', 'active', true, true),
    ('00000000-0000-0000-0000-000000000404', '00000000-0000-0000-0000-000000000001', 'teller@bank-demo.co.id', 'teller01', 'Teller One', 'active', true, false)
ON CONFLICT (id) DO UPDATE SET status = EXCLUDED.status, mfa_enabled = EXCLUDED.mfa_enabled, full_name = EXCLUDED.full_name;

-- ──────────────────────────────────────────────
-- 6. User Credentials
-- ──────────────────────────────────────────────
-- Hash: argon2id m=65536,t=3,p=4,keyLen=32, salt="somesalt", password="Password123!"
-- Generated with Go: argon2.IDKey([]byte("Password123!"), []byte("somesalt"), 3, 65536, 4, 32)
INSERT INTO user_credentials (id, user_id, credential_type, secret_hash, version, is_current)
VALUES
    ('00000000-0000-0000-0000-000000000501', '00000000-0000-0000-0000-000000000401', 'password', '$argon2id$v=19$m=65536,t=3,p=4$c29tZXNhbHQ$2EgLsEMqNccY7XTG8Bxtl5Pumi4Zcs1KkJ2cspqHCiA', 1, true),
    ('00000000-0000-0000-0000-000000000502', '00000000-0000-0000-0000-000000000402', 'password', '$argon2id$v=19$m=65536,t=3,p=4$c29tZXNhbHQ$2EgLsEMqNccY7XTG8Bxtl5Pumi4Zcs1KkJ2cspqHCiA', 1, true),
    ('00000000-0000-0000-0000-000000000503', '00000000-0000-0000-0000-000000000403', 'password', '$argon2id$v=19$m=65536,t=3,p=4$c29tZXNhbHQ$2EgLsEMqNccY7XTG8Bxtl5Pumi4Zcs1KkJ2cspqHCiA', 1, true),
    ('00000000-0000-0000-0000-000000000504', '00000000-0000-0000-0000-000000000404', 'password', '$argon2id$v=19$m=65536,t=3,p=4$c29tZXNhbHQ$2EgLsEMqNccY7XTG8Bxtl5Pumi4Zcs1KkJ2cspqHCiA', 1, true)
ON CONFLICT (id) DO UPDATE SET secret_hash = EXCLUDED.secret_hash, is_current = EXCLUDED.is_current;

-- ──────────────────────────────────────────────
-- 7. Roles & Permissions
-- ──────────────────────────────────────────────
INSERT INTO roles (id, org_id, code, name, description, is_default)
VALUES
    ('00000000-0000-0000-0000-000000000601', '00000000-0000-0000-0000-000000000001', 'super_admin', 'Super Administrator', 'Full system access', false),
    ('00000000-0000-0000-0000-000000000602', '00000000-0000-0000-0000-000000000001', 'branch_manager', 'Branch Manager', 'Branch management access', false),
    ('00000000-0000-0000-0000-000000000603', '00000000-0000-0000-0000-000000000001', 'supervisor', 'Supervisor', 'Supervisory access', false),
    ('00000000-0000-0000-0000-000000000604', '00000000-0000-0000-0000-000000000001', 'teller', 'Teller', 'Teller operations access', true)
ON CONFLICT (id) DO UPDATE SET code = EXCLUDED.code, name = EXCLUDED.name;

INSERT INTO permissions (id, code, name, description, resource_type)
VALUES
    ('00000000-0000-0000-0000-000000000701', 'user:read', 'Read Users', 'View user information', 'user'),
    ('00000000-0000-0000-0000-000000000702', 'user:write', 'Write Users', 'Create/update users', 'user'),
    ('00000000-0000-0000-0000-000000000703', 'user:delete', 'Delete Users', 'Delete users', 'user'),
    ('00000000-0000-0000-0000-000000000704', 'transaction:read', 'Read Transactions', 'View transactions', 'transaction'),
    ('00000000-0000-0000-0000-000000000705', 'transaction:write', 'Write Transactions', 'Create transactions', 'transaction'),
    ('00000000-0000-0000-0000-000000000706', 'transaction:approve', 'Approve Transactions', 'Approve transactions', 'transaction'),
    ('00000000-0000-0000-0000-000000000707', 'report:read', 'Read Reports', 'View reports', 'report'),
    ('00000000-0000-0000-0000-000000000708', 'admin:access', 'Admin Access', 'Access admin panel', 'admin')
ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, resource_type = EXCLUDED.resource_type;

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
-- redirect_uris per web client includes 3 environments:
--   1. http://localhost:PORT  (dev direct)
--   2. https://<app>.bos7.local  (dev nginx)
--   3. https://<app>.dev.ihsansolusi.co.id  (Railway/staging)
-- ──────────────────────────────────────────────
INSERT INTO oauth2_clients (id, client_id, org_id, name, description, client_type, token_endpoint_auth_method, allowed_scopes, allowed_redirect_uris, is_active, app_url, icon_name, icon_color)
VALUES
    -- Web clients (user-facing) — app_url = canonical domain URL (bos7.local for dev, updated per env)
    ('00000000-0000-0000-0000-000000000901', 'bos7-portal',         '00000000-0000-0000-0000-000000000001', 'BOS7 Portal',         'Main banking portal launcher',            'web', 'client_secret_basic', '{openid,profile,email,roles,offline_access}', '{http://localhost:3006/api/auth/callback,https://portal.bos7.local/api/auth/callback,https://portal.dev.ihsansolusi.co.id/api/auth/callback,https://bos7-portal-development.up.railway.app/api/auth/callback}', true, 'https://portal.bos7.local',         'Launch',             '#0f62fe'),
    ('00000000-0000-0000-0000-000000000902', 'workflow7-web',        '00000000-0000-0000-0000-000000000001', 'Workflow7 Web',        'Workflow & BPM management UI',            'web', 'client_secret_basic', '{openid,profile,email,roles}', '{http://localhost:3002/api/auth/callback,https://workflow.bos7.local/api/auth/callback,https://workflow.dev.ihsansolusi.co.id/api/auth/callback}', true, 'https://workflow.bos7.local',       'FlowStream',         '#8a3ffc'),
    ('00000000-0000-0000-0000-000000000907', 'auth7-ui-dev',         '00000000-0000-0000-0000-000000000001', 'Auth7 UI Dev',         'Auth7 dashboard (admin panel)',            'web', 'client_secret_basic', '{openid,profile,email,roles}', '{http://localhost:3001/api/auth/callback,https://auth.bos7.local/api/auth/callback,https://auth.dev.ihsansolusi.co.id/api/auth/callback,https://auth7-ui-development.up.railway.app/api/auth/callback}', true, NULL, NULL, NULL),
    ('00000000-0000-0000-0000-000000000908', 'bos7-template',        '00000000-0000-0000-0000-000000000001', 'BOS7 Template',        'Next.js app template / dev scaffold',     'web', 'client_secret_basic', '{openid,profile,email,roles}', '{http://localhost:3004/api/auth/callback,https://template.bos7.local/api/auth/callback,https://template.dev.ihsansolusi.co.id/api/auth/callback}', true, NULL, NULL, NULL),
    ('00000000-0000-0000-0000-000000000909', 'bos7-enterprise',      '00000000-0000-0000-0000-000000000001', 'BOS7 Enterprise',      'Enterprise management module',            'web', 'client_secret_basic', '{openid,profile,email,roles}', '{http://localhost:3003/api/auth/callback,https://enterprise.bos7.local/api/auth/callback,https://enterprise.dev.ihsansolusi.co.id/api/auth/callback}', true, 'https://enterprise.bos7.local',     'Enterprise',         '#0f62fe'),
    ('00000000-0000-0000-0000-000000000910', 'bos7-financing',       '00000000-0000-0000-0000-000000000001', 'BOS7 Financing',       'Financing (pembiayaan) module',           'web', 'client_secret_basic', '{openid,profile,email,roles}', '{http://localhost:3010/api/auth/callback,https://financing.bos7.local/api/auth/callback,https://financing.dev.ihsansolusi.co.id/api/auth/callback}', true, 'https://financing.bos7.local',      'Finance',            '#198038'),
    ('00000000-0000-0000-0000-000000000911', 'bos7-funding',         '00000000-0000-0000-0000-000000000001', 'BOS7 Funding',         'Funding (pendanaan) module',              'web', 'client_secret_basic', '{openid,profile,email,roles}', '{http://localhost:3011/api/auth/callback,https://funding.bos7.local/api/auth/callback,https://funding.dev.ihsansolusi.co.id/api/auth/callback}', true, 'https://funding.bos7.local',        'Money',              '#198038'),
    ('00000000-0000-0000-0000-000000000912', 'bos7-treasury',        '00000000-0000-0000-0000-000000000001', 'BOS7 Treasury',        'Treasury management module',              'web', 'client_secret_basic', '{openid,profile,email,roles}', '{http://localhost:3012/api/auth/callback,https://treasury.bos7.local/api/auth/callback,https://treasury.dev.ihsansolusi.co.id/api/auth/callback}', true, 'https://treasury.bos7.local',       'Currency',           '#6929c4'),
    ('00000000-0000-0000-0000-000000000913', 'bos7-smt',             '00000000-0000-0000-0000-000000000001', 'BOS7 SMT',             'SMT module',                              'web', 'client_secret_basic', '{openid,profile,email,roles}', '{http://localhost:3013/api/auth/callback,https://smt.bos7.local/api/auth/callback,https://smt.dev.ihsansolusi.co.id/api/auth/callback}', true, 'https://smt.bos7.local',            'Migrate',            '#005d5d'),
    ('00000000-0000-0000-0000-000000000914', 'bos7-accounting',      '00000000-0000-0000-0000-000000000001', 'BOS7 Accounting',      'Accounting (akuntansi) module',           'web', 'client_secret_basic', '{openid,profile,email,roles}', '{http://localhost:3014/api/auth/callback,https://accounting.bos7.local/api/auth/callback,https://accounting.dev.ihsansolusi.co.id/api/auth/callback}', true, 'https://accounting.bos7.local',     'ChartLineData',      '#0043ce'),
    ('00000000-0000-0000-0000-000000000915', 'bos7-cif',             '00000000-0000-0000-0000-000000000001', 'BOS7 CIF',             'Customer Information File module',        'web', 'client_secret_basic', '{openid,profile,email,roles}', '{http://localhost:3015/api/auth/callback,https://cif.bos7.local/api/auth/callback,https://cif.dev.ihsansolusi.co.id/api/auth/callback}', true, 'https://cif.bos7.local',            'UserIdentification', '#0f62fe'),
    ('00000000-0000-0000-0000-000000000916', 'bos7-internalaccount', '00000000-0000-0000-0000-000000000001', 'BOS7 InternalAccount', 'Internal account management module',     'web', 'client_secret_basic', '{openid,profile,email,roles}', '{http://localhost:3016/api/auth/callback,https://internalaccount.bos7.local/api/auth/callback,https://internalaccount.dev.ihsansolusi.co.id/api/auth/callback}', true, 'https://internalaccount.bos7.local','Account',            '#9f1853'),
    ('00000000-0000-0000-0000-000000000917', 'bos7-remittance',      '00000000-0000-0000-0000-000000000001', 'BOS7 Remittance',      'Remittance module',                       'web', 'client_secret_basic', '{openid,profile,email,roles}', '{http://localhost:3017/api/auth/callback,https://remittance.bos7.local/api/auth/callback,https://remittance.dev.ihsansolusi.co.id/api/auth/callback}', true, 'https://remittance.bos7.local',     'Send',               '#b28600'),
    ('00000000-0000-0000-0000-000000000918', 'bos7-batchprocessing', '00000000-0000-0000-0000-000000000001', 'BOS7 BatchProcessing', 'Batch processing module',                 'web', 'client_secret_basic', '{openid,profile,email,roles}', '{http://localhost:3018/api/auth/callback,https://batchprocessing.bos7.local/api/auth/callback,https://batchprocessing.dev.ihsansolusi.co.id/api/auth/callback}', true, 'https://batchprocessing.bos7.local','BatchJob',           '#4d5358'),
    -- Mobile client
    ('00000000-0000-0000-0000-000000000904', 'mobile-app',           '00000000-0000-0000-0000-000000000001', 'Mobile Banking App',   'iOS/Android app',                         'native',  'none',                '{openid,profile,email}',        '{bankdemo://callback}', true, NULL, NULL, NULL),
    -- Machine clients (M2M, no redirect URIs)
    ('00000000-0000-0000-0000-000000000903', 'core7-api',            '00000000-0000-0000-0000-000000000001', 'Core7 API',            'M2M service client (generic)',            'machine', 'client_secret_basic', '{openid,profile}',              '{}', true, NULL, NULL, NULL),
    ('00000000-0000-0000-0000-000000000905', 'workflow7-svc',        '00000000-0000-0000-0000-000000000001', 'Workflow7 Service',    'M2M service client for workflow7',        'machine', 'client_secret_basic', '{openid,profile}',              '{}', true, NULL, NULL, NULL),
    ('00000000-0000-0000-0000-000000000906', 'notif7-svc',           '00000000-0000-0000-0000-000000000001', 'Notif7 Service',       'M2M service client for notif7',           'machine', 'client_secret_basic', '{openid,profile}',              '{}', true, NULL, NULL, NULL)
ON CONFLICT (id) DO UPDATE SET
    description          = EXCLUDED.description,
    allowed_redirect_uris = EXCLUDED.allowed_redirect_uris,
    app_url              = EXCLUDED.app_url,
    icon_name            = EXCLUDED.icon_name,
    icon_color           = EXCLUDED.icon_color;

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
-- OAuth2 Clients (web — 3 redirect_uris per app: localhost, bos7.local, dev.ihsansolusi.co.id):
--   - bos7-portal, workflow7-web, auth7-ui-dev, bos7-template, bos7-enterprise
--   - bos7-financing, bos7-funding, bos7-treasury, bos7-smt, bos7-accounting
--   - bos7-cif, bos7-internalaccount, bos7-remittance, bos7-batchprocessing
-- OAuth2 Clients (machine / native):
--   - core7-api, workflow7-svc, notif7-svc (M2M)
--   - mobile-app (native, public)
--
-- Branches:
--   - KC-BDG-001 (Kantor Cabang Bandung)
--   - KC-JKT-001 (Kantor Cabang Jakarta)
--   - KCP-DGO-001 (KCP Dago) - child of KC-BDG-001
--   - KAS-CIM-001 (Kantor Kas Cimahi) - child of KC-JKT-001
