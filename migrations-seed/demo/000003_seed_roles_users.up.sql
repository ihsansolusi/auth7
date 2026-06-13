-- ─── Roles ────────────────────────────────────────────────────────────────────
INSERT INTO roles (id, org_id, code, name, description, is_default)
VALUES
    ('00000000-0000-0000-0002-000000000001', '00000000-0000-0000-0000-000000000001', 'SUPER_ADMIN',    'Super Admin',    'Akses penuh semua modul',        false),
    ('00000000-0000-0000-0002-000000000002', '00000000-0000-0000-0000-000000000001', 'BRANCH_MANAGER', 'Branch Manager', 'Manajer cabang',                 false),
    ('00000000-0000-0000-0002-000000000003', '00000000-0000-0000-0000-000000000001', 'SUPERVISOR',     'Supervisor',     'Supervisor operasional',         false),
    ('00000000-0000-0000-0002-000000000004', '00000000-0000-0000-0000-000000000001', 'TELLER',         'Teller',         'Teller transaksi',               true),
    ('00000000-0000-0000-0002-000000000005', '00000000-0000-0000-0000-000000000001', 'AUDITOR',        'Auditor',        'Audit trail access',             false)
ON CONFLICT (org_id, code) DO NOTHING;

-- ─── Users ────────────────────────────────────────────────────────────────────
-- Passwords:
--   bosdemo  → demo@123    ($argon2id$v=19$m=65536,t=3,p=4$ZxlitAZB7O8lb2CvXikUcg$Oh2fnb1wNAcNAPAbvJUsDhZA01xgjjwY4efttjAaE+Y)
--   others   → password123 ($argon2id$v=19$m=65536,t=3,p=4$c2VlZF9zYWx0X2ZpeGVkIQ$N+pMwLuOqjb62N8jRGpZTng1AJkGETP4yvjfe6CWPRI)
INSERT INTO users (id, org_id, username, email, full_name, status, email_verified,
                   mfa_enabled, mfa_method, mfa_reset_required, require_password_change,
                   failed_login_attempts, preferred_locale)
VALUES
    ('00000000-0000-0000-0001-000000000001', '00000000-0000-0000-0000-000000000001',
     'admin',   'admin@bankdemo.local',   'Demo Admin',          'active', true, false, '', false, false, 0, 'id'),
    ('00000000-0000-0000-0001-000000000002', '00000000-0000-0000-0000-000000000001',
     'bosdemo', 'bosdemo@bankdemo.local', 'Demo BOS',            'active', true, false, '', false, false, 0, 'id'),
    ('00000000-0000-0000-0001-000000000003', '00000000-0000-0000-0000-000000000001',
     'manager', 'manager@bankdemo.local', 'Demo Branch Manager', 'active', true, false, '', false, false, 0, 'id'),
    ('00000000-0000-0000-0001-000000000004', '00000000-0000-0000-0000-000000000001',
     'spv',     'spv@bankdemo.local',     'Demo Supervisor',     'active', true, false, '', false, false, 0, 'id'),
    ('00000000-0000-0000-0001-000000000005', '00000000-0000-0000-0000-000000000001',
     'teller',  'teller@bankdemo.local',  'Demo Teller',         'active', true, false, '', false, false, 0, 'id'),
    ('00000000-0000-0000-0001-000000000006', '00000000-0000-0000-0000-000000000001',
     'auditor', 'auditor@bankdemo.local', 'Demo Auditor',        'active', true, false, '', false, false, 0, 'id')
ON CONFLICT (username) DO NOTHING;

-- ─── Credentials (new schema: no version/is_current, UNIQUE per user+type) ───
INSERT INTO user_credentials (id, user_id, credential_type, secret_hash, created_at)
VALUES
    (gen_random_uuid(), '00000000-0000-0000-0001-000000000001', 'password',
     '$argon2id$v=19$m=65536,t=3,p=4$c2VlZF9zYWx0X2ZpeGVkIQ$N+pMwLuOqjb62N8jRGpZTng1AJkGETP4yvjfe6CWPRI',
     NOW()),
    (gen_random_uuid(), '00000000-0000-0000-0001-000000000002', 'password',
     '$argon2id$v=19$m=65536,t=3,p=4$ZxlitAZB7O8lb2CvXikUcg$Oh2fnb1wNAcNAPAbvJUsDhZA01xgjjwY4efttjAaE+Y',
     NOW()),
    (gen_random_uuid(), '00000000-0000-0000-0001-000000000003', 'password',
     '$argon2id$v=19$m=65536,t=3,p=4$c2VlZF9zYWx0X2ZpeGVkIQ$N+pMwLuOqjb62N8jRGpZTng1AJkGETP4yvjfe6CWPRI',
     NOW()),
    (gen_random_uuid(), '00000000-0000-0000-0001-000000000004', 'password',
     '$argon2id$v=19$m=65536,t=3,p=4$c2VlZF9zYWx0X2ZpeGVkIQ$N+pMwLuOqjb62N8jRGpZTng1AJkGETP4yvjfe6CWPRI',
     NOW()),
    (gen_random_uuid(), '00000000-0000-0000-0001-000000000005', 'password',
     '$argon2id$v=19$m=65536,t=3,p=4$c2VlZF9zYWx0X2ZpeGVkIQ$N+pMwLuOqjb62N8jRGpZTng1AJkGETP4yvjfe6CWPRI',
     NOW()),
    (gen_random_uuid(), '00000000-0000-0000-0001-000000000006', 'password',
     '$argon2id$v=19$m=65536,t=3,p=4$c2VlZF9zYWx0X2ZpeGVkIQ$N+pMwLuOqjb62N8jRGpZTng1AJkGETP4yvjfe6CWPRI',
     NOW())
ON CONFLICT (user_id, credential_type) DO NOTHING;

-- ─── Role assignments ─────────────────────────────────────────────────────────
INSERT INTO user_roles (id, user_id, role_id, org_id, granted_by, granted_at)
VALUES
    (gen_random_uuid(), '00000000-0000-0000-0001-000000000001', '00000000-0000-0000-0002-000000000001', '00000000-0000-0000-0000-000000000001', 'system', NOW()),
    (gen_random_uuid(), '00000000-0000-0000-0001-000000000002', '00000000-0000-0000-0002-000000000001', '00000000-0000-0000-0000-000000000001', 'system', NOW()),
    (gen_random_uuid(), '00000000-0000-0000-0001-000000000003', '00000000-0000-0000-0002-000000000002', '00000000-0000-0000-0000-000000000001', 'system', NOW()),
    (gen_random_uuid(), '00000000-0000-0000-0001-000000000004', '00000000-0000-0000-0002-000000000003', '00000000-0000-0000-0000-000000000001', 'system', NOW()),
    (gen_random_uuid(), '00000000-0000-0000-0001-000000000005', '00000000-0000-0000-0002-000000000004', '00000000-0000-0000-0000-000000000001', 'system', NOW()),
    (gen_random_uuid(), '00000000-0000-0000-0001-000000000006', '00000000-0000-0000-0002-000000000005', '00000000-0000-0000-0000-000000000001', 'system', NOW())
ON CONFLICT DO NOTHING;
