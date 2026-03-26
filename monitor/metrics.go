package monitor

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/middleware"
	"github.com/songquanpeng/one-api/model"
)

// MetricType represents the type of metric
type MetricType int

const (
	MetricTypeCounter MetricType = iota
	MetricTypeGauge
	MetricTypeHistogram
)

// Metric represents a single metric
type Metric struct {
	Name        string
	Type        MetricType
	Value       float64
	Labels      map[string]string
	Description string
}

// MetricsCollector collects and exposes metrics
type MetricsCollector struct {
	mu sync.RWMutex

	// Counters
	requestTotal        map[string]int64 // key: method_path_status
	requestErrors       map[string]int64
	tokenConsumption    map[string]int64 // key: model
	quotaConsumption    map[string]int64 // key: user_id

	// Gauges
	activeConnections   int64
	queueSize           int64
	channelConcurrency  map[int]int64

	// Histograms (simplified as averages)
	latencySum          map[string]int64
	latencyCount        map[string]int64

	// Timestamps
	startTime           time.Time
	lastCollectTime     time.Time
}

// Global metrics collector
var globalCollector *MetricsCollector
var collectorOnce sync.Once

// GetMetricsCollector returns the global metrics collector
func GetMetricsCollector() *MetricsCollector {
	collectorOnce.Do(func() {
		globalCollector = &MetricsCollector{
			requestTotal:       make(map[string]int64),
			requestErrors:      make(map[string]int64),
			tokenConsumption:   make(map[string]int64),
			quotaConsumption:   make(map[string]int64),
			channelConcurrency: make(map[int]int64),
			latencySum:         make(map[string]int64),
			latencyCount:       make(map[string]int64),
			startTime:          time.Now(),
			lastCollectTime:    time.Now(),
		}
	})
	return globalCollector
}

// RecordRequest records a request metric
func (m *MetricsCollector) RecordRequest(method, path, status string, latencyMs int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s_%s_%s", method, sanitizePath(path), status)
	m.requestTotal[key]++

	// Record latency
	m.latencySum[path] += latencyMs
	m.latencyCount[path]++

	m.lastCollectTime = time.Now()
}

// RecordError records an error metric
func (m *MetricsCollector) RecordError(errorType, path string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s_%s", errorType, sanitizePath(path))
	m.requestErrors[key]++
}

// RecordTokenConsumption records token usage
func (m *MetricsCollector) RecordTokenConsumption(model string, tokens int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.tokenConsumption[model] += tokens
}

// RecordQuotaConsumption records quota usage
func (m *MetricsCollector) RecordQuotaConsumption(userId int, quota int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.quotaConsumption[fmt.Sprintf("%d", userId)] += quota
}

// UpdateChannelConcurrency updates channel concurrency gauge
func (m *MetricsCollector) UpdateChannelConcurrency(channelId int, concurrency int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.channelConcurrency[channelId] = concurrency
}

// UpdateQueueSize updates the queue size gauge
func (m *MetricsCollector) UpdateQueueSize(size int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.queueSize = size
}

// Collect collects all metrics
func (m *MetricsCollector) Collect() []Metric {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var metrics []Metric

	// Uptime
	metrics = append(metrics, Metric{
		Name:        "oneapi_uptime_seconds",
		Type:        MetricTypeGauge,
		Value:       time.Since(m.startTime).Seconds(),
		Description: "Time since the server started",
	})

	// Request totals
	for key, count := range m.requestTotal {
		parts := strings.SplitN(key, "_", 3)
		metrics = append(metrics, Metric{
			Name:        "oneapi_requests_total",
			Type:        MetricTypeCounter,
			Value:       float64(count),
			Labels:      map[string]string{"method": parts[0], "path": parts[1], "status": parts[2]},
			Description: "Total number of requests",
		})
	}

	// Request errors
	for key, count := range m.requestErrors {
		parts := strings.SplitN(key, "_", 2)
		metrics = append(metrics, Metric{
			Name:        "oneapi_request_errors_total",
			Type:        MetricTypeCounter,
			Value:       float64(count),
			Labels:      map[string]string{"type": parts[0], "path": parts[1]},
			Description: "Total number of request errors",
		})
	}

	// Token consumption
	for model, tokens := range m.tokenConsumption {
		metrics = append(metrics, Metric{
			Name:        "oneapi_token_consumption_total",
			Type:        MetricTypeCounter,
			Value:       float64(tokens),
			Labels:      map[string]string{"model": model},
			Description: "Total token consumption",
		})
	}

	// Queue size
	metrics = append(metrics, Metric{
		Name:        "oneapi_queue_size",
		Type:        MetricTypeGauge,
		Value:       float64(m.queueSize),
		Description: "Current request queue size",
	})

	// Channel concurrency
	for channelId, concurrency := range m.channelConcurrency {
		metrics = append(metrics, Metric{
			Name:        "oneapi_channel_concurrency",
			Type:        MetricTypeGauge,
			Value:       float64(concurrency),
			Labels:      map[string]string{"channel_id": fmt.Sprintf("%d", channelId)},
			Description: "Current channel concurrency",
		})
	}

	// Latency averages
	for path, sum := range m.latencySum {
		count := m.latencyCount[path]
		if count > 0 {
			metrics = append(metrics, Metric{
				Name:        "oneapi_request_latency_ms",
				Type:        MetricTypeGauge,
				Value:       float64(sum) / float64(count),
				Labels:      map[string]string{"path": path},
				Description: "Average request latency in milliseconds",
			})
		}
	}

	return metrics
}

