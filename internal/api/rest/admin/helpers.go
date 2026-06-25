package admin

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
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
