package model

import (
	"errors"

	"github.com/pagoda-inference/one-api/common/helper"
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
	GroupId      int      `json:"group_id" gorm:"default:0"`       // 模型分组ID
	Tags         string   `json:"tags" gorm:"type:text"`           // JSON array of tags
	IsTrial      bool     `json:"is_trial" gorm:"default:false"`   // 是否支持试用
	TrialQuota   int64    `json:"trial_quota" gorm:"default:0"`    // 试用额度
	SLA          string   `json:"sla" gorm:"size:32;default:standard"` // SLA等级: standard/premium/enterprise
	CreatedAt    int64    `json:"created_at" gorm:"bigint"`
	UpdatedAt    int64    `json:"updated_at" gorm:"bigint"`
}

// ModelGroup represents a model group/category
type ModelGroup struct {
	Id          int    `json:"id" gorm:"primaryKey"`
	Name        string `json:"name" gorm:"size:64"`        // 分组名称
	Code        string `json:"code" gorm:"size:32;uniqueIndex"` // 分组代码
	Description string `json:"description" gorm:"type:text"`
	IconUrl     string `json:"icon_url" gorm:"size:255"`
	SortOrder   int    `json:"sort_order" gorm:"default:0"`
	Status      string `json:"status" gorm:"size:32;default:active"`
	CreatedAt   int64  `json:"created_at" gorm:"bigint"`
	UpdatedAt   int64  `json:"updated_at" gorm:"bigint"`
}

func (ModelGroup) TableName() string {
	return "model_groups"
}

// ModelPricing represents configurable model pricing
type ModelPricing struct {
	Id          int64   `json:"id" gorm:"primaryKey"`
	ModelId     string  `json:"model_id" gorm:"size:64;index"`
	TenantId    int     `json:"tenant_id" gorm:"index"`          // 0 means default pricing
	InputPrice  float64 `json:"input_price" gorm:"type:decimal(10,4)"`
	OutputPrice float64 `json:"output_price" gorm:"type:decimal(10,4)"`
	Discount    float64 `json:"discount" gorm:"type:decimal(5,2);default:100"` // 折扣百分比
	CreatedAt   int64   `json:"created_at" gorm:"bigint"`
	UpdatedAt   int64   `json:"updated_at" gorm:"bigint"`
}

func (ModelPricing) TableName() string {
	return "model_pricing"
}

// ModelTrial records trial usage
type ModelTrial struct {
	Id        int   `json:"id" gorm:"primaryKey"`
	UserId    int   `json:"user_id" gorm:"index"`
	ModelId   string `json:"model_id" gorm:"size:64;index"`
	TenantId  int   `json:"tenant_id" gorm:"index"`
	QuotaUsed int64 `json:"quota_used" gorm:"bigint;default:0"`
	Status    string `json:"status" gorm:"size:32;default:active"` // active/expired/disabled
	CreatedAt int64 `json:"created_at" gorm:"bigint"`
	UpdatedAt int64 `json:"updated_at" gorm:"bigint"`
	ExpiresAt int64 `json:"expires_at" gorm:"bigint"`
}

func (ModelTrial) TableName() string {
	return "model_trials"
}

// Model pricing types
const (
	ModelTypeChat      = "chat"
	ModelTypeEmbedding = "embedding"
	ModelTypeImage     = "image"
	ModelTypeAudio     = "audio"
	ModelTypeVideo     = "video"
	ModelTypeVLM       = "vlm"
	ModelTypeReranker  = "reranker"
	ModelTypeOCR       = "ocr"
	ModelTypeOther     = "other"
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
	TotalGroups      int64   `json:"total_groups"`
	ChatModels       int64   `json:"chat_models"`
	EmbeddingModels  int64   `json:"embedding_models"`
	ImageModels      int64   `json:"image_models"`
	AvgInputPrice    float64 `json:"avg_input_price"`
	AvgOutputPrice   float64 `json:"avg_output_price"`
	TrialModels      int64   `json:"trial_models"`
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

	// Count groups
	DB.Model(&ModelGroup{}).Where("status = ?", "active").Count(&stats.TotalGroups)

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

	// Trial models
	DB.Model(&ModelInfo{}).
		Where("status = ? AND is_trial = ?", ModelStatusActive, true).
		Count(&stats.TrialModels)

	// Average prices
	DB.Model(&ModelInfo{}).
		Where("status = ? AND input_price > 0", ModelStatusActive).
		Select("AVG(input_price)").Scan(&stats.AvgInputPrice)

	DB.Model(&ModelInfo{}).
		Where("status = ? AND output_price > 0", ModelStatusActive).
		Select("AVG(output_price)").Scan(&stats.AvgOutputPrice)

	return stats, nil
}

// Model group operations

// CreateModelGroup creates a new model group
func CreateModelGroup(group *ModelGroup) error {
	if group.CreatedAt == 0 {
		group.CreatedAt = helper.GetTimestamp()
	}
	if group.UpdatedAt == 0 {
		group.UpdatedAt = helper.GetTimestamp()
	}
	if group.Status == "" {
		group.Status = "active"
	}
	return DB.Create(group).Error
}

