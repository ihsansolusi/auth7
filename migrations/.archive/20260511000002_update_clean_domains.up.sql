-- Remove old -development redirect URIs and add clean ones,
-- update app_url to new domains without -development suffix

-- bos7-portal
UPDATE oauth2_clients
SET allowed_redirect_uris = array_remove(allowed_redirect_uris, 'https://bos7-portal-development.up.railway.app/api/auth/callback');
UPDATE oauth2_clients
SET allowed_redirect_uris = array_append(allowed_redirect_uris, 'https://bos7-portal.up.railway.app/api/auth/callback')
WHERE client_id = 'bos7-portal'
  AND NOT ('https://bos7-portal.up.railway.app/api/auth/callback' = ANY(allowed_redirect_uris));
UPDATE oauth2_clients SET app_url = 'https://bos7-portal.up.railway.app' WHERE client_id = 'bos7-portal';

-- bos7-template
UPDATE oauth2_clients
SET allowed_redirect_uris = array_remove(allowed_redirect_uris, 'https://bos7-template-development.up.railway.app/api/auth/callback');
UPDATE oauth2_clients
SET allowed_redirect_uris = array_append(allowed_redirect_uris, 'https://bos7-template.up.railway.app/api/auth/callback')
WHERE client_id = 'bos7-template'
  AND NOT ('https://bos7-template.up.railway.app/api/auth/callback' = ANY(allowed_redirect_uris));
UPDATE oauth2_clients SET app_url = 'https://bos7-template.up.railway.app' WHERE client_id = 'bos7-template';

-- bos7-enterprise
UPDATE oauth2_clients
SET allowed_redirect_uris = array_remove(allowed_redirect_uris, 'https://bos7-enterprise-development.up.railway.app/api/auth/callback');
UPDATE oauth2_clients
SET allowed_redirect_uris = array_append(allowed_redirect_uris, 'https://bos7-enterprise.up.railway.app/api/auth/callback')
WHERE client_id = 'bos7-enterprise'
  AND NOT ('https://bos7-enterprise.up.railway.app/api/auth/callback' = ANY(allowed_redirect_uris));
UPDATE oauth2_clients SET app_url = 'https://bos7-enterprise.up.railway.app' WHERE client_id = 'bos7-enterprise';

-- bos7-financing
UPDATE oauth2_clients
SET allowed_redirect_uris = array_remove(allowed_redirect_uris, 'https://bos7-financing-development.up.railway.app/api/auth/callback');
UPDATE oauth2_clients
SET allowed_redirect_uris = array_append(allowed_redirect_uris, 'https://bos7-financing.up.railway.app/api/auth/callback')
WHERE client_id = 'bos7-financing'
  AND NOT ('https://bos7-financing.up.railway.app/api/auth/callback' = ANY(allowed_redirect_uris));
UPDATE oauth2_clients SET app_url = 'https://bos7-financing.up.railway.app' WHERE client_id = 'bos7-financing';

-- bos7-funding
UPDATE oauth2_clients
SET allowed_redirect_uris = array_remove(allowed_redirect_uris, 'https://bos7-funding-development.up.railway.app/api/auth/callback');
UPDATE oauth2_clients
SET allowed_redirect_uris = array_append(allowed_redirect_uris, 'https://bos7-funding.up.railway.app/api/auth/callback')
WHERE client_id = 'bos7-funding'
  AND NOT ('https://bos7-funding.up.railway.app/api/auth/callback' = ANY(allowed_redirect_uris));
UPDATE oauth2_clients SET app_url = 'https://bos7-funding.up.railway.app' WHERE client_id = 'bos7-funding';

-- bos7-treasury
UPDATE oauth2_clients
SET allowed_redirect_uris = array_remove(allowed_redirect_uris, 'https://bos7-treasury-development.up.railway.app/api/auth/callback');
UPDATE oauth2_clients
SET allowed_redirect_uris = array_append(allowed_redirect_uris, 'https://bos7-treasury.up.railway.app/api/auth/callback')
WHERE client_id = 'bos7-treasury'
  AND NOT ('https://bos7-treasury.up.railway.app/api/auth/callback' = ANY(allowed_redirect_uris));
UPDATE oauth2_clients SET app_url = 'https://bos7-treasury.up.railway.app' WHERE client_id = 'bos7-treasury';

