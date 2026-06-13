-- All OAuth2 clients — extracted from running auth7 DB 2026-06-13.
-- client_secret_hash values are stored hashes (not plaintext).
-- Secrets for bos7-* web apps: see docs/security/05-OPERATIONS.md
-- M2M secrets: workflow7-m2m-dev-secret, rotate before production.

INSERT INTO oauth2_clients (
    id, client_id, org_id, name, description,
    client_type, token_endpoint_auth_method,
    allowed_scopes, allowed_redirect_uris, allowed_origins,
    client_secret_hash, public_key_jwk,
    app_url, icon_name, icon_color,
    token_expiration, refresh_token_expiration,
    allow_multiple_tokens, skip_consent_screen, is_active
) VALUES

-- ─── M2M / Machine clients ────────────────────────────────────────────────────
(
    'd372db85-cc7c-4cf0-8e4e-250a49eb3fef', 'workflow7',
    '00000000-0000-0000-0000-000000000001',
    'Workflow7 Service', 'M2M service client for workflow7',
    'machine', 'client_secret_basic',
    '["internal:read"]'::jsonb, '[]'::jsonb, '[]'::jsonb,
    'uAygWs2jgWxPmJg4gv3Ru3EPmBKz2CraeSN2Du9lyAg=', '',
    '', '', '', 900, 28800, true, true, true
),
(
    '5f949828-d9be-43d3-adaf-98a5f39683fd', '019e59b3-9b34-70f1-9428-1d8452b90abb',
    '00000000-0000-0000-0000-000000000001',
    'Enterprise Branchsync M2M', 'M2M client untuk branchsync poller di auth7',
    'machine', 'client_secret_basic',
    '["service"]'::jsonb, '[]'::jsonb, '[]'::jsonb,
    '5e4f3a842c39464f82cc8fdcbc3f2997a2402e555dc217b43a3b099b2fba662f', '',
    '', '', '', 900, 28800, false, false, true
),
(
    '0527ff51-1742-4ec9-b67c-ce144c9542a3', '0527ff51-1742-4ec9-b67c-ce144c9542a3',
    '00000000-0000-0000-0000-000000000001',
    'service7-m2m', '',
    'machine', 'client_secret_basic',
    '["service"]'::jsonb, '[]'::jsonb, '[]'::jsonb,
    'EFr8hkOvEs3VR5sRHDWZ8y/zFH3lyBEqf9u/Gpjq/rU=', '',
    '', '', '', 900, 28800, false, false, true
),

