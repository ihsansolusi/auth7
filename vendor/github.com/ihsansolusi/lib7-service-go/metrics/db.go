package metrics

import "github.com/prometheus/client_golang/prometheus"

// DBMetrics exposes connection pool health and query latency for a PostgreSQL
// pool. Populate the gauges periodically from pgxpool.Stat() and observe query
// durations in the store layer.
type DBMetrics struct {
	// ConnectionsAcquired is the current number of acquired (in-use) connections.
	ConnectionsAcquired *prometheus.GaugeVec
	// ConnectionsIdle is the current number of idle connections in the pool.
	ConnectionsIdle *prometheus.GaugeVec
	// ConnectionsTotal is the total pool capacity (max connections configured).
	ConnectionsTotal *prometheus.GaugeVec
	// QueryDuration observes per-operation query latency.
	QueryDuration *prometheus.HistogramVec
}

// NewDB creates DBMetrics and registers them with r.
// namespace is prepended to every metric name.
// The "pool" label lets a service report metrics for multiple pools.
func NewDB(r *Registry, namespace string) *DBMetrics {
	m := &DBMetrics{
		ConnectionsAcquired: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "db_connections_acquired",
				Help:      "Current number of acquired (in-use) DB connections.",
			},
			[]string{"pool"},
		),
		ConnectionsIdle: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "db_connections_idle",
				Help:      "Current number of idle DB connections in the pool.",
			},
			[]string{"pool"},
		),
		ConnectionsTotal: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "db_connections_total",
				Help:      "Total DB connection pool capacity (max connections).",
			},
			[]string{"pool"},
		),
		QueryDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "db_query_duration_seconds",
				Help:      "DB query duration in seconds partitioned by operation.",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"operation"},
		),
	}

	r.reg.MustRegister(
		m.ConnectionsAcquired,
		m.ConnectionsIdle,
		m.ConnectionsTotal,
		m.QueryDuration,
	)

	return m
}
