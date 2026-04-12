package monitor

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/pagoda-inference/one-api/common/client"
	"github.com/pagoda-inference/one-api/common/config"
	"github.com/pagoda-inference/one-api/common/logger"
	"github.com/pagoda-inference/one-api/model"
	"github.com/pagoda-inference/one-api/relay/channeltype"
)

// ChannelHealthStatus represents the health status of a channel
type ChannelHealthStatus struct {
	ChannelId      int
	ChannelName    string
	IsHealthy      bool
	LastCheckTime  time.Time
	ResponseTime   int64  // in milliseconds
	FailureCount   int
	SuccessCount   int
	LastError      string
}

// HealthChecker performs periodic health checks on channels
type HealthChecker struct {
	interval       time.Duration
	timeout        time.Duration
	checkPath      string
	stopChan       chan struct{}
	statusMap      map[int]*ChannelHealthStatus
	statusMu       sync.RWMutex
	wg             sync.WaitGroup
}

var globalHealthChecker *HealthChecker
var healthCheckerOnce sync.Once

// GetHealthChecker returns the global health checker instance
func GetHealthChecker() *HealthChecker {
	healthCheckerOnce.Do(func() {
		globalHealthChecker = &HealthChecker{
			interval:  time.Duration(config.HealthCheckInterval) * time.Second,
			timeout:   10 * time.Second,
			checkPath: "/v1/models",
			stopChan:  make(chan struct{}),
			statusMap: make(map[int]*ChannelHealthStatus),
		}
	})
	return globalHealthChecker
}

// Start begins the health check loop
func (hc *HealthChecker) Start() {
	logger.SysLog(fmt.Sprintf("health checker started with interval: %v", hc.interval))

	ticker := time.NewTicker(hc.interval)
	defer ticker.Stop()

	// Use dynamic interval - re-read config each tick
	currentInterval := hc.interval
	for {
		select {
		case <-ticker.C:
			// Re-check interval from config in case it changed
			newInterval := time.Duration(config.HealthCheckInterval) * time.Second
			if newInterval != currentInterval {
				currentInterval = newInterval
				ticker.Reset(newInterval)
				logger.SysLog(fmt.Sprintf("health checker interval updated to: %v", newInterval))
			}
			hc.checkAllChannels()
		case <-hc.stopChan:
			logger.SysLog("health checker stopped")
			return
		}
	}
}

// Stop stops the health checker
func (hc *HealthChecker) Stop() {
	close(hc.stopChan)
	hc.wg.Wait()
}

// checkAllChannels performs health checks on all enabled channels
func (hc *HealthChecker) checkAllChannels() {
	channels, err := hc.getEnabledChannels()
	if err != nil {
		logger.SysError("failed to get enabled channels: " + err.Error())
		return
	}

	if len(channels) == 0 {
		return
	}

	logger.Debugf(context.Background(), "health checking %d channels", len(channels))

	// Check channels in parallel with a limit
	semaphore := make(chan struct{}, 10) // Max 10 concurrent checks

	for _, channel := range channels {
		hc.wg.Add(1)
		go func(ch *model.Channel) {
			defer hc.wg.Done()

			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			hc.checkChannel(ch)
		}(channel)
	}

	hc.wg.Wait()
}

// getEnabledChannels retrieves all enabled channels from database
func (hc *HealthChecker) getEnabledChannels() ([]*model.Channel, error) {
	var channels []*model.Channel
	err := model.DB.Where("status = ?", model.ChannelStatusEnabled).Find(&channels).Error
	return channels, err
}

// checkChannel performs a health check on a single channel
func (hc *HealthChecker) checkChannel(channel *model.Channel) {
	ctx, cancel := context.WithTimeout(context.Background(), hc.timeout)
	defer cancel()

	// Build the health check URL
	baseURL := channel.GetBaseURL()
	if baseURL == "" {
		baseURL = "https://api.openai.com"
	}
	// For OpenAI Compatible, strip /v1 from checkPath to avoid double /v1
	checkPath := hc.checkPath
	if channel.Type == channeltype.OpenAICompatible {
		checkPath = strings.TrimPrefix(checkPath, "/v1")
	}
	checkURL := fmt.Sprintf("%s%s", baseURL, checkPath)

	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", checkURL, nil)
	if err != nil {
		hc.recordCheckResult(channel.Id, channel.Name, false, 0, err.Error())
		return
	}

	// Set authorization header
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", channel.Key))

	// Perform the request
	start := time.Now()
	resp, err := client.ImpatientHTTPClient.Do(req)
	latency := time.Since(start).Milliseconds()

	if err != nil {
		hc.recordCheckResult(channel.Id, channel.Name, false, latency, err.Error())
		return
	}
	defer resp.Body.Close()

	// Check response status
	healthy := resp.StatusCode < 500
	var errMsg string
	if !healthy {
		errMsg = fmt.Sprintf("status code: %d", resp.StatusCode)
	}

	hc.recordCheckResult(channel.Id, channel.Name, healthy, latency, errMsg)
}

