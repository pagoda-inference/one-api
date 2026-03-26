package monitor

import (
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pagoda-inference/one-api/common/logger"
	"github.com/pagoda-inference/one-api/model"
)

// ChannelStats holds statistics for a channel used in load balancing
type ChannelStats struct {
	mu sync.RWMutex

	// Request statistics
	TotalRequests  int64
	SuccessCount   int64
	FailCount      int64

	// Latency statistics
	TotalLatency   int64 // in milliseconds
	AvgLatency     int64 // rolling average

	// Concurrency tracking
	CurrentConcurrency int32

	// Timestamps
	LastUsedTime int64
	LastUpdateTime int64
}

// NewChannelStats creates a new ChannelStats instance
func NewChannelStats() *ChannelStats {
	return &ChannelStats{
		LastUpdateTime: time.Now().Unix(),
	}
}

// RecordRequest records a request result and updates statistics
func (s *ChannelStats) RecordRequest(success bool, latencyMs int64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.TotalRequests++
	s.LastUsedTime = time.Now().Unix()

	if success {
		s.SuccessCount++
		// Rolling average latency
		if s.AvgLatency == 0 {
			s.AvgLatency = latencyMs
		} else {
			s.AvgLatency = (s.AvgLatency*9 + latencyMs) / 10
		}
		s.TotalLatency += latencyMs
	} else {
		s.FailCount++
	}

	s.LastUpdateTime = time.Now().Unix()
}

// IncrementConcurrency increments the current concurrency count
func (s *ChannelStats) IncrementConcurrency() {
	atomic.AddInt32(&s.CurrentConcurrency, 1)
}

// DecrementConcurrency decrements the current concurrency count
func (s *ChannelStats) DecrementConcurrency() {
	atomic.AddInt32(&s.CurrentConcurrency, -1)
}

// GetSuccessRate returns the success rate (0.0 - 1.0)
func (s *ChannelStats) GetSuccessRate() float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.TotalRequests == 0 {
		return 1.0 // Default to healthy for new channels
	}
	return float64(s.SuccessCount) / float64(s.TotalRequests)
}

// GetAvgLatency returns the average latency
func (s *ChannelStats) GetAvgLatency() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.AvgLatency
}

// GetConcurrency returns the current concurrency
func (s *ChannelStats) GetConcurrency() int32 {
	return atomic.LoadInt32(&s.CurrentConcurrency)
}

// GetStats returns a snapshot of current statistics
func (s *ChannelStats) GetStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return map[string]interface{}{
		"total_requests":     s.TotalRequests,
		"success_count":      s.SuccessCount,
		"fail_count":         s.FailCount,
		"success_rate":       s.GetSuccessRate(),
		"avg_latency_ms":     s.AvgLatency,
		"current_concurrency": s.GetConcurrency(),
		"last_used_time":     s.LastUsedTime,
	}
}

// Scheduler implements weighted load balancing with dynamic weights
type Scheduler struct {
	mu sync.RWMutex

	// Channel statistics
	channelStats map[int]*ChannelStats

	// Weight factors
	successRateWeight float64
	latencyWeight     float64
	concurrencyWeight float64
}

// NewScheduler creates a new Scheduler instance
func NewScheduler() *Scheduler {
	return &Scheduler{
		channelStats:      make(map[int]*ChannelStats),
		successRateWeight: 0.5,  // 50% weight on success rate
		latencyWeight:     0.3,  // 30% weight on latency
		concurrencyWeight: 0.2,  // 20% weight on concurrency
	}
}

// GetChannelStats returns stats for a channel, creating if not exists
func (s *Scheduler) GetChannelStats(channelId int) *ChannelStats {
	s.mu.Lock()
	defer s.mu.Unlock()

	if stats, ok := s.channelStats[channelId]; ok {
		return stats
	}

	stats := NewChannelStats()
	s.channelStats[channelId] = stats
	return stats
}

// SelectChannel selects the best channel using weighted random selection
func (s *Scheduler) SelectChannel(channels []*model.Channel) *model.Channel {
	if len(channels) == 0 {
		return nil
	}

	if len(channels) == 1 {
		return channels[0]
	}

	// Filter out channels with open circuit breakers
	availableChannels := make([]*model.Channel, 0)
	for _, ch := range channels {
		cb := GetCircuitBreaker(ch.Id)
		if cb.AllowRequest() {
			availableChannels = append(availableChannels, ch)
		}
	}

	if len(availableChannels) == 0 {
		// All channels are circuit broken, return first channel anyway
		// This allows the request to proceed and potentially trigger recovery
		return channels[0]
	}

	if len(availableChannels) == 1 {
		return availableChannels[0]
	}

	// Calculate dynamic weights
	totalWeight := 0
	weights := make([]int, len(availableChannels))

	for i, ch := range availableChannels {
		weight := s.calculateDynamicWeight(ch)
		weights[i] = weight
		totalWeight += weight
	}

	// Weighted random selection
	if totalWeight == 0 {
		// Fallback to uniform random
		return availableChannels[rand.Intn(len(availableChannels))]
	}

	r := rand.Intn(totalWeight)
	for i, w := range weights {
		r -= w
		if r <= 0 {
			return availableChannels[i]
		}
	}

	return availableChannels[0]
}

