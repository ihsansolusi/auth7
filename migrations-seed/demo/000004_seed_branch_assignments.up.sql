-- Primary branch assignments for the 6 demo users.
-- admin/bosdemo/manager/spv/auditor → HQ (000), teller → KC BANDUNG (001)
INSERT INTO user_branch_assignments (id, user_id, branch_id, org_id, is_primary, assigned_by, assigned_at)
VALUES
    (gen_random_uuid(), '00000000-0000-0000-0001-000000000001', 'd0000000-0000-0000-0000-000000000000', '00000000-0000-0000-0000-000000000001', true, 'system', NOW()),
    (gen_random_uuid(), '00000000-0000-0000-0001-000000000002', 'd0000000-0000-0000-0000-000000000000', '00000000-0000-0000-0000-000000000001', true, 'system', NOW()),
    (gen_random_uuid(), '00000000-0000-0000-0001-000000000003', 'd0000000-0000-0000-0000-000000000000', '00000000-0000-0000-0000-000000000001', true, 'system', NOW()),
    (gen_random_uuid(), '00000000-0000-0000-0001-000000000004', 'd0000000-0000-0000-0000-000000000000', '00000000-0000-0000-0000-000000000001', true, 'system', NOW()),
    (gen_random_uuid(), '00000000-0000-0000-0001-000000000005', 'd0000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', true, 'system', NOW()),
    (gen_random_uuid(), '00000000-0000-0000-0001-000000000006', 'd0000000-0000-0000-0000-000000000000', '00000000-0000-0000-0000-000000000001', true, 'system', NOW())
ON CONFLICT (user_id, branch_id) DO NOTHING;