// sanitizePath sanitizes a path for use as a metric label
func sanitizePath(path string) string {
	// Replace dynamic segments with placeholders
	path = strings.ReplaceAll(path, "/", "_")
	path = strings.TrimPrefix(path, "_")
	if path == "" {
		path = "root"
	}
	return path
}

// ExportPrometheus exports metrics in Prometheus format
func ExportPrometheus() string {
	collector := GetMetricsCollector()
	metrics := collector.Collect()

	var builder strings.Builder

	for _, m := range metrics {
		// Write HELP
		builder.WriteString(fmt.Sprintf("# HELP %s %s\n", m.Name, m.Description))

		// Write TYPE
		var typeStr string
		switch m.Type {
		case MetricTypeCounter:
			typeStr = "counter"
		case MetricTypeGauge:
			typeStr = "gauge"
		case MetricTypeHistogram:
			typeStr = "histogram"
		}
		builder.WriteString(fmt.Sprintf("# TYPE %s %s\n", m.Name, typeStr))

		// Write metric value
		if len(m.Labels) > 0 {
			labels := make([]string, 0, len(m.Labels))
			for k, v := range m.Labels {
				labels = append(labels, fmt.Sprintf(`%s="%s"`, k, v))
			}
			builder.WriteString(fmt.Sprintf("%s{%s} %.2f\n", m.Name, strings.Join(labels, ","), m.Value))
		} else {
			builder.WriteString(fmt.Sprintf("%s %.2f\n", m.Name, m.Value))
		}
	}

	return builder.String()
}

// GetSystemStats returns system-level statistics
func GetSystemStats() map[string]interface{} {
	collector := GetMetricsCollector()

	// Get queue stats
	queueStats := middleware.GetQueueStats()

	// Get scheduler stats
	scheduler := GetScheduler()
	channelStats := scheduler.GetAllChannelStats()

	// Get circuit breaker stats
	cbStats := GetAllCircuitBreakerStats()

	return map[string]interface{}{
		"uptime_seconds":    time.Since(collector.startTime).Seconds(),
		"start_time":        collector.startTime.Unix(),
		"last_collect_time": collector.lastCollectTime.Unix(),
		"queue": map[string]interface{}{
			"current_size":    queueStats.CurrentQueueSize,
			"total_requests":  queueStats.TotalRequests,
			"rejected_count":  queueStats.RejectedRequests,
			"max_capacity":    queueStats.MaxCapacity,
			"utilization":     middleware.GetQueueUtilization(),
		},
		"channels":    channelStats,
		"circuit_breakers": cbStats,
		"config": map[string]interface{}{
			"max_concurrent_requests":  config.MaxConcurrentRequests,
			"health_check_interval":    config.HealthCheckInterval,
			"circuit_breaker_threshold": config.CircuitBreakerThreshold,
			"weighted_lb_enabled":      config.EnableWeightedLoadBalancing,
		},
	}
}

// RecordRequest is a convenience function for recording requests
func RecordRequest(method, path, status string, latencyMs int64) {
	GetMetricsCollector().RecordRequest(method, path, status, latencyMs)
}

// RecordError is a convenience function for recording errors
func RecordError(errorType, path string) {
	GetMetricsCollector().RecordError(errorType, path)
}

// RecordTokenConsumption is a convenience function for recording token usage
func RecordTokenConsumption(model string, tokens int64) {
	GetMetricsCollector().RecordTokenConsumption(model, tokens)
}