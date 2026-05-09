-- Migration: 20260422000007_create_user_credentials rollback
-- Down

DROP TABLE IF EXISTS user_credentials;
DROP TABLE IF EXISTS user_credential_history;
