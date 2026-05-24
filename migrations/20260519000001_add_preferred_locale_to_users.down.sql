ALTER TABLE users
DROP CONSTRAINT IF EXISTS users_preferred_locale_check;

ALTER TABLE users
DROP COLUMN IF EXISTS preferred_locale;
