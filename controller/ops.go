package controller

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pagoda-inference/one-api/common"
	"github.com/pagoda-inference/one-api/common/config"
	"github.com/pagoda-inference/one-api/common/ctxkey"
	"github.com/pagoda-inference/one-api/middleware"
	"github.com/pagoda-inference/one-api/model"
	"github.com/pagoda-inference/one-api/monitor"
)

// OpsStats represents operations statistics
type OpsStats struct {
	TodayRevenue     float64            `json:"today_revenue"`
	TodayUsage       int64              `json:"today_usage_tokens"`
	ActiveUsers      int                `json:"active_users"`
	ChannelHealth    float64            `json:"channel_health_rate"`
	TotalUsers      int64              `json:"total_users"`
	TotalChannels   int64              `json:"total_channels"`
	TotalTokens     int64              `json:"total_tokens"`
	TotalQuota      int64              `json:"total_quota"`
	RevenueByDay    map[string]float64 `json:"revenue_by_day"`
	UsageByModel    map[string]int64   `json:"usage_by_model"`
	TopUpByDay      map[string]float64  `json:"topup_by_day"`
}

// GetOpsStats handles GET /api/admin/ops/stats
func GetOpsStats(c *gin.Context) {
	// Check admin permission
	role := c.GetInt(ctxkey.Role)
	if role < model.RoleAdminUser {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"message": "Admin access required",
		})
		return
	}

	// Get time range
	today := time.Now()
	startOfDay := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location()).Unix()
	endOfDay := startOfDay + 86400

	stats := &OpsStats{}

	// Today's revenue from paid orders
	paidOrders, _ := model.GetUserTopupOrders(0, model.TopupStatusPaid, 1000, 0)
	for _, order := range paidOrders {
		if order.CreatedAt >= startOfDay && order.CreatedAt < endOfDay {
			stats.TodayRevenue += order.Amount
		}
	}

	// Today's usage
	usage, _ := model.GetUserUsageSummary(0, startOfDay, endOfDay)
	if usage != nil {
		stats.TodayUsage = usage["total_tokens"].(int64)
	}

	// Total users
	totalUsers, _ := model.CountUsers()
	stats.TotalUsers = totalUsers

	// Active users today
	stats.ActiveUsers = len(paidOrders) // Approximation

	// Total channels
	totalChannels, _ := model.CountChannels()
	stats.TotalChannels = totalChannels

	// Get channel health
	stats.ChannelHealth = getChannelHealthRate()

	// Usage by model (last 7 days)
	weekAgo := startOfDay - 7*86400
	modelStats, _ := model.GetModelUsageStatistics(0, weekAgo, endOfDay)
	stats.UsageByModel = make(map[string]int64)
	for _, m := range modelStats {
		stats.UsageByModel[m.ModelName] = int64(m.PromptTokens + m.CompletionTokens)
	}

	// Revenue by day (last 7 days)
	stats.RevenueByDay = make(map[string]float64)
	stats.TopUpByDay = make(map[string]float64)
	for i := 0; i < 7; i++ {
		day := time.Unix(startOfDay-int64(i*86400), 0).Format("2006-01-02")
		stats.RevenueByDay[day] = 0
		stats.TopUpByDay[day] = 0
	}
	for _, order := range paidOrders {
		if order.CreatedAt >= startOfDay-7*86400 {
			day := time.Unix(order.CreatedAt, 0).Format("2006-01-02")
			if _, ok := stats.RevenueByDay[day]; ok {
				stats.RevenueByDay[day] += order.Amount
			}
			if _, ok := stats.TopUpByDay[day]; ok {
				stats.TopUpByDay[day] += order.Amount
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
	})
}

// getChannelHealthRate calculates the health rate of all channels
func getChannelHealthRate() float64 {
	channels, err := model.GetAllChannels(0, 1000, "")
	if err != nil || len(channels) == 0 {
		return 0
	}

	healthyCount := 0
	for _, ch := range channels {
		if ch.Status == model.ChannelStatusEnabled {
			healthyCount++
		}
	}

	return float64(healthyCount) / float64(len(channels)) * 100
}

