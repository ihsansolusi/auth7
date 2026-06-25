-- Confidential groups (PROJECTION of enterprise.confidential_groups) + RBAC grants.
-- ids/cf_code MUST stay byte-identical to enterprise demo seed (source of truth).
INSERT INTO confidential_groups (id, org_id, cf_code, description, updated_at)
VALUES
    ('c0000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', 'VIP',             'Nasabah VIP / Prioritas',          NOW()),
    ('c0000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000001', 'PEP',             'Politically Exposed Person',       NOW()),
    ('c0000000-0000-0000-0000-000000000003', '00000000-0000-0000-0000-000000000001', 'INTERNAL',        'Data Pegawai Internal',            NOW()),
    ('c0000000-0000-0000-0000-000000000004', '00000000-0000-0000-0000-000000000001', 'DIREKSI',         'Data Direksi & Komisaris',         NOW()),
    ('c0000000-0000-0000-0000-000000000005', '00000000-0000-0000-0000-000000000001', 'DANA_PEMERINTAH', 'Rekening Dana Pemerintah',         NOW()),
    ('c0000000-0000-0000-0000-000000000006', '00000000-0000-0000-0000-000000000001', 'DORMANT_BLOKIR',  'Rekening Dormant / Blokir Khusus', NOW()),
    ('c0000000-0000-0000-0000-000000000007', '00000000-0000-0000-0000-000000000001', 'KORPORAT',        'Nasabah Korporat Besar',           NOW())
ON CONFLICT (id) DO UPDATE
    SET cf_code = EXCLUDED.cf_code, description = EXCLUDED.description, updated_at = EXCLUDED.updated_at;

-- Access grants: role → confidential group.
--   SUPER_ADMIN (…0002-…0001) + AUDITOR (…0002-…0005): all groups.
--   BRANCH_MANAGER (…0002-…0002): VIP, PEP, KORPORAT.
INSERT INTO confidential_accesses (id, org_id, role_id, confidential_group_id, granted_by, granted_at)
SELECT gen_random_uuid(), '00000000-0000-0000-0000-000000000001', r.role_id, g.id, 'system', NOW()
FROM (VALUES
        ('00000000-0000-0000-0002-000000000001'::uuid),   -- SUPER_ADMIN → all
        ('00000000-0000-0000-0002-000000000005'::uuid)    -- AUDITOR → all
     ) AS r(role_id)
CROSS JOIN confidential_groups g
WHERE g.org_id = '00000000-0000-0000-0000-000000000001'
ON CONFLICT (org_id, role_id, confidential_group_id) DO NOTHING;

INSERT INTO confidential_accesses (id, org_id, role_id, confidential_group_id, granted_by, granted_at)
SELECT gen_random_uuid(), '00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0002-000000000002', g.id, 'system', NOW()
FROM confidential_groups g
WHERE g.org_id = '00000000-0000-0000-0000-000000000001'
  AND g.cf_code IN ('VIP', 'PEP', 'KORPORAT')
ON CONFLICT (org_id, role_id, confidential_group_id) DO NOTHING;
