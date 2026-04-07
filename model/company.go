package model

import (
	"time"

	"github.com/pagoda-inference/one-api/common/helper"
)

// Company represents an organization/company in the platform
type Company struct {
	Id          int       `json:"id" gorm:"primarykey"`
	Name        string    `json:"name" gorm:"type:varchar(64);not null"`
	Code        string    `json:"code" gorm:"type:varchar(32);uniqueIndex"` // Unique identifier for API
	Status      int       `json:"status" gorm:"type:int;default:1"`          // 1=active, 2=disabled, 3=deleted
	QuotaLimit  int64     `json:"quota_limit" gorm:"bigint;default:0"`      // Total quota limit for company
	QuotaUsed   int64     `json:"quota_used" gorm:"bigint;default:0"`       // Quota used by company
	LogoUrl     string    `json:"logo_url" gorm:"type:varchar(255)"`        // Company logo URL
	Description string    `json:"description" gorm:"type:text"`             // Company description
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (Company) TableName() string {
	return "companies"
}

// Company status constants
const (
	CompanyStatusActive   = 1
	CompanyStatusDisabled = 2
	CompanyStatusDeleted  = 3
)

// CreateCompany creates a new company
func CreateCompany(company *Company) error {
	return DB.Create(company).Error
}

// GetCompanyById returns company by ID
func GetCompanyById(id int) (*Company, error) {
	var company Company
	err := DB.First(&company, id).Error
	if err != nil {
		return nil, err
	}
	return &company, nil
}

// GetCompanyByCode returns company by code
func GetCompanyByCode(code string) (*Company, error) {
	var company Company
	err := DB.Where("code = ?", code).First(&company).Error
	if err != nil {
		return nil, err
	}
	return &company, nil
}

// GetAllCompanies returns all companies
func GetAllCompanies() ([]*Company, error) {
	var companies []*Company
	err := DB.Order("id ASC").Find(&companies).Error
	return companies, err
}

// UpdateCompany updates company info
func UpdateCompany(company *Company) error {
	return DB.Model(company).Updates(company).Error
}

// DeleteCompany soft deletes a company
func DeleteCompany(id int) error {
	return DB.Model(&Company{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":     CompanyStatusDeleted,
		"updated_at": helper.GetTimestamp(),
	}).Error
}

// GetCompanyDepartments returns all departments of a company
func GetCompanyDepartments(companyId int) ([]*Department, error) {
	var departments []*Department
	err := DB.Where("company_id = ?", companyId).Order("id ASC").Find(&departments).Error
	return departments, err
}

// CountCompanyDepartments counts departments in a company
func CountCompanyDepartments(companyId int) (int64, error) {
	var count int64
	err := DB.Model(&Department{}).Where("company_id = ?", companyId).Count(&count).Error
	return count, err
}
