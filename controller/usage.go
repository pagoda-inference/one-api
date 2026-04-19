package controller

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pagoda-inference/one-api/common/ctxkey"
	"github.com/pagoda-inference/one-api/model"
)

// GetUsageSummary handles GET /api/usage/summary
func GetUsageSummary(c *gin.Context) {
	userId := c.GetInt(ctxkey.Id)
	if userId == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "Unauthorized",
		})
		return
	}

	// Parse time range
	startTimestamp := parseTimestamp(c.Query("start"))
	endTimestamp := parseTimestamp(c.Query("end"))

	summary, err := model.GetUserUsageSummary(userId, startTimestamp, endTimestamp)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to get usage summary: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    summary,
	})
}

// GetUsageByToken handles GET /api/usage/by-token
func GetUsageByToken(c *gin.Context) {
	userId := c.GetInt(ctxkey.Id)
	if userId == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "Unauthorized",
		})
		return
	}

	startTimestamp := parseTimestamp(c.Query("start"))
	endTimestamp := parseTimestamp(c.Query("end"))

	stats, err := model.GetTokenUsageStatistics(userId, startTimestamp, endTimestamp)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to get usage by token: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
	})
}

// GetUsageByModel handles GET /api/usage/by-model
func GetUsageByModel(c *gin.Context) {
	userId := c.GetInt(ctxkey.Id)
	if userId == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "Unauthorized",
		})
		return
	}

	startTimestamp := parseTimestamp(c.Query("start"))
	endTimestamp := parseTimestamp(c.Query("end"))

	stats, err := model.GetModelUsageStatistics(userId, startTimestamp, endTimestamp)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to get usage by model: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
	})
}

// GetUsageByChannel handles GET /api/usage/by-channel
func GetUsageByChannel(c *gin.Context) {
	userId := c.GetInt(ctxkey.Id)
	if userId == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "Unauthorized",
		})
		return
	}

	startTimestamp := parseTimestamp(c.Query("start"))
	endTimestamp := parseTimestamp(c.Query("end"))

	stats, err := model.GetChannelUsageStatistics(userId, startTimestamp, endTimestamp)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to get usage by channel: " + err.Error(),
		})
		return
	}

	// Enrich with channel names
	for _, stat := range stats {
		channel, err := model.GetChannelById(stat.ChannelId, false)
		if err == nil && channel != nil {
			stat.ChannelName = channel.Name
		} else {
			stat.ChannelName = "Unknown"
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
	})
}

// GetUsageByHour handles GET /api/usage/by-hour
func GetUsageByHour(c *gin.Context) {
	userId := c.GetInt(ctxkey.Id)
	if userId == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "Unauthorized",
		})
		return
	}

	startTimestamp := parseTimestamp(c.Query("start"))
	endTimestamp := parseTimestamp(c.Query("end"))

	stats, err := model.GetHourlyUsageStatistics(userId, startTimestamp, endTimestamp)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to get usage by hour: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
	})
}

// GetUsageDaily handles GET /api/usage/daily
func GetUsageDaily(c *gin.Context) {
	userId := c.GetInt(ctxkey.Id)
	if userId == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "Unauthorized",
		})
		return
	}

	// Default to last 7 days
	end := time.Now().Unix()
	start := end - 7*24*60*60

	startParam := c.Query("start")
	endParam := c.Query("end")

	if startParam != "" {
		start = parseTimestamp(startParam)
	}
	if endParam != "" {
		end = parseTimestamp(endParam)
	}

	stats, err := model.SearchLogsByDayAndModel(userId, int(start), int(end))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to get daily usage: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
	})
}

// AdminGetUsageSummary handles GET /api/admin/usage/summary (admin only)
func AdminGetUsageSummary(c *gin.Context) {
	startTimestamp := parseTimestamp(c.Query("start"))
	endTimestamp := parseTimestamp(c.Query("end"))
	modelName := c.Query("model")
	tokenName := c.Query("token")
	channelId, _ := strconv.Atoi(c.Query("channel"))

	quota := model.SumUsedQuota(model.LogTypeConsume, startTimestamp, endTimestamp, modelName, "", tokenName, channelId)
	tokens := model.SumUsedTokenSeparate(model.LogTypeConsume, startTimestamp, endTimestamp, modelName, "", tokenName)

	// If querying all-time (no start timestamp), add archived data
	if startTimestamp == 0 {
		if archived, err := model.GetArchivedUsage(); err == nil {
			quota += int64(archived.TotalQuota)
			tokens.PromptTokens += archived.PromptTokens
			tokens.CompletionTokens += archived.CompletionTokens
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"total_quota":       quota,
			"total_tokens":      tokens.PromptTokens + tokens.CompletionTokens,
			"total_prompt_tokens":     tokens.PromptTokens,
			"total_completion_tokens":  tokens.CompletionTokens,
			"period": gin.H{
				"start": startTimestamp,
				"end":   endTimestamp,
			},
		},
	})
}

// AdminGetUsageByUsers handles GET /api/admin/usage/by-users (admin only)
func AdminGetUsageByUsers(c *gin.Context) {
	startTimestamp := parseTimestamp(c.Query("start"))
	endTimestamp := parseTimestamp(c.Query("end"))

	stats, err := model.GetUserUsageStatistics(startTimestamp, endTimestamp)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to get user usage: " + err.Error(),
		})
		return
	}

	// Enrich with user display names
	userCache := make(map[int]string)
	for _, stat := range stats {
		if _, ok := userCache[stat.UserId]; !ok {
			user, _ := model.GetUserById(stat.UserId, false)
			if user != nil {
				userCache[stat.UserId] = user.DisplayName
			}
		}
		if stat.DisplayName == "" {
			stat.DisplayName = userCache[stat.UserId]
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"users":   stats,
			"period": gin.H{
				"start": startTimestamp,
				"end":   endTimestamp,
			},
		},
	})
}

// AdminGetUsageByModels handles GET /api/admin/usage/by-models (admin only)
func AdminGetUsageByModels(c *gin.Context) {
	startTimestamp := parseTimestamp(c.Query("start"))
	endTimestamp := parseTimestamp(c.Query("end"))

	stats, err := model.GetModelUsageStatistics(0, startTimestamp, endTimestamp) // 0 means all users
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to get model usage: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"models":  stats,
			"period": gin.H{
				"start": startTimestamp,
				"end":   endTimestamp,
			},
		},
	})
}

// parseTimestamp parses a timestamp string (Unix timestamp or RFC3339 date)
func parseTimestamp(s string) int64 {
	if s == "" {
		return 0
	}

	// Try parsing as Unix timestamp first
	if ts, err := strconv.ParseInt(s, 10, 64); err == nil {
		return ts
	}

	// Try parsing as RFC3339 date
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.Unix()
	}

	// Try parsing as date only (YYYY-MM-DD)
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t.Unix()
	}

	return 0
}