package service

import (
	"context"

	"go.opentelemetry.io/otel/trace"
	"github.com/rs/zerolog"
	"github.com/ihsansolusi/auth7/internal/store/postgres"
)

type Service struct {
	store   *postgres.Store
	tracer  trace.Tracer
	logger  zerolog.Logger
}

func New(s *postgres.Store, tracer trace.Tracer, logger zerolog.Logger) *Service {
	return &Service{
		store:  s,
		tracer: tracer,
		logger: logger,
	}
}

func (s *Service) tracerCtx(ctx context.Context, op string) context.Context {
	return ctx
}

var _ any = New
