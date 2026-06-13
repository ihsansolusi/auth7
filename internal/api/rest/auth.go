package rest

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ihsansolusi/auth7/internal/domain"
	"github.com/ihsansolusi/auth7/internal/mailer"
	"github.com/ihsansolusi/auth7/internal/messaging/nats"
	"github.com/ihsansolusi/auth7/internal/service/jwt"
	"github.com/ihsansolusi/auth7/internal/service/password"
	"github.com/ihsansolusi/auth7/internal/service/session"
	"github.com/ihsansolusi/auth7/internal/store/postgres"
	"github.com/ihsansolusi/lib7-service-go/token"
	"github.com/pquerna/otp/totp"
)

type AuthHandler struct {
	store      *postgres.Store
	hasher     *password.Hasher
	sessionSvc *session.Service
	tokenMaker  token.Maker
	eventPub    *nats.EventPublisher
	mailer      mailer.Mailer
	baseURL     string
	frontendURL string
}

func NewAuthHandler(store *postgres.Store, hasher *password.Hasher, sessionSvc *session.Service, tokenMaker token.Maker, eventPub *nats.EventPublisher, m mailer.Mailer, baseURL string, frontendURL string) *AuthHandler {
	return &AuthHandler{
		store:       store,
		hasher:      hasher,
		sessionSvc:  sessionSvc,
		tokenMaker:  tokenMaker,
		eventPub:    eventPub,
		mailer:      m,
		baseURL:     baseURL,
		frontendURL: frontendURL,
	}
}

func (h *AuthHandler) RegisterRoutes(r *gin.Engine) {
	auth := r.Group("/v1/auth")
	{
		auth.POST("/register", h.HandleRegister)
		auth.POST("/login", h.HandleLogin)
		auth.POST("/logout", h.HandleLogout)
		auth.GET("/me", h.HandleMe)
		auth.PUT("/profile", h.HandleUpdateProfile)
		auth.POST("/change-password", h.HandleChangePassword)
		auth.POST("/forgot-password", h.HandleForgotPassword)
		auth.POST("/reset-password", h.HandleResetPassword)
	}

	mfa := auth.Group("/mfa")
	{
		mfa.POST("/setup", h.HandleMFASetup)
		mfa.POST("/verify", h.HandleMFAVerify)
		mfa.POST("/disable", h.HandleMFADisable)
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

	existingUser, err := h.store.UserRepository.GetByUsername(c.Request.Context(), req.Username)
	if err == nil && existingUser != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "username already exists"})
		return
	}

	existingUser, err = h.store.UserRepository.GetByEmail(c.Request.Context(), email)
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
		PreferredLocale:        "id",
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

	if h.mailer != nil {
		go func() {
			verifyURL := fmt.Sprintf("%s/v1/auth/verify?token=%s", h.baseURL, verifyToken)
			html, _ := mailer.RenderVerificationEmail("Verifikasi Email Auth7", verifyURL)
			_ = h.mailer.Send(context.Background(), user.Email, "Verifikasi Email Auth7", html)
		}()
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
	MFACode  string `json:"mfa_code"`
}

