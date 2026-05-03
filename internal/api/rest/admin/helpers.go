package admin

import "github.com/gin-gonic/gin"

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
