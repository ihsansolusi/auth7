-- Demo users for Bank Demo org.
-- Password for all users: password123
-- Hash: argon2id v=19, m=65536, t=3, p=4, deterministic salt "seed_salt_fixed!"
-- $argon2id$v=19$m=65536,t=3,p=4$c2VlZF9zYWx0X2ZpeGVkIQ$N+pMwLuOqjb62N8jRGpZTng1AJkGETP4yvjfe6CWPRI

INSERT INTO users (id, org_id, username, email, full_name, status, email_verified,
                   mfa_enabled, mfa_method, mfa_reset_required, require_password_change,
                   failed_login_attempts, preferred_locale, created_at, updated_at, created_by, updated_by)
VALUES
    ('00000000-0000-0000-0001-000000000001', '00000000-0000-0000-0000-000000000001',
     'admin',   'admin@bankdemo.local',   'Demo Admin',         'active', true, false, '', false, false, 0, 'id', NOW(), NOW(), 'system', 'system'),
    ('00000000-0000-0000-0001-000000000002', '00000000-0000-0000-0000-000000000001',
     'bosdemo', 'bosdemo@bankdemo.local', 'Demo BOS',           'active', true, false, '', false, false, 0, 'id', NOW(), NOW(), 'system', 'system'),
    ('00000000-0000-0000-0001-000000000003', '00000000-0000-0000-0000-000000000001',
     'manager', 'manager@bankdemo.local', 'Demo Branch Manager','active', true, false, '', false, false, 0, 'id', NOW(), NOW(), 'system', 'system'),
    ('00000000-0000-0000-0001-000000000004', '00000000-0000-0000-0000-000000000001',
     'spv',     'spv@bankdemo.local',     'Demo Supervisor',    'active', true, false, '', false, false, 0, 'id', NOW(), NOW(), 'system', 'system'),
    ('00000000-0000-0000-0001-000000000005', '00000000-0000-0000-0000-000000000001',
     'teller',  'teller@bankdemo.local',  'Demo Teller',        'active', true, false, '', false, false, 0, 'id', NOW(), NOW(), 'system', 'system'),
    ('00000000-0000-0000-0001-000000000006', '00000000-0000-0000-0000-000000000001',
     'auditor', 'auditor@bankdemo.local', 'Demo Auditor',       'active', true, false, '', false, false, 0, 'id', NOW(), NOW(), 'system', 'system')
ON CONFLICT (username) WHERE deleted_at IS NULL DO NOTHING;

INSERT INTO user_credentials (id, user_id, credential_type, secret_hash, version, is_current, created_at)
SELECT
    gen_random_uuid(),
    u.id,
    'password',
    '$argon2id$v=19$m=65536,t=3,p=4$c2VlZF9zYWx0X2ZpeGVkIQ$N+pMwLuOqjb62N8jRGpZTng1AJkGETP4yvjfe6CWPRI',
    1,
    true,
    NOW()
FROM users u
WHERE u.id IN (
    '00000000-0000-0000-0001-000000000001',
    '00000000-0000-0000-0001-000000000002',
    '00000000-0000-0000-0001-000000000003',
    '00000000-0000-0000-0001-000000000004',
    '00000000-0000-0000-0001-000000000005',
    '00000000-0000-0000-0001-000000000006'
)
ON CONFLICT DO NOTHING;

-- Assign roles by matching username → role code
INSERT INTO user_roles (id, user_id, role_id, org_id, granted_by, granted_at)
SELECT gen_random_uuid(), u.id, r.id, u.org_id, 'system', NOW()
FROM users u
JOIN roles r ON r.org_id = u.org_id
WHERE (u.username = 'admin'   AND r.code = 'SUPER_ADMIN')
   OR (u.username = 'bosdemo' AND r.code = 'SUPER_ADMIN')
   OR (u.username = 'manager' AND r.code = 'BRANCH_MANAGER')
   OR (u.username = 'spv'     AND r.code = 'SUPERVISOR')
   OR (u.username = 'teller'  AND r.code = 'TELLER')
   OR (u.username = 'auditor' AND r.code = 'AUDITOR')
ON CONFLICT DO NOTHING;