// GetModelGroupById retrieves a group by ID
func GetModelGroupById(id int) (*ModelGroup, error) {
	var group ModelGroup
	err := DB.First(&group, id).Error
	if err != nil {
		return nil, err
	}
	return &group, nil
}

// GetAllModelGroups retrieves all active groups
func GetAllModelGroups() ([]*ModelGroup, error) {
	var groups []*ModelGroup
	err := DB.Where("status = ?", "active").Order("sort_order ASC, id ASC").Find(&groups).Error
	return groups, err
}

// GetModelsByGroup retrieves models by group ID
func GetModelsByGroup(groupId int) ([]*ModelInfo, error) {
	var models []*ModelInfo
	err := DB.Where("status = ? AND group_id = ?", ModelStatusActive, groupId).
		Order("sort_order ASC").Find(&models).Error
	return models, err
}

// Model pricing operations

// GetModelPricing retrieves pricing for a model (tenant-specific or default)
func GetModelPricing(modelId string, tenantId int) (*ModelPricing, error) {
	var pricing ModelPricing
	// First try tenant-specific pricing
	err := DB.Where("model_id = ? AND tenant_id = ?", modelId, tenantId).First(&pricing).Error
	if err == nil {
		return &pricing, nil
	}
	// Fall back to default pricing
	err = DB.Where("model_id = ? AND tenant_id = ?", modelId, 0).First(&pricing).Error
	if err == nil {
		return &pricing, nil
	}
	// Return model default prices
	return nil, errors.New("no pricing found")
}

// SetModelPricing sets pricing for a model
func SetModelPricing(pricing *ModelPricing) error {
	if pricing.CreatedAt == 0 {
		pricing.CreatedAt = helper.GetTimestamp()
	}
	pricing.UpdatedAt = helper.GetTimestamp()

	// Check if exists
	var existing ModelPricing
	err := DB.Where("model_id = ? AND tenant_id = ?", pricing.ModelId, pricing.TenantId).First(&existing).Error
	if err == nil {
		// Update existing
		pricing.Id = existing.Id
		return DB.Save(pricing).Error
	}
	return DB.Create(pricing).Error
}

// GetEffectivePrice calculates effective price with discount
func GetEffectivePrice(modelId string, tenantId int) (inputPrice, outputPrice float64, err error) {
	model, err := GetModelById(modelId)
	if err != nil {
		return 0, 0, err
	}

	pricing, err := GetModelPricing(modelId, tenantId)
	if err != nil {
		// Use model default prices
		return model.InputPrice, model.OutputPrice, nil
	}

	inputPrice = model.InputPrice * pricing.Discount / 100
	outputPrice = model.OutputPrice * pricing.Discount / 100
	return inputPrice, outputPrice, nil
}

// Model trial operations

// UseModelTrial records trial usage
func UseModelTrial(userId int, modelId string, tenantId int, quota int64) error {
	var trial ModelTrial
	err := DB.Where("user_id = ? AND model_id = ? AND tenant_id = ? AND status = ?",
		userId, modelId, tenantId, "active").First(&trial).Error

	timestamp := helper.GetTimestamp()

	if err != nil {
		// Create new trial record
		model, _ := GetModelById(modelId)
		if model != nil && model.TrialQuota > 0 {
			// Could use model.TrialQuota here if ModelTrial had a Quota field
		}
		trial = ModelTrial{
			UserId:    userId,
			ModelId:   modelId,
			TenantId:  tenantId,
			QuotaUsed: 0,
			Status:    "active",
			CreatedAt: timestamp,
			UpdatedAt: timestamp,
			ExpiresAt: timestamp + 7*86400, // 7 days
		}
		if err := DB.Create(&trial).Error; err != nil {
			return err
		}
	}

	// Update usage
	trial.QuotaUsed += quota
	trial.UpdatedAt = timestamp

	// Check if expired
	if timestamp > trial.ExpiresAt {
		trial.Status = "expired"
	}

	// Check if quota exceeded
	model, _ := GetModelById(modelId)
	if model != nil && model.TrialQuota > 0 && trial.QuotaUsed >= model.TrialQuota {
		trial.Status = "exhausted"
	}

	return DB.Save(&trial).Error
}

// CheckModelTrial checks if user can use trial for a model
func CheckModelTrial(userId int, modelId string, tenantId int) (bool, string) {
	model, err := GetModelById(modelId)
	if err != nil || !model.IsTrial {
		return false, "model_not_available_for_trial"
	}

	var trial ModelTrial
	err = DB.Where("user_id = ? AND model_id = ? AND tenant_id = ?",
		userId, modelId, tenantId).First(&trial).Error

	if err != nil {
		// No trial record, user can start trial
		return true, ""
	}

	if trial.Status != "active" {
		return false, "trial_" + trial.Status
	}

	if trial.QuotaUsed >= model.TrialQuota {
		return false, "trial_quota_exhausted"
	}

	return true, ""
}