-- ─── BOS7 Web Applications ────────────────────────────────────────────────────
(
    '00000000-0000-0000-0000-000000000902', 'bos7-workflow',
    '00000000-0000-0000-0000-000000000001',
    'BOS7 Workflow', 'BPM engine — persetujuan, eskalasi, monitoring proses bisnis',
    'web', 'client_secret_basic',
    '["openid","profile","email","roles","offline_access"]'::jsonb,
    '["http://localhost:3002/api/auth/callback","https://workflow.bos7.local/api/auth/callback","https://workflow.dev.ihsansolusi.co.id/api/auth/callback","https://bos7-workflow.up.railway.app/api/auth/callback"]'::jsonb,
    '[]'::jsonb,
    '2c7f4589b2fbbe3c0d3ba36bb91d6c17552be7e2374d9dd7363aff54fbe3ea83', '',
    'https://workflow.bos7.local', 'FlowStream', '#8a3ffc',
    900, 28800, false, false, true
),
(
    'bf92caf4-8d24-4e86-83af-2afc66e664c1', 'bos7-portal',
    '00000000-0000-0000-0000-000000000001',
    'BOS7 Portal', 'Main banking portal launcher',
    'web', 'client_secret_basic',
    '["openid","profile","email"]'::jsonb,
    '["http://localhost:3006/api/auth/callback","https://portal.bos7.local/api/auth/callback","https://portal.dev.ihsansolusi.co.id/api/auth/callback","https://bos7-portal.up.railway.app/api/auth/callback"]'::jsonb,
    '["http://localhost:3006"]'::jsonb,
    '8810845ac7f98ef512240d68da90af9315b6afa09f1d1710f2a2b95b2b6f4525', '',
    'https://portal.bos7.local', 'Launch', '#0f62fe',
    900, 28800, false, true, true
),
(
    '94f8da9e-910d-4595-9ec9-605fe2cb8bf1', 'bos7-enterprise',
    '00000000-0000-0000-0000-000000000001',
    'BOS7 Enterprise', 'Master data organisasi — departemen, cabang, karyawan',
    'web', 'client_secret_basic',
    '["openid","profile","email"]'::jsonb,
    '["http://localhost:3003/api/auth/callback","https://enterprise.bos7.local/api/auth/callback","https://bos7-enterprise.up.railway.app/api/auth/callback"]'::jsonb,
    '["http://localhost:3003"]'::jsonb,
    'b29557b642e1c7dc5b172e48079842ec2663fc4fa8dcdbb1c4c47acb0b5e9b69', '',
    'https://enterprise.bos7.local', 'Enterprise', '#0f62fe',
    900, 28800, false, true, true
),
(
    '2846c61d-e374-4102-b369-0d3f2a02d498', 'bos7-cif',
    '00000000-0000-0000-0000-000000000001',
    'BOS7 CIF', 'Customer Information File — data induk nasabah',
    'web', 'client_secret_basic',
    '["openid","profile","email"]'::jsonb,
    '["http://localhost:3015/api/auth/callback","https://cif.bos7.local/api/auth/callback","https://bos7-cif.up.railway.app/api/auth/callback"]'::jsonb,
    '["http://localhost:3015"]'::jsonb,
    '3efee451e84c6c0ffa08f69401606b198ab77fffac8e87d34f6ad625a8fe5f9a', '',
    'https://cif.bos7.local', 'UserIdentification', '#0f62fe',
    900, 28800, false, true, true
),
(
    '961ed14e-f998-414b-9612-27620fc83a0f', 'bos7-financing',
    '00000000-0000-0000-0000-000000000001',
    'BOS7 Financing', 'Manajemen produk dan transaksi pembiayaan',
    'web', 'client_secret_basic',
    '["openid","profile","email"]'::jsonb,
    '["http://localhost:3010/api/auth/callback","https://financing.bos7.local/api/auth/callback","https://bos7-financing.up.railway.app/api/auth/callback"]'::jsonb,
    '["http://localhost:3010"]'::jsonb,
    '6ec50d1810fa34e0860ae969e49027fd7dd71b8619a2b6904efec3287e802c42', '',
    'https://financing.bos7.local', 'Finance', '#198038',
    900, 28800, false, true, true
),
(
    '48eb6a89-7c63-492b-b22b-0b0b49f35710', 'bos7-funding',
    '00000000-0000-0000-0000-000000000001',
    'BOS7 Funding', 'Produk tabungan, deposito, dan transaksi pendanaan',
    'web', 'client_secret_basic',
    '["openid","profile","email"]'::jsonb,
    '["http://localhost:3011/api/auth/callback","https://funding.bos7.local/api/auth/callback","https://bos7-funding.up.railway.app/api/auth/callback"]'::jsonb,
    '["http://localhost:3011"]'::jsonb,
    '9b32cbf62ec99dfd155bd1fc16f0778118954793e792982c33bfa3d8210a0eca', '',
    'https://funding.bos7.local', 'Money', '#198038',
    900, 28800, false, true, true
),
(
    'c643b60d-c8b7-44d7-96c5-98bcdd036537', 'bos7-treasury',
    '00000000-0000-0000-0000-000000000001',
    'BOS7 Treasury', 'Posisi treasury, profit distribution, dan FTP',
    'web', 'client_secret_basic',
    '["openid","profile","email"]'::jsonb,
    '["http://localhost:3012/api/auth/callback","https://treasury.bos7.local/api/auth/callback","https://bos7-treasury.up.railway.app/api/auth/callback"]'::jsonb,
    '["http://localhost:3012"]'::jsonb,
    'f70fc402e06c014c155e2a0f1d3e5e0c926fa0f774c7d4595053e7b505770475', '',
    'https://treasury.bos7.local', 'Currency', '#6929c4',
    900, 28800, false, true, true
),
(
    'c636907f-2fdd-4e5e-900f-42637dca8f31', 'bos7-accounting',
    '00000000-0000-0000-0000-000000000001',
    'BOS7 Accounting', 'Chart of account, jurnal, dan laporan keuangan',
    'web', 'client_secret_basic',
    '["openid","profile","email"]'::jsonb,
    '["http://localhost:3014/api/auth/callback","https://accounting.bos7.local/api/auth/callback","https://bos7-accounting.up.railway.app/api/auth/callback"]'::jsonb,
    '["http://localhost:3014"]'::jsonb,
    '452bf6828892d97de48b52fc23cb77ed8b45856cdea2b0800decaeb20a2cd3a0', '',
    'https://accounting.bos7.local', 'ChartLineData', '#0043ce',
    900, 28800, false, true, true
),
(
    '236c8c9b-978d-4bd7-9e3b-7647b4287f96', 'bos7-internalaccount',
    '00000000-0000-0000-0000-000000000001',
    'BOS7 Internal Account', 'Rekening internal bank dan transaksi antar cabang',
    'web', 'client_secret_basic',
    '["openid","profile","email"]'::jsonb,
    '["http://localhost:3016/api/auth/callback","https://internalaccount.bos7.local/api/auth/callback","https://bos7-internalaccount.up.railway.app/api/auth/callback"]'::jsonb,
    '["http://localhost:3016"]'::jsonb,
    '7fe52794124cd18ec6edf01f9efeb06bffcf12b5f82652035da64ae16f511a52', '',
    'https://internalaccount.bos7.local', 'Account', '#9f1853',
    900, 28800, false, true, true
),
(
    '6f9434eb-7e22-4f50-8062-dfacb751e144', 'bos7-remittance',
    '00000000-0000-0000-0000-000000000001',
    'BOS7 Remittance', 'Transfer antar bank, RTGS, SKNBI, dan kliring',
    'web', 'client_secret_basic',
    '["openid","profile","email"]'::jsonb,
    '["http://localhost:3017/api/auth/callback","https://remittance.bos7.local/api/auth/callback","https://bos7-remittance.up.railway.app/api/auth/callback"]'::jsonb,
    '["http://localhost:3017"]'::jsonb,
    'ce01e0bf98e5f982fd90a68171612594df71862b13744b290e1aee9485a7fa57', '',
    'https://remittance.bos7.local', 'Send', '#b28600',
    900, 28800, false, true, true
),
(
    'db7ee46f-8d19-4d51-a93e-3cfe236204ac', 'bos7-smt',
    '00000000-0000-0000-0000-000000000001',
    'BOS7 SMT', 'Smart Migration and Transaction',
    'web', 'client_secret_basic',
    '["openid","profile","email"]'::jsonb,
    '["http://localhost:3013/api/auth/callback","https://smt.bos7.local/api/auth/callback","https://bos7-smt.up.railway.app/api/auth/callback"]'::jsonb,
    '["http://localhost:3013"]'::jsonb,
    'a5f26ffd1bd9142cc10b7ced05f71b419f30ad3fd0d9b88ad3465c3565f40d80', '',
    'https://smt.bos7.local', 'Migrate', '#005d5d',
    900, 28800, false, true, true
),
(
    'e6248bf0-0552-4e1c-baa6-4fb51c092a1f', 'bos7-batchprocessing',
    '00000000-0000-0000-0000-000000000001',
    'BOS7 Batch Processing', 'Penjadwalan dan eksekusi proses batch periodik',
    'web', 'client_secret_basic',
    '["openid","profile","email"]'::jsonb,
    '["http://localhost:3018/api/auth/callback","https://batchprocessing.bos7.local/api/auth/callback","https://bos7-batchprocessing.up.railway.app/api/auth/callback"]'::jsonb,
    '["http://localhost:3018"]'::jsonb,
    '6c367d186687cf37c5ded38dcc83655a98b600084260c5464e487de4a12c4d53', '',
    'https://batchprocessing.bos7.local', 'BatchJob', '#4d5358',
    900, 28800, false, true, true
),
(
    '46a15c66-3493-4c50-80de-318a08ccab1b', 'bos7-template',
    '00000000-0000-0000-0000-000000000001',
    'bos7-template Dev Client', '',
    'web', 'client_secret_post',
    '["openid","profile","email"]'::jsonb,
    '["http://localhost:3004/api/auth/callback","https://template.bos7.local/api/auth/callback","https://bos7-template.up.railway.app/api/auth/callback"]'::jsonb,
    '[]'::jsonb,
    'b6091476f6d158417e9c2fd5727c221e10a570a173d252c6b75b8f6c3c553eb4', '',
    '', '', '',
    900, 28800, false, false, true
),
(
    'b9279bd6-479e-4aa8-92fc-89c91f5640f8', 'b9279bd6-479e-4aa8-92fc-89c91f5640f8',
    '00000000-0000-0000-0000-000000000001',
    'bos7-template', '',
    'web', 'client_secret_post',
    '["openid","profile","email"]'::jsonb,
    '["http://localhost:3000/api/auth/callback"]'::jsonb,
    '[]'::jsonb,
    'pNdxMzG3MJtqyLxHjEX5YEdAzSjcMQw/sllEHw1HgPI=', '',
    '', '', '',
    900, 28800, false, false, true
)
ON CONFLICT (client_id) DO UPDATE
    SET client_secret_hash       = EXCLUDED.client_secret_hash,
        allowed_scopes           = EXCLUDED.allowed_scopes,
        allowed_redirect_uris    = EXCLUDED.allowed_redirect_uris,
        allowed_origins          = EXCLUDED.allowed_origins,
        name                     = EXCLUDED.name,
        description              = EXCLUDED.description,
        app_url                  = EXCLUDED.app_url,
        icon_name                = EXCLUDED.icon_name,
        icon_color               = EXCLUDED.icon_color,
        is_active                = EXCLUDED.is_active;
