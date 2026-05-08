package controller

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pagoda-inference/one-api/common/ctxkey"
	"github.com/pagoda-inference/one-api/common/logger"
	"github.com/pagoda-inference/one-api/model"
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

	userId := c.GetInt(ctxkey.Id)
	tenantIds, _ := model.GetUserTenantIds(userId)

	models, total, err := model.GetVisibleModelsForTenants(tenantIds, keyword, modelType, limit, offset)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to get models: " + err.Error(),
		})
		return
	}

	// Format response
	data := make([]gin.H, 0, len(models))
	for _, m := range models {
		data = append(data, formatModelInfo(m))
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
	return
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
	userId := c.GetInt(ctxkey.Id)
	tenantIds, _ := model.GetUserTenantIds(userId)

	// Get all active models
	var allModels []*model.ModelInfo
	var err error
	allModels, err = model.GetActiveModels("", 0, 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to get models: " + err.Error(),
		})
		return
	}

	// Filter by visibility
	var visibleModels []*model.ModelInfo
	for _, m := range allModels {
		if isModelVisibleToUser(m.VisibleToTeams, tenantIds) {
			visibleModels = append(visibleModels, m)
		}
	}

	// Count by type from visible models
	chatModels := 0
	embeddingModels := 0
	imageModels := 0
	trialModels := 0
	for _, m := range visibleModels {
		switch m.ModelType {
		case model.ModelTypeChat:
			chatModels++
		case model.ModelTypeEmbedding:
			embeddingModels++
		case model.ModelTypeImage:
			imageModels++
		}
		if m.IsTrial {
			trialModels++
		}
	}

	// Get provider count (distinct providers from visible models)
	providerSet := make(map[string]bool)
	for _, m := range visibleModels {
		if m.Provider != "" {
			providerSet[m.Provider] = true
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"total_models":     len(visibleModels),
			"total_providers":  len(providerSet),
			"total_groups":     len(providerSet),
			"chat_models":      chatModels,
			"embedding_models": embeddingModels,
			"image_models":     imageModels,
			"trial_models":     trialModels,
			"avg_input_price":  0,
			"avg_output_price": 0,
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
			"model_id":          m.Id,
			"model_name":        m.Name,
			"prompt_tokens":     promptTokens,
			"completion_tokens": completionTokens,
			"input_price":       m.InputPrice,
			"output_price":      m.OutputPrice,
			"quota_cost":        quota,
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
		"tags":         m.Tags,
		"status":       m.Status,
		"icon_url":     m.IconUrl,
		"group_id":     m.GroupId,
		"is_trial":     m.IsTrial,
		"trial_quota":  m.TrialQuota,
		"sla":          m.SLA,
	}
}

// GetMarketGroups handles GET /api/market/groups
// Returns providers from the providers table for the marketplace group filter
func GetMarketGroups(c *gin.Context) {
	providers, err := model.GetActiveProviders()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to get groups: " + err.Error(),
		})
		return
	}

	userId := c.GetInt(ctxkey.Id)
	tenantIds, _ := model.GetUserTenantIds(userId)

	// Build response with model count per provider (filtered by user visibility)
	data := make([]gin.H, 0)
	for _, p := range providers {
		var models []*model.ModelInfo
		err := model.DB.
			Where("status = ? AND LOWER(provider) = ?", model.ModelStatusActive, strings.ToLower(p.Code)).
			Find(&models).Error
		if err != nil {
			logger.SysErrorf("GetMarketGroups: failed to load models for provider=%s: %v", p.Code, err)
			continue
		}

		// Count models visible to user
		modelCount := 0
		for _, m := range models {
			if isModelVisibleToUser(m.VisibleToTeams, tenantIds) {
				modelCount++
			}
		}

		// Only include providers with visible models
		if modelCount > 0 {
			data = append(data, gin.H{
				"id":          p.Id,
				"code":        p.Code,
				"name":        p.Name,
				"logo_url":    p.LogoUrl,
				"description": p.Description,
				"model_count": modelCount,
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    data,
	})
}

// GetMarketModelsByGroup handles GET /api/market/groups/:id/models
func GetMarketModelsByGroup(c *gin.Context) {
	groupId, _ := strconv.Atoi(c.Param("id"))

	models, err := model.GetModelsByGroup(groupId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to get models: " + err.Error(),
		})
		return
	}

	data := make([]gin.H, len(models))
	for i, m := range models {
		data[i] = formatModelInfo(m)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"models": data,
			"total":  len(models),
		},
	})
}

