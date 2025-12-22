package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	NATSMessagesPublished = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "nats_messages_published_total",
			Help: "Total number of NATS messages published",
		},
	)

	NATSPublishDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "nats_publish_duration_seconds",
			Help:    "Time spent publishing NATS messages",
			Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1},
		},
	)

	UpdateCheckDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "update_check_duration_seconds",
			Help:    "Time spent checking for updates",
			Buckets: []float64{.1, .5, 1, 2, 5, 10, 30, 60},
		},
		[]string{"package_manager"},
	)

	PendingUpdatesCount = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "pending_updates_count",
			Help: "Number of pending updates on this system",
		},
	)

	NATSConnectionStatus = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "nats_connection_connected",
			Help: "NATS connection status (1 = connected, 0 = disconnected)",
		},
	)

	NATSReconnects = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "nats_reconnects_total",
			Help: "Total number of NATS reconnection attempts",
		},
	)
)
