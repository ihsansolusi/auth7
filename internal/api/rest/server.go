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

func NewServer(deps ServerDeps) *Server {
	return &Server{deps: deps}
}

func (s *Server) RegisterRoutes(r *gin.Engine, deps ServerDeps) {
	r.GET("/health/live", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	r.GET("/health/ready", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ready"})
	})

	r.GET("/.well-known/jwks.json", s.handleJWKS)
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