func (h *AuthHandler) HandleLogin(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var (
		user *domain.User
		err  error
	)
	usernameOrEmail := strings.TrimSpace(req.Username)
	if strings.Contains(usernameOrEmail, "@") {
		user, err = h.store.UserRepository.GetByEmail(c.Request.Context(), domain.NormalizeEmail(usernameOrEmail))
	} else {
		user, err = h.store.UserRepository.GetByUsername(c.Request.Context(), usernameOrEmail)
	}
	if err != nil || user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	orgID := user.OrgID

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
		if user.MFAMethod == domain.MFAMethodEmailOTP && h.mailer != nil {
			go func() {
				code := fmt.Sprintf("%06d", time.Now().Unix()%1000000)
				html, _ := mailer.RenderOTPEmail("Kode Verifikasi Login Auth7", code)
				_ = h.mailer.Send(context.Background(), user.Email, "Kode Verifikasi Login Auth7", html)
			}()
		}
		c.JSON(http.StatusOK, gin.H{
			"mfa_required": true,
			"mfa_type":    string(user.MFAMethod),
			"user_id":     user.ID.String(),
		})
		return
	}

	ipAddress := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	roles, _ := h.store.UserRoleRepository.GetRoleCodesByUser(c.Request.Context(), user.ID)

	var branchID, branchCode string
	if primaryBranch, err := h.store.UserBranchAssignmentRepository.GetPrimaryByUserID(c.Request.Context(), user.ID); err == nil && primaryBranch != nil {
		branchID = primaryBranch.BranchID.String()
		branchCode = primaryBranch.BranchCode
	} else if anyBranch, err := h.store.UserBranchAssignmentRepository.GetAnyActiveByUserID(c.Request.Context(), user.ID); err == nil && anyBranch != nil {
		branchID = anyBranch.BranchID.String()
		branchCode = anyBranch.BranchCode
	}

	claims := jwt.Claims{
		Username:   user.Username,
		Email:      user.Email,
		Roles:      roles,
		BranchID:   branchID,
		BranchCode: branchCode,
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

	if h.eventPub != nil {
		_ = h.eventPub.PublishSessionCreated(c.Request.Context(), nats.SessionCreatedEvent{
			SessionID: result.SessionID,
			OrgID:     orgID.String(),
			UserID:    user.ID.String(),
			IPAddress: ipAddress,
			CreatedAt: time.Now(),
		})
	}
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

	if err := h.sessionSvc.RevokeAllUserSessions(c.Request.Context(), claims.Subject); err != nil {
		// Fallback: revoke at least the current session
		if revokeErr := h.sessionSvc.RevokeSession(c.Request.Context(), claims.SessionID); revokeErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to logout"})
			return
		}
	}

	if h.eventPub != nil {
		_ = h.eventPub.PublishTokenRevoked(c.Request.Context(), nats.TokenRevokedEvent{
			TokenID:   claims.SessionID,
			OrgID:     claims.OrgID,
			UserID:    claims.Subject,
			RevokedBy: claims.Subject,
			Reason:    "logout",
			RevokedAt: time.Now(),
		})
		_ = h.eventPub.PublishSessionTerminated(c.Request.Context(), nats.SessionTerminatedEvent{
			SessionID:    claims.SessionID,
			OrgID:        claims.OrgID,
			UserID:       claims.Subject,
			Reason:       "logout",
			TerminatedAt: time.Now(),
		})
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
		"preferred_locale":  user.PreferredLocale,
		"status":            user.Status,
		"org_id":            user.OrgID.String(),
		"email_verified":    user.EmailVerified,
		"mfa_enabled":       user.MFAEnabled,
		"last_login_at":     user.LastLoginAt,
	})
}

type UpdateProfileRequest struct {
	FullName        string `json:"full_name"`
	Email           string `json:"email"`
	PreferredLocale string `json:"preferred_locale"`
}

