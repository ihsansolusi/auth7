package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"github.com/ihsansolusi/auth7/internal/api/rest"
	"github.com/ihsansolusi/auth7/internal/infrastructure"
	"github.com/ihsansolusi/auth7/internal/service"
	"github.com/ihsansolusi/auth7/internal/service/jwt"
	"github.com/ihsansolusi/auth7/internal/service/password"
	"github.com/ihsansolusi/auth7/internal/service/session"
	"github.com/ihsansolusi/auth7/internal/store/postgres"
	"github.com/ihsansolusi/auth7/pkg/config"
	"github.com/ihsansolusi/lib7-service-go/logging"
	"github.com/ihsansolusi/lib7-service-go/metrics"
	"github.com/ihsansolusi/lib7-service-go/shutdown"
	"github.com/ihsansolusi/lib7-service-go/token"
	"github.com/ihsansolusi/lib7-service-go/tracing"
	"go.opentelemetry.io/otel"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func startCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start",
		Short: "Start the HTTP server",
		RunE:  runStart,
	}
}

func runStart(cmd *cobra.Command, args []string) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	logLevel, err := zerolog.ParseLevel(cfg.Logging.Level)
	if err != nil {
		logLevel = zerolog.InfoLevel
	}
	logger := logging.NewLogger(logging.Options{
		Level:    logLevel,
		TimeZone: cfg.Logging.TimeZone,
		Pretty:   cfg.Logging.Pretty,
	})
	auditLoggerRaw := logging.NewAuditLogger(logging.Options{
		Level:    logLevel,
		TimeZone: cfg.Logging.TimeZone,
		Pretty:   cfg.Logging.Pretty,
	})
	auditLogger := logging.NewAuditLoggerWrapper(auditLoggerRaw)

	logger.Info().
		Str("service", cfg.Service.Name).
		Str("version", cfg.Service.Version).
		Str("profile", cfg.Service.Profile).
		Msg("starting service")

	var shutdownTracer func()
	if cfg.OTEL.Enabled {
		shutdownTracer, err = tracing.InitTracer(ctx, tracing.Options{
			ServiceName:  cfg.Service.Name,
			OTLPEndpoint: cfg.OTEL.Endpoint,
			Sampling:     cfg.OTEL.SamplingRatio,
		})
		if err != nil {
			return fmt.Errorf("init tracer: %w", err)
		}
		logger.Info().Str("endpoint", cfg.OTEL.Endpoint).Msg("OTEL tracer initialized")
	} else {
		shutdownTracer = func() {}
		logger.Info().Msg("OTEL tracing disabled")
	}
	tracer := otel.GetTracerProvider().Tracer(cfg.Service.Name)

	metricsRegistry := metrics.New(cfg.Service.Name)

	primaryPool, err := infrastructure.NewPrimaryPool(ctx, cfg.Database.Primary, logger)
	if err != nil {
		return fmt.Errorf("init primary db: %w", err)
	}

	replicaPool, err := infrastructure.NewReplicaPool(ctx, cfg.Database.Replica, logger)
	if err != nil {
		return fmt.Errorf("init replica db: %w", err)
	}

	var tokenMaker token.Maker
	switch cfg.Token.Type {
	case "paseto":
		tokenMaker, err = token.NewPASETOMaker(cfg.Token.Secret)
	default:
		tokenMaker, err = token.NewJWTMaker(cfg.Token.Secret)
	}
	if err != nil {
		return fmt.Errorf("init token maker: %w", err)
	}

	store := postgres.New(primaryPool, replicaPool)
	hasher := password.NewHasher(password.DefaultConfig())

	redisClient, err := infrastructure.NewRedis(ctx, cfg.Redis, logger)
	if err != nil {
		return fmt.Errorf("init redis: %w", err)
	}

	jwtSvc := jwt.NewService(cfg.Service.Name, []string{cfg.Service.Name})
	sessionStore := session.NewStore(redisClient, 8*time.Hour)
	refreshTokenStore := session.NewRefreshTokenStore(redisClient)
	blacklistStore := session.NewBlacklistStore(redisClient)
	sessionSvc := session.NewService(sessionStore, refreshTokenStore, blacklistStore, jwtSvc, 8*time.Hour)

	svc := service.New(store, tracer, logger)

	cfgForServer := *cfg
	cfgForServer.API.Metrics.Enabled = false

	deps := rest.ServerDeps{
		Service:     svc,
		DB:          primaryPool,
		Logger:      logger,
		Tracer:      tracer,
		Metrics:     metricsRegistry,
		AuditLogger: auditLogger,
		TokenMaker:  tokenMaker,
		Config:      &cfgForServer,
	}
	server := rest.NewServer(deps)

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	if cfg.API.Metrics.Enabled {
		metricsPath := cfg.API.Metrics.Path
		if metricsPath == "" {
			metricsPath = "/metrics"
		}
		r.GET(metricsPath, gin.WrapH(promhttp.HandlerFor(
			metricsRegistry.Prometheus(),
			promhttp.HandlerOpts{EnableOpenMetrics: true},
		)))
	}

	authHandler := rest.NewAuthHandler(store, hasher, sessionSvc, tokenMaker)
	authHandler.RegisterRoutes(r)
	server.RegisterRoutes(r, deps)

	sm := shutdown.New(10*time.Second, logger)
	sm.Register("tracer", func(ctx context.Context) error {
		shutdownTracer()
		return nil
	})
	sm.Register("redis", func(ctx context.Context) error {
		return redisClient.Close()
	})
	sm.Register("db-replica", func(ctx context.Context) error {
		if replicaPool != nil {
			replicaPool.Close()
		}
		return nil
	})
	sm.Register("db-primary", func(ctx context.Context) error {
		primaryPool.Close()
		return nil
	})

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	httpServer := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	serverErr := make(chan error, 1)
	go func() {
		logger.Info().Str("addr", addr).Msg("HTTP server listening")
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	select {
	case err := <-serverErr:
		return fmt.Errorf("http server: %w", err)
	case <-ctx.Done():
		logger.Info().Msg("shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error().Err(err).Msg("HTTP server forced shutdown")
	}

	if err := sm.Wait(shutdownCtx); err != nil {
		logger.Error().Err(err).Msg("shutdown manager error")
	}

	logger.Info().Msg("server stopped")
	return nil
}

func initCasbin(ctx context.Context, cfg config.CasbinConfig, logger zerolog.Logger) {
	const op = "cmd.initCasbin"

	if cfg.ModelPath == "" {
		logger.Warn().Str("op", op).Msg("casbin model_path not set, RBAC disabled")
		return
	}

	logger.Info().Str("op", op).Str("model", cfg.ModelPath).Msg("casbin config loaded")
}

var _ func(context.Context, config.CasbinConfig, zerolog.Logger) = initCasbin

func initGRPC(ctx context.Context, cfg *config.ExternalConfig, tracer any, logger zerolog.Logger) (*grpc.ClientConn, error) {
	const op = "cmd.initGRPC"

	if cfg.GRPC.Address == "" {
		logger.Info().Str("op", op).Msg("grpc disabled")
		return nil, nil
	}

	conn, err := grpc.NewClient(
		cfg.GRPC.Address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	logger.Info().Str("op", op).Str("addr", cfg.GRPC.Address).Msg("grpc client initialized")
	return conn, nil
}
