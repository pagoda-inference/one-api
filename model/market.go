package model

import (
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"github.com/pagoda-inference/one-api/common/helper"
)

// ModelInfo represents a model available in the marketplace
type ModelInfo struct {
	Id                   string  `json:"id" gorm:"primaryKey;size:64"`
	Name                 string  `json:"name" gorm:"size:128"`      // 显示名称
	Provider             string  `json:"provider" gorm:"size:64"`   // 提供商
	ModelType            string  `json:"model_type" gorm:"size:32"` // chat/embedding/image/audio
	Description          string  `json:"description" gorm:"type:text"`
	ContextLen           int     `json:"context_len" gorm:"default:4096"`                  // 上下文长度
	InputPrice           float64 `json:"input_price" gorm:"type:decimal(10,4);default:0"`  // 输入价格(元/千token)
	OutputPrice          float64 `json:"output_price" gorm:"type:decimal(10,4);default:0"` // 输出价格(元/千token)
	Capabilities         string  `json:"capabilities" gorm:"type:text"`                    // JSON array of capabilities
	Status               string  `json:"status" gorm:"size:32;default:active"`             // active/maintenance/deprecated
	SortOrder            int     `json:"sort_order" gorm:"default:0"`
	IconUrl              string  `json:"icon_url" gorm:"size:255"`
	GroupId              int     `json:"group_id" gorm:"default:0"`                 // 模型分组ID
	Tags                 string  `json:"tags" gorm:"type:text"`                     // JSON array of tags
	IsTrial              bool    `json:"is_trial" gorm:"default:false"`             // 是否支持试用
	TrialQuota           int64   `json:"trial_quota" gorm:"default:0"`              // 试用额度
	SLA                  string  `json:"sla" gorm:"size:32;default:standard"`       // SLA等级: standard/premium/enterprise
	RateLimitRPM         int     `json:"rate_limit_rpm" gorm:"default:0"`           // 模型级别 RPM 限流 (0=不限)
	RateLimitTPM         int     `json:"rate_limit_tpm" gorm:"default:0"`           // 模型级别 TPM 限流 (0=不限)
	VisibleScope         string  `json:"visible_scope" gorm:"size:32;default:team"` // 可见范围: public/department/team
	VisibleToTeams       string  `json:"visible_to_teams" gorm:"type:text"`         // 可见团队，格式",1,2,3,"
	VisibleToDepartments string  `json:"visible_to_departments" gorm:"type:text"`   // 可见部门，格式",1,2,3,"
	// 体验中心配置
	PlaygroundMaxTokens               int     `json:"playground_max_tokens" gorm:"default:8192"`                 // 体验中心最大token
	PlaygroundTemperature             float64 `json:"playground_temperature" gorm:"default:0.6"`                 // 体验中心温度
	PlaygroundMinP                    float64 `json:"playground_min_p" gorm:"default:0"`                         // 体验中心min_p
	PlaygroundTopP                    float64 `json:"playground_top_p" gorm:"default:0.95"`                      // 体验中心topP
	PlaygroundTopK                    int     `json:"playground_top_k" gorm:"default:20"`                        // 体验中心topK
	PlaygroundFrequencyPenalty        float64 `json:"playground_frequency_penalty" gorm:"default:0"`             // 体验中心frequency penalty
	PlaygroundPresencePenalty         float64 `json:"playground_presence_penalty" gorm:"default:0"`              // 体验中心presence penalty
	PlaygroundRepetitionPenalty       float64 `json:"playground_repetition_penalty" gorm:"default:1"`            // 体验中心repetition penalty
	PlaygroundSystemPrompt            string  `json:"playground_system_prompt" gorm:"type:text"`                 // 体验中心系统提示词
	PlaygroundEnableThinking          bool    `json:"playground_enable_thinking" gorm:"default:false"`           // 体验中心启用思考
	PlaygroundThinkingBudget          int     `json:"playground_thinking_budget" gorm:"default:4096"`            // 体验中心思考预算
	PlaygroundEnableTemperature       bool    `json:"playground_enable_temperature" gorm:"default:true"`         // 体验中心显示temperature
	PlaygroundEnableMinP              bool    `json:"playground_enable_min_p" gorm:"default:false"`              // 体验中心显示min_p
	PlaygroundEnableTopP              bool    `json:"playground_enable_top_p" gorm:"default:true"`               // 体验中心显示top_p
	PlaygroundEnableTopK              bool    `json:"playground_enable_top_k" gorm:"default:true"`               // 体验中心显示top_k
	PlaygroundEnableFrequencyPenalty  bool    `json:"playground_enable_frequency_penalty" gorm:"default:true"`   // 体验中心显示frequency penalty
	PlaygroundEnablePresencePenalty   bool    `json:"playground_enable_presence_penalty" gorm:"default:true"`    // 体验中心显示presence penalty
	PlaygroundEnableRepetitionPenalty bool    `json:"playground_enable_repetition_penalty" gorm:"default:false"` // 体验中心显示repetition penalty
	PlaygroundEnableSystemPrompt      bool    `json:"playground_enable_system_prompt" gorm:"default:true"`       // 体验中心显示system prompt
	PlaygroundEnableVL                bool    `json:"playground_enable_vl" gorm:"default:false"`                 // 体验中心启用视觉输入
	PlaygroundEnableReasoning         bool    `json:"playground_enable_reasoning" gorm:"default:false"`          // 体验中心启用推理能力
	PlaygroundEnableThinkingBudget    bool    `json:"playground_enable_thinking_budget" gorm:"default:true"`     // 体验中心显示thinking budget
	CreatedAt                         int64   `json:"created_at" gorm:"bigint"`
	UpdatedAt                         int64   `json:"updated_at" gorm:"bigint"`
}