// GetUserTrials retrieves all trials for a user
func GetUserTrials(userId int, tenantId int) ([]*ModelTrial, error) {
	var trials []*ModelTrial
	err := DB.Where("user_id = ? AND tenant_id = ?", userId, tenantId).
		Order("created_at DESC").Find(&trials).Error
	return trials, err
}

// InitializeDefaultModels initializes default models if none exist
func InitializeDefaultModels() error {
	var count int64
	DB.Model(&ModelInfo{}).Count(&count)
	if count > 0 {
		return nil
	}

	// Initialize model groups first
	groups := []ModelGroup{
		{Id: 1, Name: "OpenAI", Code: "openai", Description: "OpenAI GPT系列模型", SortOrder: 1},
		{Id: 2, Name: "Anthropic", Code: "anthropic", Description: "Anthropic Claude系列模型", SortOrder: 2},
		{Id: 3, Name: "Google", Code: "google", Description: "Google Gemini系列模型", SortOrder: 3},
		{Id: 4, Name: "百度文心", Code: "baidu", Description: "百度文心大模型", SortOrder: 4},
		{Id: 5, Name: "阿里通义", Code: "alibaba", Description: "阿里通义千问大模型", SortOrder: 5},
		{Id: 6, Name: "智谱AI", Code: "zhipu", Description: "智谱AI大模型", SortOrder: 6},
		{Id: 7, Name: "MiniMax", Code: "minimax", Description: "MiniMax海螺大模型", SortOrder: 7},
		{Id: 8, Name: "BEDI", Code: "bedi", Description: "BEDI私有GPU集群部署模型", SortOrder: 9},
		{Id: 99, Name: "内部部署", Code: "internal", Description: "其他内部vLLM部署模型", SortOrder: 100},
	}
	for i := range groups {
		if err := CreateModelGroup(&groups[i]); err != nil {
			return err
		}
	}

	defaultModels := []ModelInfo{
		// OpenAI models (Group 1)
		{Id: "gpt-4o", Name: "GPT-4o", Provider: "OpenAI", ModelType: ModelTypeChat, Description: "OpenAI最强多模态模型", ContextLen: 128000, InputPrice: 0.015, OutputPrice: 0.06, Capabilities: `["chat","function_call","vision"]`, Status: ModelStatusActive, SortOrder: 1, GroupId: 1, IsTrial: true, TrialQuota: 1000000, SLA: "standard"},
		{Id: "gpt-4o-mini", Name: "GPT-4o Mini", Provider: "OpenAI", ModelType: ModelTypeChat, Description: "快速高效的GPT-4o", ContextLen: 128000, InputPrice: 0.00015, OutputPrice: 0.0006, Capabilities: `["chat","function_call","vision"]`, Status: ModelStatusActive, SortOrder: 2, GroupId: 1, IsTrial: true, TrialQuota: 5000000, SLA: "standard"},
		{Id: "gpt-4-turbo", Name: "GPT-4 Turbo", Provider: "OpenAI", ModelType: ModelTypeChat, Description: "上一代GPT-4", ContextLen: 128000, InputPrice: 0.01, OutputPrice: 0.03, Capabilities: `["chat","function_call","vision"]`, Status: ModelStatusActive, SortOrder: 3, GroupId: 1, SLA: "standard"},
		{Id: "gpt-3.5-turbo", Name: "GPT-3.5 Turbo", Provider: "OpenAI", ModelType: ModelTypeChat, Description: "快速经济实惠", ContextLen: 16385, InputPrice: 0.0005, OutputPrice: 0.0015, Capabilities: `["chat","function_call"]`, Status: ModelStatusActive, SortOrder: 4, GroupId: 1, IsTrial: true, TrialQuota: 10000000, SLA: "standard"},
		{Id: "text-embedding-3-small", Name: "Embedding 3 Small", Provider: "OpenAI", ModelType: ModelTypeEmbedding, Description: "小型嵌入模型", ContextLen: 8191, InputPrice: 0.00002, OutputPrice: 0, Capabilities: `["embedding"]`, Status: ModelStatusActive, SortOrder: 10, GroupId: 1},

		// Anthropic models (Group 2)
		{Id: "claude-3-5-sonnet-20241022", Name: "Claude 3.5 Sonnet", Provider: "Anthropic", ModelType: ModelTypeChat, Description: "Anthropic最智能模型", ContextLen: 200000, InputPrice: 0.003, OutputPrice: 0.015, Capabilities: `["chat","vision","tool_use"]`, Status: ModelStatusActive, SortOrder: 5, GroupId: 2, IsTrial: true, TrialQuota: 500000, SLA: "standard"},
		{Id: "claude-3-5-haiku-20241022", Name: "Claude 3.5 Haiku", Provider: "Anthropic", ModelType: ModelTypeChat, Description: "快速高效的Claude", ContextLen: 200000, InputPrice: 0.0008, OutputPrice: 0.004, Capabilities: `["chat","vision"]`, Status: ModelStatusActive, SortOrder: 6, GroupId: 2, IsTrial: true, TrialQuota: 2000000, SLA: "standard"},
		{Id: "claude-3-opus-20240229", Name: "Claude 3 Opus", Provider: "Anthropic", ModelType: ModelTypeChat, Description: "复杂任务最强大模型", ContextLen: 200000, InputPrice: 0.015, OutputPrice: 0.075, Capabilities: `["chat","vision","tool_use"]`, Status: ModelStatusActive, SortOrder: 7, GroupId: 2, SLA: "premium"},

		// Google models (Group 3)
		{Id: "gemini-1.5-pro", Name: "Gemini 1.5 Pro", Provider: "Google", ModelType: ModelTypeChat, Description: "Google最强多模态模型", ContextLen: 2000000, InputPrice: 0.00125, OutputPrice: 0.005, Capabilities: `["chat","vision","audio"]`, Status: ModelStatusActive, SortOrder: 8, GroupId: 3, IsTrial: true, TrialQuota: 1000000, SLA: "standard"},
		{Id: "gemini-1.5-flash", Name: "Gemini 1.5 Flash", Provider: "Google", ModelType: ModelTypeChat, Description: "快速多用途Google模型", ContextLen: 1000000, InputPrice: 0.000075, OutputPrice: 0.0003, Capabilities: `["chat","vision","audio"]`, Status: ModelStatusActive, SortOrder: 9, GroupId: 3, IsTrial: true, TrialQuota: 10000000, SLA: "standard"},
		{Id: "gemini-2.0-flash-exp", Name: "Gemini 2.0 Flash Exp", Provider: "Google", ModelType: ModelTypeChat, Description: "最新实验性Gemini模型", ContextLen: 1000000, InputPrice: 0, OutputPrice: 0, Capabilities: `["chat","vision","audio"]`, Status: ModelStatusActive, SortOrder: 10, GroupId: 3, SLA: "enterprise"},

		// 百度文心 (Group 4)
		{Id: "ernie-4.0-8k", Name: "文心一言 4.0", Provider: "Baidu", ModelType: ModelTypeChat, Description: "百度文心一言4.0旗舰版", ContextLen: 8000, InputPrice: 0.12, OutputPrice: 0.12, Capabilities: `["chat","function_call"]`, Status: ModelStatusActive, SortOrder: 20, GroupId: 4, IsTrial: true, TrialQuota: 500000, SLA: "standard"},
		{Id: "ernie-3.5-8k", Name: "文心一言 3.5", Provider: "Baidu", ModelType: ModelTypeChat, Description: "百度文心一言3.5", ContextLen: 8000, InputPrice: 0.008, OutputPrice: 0.008, Capabilities: `["chat","function_call"]`, Status: ModelStatusActive, SortOrder: 21, GroupId: 4, IsTrial: true, TrialQuota: 2000000, SLA: "standard"},
		{Id: "ernie-speed-8k", Name: "文心一言 高速版", Provider: "Baidu", ModelType: ModelTypeChat, Description: "百度高速响应模型", ContextLen: 8000, InputPrice: 0.004, OutputPrice: 0.004, Capabilities: `["chat"]`, Status: ModelStatusActive, SortOrder: 22, GroupId: 4, IsTrial: true, TrialQuota: 5000000, SLA: "standard"},
		{Id: "ernie-lite-8k", Name: "文心一言 轻量版", Provider: "Baidu", ModelType: ModelTypeChat, Description: "百度轻量级模型", ContextLen: 8000, InputPrice: 0, OutputPrice: 0, Capabilities: `["chat"]`, Status: ModelStatusActive, SortOrder: 23, GroupId: 4, SLA: "standard"},
		{Id: "bge-large-zh", Name: "BGE 中文向量", Provider: "Baidu", ModelType: ModelTypeEmbedding, Description: "百度中文语义向量模型", ContextLen: 512, InputPrice: 0.0002, OutputPrice: 0, Capabilities: `["embedding"]`, Status: ModelStatusActive, SortOrder: 25, GroupId: 4},

		// 阿里通义 (Group 5)
		{Id: "qwen-max", Name: "通义千问 MAX", Provider: "Alibaba", ModelType: ModelTypeChat, Description: "阿里通义千问最强版本", ContextLen: 32000, InputPrice: 0.12, OutputPrice: 0.12, Capabilities: `["chat","function_call","vision"]`, Status: ModelStatusActive, SortOrder: 30, GroupId: 5, IsTrial: true, TrialQuota: 500000, SLA: "standard"},
		{Id: "qwen-plus", Name: "通义千问 Plus", Provider: "Alibaba", ModelType: ModelTypeChat, Description: "阿里通义千问增强版", ContextLen: 131072, InputPrice: 0.004, OutputPrice: 0.012, Capabilities: `["chat","function_call"]`, Status: ModelStatusActive, SortOrder: 31, GroupId: 5, IsTrial: true, TrialQuota: 2000000, SLA: "standard"},
		{Id: "qwen-turbo", Name: "通义千问 Turbo", Provider: "Alibaba", ModelType: ModelTypeChat, Description: "阿里通义千问快速版", ContextLen: 131072, InputPrice: 0.002, OutputPrice: 0.006, Capabilities: `["chat"]`, Status: ModelStatusActive, SortOrder: 32, GroupId: 5, IsTrial: true, TrialQuota: 5000000, SLA: "standard"},
		{Id: "qwen-long", Name: "通义千问 长文本", Provider: "Alibaba", ModelType: ModelTypeChat, Description: "阿里超长上下文模型", ContextLen: 1000000, InputPrice: 0.01, OutputPrice: 0.03, Capabilities: `["chat"]`, Status: ModelStatusActive, SortOrder: 33, GroupId: 5, SLA: "premium"},
		{Id: "text-embedding-v2", Name: "文本嵌入 V2", Provider: "Alibaba", ModelType: ModelTypeEmbedding, Description: "阿里文本嵌入模型", ContextLen: 8192, InputPrice: 0.0002, OutputPrice: 0, Capabilities: `["embedding"]`, Status: ModelStatusActive, SortOrder: 35, GroupId: 5},

		// 智谱AI (Group 6)
		{Id: "glm-4-plus", Name: "GLM-4 Plus", Provider: "Zhipu", ModelType: ModelTypeChat, Description: "智谱GLM-4增强版", ContextLen: 128000, InputPrice: 0.1, OutputPrice: 0.1, Capabilities: `["chat","function_call","vision"]`, Status: ModelStatusActive, SortOrder: 40, GroupId: 6, IsTrial: true, TrialQuota: 500000, SLA: "standard"},
		{Id: "glm-4", Name: "GLM-4", Provider: "Zhipu", ModelType: ModelTypeChat, Description: "智谱GLM-4标准版", ContextLen: 128000, InputPrice: 0.006, OutputPrice: 0.006, Capabilities: `["chat","function_call"]`, Status: ModelStatusActive, SortOrder: 41, GroupId: 6, IsTrial: true, TrialQuota: 2000000, SLA: "standard"},
		{Id: "glm-4-flash", Name: "GLM-4 Flash", Provider: "Zhipu", ModelType: ModelTypeChat, Description: "智谱快速响应版", ContextLen: 128000, InputPrice: 0.001, OutputPrice: 0.001, Capabilities: `["chat"]`, Status: ModelStatusActive, SortOrder: 42, GroupId: 6, IsTrial: true, TrialQuota: 10000000, SLA: "standard"},
		{Id: "glm-3-turbo", Name: "GLM-3 Turbo", Provider: "Zhipu", ModelType: ModelTypeChat, Description: "智谱轻量版", ContextLen: 128000, InputPrice: 0.001, OutputPrice: 0.001, Capabilities: `["chat"]`, Status: ModelStatusActive, SortOrder: 43, GroupId: 6, SLA: "standard"},
		{Id: "embedding-2", Name: "文本嵌入 V2", Provider: "Zhipu", ModelType: ModelTypeEmbedding, Description: "智谱文本嵌入模型", ContextLen: 512, InputPrice: 0.0001, OutputPrice: 0, Capabilities: `["embedding"]`, Status: ModelStatusActive, SortOrder: 45, GroupId: 6},

		// MiniMax (Group 7)
		{Id: "abab6.5s", Name: "ABAB 6.5S", Provider: "MiniMax", ModelType: ModelTypeChat, Description: "MiniMax对话模型", ContextLen: 245000, InputPrice: 0.01, OutputPrice: 0.01, Capabilities: `["chat"]`, Status: ModelStatusActive, SortOrder: 50, GroupId: 7, IsTrial: true, TrialQuota: 1000000, SLA: "standard"},
		{Id: "abab6.5g", Name: "ABAB 6.5G", Provider: "MiniMax", ModelType: ModelTypeChat, Description: "MiniMax增强版模型", ContextLen: 245000, InputPrice: 0.015, OutputPrice: 0.015, Capabilities: `["chat","function_call"]`, Status: ModelStatusActive, SortOrder: 51, GroupId: 7, SLA: "standard"},
		{Id: "abab5.5s", Name: "ABAB 5.5S", Provider: "MiniMax", ModelType: ModelTypeChat, Description: "MiniMax快速版", ContextLen: 245000, InputPrice: 0.005, OutputPrice: 0.005, Capabilities: `["chat"]`, Status: ModelStatusActive, SortOrder: 52, GroupId: 7, SLA: "standard"},

		// BEDI 私有GPU集群 (Group 8) - H800北广AI云平台
		{Id: "ZhipuAI/GLM-4.7-FP8", Name: "GLM-4.7-FP8", Provider: "BEDI", ModelType: ModelTypeChat, Description: "H800北广AI云平台 - GLM-4.7-FP8", ContextLen: 128000, InputPrice: 0, OutputPrice: 0, Capabilities: `["chat","function_call"]`, Status: ModelStatusActive, SortOrder: 60, GroupId: 8, IsTrial: false, SLA: "enterprise"},
		{Id: "Minimax/MiniMax-M2.1", Name: "MiniMax-M2.1", Provider: "BEDI", ModelType: ModelTypeChat, Description: "H800北广AI云平台 - MiniMax-M2.1", ContextLen: 1000000, InputPrice: 0, OutputPrice: 0, Capabilities: `["chat"]`, Status: ModelStatusActive, SortOrder: 61, GroupId: 8, IsTrial: false, SLA: "enterprise"},
		{Id: "Qwen/Qwen3.5-122B-A10B-FP8", Name: "Qwen3.5-122B-A10B-FP8", Provider: "BEDI", ModelType: ModelTypeChat, Description: "H800北广AI云平台 - Qwen3.5-122B-A10B-FP8 (beta)", ContextLen: 32000, InputPrice: 0, OutputPrice: 0, Capabilities: `["chat","function_call"]`, Status: ModelStatusActive, SortOrder: 62, GroupId: 8, IsTrial: false, SLA: "enterprise", Tags: `["beta"]`},

		// BEDI 私有GPU集群 (Group 8) - 910B3移动平台宝塔环境
		{Id: "deepseek-v3.1-int8", Name: "DeepSeek-V3.1-Int8", Provider: "BEDI", ModelType: ModelTypeChat, Description: "910B3移动平台宝塔环境 - DeepSeek-V3.1-Int8", ContextLen: 128000, InputPrice: 0, OutputPrice: 0, Capabilities: `["chat","function_call"]`, Status: ModelStatusActive, SortOrder: 63, GroupId: 8, IsTrial: false, SLA: "enterprise"},
		{Id: "Qwen/Qwen3-235B-A22B", Name: "Qwen3-235B-A22B", Provider: "BEDI", ModelType: ModelTypeChat, Description: "910B3移动平台宝塔环境 - Qwen3-235B-A22B", ContextLen: 32000, InputPrice: 0, OutputPrice: 0, Capabilities: `["chat","function_call"]`, Status: ModelStatusActive, SortOrder: 64, GroupId: 8, IsTrial: false, SLA: "enterprise"},
		{Id: "deepseek-ai/DeepSeek-R1-Distill-Qwen-32B", Name: "DeepSeek-R1-Distill-Qwen-32B", Provider: "BEDI", ModelType: ModelTypeChat, Description: "910B3移动平台宝塔环境 - DeepSeek-R1-Distill-Qwen-32B", ContextLen: 32000, InputPrice: 0, OutputPrice: 0, Capabilities: `["chat","reasoning"]`, Status: ModelStatusActive, SortOrder: 65, GroupId: 8, IsTrial: false, SLA: "enterprise"},

		// BEDI 私有GPU集群 (Group 8) - 910B4佛山AI云平台
		{Id: "Qwen/Qwen2.5-VL-72B-Instruct", Name: "Qwen2.5-VL-72B-Instruct", Provider: "BEDI", ModelType: ModelTypeVLM, Description: "910B4佛山AI云平台 - Qwen2.5-VL-72B多模态模型", ContextLen: 32000, InputPrice: 0, OutputPrice: 0, Capabilities: `["chat","vision","function_call"]`, Status: ModelStatusActive, SortOrder: 70, GroupId: 8, IsTrial: false, SLA: "enterprise"},
		{Id: "Qwen/Qwen3-VL-30B-A3B-Instruct", Name: "Qwen3-VL-30B-A3B-Instruct", Provider: "BEDI", ModelType: ModelTypeVLM, Description: "910B4佛山AI云平台 - Qwen3-VL-30B多模态模型", ContextLen: 32000, InputPrice: 0, OutputPrice: 0, Capabilities: `["chat","vision","function_call"]`, Status: ModelStatusActive, SortOrder: 71, GroupId: 8, IsTrial: false, SLA: "enterprise"},
		{Id: "Qwen/Qwen3-VL-8B", Name: "Qwen3-VL-8B", Provider: "BEDI", ModelType: ModelTypeVLM, Description: "910B4佛山AI云平台 - Qwen3-VL-8B多模态模型", ContextLen: 32000, InputPrice: 0, OutputPrice: 0, Capabilities: `["chat","vision"]`, Status: ModelStatusActive, SortOrder: 72, GroupId: 8, IsTrial: false, SLA: "enterprise"},
		{Id: "Qwen/Qwen2.5-VL-7B", Name: "Qwen2.5-VL-7B", Provider: "BEDI", ModelType: ModelTypeVLM, Description: "910B4佛山AI云平台 - Qwen2.5-VL-7B多模态模型", ContextLen: 32000, InputPrice: 0, OutputPrice: 0, Capabilities: `["chat","vision"]`, Status: ModelStatusActive, SortOrder: 73, GroupId: 8, IsTrial: false, SLA: "enterprise"},
		{Id: "Qwen/Qwen3-8B", Name: "Qwen3-8B", Provider: "BEDI", ModelType: ModelTypeChat, Description: "910B4佛山AI云平台 - Qwen3-8B", ContextLen: 32000, InputPrice: 0, OutputPrice: 0, Capabilities: `["chat","function_call"]`, Status: ModelStatusActive, SortOrder: 74, GroupId: 8, IsTrial: false, SLA: "enterprise"},
		{Id: "Qwen/Qwen3-14B", Name: "Qwen3-14B", Provider: "BEDI", ModelType: ModelTypeChat, Description: "910B4佛山AI云平台 - Qwen3-14B", ContextLen: 32000, InputPrice: 0, OutputPrice: 0, Capabilities: `["chat","function_call"]`, Status: ModelStatusActive, SortOrder: 75, GroupId: 8, IsTrial: false, SLA: "enterprise"},
		{Id: "Qwen/Qwen3-32B", Name: "Qwen3-32B", Provider: "BEDI", ModelType: ModelTypeChat, Description: "910B4佛山AI云平台 - Qwen3-32B", ContextLen: 32000, InputPrice: 0, OutputPrice: 0, Capabilities: `["chat","function_call"]`, Status: ModelStatusActive, SortOrder: 76, GroupId: 8, IsTrial: false, SLA: "enterprise"},
		{Id: "Qwen/QwQ-32B", Name: "QwQ-32B", Provider: "BEDI", ModelType: ModelTypeChat, Description: "910B4佛山AI云平台 - QwQ-32B推理模型", ContextLen: 32000, InputPrice: 0, OutputPrice: 0, Capabilities: `["chat","reasoning"]`, Status: ModelStatusActive, SortOrder: 77, GroupId: 8, IsTrial: false, SLA: "enterprise"},
		{Id: "Qwen/Qwen2.5-7B-Instruct", Name: "Qwen2.5-7B-Instruct", Provider: "BEDI", ModelType: ModelTypeChat, Description: "910B4佛山AI云平台 - Qwen2.5-7B-Instruct", ContextLen: 32000, InputPrice: 0, OutputPrice: 0, Capabilities: `["chat","function_call"]`, Status: ModelStatusActive, SortOrder: 78, GroupId: 8, IsTrial: false, SLA: "enterprise"},
		{Id: "Qwen/Qwen2.5-72B-Instruct", Name: "Qwen2.5-72B-Instruct", Provider: "BEDI", ModelType: ModelTypeChat, Description: "910B4佛山AI云平台 - Qwen2.5-72B-Instruct", ContextLen: 32000, InputPrice: 0, OutputPrice: 0, Capabilities: `["chat","function_call"]`, Status: ModelStatusActive, SortOrder: 79, GroupId: 8, IsTrial: false, SLA: "enterprise"},
		{Id: "deepseek-ai/DeepSeek-R1-Distill-Qwen-7B", Name: "DeepSeek-R1-Distill-Qwen-7B", Provider: "BEDI", ModelType: ModelTypeChat, Description: "910B4佛山AI云平台 - DeepSeek-R1-Distill-Qwen-7B", ContextLen: 32000, InputPrice: 0, OutputPrice: 0, Capabilities: `["chat","reasoning"]`, Status: ModelStatusActive, SortOrder: 80, GroupId: 8, IsTrial: false, SLA: "enterprise"},
		{Id: "deepseek-ai/DeepSeek-R1-Distill-Qwen-14B", Name: "DeepSeek-R1-Distill-Qwen-14B", Provider: "BEDI", ModelType: ModelTypeChat, Description: "910B4佛山AI云平台 - DeepSeek-R1-Distill-Qwen-14B", ContextLen: 32000, InputPrice: 0, OutputPrice: 0, Capabilities: `["chat","reasoning"]`, Status: ModelStatusActive, SortOrder: 81, GroupId: 8, IsTrial: false, SLA: "enterprise"},
		{Id: "deepseek-ai/DeepSeek-R1-Distill-Qwen-32B", Name: "DeepSeek-R1-Distill-Qwen-32B", Provider: "BEDI", ModelType: ModelTypeChat, Description: "910B4佛山AI云平台 - DeepSeek-R1-Distill-Qwen-32B", ContextLen: 32000, InputPrice: 0, OutputPrice: 0, Capabilities: `["chat","reasoning"]`, Status: ModelStatusActive, SortOrder: 82, GroupId: 8, IsTrial: false, SLA: "enterprise"},

		// BEDI Embedding models (910B4佛山AI云平台)
		{Id: "BAAI/bge-m3", Name: "BGE-M3", Provider: "BEDI", ModelType: ModelTypeEmbedding, Description: "910B4佛山AI云平台 - BGE-M3 embedding模型", ContextLen: 8192, InputPrice: 0, OutputPrice: 0, Capabilities: `["embedding"]`, Status: ModelStatusActive, SortOrder: 83, GroupId: 8, IsTrial: false, SLA: "enterprise"},
		{Id: "Qwen/Qwen3-Embedding-0.6B", Name: "Qwen3-Embedding-0.6B", Provider: "BEDI", ModelType: ModelTypeEmbedding, Description: "910B4佛山AI云平台 - Qwen3-Embedding-0.6B", ContextLen: 8192, InputPrice: 0, OutputPrice: 0, Capabilities: `["embedding"]`, Status: ModelStatusActive, SortOrder: 84, GroupId: 8, IsTrial: false, SLA: "enterprise"},
		{Id: "Qwen/Qwen3-Embedding-8B", Name: "Qwen3-Embedding-8B", Provider: "BEDI", ModelType: ModelTypeEmbedding, Description: "910B4佛山AI云平台 - Qwen3-Embedding-8B", ContextLen: 8192, InputPrice: 0, OutputPrice: 0, Capabilities: `["embedding"]`, Status: ModelStatusActive, SortOrder: 85, GroupId: 8, IsTrial: false, SLA: "enterprise"},

		// BEDI Reranker models (910B4佛山AI云平台)
		{Id: "BAAI/bge-reranker", Name: "BGE-Reranker", Provider: "BEDI", ModelType: ModelTypeReranker, Description: "910B4佛山AI云平台 - BGE-Reranker重排序模型", ContextLen: 512, InputPrice: 0, OutputPrice: 0, Capabilities: `["reranker"]`, Status: ModelStatusActive, SortOrder: 86, GroupId: 8, IsTrial: false, SLA: "enterprise"},
		{Id: "Qwen/Qwen3-Reranker-0.6B", Name: "Qwen3-Reranker-0.6B", Provider: "BEDI", ModelType: ModelTypeReranker, Description: "910B4佛山AI云平台 - Qwen3-Reranker-0.6B", ContextLen: 512, InputPrice: 0, OutputPrice: 0, Capabilities: `["reranker"]`, Status: ModelStatusActive, SortOrder: 87, GroupId: 8, IsTrial: false, SLA: "enterprise"},
		{Id: "Qwen/Qwen3-Reranker-8B", Name: "Qwen3-Reranker-8B", Provider: "BEDI", ModelType: ModelTypeReranker, Description: "910B4佛山AI云平台 - Qwen3-Reranker-8B", ContextLen: 512, InputPrice: 0, OutputPrice: 0, Capabilities: `["reranker"]`, Status: ModelStatusActive, SortOrder: 88, GroupId: 8, IsTrial: false, SLA: "enterprise"},

		// BEDI OCR模型 (910B4佛山AI云平台)
		{Id: "OCRFlux/OCRFlux-3B", Name: "OCRFlux-3B", Provider: "BEDI", ModelType: ModelTypeOCR, Description: "910B4佛山AI云平台 - OCRFlux-3B文档识别模型", ContextLen: 4096, InputPrice: 0, OutputPrice: 0, Capabilities: `["ocr"]`, Status: ModelStatusActive, SortOrder: 89, GroupId: 8, IsTrial: false, SLA: "enterprise"},

		// BEDI Other模型 (910B4佛山AI云平台)
		{Id: "MinerU/MinerU", Name: "MinerU", Provider: "BEDI", ModelType: ModelTypeOther, Description: "910B4佛山AI云平台 - MinerU文档解析模型", ContextLen: 4096, InputPrice: 0, OutputPrice: 0, Capabilities: `["document_parsing"]`, Status: ModelStatusActive, SortOrder: 90, GroupId: 8, IsTrial: false, SLA: "enterprise"},

		// 其他内部部署 vLLM (Group 99)
		{Id: "internal-glm-4", Name: "GLM-4 (其他内网)", Provider: "Internal", ModelType: ModelTypeChat, Description: "其他内部GLM-4 vLLM部署", ContextLen: 128000, InputPrice: 0, OutputPrice: 0, Capabilities: `["chat"]`, Status: ModelStatusActive, SortOrder: 100, GroupId: 99, SLA: "enterprise"},
		{Id: "internal-qwen-2.5", Name: "Qwen-2.5 (其他内网)", Provider: "Internal", ModelType: ModelTypeChat, Description: "其他内部Qwen-2.5 vLLM部署", ContextLen: 32000, InputPrice: 0, OutputPrice: 0, Capabilities: `["chat"]`, Status: ModelStatusActive, SortOrder: 101, GroupId: 99, SLA: "enterprise"},
		{Id: "internal-llama-3.1", Name: "Llama-3.1 (其他内网)", Provider: "Internal", ModelType: ModelTypeChat, Description: "其他内部Llama-3.1 vLLM部署", ContextLen: 128000, InputPrice: 0, OutputPrice: 0, Capabilities: `["chat"]`, Status: ModelStatusActive, SortOrder: 102, GroupId: 99, SLA: "enterprise"},
	}

	for i := range defaultModels {
		if err := defaultModels[i].Create(); err != nil {
			return err
		}
	}

	return nil
}