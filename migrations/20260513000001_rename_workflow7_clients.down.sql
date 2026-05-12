-- Revert bos7-workflow → workflow7-web
UPDATE oauth2_clients
SET
    client_id           = 'workflow7-web',
    name                = 'Workflow7 Web',
    client_secret_hash  = 'dc63d2325d239b92d0f169c2631ae637f8c9b08fcab8d4d276b6acd404a617ee',
    allowed_redirect_uris = '{http://localhost:3002/api/auth/callback,https://workflow.bos7.local/api/auth/callback,https://workflow.dev.ihsansolusi.co.id/api/auth/callback}',
    app_url             = 'https://workflow.bos7.local'
WHERE id = '00000000-0000-0000-0000-000000000902';

-- Revert workflow7 → workflow7-svc
UPDATE oauth2_clients
SET
    client_id          = 'workflow7-svc',
    client_secret_hash = '6577925abc0d51b6559284157367fde87b09e87d58da54fdc5551b98fedb7f5e'
WHERE id = '00000000-0000-0000-0000-000000000905';
