package branch

import "errors"

var (
	ErrInvalidBranchType    = errors.New("invalid branch type parameters")
	ErrBranchTypeExists     = errors.New("branch type already exists")
	ErrInvalidBranch       = errors.New("invalid branch parameters")
	ErrBranchExists        = errors.New("branch already exists")
	ErrBranchNotActive     = errors.New("branch is not active")
	ErrCannotHaveChildren   = errors.New("branch type cannot have children")
	ErrUserCannotAccessBranch = errors.New("user does not have access to this branch")
)