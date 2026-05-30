-- Revert Railway domain updates (remove Railway URIs, restore bos7.local app_urls)

UPDATE oauth2_clients SET app_url = 'http://bos7-enterprise.bos7.local:3003'
WHERE client_id = 'bos7-enterprise';

UPDATE oauth2_clients SET app_url = 'http://bos7-financing.bos7.local:3010'
WHERE client_id = 'bos7-financing';

UPDATE oauth2_clients SET app_url = 'http://bos7-funding.bos7.local:3011'
WHERE client_id = 'bos7-funding';

UPDATE oauth2_clients SET app_url = 'http://bos7-treasury.bos7.local:3012'
WHERE client_id = 'bos7-treasury';

UPDATE oauth2_clients SET app_url = 'http://bos7-smt.bos7.local:3013'
WHERE client_id = 'bos7-smt';

UPDATE oauth2_clients SET app_url = 'http://bos7-accounting.bos7.local:3014'
WHERE client_id = 'bos7-accounting';

UPDATE oauth2_clients SET app_url = 'http://bos7-cif.bos7.local:3015'
WHERE client_id = 'bos7-cif';

UPDATE oauth2_clients SET app_url = 'http://bos7-internalaccount.bos7.local:3016'
WHERE client_id = 'bos7-internalaccount';

UPDATE oauth2_clients SET app_url = 'http://bos7-remittance.bos7.local:3017'
WHERE client_id = 'bos7-remittance';

UPDATE oauth2_clients SET app_url = 'http://bos7-batchprocessing.bos7.local:3018'
WHERE client_id = 'bos7-batchprocessing';

UPDATE oauth2_clients SET app_url = 'http://bos7-template.bos7.local:3004'
WHERE client_id = 'bos7-template';
