package model

import (
	"fmt"
	"strings"

	"github.com/pagoda-inference/one-api/common/helper"
	"gorm.io/gorm"
)

// ResolveAndUpsertUserOrg ensures user has company/department and active org membership.
// This is intentionally conservative: if user already has explicit company/department,
// it will preserve existing values and only backfill membership.
func ResolveAndUpsertUserOrg(user *User, source string) error {
	if user == nil || user.Id == 0 {
		return nil
	}

	targetCompanyID := user.CompanyId
	targetDepartmentID := user.DepartmentId

	if targetCompanyID == 0 || targetDepartmentID == 0 {
		pools, err := EnsureDefaultOrgPools()
		if err != nil {
			return err
		}
		switch source {
		case "lark_external":
			targetCompanyID = pools.ExternalCompanyId
			targetDepartmentID = pools.ExternalDepartmentId
		case "lark", "lark_formal":
			targetCompanyID = pools.FormalCompanyId
			targetDepartmentID = pools.FormalDepartmentId
		default:
			targetCompanyID = pools.ExternalCompanyId
			targetDepartmentID = pools.ExternalDepartmentId
		}
	}
	user.CompanyId = targetCompanyID
	user.DepartmentId = targetDepartmentID
	user.OrgSource = source

	now := helper.GetTimestamp()

	return DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&User{}).Where("id = ?", user.Id).Updates(map[string]interface{}{
			"company_id":    targetCompanyID,
			"department_id": targetDepartmentID,
			"org_source":    source,
		}).Error; err != nil {
			return err
		}

		mem := &UserOrgMembership{}
		if err := tx.Where("user_id = ? AND company_id = ?", user.Id, targetCompanyID).First(mem).Error; err != nil {
			if err != gorm.ErrRecordNotFound {
				return err
			}
			mem = &UserOrgMembership{
				UserId:       user.Id,
				CompanyId:    targetCompanyID,
				DepartmentId: targetDepartmentID,
				Role:         OrgRoleMember,
				Source:       source,
				Status:       OrgMembershipStatusActive,
				CreatedAt:    now,
				UpdatedAt:    now,
			}
			if cerr := tx.Create(mem).Error; cerr != nil {
				return cerr
			}
		} else {
			if err := tx.Model(mem).Updates(map[string]interface{}{
				"department_id": targetDepartmentID,
				"source":        source,
				"status":        OrgMembershipStatusActive,
				"updated_at":    now,
			}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func ensureDepartmentUnderCompany(tx *gorm.DB, companyID int, departmentName string) (int, error) {
	departmentName = strings.TrimSpace(departmentName)
	if companyID <= 0 || departmentName == "" {
		return 0, fmt.Errorf("invalid company or department name")
	}
	var dept Department
	if err := tx.Where("company_id = ? AND name = ? AND status = ?", companyID, departmentName, DepartmentStatusActive).First(&dept).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			return 0, err
		}
		dept = Department{
			CompanyId: companyID,
			Name:      departmentName,
			Code:      fmt.Sprintf("lark_dept_%d", helper.GetTimestamp()),
			Status:    DepartmentStatusActive,
		}
		if err = tx.Create(&dept).Error; err != nil {
			return 0, err
		}
	}
	return dept.Id, nil
}

// ResolveAndUpsertUserDepartmentByName updates user's department under current company and keeps membership in sync.
func ResolveAndUpsertUserDepartmentByName(user *User, departmentName string, source string) error {
	if user == nil || user.Id == 0 || user.CompanyId <= 0 {
		return nil
	}
	departmentName = strings.TrimSpace(departmentName)
	if departmentName == "" {
		return nil
	}

	now := helper.GetTimestamp()
	return DB.Transaction(func(tx *gorm.DB) error {
		departmentID, err := ensureDepartmentUnderCompany(tx, user.CompanyId, departmentName)
		if err != nil {
			return err
		}
		if err = tx.Model(&User{}).Where("id = ?", user.Id).Updates(map[string]interface{}{
			"department_id": departmentID,
			"org_source":    source,
		}).Error; err != nil {
			return err
		}

		mem := &UserOrgMembership{}
		if err = tx.Where("user_id = ? AND company_id = ?", user.Id, user.CompanyId).First(mem).Error; err != nil {
			if err != gorm.ErrRecordNotFound {
				return err
			}
			mem = &UserOrgMembership{
				UserId:       user.Id,
				CompanyId:    user.CompanyId,
				DepartmentId: departmentID,
				Role:         OrgRoleMember,
				Source:       source,
				Status:       OrgMembershipStatusActive,
				CreatedAt:    now,
				UpdatedAt:    now,
			}
			if err = tx.Create(mem).Error; err != nil {
				return err
			}
		} else {
			if err = tx.Model(mem).Updates(map[string]interface{}{
				"department_id": departmentID,
				"source":        source,
				"status":        OrgMembershipStatusActive,
				"updated_at":    now,
			}).Error; err != nil {
				return err
			}
		}
		user.DepartmentId = departmentID
		return nil
	})
}
