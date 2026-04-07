package model

import (
	"time"

	"github.com/pagoda-inference/one-api/common/helper"
)

// Department represents a department within a company
type Department struct {
	Id          int       `json:"id" gorm:"primarykey"`
	CompanyId   int       `json:"company_id" gorm:"type:int;index"`
	Name        string    `json:"name" gorm:"type:varchar(64);not null"`
	Code        string    `json:"code" gorm:"type:varchar(32)"`
	Status      int       `json:"status" gorm:"type:int;default:1"` // 1=active, 2=disabled
	QuotaLimit  int64     `json:"quota_limit" gorm:"bigint;default:0"` // Department quota limit
	QuotaUsed   int64     `json:"quota_used" gorm:"bigint;default:0"`
	Description string    `json:"description" gorm:"type:text"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (Department) TableName() string {
	return "departments"
}

// Department status constants
const (
	DepartmentStatusActive   = 1
	DepartmentStatusDisabled = 2
)

// CreateDepartment creates a new department
func CreateDepartment(department *Department) error {
	return DB.Create(department).Error
}

// GetDepartmentById returns department by ID
func GetDepartmentById(id int) (*Department, error) {
	var department Department
	err := DB.First(&department, id).Error
	if err != nil {
		return nil, err
	}
	return &department, nil
}

// GetDepartmentsByCompany returns all departments of a company
func GetDepartmentsByCompany(companyId int) ([]*Department, error) {
	var departments []*Department
	err := DB.Where("company_id = ? AND status = ?", companyId, DepartmentStatusActive).Order("id ASC").Find(&departments).Error
	return departments, err
}

// UpdateDepartment updates department info
func UpdateDepartment(department *Department) error {
	return DB.Model(department).Updates(department).Error
}

// DeleteDepartment soft deletes a department
func DeleteDepartment(id int) error {
	return DB.Model(&Department{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":     DepartmentStatusDisabled,
		"updated_at": helper.GetTimestamp(),
	}).Error
}

// GetDepartmentTeams returns all teams of a department
func GetDepartmentTeams(departmentId int) ([]*Tenant, error) {
	var teams []*Tenant
	err := DB.Where("department_id = ? AND status = ?", departmentId, TenantStatusActive).Order("id ASC").Find(&teams).Error
	return teams, err
}

// CountDepartmentTeams counts teams in a department
func CountDepartmentTeams(departmentId int) (int64, error) {
	var count int64
	err := DB.Model(&Tenant{}).Where("department_id = ? AND status = ?", departmentId, TenantStatusActive).Count(&count).Error
	return count, err
}
