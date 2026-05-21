-- Seed: bosdemo super admin user for manual verification (password: demo@123)

-- 1) user record
INSERT INTO users (
  id, org_id, username, email, full_name, status,
  email_verified, mfa_enabled
) VALUES (
  'bbbbbbbb-d3e0-0000-0000-bbbbbbbbbbbb',
  '00000000-0000-0000-0000-000000000001',
  'bosdemo',
  'bosdemo@bank.local',
  'BOS Demo (Super Admin)',
  'active',
  TRUE,
  FALSE
)
ON CONFLICT (id) DO UPDATE
  SET status = EXCLUDED.status,
      email_verified = TRUE,
      full_name = EXCLUDED.full_name,
      email = EXCLUDED.email;

-- 2) password credential (argon2id of 'demo@123')
-- mark any older credentials non-current first, then insert the fresh one
UPDATE user_credentials
  SET is_current = FALSE
  WHERE user_id = 'bbbbbbbb-d3e0-0000-0000-bbbbbbbbbbbb';

INSERT INTO user_credentials (
  id, user_id, credential_type, secret_hash, version, is_current
) VALUES (
  'cccccccc-d3e0-0000-0000-cccccccccccc',
  'bbbbbbbb-d3e0-0000-0000-bbbbbbbbbbbb',
  'password',
  '$argon2id$v=19$m=65536,t=3,p=4$ZxlitAZB7O8lb2CvXikUcg$Oh2fnb1wNAcNAPAbvJUsDhZA01xgjjwY4efttjAaE+Y',
  1,
  TRUE
)
ON CONFLICT (id) DO UPDATE
  SET secret_hash = EXCLUDED.secret_hash,
      is_current = TRUE;

-- 3) Role bindings — bosdemo gets BOTH role variants:
--   a) super_admin (lowercase, auth7-canonical) — REQUIRED by auth7 admin middleware
--      which whitelists ["admin","super_admin"] (lowercase only).
--   b) SUPER_ADMIN (uppercase, ibent-seeded) — kept so any bos7-enterprise UI gate that
--      reads the uppercase form keeps working.
-- Drop any stale bosdemo bindings to other roles first (idempotent re-runs).
DELETE FROM user_roles
  WHERE user_id = 'bbbbbbbb-d3e0-0000-0000-bbbbbbbbbbbb'
    AND role_id NOT IN (
      '00000000-0000-0000-0000-000000000601',
      '3247e63d-5369-5270-b093-02a54970c7ae'
    );

INSERT INTO user_roles (
  id, user_id, role_id, org_id, branch_id, granted_by
) VALUES (
  'dddddddd-d3e0-0000-0000-dddddddddddd',
  'bbbbbbbb-d3e0-0000-0000-bbbbbbbbbbbb',
  '00000000-0000-0000-0000-000000000601',  -- super_admin (lowercase)
  '00000000-0000-0000-0000-000000000001',
  '5c7850c8-0c4e-5e5b-b899-c7b933122888',  -- KANTOR PUSAT
  'bbbbbbbb-d3e0-0000-0000-bbbbbbbbbbbb'
)
ON CONFLICT (id) DO UPDATE SET role_id = EXCLUDED.role_id;

INSERT INTO user_roles (
  id, user_id, role_id, org_id, branch_id, granted_by
) VALUES (
  'dddddddd-d3e0-0001-0000-dddddddddddd',
  'bbbbbbbb-d3e0-0000-0000-bbbbbbbbbbbb',
  '3247e63d-5369-5270-b093-02a54970c7ae',  -- SUPER_ADMIN (uppercase)
  '00000000-0000-0000-0000-000000000001',
  '5c7850c8-0c4e-5e5b-b899-c7b933122888',
  'bbbbbbbb-d3e0-0000-0000-bbbbbbbbbbbb'
)
ON CONFLICT (id) DO UPDATE SET role_id = EXCLUDED.role_id;

-- 4) Primary branch assignment (KANTOR PUSAT)
INSERT INTO user_branch_assignments (
  id, user_id, branch_id, is_primary
) VALUES (
  'eeeeeeee-d3e0-0000-0000-eeeeeeeeeeee',
  'bbbbbbbb-d3e0-0000-0000-bbbbbbbbbbbb',
  '5c7850c8-0c4e-5e5b-b899-c7b933122888',
  TRUE
)
ON CONFLICT (user_id, branch_id) DO UPDATE
  SET is_primary = EXCLUDED.is_primary;
