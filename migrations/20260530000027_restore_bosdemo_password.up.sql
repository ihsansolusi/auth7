-- Restore bosdemo's password to 'demo@123' to preserve operator muscle memory.
--
-- Demo seed (migration 23) sets all 6 demo users to 'password123'. The bosdemo
-- user historically had 'demo@123' (per the legacy seed/demo/seed_005_bosdemo.sql
-- fixture that was applied to all pre-rebaseline environments). Engineers
-- testing manually expect demo@123 for bosdemo and password123 for the others.
--
-- argon2id hash of 'demo@123' (carried over from seed_005_bosdemo.sql):
--   $argon2id$v=19$m=65536,t=3,p=4$ZxlitAZB7O8lb2CvXikUcg$Oh2fnb1wNAcNAPAbvJUsDhZA01xgjjwY4efttjAaE+Y

UPDATE user_credentials
SET secret_hash = '$argon2id$v=19$m=65536,t=3,p=4$ZxlitAZB7O8lb2CvXikUcg$Oh2fnb1wNAcNAPAbvJUsDhZA01xgjjwY4efttjAaE+Y'
WHERE user_id = '00000000-0000-0000-0001-000000000002'
  AND is_current = true;
