package admin

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ihsansolusi/auth7/internal/domain"
	"github.com/jackc/pgx/v5"
)

// claimsOrgID extracts org_id from the JWT claims stored in gin context by bearerMW.
func claimsOrgID(c *gin.Context) string {
	raw, ok := c.Get("claims")
	if !ok {
		return ""
	}
	type orgGetter interface{ GetOrgID() string }
	if g, ok := raw.(orgGetter); ok {
		return g.GetOrgID()
	}
	return ""
}

// claimsSubject extracts the subject (user ID) from JWT claims.
func claimsSubject(c *gin.Context) string {
	raw, ok := c.Get("claims")
	if !ok {
		return ""
	}
	type subGetter interface{ GetSubject() string }
	if g, ok := raw.(subGetter); ok {
		return g.GetSubject()
	}
	return ""
}

// requireOrgID resolves the authoritative organization scope for an admin request.
//
// The JWT org claim is authoritative. A query `?org_id` is only honored when the
// claim has no org — i.e. platform/super_admin tokens operating cross-org — and the
// AdminAuth middleware (RequireOrgMatch) has already rejected any query org that
// conflicts with a non-empty claim org. This makes the caller's token, not a
// client-supplied query param, the source of truth for tenant scoping.
//
// Returns false (and writes 400) when no valid org can be resolved.
func requireOrgID(c *gin.Context) (uuid.UUID, bool) {
	orgStr := claimsOrgID(c)
	if orgStr == "" {
		orgStr = c.Query("org_id")
	}
	if orgStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "org_id required"})
		return uuid.Nil, false
	}
	orgID, err := uuid.Parse(orgStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid org_id"})
		return uuid.Nil, false
	}
	return orgID, true
}

// respondError maps a service/store error to a canonical HTTP status for the
// legacy /admin/v1 admin handlers. It preserves the existing {"error": "<code>"}
// body shape that current consumers (e.g. the bos7-enterprise BFF, which forwards
// the upstream status) already parse — only the status code is corrected.
//
//	not-found         → 404   (domain.ErrNotFound / pgx.ErrNoRows)
//	already-exists    → 409   (domain.ErrAlreadyExists)
//	permission denied → 403   (domain.ErrPermissionDenied)
//	otherwise         → 500
//
// Callers keep their own contextual error logging before calling this helper.
func respondError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrNotFound), errors.Is(err, pgx.ErrNoRows):
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
	case errors.Is(err, domain.ErrAlreadyExists):
		c.JSON(http.StatusConflict, gin.H{"error": "already_exists"})
	case errors.Is(err, domain.ErrPermissionDenied):
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
	}
}
