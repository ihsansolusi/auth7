ALTER TABLE oauth2_authorization_codes
    DROP COLUMN IF EXISTS roles,
    DROP COLUMN IF EXISTS branch_id;
