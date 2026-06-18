package rest

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ihsansolusi/auth7/internal/mailer"
	"github.com/ihsansolusi/auth7/internal/service/audit"
	jwtpkg "github.com/ihsansolusi/auth7/internal/service/jwt"
	"github.com/ihsansolusi/auth7/internal/store/postgres"
)

// RegisterInternalV1Routes wires service-to-service (M2M-only) endpoints
// under /internal/v1. These are called by other Core7 services (workflow7,
// audit7, etc.) using OAuth2 client_credentials tokens issued by auth7's own
// /oauth2/token endpoint.
//
// The endpoints here MUST NOT be reachable by user JWTs — m2mOnlyMW enforces
// that. Add user-facing reads to /admin/v1 (JWT + AdminAuth) instead.
func (s *Server) RegisterInternalV1Routes(r *gin.Engine, m mailer.Mailer) {
	store, ok := s.deps.Store.(*postgres.Store)
	if !ok {
		s.deps.Logger.Warn().Msg("internal routes: store type assertion failed, skipping")
		return
	}
	jwtSvc, ok := s.deps.JWTSvc.(*jwtpkg.Service)
	if !ok {
		s.deps.Logger.Warn().Msg("internal routes: jwtSvc type assertion failed, skipping")
		return
	}

	auditSvc := audit.NewService(audit.NewPGStore(store.Pool()))
	a7url, a7key := s.audit7Settings()
	auditSvc.SetForwarder(audit.NewAudit7Forwarder(a7url, a7key, s.deps.Logger))

	internalV1 := r.Group("/internal/v1")
	internalV1.Use(m2mOnlyMW(jwtSvc))

	internalV1.GET("/user-context/:user_id", s.handleInternalUserContext(store))

	// workflow7 service-task callbacks for the user lifecycle + assignments.
	newUserWfHandler(
		newAdminUserSvc(store),
		newAdminUserRoleSvc(store),
		newAdminBranchSvc(store),
		store,
		auditSvc,
		m,
		s.deps.Logger,
	).registerRoutes(internalV1)

	// workflow7 service-task callbacks for the role lifecycle + permission assignment.
	newRoleWfHandler(newAdminRoleSvc(store), auditSvc, s.deps.Logger).registerRoutes(internalV1)

	// workflow7 service-task callbacks for the OAuth2 client lifecycle.
	newOAuth2ClientWfHandler(newAdminOAuth2ClientSvc(store), auditSvc, s.deps.Logger).registerRoutes(internalV1)
}

// audit7Settings resolves the central audit7 URL + service key, preferring the
// config file (always loaded) and falling back to env vars for env-driven
// deployments. Empty URL disables forwarding.
func (s *Server) audit7Settings() (string, string) {
	url, key := "", ""
	if s.deps.Config != nil {
		url = s.deps.Config.Audit7.URL
		key = s.deps.Config.Audit7.ServiceKey
	}
	if url == "" {
		url = os.Getenv("AUDIT7_URL")
	}
	if key == "" {
		key = os.Getenv("AUDIT7_SERVICE_KEY")
	}
	return url, key
}

// m2mOnlyMW verifies the Bearer token against auth7's own JWT service and
// rejects any token that does NOT carry a client_id claim (i.e. user tokens).
// Tokens issued via OAuth2 client_credentials grant have ClientID populated;
// user tokens leave it empty.
func m2mOnlyMW(jwtSvc *jwtpkg.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if len(auth) < 8 || auth[:7] != "Bearer " {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid_token"})
			return
		}
		claims, err := jwtSvc.VerifyAccessToken(auth[7:])
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid_token"})
			return
		}
		if claims.ClientID == "" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "endpoint restricted to service callers"})
			return
		}
		c.Next()
	}
}

// handleInternalUserContext — GET /internal/v1/user-context/:user_id
//
// Returns the identity + primary active branch projection + global roles for
// the requested user. Designed to populate workflow7's adapter.UserContext
// shape so the audit envelope can carry username/org_id/branch_code without
// the caller round-tripping multiple endpoints.
//
// Fields NOT returned (data lives elsewhere):
//   - branch_name, branch_level, parent_branch_id — owned by enterprise domain.
//     Callers that need branch hierarchy must compose with the enterprise service.
func (s *Server) handleInternalUserContext(store *postgres.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := uuid.Parse(c.Param("user_id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
			return
		}

		ctx := c.Request.Context()

		user, err := store.UserRepository.GetByID(ctx, userID)
		if err != nil {
			s.deps.Logger.Error().Err(err).Str("user_id", userID.String()).Msg("user-context: GetByID failed")
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		if user == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}

		var branchID, branchCode, branchName string
		if primary, err := store.UserBranchAssignmentRepository.GetPrimaryByUserID(ctx, userID); err == nil && primary != nil {
			branchID = primary.BranchID.String()
			branchCode = primary.BranchCode
			branchName = primary.BranchName
		} else if any, err := store.UserBranchAssignmentRepository.GetAnyActiveByUserID(ctx, userID); err == nil && any != nil {
			branchID = any.BranchID.String()
			branchCode = any.BranchCode
			branchName = any.BranchName
		}

		roles, _ := store.UserRoleRepository.GetRoleCodesByUser(ctx, userID)
		if roles == nil {
			roles = []string{}
		}

		c.JSON(http.StatusOK, gin.H{
			"user_id":          user.ID.String(),
			"username":         user.Username,
			"name":             user.FullName,
			"org_id":           user.OrgID.String(),
			"branch_id":        branchID,
			"branch_code":      branchCode,
			"branch_name":      branchName,
			"branch_level":     "",
			"parent_branch_id": nil,
			"roles":            roles,
		})
	}
}
