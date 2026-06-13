-- Demo organization — UUID shared across all Core7 services.
INSERT INTO organizations (id, code, name, domain, status, settings)
VALUES (
    '00000000-0000-0000-0000-000000000001',
    'DEMO',
    'Bank Demo',
    'bankdemo.local',
    'active',
    '{"session_policy":{"max_sessions":3,"session_ttl_hours":8},"mfa_policy":{"required":false,"allow_totp":true,"allow_email_otp":true},"password_policy":{"min_length":8,"require_number":true,"require_special":false,"require_uppercase":true}}'::jsonb
) ON CONFLICT (id) DO NOTHING;
