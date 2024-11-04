// internal/metrics/metrics.go
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	EventsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "lab_events_total",
			Help: "Total number of lab events",
		},
		[]string{"course", "lab", "event_type"},
	)

	LabScoreHistogram = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "lab_score",
			Help:    "Distribution of lab scores",
			Buckets: prometheus.LinearBuckets(0, 5, 10),
		},
		[]string{"course", "lab"},
	)

	APIRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "api_request_duration_seconds",
			Help:    "API request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"path", "method", "status"},
	)
)
