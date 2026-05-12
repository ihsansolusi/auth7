-- Add test admin users for Railway testing
-- Password: Password123! (argon2id, same as existing seed users)

INSERT INTO users (id, org_id, email, username, full_name, status, email_verified, mfa_enabled)
VALUES
    ('00000000-0000-0000-0000-000000000411', '00000000-0000-0000-0000-000000000001', 'pepsimanps4@bank.co.id', 'pepsimanps4', 'Pepsiman PS4', 'active', true, false),
    ('00000000-0000-0000-0000-000000000412', '00000000-0000-0000-0000-000000000001', 'tata.taufik@bank.co.id', 'tata.taufik', 'Tata Taufik', 'active', true, false)
ON CONFLICT (email) DO UPDATE SET
    status = 'active',
    email_verified = true,
    full_name = EXCLUDED.full_name;

INSERT INTO user_credentials (id, user_id, credential_type, secret_hash, version, is_current)
VALUES
    ('00000000-0000-0000-0000-000000000511', '00000000-0000-0000-0000-000000000411', 'password', '$argon2id$v=19$m=65536,t=3,p=4$c29tZXNhbHQ$2EgLsEMqNccY7XTG8Bxtl5Pumi4Zcs1KkJ2cspqHCiA', 1, true),
    ('00000000-0000-0000-0000-000000000512', '00000000-0000-0000-0000-000000000412', 'password', '$argon2id$v=19$m=65536,t=3,p=4$c29tZXNhbHQ$2EgLsEMqNccY7XTG8Bxtl5Pumi4Zcs1KkJ2cspqHCiA', 1, true)
ON CONFLICT (id) DO NOTHING;

INSERT INTO user_roles (id, user_id, role_id, branch_id, org_id, granted_by, granted_at)
VALUES
    ('00000000-0000-0000-0000-000000000811', '00000000-0000-0000-0000-000000000411', '00000000-0000-0000-0000-000000000601', '00000000-0000-0000-0000-000000000201', '00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000401', NOW()),
    ('00000000-0000-0000-0000-000000000812', '00000000-0000-0000-0000-000000000412', '00000000-0000-0000-0000-000000000601', '00000000-0000-0000-0000-000000000201', '00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000401', NOW())
ON CONFLICT (id) DO NOTHING;
