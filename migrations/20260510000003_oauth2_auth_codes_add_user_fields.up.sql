ALTER TABLE oauth2_authorization_codes
    ADD COLUMN IF NOT EXISTS username VARCHAR(100),
    ADD COLUMN IF NOT EXISTS email    VARCHAR(255);
