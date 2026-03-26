package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/middleware"
	"github.com/songquanpeng/one-api/monitor"
)

// GetMetrics handles GET /metrics (Prometheus format)
func GetMetrics(c *gin.Context) {
	metrics := monitor.ExportPrometheus()
	c.Data(http.StatusOK, "text/plain; charset=utf-8", []byte(metrics))
}

// GetMetricsJSON handles GET /api/metrics (JSON format)
func GetMetricsJSON(c *gin.Context) {
	stats := monitor.GetSystemStats()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
	})
}

// GetHealth handles GET /health
func GetHealth(c *gin.Context) {
	// Basic health check
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
	})
}

// GetDetailedHealth handles GET /api/health
func GetDetailedHealth(c *gin.Context) {
	queueStats := middleware.GetQueueStats()
	utilization := middleware.GetQueueUtilization()

	health := gin.H{
		"status": "ok",
		"queue": gin.H{
			"current_size":   queueStats.CurrentQueueSize,
			"max_capacity":   queueStats.MaxCapacity,
			"utilization":    utilization,
			"total_requests": queueStats.TotalRequests,
			"rejected":       queueStats.RejectedRequests,
		},
	}

	// Check if queue is under pressure
	if utilization > 0.8 {
		health["status"] = "degraded"
		health["warning"] = "Queue utilization is high"
	}

	c.JSON(http.StatusOK, health)
}