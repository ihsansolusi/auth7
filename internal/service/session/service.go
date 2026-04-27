package session

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ihsansolusi/auth7/internal/service/jwt"
)

type Service struct {
	sessionStore      *Store
	refreshTokenStore *RefreshTokenStore
	blacklistStore    *BlacklistStore
	jwtService       *jwt.Service
	defaultTTL        time.Duration
}

func NewService(
	sessionStore *Store,
	refreshTokenStore *RefreshTokenStore,
	blacklistStore *BlacklistStore,
	jwtService *jwt.Service,
	defaultTTL time.Duration,
) *Service {
	return &Service{
		sessionStore:      sessionStore,
		refreshTokenStore: refreshTokenStore,
		blacklistStore:    blacklistStore,
		jwtService:       jwtService,
		defaultTTL:        defaultTTL,
	}
}

type LoginResult struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int
	SessionID    string
	UserID       uuid.UUID
	OrgID        uuid.UUID
}

func (s *Service) CreateSession(ctx context.Context, userID, orgID uuid.UUID, ipAddress, userAgent string, claims jwt.Claims) (*LoginResult, error) {
	const op = "session.Service.CreateSession"

	sessionID := uuid.New().String()

	sessionData := &SessionData{
		ID:         sessionID,
		UserID:     userID.String(),
		OrgID:      orgID.String(),
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
		Roles:      claims.Roles,
		MFAVerified: true,
	}

	if err := s.sessionStore.Create(ctx, sessionData); err != nil {
		return nil, fmt.Errorf("%s: create session: %w", op, err)
	}

	accessToken, _, err := s.jwtService.IssueAccessToken(sessionID, userID, orgID, claims)
	if err != nil {
		return nil, fmt.Errorf("%s: issue access token: %w", op, err)
	}

	familyID := uuid.New().String()
	refreshToken := jwt.GenerateRefreshToken()
	refreshTokenHash := jwt.HashToken(refreshToken)

	if err := s.refreshTokenStore.Create(ctx, &RefreshTokenData{
		TokenID:   uuid.New().String(),
		FamilyID:  familyID,
		TokenHash: refreshTokenHash,
		UserID:    userID.String(),
		OrgID:     orgID.String(),
		ClientID:  claims.ClientID,
		SessionID: sessionID,
		Scopes:    claims.Roles,
		IssuedAt:  time.Now().Unix(),
		ExpiresAt: time.Now().Add(8 * time.Hour).Unix(),
	}); err != nil {
		return nil, fmt.Errorf("%s: store refresh token: %w", op, err)
	}

	return &LoginResult{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:   900,
		SessionID:    sessionID,
		UserID:       userID,
		OrgID:        orgID,
	}, nil
}

func (s *Service) ValidateSession(ctx context.Context, sessionID, ipAddress string) (*SessionData, error) {
	const op = "session.Service.ValidateSession"

	session, err := s.sessionStore.Get(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("%s: get session: %w", op, err)
	}

	if session == nil {
		return nil, nil
	}

	if session.IPAddress != ipAddress {
		s.sessionStore.Delete(ctx, sessionID)
		return nil, fmt.Errorf("%s: IP address mismatch", op)
	}

	s.sessionStore.UpdateLastUsed(ctx, sessionID)

	return session, nil
}

