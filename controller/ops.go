package controller

import (
	"fmt"
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
	"github.com/pagoda-inference/one-api/relay/channeltype"
)

// OpsStats represents operations statistics
type OpsStats struct {
	TodayRevenue     float64            `json:"today_revenue"`
	TodayUsage       int64              `json:"today_usage_tokens"`
	TodayTokens      int64              `json:"today_tokens"`
	ActiveUsers      int                `json:"active_users"`
	ChannelHealth    float64            `json:"channel_health_rate"`
	TotalUsers       int64              `json:"total_users"`
	TotalChannels    int64              `json:"total_channels"`
	TotalTokens      int64              `json:"total_tokens"`
	TotalQuota       int64              `json:"total_quota"`
	RevenueByDay     map[string]float64 `json:"revenue_by_day"`
	UsageByModel     map[string]int64   `json:"usage_by_model"`
	TopUpByDay       map[string]float64 `json:"topup_by_day"`
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
		if v, ok := usage["total_tokens"].(int64); ok {
			stats.TodayTokens = v
			stats.TodayUsage = v
		} else if v, ok := usage["total_tokens"].(int); ok {
			stats.TodayTokens = int64(v)
			stats.TodayUsage = int64(v)
		}
	}

	// Total users
	totalUsers, _ := model.CountUsers()
	stats.TotalUsers = totalUsers

	// Active users today - count distinct users with activity (from paid orders + from logs)
	activeUserSet := make(map[int]bool)
	for _, order := range paidOrders {
		if order.CreatedAt >= startOfDay && order.CreatedAt < endOfDay {
			activeUserSet[order.UserId] = true
		}
	}
	// Also count users who made API calls today (from logs)
	todayLogsUsers, _ := model.GetActiveUsersByLogs(startOfDay, endOfDay)
	for _, uid := range todayLogsUsers {
		activeUserSet[uid] = true
	}
	stats.ActiveUsers = len(activeUserSet)

	// Total quota - sum of all users' quota
	allUsers, _ := model.GetAllUsers(0, 10000, "", "")
	var totalQuota int64
	for _, u := range allUsers {
		totalQuota += u.Quota
	}
	stats.TotalQuota = totalQuota

	// Total tokens consumed (all time)
	allTimeUsage, _ := model.GetUserUsageSummary(0, 0, 0)
	if allTimeUsage != nil {
		if v, ok := allTimeUsage["total_tokens"].(int64); ok {
			stats.TotalTokens = v
		} else if v, ok := allTimeUsage["total_tokens"].(int); ok {
			stats.TotalTokens = int64(v)
		}
	}

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
	users, _ := model.GetAllUsers(0, 1000, "", "")
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
		TypeName     string  `json:"type_name"`
		Group        string  `json:"group"`
		BaseURL      string  `json:"base_url"`
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
			TypeName:   channeltype.GetTypeName(ch.Type),
			Group:      ch.Group,
			BaseURL:    ch.GetBaseURL(),
			Priority:   int(ch.GetPriority()),
			IsEnabled:  ch.Status == model.ChannelStatusEnabled,
		}

		if stats != nil {
			total := stats.SuccessCount + stats.FailCount
			if total > 0 {
				info.SuccessRate = float64(stats.SuccessCount) / float64(total) * 100
			} else {
				// Default to 100% for new channels with no requests
				info.SuccessRate = 100
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
	fmt.Printf("[DEBUG GetOpsUsers] role=%d, RoleAdminUser=%d\n", role, model.RoleAdminUser)
	if role < model.RoleAdminUser {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "Admin access required"})
		return
	}

	limit, _ := strconv.Atoi(c.Query("limit"))
	offset, _ := strconv.Atoi(c.Query("offset"))
	keyword := c.Query("keyword")
	fmt.Printf("[DEBUG GetOpsUsers] limit=%d, offset=%d, keyword=%s\n", limit, offset, keyword)
	if limit <= 0 || limit > 1000 {
		limit = 1000 // Increased to allow fetching more users for search
	}
	if offset < 0 {
		offset = 0
	}

	users, err := model.GetAllUsers(offset, limit, "", keyword)
	fmt.Printf("[DEBUG GetOpsUsers] after GetAllUsers: users=%d, err=%v\n", len(users), err)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Failed to get users"})
		return
	}

	// Note: total count with keyword filter would require a separate count query
	// For now, we just return the filtered count as total
	total := len(users)

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

// AlertConfig option keys
const (
	AlertKeyChannelFailureThreshold = "AlertChannelFailureThreshold"
	AlertKeyQueueUtilizationAlert  = "AlertQueueUtilizationAlert"
	AlertKeyErrorRateAlert         = "AlertErrorRateAlert"
	AlertKeyLatencyThreshold        = "AlertLatencyThreshold"
	AlertKeyAlertEmail             = "AlertEmail"
	AlertKeyAlertWebhook           = "AlertWebhook"
	AlertKeyEnabled                = "AlertEnabled"
	// System config keys
	SysKeyMaxConcurrentRequests    = "SysMaxConcurrentRequests"
	SysKeyRequestQueueTimeout      = "SysRequestQueueTimeout"
	SysKeyHealthCheckInterval      = "SysHealthCheckInterval"
	SysKeyHealthCheckFailThreshold = "SysHealthCheckFailThreshold"
	SysKeyCircuitBreakerThreshold = "SysCircuitBreakerThreshold"
	SysKeyCircuitBreakerTimeout    = "SysCircuitBreakerTimeout"
	SysKeyRelayTimeout             = "SysRelayTimeout"
)

