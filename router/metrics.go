package router

import (
	"github.com/gin-gonic/gin"
	"github.com/pagoda-inference/one-api/common/config"
	"github.com/pagoda-inference/one-api/controller"
)

// SetMetricsRouter sets up the metrics and health check routes
func SetMetricsRouter(router *gin.Engine) {
	// Health check endpoint (no auth required)
	router.GET("/health", controller.GetHealth)
	router.GET("/api/health", controller.GetDetailedHealth)

	// Metrics endpoint (Prometheus format)
	if config.MetricsEnabled {
		metricsPath := config.MetricsPath
		if metricsPath == "" {
			metricsPath = "/metrics"
		}
		router.GET(metricsPath, controller.GetMetrics)
		router.GET("/api/metrics", controller.GetMetricsJSON)
	}
}