package grpc

import (
	"context"

	"github.com/google/uuid"
	"github.com/ihsansolusi/auth7/internal/service/jwt"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ServerDeps struct {
	JWTSvc interface {
		IssueAccessToken(sessionID string, userID, orgID uuid.UUID, claims jwt.Claims) (string, *jwt.AccessToken, error)
		VerifyAccessToken(tokenString string) (*jwt.Claims, error)
		GetJWKS() []map[string]interface{}
	}
	SessionSvc any
	Logger     zerolog.Logger
	Tracer     any
}

type AuthServiceServer interface {
	VerifyToken(ctx context.Context, req *VerifyTokenRequest) (*VerifyTokenResponse, error)
	CheckPermission(ctx context.Context, req *CheckPermissionRequest) (*CheckPermissionResponse, error)
	GetUserInfo(ctx context.Context, req *GetUserInfoRequest) (*GetUserInfoResponse, error)
}

type TokenServiceServer interface {
	IssueToken(ctx context.Context, req *IssueTokenRequest) (*IssueTokenResponse, error)
	RefreshToken(ctx context.Context, req *RefreshTokenRequest) (*RefreshTokenResponse, error)
	RevokeToken(ctx context.Context, req *RevokeTokenRequest) (*RevokeTokenResponse, error)
}

type authServer struct {
	deps ServerDeps
}

type tokenServer struct {
	deps ServerDeps
}

type VerifyTokenRequest struct {
	Token            string
	ExpectedAudience string
}

type VerifyTokenResponse struct {
	Valid     bool
	UserID    string
	OrgID     string
	ClientID  string
	Roles     []string
	Scope     string
	ExpiresAt int64
	Error     string
}

type CheckPermissionRequest struct {
	UserID      string
	Permission string
	BranchID   string
	OrgID      string
}

type CheckPermissionResponse struct {
	Allowed bool
	Reason  string
}

type GetUserInfoRequest struct {
	UserID string
	OrgID  string
}

type GetUserInfoResponse struct {
	UserID        string
	Username      string
	Email         string
	FullName      string
	OrgID         string
	Roles         []string
	EmailVerified bool
	MfaEnabled    bool
	Status        string
}

type IssueTokenRequest struct {
	UserID    string
	ClientID  string
	Roles     []string
	Scope     string
	ExpiresIn int64
	OrgID     string
}

type IssueTokenResponse struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64
	TokenType    string
	Scope        string
}

type RefreshTokenRequest struct {
	RefreshToken string
}

type RefreshTokenResponse struct {
	AccessToken string
	ExpiresIn   int64
	TokenType   string
	Scope       string
}

type RevokeTokenRequest struct {
	Token        string
	TokenTypeHint string
}

type RevokeTokenResponse struct{}

func (s *authServer) VerifyToken(ctx context.Context, req *VerifyTokenRequest) (*VerifyTokenResponse, error) {
	const op = "grpc.authServer.VerifyToken"

	ctx, span := otel.GetTracerProvider().Tracer("auth7").Start(ctx, op)
	defer span.End()

	claims, err := s.deps.JWTSvc.VerifyAccessToken(req.Token)
	if err != nil {
		return &VerifyTokenResponse{
			Valid: false,
			Error: err.Error(),
		}, nil
	}

	userID, _ := uuid.Parse(claims.Subject)
	orgID, _ := uuid.Parse(claims.OrgID)

	return &VerifyTokenResponse{
		Valid:     true,
		UserID:    userID.String(),
		OrgID:     orgID.String(),
		ClientID:  claims.ClientID,
		Roles:     claims.Roles,
		Scope:     claims.Scope,
		ExpiresAt: claims.ExpiresAt.Time.Unix(),
	}, nil
}

