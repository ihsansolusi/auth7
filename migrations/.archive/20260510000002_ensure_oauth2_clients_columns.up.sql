-- Ensure app_url, icon_name, icon_color exist on oauth2_clients.
-- Idempotent re-apply in case 20260509000002 ran on wrong search_path.
ALTER TABLE oauth2_clients
    ADD COLUMN IF NOT EXISTS app_url    VARCHAR(512),
    ADD COLUMN IF NOT EXISTS icon_name  VARCHAR(64),
    ADD COLUMN IF NOT EXISTS icon_color VARCHAR(32);
