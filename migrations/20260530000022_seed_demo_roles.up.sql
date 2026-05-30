INSERT INTO roles (id, org_id, code, name, description, is_default, created_at, updated_at)
VALUES
    (gen_random_uuid(), '00000000-0000-0000-0000-000000000001', 'SUPER_ADMIN',    'Super Admin',    'Akses penuh semua modul',      false, NOW(), NOW()),
    (gen_random_uuid(), '00000000-0000-0000-0000-000000000001', 'BRANCH_MANAGER', 'Branch Manager', 'Manajer cabang',               false, NOW(), NOW()),
    (gen_random_uuid(), '00000000-0000-0000-0000-000000000001', 'SUPERVISOR',     'Supervisor',     'Supervisor operasional',       false, NOW(), NOW()),
    (gen_random_uuid(), '00000000-0000-0000-0000-000000000001', 'TELLER',         'Teller',         'Teller transaksi',             true,  NOW(), NOW()),
    (gen_random_uuid(), '00000000-0000-0000-0000-000000000001', 'AUDITOR',        'Auditor',        'Audit trail access',           false, NOW(), NOW())
ON CONFLICT (org_id, code) DO NOTHING;