-- bos7-smt
UPDATE oauth2_clients
SET allowed_redirect_uris = array_remove(allowed_redirect_uris, 'https://bos7-smt-development.up.railway.app/api/auth/callback');
UPDATE oauth2_clients
SET allowed_redirect_uris = array_append(allowed_redirect_uris, 'https://bos7-smt.up.railway.app/api/auth/callback')
WHERE client_id = 'bos7-smt'
  AND NOT ('https://bos7-smt.up.railway.app/api/auth/callback' = ANY(allowed_redirect_uris));
UPDATE oauth2_clients SET app_url = 'https://bos7-smt.up.railway.app' WHERE client_id = 'bos7-smt';

-- bos7-accounting
UPDATE oauth2_clients
SET allowed_redirect_uris = array_remove(allowed_redirect_uris, 'https://bos7-accounting-development.up.railway.app/api/auth/callback');
UPDATE oauth2_clients
SET allowed_redirect_uris = array_append(allowed_redirect_uris, 'https://bos7-accounting.up.railway.app/api/auth/callback')
WHERE client_id = 'bos7-accounting'
  AND NOT ('https://bos7-accounting.up.railway.app/api/auth/callback' = ANY(allowed_redirect_uris));
UPDATE oauth2_clients SET app_url = 'https://bos7-accounting.up.railway.app' WHERE client_id = 'bos7-accounting';

-- bos7-cif
UPDATE oauth2_clients
SET allowed_redirect_uris = array_remove(allowed_redirect_uris, 'https://bos7-cif-development.up.railway.app/api/auth/callback');
UPDATE oauth2_clients
SET allowed_redirect_uris = array_append(allowed_redirect_uris, 'https://bos7-cif.up.railway.app/api/auth/callback')
WHERE client_id = 'bos7-cif'
  AND NOT ('https://bos7-cif.up.railway.app/api/auth/callback' = ANY(allowed_redirect_uris));
UPDATE oauth2_clients SET app_url = 'https://bos7-cif.up.railway.app' WHERE client_id = 'bos7-cif';

-- bos7-internalaccount
UPDATE oauth2_clients
SET allowed_redirect_uris = array_remove(allowed_redirect_uris, 'https://bos7-internalaccount-development.up.railway.app/api/auth/callback');
UPDATE oauth2_clients
SET allowed_redirect_uris = array_append(allowed_redirect_uris, 'https://bos7-internalaccount.up.railway.app/api/auth/callback')
WHERE client_id = 'bos7-internalaccount'
  AND NOT ('https://bos7-internalaccount.up.railway.app/api/auth/callback' = ANY(allowed_redirect_uris));
UPDATE oauth2_clients SET app_url = 'https://bos7-internalaccount.up.railway.app' WHERE client_id = 'bos7-internalaccount';

-- bos7-remittance
UPDATE oauth2_clients
SET allowed_redirect_uris = array_remove(allowed_redirect_uris, 'https://bos7-remittance-development.up.railway.app/api/auth/callback');
UPDATE oauth2_clients
SET allowed_redirect_uris = array_append(allowed_redirect_uris, 'https://bos7-remittance.up.railway.app/api/auth/callback')
WHERE client_id = 'bos7-remittance'
  AND NOT ('https://bos7-remittance.up.railway.app/api/auth/callback' = ANY(allowed_redirect_uris));
UPDATE oauth2_clients SET app_url = 'https://bos7-remittance.up.railway.app' WHERE client_id = 'bos7-remittance';

-- bos7-batchprocessing
UPDATE oauth2_clients
SET allowed_redirect_uris = array_remove(allowed_redirect_uris, 'https://bos7-batchprocessing-development.up.railway.app/api/auth/callback');
UPDATE oauth2_clients
SET allowed_redirect_uris = array_append(allowed_redirect_uris, 'https://bos7-batchprocessing.up.railway.app/api/auth/callback')
WHERE client_id = 'bos7-batchprocessing'
  AND NOT ('https://bos7-batchprocessing.up.railway.app/api/auth/callback' = ANY(allowed_redirect_uris));
UPDATE oauth2_clients SET app_url = 'https://bos7-batchprocessing.up.railway.app' WHERE client_id = 'bos7-batchprocessing';
