package authz

import "errors"

var (
	ErrRoleExists        = errors.New("role already exists")
	ErrRoleNotFound      = errors.New("role not found")
	ErrPermissionExists   = errors.New("permission already exists")
	ErrPermissionNotFound = errors.New("permission not found")
	ErrUserRoleNotFound  = errors.New("user role not found")
	ErrPolicyNotFound    = errors.New("policy not found")
	ErrAccessDenied      = errors.New("access denied")
)