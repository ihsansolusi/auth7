ALTER TABLE users
ADD COLUMN IF NOT EXISTS preferred_locale VARCHAR(8) NOT NULL DEFAULT 'id';

ALTER TABLE users
DROP CONSTRAINT IF EXISTS users_preferred_locale_check;

ALTER TABLE users
ADD CONSTRAINT users_preferred_locale_check CHECK (preferred_locale IN ('id', 'en'));
