package core_metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

// Metrics holds all Prometheus metrics for the application.
// Labels are limited to prevent cardinality explosion:
// - No user IDs, full URLs, or arbitrary strings in labels
// - Use predefined values for method, status_class, endpoint_group
type Metrics struct {
	// HTTP server metrics
	HTTPRequestsTotal    *prometheus.CounterVec
	HTTPRequestDuration  *prometheus.HistogramVec
	HTTPRequestsInFlight prometheus.Gauge

	// Upstream (OpenWeather API) metrics
	UpstreamRequestsTotal   *prometheus.CounterVec
	UpstreamRequestDuration *prometheus.HistogramVec
	UpstreamErrors          *prometheus.CounterVec

	// Queue metrics (rate limiter)
	QueueDepth        prometheus.Gauge
	QueueWaitDuration prometheus.Histogram
	QueueRejections   prometheus.Counter
}

// NewMetrics creates and registers all application metrics with the provided registry.
// If registry is nil, uses the default Prometheus registry.
func NewMetrics(registry prometheus.Registerer) *Metrics {
	if registry == nil {
		registry = prometheus.DefaultRegisterer
	}

	m := &Metrics{
		// HTTP server metrics
		HTTPRequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_requests_total",
				Help: "Total number of HTTP requests processed by method and status class",
			},
			[]string{"method", "status_class"}, // 2xx, 3xx, 4xx, 5xx
		),

		HTTPRequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_request_duration_seconds",
				Help:    "HTTP request latency distribution in seconds",
				Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0},
			},
			[]string{"method", "status_class"},
		),

		HTTPRequestsInFlight: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "http_requests_in_flight",
				Help: "Current number of HTTP requests being processed",
			},
		),

		// Upstream metrics (OpenWeather API)
		UpstreamRequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "upstream_requests_total",
				Help: "Total number of upstream API requests by endpoint group and status",
			},
			[]string{"endpoint_group", "status"}, // endpoint_group: weather, geocoding; status: success, error
		),

		UpstreamRequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "upstream_request_duration_seconds",
				Help:    "Upstream API request latency distribution in seconds",
				Buckets: []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0},
			},
			[]string{"endpoint_group"},
		),

		UpstreamErrors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "upstream_errors_total",
				Help: "Total number of upstream API errors by type",
			},
			[]string{"endpoint_group", "error_type"}, // error_type: timeout, auth, rate_limit, server_error, network
		),

		// Queue metrics (rate limiter)
		QueueDepth: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "queue_depth",
				Help: "Current number of requests waiting in the rate limiter queue",
			},
		),

		QueueWaitDuration: prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "queue_wait_duration_seconds",
				Help:    "Time spent waiting in the rate limiter queue",
				Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0},
			},
		),

		QueueRejections: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "queue_rejections_total",
				Help: "Total number of requests rejected due to full queue",
			},
		),
	}

	// Register all metrics
	registry.MustRegister(
		m.HTTPRequestsTotal,
		m.HTTPRequestDuration,
		m.HTTPRequestsInFlight,
		m.UpstreamRequestsTotal,
		m.UpstreamRequestDuration,
		m.UpstreamErrors,
		m.QueueDepth,
		m.QueueWaitDuration,
		m.QueueRejections,
	)

	// Register Go runtime metrics (memory, goroutines, GC)
	registry.MustRegister(collectors.NewGoCollector())
	// Register process metrics (CPU, memory, file descriptors)
	registry.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))

	return m
}

// StatusClass converts HTTP status code to a class for low-cardinality labels
func StatusClass(code int) string {
	switch {
	case code >= 200 && code < 300:
		return "2xx"
	case code >= 300 && code < 400:
		return "3xx"
	case code >= 400 && code < 500:
		return "4xx"
	case code >= 500 && code < 600:
		return "5xx"
	default:
		return "unknown"
	}
}
