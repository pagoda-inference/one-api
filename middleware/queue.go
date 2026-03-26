package middleware

import (
	"net/http"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pagoda-inference/one-api/common/config"
	"github.com/pagoda-inference/one-api/common/logger"
)

// RequestQueue implements a semaphore-based request queue for backpressure control
type RequestQueue struct {
	semaphore chan struct{}
	timeout   time.Duration
	stats     QueueStats
}

// QueueStats holds statistics for the request queue
type QueueStats struct {
	CurrentQueueSize int64
	TotalRequests    int64
	RejectedRequests int64
	MaxCapacity      int64
}

// NewRequestQueue creates a new request queue with the specified capacity
func NewRequestQueue(maxConcurrent int, timeout time.Duration) *RequestQueue {
	return &RequestQueue{
		semaphore: make(chan struct{}, maxConcurrent),
		timeout:   timeout,
		stats: QueueStats{
			MaxCapacity: int64(maxConcurrent),
		},
	}
}

// Acquire attempts to acquire a slot in the queue
// Returns true if successful, false if queue is full and request should be rejected
func (q *RequestQueue) Acquire(c *gin.Context) bool {
	select {
	case q.semaphore <- struct{}{}:
		atomic.AddInt64(&q.stats.CurrentQueueSize, 1)
		atomic.AddInt64(&q.stats.TotalRequests, 1)
		return true
	case <-time.After(q.timeout):
		atomic.AddInt64(&q.stats.RejectedRequests, 1)
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": gin.H{
				"message": "Service temporarily unavailable, please retry later",
				"type":    "server_busy",
				"code":    "queue_full",
			},
		})
		c.Abort()
		return false
	}
}

// Release releases a slot in the queue
func (q *RequestQueue) Release() {
	<-q.semaphore
	atomic.AddInt64(&q.stats.CurrentQueueSize, -1)
}

// GetStats returns the current queue statistics
func (q *RequestQueue) GetStats() QueueStats {
	return QueueStats{
		CurrentQueueSize: atomic.LoadInt64(&q.stats.CurrentQueueSize),
		TotalRequests:    atomic.LoadInt64(&q.stats.TotalRequests),
		RejectedRequests: atomic.LoadInt64(&q.stats.RejectedRequests),
		MaxCapacity:      q.stats.MaxCapacity,
	}
}

// GetUtilization returns the current queue utilization (0.0 - 1.0)
func (q *RequestQueue) GetUtilization() float64 {
	current := atomic.LoadInt64(&q.stats.CurrentQueueSize)
	max := q.stats.MaxCapacity
	if max == 0 {
		return 0
	}
	return float64(current) / float64(max)
}

// Global request queue instance
var globalRequestQueue *RequestQueue

// InitRequestQueue initializes the global request queue
func InitRequestQueue() {
	maxConcurrent := config.MaxConcurrentRequests
	if maxConcurrent <= 0 {
		maxConcurrent = 1000
	}

	timeout := time.Duration(config.RequestQueueTimeout) * time.Second
	if timeout <= 0 {
		timeout = 5 * time.Second
	}

	globalRequestQueue = NewRequestQueue(maxConcurrent, timeout)
	logger.SysLog("request queue initialized: max_concurrent=%d, timeout=%v", maxConcurrent, timeout)
}

// GetRequestQueue returns the global request queue instance
func GetRequestQueue() *RequestQueue {
	if globalRequestQueue == nil {
		InitRequestQueue()
	}
	return globalRequestQueue
}

// QueueMiddleware returns a Gin middleware for request queue management
func QueueMiddleware() gin.HandlerFunc {
	queue := GetRequestQueue()

	return func(c *gin.Context) {
		// Skip queue for health check and metrics endpoints
		path := c.Request.URL.Path
		if path == "/health" || path == "/metrics" || path == "/api/status" {
			c.Next()
			return
		}

		// Try to acquire a slot
		if !queue.Acquire(c) {
			// Request was rejected due to queue being full
			logger.Warnf(c.Request.Context(), "request rejected due to queue full: %s %s", c.Request.Method, path)
			return
		}

		// Ensure release on completion
		defer queue.Release()

		// Process request
		c.Next()
	}
}

// GetQueueStats returns the current queue statistics
func GetQueueStats() QueueStats {
	if globalRequestQueue == nil {
		return QueueStats{}
	}
	return globalRequestQueue.GetStats()
}

// GetQueueUtilization returns the current queue utilization
func GetQueueUtilization() float64 {
	if globalRequestQueue == nil {
		return 0
	}
	return globalRequestQueue.GetUtilization()
}