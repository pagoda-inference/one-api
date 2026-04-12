package monitor

import (
	"fmt"
	"sync"
	"time"

	"github.com/pagoda-inference/one-api/common/config"
	"github.com/pagoda-inference/one-api/common/logger"
	"github.com/pagoda-inference/one-api/model"
)

// State represents the circuit breaker state
type State int

const (
	StateClosed State = iota   // Normal operation, requests pass through
	StateOpen                  // Failing, requests are blocked
	StateHalfOpen              // Testing if recovered, limited requests allowed
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	mu               sync.RWMutex
	state            State
	failureCount     int
	successCount     int
	failureThreshold int
	successThreshold int
	timeout          time.Duration
	lastFailureTime  time.Time
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker() *CircuitBreaker {
	return &CircuitBreaker{
		state:            StateClosed,
		failureThreshold: config.CircuitBreakerThreshold,
		successThreshold: config.CircuitBreakerSuccessThreshold,
		timeout:          time.Duration(config.CircuitBreakerTimeout) * time.Second,
	}
}

// AllowRequest checks if a request should be allowed
func (cb *CircuitBreaker) AllowRequest() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	switch cb.state {
	case StateClosed:
		return true
	case StateOpen:
		// Check if timeout has passed to transition to half-open
		if time.Since(cb.lastFailureTime) > time.Duration(config.CircuitBreakerTimeout)*time.Second {
			return true // Allow one request to test
		}
		return false
	case StateHalfOpen:
		return true // Allow requests in half-open state for testing
	}
	return false
}

// RecordSuccess records a successful request
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failureCount = 0

	if cb.state == StateHalfOpen {
		cb.successCount++
		if cb.successCount >= config.CircuitBreakerSuccessThreshold {
			cb.state = StateClosed
			cb.successCount = 0
			logger.SysLog("circuit breaker: state changed to closed (recovered)")
		}
	}
}

// RecordFailure records a failed request
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failureCount++
	cb.lastFailureTime = time.Now()

	if cb.state == StateHalfOpen {
		// Failure in half-open state, go back to open
		cb.state = StateOpen
		cb.successCount = 0
		logger.SysLog("circuit breaker: state changed to open (probe failed)")
	} else if cb.failureCount >= config.CircuitBreakerThreshold {
		// Too many failures, open the circuit
		cb.state = StateOpen
		logger.SysLog(fmt.Sprintf("circuit breaker: state changed to open (failures: %d)", cb.failureCount))
	}
}

// GetState returns the current state
func (cb *CircuitBreaker) GetState() State {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// GetStats returns circuit breaker statistics
func (cb *CircuitBreaker) GetStats() map[string]interface{} {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	return map[string]interface{}{
		"state":            cb.state.String(),
		"failure_count":    cb.failureCount,
		"success_count":    cb.successCount,
		"last_failure":     cb.lastFailureTime.Format(time.RFC3339),
		"failure_threshold": cb.failureThreshold,
		"success_threshold": cb.successThreshold,
		"timeout_seconds":   cb.timeout.Seconds(),
	}
}

// Reset resets the circuit breaker to closed state
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.state = StateClosed
	cb.failureCount = 0
	cb.successCount = 0
}

// Global circuit breaker manager
var (
	circuitBreakers = make(map[int]*CircuitBreaker)
	cbMu            sync.RWMutex
)

// GetCircuitBreaker returns the circuit breaker for a channel
func GetCircuitBreaker(channelId int) *CircuitBreaker {
	cbMu.Lock()
	defer cbMu.Unlock()

	if cb, ok := circuitBreakers[channelId]; ok {
		return cb
	}

	cb := NewCircuitBreaker()
	circuitBreakers[channelId] = cb
	return cb
}

// RemoveCircuitBreaker removes the circuit breaker for a channel
func RemoveCircuitBreaker(channelId int) {
	cbMu.Lock()
	defer cbMu.Unlock()

	delete(circuitBreakers, channelId)
}

// GetAllCircuitBreakerStats returns stats for all circuit breakers
func GetAllCircuitBreakerStats() map[int]map[string]interface{} {
	cbMu.RLock()
	defer cbMu.RUnlock()

	stats := make(map[int]map[string]interface{})
	for id, cb := range circuitBreakers {
		stats[id] = cb.GetStats()
	}
	return stats
}

// RecordChannelFailure records a failure for a channel and handles auto-disable
func RecordChannelFailure(channelId int, channelName string, errMsg string) {
	cb := GetCircuitBreaker(channelId)
	cb.RecordFailure()

	// Also record in metric system
	Emit(channelId, false)

	// Check if we should disable the channel based on errors
	// This is handled by the existing monitor.ShouldDisableChannel logic
	logger.SysLog(fmt.Sprintf("channel #%d failure recorded: %s (cb state: %s)", channelId, errMsg, cb.GetState().String()))
}

// RecordChannelSuccess records a success for a channel
func RecordChannelSuccess(channelId int) {
	cb := GetCircuitBreaker(channelId)
	cb.RecordSuccess()

	// Also record in metric system
	Emit(channelId, true)
}

// IsChannelAvailable checks if a channel is available (circuit breaker allows requests)
func IsChannelAvailable(channelId int) bool {
	cb := GetCircuitBreaker(channelId)

	// First check if circuit breaker allows
	if !cb.AllowRequest() {
		return false
	}

	// Also check if channel is enabled in database
	channel, err := model.GetChannelById(channelId, false)
	if err != nil {
		return false
	}

	return channel.Status == model.ChannelStatusEnabled
}

// TryRecoverChannel attempts to recover a channel if it was auto-disabled
func TryRecoverChannel(channelId int) {
	channel, err := model.GetChannelById(channelId, true)
	if err != nil {
		return
	}

	// Only try to recover auto-disabled channels
	if channel.Status != model.ChannelStatusAutoDisabled {
		return
	}

	cb := GetCircuitBreaker(channelId)
	if cb.GetState() == StateClosed {
		// Circuit breaker has recovered, re-enable the channel
		EnableChannel(channelId, channel.Name)
	}
}