ALTER TABLE oauth2_authorization_codes
    ADD COLUMN IF NOT EXISTS roles TEXT[] DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS branch_id VARCHAR(36);

COMMENT ON COLUMN oauth2_authorization_codes.roles IS 'User roles at auth code issuance time, forwarded to access token';
COMMENT ON COLUMN oauth2_authorization_codes.branch_id IS 'Active branch_id at auth code issuance time, forwarded to access token';