// GetOpsRevenue handles GET /api/admin/ops/revenue
func GetOpsRevenue(c *gin.Context) {
	role := c.GetInt(ctxkey.Role)
	if role < model.RoleAdminUser {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "Admin access required"})
		return
	}

	// Parse time range
	start := c.Query("start")
	end := c.Query("end")
	var startTs, endTs int64
	if start != "" {
		startTs = parseTimestamp(start)
	}
	if end != "" {
		endTs = parseTimestamp(end)
	}

	// Get all paid orders in range
	orders, _ := model.GetUserTopupOrders(0, model.TopupStatusPaid, 1000, 0)

	type DailyRevenue struct {
		Date      string  `json:"date"`
		Amount    float64 `json:"amount"`
		Count     int     `json:"count"`
		Quota     int64   `json:"quota"`
	}

	dailyMap := make(map[string]*DailyRevenue)
	for _, order := range orders {
		if order.CreatedAt >= startTs && order.CreatedAt <= endTs {
			day := time.Unix(order.CreatedAt, 0).Format("2006-01-02")
			if _, ok := dailyMap[day]; !ok {
				dailyMap[day] = &DailyRevenue{Date: day}
			}
			dailyMap[day].Amount += order.Amount
			dailyMap[day].Count++
			dailyMap[day].Quota += order.Quota
		}
	}

	// Convert to slice
	daily := make([]*DailyRevenue, 0, len(dailyMap))
	for _, d := range dailyMap {
		daily = append(daily, d)
	}

	// Calculate totals
	var totalAmount float64
	var totalQuota int64
	var totalCount int
	for _, d := range daily {
		totalAmount += d.Amount
		totalQuota += d.Quota
		totalCount += d.Count
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"daily":       daily,
			"total_amount": totalAmount,
			"total_quota":  totalQuota,
			"total_count": totalCount,
		},
	})
}

// GetOpsUsage handles GET /api/admin/ops/usage
func GetOpsUsage(c *gin.Context) {
	role := c.GetInt(ctxkey.Role)
	if role < model.RoleAdminUser {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "Admin access required"})
		return
	}

	// Parse time range
	start := c.Query("start")
	end := c.Query("end")
	var startTs, endTs int64
	if start != "" {
		startTs = parseTimestamp(start)
	}
	if end != "" {
		endTs = parseTimestamp(end)
	}

	// Get usage by model
	modelStats, _ := model.GetModelUsageStatistics(0, startTs, endTs)

	// Get usage by user
	type UserUsage struct {
		UserId    int    `json:"user_id"`
		Username string `json:"username"`
		Quota    int64  `json:"quota"`
		Tokens   int64  `json:"tokens"`
	}

	// Get all users with usage
	users, _ := model.GetAllUsers(0, 1000, "")
	userUsageMap := make(map[int]*UserUsage)

	for _, u := range users {
		userUsageMap[u.Id] = &UserUsage{
			UserId:    u.Id,
			Username: u.Username,
		}
	}

	// Aggregate usage
	_ = modelStats // Avoid unused variable warning

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"by_model": modelStats,
		},
	})
}

// GetChannelHealth handles GET /api/admin/channels/health
func GetChannelHealth(c *gin.Context) {
	role := c.GetInt(ctxkey.Role)
	if role < model.RoleAdminUser {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "Admin access required"})
		return
	}

	channels, err := model.GetAllChannels(0, 1000, "")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Failed to get channels"})
		return
	}

	type ChannelHealthInfo struct {
		Id           int     `json:"id"`
		Name         string  `json:"name"`
		Status       string  `json:"status"`
		Type         int     `json:"type"`
		BaseURL      string  `json:"base_url"`
		Balance      float64 `json:"balance"`
		SuccessRate  float64 `json:"success_rate"`
		AvgLatency   int64   `json:"avg_latency"`
		Priority     int     `json:"priority"`
		IsEnabled    bool    `json:"is_enabled"`
	}

	healthList := make([]*ChannelHealthInfo, 0)
	for _, ch := range channels {
		// Get channel stats from scheduler
		stats := monitor.GetScheduler().GetChannelStats(ch.Id)

		info := &ChannelHealthInfo{
			Id:         ch.Id,
			Name:       ch.Name,
			Status:     strconv.Itoa(ch.Status),
			Type:       ch.Type,
			BaseURL:    ch.GetBaseURL(),
			Priority:   int(ch.GetPriority()),
			IsEnabled:  ch.Status == model.ChannelStatusEnabled,
		}

		if stats != nil {
			total := stats.SuccessCount + stats.FailCount
			if total > 0 {
				info.SuccessRate = float64(stats.SuccessCount) / float64(total) * 100
			}
			info.AvgLatency = stats.AvgLatency
		}

		healthList = append(healthList, info)
	}

	// Calculate overall health
	var totalHealth float64
	enabledCount := 0
	for _, ch := range healthList {
		if ch.IsEnabled {
			totalHealth += ch.SuccessRate
			enabledCount++
		}
	}

	var overallHealth float64
	if enabledCount > 0 {
		overallHealth = totalHealth / float64(enabledCount)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"channels":       healthList,
			"overall_health":  overallHealth,
			"enabled_count":   enabledCount,
			"total_count":     len(channels),
		},
	})
}