// GetAlertConfig handles GET /api/admin/alerts/config
func GetAlertConfig(c *gin.Context) {
	role := c.GetInt(ctxkey.Role)
	if role < model.RoleAdminUser {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "Admin access required"})
		return
	}

	// Read from OptionMap with fallback to config defaults
	config.OptionMapRWMutex.RLock()
	alertConfig := gin.H{
		// Alert settings
		"channel_failure_threshold": getAlertConfigValue(AlertKeyChannelFailureThreshold, config.CircuitBreakerThreshold),
		"queue_utilization_alert":   getAlertConfigValue(AlertKeyQueueUtilizationAlert, 80),
		"error_rate_alert":          getAlertConfigValue(AlertKeyErrorRateAlert, 1),
		"latency_threshold":         getAlertConfigValue(AlertKeyLatencyThreshold, 5000),
		"alert_email":               getAlertConfigValue(AlertKeyAlertEmail, ""),
		"alert_webhook":             getAlertConfigValue(AlertKeyAlertWebhook, ""),
		"enabled":                  getAlertConfigValue(AlertKeyEnabled, true),
		// System config
		"max_concurrent_requests":    getAlertConfigValue(SysKeyMaxConcurrentRequests, config.MaxConcurrentRequests),
		"request_queue_timeout":     getAlertConfigValue(SysKeyRequestQueueTimeout, config.RequestQueueTimeout),
		"health_check_interval":     getAlertConfigValue(SysKeyHealthCheckInterval, config.HealthCheckInterval),
		"health_check_fail_threshold": getAlertConfigValue(SysKeyHealthCheckFailThreshold, config.HealthCheckFailThreshold),
		"circuit_breaker_threshold": getAlertConfigValue(SysKeyCircuitBreakerThreshold, config.CircuitBreakerThreshold),
		"circuit_breaker_timeout":   getAlertConfigValue(SysKeyCircuitBreakerTimeout, config.CircuitBreakerTimeout),
		"relay_timeout":             getAlertConfigValue(SysKeyRelayTimeout, config.RelayTimeout),
	}
	config.OptionMapRWMutex.RUnlock()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    alertConfig,
	})
}

// getAlertConfigValue retrieves an alert config value from OptionMap with a default fallback
func getAlertConfigValue(key string, defaultVal interface{}) interface{} {
	if val, ok := config.OptionMap[key]; ok {
		switch defaultVal.(type) {
		case int:
			if intVal, err := strconv.Atoi(val); err == nil {
				return intVal
			}
		case int64:
			if intVal, err := strconv.ParseInt(val, 10, 64); err == nil {
				return intVal
			}
		case float64:
			if floatVal, err := strconv.ParseFloat(val, 64); err == nil {
				return floatVal
			}
		case bool:
			return val == "true"
		case string:
			return val
		}
	}
	return defaultVal
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

	// Save alert config fields
	if v, ok := req["channel_failure_threshold"]; ok {
		model.UpdateOption(AlertKeyChannelFailureThreshold, fmt.Sprintf("%v", v))
	}
	if v, ok := req["queue_utilization_alert"]; ok {
		model.UpdateOption(AlertKeyQueueUtilizationAlert, fmt.Sprintf("%v", v))
	}
	if v, ok := req["error_rate_alert"]; ok {
		model.UpdateOption(AlertKeyErrorRateAlert, fmt.Sprintf("%v", v))
	}
	if v, ok := req["latency_threshold"]; ok {
		model.UpdateOption(AlertKeyLatencyThreshold, fmt.Sprintf("%v", v))
	}
	if v, ok := req["alert_email"]; ok {
		model.UpdateOption(AlertKeyAlertEmail, fmt.Sprintf("%v", v))
	}
	if v, ok := req["alert_webhook"]; ok {
		model.UpdateOption(AlertKeyAlertWebhook, fmt.Sprintf("%v", v))
	}
	if v, ok := req["enabled"]; ok {
		model.UpdateOption(AlertKeyEnabled, fmt.Sprintf("%v", v))
	}

	// Save system config fields
	if v, ok := req["max_concurrent_requests"]; ok {
		model.UpdateOption(SysKeyMaxConcurrentRequests, fmt.Sprintf("%v", v))
		model.UpdateConfigInt("MaxConcurrentRequests", v)
	}
	if v, ok := req["request_queue_timeout"]; ok {
		model.UpdateOption(SysKeyRequestQueueTimeout, fmt.Sprintf("%v", v))
		model.UpdateConfigInt("RequestQueueTimeout", v)
	}
	if v, ok := req["health_check_interval"]; ok {
		model.UpdateOption(SysKeyHealthCheckInterval, fmt.Sprintf("%v", v))
		model.UpdateConfigInt("HealthCheckInterval", v)
	}
	if v, ok := req["health_check_fail_threshold"]; ok {
		model.UpdateOption(SysKeyHealthCheckFailThreshold, fmt.Sprintf("%v", v))
		model.UpdateConfigInt("HealthCheckFailThreshold", v)
	}
	if v, ok := req["circuit_breaker_threshold"]; ok {
		model.UpdateOption(SysKeyCircuitBreakerThreshold, fmt.Sprintf("%v", v))
		model.UpdateConfigInt("CircuitBreakerThreshold", v)
	}
	if v, ok := req["circuit_breaker_timeout"]; ok {
		model.UpdateOption(SysKeyCircuitBreakerTimeout, fmt.Sprintf("%v", v))
		model.UpdateConfigInt("CircuitBreakerTimeout", v)
	}
	if v, ok := req["relay_timeout"]; ok {
		model.UpdateOption(SysKeyRelayTimeout, fmt.Sprintf("%v", v))
		model.UpdateConfigInt("RelayTimeout", v)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "配置已更新，部分配置需重启服务生效",
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