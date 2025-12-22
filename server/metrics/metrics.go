package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// HTTP Metrics
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "route", "status_code"},
	)

	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets, // [.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10]
		},
		[]string{"method", "route"},
	)

	// NATS Metrics
	NATSMessagesReceived = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nats_messages_received_total",
			Help: "Total number of NATS messages received",
		},
		[]string{"subject"},
	)

	NATSMessagesReceivedDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "nats_message_processing_duration_seconds",
			Help:    "Time spent processing NATS messages",
			Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1},
		},
		[]string{"subject"},
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

	// Storage Metrics
	StorageOperationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "storage_operation_duration_seconds",
			Help:    "Storage operation duration in seconds",
			Buckets: []float64{.0001, .0005, .001, .005, .01, .025, .05, .1},
		},
		[]string{"operation"},
	)

	StorageOperationErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "storage_operation_errors_total",
			Help: "Total number of storage operation errors",
		},
		[]string{"operation"},
	)

	// WebSocket Metrics
	WebSocketConnections = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "websocket_connections_active",
			Help: "Number of active WebSocket connections",
		},
	)

	WebSocketMessagesBroadcast = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "websocket_messages_broadcast_total",
			Help: "Total number of WebSocket messages broadcast",
		},
	)

	WebSocketBroadcastDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "websocket_broadcast_duration_seconds",
			Help:    "Time spent broadcasting WebSocket messages",
			Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5},
		},
	)

	// Business Metrics
	SystemsMonitored = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "systems_monitored_total",
			Help: "Total number of systems being monitored",
		},
	)

	SystemsWithUpdates = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "systems_with_updates_total",
			Help: "Number of systems with pending updates",
		},
	)

	TotalPendingUpdates = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "pending_updates_total",
			Help: "Total number of pending updates across all systems",
		},
	)
)