// GetModelTrial handles GET /api/market/models/:id/trial
func GetModelTrial(c *gin.Context) {
	userId := c.GetInt(ctxkey.Id)
	modelId := c.Param("id")

	// Get tenant ID from user (default to 0 for standalone users)
	tenantId := 0

	canUse, reason := model.CheckModelTrial(userId, modelId, tenantId)
	if !canUse {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": gin.H{
				"available": false,
				"reason":    reason,
			},
		})
		return
	}

	// Get trial info
	trials, _ := model.GetUserTrials(userId, tenantId)
	var currentTrial *model.ModelTrial
	for _, t := range trials {
		if t.ModelId == modelId {
			currentTrial = t
			break
		}
	}

	modelInfo, _ := model.GetModelById(modelId)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"available":       true,
			"model":           formatModelInfo(modelInfo),
			"quota_used":      currentTrial.QuotaUsed,
			"quota_limit":     modelInfo.TrialQuota,
			"quota_remaining": modelInfo.TrialQuota - currentTrial.QuotaUsed,
		},
	})
}

// StartModelTrial handles POST /api/market/models/:id/trial
func StartModelTrial(c *gin.Context) {
	userId := c.GetInt(ctxkey.Id)
	modelId := c.Param("id")

	// Get tenant ID from user
	tenantId := 0

	// Check if trial is available
	canUse, reason := model.CheckModelTrial(userId, modelId, tenantId)
	if !canUse {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": reason,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Trial started successfully",
	})
}

// GetUserTrials handles GET /api/market/trials
func GetUserTrials(c *gin.Context) {
	userId := c.GetInt(ctxkey.Id)
	tenantId := 0 // Default tenant

	trials, err := model.GetUserTrials(userId, tenantId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to get trials",
		})
		return
	}

	// Enrich with model info
	data := make([]gin.H, len(trials))
	for i, t := range trials {
		modelInfo, _ := model.GetModelById(t.ModelId)
		data[i] = gin.H{
			"trial":       t,
			"model":       formatModelInfo(modelInfo),
			"quota_used":  t.QuotaUsed,
			"quota_limit": modelInfo.TrialQuota,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    data,
	})
}

// GetPlaygroundModels handles GET /api/market/playground/models
// Returns trial models with their playground configuration for ChatPlayground
func GetPlaygroundModels(c *gin.Context) {
	userId := c.GetInt(ctxkey.Id)
	tenantIds, _ := model.GetUserTenantIds(userId)

	models, err := model.GetTrialModels()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to get models: " + err.Error(),
		})
		return
	}

	// Filter by visibility and format response
	data := make([]gin.H, 0)
	for _, m := range models {
		if !isModelVisibleToUser(m.VisibleToTeams, tenantIds) {
			continue
		}
		isVL := m.PlaygroundEnableVL
		if !isVL && strings.Contains(strings.ToLower(m.Capabilities), "vision") {
			isVL = true
		}
		isReasoning := m.PlaygroundEnableReasoning
		if !isReasoning && strings.Contains(strings.ToLower(m.Capabilities), "reasoning") {
			isReasoning = true
		}
		data = append(data, gin.H{
			"id":                        m.Id,
			"name":                      m.Name,
			"model_type":                m.ModelType,
			"is_vl":                     isVL,
			"is_reasoning":              isReasoning,
			"max_tokens":                m.PlaygroundMaxTokens,
			"temperature":               m.PlaygroundTemperature,
			"min_p":                     m.PlaygroundMinP,
			"top_p":                     m.PlaygroundTopP,
			"top_k":                     m.PlaygroundTopK,
			"frequency_penalty":         m.PlaygroundFrequencyPenalty,
			"presence_penalty":          m.PlaygroundPresencePenalty,
			"repetition_penalty":        m.PlaygroundRepetitionPenalty,
			"system_prompt":             m.PlaygroundSystemPrompt,
			"enable_thinking":           m.PlaygroundEnableThinking,
			"thinking_budget":           m.PlaygroundThinkingBudget,
			"enable_temperature":        m.PlaygroundEnableTemperature,
			"enable_min_p":              m.PlaygroundEnableMinP,
			"enable_top_p":              m.PlaygroundEnableTopP,
			"enable_top_k":              m.PlaygroundEnableTopK,
			"enable_frequency_penalty":  m.PlaygroundEnableFrequencyPenalty,
			"enable_presence_penalty":   m.PlaygroundEnablePresencePenalty,
			"enable_repetition_penalty": m.PlaygroundEnableRepetitionPenalty,
			"enable_system_prompt":      m.PlaygroundEnableSystemPrompt,
			"enable_thinking_budget":    m.PlaygroundEnableThinkingBudget,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    data,
	})
}

