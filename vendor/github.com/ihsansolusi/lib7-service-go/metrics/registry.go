package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

// Registry wraps a custom Prometheus registry and exposes the standard HTTP
// metrics that every service should collect out-of-the-box.
//
// Use the fields directly in middleware to record observations, and pass
// Prometheus() to promhttp.HandlerFor to serve the /metrics endpoint.
type Registry struct {
	reg *prometheus.Registry

	// Standard HTTP metrics — populated by middleware.
	RequestsTotal    *prometheus.CounterVec
	RequestDuration  *prometheus.HistogramVec
	RequestsInFlight prometheus.Gauge
	ResponseSize     *prometheus.HistogramVec
}

// New creates a Registry with standard HTTP and Go runtime metrics pre-registered.
// namespace is prepended to every metric name (e.g. "myservice").
func New(namespace string) *Registry {
	reg := prometheus.NewRegistry()

	// Go runtime and process metrics.
	reg.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)

	r := &Registry{reg: reg}

	r.RequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "http_requests_total",
			Help:      "Total HTTP requests partitioned by method, path, and status code.",
		},
		[]string{"method", "path", "status"},
	)

	r.RequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "http_request_duration_seconds",
			Help:      "HTTP request latency in seconds.",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	r.RequestsInFlight = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "http_requests_in_flight",
			Help:      "Current number of HTTP requests being processed.",
		},
	)

	r.ResponseSize = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "http_response_size_bytes",
			Help:      "HTTP response body size in bytes.",
			Buckets:   prometheus.ExponentialBuckets(100, 10, 7),
		},
		[]string{"method", "path"},
	)

	reg.MustRegister(
		r.RequestsTotal,
		r.RequestDuration,
		r.RequestsInFlight,
		r.ResponseSize,
	)

	return r
}

// Prometheus returns the underlying *prometheus.Registry for use with
// promhttp.HandlerFor to serve the /metrics scrape endpoint.
func (r *Registry) Prometheus() *prometheus.Registry {
	return r.reg
}