func (s *authServer) CheckPermission(ctx context.Context, req *CheckPermissionRequest) (*CheckPermissionResponse, error) {
	const op = "grpc.authServer.CheckPermission"

	ctx, span := otel.GetTracerProvider().Tracer("auth7").Start(ctx, op)
	defer span.End()

	return &CheckPermissionResponse{
		Allowed: true,
		Reason:  "permission check not yet implemented",
	}, nil
}

func (s *authServer) GetUserInfo(ctx context.Context, req *GetUserInfoRequest) (*GetUserInfoResponse, error) {
	const op = "grpc.authServer.GetUserInfo"

	ctx, span := otel.GetTracerProvider().Tracer("auth7").Start(ctx, op)
	defer span.End()

	return &GetUserInfoResponse{
		UserID:        req.UserID,
		Username:      "user",
		Email:         "user@example.com",
		FullName:      "User",
		OrgID:         req.OrgID,
		Roles:         []string{},
		EmailVerified: true,
		MfaEnabled:    false,
		Status:        "active",
	}, nil
}

func (s *tokenServer) IssueToken(ctx context.Context, req *IssueTokenRequest) (*IssueTokenResponse, error) {
	const op = "grpc.tokenServer.IssueToken"

	ctx, span := otel.GetTracerProvider().Tracer("auth7").Start(ctx, op)
	defer span.End()

	userID, _ := uuid.Parse(req.UserID)
	orgID, _ := uuid.Parse(req.OrgID)

	claims := jwt.Claims{
		ClientID: req.ClientID,
		Roles:   req.Roles,
		Scope:   req.Scope,
	}

	sessionID := uuid.New().String()
	accessToken, _, err := s.deps.JWTSvc.IssueAccessToken(sessionID, userID, orgID, claims)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%s: %v", op, err)
	}

	expiresIn := int64(900)
	if req.ExpiresIn > 0 {
		expiresIn = req.ExpiresIn
	}

	return &IssueTokenResponse{
		AccessToken:  accessToken,
		RefreshToken: "",
		ExpiresIn:    expiresIn,
		TokenType:    "Bearer",
		Scope:        req.Scope,
	}, nil
}

func (s *tokenServer) RefreshToken(ctx context.Context, req *RefreshTokenRequest) (*RefreshTokenResponse, error) {
	const op = "grpc.tokenServer.RefreshToken"

	ctx, span := otel.GetTracerProvider().Tracer("auth7").Start(ctx, op)
	defer span.End()

	return &RefreshTokenResponse{
		AccessToken: "",
		ExpiresIn:   900,
		TokenType:   "Bearer",
		Scope:       "",
	}, nil
}

func (s *tokenServer) RevokeToken(ctx context.Context, req *RevokeTokenRequest) (*RevokeTokenResponse, error) {
	const op = "grpc.tokenServer.RevokeToken"

	ctx, span := otel.GetTracerProvider().Tracer("auth7").Start(ctx, op)
	defer span.End()

	return &RevokeTokenResponse{}, nil
}

func NewServer(deps ServerDeps) *grpc.Server {
	return grpc.NewServer()
}

func RegisterServices(srv *grpc.Server, auth AuthServiceServer, token TokenServiceServer) {
}

type GRPCServer struct {
	server *grpc.Server
	auth   AuthServiceServer
	token  TokenServiceServer
}

func (s *GRPCServer) Register(srv *grpc.Server) {
}

func (s *GRPCServer) AuthService() AuthServiceServer {
	return s.auth
}

func (s *GRPCServer) TokenService() TokenServiceServer {
	return s.token
}

type AuthServiceRegistry struct {
	auth  AuthServiceServer
	token TokenServiceServer
}

func NewAuthServiceRegistry(auth AuthServiceServer, token TokenServiceServer) *AuthServiceRegistry {
	return &AuthServiceRegistry{
		auth:  auth,
		token: token,
	}
}

func (r *AuthServiceRegistry) GetAuthService() AuthServiceServer {
	return r.auth
}

func (r *AuthServiceRegistry) GetTokenService() TokenServiceServer {
	return r.token
}
