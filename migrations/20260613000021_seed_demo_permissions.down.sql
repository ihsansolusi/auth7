-- Remove the demo permission seed (only the codes inserted by the up migration).
DELETE FROM permissions WHERE code IN (
  'user:read','user:create','user:update','user:delete','user:lock','user:assign-role','user:assign-branch',
  'role:read','role:create','role:update','role:delete','role:assign-permission',
  'permission:read',
  'branch:read','branch:create','branch:update','branch:delete',
  'branch-type:read','branch-type:manage',
  'oauth2-client:read','oauth2-client:manage',
  'session:read','session:revoke',
  'audit:read',
  'access-management:view'
);