// UpdatePlaygroundModel handles PUT /api/admin/market/playground/models/:id
// Updates playground configuration for a model (admin only)
func UpdatePlaygroundModel(c *gin.Context) {
	modelId := strings.TrimPrefix(c.Param("id"), "/")

	var req struct {
		MaxTokens               int     `json:"max_tokens"`
		Temperature             float64 `json:"temperature"`
		MinP                    float64 `json:"min_p"`
		TopP                    float64 `json:"top_p"`
		TopK                    int     `json:"top_k"`
		FrequencyPenalty        float64 `json:"frequency_penalty"`
		PresencePenalty         float64 `json:"presence_penalty"`
		RepetitionPenalty       float64 `json:"repetition_penalty"`
		SystemPrompt            string  `json:"system_prompt"`
		EnableThinking          bool    `json:"enable_thinking"`
		ThinkingBudget          int     `json:"thinking_budget"`
		EnableTemperature       bool    `json:"enable_temperature"`
		EnableMinP              bool    `json:"enable_min_p"`
		EnableTopP              bool    `json:"enable_top_p"`
		EnableTopK              bool    `json:"enable_top_k"`
		EnableFrequencyPenalty  bool    `json:"enable_frequency_penalty"`
		EnablePresencePenalty   bool    `json:"enable_presence_penalty"`
		EnableRepetitionPenalty bool    `json:"enable_repetition_penalty"`
		EnableSystemPrompt      bool    `json:"enable_system_prompt"`
		EnableVL                bool    `json:"is_vl"`
		EnableReasoning         bool    `json:"is_reasoning"`
		EnableThinkingBudget    bool    `json:"enable_thinking_budget"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid request",
		})
		return
	}

	if err := model.UpdatePlaygroundConfig(
		modelId,
		req.MaxTokens,
		req.Temperature,
		req.MinP,
		req.TopP,
		req.TopK,
		req.FrequencyPenalty,
		req.PresencePenalty,
		req.RepetitionPenalty,
		req.SystemPrompt,
		req.EnableThinking,
		req.ThinkingBudget,
		req.EnableTemperature,
		req.EnableMinP,
		req.EnableTopP,
		req.EnableTopK,
		req.EnableFrequencyPenalty,
		req.EnablePresencePenalty,
		req.EnableRepetitionPenalty,
		req.EnableSystemPrompt,
		req.EnableVL,
		req.EnableReasoning,
		req.EnableThinkingBudget,
	); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to update model",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Model updated successfully",
	})
}

// SetModelPricing handles POST /api/admin/market/pricing
func SetModelPricing(c *gin.Context) {
	var req struct {
		ModelId     string  `json:"model_id" binding:"required"`
		TenantId    int     `json:"tenant_id"` // 0 for default pricing
		InputPrice  float64 `json:"input_price"`
		OutputPrice float64 `json:"output_price"`
		Discount    float64 `json:"discount"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid request",
		})
		return
	}

	pricing := &model.ModelPricing{
		ModelId:     req.ModelId,
		TenantId:    req.TenantId,
		InputPrice:  req.InputPrice,
		OutputPrice: req.OutputPrice,
		Discount:    req.Discount,
	}

	if err := model.SetModelPricing(pricing); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to set pricing",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Pricing set successfully",
	})
}

// GetModelPricing handles GET /api/market/models/:id/pricing
func GetModelPricing(c *gin.Context) {
	modelId := c.Param("id")
	tenantId := 0 // Default tenant

	modelInfo, err := model.GetModelById(modelId)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "Model not found",
		})
		return
	}

	inputPrice, outputPrice, _ := model.GetEffectivePrice(modelId, tenantId)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"model_id":       modelId,
			"input_price":    inputPrice,
			"output_price":   outputPrice,
			"default_input":  modelInfo.InputPrice,
			"default_output": modelInfo.OutputPrice,
		},
	})
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

	user, err := model.GetUserById(userId, true)
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
	tokens, _ := model.GetAllUserTokens(userId, 0, 5, "")

	// Get recent orders
	orders, _ := model.GetUserTopupOrders(userId, "", 5, 0)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"user": gin.H{
				"id":           user.Id,
				"username":     user.Username,
				"email":        user.Email,
				"quota":        user.Quota,
				"display_name": user.DisplayName,
			},
			"usage": stats,
			"market": gin.H{
				"total_models":    marketStats.TotalModels,
				"total_providers": marketStats.TotalProviders,
			},
			"recent_tokens": tokens,
			"recent_orders": orders,
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