func (h *AuthHandler) HandleUpdateProfile(c *gin.Context) {
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
	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "message": err.Error()})
		return
	}
	if req.FullName != "" {
		user.FullName = req.FullName
	}
	if req.Email != "" {
		user.Email = req.Email
	}
	if req.PreferredLocale != "" {
		if req.PreferredLocale != "id" && req.PreferredLocale != "en" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "message": "preferred_locale must be one of: id, en"})
			return
		}
		user.PreferredLocale = req.PreferredLocale
	}
	user.UpdatedAt = time.Now()
	if err := h.store.UserRepository.Update(c.Request.Context(), user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update_failed", "message": "Gagal memperbarui profil"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"id":             user.ID.String(),
		"username":       user.Username,
		"email":          user.Email,
		"full_name":      user.FullName,
		"preferred_locale": user.PreferredLocale,
		"status":         user.Status,
		"org_id":         user.OrgID.String(),
		"email_verified": user.EmailVerified,
		"mfa_enabled":    user.MFAEnabled,
		"updated_at":     user.UpdatedAt,
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

type MFAVerifyRequest struct {
	UserID string `json:"user_id" binding:"required"`
	Code   string `json:"code" binding:"required"`
}

func (h *AuthHandler) HandleMFAVerify(c *gin.Context) {
	var req MFAVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	cfg, err := h.store.MFAConfigRepository.GetByUserID(c.Request.Context(), userID)
	if err != nil || cfg == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "mfa config not found"})
		return
	}

	if !cfg.IsFullyEnabled() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "mfa not fully enabled"})
		return
	}

	verified := false
	if cfg.IsTOTPEnabled && len(cfg.TOTPSecretEncrypted) > 0 {
		secretBytes, decErr := h.decryptTOTPSecret(cfg.TOTPSecretEncrypted, cfg.TOTPSecretIV)
		if decErr == nil {
			verified = validateTOTPCode(secretBytes, req.Code)
		}
	}

	if !verified && cfg.IsEmailOTPEnabled {
		emailCode, emailErr := h.store.EmailOTPCodeRepository.GetActiveByUserID(c.Request.Context(), userID)
		if emailErr == nil && emailCode != nil && emailCode.IsValid() && emailCode.Code == req.Code {
			verified = true
			h.store.EmailOTPCodeRepository.MarkUsed(c.Request.Context(), emailCode.ID)
		}
	}

	if !verified {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid code"})
		return
	}

	user, err := h.store.UserRepository.GetByID(c.Request.Context(), userID)
	if err == nil && user != nil {
		user.MFAEnabled = true
		user.UpdatedAt = time.Now()
		h.store.UserRepository.Update(c.Request.Context(), user)
	}

	c.JSON(http.StatusOK, gin.H{
		"verified": true,
	})
}

type MFADisableRequest struct {
	UserID string `json:"user_id" binding:"required"`
}

func (h *AuthHandler) HandleMFADisable(c *gin.Context) {
	var req MFADisableRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	cfg, err := h.store.MFAConfigRepository.GetByUserID(c.Request.Context(), userID)
	if err != nil || cfg == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "mfa config not found"})
		return
	}

	cfg.IsTOTPEnabled = false
	cfg.IsEmailOTPEnabled = false
	cfg.IsBackupCodesEnabled = false
	cfg.TOTPSecretEncrypted = nil
	cfg.TOTPSecretIV = nil
	cfg.BackupCodesHash = nil
	cfg.MFAEnabledAt = nil
	cfg.UpdatedAt = time.Now()

	if err := h.store.MFAConfigRepository.Update(c.Request.Context(), cfg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to disable mfa"})
		return
	}

	user, err := h.store.UserRepository.GetByID(c.Request.Context(), userID)
	if err == nil && user != nil {
		user.MFAEnabled = false
		user.MFAMethod = domain.MFAMethodNone
		user.UpdatedAt = time.Now()
		h.store.UserRepository.Update(c.Request.Context(), user)
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required"`
}