// ModelGroup is deprecated - use channels.group for provider concept
// type ModelGroup struct {
// 	Id          int    `json:"id" gorm:"primaryKey"`
// 	Name        string `json:"name" gorm:"size:64"`
// 	Code        string `json:"code" gorm:"size:32;uniqueIndex"`
// 	Description string `json:"description" gorm:"type:text"`
// 	IconUrl     string `json:"icon_url" gorm:"size:255"`
// 	SortOrder   int    `json:"sort_order" gorm:"default:0"`
// 	Status      string `json:"status" gorm:"size:32;default:active"`
// 	CreatedAt   int64  `json:"created_at" gorm:"bigint"`
// 	UpdatedAt   int64  `json:"updated_at" gorm:"bigint"`
// }

// func (ModelGroup) TableName() string {
// 	return "model_groups"
// }

// ModelPricing represents configurable model pricing
type ModelPricing struct {
	Id          int64   `json:"id" gorm:"primaryKey"`
	ModelId     string  `json:"model_id" gorm:"size:64;index"`
	TenantId    int     `json:"tenant_id" gorm:"index"` // 0 means default pricing
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
	Id        int    `json:"id" gorm:"primaryKey"`
	UserId    int    `json:"user_id" gorm:"index"`
	ModelId   string `json:"model_id" gorm:"size:64;index"`
	TenantId  int    `json:"tenant_id" gorm:"index"`
	QuotaUsed int64  `json:"quota_used" gorm:"bigint;default:0"`
	Status    string `json:"status" gorm:"size:32;default:active"` // active/expired/disabled
	CreatedAt int64  `json:"created_at" gorm:"bigint"`
	UpdatedAt int64  `json:"updated_at" gorm:"bigint"`
	ExpiresAt int64  `json:"expires_at" gorm:"bigint"`
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
	if m.VisibleScope == "" {
		m.VisibleScope = "team"
	}
	m.ModelType = strings.ToLower(strings.TrimSpace(m.ModelType))
	return DB.Create(m).Error
}

