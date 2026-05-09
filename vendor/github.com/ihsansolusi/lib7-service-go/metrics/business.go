package metrics

import "github.com/prometheus/client_golang/prometheus"

// BusinessMetrics provides factory helpers for registering custom business
// metrics against the service's Prometheus registry.
//
// Example:
//
//	bm := metrics.NewBusiness(reg)
//	transfersTotal = bm.RegisterCounter(
//	    "transfers_total",
//	    "Total transfers by status and destination bank",
//	    []string{"status", "destination_bank"},
//	)
type BusinessMetrics struct {
	registry *prometheus.Registry
}

// NewBusiness creates a BusinessMetrics bound to the given Registry.
func NewBusiness(r *Registry) *BusinessMetrics {
	return &BusinessMetrics{registry: r.reg}
}

// RegisterCounter creates and registers a new CounterVec with the given name,
// help text, and label names.
func (m *BusinessMetrics) RegisterCounter(name, help string, labels []string) *prometheus.CounterVec {
	c := prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: name, Help: help},
		labels,
	)
	m.registry.MustRegister(c)
	return c
}

// RegisterHistogram creates and registers a new HistogramVec with the given
// name, help text, label names, and bucket boundaries.
func (m *BusinessMetrics) RegisterHistogram(name, help string, labels []string, buckets []float64) *prometheus.HistogramVec {
	h := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{Name: name, Help: help, Buckets: buckets},
		labels,
	)
	m.registry.MustRegister(h)
	return h
}

// RegisterGauge creates and registers a new GaugeVec with the given name,
// help text, and label names.
func (m *BusinessMetrics) RegisterGauge(name, help string, labels []string) *prometheus.GaugeVec {
	g := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{Name: name, Help: help},
		labels,
	)
	m.registry.MustRegister(g)
	return g
}
