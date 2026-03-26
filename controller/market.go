package controller

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common/ctxkey"
	"github.com/songquanpeng/one-api/model"
)

// GetMarketModels handles GET /api/market/models
func GetMarketModels(c *gin.Context) {
	modelType := c.Query("type")
	keyword := c.Query("q")
	limit, _ := strconv.Atoi(c.Query("limit"))
	offset, _ := strconv.Atoi(c.Query("offset"))

	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	var err error
	var models []*model.ModelInfo

	if keyword != "" {
		models, err = model.SearchModels(keyword, modelType, limit, offset)
	} else {
		models, err = model.GetActiveModels(modelType, limit, offset)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to get models: " + err.Error(),
		})
		return
	}

	total, _ := model.CountActiveModels(modelType)

	// Format response
	data := make([]gin.H, len(models))
	for i, m := range models {
		data[i] = formatModelInfo(m)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"models": data,
			"total":  total,
			"limit":  limit,
			"offset": offset,
		},
	})
}

// GetMarketModel handles GET /api/market/models/:id
func GetMarketModel(c *gin.Context) {
	modelId := c.Param("id")

	m, err := model.GetModelById(modelId)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "Model not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    formatModelInfo(m),
	})
}

// GetMarketProviders handles GET /api/market/providers
func GetMarketProviders(c *gin.Context) {
	providers, err := model.GetAllActiveProviders()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to get providers: " + err.Error(),
		})
		return
	}

	// Get model count per provider
	providerData := make([]gin.H, len(providers))
	for i, p := range providers {
		models, _ := model.GetModelsByProvider(p)
		providerData[i] = gin.H{
			"name":   p,
			"models": len(models),
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    providerData,
	})
}

// GetMarketStats handles GET /api/market/stats
func GetMarketStats(c *gin.Context) {
	stats, err := model.GetModelMarketStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to get stats: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"total_models":     stats.TotalModels,
			"total_providers":  stats.TotalProviders,
			"chat_models":      stats.ChatModels,
			"embedding_models": stats.EmbeddingModels,
			"image_models":     stats.ImageModels,
			"avg_input_price":  stats.AvgInputPrice,
			"avg_output_price": stats.AvgOutputPrice,
		},
	})
}

// CalculatePrice handles GET /api/market/calculate
func CalculatePrice(c *gin.Context) {
	modelId := c.Query("model_id")
	promptTokens, _ := strconv.Atoi(c.Query("prompt_tokens"))
	completionTokens, _ := strconv.Atoi(c.Query("completion_tokens"))

	if modelId == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "model_id is required",
		})
		return
	}

	m, err := model.GetModelById(modelId)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "Model not found",
		})
		return
	}

	quota := m.CalculateQuota(promptTokens, completionTokens)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"model_id":         m.Id,
			"model_name":       m.Name,
			"prompt_tokens":     promptTokens,
			"completion_tokens": completionTokens,
			"input_price":      m.InputPrice,
			"output_price":     m.OutputPrice,
			"quota_cost":       quota,
		},
	})
}

// formatModelInfo formats a ModelInfo for API response
func formatModelInfo(m *model.ModelInfo) gin.H {
	return gin.H{
		"id":           m.Id,
		"name":         m.Name,
		"provider":     m.Provider,
		"model_type":   m.ModelType,
		"description":  m.Description,
		"context_len":  m.ContextLen,
		"input_price":  m.InputPrice,
		"output_price": m.OutputPrice,
		"capabilities": m.Capabilities,
		"status":       m.Status,
		"icon_url":     m.IconUrl,
	}
}

// GetUserDashboardV2 handles GET /api/dashboard (new version)
func GetUserDashboardV2(c *gin.Context) {
	userId := c.GetInt(ctxkey.Id)
	if userId == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "Unauthorized",
		})
		return
	}

	user, err := model.GetUserById(userId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to get user",
		})
		return
	}

	// Get stats
	stats, _ := model.GetUserUsageSummary(userId, 0, 0)
	marketStats, _ := model.GetModelMarketStats()

	// Get recent tokens
	tokens, _ := model.GetUserTokens(userId, 5, 0)

	// Get recent orders
	orders, _ := model.GetUserTopupOrders(userId, "", 5, 0)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"user": gin.H{
				"id":           user.Id,
				"username":     user.Username,
				"email":       user.Email,
				"quota":        user.Quota,
				"display_name": user.DisplayName,
			},
			"usage": stats,
			"market": gin.H{
				"total_models":    marketStats.TotalModels,
				"total_providers": marketStats.TotalProviders,
			},
			"recent_tokens": tokens,
			"recent_orders":  orders,
		},
	})
}

// GetUsageDetail handles GET /api/usage/detail
func GetUsageDetail(c *gin.Context) {
	userId := c.GetInt(ctxkey.Id)
	if userId == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "Unauthorized",
		})
		return
	}

	modelName := c.Query("model")
	start := c.Query("start")
	end := c.Query("end")

	// Parse timestamps
	var startTs, endTs int64
	if start != "" {
		startTs = parseTimestamp(start)
	}
	if end != "" {
		endTs = parseTimestamp(end)
	}

	// Get usage by model
	modelStats, err := model.GetModelUsageStatistics(userId, startTs, endTs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to get usage by model",
		})
		return
	}

	// Get usage by day
	dailyStats, err := model.SearchLogsByDayAndModel(userId, int(startTs), int(endTs))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to get daily usage",
		})
		return
	}

	// Filter by model if specified
	if modelName != "" {
		filtered := make([]*model.LogStatistic, 0)
		for _, s := range dailyStats {
			if s.ModelName == modelName {
				filtered = append(filtered, s)
			}
		}
		dailyStats = filtered
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"by_model": modelStats,
			"by_day":   dailyStats,
		},
	})
}