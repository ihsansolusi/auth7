package rest

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ihsansolusi/auth7/internal/service/mfa"
)

type MFAHandler struct {
	mfaSvc *mfa.MFAService
}

func NewMFAHandler(mfaSvc *mfa.MFAService) *MFAHandler {
	return &MFAHandler{mfaSvc: mfaSvc}
}

func (h *MFAHandler) RegisterRoutes(r *gin.RouterGroup, mfaSvc *mfa.MFAService) {
	h.mfaSvc = mfaSvc

	mfaGroup := r.Group("/mfa")
	{
		mfaGroup.POST("/enroll/totp", h.EnrollTOTP)
		mfaGroup.POST("/verify/totp", h.VerifyTOTP)
		mfaGroup.POST("/enroll/email", h.EnrollEmailOTP)
		mfaGroup.POST("/verify/email", h.VerifyEmailOTP)
		mfaGroup.POST("/backup-codes/generate", h.GenerateBackupCodes)
		mfaGroup.POST("/verify/backup", h.VerifyBackupCode)
		mfaGroup.GET("/config/:user_id", h.GetMFAConfig)
		mfaGroup.POST("/step-up", h.StepUpAuth)
		mfaGroup.POST("/setup", h.SetupMFA)
	}
}

type EnrollTOTPRequest struct {
	UserID string `json:"user_id" binding:"required"`
}

func (h *MFAHandler) EnrollTOTP(c *gin.Context) {
	var req EnrollTOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	output, err := h.mfaSvc.EnrollTOTP(c.Request.Context(), mfa.EnrollTOTPInput{UserID: userID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"secret":      output.Secret,
		"qr_code_url": output.QRCodeData,
	})
}

type VerifyTOTPRequest struct {
	UserID string `json:"user_id" binding:"required"`
	Code   string `json:"code" binding:"required"`
}

func (h *MFAHandler) VerifyTOTP(c *gin.Context) {
	var req VerifyTOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	if err := h.mfaSvc.EnableTOTP(c.Request.Context(), userID, req.Code); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "totp_enabled"})
}

type EnrollEmailOTPRequest struct {
	UserID string `json:"user_id" binding:"required"`
	Email  string `json:"email" binding:"required,email"`
}

func (h *MFAHandler) EnrollEmailOTP(c *gin.Context) {
	var req EnrollEmailOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	code, err := h.mfaSvc.EnrollEmailOTP(c.Request.Context(), mfa.EnrollEmailOTPInput{
		UserID: userID,
		Email:  req.Email,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "verification code sent",
		"code_sent_to": req.Email,
	})
}

type VerifyEmailOTPRequest struct {
	UserID  string `json:"user_id" binding:"required"`
	Code    string `json:"code" binding:"required,len=6"`
	Purpose string `json:"purpose"`
}

func (h *MFAHandler) VerifyEmailOTP(c *gin.Context) {
	var req VerifyEmailOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	purpose := "mfa_login"
	if req.Purpose != "" {
		purpose = req.Purpose
	}

	if err := h.mfaSvc.VerifyEmailOTP(c.Request.Context(), mfa.VerifyEmailOTPInput{
		UserID:  userID,
		Code:    req.Code,
		Purpose: purpose,
	}); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "verified"})
}

type GenerateBackupCodesRequest struct {
	UserID string `json:"user_id" binding:"required"`
}

func (h *MFAHandler) GenerateBackupCodes(c *gin.Context) {
	var req GenerateBackupCodesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	output, err := h.mfaSvc.GenerateBackupCodes(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"codes": output.Codes,
	})
}

type VerifyBackupCodeRequest struct {
	UserID string `json:"user_id" binding:"required"`
	Code   string `json:"code" binding:"required"`
}

func (h *MFAHandler) VerifyBackupCode(c *gin.Context) {
	var req VerifyBackupCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	if err := h.mfaSvc.VerifyBackupCode(c.Request.Context(), mfa.VerifyBackupCodeInput{
		UserID: userID,
		Code:   req.Code,
	}); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "verified"})
}

func (h *MFAHandler) GetMFAConfig(c *gin.Context) {
	userIDStr := c.Param("user_id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	output, err := h.mfaSvc.GetMFAConfig(c.Request.Context(), mfa.GetMFAConfigInput{UserID: userID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, output)
}

type StepUpAuthRequest struct {
	UserID string `json:"user_id" binding:"required"`
	Method string `json:"method" binding:"required"`
	Code   string `json:"code" binding:"required"`
}

func (h *MFAHandler) StepUpAuth(c *gin.Context) {
	var req StepUpAuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	output, err := h.mfaSvc.StepUpAuth(c.Request.Context(), mfa.StepUpAuthInput{
		UserID: userID,
		Method: req.Method,
		Code:   req.Code,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, output)
}

type SetupMFARequest struct {
	UserID   string `json:"user_id" binding:"required"`
	Method   string `json:"method" binding:"required"`
	Email    string `json:"email"`
	TOTPCode string `json:"totp_code"`
}

func (h *MFAHandler) SetupMFA(c *gin.Context) {
	var req SetupMFARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	if err := h.mfaSvc.SetupMFA(c.Request.Context(), mfa.SetupMFAInput{
		UserID:   userID,
		Method:   req.Method,
		Email:    req.Email,
		TOTPCode: req.TOTPCode,
	}); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "mfa_configured"})
}