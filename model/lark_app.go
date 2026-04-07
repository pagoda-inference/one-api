package model

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
)

// LarkOAuthApp represents a Lark OAuth application configuration
type LarkOAuthApp struct {
	Id           int    `json:"id" gorm:"primarykey"`
	Name         string `json:"name" gorm:"type:varchar(64);not null"` // Display name like "北电数智"
	ClientId     string `json:"client_id" gorm:"type:varchar(128);not null"`
	ClientSecret string `json:"-" gorm:"type:varchar(256);not null"` // Never expose in JSON
	Enabled      bool   `json:"enabled" gorm:"default:true"`
	Sort         int    `json:"sort" gorm:"default:0"` // Sort order
	CreatedAt    int64  `json:"created_at" gorm:"bigint"`
	UpdatedAt    int64  `json:"updated_at" gorm:"bigint"`
}

// TableName specifies the table name for LarkOAuthApp
func (LarkOAuthApp) TableName() string {
	return "lark_oauth_apps"
}

func (l *LarkOAuthApp) BeforeCreate(tx *gorm.DB) error {
	l.CreatedAt = time.Now().Unix()
	l.UpdatedAt = time.Now().Unix()
	return nil
}

func (l *LarkOAuthApp) BeforeUpdate(tx *gorm.DB) error {
	l.UpdatedAt = time.Now().Unix()
	return nil
}

// GetLarkOAuthApps returns all enabled Lark OAuth apps ordered by sort
func GetLarkOAuthApps() ([]*LarkOAuthApp, error) {
	var apps []*LarkOAuthApp
	err := DB.Where("enabled = ?", true).Order("sort ASC, id ASC").Find(&apps).Error
	if err != nil {
		return nil, err
	}
	return apps, nil
}

// GetAllLarkOAuthApps returns all Lark OAuth apps (including disabled)
func GetAllLarkOAuthApps() ([]*LarkOAuthApp, error) {
	var apps []*LarkOAuthApp
	err := DB.Order("sort ASC, id ASC").Find(&apps).Error
	if err != nil {
		return nil, err
	}
	return apps, nil
}

// GetLarkOAuthAppById returns a Lark OAuth app by ID
func GetLarkOAuthAppById(id int) (*LarkOAuthApp, error) {
	var app LarkOAuthApp
	err := DB.First(&app, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("Lark OAuth app not found")
		}
		return nil, err
	}
	return &app, nil
}

// CreateLarkOAuthApp creates a new Lark OAuth app
func CreateLarkOAuthApp(app *LarkOAuthApp) error {
	return DB.Create(app).Error
}

// UpdateLarkOAuthApp updates an existing Lark OAuth app
func UpdateLarkOAuthApp(app *LarkOAuthApp) error {
	return DB.Save(app).Error
}

// DeleteLarkOAuthApp deletes a Lark OAuth app
func DeleteLarkOAuthApp(id int) error {
	return DB.Delete(&LarkOAuthApp{}, id).Error
}

// GetEnabledLarkAppsCount returns the count of enabled Lark OAuth apps
func GetEnabledLarkAppsCount() (int64, error) {
	var count int64
	err := DB.Model(&LarkOAuthApp{}).Where("enabled = ?", true).Count(&count).Error
	return count, err
}

// EnsureLarkOAuthAppsTable ensures the lark_oauth_apps table exists
func EnsureLarkOAuthAppsTable() error {
	return DB.AutoMigrate(&LarkOAuthApp{})
}

// LarkOAuthAppEdit represents the editable fields for Lark OAuth app
type LarkOAuthAppEdit struct {
	Name         string `json:"name" binding:"required,max=64"`
	ClientId     string `json:"client_id" binding:"required,max=128"`
	ClientSecret string `json:"client_secret" binding:"required,max=256"`
	Enabled      bool   `json:"enabled"`
	Sort         int    `json:"sort"`
}

// ToLarkOAuthApp converts LarkOAuthAppEdit to LarkOAuthApp
func (e *LarkOAuthAppEdit) ToLarkOAuthApp() *LarkOAuthApp {
	return &LarkOAuthApp{
		Name:         e.Name,
		ClientId:     e.ClientId,
		ClientSecret: e.ClientSecret,
		Enabled:      e.Enabled,
		Sort:         e.Sort,
	}
}

// LarkOAuthAppPublic represents the public info of Lark OAuth app (excludes secret)
type LarkOAuthAppPublic struct {
	Id        int    `json:"id"`
	Name      string `json:"name"`
	ClientId  string `json:"client_id"`
	Enabled   bool   `json:"enabled"`
	Sort      int    `json:"sort"`
	CreatedAt int64  `json:"created_at"`
}

// ToPublic converts LarkOAuthApp to LarkOAuthAppPublic
func (l *LarkOAuthApp) ToPublic() *LarkOAuthAppPublic {
	return &LarkOAuthAppPublic{
		Id:        l.Id,
		Name:      l.Name,
		ClientId:  l.ClientId,
		Enabled:   l.Enabled,
		Sort:      l.Sort,
		CreatedAt: l.CreatedAt,
	}
}

// GetPublicLarkOAuthApps returns all enabled Lark OAuth apps in public format
func GetPublicLarkOAuthApps(ctx context.Context) ([]*LarkOAuthAppPublic, error) {
	apps, err := GetLarkOAuthApps()
	if err != nil {
		return nil, err
	}
	publicApps := make([]*LarkOAuthAppPublic, len(apps))
	for i, app := range apps {
		publicApps[i] = app.ToPublic()
	}
	return publicApps, nil
}