package model

import (
	"errors"
	"time"

	"github.com/pagoda-inference/one-api/common/helper"
)

// Provider represents an AI model provider (e.g., BEDI, OpenAI, Anthropic)
type Provider struct {
	Id          int    `json:"id" gorm:"primaryKey"`
	Code        string `json:"code" gorm:"uniqueIndex;size:64"` // e.g., "bedi", "openai"
	Name        string `json:"name" gorm:"size:128"`
	LogoUrl     string `json:"logo_url" gorm:"size:255"`
	Description string `json:"description" gorm:"type:text"`
	Website     string `json:"website" gorm:"size:255"`
	Status      string `json:"status" gorm:"size:32;default:active"` // active/maintenance/disabled
	SortOrder   int    `json:"sort_order" gorm:"default:0"`
	CreatedAt   int64  `json:"created_at" gorm:"bigint"`
	UpdatedAt   int64  `json:"updated_at" gorm:"bigint"`
}

// TableName returns the table name for Provider
func (Provider) TableName() string {
	return "providers"
}

// Create creates a new Provider
func (p *Provider) Create() error {
	if p.Code == "" {
		return errors.New("provider code is required")
	}
	if p.Name == "" {
		p.Name = p.Code
	}
	if p.CreatedAt == 0 {
		p.CreatedAt = time.Now().Unix()
	}
	if p.UpdatedAt == 0 {
		p.UpdatedAt = time.Now().Unix()
	}
	if p.Status == "" {
		p.Status = "active"
	}
	return DB.Create(p).Error
}

// Update updates a Provider
func (p *Provider) Update() error {
	p.UpdatedAt = time.Now().Unix()
	return DB.Save(p).Error
}

// GetProviderById retrieves a Provider by ID
func GetProviderById(id int) (*Provider, error) {
	var provider Provider
	err := DB.First(&provider, id).Error
	if err != nil {
		return nil, err
	}
	return &provider, nil
}

// GetProviderByCode retrieves a Provider by code
func GetProviderByCode(code string) (*Provider, error) {
	var provider Provider
	err := DB.Where("code = ?", code).First(&provider).Error
	if err != nil {
		return nil, err
	}
	return &provider, nil
}

// GetAllProviders retrieves all providers
func GetAllProviders() ([]*Provider, error) {
	var providers []*Provider
	err := DB.Order("sort_order ASC, id ASC").Find(&providers).Error
	return providers, err
}

// GetActiveProviders retrieves all active providers
func GetActiveProviders() ([]*Provider, error) {
	var providers []*Provider
	err := DB.Where("status = ?", "active").Order("sort_order ASC, id ASC").Find(&providers).Error
	return providers, err
}

// DeleteProvider deletes a Provider by ID
func DeleteProvider(id int) error {
	return DB.Delete(&Provider{}, "id = ?", id).Error
}

// Provider status constants
const (
	ProviderStatusActive      = "active"
	ProviderStatusMaintenance = "maintenance"
	ProviderStatusDisabled    = "disabled"
)

// InitializeDefaultProviders initializes default providers if none exist
func InitializeDefaultProviders() error {
	var count int64
	DB.Model(&Provider{}).Count(&count)
	if count > 0 {
		return nil
	}

	defaultProviders := []Provider{
		{Code: "bedi", Name: "BEDI", Description: "BEDI 私有GPU集群", Website: "https://bedicloud.net", Status: ProviderStatusActive, SortOrder: 1},
		{Code: "openai", Name: "OpenAI", Description: "OpenAI 官方API", Website: "https://openai.com", Status: ProviderStatusActive, SortOrder: 10},
		{Code: "anthropic", Name: "Anthropic", Description: "Anthropic Claude API", Website: "https://anthropic.com", Status: ProviderStatusActive, SortOrder: 20},
		{Code: "google", Name: "Google", Description: "Google Gemini API", Website: "https://ai.google.dev", Status: ProviderStatusActive, SortOrder: 30},
		{Code: "baidu", Name: "百度", Description: "百度文心一言", Website: "https://cloud.baidu.com", Status: ProviderStatusActive, SortOrder: 40},
		{Code: "alibaba", Name: "阿里", Description: "阿里通义千问", Website: "https://dashscope.console.aliyun.com", Status: ProviderStatusActive, SortOrder: 50},
		{Code: "zhipu", Name: "智谱AI", Description: "智谱GLM大模型", Website: "https://open.bigmodel.cn", Status: ProviderStatusActive, SortOrder: 60},
		{Code: "minimax", Name: "MiniMax", Description: "MiniMax 海螺AI", Website: "https://www.minimax.io", Status: ProviderStatusActive, SortOrder: 70},
	}

	createdAt := helper.GetTimestamp()
	for i := range defaultProviders {
		defaultProviders[i].CreatedAt = createdAt
		defaultProviders[i].UpdatedAt = createdAt
		if err := DB.Create(&defaultProviders[i]).Error; err != nil {
			return err
		}
	}

	return nil
}