// Update updates a model info record
func (m *ModelInfo) Update() error {
	m.UpdatedAt = helper.GetTimestamp()
	m.ModelType = strings.ToLower(strings.TrimSpace(m.ModelType))
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

// GetMarketModelById is an alias for GetModelById
func GetMarketModelById(id string) (*ModelInfo, error) {
	return GetModelById(id)
}

// GetAllMarketModels retrieves all models (including inactive) for admin management
func GetAllMarketModels() ([]*ModelInfo, error) {
	var models []*ModelInfo
	err := DB.Order("sort_order ASC, id ASC").Find(&models).Error
	return models, err
}

// DeleteMarketModel deletes a model by ID and reorders subsequent models
func DeleteMarketModel(id string) error {
	// Get the sort_order of the model being deleted
	var model ModelInfo
	if err := DB.First(&model, "id = ?", id).Error; err != nil {
		return err
	}
	deletedSortOrder := model.SortOrder

	// Delete the model
	if err := DB.Delete(&ModelInfo{}, "id = ?", id).Error; err != nil {
		return err
	}

	// Reorder: decrease sort_order by 1 for all models with sort_order > deleted model's sort_order
	if deletedSortOrder > 0 {
		DB.Model(&ModelInfo{}).
			Where("sort_order > ?", deletedSortOrder).
			Update("sort_order", gorm.Expr("sort_order - 1"))
	}

	return nil
}

// GetAllModels retrieves all models without status filter (for marketplace display)
func GetAllModels(modelType string, limit int, offset int) ([]*ModelInfo, error) {
	var models []*ModelInfo
	query := DB
	if modelType != "" {
		query = query.Where("model_type = ?", strings.ToLower(strings.TrimSpace(modelType)))
	}
	if limit <= 0 {
		limit = 100
	}
	err := query.Order("sort_order ASC, id ASC").Limit(limit).Offset(offset).Find(&models).Error
	return models, err
}

// GetActiveModels retrieves all active models
func GetActiveModels(modelType string, limit int, offset int) ([]*ModelInfo, error) {
	var models []*ModelInfo
	query := DB.Where("status = ?", ModelStatusActive)
	if modelType != "" {
		query = query.Where("model_type = ?", strings.ToLower(strings.TrimSpace(modelType)))
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
		query = query.Where("model_type = ?", strings.ToLower(strings.TrimSpace(modelType)))
	}
	if limit <= 0 {
		limit = 100
	}
	err := query.Order("sort_order ASC").Limit(limit).Offset(offset).Find(&models).Error
	return models, err
}

// GetVisibleModelsForTenants retrieves active models visible to given tenant IDs with accurate pagination.
func GetVisibleModelsForTenants(tenantIds []int, departmentIds []int, keyword string, modelType string, limit int, offset int) ([]*ModelInfo, int64, error) {
	var (
		models []*ModelInfo
		total  int64
	)
	query := DB.Model(&ModelInfo{}).Where("status = ?", ModelStatusActive)
	if keyword != "" {
		query = query.Where("name LIKE ? OR provider LIKE ? OR id LIKE ?",
			"%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	}
	if modelType != "" {
		query = query.Where("model_type = ?", strings.ToLower(strings.TrimSpace(modelType)))
	}

	// Visibility filter by scope:
	// public: all users
	// department: department match
	// team(default): team match
	visibleQuery := query.Where("(visible_scope = 'public')")
	for _, did := range departmentIds {
		visibleQuery = visibleQuery.Or("(visible_scope = 'department' AND visible_to_departments LIKE ?)", fmt.Sprintf("%%,%d,%%", did))
	}
	visibleQuery = visibleQuery.Or("((visible_scope = '' OR visible_scope IS NULL OR visible_scope = 'team') AND (visible_to_teams = '' OR visible_to_teams IS NULL))")
	for _, tid := range tenantIds {
		visibleQuery = visibleQuery.Or("((visible_scope = '' OR visible_scope IS NULL OR visible_scope = 'team') AND visible_to_teams LIKE ?)", fmt.Sprintf("%%,%d,%%", tid))
	}

	if err := visibleQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if limit <= 0 {
		limit = 100
	}
	if err := visibleQuery.Order("sort_order ASC, id ASC").Limit(limit).Offset(offset).Find(&models).Error; err != nil {
		return nil, 0, err
	}
	return models, total, nil
}

// GetModelsByProvider retrieves models by provider (case-insensitive)
func GetModelsByProvider(provider string) ([]*ModelInfo, error) {
	var models []*ModelInfo
	err := DB.Where("status = ? AND LOWER(provider) = ?", ModelStatusActive, strings.ToLower(provider)).
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
		query = query.Where("model_type = ?", strings.ToLower(strings.TrimSpace(modelType)))
	}
	err := query.Count(&count).Error
	return count, err
}

// CountAllModels counts all models without status filter
func CountAllModels(modelType string) (int64, error) {
	var count int64
	query := DB.Model(&ModelInfo{})
	if modelType != "" {
		query = query.Where("model_type = ?", strings.ToLower(strings.TrimSpace(modelType)))
	}
	err := query.Count(&count).Error
	return count, err
}

// GetTrialModels retrieves all trial models
func GetTrialModels() ([]*ModelInfo, error) {
	var models []*ModelInfo
	err := DB.Where("is_trial = ? AND status = ?", true, ModelStatusActive).
		Order("sort_order ASC, id ASC").
		Find(&models).Error
	return models, err
}

// UpdatePlaygroundConfig updates playground configuration for a model
func UpdatePlaygroundConfig(modelId string, maxTokens int, temperature float64, minP float64, topP float64, topK int, frequencyPenalty float64, presencePenalty float64, repetitionPenalty float64, systemPrompt string, enableThinking bool, thinkingBudget int, enableTemperature bool, enableMinP bool, enableTopP bool, enableTopK bool, enableFrequencyPenalty bool, enablePresencePenalty bool, enableRepetitionPenalty bool, enableSystemPrompt bool, enableVL bool, enableReasoning bool, enableThinkingBudget bool) error {
	updates := map[string]interface{}{
		"playground_max_tokens":                maxTokens,
		"playground_temperature":               temperature,
		"playground_min_p":                     minP,
		"playground_top_p":                     topP,
		"playground_top_k":                     topK,
		"playground_frequency_penalty":         frequencyPenalty,
		"playground_presence_penalty":          presencePenalty,
		"playground_repetition_penalty":        repetitionPenalty,
		"playground_system_prompt":             systemPrompt,
		"playground_enable_thinking":           enableThinking,
		"playground_thinking_budget":           thinkingBudget,
		"playground_enable_temperature":        enableTemperature,
		"playground_enable_min_p":              enableMinP,
		"playground_enable_top_p":              enableTopP,
		"playground_enable_top_k":              enableTopK,
		"playground_enable_frequency_penalty":  enableFrequencyPenalty,
		"playground_enable_presence_penalty":   enablePresencePenalty,
		"playground_enable_repetition_penalty": enableRepetitionPenalty,
		"playground_enable_system_prompt":      enableSystemPrompt,
		"playground_enable_vl":                 enableVL,
		"playground_enable_reasoning":          enableReasoning,
		"playground_enable_thinking_budget":    enableThinkingBudget,
		"updated_at":                           helper.GetTimestamp(),
	}
	filtered := make(map[string]interface{}, len(updates))
	colTypes, err := DB.Migrator().ColumnTypes(&ModelInfo{})
	if err != nil {
		return err
	}
	existingCols := make(map[string]struct{}, len(colTypes))
	for _, c := range colTypes {
		existingCols[strings.ToLower(c.Name())] = struct{}{}
	}
	for k, v := range updates {
		if _, ok := existingCols[strings.ToLower(k)]; ok {
			filtered[k] = v
		}
	}
	if len(filtered) == 0 {
		return fmt.Errorf("no updatable playground columns found in model_info")
	}
	return DB.Model(&ModelInfo{}).Where("id = ?", modelId).Updates(filtered).Error
}

// CalculateQuota calculates quota cost for a model
// InputPrice and OutputPrice are in yuan per 1000 tokens
func (m *ModelInfo) CalculateQuota(promptTokens, completionTokens int) int64 {
	inputQuota := int64(float64(promptTokens) * m.InputPrice / 1000)
	outputQuota := int64(float64(completionTokens) * m.OutputPrice / 1000)
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
	TotalModels     int64   `json:"total_models"`
	TotalProviders  int64   `json:"total_providers"`
	TotalGroups     int64   `json:"total_groups"`
	ChatModels      int64   `json:"chat_models"`
	EmbeddingModels int64   `json:"embedding_models"`
	ImageModels     int64   `json:"image_models"`
	AvgInputPrice   float64 `json:"avg_input_price"`
	AvgOutputPrice  float64 `json:"avg_output_price"`
	TrialModels     int64   `json:"trial_models"`
}

// GetModelMarketStats returns marketplace statistics
func GetModelMarketStats() (*ModelMarketStats, error) {
	stats := &ModelMarketStats{}

	// Count total active models
	DB.Model(&ModelInfo{}).Where("status = ?", ModelStatusActive).Count(&stats.TotalModels)

	// Count providers from providers table (channels)
	var providerCount int64
	DB.Model(&Provider{}).Where("status = ?", "active").Count(&providerCount)
	fmt.Printf("[DEBUG] Provider count: %d\n", providerCount)
	stats.TotalProviders = providerCount

	// Count groups - use providers count
	stats.TotalGroups = stats.TotalProviders

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

	// Average prices - use COALESCE to handle NULL when no models have prices
	DB.Model(&ModelInfo{}).
		Where("status = ? AND input_price > 0", ModelStatusActive).
		Select("COALESCE(AVG(input_price), 0)").Scan(&stats.AvgInputPrice)

	DB.Model(&ModelInfo{}).
		Where("status = ? AND output_price > 0", ModelStatusActive).
		Select("COALESCE(AVG(output_price), 0)").Scan(&stats.AvgOutputPrice)

	return stats, nil
}

// Model group operations - deprecated, use channels.group for provider concept

// func CreateModelGroup(group *ModelGroup) error { ... }
// func GetModelGroupById(id int) (*ModelGroup, error) { ... }
// func GetAllModelGroups() ([]*ModelGroup, error) { ... }
// func GetAllMarketGroups() ([]*ModelGroup, error) { ... }
// func UpdateModelGroup(group *ModelGroup) error { ... }
// func DeleteModelGroup(id int) error { ... }

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

// SyncAllModelStatusByChannelAvailability reconciles model_info.status using channel/ability availability.
// Rule:
// - models with at least one enabled channel -> active
// - models with zero enabled channels      -> maintenance
// Only active/maintenance statuses are auto-managed. Deprecated models are untouched.
func SyncAllModelStatusByChannelAvailability() error {
	return syncModelStatusByChannelAvailability(nil)
}

// SyncModelStatusByChannelId reconciles statuses for models affected by a specific channel.
func SyncModelStatusByChannelId(channelId int) error {
	var modelIDs []string
	if err := DB.Model(&Ability{}).
		Where("channel_id = ?", channelId).
		Distinct("model").
		Pluck("model", &modelIDs).Error; err != nil {
		return err
	}
	if len(modelIDs) == 0 {
		return nil
	}
	return syncModelStatusByChannelAvailability(modelIDs)
}

func syncModelStatusByChannelAvailability(modelIDs []string) error {
	availableModels := DB.Table("abilities AS a").
		Select("a.model").
		Joins("JOIN channels c ON c.id = a.channel_id").
		Where("a.enabled = ? AND c.status = ?", true, ChannelStatusEnabled).
		Group("a.model")

	scope := DB.Model(&ModelInfo{})
	if len(modelIDs) > 0 {
		scope = scope.Where("id IN ?", modelIDs)
		availableModels = availableModels.Where("a.model IN ?", modelIDs)
	}

	now := helper.GetTimestamp()
	if err := scope.
		Where("status = ? AND id IN (?)", ModelStatusMaintenance, availableModels).
		Updates(map[string]any{
			"status":     ModelStatusActive,
			"updated_at": now,
		}).Error; err != nil {
		return err
	}

	if err := scope.
		Where("status = ? AND id NOT IN (?)", ModelStatusActive, availableModels).
		Updates(map[string]any{
			"status":     ModelStatusMaintenance,
			"updated_at": now,
		}).Error; err != nil {
		return err
	}

	return nil
}

// InitializeDefaultModels initializes default models if none exist
func InitializeDefaultModels() error {
	return nil
}
