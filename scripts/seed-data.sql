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
INSERT INTO oauth2_clients (id, client_id, org_id, name, description, client_type, token_endpoint_auth_method, allowed_scopes, allowed_redirect_uris, is_active, app_url, icon_name, icon_color, client_secret_hash)
VALUES
    -- Web clients (user-facing) — app_url = canonical domain URL (bos7.local for dev, updated per env)
    ('00000000-0000-0000-0000-000000000901', 'bos7-portal',         '00000000-0000-0000-0000-000000000001', 'BOS7 Portal',         'Main banking portal launcher',            'web', 'client_secret_basic', '{openid,profile,email,roles,offline_access}', '{http://localhost:3006/api/auth/callback,https://portal.bos7.local/api/auth/callback,https://portal.dev.ihsansolusi.co.id/api/auth/callback,https://bos7-portal.up.railway.app/api/auth/callback}', true, 'https://bos7-portal.up.railway.app',         'Launch',             '#0f62fe', '8810845ac7f98ef512240d68da90af9315b6afa09f1d1710f2a2b95b2b6f4525'),
    ('00000000-0000-0000-0000-000000000902', 'workflow7-web',        '00000000-0000-0000-0000-000000000001', 'Workflow7 Web',        'Workflow & BPM management UI',            'web', 'client_secret_basic', '{openid,profile,email,roles}', '{http://localhost:3002/api/auth/callback,https://workflow.bos7.local/api/auth/callback,https://workflow.dev.ihsansolusi.co.id/api/auth/callback}', true, 'https://workflow.bos7.local',       'FlowStream',         '#8a3ffc', 'dc63d2325d239b92d0f169c2631ae637f8c9b08fcab8d4d276b6acd404a617ee'),
    ('00000000-0000-0000-0000-000000000907', 'auth7-ui-dev',         '00000000-0000-0000-0000-000000000001', 'Auth7 UI Dev',         'Auth7 dashboard (admin panel)',            'web', 'client_secret_basic', '{openid,profile,email,roles}', '{http://localhost:3001/api/auth/callback,https://auth.bos7.local/api/auth/callback,https://auth.dev.ihsansolusi.co.id/api/auth/callback,https://account.up.railway.app/api/auth/callback}', true, NULL, NULL, NULL, '22d413a226a293dc8704aa2de30c0156dc515cf0649d225a547a781ed29f2d8a'),
    ('00000000-0000-0000-0000-000000000908', 'bos7-template',        '00000000-0000-0000-0000-000000000001', 'BOS7 Template',        'Next.js app template / dev scaffold',     'web', 'client_secret_basic', '{openid,profile,email,roles}', '{http://localhost:3004/api/auth/callback,https://template.bos7.local/api/auth/callback,https://template.dev.ihsansolusi.co.id/api/auth/callback}', true, NULL, NULL, NULL, 'b6091476f6d158417e9c2fd5727c221e10a570a173d252c6b75b8f6c3c553eb4'),
    ('00000000-0000-0000-0000-000000000909', 'bos7-enterprise',      '00000000-0000-0000-0000-000000000001', 'BOS7 Enterprise',      'Enterprise management module',            'web', 'client_secret_basic', '{openid,profile,email,roles}', '{http://localhost:3003/api/auth/callback,https://enterprise.bos7.local/api/auth/callback,https://enterprise.dev.ihsansolusi.co.id/api/auth/callback,https://bos7-enterprise.up.railway.app/api/auth/callback}', true, 'https://bos7-enterprise.up.railway.app',     'Enterprise',         '#0f62fe', 'b29557b642e1c7dc5b172e48079842ec2663fc4fa8dcdbb1c4c47acb0b5e9b69'),
    ('00000000-0000-0000-0000-000000000910', 'bos7-financing',       '00000000-0000-0000-0000-000000000001', 'BOS7 Financing',       'Financing (pembiayaan) module',           'web', 'client_secret_basic', '{openid,profile,email,roles}', '{http://localhost:3010/api/auth/callback,https://financing.bos7.local/api/auth/callback,https://financing.dev.ihsansolusi.co.id/api/auth/callback,https://bos7-financing.up.railway.app/api/auth/callback}', true, 'https://bos7-financing.up.railway.app',      'Finance',            '#198038', '6ec50d1810fa34e0860ae969e49027fd7dd71b8619a2b6904efec3287e802c42'),
    ('00000000-0000-0000-0000-000000000911', 'bos7-funding',         '00000000-0000-0000-0000-000000000001', 'BOS7 Funding',         'Funding (pendanaan) module',              'web', 'client_secret_basic', '{openid,profile,email,roles}', '{http://localhost:3011/api/auth/callback,https://funding.bos7.local/api/auth/callback,https://funding.dev.ihsansolusi.co.id/api/auth/callback,https://bos7-funding.up.railway.app/api/auth/callback}', true, 'https://bos7-funding.up.railway.app',        'Money',              '#198038', '9b32cbf62ec99dfd155bd1fc16f0778118954793e792982c33bfa3d8210a0eca'),
    ('00000000-0000-0000-0000-000000000912', 'bos7-treasury',        '00000000-0000-0000-0000-000000000001', 'BOS7 Treasury',        'Treasury management module',              'web', 'client_secret_basic', '{openid,profile,email,roles}', '{http://localhost:3012/api/auth/callback,https://treasury.bos7.local/api/auth/callback,https://treasury.dev.ihsansolusi.co.id/api/auth/callback,https://bos7-treasury.up.railway.app/api/auth/callback}', true, 'https://bos7-treasury.up.railway.app',       'Currency',           '#6929c4', 'f70fc402e06c014c155e2a0f1d3e5e0c926fa0f774c7d4595053e7b505770475'),
    ('00000000-0000-0000-0000-000000000913', 'bos7-smt',             '00000000-0000-0000-0000-000000000001', 'BOS7 SMT',             'SMT module',                              'web', 'client_secret_basic', '{openid,profile,email,roles}', '{http://localhost:3013/api/auth/callback,https://smt.bos7.local/api/auth/callback,https://smt.dev.ihsansolusi.co.id/api/auth/callback,https://bos7-smt.up.railway.app/api/auth/callback}', true, 'https://bos7-smt.up.railway.app',            'Migrate',            '#005d5d', 'a5f26ffd1bd9142cc10b7ced05f71b419f30ad3fd0d9b88ad3465c3565f40d80'),
    ('00000000-0000-0000-0000-000000000914', 'bos7-accounting',      '00000000-0000-0000-0000-000000000001', 'BOS7 Accounting',      'Accounting (akuntansi) module',           'web', 'client_secret_basic', '{openid,profile,email,roles}', '{http://localhost:3014/api/auth/callback,https://accounting.bos7.local/api/auth/callback,https://accounting.dev.ihsansolusi.co.id/api/auth/callback,https://bos7-accounting.up.railway.app/api/auth/callback}', true, 'https://bos7-accounting.up.railway.app',     'ChartLineData',      '#0043ce', '452bf6828892d97de48b52fc23cb77ed8b45856cdea2b0800decaeb20a2cd3a0'),
    ('00000000-0000-0000-0000-000000000915', 'bos7-cif',             '00000000-0000-0000-0000-000000000001', 'BOS7 CIF',             'Customer Information File module',        'web', 'client_secret_basic', '{openid,profile,email,roles}', '{http://localhost:3015/api/auth/callback,https://cif.bos7.local/api/auth/callback,https://cif.dev.ihsansolusi.co.id/api/auth/callback,https://bos7-cif.up.railway.app/api/auth/callback}', true, 'https://bos7-cif.up.railway.app',            'UserIdentification', '#0f62fe', '3efee451e84c6c0ffa08f69401606b198ab77fffac8e87d34f6ad625a8fe5f9a'),
    ('00000000-0000-0000-0000-000000000916', 'bos7-internalaccount', '00000000-0000-0000-0000-000000000001', 'BOS7 InternalAccount', 'Internal account management module',     'web', 'client_secret_basic', '{openid,profile,email,roles}', '{http://localhost:3016/api/auth/callback,https://internalaccount.bos7.local/api/auth/callback,https://internalaccount.dev.ihsansolusi.co.id/api/auth/callback,https://bos7-internalaccount.up.railway.app/api/auth/callback}', true, 'https://bos7-internalaccount.up.railway.app','Account',            '#9f1853', '7fe52794124cd18ec6edf01f9efeb06bffcf12b5f82652035da64ae16f511a52'),
    ('00000000-0000-0000-0000-000000000917', 'bos7-remittance',      '00000000-0000-0000-0000-000000000001', 'BOS7 Remittance',      'Remittance module',                       'web', 'client_secret_basic', '{openid,profile,email,roles}', '{http://localhost:3017/api/auth/callback,https://remittance.bos7.local/api/auth/callback,https://remittance.dev.ihsansolusi.co.id/api/auth/callback,https://bos7-remittance.up.railway.app/api/auth/callback}', true, 'https://bos7-remittance.up.railway.app',     'Send',               '#b28600', 'ce01e0bf98e5f982fd90a68171612594df71862b13744b290e1aee9485a7fa57'),
    ('00000000-0000-0000-0000-000000000918', 'bos7-batchprocessing', '00000000-0000-0000-0000-000000000001', 'BOS7 BatchProcessing', 'Batch processing module',                 'web', 'client_secret_basic', '{openid,profile,email,roles}', '{http://localhost:3018/api/auth/callback,https://batchprocessing.bos7.local/api/auth/callback,https://batchprocessing.dev.ihsansolusi.co.id/api/auth/callback,https://bos7-batchprocessing.up.railway.app/api/auth/callback}', true, 'https://bos7-batchprocessing.up.railway.app','BatchJob',           '#4d5358', '6c367d186687cf37c5ded38dcc83655a98b600084260c5464e487de4a12c4d53'),
    -- Mobile client
    ('00000000-0000-0000-0000-000000000904', 'mobile-app',           '00000000-0000-0000-0000-000000000001', 'Mobile Banking App',   'iOS/Android app',                         'native',  'none',                '{openid,profile,email}',        '{bankdemo://callback}', true, NULL, NULL, NULL, NULL),
    -- Machine clients (M2M, no redirect URIs)
    ('00000000-0000-0000-0000-000000000903', 'core7-api',            '00000000-0000-0000-0000-000000000001', 'Core7 API',            'M2M service client (generic)',            'machine', 'client_secret_basic', '{openid,profile}',              '{}', true, NULL, NULL, NULL, 'a4f32975344b8dc320d7e64ea5f3f51c8ab18ed23d0b94cba5cbd551945e8e73'),
    ('00000000-0000-0000-0000-000000000905', 'workflow7-svc',        '00000000-0000-0000-0000-000000000001', 'Workflow7 Service',    'M2M service client for workflow7',        'machine', 'client_secret_basic', '{openid,profile}',              '{}', true, NULL, NULL, NULL, '6577925abc0d51b6559284157367fde87b09e87d58da54fdc5551b98fedb7f5e'),
    ('00000000-0000-0000-0000-000000000906', 'notif7-svc',           '00000000-0000-0000-0000-000000000001', 'Notif7 Service',       'M2M service client for notif7',           'machine', 'client_secret_basic', '{openid,profile}',              '{}', true, NULL, NULL, NULL, '6ddecefc22baff5b752f8a689a679002e04322c48112a4ea3d54a4ab86ab971d')
ON CONFLICT (id) DO UPDATE SET
    description          = EXCLUDED.description,
    allowed_redirect_uris = EXCLUDED.allowed_redirect_uris,
    app_url              = EXCLUDED.app_url,
    icon_name            = EXCLUDED.icon_name,
    icon_color           = EXCLUDED.icon_color,
    client_secret_hash   = EXCLUDED.client_secret_hash;

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
