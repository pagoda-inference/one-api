package model

import (
	"errors"
	"time"

	"github.com/songquanpeng/one-api/common/helper"
)

// ModelInfo represents a model available in the marketplace
type ModelInfo struct {
	Id           string   `json:"id" gorm:"primaryKey;size:64"`
	Name         string   `json:"name" gorm:"size:128"`        // 显示名称
	Provider     string   `json:"provider" gorm:"size:64"`    // 提供商
	ModelType    string   `json:"model_type" gorm:"size:32"` // chat/embedding/image/audio
	Description  string   `json:"description" gorm:"type:text"`
	ContextLen   int      `json:"context_len" gorm:"default:4096"` // 上下文长度
	InputPrice   float64  `json:"input_price" gorm:"type:decimal(10,4);default:0"`  // 输入价格(元/千token)
	OutputPrice  float64  `json:"output_price" gorm:"type:decimal(10,4);default:0"` // 输出价格(元/千token)
	Capabilities string   `json:"capabilities" gorm:"type:text"` // JSON array of capabilities
	Status       string   `json:"status" gorm:"size:32;default:active"` // active/maintenance/deprecated
	SortOrder    int      `json:"sort_order" gorm:"default:0"`
	IconUrl      string   `json:"icon_url" gorm:"size:255"`
	CreatedAt    int64    `json:"created_at" gorm:"bigint"`
	UpdatedAt    int64    `json:"updated_at" gorm:"bigint"`
}

// Model pricing types
const (
	ModelTypeChat      = "chat"
	ModelTypeEmbedding = "embedding"
	ModelTypeImage     = "image"
	ModelTypeAudio     = "audio"
	ModelTypeVideo     = "video"
)

// Model status constants
const (
	ModelStatusActive      = "active"
	ModelStatusMaintenance = "maintenance"
	ModelStatusDeprecated  = "deprecated"
)

// TableName for ModelInfo
func (ModelInfo) TableName() string {
	return "model_info"
}

// Create creates a new model info record
func (m *ModelInfo) Create() error {
	if m.Id == "" {
		return errors.New("model id is required")
	}
	if m.Name == "" {
		m.Name = m.Id
	}
	if m.CreatedAt == 0 {
		m.CreatedAt = helper.GetTimestamp()
	}
	if m.UpdatedAt == 0 {
		m.UpdatedAt = helper.GetTimestamp()
	}
	if m.Status == "" {
		m.Status = ModelStatusActive
	}
	return DB.Create(m).Error
}

// Update updates a model info record
func (m *ModelInfo) Update() error {
	m.UpdatedAt = helper.GetTimestamp()
	return DB.Save(m).Error
}

