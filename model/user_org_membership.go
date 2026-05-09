package model

import (
	"fmt"
	"strings"

	"github.com/pagoda-inference/one-api/common/helper"
	"gorm.io/gorm"
)

const (
	OrgRolePlatformAdmin   = "platform_admin"
	OrgRoleDepartmentAdmin = "department_admin"
	OrgRoleTeamAdmin       = "team_admin"
	OrgRoleMember          = "member"
)

const (
	OrgMembershipStatusActive   = 1
	OrgMembershipStatusDisabled = 2
)

const (
	DefaultCompanyFormalName      = "正式员工池"
	DefaultCompanyExternalName    = "外部注册池"
	DefaultDepartmentFormalName   = "默认部门"
	DefaultDepartmentExternalName = "默认部门"
)

type UserOrgMembership struct {
	Id           int    `json:"id" gorm:"primarykey"`
	UserId       int    `json:"user_id" gorm:"type:int;index:idx_user_org_unique,unique"`
	CompanyId    int    `json:"company_id" gorm:"type:int;index:idx_user_org_unique,unique"`
	DepartmentId int    `json:"department_id" gorm:"type:int;index"`
	TenantId     int    `json:"tenant_id" gorm:"type:int;default:0;index"`
	Role         string `json:"role" gorm:"type:varchar(32);default:'member'"`
	Source       string `json:"source" gorm:"type:varchar(32);default:'migration'"`
	Status       int    `json:"status" gorm:"type:int;default:1"`
	CreatedAt    int64  `json:"created_at" gorm:"bigint"`
	UpdatedAt    int64  `json:"updated_at" gorm:"bigint"`
}

func (UserOrgMembership) TableName() string {
	return "user_org_memberships"
}

type OrgMigrationReport struct {
	TotalUsers int      `json:"total_users"`
	Updated    int      `json:"updated"`
	Skipped    int      `json:"skipped"`
	Failed     int      `json:"failed"`
	Examples   []string `json:"examples"`
}

type OrgPoolRefs struct {
	FormalCompanyId      int
	FormalDepartmentId   int
	ExternalCompanyId    int
	ExternalDepartmentId int
}

func ensureDefaultCompanyAndDepartment(companyName string, departmentName string, codePrefix string) (int, int, error) {
	var company Company
	err := DB.Where("name = ? AND status = ?", companyName, CompanyStatusActive).First(&company).Error
	if err != nil {
		company = Company{
			Name:   companyName,
			Code:   fmt.Sprintf("%s_%d", codePrefix, helper.GetTimestamp()),
			Status: CompanyStatusActive,
		}
		if err = DB.Create(&company).Error; err != nil {
			return 0, 0, err
		}
	}

	var department Department
	err = DB.Where("company_id = ? AND name = ? AND status = ?", company.Id, departmentName, DepartmentStatusActive).First(&department).Error
	if err != nil {
		department = Department{
			CompanyId: company.Id,
			Name:      departmentName,
			Code:      fmt.Sprintf("%s_dept_%d", codePrefix, helper.GetTimestamp()),
			Status:    DepartmentStatusActive,
		}
		if err = DB.Create(&department).Error; err != nil {
			return 0, 0, err
		}
	}
	return company.Id, department.Id, nil
}

func EnsureDefaultOrgPools() (*OrgPoolRefs, error) {
	formalCompanyID, formalDepartmentID, err := ensureDefaultCompanyAndDepartment(
		DefaultCompanyFormalName, DefaultDepartmentFormalName, "formal",
	)
	if err != nil {
		return nil, err
	}
	externalCompanyID, externalDepartmentID, err := ensureDefaultCompanyAndDepartment(
		DefaultCompanyExternalName, DefaultDepartmentExternalName, "external",
	)
	if err != nil {
		return nil, err
	}
	return &OrgPoolRefs{
		FormalCompanyId:      formalCompanyID,
		FormalDepartmentId:   formalDepartmentID,
		ExternalCompanyId:    externalCompanyID,
		ExternalDepartmentId: externalDepartmentID,
	}, nil
}

func classifyUserOrgSource(user *User) (isFormal bool, source string) {
	if strings.TrimSpace(user.LarkId) != "" {
		return true, "lark"
	}
	return false, "password"
}

