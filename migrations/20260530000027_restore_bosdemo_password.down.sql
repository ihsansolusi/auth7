-- Restore bosdemo password back to 'password123' (the migration-23 default).
-- argon2id hash of 'password123' (per migration 23 demo seed):
--   $argon2id$v=19$m=65536,t=3,p=4$c2VlZF9zYWx0X2ZpeGVkIQ$N+pMwLuOqjb62N8jRGpZTng1AJkGETP4yvjfe6CWPRI

UPDATE user_credentials
SET secret_hash = '$argon2id$v=19$m=65536,t=3,p=4$c2VlZF9zYWx0X2ZpeGVkIQ$N+pMwLuOqjb62N8jRGpZTng1AJkGETP4yvjfe6CWPRI'
WHERE user_id = '00000000-0000-0000-0001-000000000002'
  AND is_current = true;
