-- Production organization — same canonical UUID as demo.
-- org settings disesuaikan untuk implementasi (MFA required, session lebih ketat).
INSERT INTO organizations (id, code, name, domain, status, settings)
VALUES (
    '00000000-0000-0000-0000-000000000001',
    'BANK',
    'Bank',
    'bank.internal',
    'active',
    '{"session_policy":{"max_sessions":1,"session_ttl_hours":8},"mfa_policy":{"required":true,"allow_totp":true,"allow_email_otp":true},"password_policy":{"min_length":10,"require_number":true,"require_special":true,"require_uppercase":true}}'::jsonb
) ON CONFLICT (id) DO UPDATE
    SET settings = EXCLUDED.settings;
