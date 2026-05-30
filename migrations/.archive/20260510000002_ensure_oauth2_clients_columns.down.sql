-- Rollback: remove columns added by ensure migration
ALTER TABLE oauth2_clients
    DROP COLUMN IF EXISTS app_url,
    DROP COLUMN IF EXISTS icon_name,
    DROP COLUMN IF EXISTS icon_color;