// GetModelById retrieves a model by ID
func GetModelById(id string) (*ModelInfo, error) {
	var model ModelInfo
	err := DB.First(&model, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &model, nil
}

// GetActiveModels retrieves all active models
func GetActiveModels(modelType string, limit int, offset int) ([]*ModelInfo, error) {
	var models []*ModelInfo
	query := DB.Where("status = ?", ModelStatusActive)
	if modelType != "" {
		query = query.Where("model_type = ?", modelType)
	}
	if limit <= 0 {
		limit = 100
	}
	err := query.Order("sort_order ASC, id ASC").Limit(limit).Offset(offset).Find(&models).Error
	return models, err
}

// SearchModels searches models by name or provider
func SearchModels(keyword string, modelType string, limit int, offset int) ([]*ModelInfo, error) {
	var models []*ModelInfo
	query := DB.Where("status = ?", ModelStatusActive)
	if keyword != "" {
		query = query.Where("name LIKE ? OR provider LIKE ? OR id LIKE ?",
			"%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	}
	if modelType != "" {
		query = query.Where("model_type = ?", modelType)
	}
	if limit <= 0 {
		limit = 100
	}
	err := query.Order("sort_order ASC").Limit(limit).Offset(offset).Find(&models).Error
	return models, err
}

// GetModelsByProvider retrieves models by provider
func GetModelsByProvider(provider string) ([]*ModelInfo, error) {
	var models []*ModelInfo
	err := DB.Where("status = ? AND provider = ?", ModelStatusActive, provider).
		Order("sort_order ASC").Find(&models).Error
	return models, err
}

// GetAllActiveProviders retrieves all active providers
func GetAllActiveProviders() ([]string, error) {
	var providers []string
	err := DB.Model(&ModelInfo{}).
		Where("status = ?", ModelStatusActive).
		Distinct("provider").
		Pluck("provider", &providers).Error
	return providers, err
}

// CountActiveModels counts active models
func CountActiveModels(modelType string) (int64, error) {
	var count int64
	query := DB.Model(&ModelInfo{}).Where("status = ?", ModelStatusActive)
	if modelType != "" {
		query = query.Where("model_type = ?", modelType)
	}
	err := query.Count(&count).Error
	return count, err
}

// CalculateQuota calculates quota cost for a model
func (m *ModelInfo) CalculateQuota(promptTokens, completionTokens int) int64 {
	inputQuota := int64(float64(promptTokens) * m.InputPrice)
	outputQuota := int64(float64(completionTokens) * m.OutputPrice)
	return inputQuota + outputQuota
}

// GetCapabilityList returns capabilities as a slice
func (m *ModelInfo) GetCapabilityList() []string {
	if m.Capabilities == "" {
		return []string{}
	}
	// Parse JSON array string
	var caps []string
	// Simple parsing, in production use json.Unmarshal
	return caps
}

// ModelMarketStats represents marketplace statistics
type ModelMarketStats struct {
	TotalModels      int64   `json:"total_models"`
	TotalProviders   int64   `json:"total_providers"`
	ChatModels       int64   `json:"chat_models"`
	EmbeddingModels  int64   `json:"embedding_models"`
	ImageModels      int64   `json:"image_models"`
	AvgInputPrice    float64 `json:"avg_input_price"`
	AvgOutputPrice   float64 `json:"avg_output_price"`
}

// GetModelMarketStats returns marketplace statistics
func GetModelMarketStats() (*ModelMarketStats, error) {
	stats := &ModelMarketStats{}

	// Count total active models
	DB.Model(&ModelInfo{}).Where("status = ?", ModelStatusActive).Count(&stats.TotalModels)

	// Count providers
	DB.Model(&ModelInfo{}).
		Where("status = ?", ModelStatusActive).
		Distinct("provider").
		Count(&stats.TotalProviders)

	// Count by type
	DB.Model(&ModelInfo{}).
		Where("status = ? AND model_type = ?", ModelStatusActive, ModelTypeChat).
		Count(&stats.ChatModels)

	DB.Model(&ModelInfo{}).
		Where("status = ? AND model_type = ?", ModelStatusActive, ModelTypeEmbedding).
		Count(&stats.EmbeddingModels)

	DB.Model(&ModelInfo{}).
		Where("status = ? AND model_type = ?", ModelStatusActive, ModelTypeImage).
		Count(&stats.ImageModels)

	// Average prices
	DB.Model(&ModelInfo{}).
		Where("status = ? AND input_price > 0", ModelStatusActive).
		Select("AVG(input_price)").Scan(&stats.AvgInputPrice)

	DB.Model(&ModelInfo{}).
		Where("status = ? AND output_price > 0", ModelStatusActive).
		Select("AVG(output_price)").Scan(&stats.AvgOutputPrice)

	return stats, nil
}

// InitializeDefaultModels initializes default models if none exist
func InitializeDefaultModels() error {
	var count int64
	DB.Model(&ModelInfo{}).Count(&count)
	if count > 0 {
		return nil
	}

	defaultModels := []ModelInfo{
		// OpenAI models
		{Id: "gpt-4o", Name: "GPT-4o", Provider: "OpenAI", ModelType: ModelTypeChat, Description: "OpenAI's most capable model", ContextLen: 128000, InputPrice: 0.0025, OutputPrice: 0.01, Capabilities: `["chat","function_call","vision"]`, Status: ModelStatusActive, SortOrder: 1},
		{Id: "gpt-4o-mini", Name: "GPT-4o Mini", Provider: "OpenAI", ModelType: ModelTypeChat, Description: "Fast and affordable GPT-4o", ContextLen: 128000, InputPrice: 0.00015, OutputPrice: 0.0006, Capabilities: `["chat","function_call","vision"]`, Status: ModelStatusActive, SortOrder: 2},
		{Id: "gpt-4-turbo", Name: "GPT-4 Turbo", Provider: "OpenAI", ModelType: ModelTypeChat, Description: "Previous generation GPT-4", ContextLen: 128000, InputPrice: 0.01, OutputPrice: 0.03, Capabilities: `["chat","function_call","vision"]`, Status: ModelStatusActive, SortOrder: 3},
		{Id: "gpt-3.5-turbo", Name: "GPT-3.5 Turbo", Provider: "OpenAI", ModelType: ModelTypeChat, Description: "Fast and affordable chat model", ContextLen: 16385, InputPrice: 0.0005, OutputPrice: 0.0015, Capabilities: `["chat","function_call"]`, Status: ModelStatusActive, SortOrder: 4},
		{Id: "text-embedding-3-small", Name: "Embedding 3 Small", Provider: "OpenAI", ModelType: ModelTypeEmbedding, Description: "OpenAI's small embedding model", ContextLen: 8191, InputPrice: 0.00002, OutputPrice: 0, Capabilities: `["embedding"]`, Status: ModelStatusActive, SortOrder: 10},

		// Anthropic models
		{Id: "claude-3-5-sonnet-20241022", Name: "Claude 3.5 Sonnet", Provider: "Anthropic", ModelType: ModelTypeChat, Description: "Anthropic's most intelligent model", ContextLen: 200000, InputPrice: 0.003, OutputPrice: 0.015, Capabilities: `["chat","vision","tool_use"]`, Status: ModelStatusActive, SortOrder: 5},
		{Id: "claude-3-5-haiku-20241022", Name: "Claude 3.5 Haiku", Provider: "Anthropic", ModelType: ModelTypeChat, Description: "Fast and affordable Claude", ContextLen: 200000, InputPrice: 0.0008, OutputPrice: 0.004, Capabilities: `["chat","vision"]`, Status: ModelStatusActive, SortOrder: 6},
		{Id: "claude-3-opus-20240229", Name: "Claude 3 Opus", Provider: "Anthropic", ModelType: ModelTypeChat, Description: "Most capable Claude for complex tasks", ContextLen: 200000, InputPrice: 0.015, OutputPrice: 0.075, Capabilities: `["chat","vision","tool_use"]`, Status: ModelStatusActive, SortOrder: 7},

		// Google models
		{Id: "gemini-1.5-pro", Name: "Gemini 1.5 Pro", Provider: "Google", ModelType: ModelTypeChat, Description: "Google's most capable multimodal model", ContextLen: 2000000, InputPrice: 0.00125, OutputPrice: 0.005, Capabilities: `["chat","vision","audio"]`, Status: ModelStatusActive, SortOrder: 8},
		{Id: "gemini-1.5-flash", Name: "Gemini 1.5 Flash", Provider: "Google", ModelType: ModelTypeChat, Description: "Fast and versatile Google model", ContextLen: 1000000, InputPrice: 0.000075, OutputPrice: 0.0003, Capabilities: `["chat","vision","audio"]`, Status: ModelStatusActive, SortOrder: 9},

		// Custom vLLM models (your internal deployment)
		{Id: "glm-4", Name: "GLM-4", Provider: "Internal", ModelType: ModelTypeChat, Description: "GLM-4 via vLLM", ContextLen: 128000, InputPrice: 0, OutputPrice: 0, Capabilities: `["chat"]`, Status: ModelStatusActive, SortOrder: 100},
		{Id: "qwen-2.5", Name: "Qwen 2.5", Provider: "Internal", ModelType: ModelTypeChat, Description: "Qwen 2.5 via vLLM", ContextLen: 32000, InputPrice: 0, OutputPrice: 0, Capabilities: `["chat"]`, Status: ModelStatusActive, SortOrder: 101},
	}

	for i := range defaultModels {
		if err := defaultModels[i].Create(); err != nil {
			return err
		}
	}

	return nil
}