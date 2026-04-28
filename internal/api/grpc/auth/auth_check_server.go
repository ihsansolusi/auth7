package auth

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/ihsansolusi/auth7/internal/domain"
	"github.com/ihsansolusi/auth7/internal/service/authz"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AuthCheckServer struct {
	checker *authz.PermissionChecker
}

func NewAuthCheckServer(checker *authz.PermissionChecker) *AuthCheckServer {
	return &AuthCheckServer{
		checker: checker,
	}
}

type CheckPermissionRequest struct {
	UserID      string
	Permission string
	BranchID   string
	OrgID      string
}

type CheckPermissionResponse struct {
	Allowed   bool
	Reason    string
	FieldMasks []domain.FieldMask
}

type GetUserPermissionsRequest struct {
	UserID   string
	OrgID    string
	BranchID string
}

type GetUserPermissionsResponse struct {
	Permissions []string
	BranchScope string
}

type CheckDataAccessRequest struct {
	UserID       string
	OrgID        string
	BranchID     string
	Permission   string
	ResourceType string
	ResourceId   string
}

type CheckDataAccessResponse struct {
	Allowed     bool
	Reason      string
	BranchScope string
	FieldMasks  []domain.FieldMask
}

func (s *AuthCheckServer) CheckPermission(ctx context.Context, req *CheckPermissionRequest) (*CheckPermissionResponse, error) {
	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user_id")
	}

	orgID, err := uuid.Parse(req.OrgID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid org_id")
	}

	branchID, err := uuid.Parse(req.BranchID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid branch_id")
	}

	authCtx := &domain.AuthContext{
		UserID:      userID,
		OrgID:       orgID,
		BranchID:    branchID,
		BranchScope: domain.BranchScopeAssigned,
	}

	result, err := s.checker.CheckPermission(ctx, authCtx, req.Permission)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "check permission failed: %v", err)
	}

	return &CheckPermissionResponse{
		Allowed:   result.Allowed,
		Reason:    result.Reason,
		FieldMasks: result.FieldMasks,
	}, nil
}

func (s *AuthCheckServer) GetUserPermissions(ctx context.Context, req *GetUserPermissionsRequest) (*GetUserPermissionsResponse, error) {
	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user_id")
	}

	orgID, err := uuid.Parse(req.OrgID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid org_id")
	}

	branchID, err := uuid.Parse(req.BranchID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid branch_id")
	}

	perms, err := s.getUserPermissions(ctx, userID, orgID, branchID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get permissions failed: %v", err)
	}

	return &GetUserPermissionsResponse{
		Permissions: perms,
		BranchScope: string(domain.BranchScopeAssigned),
	}, nil
}

func (s *AuthCheckServer) CheckDataAccess(ctx context.Context, req *CheckDataAccessRequest) (*CheckDataAccessResponse, error) {
	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user_id")
	}

	orgID, err := uuid.Parse(req.OrgID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid org_id")
	}

	branchID, err := uuid.Parse(req.BranchID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid branch_id")
	}

	resourceID, err := uuid.Parse(req.ResourceId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid resource_id")
	}

	authCtx := &domain.AuthContext{
		UserID:      userID,
		OrgID:       orgID,
		BranchID:    branchID,
		BranchScope: domain.BranchScopeAssigned,
	}

	result, err := s.checker.CheckDataAccess(ctx, authCtx, req.Permission, req.ResourceType, resourceID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "check data access failed: %v", err)
	}

	return &CheckDataAccessResponse{
		Allowed:     result.Allowed,
		Reason:      result.Reason,
		BranchScope: string(authCtx.BranchScope),
		FieldMasks:  result.FieldMasks,
	}, nil
}

func (s *AuthCheckServer) getUserPermissions(ctx context.Context, userID, orgID, branchID uuid.UUID) ([]string, error) {
	return nil, fmt.Errorf("not implemented")
}
