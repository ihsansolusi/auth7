-- Demo seed: baseline permission catalog so the Access Management → Roles →
-- Manage Permissions UI has data to pick from. Global (org-agnostic) reference
-- data; idempotent via the unique `code` constraint. Safe to drop later.
INSERT INTO permissions (code, name, description, resource_type) VALUES
  ('user:read',              'View Users',            'View users and their details',          'user'),
  ('user:create',            'Create User',           'Create new users',                      'user'),
  ('user:update',            'Update User',           'Edit user details',                     'user'),
  ('user:delete',            'Delete User',           'Delete (soft) users',                   'user'),
  ('user:lock',              'Lock/Unlock User',      'Lock or unlock user access',            'user'),
  ('user:assign-role',       'Assign User Roles',     'Assign or revoke roles for a user',     'user'),
  ('user:assign-branch',     'Assign User Branches',  'Assign or revoke branches for a user',  'user'),
  ('role:read',              'View Roles',            'View roles and their permissions',      'role'),
  ('role:create',            'Create Role',           'Create new roles',                      'role'),
  ('role:update',            'Update Role',           'Edit role details',                     'role'),
  ('role:delete',            'Delete Role',           'Delete roles',                          'role'),
  ('role:assign-permission', 'Assign Permissions',    'Assign permissions to a role',          'role'),
  ('permission:read',        'View Permissions',      'View the permission catalog',           'permission'),
  ('branch:read',            'View Branches',         'View branches',                         'branch'),
  ('branch:create',          'Create Branch',         'Create branches',                       'branch'),
  ('branch:update',          'Update Branch',         'Edit branches',                         'branch'),
  ('branch:delete',          'Delete Branch',         'Delete branches',                       'branch'),
  ('branch-type:read',       'View Branch Types',     'View branch types',                     'branch_type'),
  ('branch-type:manage',     'Manage Branch Types',   'Create/edit/delete branch types',       'branch_type'),
  ('oauth2-client:read',     'View OAuth2 Clients',   'View OAuth2 clients',                    'oauth2_client'),
  ('oauth2-client:manage',   'Manage OAuth2 Clients', 'Create/edit/delete OAuth2 clients',     'oauth2_client'),
  ('session:read',           'View Sessions',         'View active sessions',                  'session'),
  ('session:revoke',         'Revoke Sessions',       'Revoke active sessions',                'session'),
  ('audit:read',             'View Audit Log',        'View the authentication audit log',     'audit'),
  ('access-management:view', 'Access Management',     'Access the Access Management module',   'access-management')
ON CONFLICT (code) DO NOTHING;
