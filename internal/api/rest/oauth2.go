package rest

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ihsansolusi/auth7/internal/domain"
	oauth2svc "github.com/ihsansolusi/auth7/internal/service/oauth2"
	"github.com/ihsansolusi/auth7/internal/service/session"
)

func (s *Server) RegisterOAuth2Routes(r *gin.Engine) {
	oauth := r.Group("/oauth2")
	{
		oauth.GET("/authorize", s.handleAuthorize)
		oauth.POST("/authorize-with-session", s.handleAuthorizeWithSession)
		oauth.POST("/token", s.handleToken)
		oauth.POST("/introspect", s.handleIntrospect)
		oauth.GET("/userinfo", s.handleUserInfo)
		oauth.POST("/register", s.handleDCR)
	}
}

// handleAuthorize — GET /oauth2/authorize
// Validates client and redirect_uri, then issues an authorization code.
// For E2E/CLI testing: accepts user_id via HTTP Basic Auth username (no password check).
func (s *Server) handleAuthorize(c *gin.Context) {
	clientID := c.Query("client_id")
	redirectURI := c.Query("redirect_uri")
	responseType := c.Query("response_type")
	scope := c.DefaultQuery("scope", "openid")
	state := c.Query("state")
	codeChallenge := c.Query("code_challenge")
	codeChallengeMethod := c.DefaultQuery("code_challenge_method", "S256")

	if responseType != "code" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported_response_type"})
		return
	}

	if clientID == "" || redirectURI == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}

	if s.deps.OAuth2ClientSvc == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
		return
	}

	// Validate client exists and redirect_uri is allowed
	client, err := s.deps.OAuth2ClientSvc.GetByClientID(c.Request.Context(), clientID)
	if err != nil {
		s.deps.Logger.Error().Err(err).Str("client_id", clientID).Msg("GetByClientID failed")
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_client"})
		return
	}

	if !client.IsActive {
		s.deps.Logger.Warn().Str("client_id", clientID).Msg("client is inactive")
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_client"})
		return
	}

	if !client.ValidateRedirectURI(redirectURI) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_redirect_uri"})
		return
	}

	// For E2E testing: accept user_id via HTTP Basic Auth username field.
	// In production this would validate a session cookie.
	userIDStr, _, basicOK := c.Request.BasicAuth()
	if !basicOK || userIDStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":             "login_required",
			"error_description": "Provide user_id as HTTP Basic Auth username for E2E testing",
		})
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "error_description": "invalid user_id format"})
		return
	}

	if s.deps.OAuth2AuthCodeSvc == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
		return
	}

	authCode, err := s.deps.OAuth2AuthCodeSvc.CreateAuthCode(c.Request.Context(), oauth2svc.AuthCodeParams{
		ClientID:            clientID,
		RedirectURI:         redirectURI,
		Scope:               scope,
		UserID:              userID,
		OrgID:               client.OrgID,
		CodeChallenge:       codeChallenge,
		CodeChallengeMethod: codeChallengeMethod,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
		return
	}

	location := redirectURI + "?code=" + authCode.Code
	if state != "" {
		location += "&state=" + state
	}
	c.Redirect(http.StatusFound, location)
}

