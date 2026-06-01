ALTER TABLE users
    ALTER COLUMN created_by DROP DEFAULT,
    ALTER COLUMN created_by TYPE VARCHAR(36) USING created_by::text,
    ALTER COLUMN created_by SET DEFAULT 'system',
    ALTER COLUMN updated_by DROP DEFAULT,
    ALTER COLUMN updated_by TYPE VARCHAR(36) USING updated_by::text,
    ALTER COLUMN updated_by SET DEFAULT 'system';

UPDATE users SET created_by = 'system' WHERE created_by = '00000000-0000-0000-0000-000000000000'::uuid;
UPDATE users SET updated_by = 'system' WHERE updated_by = '00000000-0000-0000-0000-000000000000'::uuid;
