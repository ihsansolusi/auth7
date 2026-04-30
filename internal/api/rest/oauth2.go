package rest

import (
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (s *Server) RegisterOAuth2Routes(r *gin.Engine) {
	oauth := r.Group("/oauth2")
	{
		oauth.GET("/authorize", s.handleAuthorize)
		oauth.POST("/token", s.handleToken)
		oauth.GET("/userinfo", s.handleUserInfo)
		oauth.POST("/register", s.handleDCR)
	}
}

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

	c.JSON(http.StatusOK, gin.H{
		"client_id":              clientID,
		"redirect_uri":           redirectURI,
		"scope":                  scope,
		"state":                  state,
		"code_challenge":         codeChallenge,
		"code_challenge_method":  codeChallengeMethod,
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

func (s *Server) handleTokenExchange(c *gin.Context) {
	code := c.PostForm("code")
	_ = c.PostForm("code_verifier")
	_ = c.PostForm("redirect_uri")
	clientID, _, ok := parseBasicAuth(c.GetHeader("Authorization"))

	if !ok {
		clientID = c.PostForm("client_id")
		_ = c.PostForm("client_secret")
	}

	if code == "" || clientID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  "placeholder",
		"token_type":   "Bearer",
		"expires_in":   900,
		"refresh_token": "placeholder",
		"scope":        "openid profile",
	})
}

func (s *Server) handleTokenRefresh(c *gin.Context) {
	refreshToken := c.PostForm("refresh_token")

	if refreshToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  "placeholder",
		"token_type":    "Bearer",
		"expires_in":    900,
		"refresh_token": "placeholder",
		"scope":         "openid profile",
	})
}

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

	c.JSON(http.StatusOK, gin.H{
		"access_token":  "placeholder",
		"token_type":    "Bearer",
		"expires_in":    3600,
		"scope":          scope,
	})
}

func (s *Server) handleUserInfo(c *gin.Context) {
	auth := c.GetHeader("Authorization")
	if auth == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_token"})
		return
	}

	token := strings.TrimPrefix(auth, "Bearer ")
	if token == auth {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"sub":   "user-uuid",
		"name":  "John Doe",
		"email": "john@example.com",
		"scope": "openid profile",
	})
}

func (s *Server) handleDCR(c *gin.Context) {
	var req DCRRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}

	clientID := uuid.New().String()
	clientSecret := uuid.New().String()

	c.JSON(http.StatusCreated, DCRResponse{
		ClientID:              clientID,
		ClientSecret:          clientSecret,
		ClientIDIssuedAt:      1713798000,
		ClientSecretExpiresAt: 0,
		RedirectURIs:          req.RedirectURIs,
		GrantTypes:            req.GrantTypes,
		ResponseTypes:         []string{"code"},
		Scope:                 req.Scope,
		TokenEndpointAuthMethod: req.TokenEndpointAuthMethod,
	})
}

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

type DCRRequest struct {
	RedirectURIs          []string `json:"redirect_uris"`
	GrantTypes           []string `json:"grant_types"`
	Scope                string   `json:"scope"`
	TokenEndpointAuthMethod string `json:"token_endpoint_auth_method"`
}

type DCRResponse struct {
	ClientID              string   `json:"client_id"`
	ClientSecret          string   `json:"client_secret"`
	ClientIDIssuedAt      int64    `json:"client_id_issued_at"`
	ClientSecretExpiresAt int64    `json:"client_secret_expires_at"`
	RedirectURIs          []string `json:"redirect_uris"`
	GrantTypes            []string `json:"grant_types"`
	ResponseTypes         []string `json:"response_types"`
	Scope                 string   `json:"scope"`
	TokenEndpointAuthMethod string `json:"token_endpoint_auth_method"`
}