// recordCheckResult records the health check result and updates circuit breaker
func (hc *HealthChecker) recordCheckResult(channelId int, channelName string, healthy bool, latency int64, errMsg string) {
	hc.statusMu.Lock()

	status, exists := hc.statusMap[channelId]
	if !exists {
		status = &ChannelHealthStatus{
			ChannelId:   channelId,
			ChannelName: channelName,
		}
		hc.statusMap[channelId] = status
	}

	status.LastCheckTime = time.Now()
	status.ResponseTime = latency
	status.LastError = errMsg

	if healthy {
		status.IsHealthy = true
		status.SuccessCount++
		status.FailureCount = 0
		RecordChannelSuccess(channelId)

		// Try to recover the channel if it was auto-disabled
		TryRecoverChannel(channelId)
	} else {
		status.IsHealthy = false
		status.FailureCount++
		RecordChannelFailure(channelId, channelName, errMsg)

		// Check if we should disable the channel
		hc.maybeDisableChannel(channelId, channelName)
	}

	hc.statusMu.Unlock()

	// Log the result
	if healthy {
		logger.Debugf(context.Background(), "health check passed for channel #%d (%s), latency: %dms",
			channelId, channelName, latency)
	} else {
		logger.SysLog(fmt.Sprintf("health check failed for channel #%d (%s): %s",
			channelId, channelName, errMsg))
	}
}

// maybeDisableChannel checks if a channel should be disabled based on health check failures
func (hc *HealthChecker) maybeDisableChannel(channelId int, channelName string) {
	cb := GetCircuitBreaker(channelId)

	// If circuit breaker is open, disable the channel
	if cb.GetState() == StateOpen {
		channel, err := model.GetChannelById(channelId, false)
		if err != nil {
			return
		}

		// Only disable if it's currently enabled
		if channel.Status == model.ChannelStatusEnabled {
			reason := fmt.Sprintf("health check failures exceeded threshold (%d)", config.CircuitBreakerThreshold)
			DisableChannel(channelId, channelName, reason)
		}
	}
}

// GetChannelHealthStatus returns the health status of a specific channel
func (hc *HealthChecker) GetChannelHealthStatus(channelId int) *ChannelHealthStatus {
	hc.statusMu.RLock()
	defer hc.statusMu.RUnlock()

	if status, ok := hc.statusMap[channelId]; ok {
		return status
	}
	return nil
}

// GetAllChannelHealthStatus returns health status for all channels
func (hc *HealthChecker) GetAllChannelHealthStatus() map[int]*ChannelHealthStatus {
	hc.statusMu.RLock()
	defer hc.statusMu.RUnlock()

	result := make(map[int]*ChannelHealthStatus)
	for k, v := range hc.statusMap {
		result[k] = v
	}
	return result
}

// CheckChannelNow performs an immediate health check on a specific channel
func (hc *HealthChecker) CheckChannelNow(channelId int) error {
	channel, err := model.GetChannelById(channelId, true)
	if err != nil {
		return err
	}

	hc.checkChannel(channel)
	return nil
}

// StartHealthChecker initializes and starts the global health checker
func StartHealthChecker() {
	if config.HealthCheckInterval <= 0 {
		logger.SysLog("health checker disabled (HEALTH_CHECK_INTERVAL <= 0)")
		return
	}

	hc := GetHealthChecker()
	go hc.Start()
}

// GetHealthStatus returns the health status for a channel
func GetHealthStatus(channelId int) *ChannelHealthStatus {
	hc := GetHealthChecker()
	return hc.GetChannelHealthStatus(channelId)
}

// TriggerHealthCheck triggers an immediate health check for a channel
func TriggerHealthCheck(channelId int) error {
	hc := GetHealthChecker()
	return hc.CheckChannelNow(channelId)
}