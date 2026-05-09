package model

import (
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
		if source == "lark" {
			targetCompanyID = pools.FormalCompanyId
			targetDepartmentID = pools.FormalDepartmentId
		} else {
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
