package rest

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ihsansolusi/auth7/internal/domain"
	"github.com/ihsansolusi/auth7/internal/service/jwt"
	"github.com/ihsansolusi/auth7/internal/service/password"
	"github.com/ihsansolusi/auth7/internal/service/session"
	"github.com/ihsansolusi/auth7/internal/store/postgres"
	"github.com/ihsansolusi/lib7-service-go/token"
)

type AuthHandler struct {
	store      *postgres.Store
	hasher     *password.Hasher
	sessionSvc *session.Service
	tokenMaker token.Maker
}

func NewAuthHandler(store *postgres.Store, hasher *password.Hasher, sessionSvc *session.Service, tokenMaker token.Maker) *AuthHandler {
	return &AuthHandler{
		store:      store,
		hasher:     hasher,
		sessionSvc: sessionSvc,
		tokenMaker: tokenMaker,
	}
}

func (h *AuthHandler) RegisterRoutes(r *gin.Engine) {
	auth := r.Group("/v1/auth")
	{
		auth.POST("/register", h.HandleRegister)
		auth.POST("/login", h.HandleLogin)
		auth.POST("/logout", h.HandleLogout)
		auth.GET("/me", h.HandleMe)
	}
}

type RegisterRequest struct {
	OrgID    string `json:"org_id" binding:"required"`
	Username string `json:"username" binding:"required,min=3,max=100"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
	FullName string `json:"full_name" binding:"required,min=1,max=255"`
}

func (h *AuthHandler) HandleRegister(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	orgID, err := uuid.Parse(req.OrgID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid org_id"})
		return
	}

	email := domain.NormalizeEmail(req.Email)

	existingUser, err := h.store.UserRepository.GetByUsername(c.Request.Context(), orgID, req.Username)
	if err == nil && existingUser != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "username already exists"})
		return
	}

	existingUser, err = h.store.UserRepository.GetByEmail(c.Request.Context(), orgID, email)
	if err == nil && existingUser != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "email already exists"})
		return
	}

	if err := domain.DefaultPasswordPolicy.Validate(req.Password, req.Username, email); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	passwordHash, err := h.hasher.Hash(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}

	now := time.Now()
	user := &domain.User{
		ID:                     uuid.Must(uuid.NewV7()),
		OrgID:                  orgID,
		Username:               req.Username,
		Email:                  email,
		FullName:               req.FullName,
		Status:                 domain.UserStatusPendingVerification,
		EmailVerified:          false,
		MFAEnabled:             false,
		RequirePasswordChange:  false,
		FailedLoginAttempts:    0,
		CreatedAt:              now,
		UpdatedAt:              now,
	}

	if err := user.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.store.UserRepository.Create(c.Request.Context(), user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
		return
	}

	cred := &domain.UserCredential{
		ID:             uuid.Must(uuid.NewV7()),
		UserID:         user.ID,
		CredentialType: domain.CredentialTypePassword,
		SecretHash:     passwordHash,
		Version:        1,
		IsCurrent:      true,
		CreatedAt:      now,
	}

	if err := h.store.CredentialRepository.Create(c.Request.Context(), cred); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create credential"})
		return
	}

	verifyToken := uuid.New().String()
	vt := &domain.VerificationToken{
		ID:        uuid.Must(uuid.NewV7()),
		UserID:    user.ID,
		Token:     verifyToken,
		TokenType: domain.TokenTypeEmailVerification,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: now,
	}

	if err := h.store.VerificationTokenRepository.Create(c.Request.Context(), vt); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create verification token"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"user": gin.H{
			"id":         user.ID.String(),
			"username":   user.Username,
			"email":      user.Email,
			"full_name":  user.FullName,
			"status":     user.Status,
			"org_id":     user.OrgID.String(),
		},
		"verify_token": verifyToken,
	})
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	OrgID    string `json:"org_id" binding:"required"`
	MFACode  string `json:"mfa_code"`
}

func (h *AuthHandler) HandleLogin(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	orgID, err := uuid.Parse(req.OrgID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid org_id"})
		return
	}

	var user *domain.User
	usernameOrEmail := strings.TrimSpace(req.Username)
	if strings.Contains(usernameOrEmail, "@") {
		user, err = h.store.UserRepository.GetByEmail(c.Request.Context(), orgID, domain.NormalizeEmail(usernameOrEmail))
	} else {
		user, err = h.store.UserRepository.GetByUsername(c.Request.Context(), orgID, usernameOrEmail)
	}
	if err != nil || user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	if !user.CanLogin() {
		c.JSON(http.StatusForbidden, gin.H{"error": "account not available for login"})
		return
	}

	cred, err := h.store.CredentialRepository.GetCurrentByUserID(c.Request.Context(), user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get credential"})
		return
	}

	if !h.hasher.Verify(req.Password, cred.SecretHash) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	if user.MFAEnabled && req.MFACode == "" {
		c.JSON(http.StatusOK, gin.H{
			"mfa_required": true,
			"mfa_type":    string(user.MFAMethod),
			"user_id":     user.ID.String(),
		})
		return
	}

	ipAddress := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	claims := jwt.Claims{
		Roles: []string{},
	}

	result, err := h.sessionSvc.CreateSession(c.Request.Context(), user.ID, orgID, ipAddress, userAgent, claims)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create session"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  result.AccessToken,
		"refresh_token": result.RefreshToken,
		"token_type":    "Bearer",
		"expires_in":    result.ExpiresIn,
		"session_id":    result.SessionID,
		"user": gin.H{
			"id":         user.ID.String(),
			"username":   user.Username,
			"email":      user.Email,
			"full_name":  user.FullName,
			"mfa_enabled": user.MFAEnabled,
		},
	})
}

func (h *AuthHandler) HandleLogout(c *gin.Context) {
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

	claims, err := h.sessionSvc.VerifyAccessToken(c.Request.Context(), tokenStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_token"})
		return
	}

	if err := h.sessionSvc.RevokeSession(c.Request.Context(), claims.SessionID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to logout"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "logged_out"})
}

func (h *AuthHandler) HandleMe(c *gin.Context) {
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

	claims, err := h.sessionSvc.VerifyAccessToken(c.Request.Context(), tokenStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_token"})
		return
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_token"})
		return
	}

	user, err := h.store.UserRepository.GetByID(c.Request.Context(), userID)
	if err != nil || user == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":                user.ID.String(),
		"username":          user.Username,
		"email":             user.Email,
		"full_name":         user.FullName,
		"status":            user.Status,
		"org_id":            user.OrgID.String(),
		"email_verified":    user.EmailVerified,
		"mfa_enabled":       user.MFAEnabled,
		"last_login_at":     user.LastLoginAt,
	})
}

type MFASetupRequest struct {
	UserID   string `json:"user_id" binding:"required"`
	Method   string `json:"method" binding:"required"`
	Email    string `json:"email"`
	TOTPCode string `json:"totp_code"`
}

func (h *AuthHandler) HandleMFASetup(c *gin.Context) {
	var req MFASetupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	user, err := h.store.UserRepository.GetByID(c.Request.Context(), userID)
	if err != nil || user == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	switch req.Method {
	case "totp":
		if req.TOTPCode == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "totp_code required"})
			return
		}
	case "email_otp":
		if req.Email == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "email required"})
			return
		}
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid method"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "mfa_configured",
		"method":  req.Method,
		"user_id": userID.String(),
	})
}