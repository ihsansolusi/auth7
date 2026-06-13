-- 000006_seed_ibankdb_employees.down.sql
-- Rollback: hapus ibankdb employees + ibent roles (jaga 6 demo users hardcoded)

-- Hapus assignments untuk ibankdb users (UUID bukan 00000000-0000-0000-0001-*)
DELETE FROM user_branch_assignments
WHERE user_id IN (
    SELECT id FROM users
    WHERE org_id = '00000000-0000-0000-0000-000000000001'
      AND id NOT LIKE '00000000-0000-0000-0001-%'
);
DELETE FROM user_roles
WHERE user_id IN (
    SELECT id FROM users
    WHERE org_id = '00000000-0000-0000-0000-000000000001'
      AND id NOT LIKE '00000000-0000-0000-0001-%'
);
DELETE FROM user_credentials
WHERE user_id IN (
    SELECT id FROM users
    WHERE org_id = '00000000-0000-0000-0000-000000000001'
      AND id NOT LIKE '00000000-0000-0000-0001-%'
);
DELETE FROM users
WHERE org_id = '00000000-0000-0000-0000-000000000001'
  AND id NOT LIKE '00000000-0000-0000-0001-%';

-- Hapus ibent roles (code tidak termasuk 5 simplified roles)
DELETE FROM roles
WHERE org_id = '00000000-0000-0000-0000-000000000001'
  AND code NOT IN ('SUPER_ADMIN', 'BRANCH_MANAGER', 'SUPERVISOR', 'TELLER', 'AUDITOR');