func (h *AuthHandler) HandleChangePassword(c *gin.Context) {
	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

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

	cred, err := h.store.CredentialRepository.GetCurrentByUserID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get credential"})
		return
	}

	if !h.hasher.Verify(req.CurrentPassword, cred.SecretHash) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "current password is incorrect"})
		return
	}

	if err := domain.DefaultPasswordPolicy.Validate(req.NewPassword, claims.Username, claims.Email); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	newHash, err := h.hasher.Hash(req.NewPassword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}

	if err := h.store.CredentialRepository.Replace(c.Request.Context(), userID, domain.CredentialTypePassword, newHash, domain.DefaultPasswordPolicy.HistoryCount); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update credential"})
		return
	}

	user, err := h.store.UserRepository.GetByID(c.Request.Context(), userID)
	if err == nil && user != nil {
		now := time.Now()
		user.PasswordChangedAt = &now
		user.UpdatedAt = now
		h.store.UserRepository.Update(c.Request.Context(), user)
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

type ForgotPasswordRequest struct {
	Email     string `json:"email"      binding:"required,email"`
	ReturnURL string `json:"return_url"` // optional: calling app origin, embedded in email link
}

func (h *AuthHandler) HandleForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	email := domain.NormalizeEmail(req.Email)
	user, err := h.store.UserRepository.GetByEmail(c.Request.Context(), email)
	if err != nil || user == nil {
		c.JSON(http.StatusOK, gin.H{"message": "If the email exists, a reset link has been sent"})
		return
	}

	resetToken := uuid.New().String()
	vt := &domain.VerificationToken{
		ID:        uuid.Must(uuid.NewV7()),
		UserID:    user.ID,
		Token:     resetToken,
		TokenType: domain.TokenTypePasswordRecovery,
		ExpiresAt: time.Now().Add(15 * time.Minute),
		CreatedAt: time.Now(),
	}

	if err := h.store.VerificationTokenRepository.Create(c.Request.Context(), vt); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create reset token"})
		return
	}

	if h.mailer != nil {
		go func() {
			uiBase := h.frontendURL
			if uiBase == "" {
				uiBase = h.baseURL
			}
			resetURL := fmt.Sprintf("%s/reset-password?token=%s", uiBase, resetToken)
			if req.ReturnURL != "" {
				resetURL += "&return_to=" + req.ReturnURL
			}
			html, _ := mailer.RenderResetEmail("Reset Password Auth7", resetURL)
			_ = h.mailer.Send(context.Background(), user.Email, "Reset Password Auth7", html)
		}()
	}

	// Return reset_token for dev/test use — auth7-ui proxy strips it before sending to browser
	c.JSON(http.StatusOK, gin.H{
		"message":     "If the email exists, a reset link has been sent",
		"reset_token": resetToken,
	})
}

type ResetPasswordRequest struct {
	Token        string `json:"token" binding:"required"`
	NewPassword  string `json:"new_password" binding:"required"`
}

func (h *AuthHandler) HandleResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	vt, err := h.store.VerificationTokenRepository.GetByToken(c.Request.Context(), req.Token)
	if err != nil || vt == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid or expired reset token"})
		return
	}

	if !vt.IsValid() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid or expired reset token"})
		return
	}

	if vt.TokenType != domain.TokenTypePasswordRecovery {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid token type"})
		return
	}

	if err := domain.DefaultPasswordPolicy.Validate(req.NewPassword, "", ""); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	newHash, err := h.hasher.Hash(req.NewPassword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}

	_, existErr := h.store.CredentialRepository.GetCurrentByUserID(c.Request.Context(), vt.UserID)
	if existErr == nil {
		if err := h.store.CredentialRepository.Replace(c.Request.Context(), vt.UserID, domain.CredentialTypePassword, newHash, domain.DefaultPasswordPolicy.HistoryCount); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update password"})
			return
		}
	} else {
		if err := h.store.CredentialRepository.Create(c.Request.Context(), &domain.UserCredential{
			ID:             uuid.Must(uuid.NewV7()),
			UserID:         vt.UserID,
			CredentialType: domain.CredentialTypePassword,
			SecretHash:     newHash,
			CreatedAt:      time.Now(),
		}); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update password"})
			return
		}
	}

	h.store.VerificationTokenRepository.MarkUsed(c.Request.Context(), vt.ID)

	user, err := h.store.UserRepository.GetByID(c.Request.Context(), vt.UserID)
	if err == nil && user != nil {
		now := time.Now()
		user.PasswordChangedAt = &now
		user.UpdatedAt = now
		h.store.UserRepository.Update(c.Request.Context(), user)
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *AuthHandler) decryptTOTPSecret(encrypted, iv []byte) (string, error) {
	key := []byte("auth7-mfa-secret-key-32bytes!!")
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	if len(encrypted) < aes.BlockSize {
		return "", err
	}
	stream := cipher.NewCFBDecrypter(block, iv)
	decrypted := make([]byte, len(encrypted))
	stream.XORKeyStream(decrypted, encrypted)
	return string(decrypted), nil
}

func validateTOTPCode(secret, code string) bool {
	return totp.Validate(code, secret)
}
