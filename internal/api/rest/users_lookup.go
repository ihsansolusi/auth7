package rest

// User lookup endpoint for cross-service pickers (e.g., bos7-financing
// pejabat form needs to pick a user from the same org).
//
// Auth: user Bearer JWT (delegated via BFF token exchange). Reads
// org_id from the verified claims — never trusts a body field. The
// caller's org_id scopes the query so cross-tenant enumeration is
// blocked even if the BFF mis-routes.
//
// Response shape: lib7 DataTable contract (data + columnTypes +
// altSortColumns/altSearchColumns + allowNext/allowPrev). Written
// inline to avoid bumping the auth7 lib7-service-go version just for
// this endpoint.

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	jwtpkg "github.com/ihsansolusi/auth7/internal/service/jwt"
	"github.com/ihsansolusi/auth7/internal/store/postgres"
)

// dataTableRequest is the cursor envelope from ServerDataTable. Only
// the fields we actually use are extracted; unknown keys are ignored.
type dataTableRequest struct {
	ReqType    string                 `json:"reqType"`
	PageSize   int                    `json:"pageSize"`
	TopData    map[string]interface{} `json:"topData"`
	BottomData map[string]interface{} `json:"bottomData"`
	SearchText *string                `json:"searchText"`
}

type userLookupRow struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	FullName string `json:"full_name"`
	Email    string `json:"email"`
	Status   string `json:"status"`
}

// handleUserLookup serves POST /v1/users/lookup/query — DataTable cursor
// pagination over auth7.users, scoped to the caller's org. Ordering is
// (username ASC, id ASC) with a composite cursor.
func (s *Server) handleUserLookup(c *gin.Context) {
	auth := c.GetHeader("Authorization")
	tokenStr := trimBearer(auth)
	if tokenStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_token"})
		return
	}
	verifier, ok := s.deps.SessionSvc.(interface {
		VerifyAccessToken(ctx context.Context, token string) (*jwtpkg.Claims, error)
	})
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "session service unavailable"})
		return
	}
	claims, err := verifier.VerifyAccessToken(c.Request.Context(), tokenStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_token"})
		return
	}
	orgID, err := uuid.Parse(claims.OrgID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid org_id in token"})
		return
	}

	store, ok := s.deps.Store.(*postgres.Store)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "store unavailable"})
		return
	}

	var req dataTableRequest
	if c.Request.ContentLength > 0 {
		_ = json.NewDecoder(c.Request.Body).Decode(&req)
	}
	if req.PageSize <= 0 || req.PageSize > 100 {
		req.PageSize = 20
	}
	search := ""
	if req.SearchText != nil {
		search = strings.TrimSpace(*req.SearchText)
	}
	fetch := req.PageSize + 1
	like := "%" + search + "%"

	// Composite cursor: (username, id). username NOT unique here; id
	// is the tiebreaker.
	var (
		sqlQ string
		args []any
	)
	switch req.ReqType {
	case "prev", "last":
		curU, _ := req.TopData["username"].(string)
		curID, _ := req.TopData["id"].(string)
		if curU != "" && req.ReqType == "prev" {
			sqlQ = `SELECT id, username, full_name, email, status FROM users
				WHERE org_id=$1 AND deleted_at IS NULL
				  AND (username, id::text) < ($2, $3)
				  AND ($4='' OR username ILIKE $5 OR full_name ILIKE $5 OR email ILIKE $5)
				ORDER BY username DESC, id DESC LIMIT $6`
			args = []any{orgID, curU, curID, search, like, fetch}
		} else {
			sqlQ = `SELECT id, username, full_name, email, status FROM users
				WHERE org_id=$1 AND deleted_at IS NULL
				  AND ($2='' OR username ILIKE $3 OR full_name ILIKE $3 OR email ILIKE $3)
				ORDER BY username DESC, id DESC LIMIT $4`
			args = []any{orgID, search, like, fetch}
		}
	default:
		curU, _ := req.BottomData["username"].(string)
		curID, _ := req.BottomData["id"].(string)
		if curU != "" {
			sqlQ = `SELECT id, username, full_name, email, status FROM users
				WHERE org_id=$1 AND deleted_at IS NULL
				  AND (username, id::text) > ($2, $3)
				  AND ($4='' OR username ILIKE $5 OR full_name ILIKE $5 OR email ILIKE $5)
				ORDER BY username ASC, id ASC LIMIT $6`
			args = []any{orgID, curU, curID, search, like, fetch}
		} else {
			sqlQ = `SELECT id, username, full_name, email, status FROM users
				WHERE org_id=$1 AND deleted_at IS NULL
				  AND ($2='' OR username ILIKE $3 OR full_name ILIKE $3 OR email ILIKE $3)
				ORDER BY username ASC, id ASC LIMIT $4`
			args = []any{orgID, search, like, fetch}
		}
	}

	rows, qErr := store.Pool().Query(c.Request.Context(), sqlQ, args...)
	if qErr != nil {
		s.deps.Logger.Error().Err(qErr).Msg("user lookup query failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}
	defer rows.Close()

	items := make([]userLookupRow, 0, fetch)
	for rows.Next() {
		var r userLookupRow
		if err := rows.Scan(&r.ID, &r.Username, &r.FullName, &r.Email, &r.Status); err != nil {
			s.deps.Logger.Error().Err(err).Msg("user lookup scan failed")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
			return
		}
		items = append(items, r)
	}

	hasMore := len(items) > req.PageSize
	if hasMore {
		items = items[:req.PageSize]
	}
	if req.ReqType == "prev" || req.ReqType == "last" {
		for i, j := 0, len(items)-1; i < j; i, j = i+1, j-1 {
			items[i], items[j] = items[j], items[i]
		}
	}

	allowNext := hasMore && req.ReqType != "last"
	allowPrev := req.ReqType == "next" || req.ReqType == "prev" || req.ReqType == "last"

	data := make([]any, len(items))
	for i, r := range items {
		data[i] = r
	}

	c.JSON(http.StatusOK, gin.H{
		"data": data,
		"columnTypes": map[string]string{
			"username":  "String",
			"full_name": "String",
			"email":     "String",
			"status":    "String",
		},
		"altSortColumns":   []string{"username", "full_name", "email"},
		"altSearchColumns": []string{"username", "full_name", "email"},
		"allowNext":        allowNext,
		"allowPrev":        allowPrev,
	})
}
