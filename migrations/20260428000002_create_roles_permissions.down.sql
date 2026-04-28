-- Migration: Rollback roles and permissions tables
-- Issue: #70

DROP INDEX IF EXISTS idx_user_roles_branch;
DROP INDEX IF EXISTS idx_user_roles_org;
DROP INDEX IF EXISTS idx_user_roles_role;
DROP INDEX IF EXISTS idx_user_roles_user;
DROP INDEX IF EXISTS idx_permissions_resource;
DROP INDEX IF EXISTS idx_permissions_code;
DROP INDEX IF EXISTS idx_roles_code;
DROP INDEX IF EXISTS idx_roles_org;
DROP TABLE IF EXISTS user_roles;
DROP TABLE IF EXISTS role_permissions;
DROP TABLE IF EXISTS permissions;
DROP TABLE IF EXISTS roles;
