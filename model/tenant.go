package model

import (
	"fmt"
	"strings"
	"time"

	"github.com/pagoda-inference/one-api/common/helper"
)

const (
	TenantStatusActive   = 1
	TenantStatusDisabled = 2
	TenantStatusDeleted  = 3
)

// Tenant represents an organization/team in multi-tenant system
type Tenant struct {
	Id           int       `json:"id" gorm:"primarykey"`
	Name         string    `json:"name" gorm:"type:varchar(64);not null"`
	Code         string    `json:"code" gorm:"type:varchar(32);uniqueIndex"` // Unique identifier for API
	Status       int       `json:"status" gorm:"type:int;default:1"`
	OwnerId      int       `json:"owner_id" gorm:"type:int;index"`           // Owner user ID
	QuotaLimit   int64     `json:"quota_limit" gorm:"bigint;default:0"`      // Total quota limit for tenant
	QuotaUsed    int64     `json:"quota_used" gorm:"bigint;default:0"`       // Quota used by tenant
	MaxUsers     int       `json:"max_users" gorm:"type:int;default:10"`     // Max users allowed
	MaxChannels  int       `json:"max_channels" gorm:"type:int;default:5"`   // Max channels allowed
	Settings     string    `json:"settings" gorm:"type:text"`               // JSON settings
	CompanyId    int       `json:"company_id" gorm:"type:int;index"`         // Belongs to which company
	DepartmentId int       `json:"department_id" gorm:"type:int;index"`     // Belongs to which department
	// Rate limiting fields (0 means unlimited)
	RateLimitRpm         int       `json:"rate_limit_rpm" gorm:"type:int;default:0"`
	RateLimitTpm         int       `json:"rate_limit_tpm" gorm:"type:int;default:0"`
	RateLimitConcurrent   int       `json:"rate_limit_concurrent" gorm:"type:int;default:0"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

func (Tenant) TableName() string {
	return "tenants"
}

// UserTenantRole maps users to tenants with specific roles
type UserTenantRole struct {
	Id          int      `json:"id" gorm:"primarykey"`
	UserId      int      `json:"user_id" gorm:"type:int;index"`
	TenantId    int      `json:"tenant_id" gorm:"type:int;index"`
	Role        int      `json:"role" gorm:"type:int;default:2"` // 0=owner, 1=admin, 2=member, 3=viewer
	Permissions string   `json:"permissions" gorm:"type:text"`   // JSON array of extra permissions
	InviterId   int      `json:"inviter_id" gorm:"type:int"`
	Status      int      `json:"status" gorm:"type:int;default:1"` // 1=active, 2=disabled
	CreatedAt   int64    `json:"created_at" gorm:"bigint"`
	UpdatedAt   int64    `json:"updated_at" gorm:"bigint"`
}

func (UserTenantRole) TableName() string {
	return "user_tenant_roles"
}

// TenantRole constants
const (
	RoleOwner  = 0
	RoleAdmin  = 1
	RoleMember = 2
	RoleViewer = 3
)

// Permission constants
const (
	PermissionTopup       = "topup"        // Recharge quota
	PermissionManageUser = "manage_user"  // Manage users
	PermissionAllocQuota = "alloc_quota"  // Allocate quota
	PermissionViewUsage  = "view_usage"   // View usage
	PermissionManageAPI  = "manage_api"   // Manage API keys
	PermissionViewBilling = "view_billing" // View billing
	PermissionManageChannel = "manage_channel" // Manage channels
	PermissionManageModel = "manage_model" // Manage models
	PermissionViewLogs   = "view_logs"    // View logs
	PermissionExport     = "export"       // Export data
)

// GetUserTenants returns all tenants a user belongs to
func GetUserTenants(userId int) ([]*UserTenantRole, error) {
	var roles []*UserTenantRole
	err := DB.Where("user_id = ? AND status = ?", userId, 1).Find(&roles).Error
	return roles, err
}

// GetUserTenantIds returns all tenant IDs a user belongs to
func GetUserTenantIds(userId int) ([]int, error) {
	roles, err := GetUserTenants(userId)
	if err != nil {
		return nil, err
	}
	tenantIds := make([]int, 0, len(roles))
	for _, role := range roles {
		tenantIds = append(tenantIds, role.TenantId)
	}
	return tenantIds, nil
}

// GetTenantById returns tenant by ID
func GetTenantById(id int) (*Tenant, error) {
	var tenant Tenant
	err := DB.First(&tenant, id).Error
	if err != nil {
		return nil, err
	}
	return &tenant, nil
}

// GetAllTenants returns all tenants (for root user)
func GetAllTenants() ([]*Tenant, error) {
	var tenants []*Tenant
	err := DB.Where("status != ?", TenantStatusDeleted).Order("id ASC").Find(&tenants).Error
	return tenants, err
}

// DeleteTenant deletes a tenant and cleans up related data
func DeleteTenant(tenantId int) error {
	// 1. Delete all user_tenant_roles for this tenant
	if err := DB.Where("tenant_id = ?", tenantId).Delete(&UserTenantRole{}).Error; err != nil {
		return err
	}

	// 2. Clean up model_info.visible_to_teams that references this tenant
	// Get all models with this tenant in visible_to_teams
	var models []*ModelInfo
	if err := DB.Where("visible_to_teams LIKE ?", "%,"+fmt.Sprintf("%d", tenantId)+",%").Find(&models).Error; err != nil {
		return err
	}
	// Remove this tenant from visible_to_teams
	for _, m := range models {
		newVisible := strings.ReplaceAll(m.VisibleToTeams, ","+fmt.Sprintf("%d", tenantId)+",", ",")
		DB.Model(m).Update("visible_to_teams", newVisible)
	}

	// 3. Delete the tenant
	return DB.Delete(&Tenant{}, "id = ?", tenantId).Error
}

// GetTenantByCode returns tenant by code
func GetTenantByCode(code string) (*Tenant, error) {
	var tenant Tenant
	err := DB.Where("code = ?", code).First(&tenant).Error
	if err != nil {
		return nil, err
	}
	return &tenant, nil
}

// CreateTenant creates a new tenant
func CreateTenant(tenant *Tenant) error {
	return DB.Create(tenant).Error
}

// UpdateTenant updates tenant info
func UpdateTenant(tenant *Tenant) error {
	return DB.Model(tenant).Updates(tenant).Error
}

// AddUserToTenant adds a user to a tenant with a role
func AddUserToTenant(userId, tenantId, role, inviterId int) error {
	userRole := &UserTenantRole{
		UserId:      userId,
		TenantId:    tenantId,
		Role:        role,
		InviterId:   inviterId,
		Status:      1,
		CreatedAt:   helper.GetTimestamp(),
		UpdatedAt:   helper.GetTimestamp(),
	}
	return DB.Create(userRole).Error
}

// RemoveUserFromTenant removes a user from a tenant
func RemoveUserFromTenant(userId, tenantId int) error {
	return DB.Where("user_id = ? AND tenant_id = ?", userId, tenantId).Delete(&UserTenantRole{}).Error
}

// UpdateUserRoleInTenant updates user's role in a tenant
func UpdateUserRoleInTenant(userId, tenantId, role int) error {
	return DB.Model(&UserTenantRole{}).
		Where("user_id = ? AND tenant_id = ?", userId, tenantId).
		Updates(map[string]interface{}{
			"role":       role,
			"updated_at": helper.GetTimestamp(),
		}).Error
}

// GetTenantUsers returns all users in a tenant
func GetTenantUsers(tenantId int) ([]*User, error) {
	var userIds []int
	err := DB.Model(&UserTenantRole{}).
		Where("tenant_id = ? AND status = ?", tenantId, 1).
		Pluck("user_id", &userIds).Error
	if err != nil {
		return nil, err
	}

	if len(userIds) == 0 {
		return []*User{}, nil
	}

	var users []*User
	err = DB.Where("id IN ?", userIds).Find(&users).Error
	return users, err
}

// GetUserRoleInTenant returns user's role in a tenant
func GetUserRoleInTenant(userId, tenantId int) (*UserTenantRole, error) {
	var role UserTenantRole
	err := DB.Where("user_id = ? AND tenant_id = ?", userId, tenantId).First(&role).Error
	if err != nil {
		return nil, err
	}
	return &role, nil
}

// HasPermission checks if user has specific permission in tenant
func HasPermission(userId, tenantId int, permission string) bool {
	role, err := GetUserRoleInTenant(userId, tenantId)
	if err != nil {
		return false
	}

	// Owner and Admin have all permissions
	if role.Role == RoleOwner || role.Role == RoleAdmin {
		return true
	}

	// Check if permission is in the permissions JSON field
	// For simplicity, just check role-based permissions
	switch permission {
	case PermissionViewUsage, PermissionViewBilling:
		return true // All roles can view
	case PermissionManageAPI:
		return role.Role == RoleMember || role.Role == RoleAdmin
	default:
		return false
	}
}

// CountTenantUsers counts users in a tenant
func CountTenantUsers(tenantId int) (int64, error) {
	var count int64
	err := DB.Model(&UserTenantRole{}).
		Where("tenant_id = ? AND status = ?", tenantId, 1).
		Count(&count).Error
	return count, err
}

// AllocateQuotaToUser allocates quota from tenant to a user
type QuotaAllocation struct {
	Id        int   `json:"id" gorm:"primarykey"`
	TenantId  int   `json:"tenant_id" gorm:"type:int;index"`
	UserId    int   `json:"user_id" gorm:"type:int;index"`
	Quota     int64 `json:"quota" gorm:"bigint"` // Allocated quota
	UsedQuota int64 `json:"used_quota" gorm:"bigint;default:0"`
	CreatedAt int64 `json:"created_at" gorm:"bigint"`
	UpdatedAt int64 `json:"updated_at" gorm:"bigint"`
}

func (QuotaAllocation) TableName() string {
	return "quota_allocations"
}

// AllocateUserQuota allocates quota to a user within tenant
func AllocateUserQuota(tenantId, userId int, quota int64) error {
	var alloc QuotaAllocation
	err := DB.Where("tenant_id = ? AND user_id = ?", tenantId, userId).First(&alloc).Error

	timestamp := helper.GetTimestamp()

	if err != nil {
		// Create new allocation
		alloc = QuotaAllocation{
			TenantId:  tenantId,
			UserId:    userId,
			Quota:     quota,
			UsedQuota: 0,
			CreatedAt: timestamp,
			UpdatedAt: timestamp,
		}
		return DB.Create(&alloc).Error
	}

	// Update existing allocation
	return DB.Model(&alloc).Updates(map[string]interface{}{
		"quota":      alloc.Quota + quota,
		"updated_at": timestamp,
	}).Error
}

// GetUserQuotaAllocation returns user's quota allocation in tenant
func GetUserQuotaAllocation(tenantId, userId int) (*QuotaAllocation, error) {
	var alloc QuotaAllocation
	err := DB.Where("tenant_id = ? AND user_id = ?", tenantId, userId).First(&alloc).Error
	if err != nil {
		return nil, err
	}
	return &alloc, nil
}

// GetAllQuotaAllocations returns all quota allocations in a tenant
func GetAllQuotaAllocations(tenantId int) ([]*QuotaAllocation, error) {
	var allocs []*QuotaAllocation
	err := DB.Where("tenant_id = ?", tenantId).Find(&allocs).Error
	return allocs, err
}

// AuditLog records user actions for audit
type AuditLog struct {
	Id        int    `json:"id" gorm:"primarykey"`
	TenantId  int    `json:"tenant_id" gorm:"type:int;index"`
	UserId    int    `json:"user_id" gorm:"type:int;index"`
	Action    string `json:"action" gorm:"type:varchar(64)"` // create_user, delete_user, allocate_quota, etc.
	Target    string `json:"target" gorm:"type:varchar(64)"` // Target type: user, channel, model, etc.
	TargetId  int    `json:"target_id" gorm:"type:int"`
	Details   string `json:"details" gorm:"type:text"` // JSON details
	IP        string `json:"ip" gorm:"type:varchar(64)"`
	UserAgent string `json:"user_agent" gorm:"type:text"`
	CreatedAt int64  `json:"created_at" gorm:"bigint"`
}

func (AuditLog) TableName() string {
	return "audit_logs"
}

// RecordAuditLog records an audit log entry
func RecordAuditLog(tenantId, userId int, action, target string, targetId int, details string, ip string, userAgent string) error {
	log := &AuditLog{
		TenantId:  tenantId,
		UserId:    userId,
		Action:    action,
		Target:    target,
		TargetId:  targetId,
		Details:   details,
		IP:        ip,
		UserAgent: userAgent,
		CreatedAt: helper.GetTimestamp(),
	}
	return DB.Create(log).Error
}

// GetAuditLogs returns audit logs for a tenant
func GetAuditLogs(tenantId int, startIdx, num int) ([]*AuditLog, error) {
	var logs []*AuditLog
	err := DB.Where("tenant_id = ?", tenantId).
		Order("created_at desc").
		Limit(num).
		Offset(startIdx).
		Find(&logs).Error
	return logs, err
}

// CountAuditLogs counts audit logs for a tenant
func CountAuditLogs(tenantId int) (int64, error) {
	var count int64
	err := DB.Model(&AuditLog{}).Where("tenant_id = ?", tenantId).Count(&count).Error
	return count, err
}