func MigrateUsersToDefaultOrgPools(apply bool, limit int) (*OrgMigrationReport, error) {
	if limit <= 0 {
		limit = 100000
	}
	report := &OrgMigrationReport{Examples: make([]string, 0, 20)}

	pools, err := EnsureDefaultOrgPools()
	if err != nil {
		return nil, err
	}

	var users []*User
	if err = DB.Where("status != ?", UserStatusDeleted).Order("id ASC").Limit(limit).Find(&users).Error; err != nil {
		return nil, err
	}
	report.TotalUsers = len(users)

	now := helper.GetTimestamp()

	for _, u := range users {
		isFormal, source := classifyUserOrgSource(u)
		targetCompanyID := pools.ExternalCompanyId
		targetDepartmentID := pools.ExternalDepartmentId
		if isFormal {
			targetCompanyID = pools.FormalCompanyId
			targetDepartmentID = pools.FormalDepartmentId
		}
		// Preserve explicit user assignment if it already exists.
		if u.CompanyId > 0 && u.DepartmentId > 0 {
			targetCompanyID = u.CompanyId
			targetDepartmentID = u.DepartmentId
			if strings.TrimSpace(u.OrgSource) != "" {
				source = u.OrgSource
			}
		}

		if u.CompanyId == targetCompanyID && u.DepartmentId == targetDepartmentID {
			report.Skipped++
			continue
		}

		if !apply {
			report.Updated++
			if len(report.Examples) < 20 {
				report.Examples = append(report.Examples, fmt.Sprintf("user=%d -> company=%d department=%d source=%s", u.Id, targetCompanyID, targetDepartmentID, source))
			}
			continue
		}

		err = DB.Transaction(func(tx *gorm.DB) error {
			if e := tx.Model(&User{}).Where("id = ?", u.Id).Updates(map[string]interface{}{
				"company_id":    targetCompanyID,
				"department_id": targetDepartmentID,
				"org_source":    source,
			}).Error; e != nil {
				return e
			}

			mem := &UserOrgMembership{}
			if e := tx.Where("user_id = ? AND company_id = ?", u.Id, targetCompanyID).First(mem).Error; e != nil {
				mem = &UserOrgMembership{
					UserId:       u.Id,
					CompanyId:    targetCompanyID,
					DepartmentId: targetDepartmentID,
					Role:         OrgRoleMember,
					Source:       "migration",
					Status:       OrgMembershipStatusActive,
					CreatedAt:    now,
					UpdatedAt:    now,
				}
				if ce := tx.Create(mem).Error; ce != nil {
					return ce
				}
			} else {
				if ue := tx.Model(mem).Updates(map[string]interface{}{
					"department_id": targetDepartmentID,
					"status":        OrgMembershipStatusActive,
					"updated_at":    now,
				}).Error; ue != nil {
					return ue
				}
			}
			return nil
		})
		if err != nil {
			report.Failed++
			if len(report.Examples) < 20 {
				report.Examples = append(report.Examples, fmt.Sprintf("user=%d err=%v", u.Id, err))
			}
			continue
		}
		report.Updated++
	}

	return report, nil
}

func EnsureOrgPoolsOnStartup() error {
	_, err := EnsureDefaultOrgPools()
	return err
}

func GetActiveUserOrgMemberships(userId int) ([]*UserOrgMembership, error) {
	var items []*UserOrgMembership
	err := DB.Where("user_id = ? AND status = ?", userId, OrgMembershipStatusActive).Find(&items).Error
	return items, err
}

func GetUserOrgMembershipByTenant(userId int, tenantId int) (*UserOrgMembership, error) {
	var item UserOrgMembership
	err := DB.Where("user_id = ? AND tenant_id = ? AND status = ?", userId, tenantId, OrgMembershipStatusActive).First(&item).Error
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func GetUserOrgMembershipByDepartment(userId int, departmentId int) (*UserOrgMembership, error) {
	var item UserOrgMembership
	err := DB.Where("user_id = ? AND department_id = ? AND status = ?", userId, departmentId, OrgMembershipStatusActive).First(&item).Error
	if err != nil {
		return nil, err
	}
	return &item, nil
}
