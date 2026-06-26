// Package authcheck serves auth7's AuthCheckService over gRPC using a JSON codec
// (no protobuf codegen), matching the lib7-service-go/auth7grpc client contract.
// It is the gRPC transport for the PDP; the REST transport
// (internal/api/rest internal_authz.go) shares the same decision core
// (authz.PermissionChecker + authz.ResolveAuthContext).
package authcheck

import (
	"context"

	"github.com/google/uuid"
	"github.com/ihsansolusi/auth7/internal/service/authz"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Wire types — JSON shape MUST match lib7-service-go/auth7grpc.
type CheckPermissionRequest struct {
	UserID     string `json:"user_id"`
	Permission string `json:"permission"`
	BranchID   string `json:"branch_id"`
	OrgID      string `json:"org_id"`
}

type CheckPermissionResponse struct {
	Allowed    bool        `json:"allowed"`
	Reason     string      `json:"reason"`
	FieldMasks []FieldMask `json:"field_masks"`
}

type FieldMask struct {
	Field     string `json:"field"`
	MaskValue string `json:"mask_value"`
	Reason    string `json:"reason"`
}

// Server implements auth.v1.AuthCheckService.CheckPermission. It resolves the
// user's effective permissions from auth7's own role data, then runs the shared
// PermissionChecker (role-based + operational-hours time-gate).
type Server struct {
	userRoleSvc authz.UserRolesGetter
	roleSvc     authz.RolePermsGetter
	checker     *authz.PermissionChecker
	logger      zerolog.Logger
}

func NewServer(userRoleSvc authz.UserRolesGetter, roleSvc authz.RolePermsGetter, checker *authz.PermissionChecker, logger zerolog.Logger) *Server {
	return &Server{userRoleSvc: userRoleSvc, roleSvc: roleSvc, checker: checker, logger: logger}
}

func (s *Server) CheckPermission(ctx context.Context, req *CheckPermissionRequest) (*CheckPermissionResponse, error) {
	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}
	orgID, err := uuid.Parse(req.OrgID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid org_id")
	}
	branchID, err := uuid.Parse(req.BranchID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid branch_id")
	}
	if req.Permission == "" {
		return nil, status.Error(codes.InvalidArgument, "permission required")
	}

	authCtx, err := authz.ResolveAuthContext(ctx, s.userRoleSvc, s.roleSvc, userID, orgID, branchID)
	if err != nil {
		s.logger.Error().Err(err).Str("user", req.UserID).Msg("authcheck resolve auth context failed")
		return nil, status.Error(codes.Internal, "resolve auth context failed")
	}

	result, err := s.checker.CheckPermission(ctx, authCtx, req.Permission)
	if err != nil {
		s.logger.Error().Err(err).Str("permission", req.Permission).Msg("authcheck check permission failed")
		return nil, status.Error(codes.Internal, "check permission failed")
	}

	masks := make([]FieldMask, 0, len(result.FieldMasks))
	for _, m := range result.FieldMasks {
		masks = append(masks, FieldMask{Field: m.Field, MaskValue: m.MaskValue, Reason: m.Reason})
	}
	return &CheckPermissionResponse{Allowed: result.Allowed, Reason: result.Reason, FieldMasks: masks}, nil
}

// ── manual gRPC service registration (no protobuf codegen) ──────────────────

type authCheckService interface {
	CheckPermission(context.Context, *CheckPermissionRequest) (*CheckPermissionResponse, error)
}

func checkPermissionHandler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CheckPermissionRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(authCheckService).CheckPermission(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/auth.v1.AuthCheckService/CheckPermission"}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(authCheckService).CheckPermission(ctx, req.(*CheckPermissionRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// serviceDesc matches the method path lib7-service-go/auth7grpc invokes:
// "/auth.v1.AuthCheckService/CheckPermission".
var serviceDesc = grpc.ServiceDesc{
	ServiceName: "auth.v1.AuthCheckService",
	HandlerType: (*authCheckService)(nil),
	Methods: []grpc.MethodDesc{
		{MethodName: "CheckPermission", Handler: checkPermissionHandler},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "auth/v1/auth.proto",
}

// Register registers the AuthCheckService on the given gRPC server.
func Register(s *grpc.Server, impl *Server) {
	s.RegisterService(&serviceDesc, impl)
}
