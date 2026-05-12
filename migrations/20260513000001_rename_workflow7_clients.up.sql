-- Rename workflow7-web → bos7-workflow (web client)
UPDATE oauth2_clients
SET
    client_id           = 'bos7-workflow',
    name                = 'BOS7 Workflow',
    client_secret_hash  = '2c7f4589b2fbbe3c0d3ba36bb91d6c17552be7e2374d9dd7363aff54fbe3ea83',
    allowed_redirect_uris = '{http://localhost:3002/api/auth/callback,https://workflow.bos7.local/api/auth/callback,https://workflow.dev.ihsansolusi.co.id/api/auth/callback,https://bos7-workflow.up.railway.app/api/auth/callback}',
    app_url             = 'https://bos7-workflow.up.railway.app'
WHERE id = '00000000-0000-0000-0000-000000000902';

-- Rename workflow7-svc → workflow7 (M2M client)
UPDATE oauth2_clients
SET
    client_id          = 'workflow7',
    client_secret_hash = 'b80ca05acda3816c4f98983882fdd1bb710f9812b3d82ada7923760eef65c808'
WHERE id = '00000000-0000-0000-0000-000000000905';
