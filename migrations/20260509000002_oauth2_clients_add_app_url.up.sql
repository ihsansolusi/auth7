-- Migration: Add app_url, icon_name, icon_color to oauth2_clients
ALTER TABLE oauth2_clients
    ADD COLUMN IF NOT EXISTS app_url    VARCHAR(512),
    ADD COLUMN IF NOT EXISTS icon_name  VARCHAR(64),
    ADD COLUMN IF NOT EXISTS icon_color VARCHAR(32);
