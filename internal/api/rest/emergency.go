package rest

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type EmergencyHandler struct {
	emergencySvc any
}

func NewEmergencyHandler(emergencySvc any) *EmergencyHandler {
	return &EmergencyHandler{emergencySvc: emergencySvc}
}

func (h *EmergencyHandler) RegisterRoutes(r *gin.RouterGroup) {
	emergency := r.Group("/emergency")
	{
		emergency.POST("/revoke-all-tokens", h.RevokeAllTokens)
		emergency.POST("/force-logout", h.ForceLogoutAllUsers)
		emergency.POST("/key-rotation", h.EmergencyKeyRotation)
		emergency.GET("/status", h.GetSecurityStatus)
	}
}

type EmergencyRequest struct {
	OrgID string `json:"org_id" binding:"required"`
}

type EmergencyResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func (h *EmergencyHandler) RevokeAllTokens(c *gin.Context) {
	var req EmergencyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "message": err.Error()})
		return
	}

	orgID, err := uuid.Parse(req.OrgID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}

	if emergencySvc, ok := h.emergencySvc.(interface {
		RevokeAllTokens(ctx any, orgID uuid.UUID) error
	}); ok {
		if err := emergencySvc.RevokeAllTokens(c.Request.Context(), orgID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed_to_revoke_tokens"})
			return
		}
	}

	c.JSON(http.StatusOK, EmergencyResponse{
		Success: true,
		Message: "All tokens revoked successfully",
	})
}

func (h *EmergencyHandler) ForceLogoutAllUsers(c *gin.Context) {
	var req EmergencyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "message": err.Error()})
		return
	}

	orgID, err := uuid.Parse(req.OrgID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}

	if emergencySvc, ok := h.emergencySvc.(interface {
		ForceLogoutAllUsers(ctx any, orgID uuid.UUID) error
	}); ok {
		if err := emergencySvc.ForceLogoutAllUsers(c.Request.Context(), orgID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed_to_force_logout"})
			return
		}
	}

	c.JSON(http.StatusOK, EmergencyResponse{
		Success: true,
		Message: "All users logged out successfully",
	})
}

func (h *EmergencyHandler) EmergencyKeyRotation(c *gin.Context) {
	var req EmergencyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "message": err.Error()})
		return
	}

	orgID, err := uuid.Parse(req.OrgID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}

	if emergencySvc, ok := h.emergencySvc.(interface {
		EmergencyKeyRotation(ctx any, orgID uuid.UUID) error
	}); ok {
		if err := emergencySvc.EmergencyKeyRotation(c.Request.Context(), orgID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed_to_rotate_keys"})
			return
		}
	}

	c.JSON(http.StatusOK, EmergencyResponse{
		Success: true,
		Message: "Emergency key rotation completed successfully",
	})
}

func (h *EmergencyHandler) GetSecurityStatus(c *gin.Context) {
	orgIDStr := c.Query("org_id")
	if orgIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "org_id required"})
		return
	}

	orgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_org_id"})
		return
	}

	if emergencySvc, ok := h.emergencySvc.(interface {
		GetSecurityStatus(ctx any, orgID uuid.UUID) (map[string]interface{}, error)
	}); ok {
		status, err := emergencySvc.GetSecurityStatus(c.Request.Context(), orgID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed_to_get_status"})
			return
		}
		c.JSON(http.StatusOK, status)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"org_id": orgIDStr,
		"status": "active",
	})
}