-- Demo branches + per-user primary branch assignments + workflow7 M2M OAuth2
-- client. Completes the demo fixture so fresh-clone smoke tests work out of
-- the box without manual SQL.
--
-- Branch UUIDs follow the canonical enterprise convention
--   d0000000-0000-0000-0000-0000000000XX (XX = decimal branch_code)
-- so this auth7 projection is byte-identical to core7_enterprise.branches.id
-- for the same codes. Lets `user_branch_assignments` reference the same UUID
-- the enterprise service uses in its own tables (transaction journals,
-- account ownership, etc.) without translation.
--
-- For full sync (all 70+ branches), use scripts/sync-branches-from-enterprise.sh.
-- This migration only seeds the two used by demo users.

-- ─── 1. Branches (minimal projection: id, org_id, branch_code, is_active) ────
-- branch_name added by migration 28 — populated via UPDATE there to avoid
-- needing schema-conditional logic in this older migration.
INSERT INTO branches (id, org_id, branch_code, is_active) VALUES
    ('d0000000-0000-0000-0000-000000000000',
     '00000000-0000-0000-0000-000000000001',
     '000', true),
    ('d0000000-0000-0000-0000-000000000001',
     '00000000-0000-0000-0000-000000000001',
     '001', true)
ON CONFLICT (id) DO UPDATE
    SET branch_code = EXCLUDED.branch_code,
        is_active   = EXCLUDED.is_active;

-- ─── 2. Primary branch assignments for the 6 demo users ──────────────────────
-- admin / bosdemo / manager / spv / auditor → HQ (000)
-- teller → KANTOR CABANG BANDUNG (001)
INSERT INTO user_branch_assignments (user_id, branch_id, org_id, is_primary)
VALUES
    ('00000000-0000-0000-0001-000000000001', 'd0000000-0000-0000-0000-000000000000', '00000000-0000-0000-0000-000000000001', true),
    ('00000000-0000-0000-0001-000000000002', 'd0000000-0000-0000-0000-000000000000', '00000000-0000-0000-0000-000000000001', true),
    ('00000000-0000-0000-0001-000000000003', 'd0000000-0000-0000-0000-000000000000', '00000000-0000-0000-0000-000000000001', true),
    ('00000000-0000-0000-0001-000000000004', 'd0000000-0000-0000-0000-000000000000', '00000000-0000-0000-0000-000000000001', true),
    ('00000000-0000-0000-0001-000000000005', 'd0000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', true),
    ('00000000-0000-0000-0001-000000000006', 'd0000000-0000-0000-0000-000000000000', '00000000-0000-0000-0000-000000000001', true)
ON CONFLICT (user_id, branch_id) WHERE revoked_at IS NULL DO NOTHING;

-- ─── 3. workflow7 M2M OAuth2 client ──────────────────────────────────────────
-- client_id     = workflow7
-- client_secret = workflow7-m2m-dev-secret  (DEV ONLY — rotate in real deployments)
-- hash format   = base64(sha256(plain_secret)) per verifyClientSecret in oauth2.go
INSERT INTO oauth2_clients (
    id, client_id, org_id, name, description,
    client_type, token_endpoint_auth_method,
    allowed_scopes, allowed_redirect_uris, allowed_origins,
    client_secret_hash,
    token_expiration, refresh_token_expiration,
    allow_multiple_tokens, skip_consent_screen, is_active
) VALUES (
    '99999999-9999-9999-9999-999999999991',
    'workflow7',
    '00000000-0000-0000-0000-000000000001',
    'Workflow7 Engine',
    'M2M client used by workflow7 to call /internal/v1/user-context and other auth7 internal endpoints. DEV secret — rotate before production.',
    'machine', 'client_secret_basic',
    '["internal:read"]'::jsonb, '[]'::jsonb, '[]'::jsonb,
    'PpFTJqb3cNuzQWOKTvJt9cmKdggBOtjmL/xPKZzup0k=',
    900, 28800, true, true, true
) ON CONFLICT (client_id) DO UPDATE
    SET client_secret_hash = EXCLUDED.client_secret_hash,
        allowed_scopes     = EXCLUDED.allowed_scopes,
        client_type        = EXCLUDED.client_type,
        is_active          = true;
