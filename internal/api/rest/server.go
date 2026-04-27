package rest

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/ihsansolusi/auth7/pkg/config"
	"github.com/ihsansolusi/lib7-service-go/metrics"
	"github.com/ihsansolusi/lib7-service-go/logging"
	"github.com/ihsansolusi/lib7-service-go/token"
	"go.opentelemetry.io/otel/trace"
)

type ServerDeps struct {
	Service any
	DB      any
	Logger  zerolog.Logger
	Tracer  trace.Tracer
	Metrics *metrics.Registry
	AuditLogger *logging.AuditLogger
	TokenMaker  token.Maker
	JWTSvc     any
	SessionSvc any
	Config     *config.Config
}

type Server struct {
	deps ServerDeps
}

type OIDCDiscovery struct {
	Issuer                            string   `json:"issuer"`
	AuthorizationEndpoint             string   `json:"authorization_endpoint"`
	TokenEndpoint                     string   `json:"token_endpoint"`
	UserInfoEndpoint                  string   `json:"userinfo_endpoint"`
	JwksURI                           string   `json:"jwks_uri"`
	RegistrationEndpoint              string   `json:"registration_endpoint"`
	ScopesSupported                   []string `json:"scopes_supported"`
	ResponseTypesSupported            []string `json:"response_types_supported"`
	GrantTypesSupported               []string `json:"grant_types_supported"`
	SubjectTypesSupported             []string `json:"subject_types_supported"`
	IDTokenSigningAlgValuesSupported  []string `json:"id_token_signing_alg_values_supported"`
	TokenEndpointAuthMethodsSupported []string `json:"token_endpoint_auth_methods_supported"`
	CodeChallengeMethodsSupported     []string `json:"code_challenge_methods_supported"`
}

func NewServer(deps ServerDeps) *Server {
	return &Server{deps: deps}
}

func (s *Server) RegisterRoutes(r *gin.Engine, deps ServerDeps) {
	s.deps = deps

	r.GET("/health/live", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	r.GET("/health/ready", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ready"})
	})

	r.GET("/.well-known/jwks.json", s.handleJWKS)
	r.GET("/.well-known/openid-configuration", s.handleOIDCDiscovery)

	s.RegisterOAuth2Routes(r)
	s.RegisterBranchRoutes(r)
}

func (s *Server) handleJWKS(c *gin.Context) {
	type JWKSResponse struct {
		Keys []map[string]interface{} `json:"keys"`
	}

	var keys []map[string]interface{}
	if jwtSvc, ok := s.deps.JWTSvc.(interface{ GetJWKS() []map[string]interface{} }); ok {
		keys = jwtSvc.GetJWKS()
	}

	c.JSON(http.StatusOK, JWKSResponse{Keys: keys})
}

func (s *Server) handleOIDCDiscovery(c *gin.Context) {
	discovery := OIDCDiscovery{
		Issuer:                            "https://auth7.bank.co.id",
		AuthorizationEndpoint:             "https://auth7.bank.co.id/oauth2/authorize",
		TokenEndpoint:                     "https://auth7.bank.co.id/oauth2/token",
		UserInfoEndpoint:                  "https://auth7.bank.co.id/oauth2/userinfo",
		JwksURI:                           "https://auth7.bank.co.id/.well-known/jwks.json",
		RegistrationEndpoint:              "https://auth7.bank.co.id/oauth2/register",
		ScopesSupported:                   []string{"openid", "profile", "email", "roles"},
		ResponseTypesSupported:            []string{"code"},
		GrantTypesSupported:               []string{"authorization_code", "refresh_token", "client_credentials"},
		SubjectTypesSupported:             []string{"public"},
		IDTokenSigningAlgValuesSupported:  []string{"RS256"},
		TokenEndpointAuthMethodsSupported: []string{"client_secret_basic", "client_secret_post"},
		CodeChallengeMethodsSupported:     []string{"S256"},
	}

	c.JSON(http.StatusOK, discovery)
}
