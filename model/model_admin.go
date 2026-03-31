package model

import (
	"gorm.io/gorm"
)

// Model 模型表（后台管理用）
type Model struct {
	Id           string  `json:"id" gorm:"primaryKey;size:64"`
	Name         string  `json:"name" gorm:"size:128;not null"`
	Provider     string  `json:"provider" gorm:"size:64"`
	ModelType    string  `json:"model_type" gorm:"size:32"`
	Description  string  `json:"description" gorm:"type:text"`
	ContextLen   int     `json:"context_len"`
	InputPrice   float64 `json:"input_price" gorm:"type:decimal(10,6);default:0"`
	OutputPrice  float64 `json:"output_price" gorm:"type:decimal(10,6);default:0"`
	Capabilities string  `json:"capabilities" gorm:"type:text"` // JSON array
	Status       string  `json:"status" gorm:"size:32;default:active"`
	IconUrl      string  `json:"icon_url" gorm:"size:255"`
	SortOrder    int     `json:"sort_order" gorm:"default:0"`
	CreatedTime  int64   `json:"created_time" gorm:"bigint"`
	UpdatedTime  int64   `json:"updated_time" gorm:"bigint"`
}

func (Model) TableName() string {
	return "models"
}

// AutoMigrateModel 自动迁移模型表
func AutoMigrateModel() error {
	return DB.AutoMigrate(&Model{})
}

// GetAllAdminModels 获取所有模型（管理员用）
func GetAllAdminModels() ([]Model, error) {
	var models []Model
	err := DB.Order("sort_order ASC, created_time DESC").Find(&models).Error
	return models, err
}

// GetAdminModelById 根据ID获取模型（管理员用）
func GetAdminModelById(id string) (*Model, error) {
	var model Model
	err := DB.First(&model, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &model, nil
}

// CreateModel 创建模型
func CreateModel(m *Model) error {
	return DB.Create(m).Error
}

// UpdateModel 更新模型
func UpdateModel(m *Model) error {
	return DB.Save(m).Error
}

// DeleteModel 删除模型
func DeleteModel(id string) error {
	return DB.Delete(&Model{}, "id = ?", id).Error
}

// GetAdminModelsByStatus 根据状态获取模型
func GetAdminModelsByStatus(status string) ([]Model, error) {
	var models []Model
	err := DB.Where("status = ?", status).Order("sort_order ASC").Find(&models).Error
	return models, err
}

// GetAdminActiveModels 获取所有上线模型
func GetAdminActiveModels() ([]Model, error) {
	return GetAdminModelsByStatus("active")
}

// UpdateModelStatus 更新模型状态
func UpdateModelStatus(id, status string) error {
	return DB.Model(&Model{}).Where("id = ?", id).Update("status", status).Error
}

// UpdateModelSortOrder 更新模型排序
func UpdateModelSortOrder(id string, order int) error {
	return DB.Model(&Model{}).Where("id = ?", id).Update("sort_order", order).Error
}

// BatchUpdateSortOrder 批量更新排序
func BatchUpdateSortOrder(updates []struct {
	ID        string `json:"id"`
	SortOrder int    `json:"sort_order"`
}) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		for _, u := range updates {
			if err := tx.Model(&Model{}).Where("id = ?", u.ID).Update("sort_order", u.SortOrder).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
