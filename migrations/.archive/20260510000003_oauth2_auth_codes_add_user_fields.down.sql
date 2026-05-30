ALTER TABLE oauth2_authorization_codes
    DROP COLUMN IF EXISTS username,
    DROP COLUMN IF EXISTS email;