func (s *Service) RefreshTokens(ctx context.Context, refreshToken, ipAddress string) (*LoginResult, error) {
	const op = "session.Service.RefreshTokens"

	tokenHash := jwt.HashToken(refreshToken)

	tokenData, err := s.refreshTokenStore.GetByHash(ctx, tokenHash)
	if err != nil {
		return nil, fmt.Errorf("%s: get token: %w", op, err)
	}

	if tokenData == nil {
		return nil, fmt.Errorf("%s: token not found", op)
	}

	if tokenData.RevokedAt != nil {
		s.refreshTokenStore.RevokeFamily(ctx, tokenData.FamilyID)
		return nil, fmt.Errorf("%s: token revoked", op)
	}

	isReuse, err := s.refreshTokenStore.IsReuse(ctx, tokenHash)
	if err != nil {
		return nil, fmt.Errorf("%s: check reuse: %w", op, err)
	}

	if isReuse {
		s.refreshTokenStore.RevokeFamily(ctx, tokenData.FamilyID)
		s.sessionStore.Delete(ctx, tokenData.SessionID)
		return nil, fmt.Errorf("%s: token reuse detected", op)
	}

	userID, _ := uuid.Parse(tokenData.UserID)
	orgID, _ := uuid.Parse(tokenData.OrgID)

	session, err := s.sessionStore.Get(ctx, tokenData.SessionID)
	if err != nil || session == nil {
		return nil, fmt.Errorf("%s: session not found", op)
	}

	if session.IPAddress != ipAddress {
		s.sessionStore.Delete(ctx, tokenData.SessionID)
		return nil, fmt.Errorf("%s: IP address mismatch", op)
	}

	s.refreshTokenStore.MarkUsed(ctx, tokenHash)

	newSessionID := uuid.New().String()
	session.ID = newSessionID
	session.IPAddress = ipAddress

	if err := s.sessionStore.Create(ctx, session); err != nil {
		return nil, fmt.Errorf("%s: create new session: %w", op, err)
	}

	claims := jwt.Claims{
		ClientID: tokenData.ClientID,
		Roles:    tokenData.Scopes,
	}

	accessToken, _, err := s.jwtService.IssueAccessToken(newSessionID, userID, orgID, claims)
	if err != nil {
		return nil, fmt.Errorf("%s: issue access token: %w", op, err)
	}

	newFamilyID := uuid.New().String()
	newRefreshToken := jwt.GenerateRefreshToken()
	newRefreshTokenHash := jwt.HashToken(newRefreshToken)

	if err := s.refreshTokenStore.Create(ctx, &RefreshTokenData{
		TokenID:   uuid.New().String(),
		FamilyID:  newFamilyID,
		TokenHash: newRefreshTokenHash,
		UserID:    userID.String(),
		OrgID:     orgID.String(),
		ClientID:  tokenData.ClientID,
		SessionID: newSessionID,
		Scopes:    tokenData.Scopes,
		IssuedAt:  time.Now().Unix(),
		ExpiresAt: time.Now().Add(8 * time.Hour).Unix(),
	}); err != nil {
		return nil, fmt.Errorf("%s: store refresh token: %w", op, err)
	}

	return &LoginResult{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		ExpiresIn:   900,
		SessionID:    newSessionID,
		UserID:       userID,
		OrgID:        orgID,
	}, nil
}

func (s *Service) RevokeSession(ctx context.Context, sessionID string) error {
	const op = "session.Service.RevokeSession"

	if err := s.sessionStore.Delete(ctx, sessionID); err != nil {
		return fmt.Errorf("%s: delete session: %w", op, err)
	}

	s.blacklistStore.AddSession(ctx, sessionID, 8*time.Hour)

	return nil
}

func (s *Service) RevokeAllUserSessions(ctx context.Context, userID string) error {
	const op = "session.Service.RevokeAllUserSessions"

	if err := s.sessionStore.DeleteByUser(ctx, userID); err != nil {
		return fmt.Errorf("%s: delete user sessions: %w", op, err)
	}

	return nil
}

func (s *Service) RevokeAllOrgSessions(ctx context.Context, orgID string) error {
	const op = "session.Service.RevokeAllOrgSessions"

	if err := s.sessionStore.DeleteByOrg(ctx, orgID); err != nil {
		return fmt.Errorf("%s: delete org sessions: %w", op, err)
	}

	return nil
}

func (s *Service) GetSession(ctx context.Context, sessionID string) (*SessionData, error) {
	return s.sessionStore.Get(ctx, sessionID)
}

func (s *Service) VerifyAccessToken(ctx context.Context, accessToken string) (*jwt.Claims, error) {
	claims, err := s.jwtService.VerifyAccessToken(accessToken)
	if err != nil {
		return nil, err
	}

	revoked, err := s.blacklistStore.IsSessionRevoked(ctx, claims.SessionID)
	if err != nil {
		return nil, err
	}

	if revoked {
		return nil, fmt.Errorf("session revoked")
	}

	return claims, nil
}

func joinScopes(scopes []string) string {
	return strings.Join(scopes, " ")
}
