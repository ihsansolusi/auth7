-- Rollback: Remove app_url, icon_name, icon_color from oauth2_clients
ALTER TABLE oauth2_clients
    DROP COLUMN IF EXISTS app_url,
    DROP COLUMN IF EXISTS icon_name,
    DROP COLUMN IF EXISTS icon_color;