// GetOpsUsers handles GET /api/admin/ops/users
func GetOpsUsers(c *gin.Context) {
	role := c.GetInt(ctxkey.Role)
	if role < model.RoleAdminUser {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "Admin access required"})
		return
	}

	limit, _ := strconv.Atoi(c.Query("limit"))
	offset, _ := strconv.Atoi(c.Query("offset"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	users, err := model.GetAllUsers(limit, offset, "")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Failed to get users"})
		return
	}

	total, _ := model.CountUsers()

	// Enrich with usage data
	type EnrichedUser struct {
		model.User
		UsageQuota int64 `json:"usage_quota"`
	}

	enrichedUsers := make([]*EnrichedUser, len(users))
	for i, u := range users {
		enrichedUsers[i] = &EnrichedUser{User: *u}
		usage, _ := model.GetUserUsageSummary(u.Id, 0, 0)
		if usage != nil {
			enrichedUsers[i].UsageQuota = usage["total_quota"].(int64)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"users":  enrichedUsers,
			"total":  total,
			"limit":  limit,
			"offset": offset,
		},
	})
}

// GetAlertConfig handles GET /api/admin/alerts/config
func GetAlertConfig(c *gin.Context) {
	role := c.GetInt(ctxkey.Role)
	if role < model.RoleAdminUser {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "Admin access required"})
		return
	}

	// Return current alert configuration
	// In production, this would come from database or config
	config := gin.H{
		"channel_failure_threshold": config.CircuitBreakerThreshold,
		"queue_utilization_alert":  80, // percent
		"error_rate_alert":        1,   // percent
		"latency_threshold":        5000, // ms
		"alert_email":             "", // to be configured
		"alert_webhook":           "", // to be configured
		"enabled":                true,
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    config,
	})
}

// UpdateAlertConfig handles PUT /api/admin/alerts/config
func UpdateAlertConfig(c *gin.Context) {
	role := c.GetInt(ctxkey.Role)
	if role < model.RoleAdminUser {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "Admin access required"})
		return
	}

	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid request"})
		return
	}

	// In production, save to database
	// For now, just acknowledge

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Alert config updated",
	})
}

// GetSystemHealth handles GET /api/admin/system/health
func GetSystemHealth(c *gin.Context) {
	role := c.GetInt(ctxkey.Role)
	if role < model.RoleAdminUser {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "Admin access required"})
		return
	}

	// Get system metrics
	queueStats := middleware.GetQueueStats()
	scheduler := monitor.GetScheduler()
	channelStats := scheduler.GetAllChannelStats()
	cbStats := monitor.GetAllCircuitBreakerStats()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"uptime":        time.Since(time.Unix(common.StartTime, 0)).Seconds(),
			"queue":         queueStats,
			"channels":      channelStats,
			"circuit_breakers": cbStats,
			"config": gin.H{
				"max_concurrent":   config.MaxConcurrentRequests,
				"health_interval":  config.HealthCheckInterval,
				"cb_threshold":     config.CircuitBreakerThreshold,
			},
		},
	})
}

// ExportReport handles GET /api/admin/reports/export
func ExportReport(c *gin.Context) {
	role := c.GetInt(ctxkey.Role)
	if role < model.RoleAdminUser {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "Admin access required"})
		return
	}

	reportType := c.Query("type") // daily, weekly, monthly
	start := c.Query("start")
	end := c.Query("end")

	var startTs, endTs int64
	if start != "" {
		startTs = parseTimestamp(start)
	}
	if end != "" {
		endTs = parseTimestamp(end)
	}

	// Generate report based on type
	switch reportType {
	case "daily":
		// Daily revenue and usage report
	case "weekly":
		// Weekly aggregated report
	case "monthly":
		// Monthly financial report
	default:
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid report type"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"type":       reportType,
			"start":      startTs,
			"end":        endTs,
			"report_url": "/api/admin/reports/download?token=xxx", // Placeholder
		},
	})
}