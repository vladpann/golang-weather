package core_metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestNewMetrics(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := NewMetrics(registry)

	if metrics == nil {
		t.Fatal("NewMetrics returned nil")
	}

	// Verify HTTP metrics are registered
	if metrics.HTTPRequestsTotal == nil {
		t.Error("HTTPRequestsTotal is nil")
	}
	if metrics.HTTPRequestDuration == nil {
		t.Error("HTTPRequestDuration is nil")
	}
	if metrics.HTTPRequestsInFlight == nil {
		t.Error("HTTPRequestsInFlight is nil")
	}

	// Verify upstream metrics are registered
	if metrics.UpstreamRequestsTotal == nil {
		t.Error("UpstreamRequestsTotal is nil")
	}
	if metrics.UpstreamRequestDuration == nil {
		t.Error("UpstreamRequestDuration is nil")
	}
	if metrics.UpstreamErrors == nil {
		t.Error("UpstreamErrors is nil")
	}

	// Verify queue metrics are registered
	if metrics.QueueDepth == nil {
		t.Error("QueueDepth is nil")
	}
	if metrics.QueueWaitDuration == nil {
		t.Error("QueueWaitDuration is nil")
	}
	if metrics.QueueRejections == nil {
		t.Error("QueueRejections is nil")
	}
}

func TestHTTPMetricsRecording(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := NewMetrics(registry)

	// Record some HTTP requests
	metrics.HTTPRequestsTotal.WithLabelValues("GET", "2xx").Inc()
	metrics.HTTPRequestsTotal.WithLabelValues("GET", "2xx").Inc()
	metrics.HTTPRequestsTotal.WithLabelValues("POST", "4xx").Inc()

	metrics.HTTPRequestDuration.WithLabelValues("GET", "2xx").Observe(0.1)
	metrics.HTTPRequestDuration.WithLabelValues("GET", "2xx").Observe(0.2)

	// Verify counter values
	count := testutil.ToFloat64(metrics.HTTPRequestsTotal.WithLabelValues("GET", "2xx"))
	if count != 2 {
		t.Errorf("Expected 2 GET/2xx requests, got %f", count)
	}

	count = testutil.ToFloat64(metrics.HTTPRequestsTotal.WithLabelValues("POST", "4xx"))
	if count != 1 {
		t.Errorf("Expected 1 POST/4xx request, got %f", count)
	}
}

func TestUpstreamMetricsRecording(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := NewMetrics(registry)

	// Record upstream requests
	metrics.UpstreamRequestsTotal.WithLabelValues("weather", "success").Inc()
	metrics.UpstreamRequestsTotal.WithLabelValues("weather", "success").Inc()
	metrics.UpstreamRequestsTotal.WithLabelValues("weather", "error").Inc()
	metrics.UpstreamRequestsTotal.WithLabelValues("geocoding", "success").Inc()

	metrics.UpstreamRequestDuration.WithLabelValues("weather").Observe(0.5)
	metrics.UpstreamErrors.WithLabelValues("weather", "timeout").Inc()

	// Verify values
	count := testutil.ToFloat64(metrics.UpstreamRequestsTotal.WithLabelValues("weather", "success"))
	if count != 2 {
		t.Errorf("Expected 2 successful weather requests, got %f", count)
	}

	count = testutil.ToFloat64(metrics.UpstreamRequestsTotal.WithLabelValues("weather", "error"))
	if count != 1 {
		t.Errorf("Expected 1 failed weather request, got %f", count)
	}

	count = testutil.ToFloat64(metrics.UpstreamErrors.WithLabelValues("weather", "timeout"))
	if count != 1 {
		t.Errorf("Expected 1 timeout error, got %f", count)
	}
}

func TestQueueMetricsRecording(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := NewMetrics(registry)

	// Record queue metrics
	metrics.QueueDepth.Set(5)
	metrics.QueueWaitDuration.Observe(0.1)
	metrics.QueueWaitDuration.Observe(0.2)
	metrics.QueueRejections.Inc()

	// Verify values
	depth := testutil.ToFloat64(metrics.QueueDepth)
	if depth != 5 {
		t.Errorf("Expected queue depth 5, got %f", depth)
	}

	rejections := testutil.ToFloat64(metrics.QueueRejections)
	if rejections != 1 {
		t.Errorf("Expected 1 rejection, got %f", rejections)
	}
}

func TestStatusClass(t *testing.T) {
	tests := []struct {
		code     int
		expected string
	}{
		{200, "2xx"},
		{201, "2xx"},
		{299, "2xx"},
		{301, "3xx"},
		{302, "3xx"},
		{400, "4xx"},
		{404, "4xx"},
		{499, "4xx"},
		{500, "5xx"},
		{503, "5xx"},
		{599, "5xx"},
		{100, "unknown"},
		{600, "unknown"},
	}

	for _, tt := range tests {
		result := StatusClass(tt.code)
		if result != tt.expected {
			t.Errorf("StatusClass(%d) = %s, want %s", tt.code, result, tt.expected)
		}
	}
}