// SelectLeastConnections selects the channel with the least current connections
func (s *Scheduler) SelectLeastConnections(channels []*model.Channel) *model.Channel {
	if len(channels) == 0 {
		return nil
	}

	var selected *model.Channel
	minConns := int32(1<<31 - 1)

	for _, ch := range channels {
		cb := GetCircuitBreaker(ch.Id)
		if !cb.AllowRequest() {
			continue
		}

		stats := s.GetChannelStats(ch.Id)
		conns := stats.GetConcurrency()

		if conns < minConns {
			minConns = conns
			selected = ch
		}
	}

	if selected != nil {
		stats := s.GetChannelStats(selected.Id)
		stats.IncrementConcurrency()
	}

	return selected
}

// calculateDynamicWeight calculates the dynamic weight for a channel
func (s *Scheduler) calculateDynamicWeight(channel *model.Channel) int {
	baseWeight := channel.GetWeight()
	if baseWeight <= 0 {
		baseWeight = 1
	}

	stats := s.GetChannelStats(channel.Id)

	// Success rate factor (0.0 - 1.0)
	successRate := stats.GetSuccessRate()

	// Latency factor (lower is better, normalized)
	avgLatency := stats.GetAvgLatency()
	var latencyFactor float64 = 1.0
	if avgLatency > 0 {
		// Normalize: 100ms = 1.0, 1000ms = 0.5, 5000ms = 0.17
		latencyFactor = 100.0 / float64(avgLatency+100)
		if latencyFactor > 1.0 {
			latencyFactor = 1.0
		}
	}

	// Concurrency factor (lower is better)
	conns := stats.GetConcurrency()
	var concurrencyFactor float64 = 1.0
	if conns > 0 {
		concurrencyFactor = 1.0 / (1.0 + float64(conns)/10.0)
	}

	// Calculate weighted score
	score := float64(baseWeight) *
		(successRate*s.successRateWeight +
			latencyFactor*s.latencyWeight +
			concurrencyFactor*s.concurrencyWeight)

	// Ensure minimum weight of 1
	result := int(score * 10) // Scale up for better granularity
	if result < 1 {
		result = 1
	}

	return result
}

// RecordRequest records a request result for a channel
func (s *Scheduler) RecordRequest(channelId int, success bool, latencyMs int64) {
	stats := s.GetChannelStats(channelId)
	stats.RecordRequest(success, latencyMs)
}

// ReleaseChannel releases a channel (decrements concurrency)
func (s *Scheduler) ReleaseChannel(channelId int) {
	stats := s.GetChannelStats(channelId)
	stats.DecrementConcurrency()
}

// GetChannelWeight returns the calculated dynamic weight for a channel
func (s *Scheduler) GetChannelWeight(channel *model.Channel) int {
	return s.calculateDynamicWeight(channel)
}

// GetAllChannelStats returns statistics for all channels
func (s *Scheduler) GetAllChannelStats() map[int]map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[int]map[string]interface{})
	for id, stats := range s.channelStats {
		result[id] = stats.GetStats()
	}
	return result
}

// Global scheduler instance
var globalScheduler *Scheduler
var schedulerOnce sync.Once

// GetScheduler returns the global scheduler instance
func GetScheduler() *Scheduler {
	schedulerOnce.Do(func() {
		globalScheduler = NewScheduler()
	})
	return globalScheduler
}

// SelectChannelWithWeight selects a channel using weighted load balancing
func SelectChannelWithWeight(channels []*model.Channel) *model.Channel {
	scheduler := GetScheduler()
	return scheduler.SelectChannel(channels)
}

// SelectChannelLeastConns selects a channel with least connections
func SelectChannelLeastConns(channels []*model.Channel) *model.Channel {
	scheduler := GetScheduler()
	return scheduler.SelectLeastConnections(channels)
}

// RecordChannelRequest records a request result for load balancing statistics
func RecordChannelRequest(channelId int, success bool, latencyMs int64) {
	scheduler := GetScheduler()
	scheduler.RecordRequest(channelId, success, latencyMs)
}

// ReleaseChannelConcurrency releases a channel's concurrency slot
func ReleaseChannelConcurrency(channelId int) {
	scheduler := GetScheduler()
	scheduler.ReleaseChannel(channelId)
}

// init registers the scheduler
func init() {
	// Initialize scheduler on package load
	_ = GetScheduler()
	logger.SysLog("load balancer scheduler initialized")
}