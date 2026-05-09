// Package tracing provides OTEL tracer initialization for core7 services.
package tracing

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

// Options configures the OTEL tracer.
type Options struct {
	ServiceName  string
	OTLPEndpoint string  // e.g. "otel-collector:4317"
	Sampling     float64 // 1.0 dev, 0.1 prod
}

// InitTracer sets up the global OTEL TracerProvider with an OTLP gRPC exporter.
// The returned function must be called on shutdown to flush and close the exporter.
func InitTracer(ctx context.Context, opts Options) (func(), error) {
	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(opts.OTLPEndpoint),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("create OTLP exporter: %w", err)
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(opts.ServiceName),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("create resource: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(opts.Sampling)),
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	otel.SetTracerProvider(tp)

	return func() { //nolint:contextcheck
		_ = tp.Shutdown(context.Background())
	}, nil
}