// handleAuthorizeWithSession — POST /oauth2/authorize-with-session
// Called by auth7-ui after successful login. Validates Bearer token and issues auth code.
// Request body: { client_id, redirect_uri, response_type, scope, state, code_challenge, code_challenge_method }
// Response: { redirect_url: "http://client/callback?code=...&state=..." }
func (s *Server) handleAuthorizeWithSession(c *gin.Context) {
	// Extract and validate Bearer token
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_token", "error_description": "Bearer token required"})
		return
	}
	tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

	// Validate token via SessionSvc
	sessionSvc, ok := s.deps.SessionSvc.(*session.Service)
	if !ok || sessionSvc == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
		return
	}

	claims, err := sessionSvc.VerifyAccessToken(c.Request.Context(), tokenStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_token", "error_description": "Token tidak valid atau sudah expired"})
		return
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_token"})
		return
	}

	// Parse request body
	var req struct {
		ClientID            string `json:"client_id" binding:"required"`
		RedirectURI         string `json:"redirect_uri" binding:"required"`
		ResponseType        string `json:"response_type"`
		Scope               string `json:"scope"`
		State               string `json:"state"`
		CodeChallenge       string `json:"code_challenge"`
		CodeChallengeMethod string `json:"code_challenge_method"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "error_description": err.Error()})
		return
	}

	if req.ResponseType != "" && req.ResponseType != "code" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported_response_type"})
		return
	}

	if s.deps.OAuth2ClientSvc == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
		return
	}

	// Validate client and redirect_uri
	client, err := s.deps.OAuth2ClientSvc.GetByClientID(c.Request.Context(), req.ClientID)
	if err != nil {
		s.deps.Logger.Error().Err(err).Str("client_id", req.ClientID).Msg("handleAuthorizeWithSession: GetByClientID failed")
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_client"})
		return
	}

	if !client.IsActive {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_client", "error_description": "client is inactive"})
		return
	}

	if !client.ValidateRedirectURI(req.RedirectURI) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_redirect_uri"})
		return
	}

	scope := req.Scope
	if scope == "" {
		scope = "openid"
	}
	codeChallenge := req.CodeChallenge
	codeChallengeMethod := req.CodeChallengeMethod
	if codeChallengeMethod == "" {
		codeChallengeMethod = "S256"
	}

	if s.deps.OAuth2AuthCodeSvc == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
		return
	}

	authCode, err := s.deps.OAuth2AuthCodeSvc.CreateAuthCode(c.Request.Context(), oauth2svc.AuthCodeParams{
		ClientID:            req.ClientID,
		RedirectURI:         req.RedirectURI,
		Scope:               scope,
		UserID:              userID,
		Username:            claims.Username,
		Email:               claims.Email,
		OrgID:               client.OrgID,
		Roles:               claims.Roles,
		BranchID:            claims.BranchID,
		CodeChallenge:       codeChallenge,
		CodeChallengeMethod: codeChallengeMethod,
	})
	if err != nil {
		s.deps.Logger.Error().Err(err).Msg("handleAuthorizeWithSession: CreateAuthCode failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
		return
	}

	redirectURL := req.RedirectURI + "?code=" + authCode.Code
	if req.State != "" {
		redirectURL += "&state=" + req.State
	}

	c.JSON(http.StatusOK, gin.H{
		"redirect_url": redirectURL,
	})
}

func (s *Server) handleToken(c *gin.Context) {
	grantType := c.PostForm("grant_type")
	switch grantType {
	case "authorization_code":
		s.handleTokenExchange(c)
	case "refresh_token":
		s.handleTokenRefresh(c)
	case "client_credentials":
		s.handleClientCredentials(c)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported_grant_type"})
	}
}

// handleTokenExchange — POST /oauth2/token grant_type=authorization_code
func (s *Server) handleTokenExchange(c *gin.Context) {
	code := c.PostForm("code")
	codeVerifier := c.PostForm("code_verifier")
	redirectURI := c.PostForm("redirect_uri")
	clientID, clientSecret, ok := parseBasicAuth(c.GetHeader("Authorization"))
	if !ok {
		clientID = c.PostForm("client_id")
		clientSecret = c.PostForm("client_secret")
	}

	if code == "" || clientID == "" || redirectURI == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}

	if s.deps.OAuth2TokenSvc == nil || s.deps.OAuth2ClientSvc == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
		return
	}

	// Authenticate client
	client, err := s.deps.OAuth2ClientSvc.GetByClientID(c.Request.Context(), clientID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_client"})
		return
	}
	if client.IsConfidential() && !verifyClientSecret(clientSecret, client.ClientSecretHash) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_client"})
		return
	}

	resp, err := s.deps.OAuth2TokenSvc.ExchangeCodeForTokens(c.Request.Context(), code, codeVerifier, redirectURI)
	if err != nil {
		oe := mapOAuthError(err)
		c.JSON(oe.status, gin.H{"error": oe.code, "error_description": oe.desc})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (s *Server) handleTokenRefresh(c *gin.Context) {
	if c.PostForm("refresh_token") == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}
	c.JSON(http.StatusBadRequest, gin.H{
		"error":             "unsupported_grant_type",
		"error_description": "refresh_token grant not yet implemented",
	})
}

// handleClientCredentials — POST /oauth2/token grant_type=client_credentials
func (s *Server) handleClientCredentials(c *gin.Context) {
	clientID, clientSecret, ok := parseBasicAuth(c.GetHeader("Authorization"))
	if !ok {
		clientID = c.PostForm("client_id")
		clientSecret = c.PostForm("client_secret")
	}
	scope := c.PostForm("scope")

	if clientID == "" || clientSecret == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_client"})
		return
	}

	if s.deps.OAuth2ClientSvc == nil || s.deps.OAuth2TokenSvc == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
		return
	}

	client, err := s.deps.OAuth2ClientSvc.GetByClientID(c.Request.Context(), clientID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_client"})
		return
	}
	if !client.IsActive {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_client"})
		return
	}
	if !verifyClientSecret(clientSecret, client.ClientSecretHash) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_client"})
		return
	}

	resp, err := s.deps.OAuth2TokenSvc.ClientCredentials(c.Request.Context(), clientID, scope)
	if err != nil {
		oe := mapOAuthError(err)
		c.JSON(oe.status, gin.H{"error": oe.code, "error_description": oe.desc})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// handleIntrospect — POST /oauth2/introspect (RFC 7662)
func (s *Server) handleIntrospect(c *gin.Context) {
	token := c.PostForm("token")
	tokenTypeHint := c.PostForm("token_type_hint")

	if token == "" {
		// Also check Authorization header
		if auth := c.GetHeader("Authorization"); strings.HasPrefix(auth, "Bearer ") {
			token = auth[7:]
		}
	}

	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "error_description": "token is required"})
		return
	}

	_, _, ok := parseBasicAuth(c.GetHeader("Authorization"))
	if !ok {
		clientID := c.PostForm("client_id")
		clientSecret := c.PostForm("client_secret")
		if clientID == "" || clientSecret == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_client"})
			return
		}
		// Validate client
		if s.deps.OAuth2ClientSvc != nil {
			client, err := s.deps.OAuth2ClientSvc.GetByClientID(c.Request.Context(), clientID)
			if err != nil || !client.IsActive || !verifyClientSecret(clientSecret, client.ClientSecretHash) {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_client"})
				return
			}
		}
	}

	if s.deps.OAuth2TokenSvc == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
		return
	}

	resp, err := s.deps.OAuth2TokenSvc.IntrospectToken(c.Request.Context(), token)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
		return
	}

	c.JSON(http.StatusOK, resp)
	_ = tokenTypeHint
}

// handleUserInfo — GET /oauth2/userinfo
func (s *Server) handleUserInfo(c *gin.Context) {
	auth := c.GetHeader("Authorization")
	if auth == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_token"})
		return
	}
	tokenStr := strings.TrimPrefix(auth, "Bearer ")
	if tokenStr == auth {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_token"})
		return
	}

	// Check session blacklist before returning userinfo (SSO logout propagation)
	if sessionSvc, ok := s.deps.SessionSvc.(*session.Service); ok && sessionSvc != nil {
		if _, err := sessionSvc.VerifyAccessToken(c.Request.Context(), tokenStr); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_token"})
			return
		}
	}

	type jwtVerifier interface {
		VerifyAccessToken(string) (interface{}, error)
	}

	// Use OIDCService if available
	if s.deps.OIDCSvc != nil {
		info, err := s.deps.OIDCSvc.UserInfo(c.Request.Context(), tokenStr)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_token"})
			return
		}
		c.JSON(http.StatusOK, info)
		return
	}

	// Fallback: verify JWT directly via JWTSvc
	type verifier interface {
		VerifyAccessToken(string) (interface{}, error)
	}
	if _, ok := s.deps.JWTSvc.(verifier); !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
		return
	}

	// Use concrete jwt.Service type
	jwtSvc := s.deps.JWTSvc
	type jwtService interface {
		VerifyAccessToken(string) (*jwtClaimsResult, error)
	}
	_ = jwtSvc
	c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_token", "error_description": "OIDCSvc not configured"})
}

// handleDCR — POST /oauth2/register (RFC 7591)
func (s *Server) handleDCR(c *gin.Context) {
	var req DCRRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}

	if s.deps.OAuth2ClientSvc == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
		return
	}

	grantTypes := req.GrantTypes
	if len(grantTypes) == 0 {
		grantTypes = []string{"authorization_code"}
	}

	clientType := domain.ClientTypeWeb
	for _, g := range grantTypes {
		if g == string(domain.GrantTypeClientCredentials) {
			clientType = domain.ClientTypeMachine
			break
		}
	}

	authMethod := domain.TokenEndpointAuthMethod(req.TokenEndpointAuthMethod)
	if authMethod == "" {
		authMethod = domain.AuthMethodClientSecretBasic
	}

	// Determine org_id: from config or fallback to default system org
	orgID := uuid.Nil
	if s.deps.Config != nil && s.deps.Config.Service.DefaultOrgID != "" {
		if parsed, err := uuid.Parse(s.deps.Config.Service.DefaultOrgID); err == nil {
			orgID = parsed
		}
	}
	if orgID == uuid.Nil {
		orgID = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	}

	// Generate plain secret to return to caller, then hash for storage
	plainSecret := dcrGenerateSecret()
	secretHash := hashClientSecret(plainSecret)

	clientName := req.ClientName
	if clientName == "" {
		clientName = "DCR Client " + uuid.New().String()[:8]
	}

	client, err := s.deps.OAuth2ClientSvc.CreateWithSecretHash(
		c.Request.Context(),
		orgID,
		oauth2svc.CreateClientParams{
			Name:                   clientName,
			ClientType:             clientType,
			TokenEndpointAuthMethod: authMethod,
			AllowedScopes:          strings.Fields(req.Scope),
			AllowedRedirectURIs:    req.RedirectURIs,
			TokenExpiration:        900,
			RefreshTokenExpiration: 28800,
		},
		secretHash,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error", "error_description": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, DCRResponse{
		ClientID:                client.ID.String(),
		ClientSecret:            plainSecret,
		ClientIDIssuedAt:        client.CreatedAt.Unix(),
		ClientSecretExpiresAt:   0,
		RedirectURIs:            client.AllowedRedirectURIs,
		GrantTypes:              grantTypes,
		ResponseTypes:           []string{"code"},
		Scope:                   req.Scope,
		TokenEndpointAuthMethod: string(client.TokenEndpointAuthMethod),
	})
}

// --------------------------------------------------------------------------
// Helpers
// --------------------------------------------------------------------------

func parseBasicAuth(auth string) (clientID, clientSecret string, ok bool) {
	if auth == "" {
		return "", "", false
	}
	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 || parts[0] != "Basic" {
		return "", "", false
	}
	payload, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return "", "", false
	}
	creds := strings.SplitN(string(payload), ":", 2)
	if len(creds) != 2 {
		return "", "", false
	}
	return creds[0], creds[1], true
}

// verifyClientSecret checks plain secret against SHA-256 hash stored in DB.
// Supports both hex-encoded (seed-data.sql style) and base64-encoded hashes.
func verifyClientSecret(plainSecret, storedHash string) bool {
	if storedHash == "" {
		return false
	}
	h := sha256.Sum256([]byte(plainSecret))
	// Try hex first (used in seed-data.sql)
	if hex.EncodeToString(h[:]) == storedHash {
		return true
	}
	// Fallback: base64 (used by DCR-generated clients)
	return base64.StdEncoding.EncodeToString(h[:]) == storedHash
}

func hashClientSecret(secret string) string {
	h := sha256.Sum256([]byte(secret))
	return base64.StdEncoding.EncodeToString(h[:])
}

func dcrGenerateSecret() string {
	b := make([]byte, 32)
	rand.Read(b) //nolint:errcheck
	return base64.URLEncoding.EncodeToString(b)
}

type oauthErrResult struct {
	status int
	code   string
	desc   string
}

func mapOAuthError(err error) oauthErrResult {
	switch {
	case errors.Is(err, oauth2svc.ErrInvalidClient):
		return oauthErrResult{http.StatusUnauthorized, "invalid_client", err.Error()}
	case errors.Is(err, oauth2svc.ErrInvalidGrant):
		return oauthErrResult{http.StatusBadRequest, "invalid_grant", err.Error()}
	case errors.Is(err, oauth2svc.ErrCodeAlreadyUsed):
		return oauthErrResult{http.StatusBadRequest, "invalid_grant", "authorization code already used"}
	case errors.Is(err, oauth2svc.ErrCodeExpired):
		return oauthErrResult{http.StatusBadRequest, "invalid_grant", "authorization code expired"}
	case errors.Is(err, oauth2svc.ErrInvalidCodeVerifier):
		return oauthErrResult{http.StatusBadRequest, "invalid_grant", "PKCE code verifier mismatch"}
	case errors.Is(err, oauth2svc.ErrInvalidRedirectURI):
		return oauthErrResult{http.StatusBadRequest, "invalid_grant", "redirect_uri mismatch"}
	case errors.Is(err, oauth2svc.ErrUnauthorizedClient):
		return oauthErrResult{http.StatusUnauthorized, "unauthorized_client", err.Error()}
	default:
		return oauthErrResult{http.StatusInternalServerError, "server_error", err.Error()}
	}
}

// jwtClaimsResult is used as a placeholder type (not actually used at runtime).
type jwtClaimsResult struct{}

type DCRRequest struct {
	ClientName              string   `json:"client_name"`
	RedirectURIs            []string `json:"redirect_uris"`
	GrantTypes              []string `json:"grant_types"`
	Scope                   string   `json:"scope"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
}

type DCRResponse struct {
	ClientID                string   `json:"client_id"`
	ClientSecret            string   `json:"client_secret"`
	ClientIDIssuedAt        int64    `json:"client_id_issued_at"`
	ClientSecretExpiresAt   int64    `json:"client_secret_expires_at"`
	RedirectURIs            []string `json:"redirect_uris"`
	GrantTypes              []string `json:"grant_types"`
	ResponseTypes           []string `json:"response_types"`
	Scope                   string   `json:"scope"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
}

// unused — satisfies import
var _ = time.Now
