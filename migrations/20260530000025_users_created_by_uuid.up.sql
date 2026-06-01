-- Promote users.created_by / users.updated_by from VARCHAR(36) to UUID.
--
-- Background: the original schema declared these columns as VARCHAR(36) with a
-- string default of 'system'. The domain User struct types them as *uuid.UUID
-- and pgx silently fails to Scan a non-UUID string into that field, surfacing
-- as a generic error in every UserRepository.GetByID call against seeded rows.
-- Surfaced by the smoke test of /internal/v1/user-context (Wave 17 #457).
--
-- Conversion strategy:
--   1. Cast existing values: 'system' (the legacy sentinel) → the zero UUID.
--      Anything else is assumed to already be a UUID string.
--   2. Alter column type to UUID.
--   3. Change default to the zero UUID so future inserts without an explicit
--      actor still pass Scan.

UPDATE users SET created_by = '00000000-0000-0000-0000-000000000000'
WHERE created_by IS NULL OR created_by = '' OR created_by = 'system';

UPDATE users SET updated_by = '00000000-0000-0000-0000-000000000000'
WHERE updated_by IS NULL OR updated_by = '' OR updated_by = 'system';

ALTER TABLE users
    ALTER COLUMN created_by DROP DEFAULT,
    ALTER COLUMN created_by TYPE UUID USING created_by::uuid,
    ALTER COLUMN created_by SET DEFAULT '00000000-0000-0000-0000-000000000000'::uuid,
    ALTER COLUMN updated_by DROP DEFAULT,
    ALTER COLUMN updated_by TYPE UUID USING updated_by::uuid,
    ALTER COLUMN updated_by SET DEFAULT '00000000-0000-0000-0000-000000000000'::uuid;
