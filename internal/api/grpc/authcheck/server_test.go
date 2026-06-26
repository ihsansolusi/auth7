package authcheck

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/ihsansolusi/auth7/internal/domain"
	"github.com/ihsansolusi/auth7/internal/service/authz"
	"github.com/rs/zerolog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeUserRoles struct{ roles []*domain.UserRole }

func (f fakeUserRoles) GetUserRoles(_ interface{}, _ uuid.UUID) ([]*domain.UserRole, error) {
	return f.roles, nil
}

type fakeRolePerms struct{ byRole map[uuid.UUID][]*domain.Permission }

func (f fakeRolePerms) GetPermissions(_ interface{}, roleID uuid.UUID) ([]*domain.Permission, error) {
	return f.byRole[roleID], nil
}

func newServer(roles []*domain.UserRole, perms map[uuid.UUID][]*domain.Permission) *Server {
	return NewServer(
		fakeUserRoles{roles: roles},
		fakeRolePerms{byRole: perms},
		authz.NewTimeGatedChecker(nil, nil), // no time-gate → role-based decision
		zerolog.Nop(),
	)
}

func TestGRPCCheckPermission(t *testing.T) {
	org, branch, user, roleID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	roles := []*domain.UserRole{{RoleID: roleID, OrgID: org, BranchID: nil}}
	perms := map[uuid.UUID][]*domain.Permission{roleID: {{Code: "report:view"}}}
	srv := newServer(roles, perms)

	base := func(perm string) *CheckPermissionRequest {
		return &CheckPermissionRequest{UserID: user.String(), OrgID: org.String(), BranchID: branch.String(), Permission: perm}
	}

	t.Run("granted → allowed", func(t *testing.T) {
		resp, err := srv.CheckPermission(context.Background(), base("report:view"))
		if err != nil || !resp.Allowed {
			t.Fatalf("expected allowed, got allowed=%v err=%v", resp, err)
		}
	})

	t.Run("ungranted → denied", func(t *testing.T) {
		resp, err := srv.CheckPermission(context.Background(), base("transaction:create"))
		if err != nil || resp.Allowed {
			t.Fatalf("expected denied, got allowed=%v err=%v", resp, err)
		}
	})

	t.Run("invalid user_id → InvalidArgument", func(t *testing.T) {
		_, err := srv.CheckPermission(context.Background(), &CheckPermissionRequest{UserID: "nope", OrgID: org.String(), BranchID: branch.String(), Permission: "x"})
		if status.Code(err) != codes.InvalidArgument {
			t.Fatalf("expected InvalidArgument, got %v", err)
		}
	})
}
