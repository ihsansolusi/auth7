DELETE FROM oauth2_clients WHERE client_id = 'workflow7';

DELETE FROM user_branch_assignments
WHERE branch_id IN (
    '5c7850c8-0c4e-5e5b-b899-c7b933122888',
    '30f4c0e9-0540-5aa7-937f-d620f2cc6293'
);

DELETE FROM branches WHERE id IN (
    '5c7850c8-0c4e-5e5b-b899-c7b933122888',
    '30f4c0e9-0540-5aa7-937f-d620f2cc6293'
);
