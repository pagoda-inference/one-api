package controller

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/pagoda-inference/one-api/common/config"
	"github.com/pagoda-inference/one-api/common/ctxkey"
	"github.com/pagoda-inference/one-api/common/helper"
	"github.com/pagoda-inference/one-api/model"
)

// TenantConstants
const (
	ActionCreateUser    = "create_user"
	ActionDeleteUser    = "delete_user"
	ActionUpdateUser    = "update_user"
	ActionAllocQuota    = "allocate_quota"
	ActionCreateChannel = "create_channel"
	ActionDeleteChannel = "delete_channel"
	ActionUpdateChannel = "update_channel"
	ActionCreateToken   = "create_token"
	ActionDeleteToken   = "delete_token"
	ActionLogin         = "login"
	ActionLogout        = "logout"
)

func getTenantScopePermission(userId int, tenantId int) (isDepartmentAdmin bool, isTeamAdmin bool, err error) {
	tenant, err := model.GetTenantById(tenantId)
	if err != nil {
		return false, false, err
	}
	if tenant.DepartmentId > 0 {
		if model.IsDepartmentAdminInDepartment(userId, tenant.DepartmentId) {
			return true, false, nil
		}
	}
	if model.IsTeamAdminInTenant(userId, tenantId) {
		return false, true, nil
	}
	return false, false, nil
}

type TenantPermissions struct {
	CanInviteMember   bool `json:"can_invite_member"`
	CanRemoveMember   bool `json:"can_remove_member"`
	CanSetQuota       bool `json:"can_set_quota"`
	CanSetMemberRole  bool `json:"can_set_member_role"`
	CanDeleteTeam     bool `json:"can_delete_team"`
	CanGrantTeamAdmin bool `json:"can_grant_team_admin"`
	CanUpdateTeam     bool `json:"can_update_team"`
	IsDepartmentAdmin bool `json:"is_department_admin"`
	IsTeamAdmin       bool `json:"is_team_admin"`
	IsOwner           bool `json:"is_owner"`
}

func buildTenantPermissions(userId, userRole, tenantId int, tenant *model.Tenant) TenantPermissions {
	p := TenantPermissions{}
	if userRole == model.RoleRootUser {
		p.CanInviteMember = true
		p.CanRemoveMember = true
		p.CanSetQuota = true
		p.CanSetMemberRole = true
		p.CanDeleteTeam = true
		p.CanGrantTeamAdmin = true
		p.CanUpdateTeam = true
		return p
	}
	if tenant != nil && tenant.DepartmentId > 0 {
		p.IsDepartmentAdmin = model.IsDepartmentAdminInDepartment(userId, tenant.DepartmentId)
	}
	p.IsTeamAdmin = model.IsTeamAdminInTenant(userId, tenantId)
	if role, err := model.GetUserRoleInTenant(userId, tenantId); err == nil && role != nil {
		if role.Role == model.RoleOwner {
			p.IsOwner = true
		}
		// Canonical team permission source:
		// user_tenant_roles.role=admin means this user is team admin in that team.
		if role.Role == model.RoleAdmin {
			p.IsTeamAdmin = true
		}
	}
	manage := p.IsDepartmentAdmin || p.IsTeamAdmin || p.IsOwner
	// Simplified policy:
	// - department_admin/owner manage membership lifecycle (invite/remove)
	// - team_admin manages quota + member/viewer role only
	p.CanInviteMember = p.IsDepartmentAdmin || p.IsOwner
	p.CanRemoveMember = p.IsDepartmentAdmin || p.IsOwner
	p.CanSetQuota = manage
	p.CanSetMemberRole = p.IsDepartmentAdmin || p.IsTeamAdmin || p.IsOwner
	p.CanDeleteTeam = p.IsDepartmentAdmin || p.IsOwner
	p.CanGrantTeamAdmin = p.IsDepartmentAdmin || p.IsOwner
	p.CanUpdateTeam = p.IsDepartmentAdmin || p.IsOwner
	return p
}

func buildTenantPermissionDebug(userId, tenantId int, tenant *model.Tenant) gin.H {
	roleVal := -1
	if role, err := model.GetUserRoleInTenant(userId, tenantId); err == nil && role != nil {
		roleVal = role.Role
	}
	deptID := 0
	if tenant != nil {
		deptID = tenant.DepartmentId
	}
	return gin.H{
		"user_id":               userId,
		"tenant_id":             tenantId,
		"tenant_department_id":  deptID,
		"is_department_admin":   deptID > 0 && model.IsDepartmentAdminInDepartment(userId, deptID),
		"is_team_admin":         model.IsTeamAdminInTenant(userId, tenantId),
		"is_team_admin_by_role": roleVal == model.RoleAdmin,
		"tenant_role":           roleVal,
	}
}

func canAccessTenantInV2(userId int, tenantId int) bool {
	isDeptAdmin, isTeamAdmin, err := getTenantScopePermission(userId, tenantId)
	if err != nil {
		return false
	}
	return isDeptAdmin || isTeamAdmin
}

func canManageTenantInV2(userId int, tenantId int) bool {
	tenant, err := model.GetTenantById(tenantId)
	if err != nil {
		return false
	}
	p := buildTenantPermissions(userId, model.RoleCommonUser, tenantId, tenant)
	return p.CanInviteMember || p.CanSetMemberRole || p.CanSetQuota
}

func canManageTargetUserInV2(userId int, tenantId int, targetUserId int) bool {
	isDeptAdmin, isTeamAdmin, err := getTenantScopePermission(userId, tenantId)
	if err != nil {
		return false
	}
	if !isDeptAdmin && !isTeamAdmin {
		return false
	}
	tenant, err := model.GetTenantById(tenantId)
	if err != nil {
		return false
	}
	if tenant.DepartmentId <= 0 {
		return false
	}
	// Scope rule: department_admin/team_admin can only operate users in same department.
	// Prefer org-membership table to avoid stale users.department_id.
	if model.HasActiveUserOrgMembershipInDepartment(targetUserId, tenant.DepartmentId) {
		return true
	}
	targetUser, err := model.GetUserById(targetUserId, false)
	if err != nil || targetUser == nil {
		return false
	}
	return targetUser.DepartmentId == tenant.DepartmentId
}

// CreateTenant handles POST /api/tenant
func CreateTenant(c *gin.Context) {
	var req struct {
		Name         string `json:"name" binding:"required"`
		Code         string `json:"code" binding:"required"`
		CompanyId    int    `json:"company_id"`
		DepartmentId int    `json:"department_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid request"})
		return
	}

	userId := c.GetInt(ctxkey.Id)
	userRole := c.GetInt(ctxkey.Role)

	// Non-root users should always create teams inside their own org scope.
	// This guarantees root can locate new teams under company/department tree.
	if userRole != model.RoleRootUser {
		// 1) Prefer explicit user profile assignment.
		if (req.CompanyId <= 0 || req.DepartmentId <= 0) && userId > 0 {
			if u, uErr := model.GetUserById(userId, true); uErr == nil && u != nil {
				if u.CompanyId > 0 && u.DepartmentId > 0 {
					req.CompanyId = u.CompanyId
					req.DepartmentId = u.DepartmentId
				}
			}
		}
		// 2) Fallback to org memberships.
		if req.CompanyId <= 0 || req.DepartmentId <= 0 {
			if memberships, mErr := model.GetActiveUserOrgMemberships(userId); mErr == nil {
				for _, m := range memberships {
					if m == nil || m.CompanyId <= 0 || m.DepartmentId <= 0 {
						continue
					}
					// Prefer admin-scoped membership first.
					if m.Role == model.OrgRoleDepartmentAdmin || m.Role == model.OrgRoleTeamAdmin {
						req.CompanyId = m.CompanyId
						req.DepartmentId = m.DepartmentId
						break
					}
					// Fallback to any active membership.
					if req.CompanyId <= 0 || req.DepartmentId <= 0 {
						req.CompanyId = m.CompanyId
						req.DepartmentId = m.DepartmentId
					}
				}
			}
		}
		// 3) Hard fail to prevent orphan teams (department_id=0) invisible in root tree.
		if req.CompanyId <= 0 || req.DepartmentId <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "Cannot resolve company/department for team creation, please sync org memberships first",
			})
			return
		}
	}

	tenant := &model.Tenant{
		Name:         req.Name,
		Code:         req.Code,
		Status:       model.TenantStatusActive,
		OwnerId:      userId,
		CompanyId:    req.CompanyId,
		DepartmentId: req.DepartmentId,
	}

	if err := model.CreateTenant(tenant); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Failed to create tenant: " + err.Error()})
		return
	}

	// Add creator as owner
	if err := model.AddUserToTenant(userId, tenant.Id, model.RoleOwner, 0); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Failed to add user to tenant: " + err.Error()})
		return
	}
	// Keep org-membership tenant linkage for team admin lookup in org-v2.
	if config.OrgMembershipV2Enabled && tenant.CompanyId > 0 && tenant.DepartmentId > 0 {
		_ = model.UpsertDepartmentMembershipRole(userId, tenant.CompanyId, tenant.DepartmentId, model.OrgRoleTeamAdmin, "team_create")
		model.DB.Model(&model.UserOrgMembership{}).
			Where("user_id = ? AND company_id = ? AND department_id = ?", userId, tenant.CompanyId, tenant.DepartmentId).
			Updates(map[string]interface{}{
				"tenant_id":  tenant.Id,
				"updated_at": helper.GetTimestamp(),
			})
	}

	// Record audit log
	model.RecordAuditLog(tenant.Id, userId, ActionCreateUser, "tenant", tenant.Id, "", c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    tenant,
	})
}

// GetMyTenants handles GET /api/tenant
func GetMyTenants(c *gin.Context) {
	userId := c.GetInt(ctxkey.Id)
	userRole := c.GetInt(ctxkey.Role)

	type TenantInfo struct {
		model.Tenant
		Role       int   `json:"role"`
		QuotaAlloc int64 `json:"quota_alloc"`
		UsedQuota  int64 `json:"used_quota"`
		UserCount  int64 `json:"user_count"`
	}

	var tenants []*TenantInfo

	// Root user (role=100) can see all tenants
	if userRole == model.RoleRootUser {
		allTenants, err := model.GetAllTenants()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Failed to get tenants"})
			return
		}
		for _, tenant := range allTenants {
			count, _ := model.CountTenantUsers(tenant.Id)
			tenants = append(tenants, &TenantInfo{
				Tenant:    *tenant,
				Role:      0, // root has owner-level access
				UserCount: count,
			})
		}
	} else {
		roles, err := model.GetUserTenants(userId)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Failed to get tenants"})
			return
		}

		tenants = make([]*TenantInfo, 0, len(roles))
		tenantMap := make(map[int]*TenantInfo)
		for _, role := range roles {
			tenant, err := model.GetTenantById(role.TenantId)
			if err != nil {
				continue
			}

			info := &TenantInfo{
				Tenant: *tenant,
				Role:   role.Role,
			}

			// Get user's quota allocation
			alloc, err := model.GetUserQuotaAllocation(role.TenantId, userId)
			if err == nil {
				info.QuotaAlloc = alloc.Quota
				info.UsedQuota = alloc.UsedQuota
			}

			// Get user count
			count, _ := model.CountTenantUsers(role.TenantId)
			info.UserCount = count

			tenants = append(tenants, info)
			tenantMap[tenant.Id] = info
		}

		// department_admin should be able to view all teams under managed departments.
		// Merge those teams into the same response (without duplicates).
		if memberships, mErr := model.GetActiveUserOrgMemberships(userId); mErr == nil {
			for _, m := range memberships {
				if m == nil || m.Role != model.OrgRoleDepartmentAdmin || m.DepartmentId <= 0 {
					continue
				}
				deptTeams, tErr := model.GetDepartmentTeams(m.DepartmentId)
				if tErr != nil {
					continue
				}
				for _, t := range deptTeams {
					if t == nil {
						continue
					}
					if _, exists := tenantMap[t.Id]; exists {
						continue
					}
					count, _ := model.CountTenantUsers(t.Id)
					info := &TenantInfo{
						Tenant:    *t,
						Role:      model.RoleAdmin, // manage-level view for department admin
						UserCount: count,
					}
					tenants = append(tenants, info)
					tenantMap[t.Id] = info
				}
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    tenants,
	})
}

// GetTenant handles GET /api/tenant/:id
func GetTenant(c *gin.Context) {
	tenantId, _ := strconv.Atoi(c.Param("id"))
	userId := c.GetInt(ctxkey.Id)
	userRole := c.GetInt(ctxkey.Role)

	// Check if user has access to this tenant.
	// root: always allowed
	// org-v2 department/team admin: allowed even if not in user_tenant_roles
	// legacy: must be tenant member
	var role *model.UserTenantRole
	var err error
	if userRole != model.RoleRootUser {
		role, err = model.GetUserRoleInTenant(userId, tenantId)
		if err != nil {
			if config.OrgMembershipV2Enabled && canAccessTenantInV2(userId, tenantId) {
				role = &model.UserTenantRole{Role: model.RoleAdmin, Status: 1}
			} else {
				c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "Access denied"})
				return
			}
		}
		// If user is department/team admin in org-v2, grant manage-level view in this page.
		// This ensures member-management actions (invite/quota/role edit) are visible.
		if config.OrgMembershipV2Enabled && canManageTenantInV2(userId, tenantId) && role != nil && role.Role > model.RoleAdmin {
			role.Role = model.RoleAdmin
		}
	} else {
		role = &model.UserTenantRole{Role: model.RoleOwner, Status: 1}
	}

	tenant, err := model.GetTenantById(tenantId)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "Tenant not found"})
		return
	}

	// Get quota allocations
	allocs, _ := model.GetAllQuotaAllocations(tenantId)

	perms := buildTenantPermissions(userId, userRole, tenantId, tenant)
	canManage := perms.CanInviteMember || perms.CanSetQuota || perms.CanSetMemberRole

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"tenant":            tenant,
			"user_role":         role,
			"can_manage":        canManage,
			"permissions":       perms,
			"permission_debug":  buildTenantPermissionDebug(userId, tenantId, tenant),
			"quota_allocations": allocs,
		},
	})
}

// DeleteTenantScoped handles DELETE /api/tenant/:id
// Permission:
// - root can delete any team
// - legacy owner can delete own team
// - org-v2 department_admin can delete teams under same department
func DeleteTenantScoped(c *gin.Context) {
	tenantId, _ := strconv.Atoi(c.Param("id"))
	userId := c.GetInt(ctxkey.Id)
	userRole := c.GetInt(ctxkey.Role)

	tenant, err := model.GetTenantById(tenantId)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "Tenant not found"})
		return
	}

	allowed := false
	if userRole == model.RoleRootUser {
		allowed = true
	}
	if !allowed {
		if role, e := model.GetUserRoleInTenant(userId, tenantId); e == nil && role.Role == model.RoleOwner {
			allowed = true
		}
	}
	if !allowed {
		perms := buildTenantPermissions(userId, userRole, tenantId, tenant)
		allowed = perms.CanDeleteTeam
	}
	if !allowed {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "Permission denied"})
		return
	}

	if err := model.DeleteTenant(tenantId); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Failed to delete tenant: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Tenant deleted successfully",
	})
}

// GetAllTenantsForAdmin handles GET /api/admin/tenants (for admin use like model visibility config)
func GetAllTenantsForAdmin(c *gin.Context) {
	userRole := c.GetInt(ctxkey.Role)
	if userRole != model.RoleRootUser {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "Admin access required"})
		return
	}

	tenants, err := model.GetAllTenants()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Failed to get tenants"})
		return
	}

	// Return simple list with just id and name
	type SimpleTenant struct {
		Id   int    `json:"id"`
		Name string `json:"name"`
	}
	result := make([]SimpleTenant, 0, len(tenants))
	for _, t := range tenants {
		result = append(result, SimpleTenant{Id: t.Id, Name: t.Name})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}

// DeleteTenant handles DELETE /api/admin/tenants/:id
func DeleteTenant(c *gin.Context) {
	userRole := c.GetInt(ctxkey.Role)
	if userRole != model.RoleRootUser {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "Admin access required"})
		return
	}

	tenantId, _ := strconv.Atoi(c.Param("id"))
	if err := model.DeleteTenant(tenantId); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Failed to delete tenant: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Tenant deleted successfully",
	})
}

// UpdateTenant handles PUT /api/tenant/:id
func UpdateTenant(c *gin.Context) {
	tenantId, _ := strconv.Atoi(c.Param("id"))
	userId := c.GetInt(ctxkey.Id)
	userRole := c.GetInt(ctxkey.Role)

	tenant, err := model.GetTenantById(tenantId)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "Tenant not found"})
		return
	}
	perms := buildTenantPermissions(userId, userRole, tenantId, tenant)
	if !perms.CanUpdateTeam {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "Permission denied"})
		return
	}

	var req struct {
		Name                string `json:"name"`
		Settings            string `json:"settings"`
		MaxUsers            int    `json:"max_users"`
		RateLimitRpm        int    `json:"rate_limit_rpm"`
		RateLimitTpm        int    `json:"rate_limit_tpm"`
		RateLimitConcurrent int    `json:"rate_limit_concurrent"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid request"})
		return
	}

	if req.Name != "" {
		tenant.Name = req.Name
	}
	if req.Settings != "" {
		tenant.Settings = req.Settings
	}
	if req.MaxUsers > 0 {
		tenant.MaxUsers = req.MaxUsers
	}
	if req.RateLimitRpm >= 0 {
		tenant.RateLimitRpm = req.RateLimitRpm
	}
	if req.RateLimitTpm >= 0 {
		tenant.RateLimitTpm = req.RateLimitTpm
	}
	if req.RateLimitConcurrent >= 0 {
		tenant.RateLimitConcurrent = req.RateLimitConcurrent
	}

	if err := model.UpdateTenant(tenant); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Failed to update tenant"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    tenant,
	})
}

// InviteUser handles POST /api/tenant/:id/users
func InviteUser(c *gin.Context) {
	tenantId, _ := strconv.Atoi(c.Param("id"))
	userId := c.GetInt(ctxkey.Id)
	userRole := c.GetInt(ctxkey.Role)

	tenant, tErr := model.GetTenantById(tenantId)
	if tErr != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "Tenant not found"})
		return
	}
	perms := buildTenantPermissions(userId, userRole, tenantId, tenant)
	if !perms.CanInviteMember {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "Permission denied"})
		return
	}

	var req struct {
		UserId int   `json:"user_id" binding:"required"`
		Role   int   `json:"role" binding:"required"`
		Quota  int64 `json:"quota"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid request"})
		return
	}
	if config.OrgMembershipV2Enabled && userRole != model.RoleRootUser {
		if !canManageTargetUserInV2(userId, tenantId, req.UserId) {
			c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "Target user is out of department scope"})
			return
		}
	}

	// Validate role
	if req.Role < model.RoleAdmin || req.Role > model.RoleViewer {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid role"})
		return
	}
	// team_admin can only invite as member/viewer.
	if perms.IsTeamAdmin && req.Role < model.RoleMember {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "Team admin can only invite member/viewer"})
		return
	}

	// Check user count limit
	count, _ := model.CountTenantUsers(tenantId)
	if count >= int64(tenant.MaxUsers) {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Max users reached"})
		return
	}

	// Add user to tenant
	if err := model.AddUserToTenant(req.UserId, tenantId, req.Role, userId); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Failed to invite user"})
		return
	}

	// Allocate quota if specified
	if req.Quota > 0 {
		model.AllocateUserQuota(tenantId, req.UserId, req.Quota)
	}

	// Record audit log
	model.RecordAuditLog(tenantId, userId, ActionCreateUser, "user", req.UserId,
		`{"role":`+strconv.Itoa(req.Role)+`,"quota":`+strconv.FormatInt(req.Quota, 10)+`}`,
		c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "User invited successfully",
	})
}

// RemoveUser handles DELETE /api/tenant/:id/users/:userId
func RemoveUser(c *gin.Context) {
	tenantId, _ := strconv.Atoi(c.Param("id"))
	targetUserId, _ := strconv.Atoi(c.Param("userId"))
	userId := c.GetInt(ctxkey.Id)
	userRole := c.GetInt(ctxkey.Role)

	tenant, tErr := model.GetTenantById(tenantId)
	if tErr != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "Tenant not found"})
		return
	}
	perms := buildTenantPermissions(userId, userRole, tenantId, tenant)
	// Check permission
	var role *model.UserTenantRole
	role, _ = model.GetUserRoleInTenant(userId, tenantId)
	if !perms.CanRemoveMember {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "Permission denied"})
		return
	}

	// Cannot remove owner (unless root)
	targetRole, _ := model.GetUserRoleInTenant(targetUserId, tenantId)
	if targetRole != nil && targetRole.Role == model.RoleOwner && userRole != model.RoleRootUser {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Cannot remove owner"})
		return
	}

	// Cannot remove self if not owner (unless root)
	if targetUserId == userId && (role == nil || role.Role != model.RoleOwner) && userRole != model.RoleRootUser {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Cannot remove yourself"})
		return
	}
	if config.OrgMembershipV2Enabled && userRole != model.RoleRootUser {
		if !canManageTargetUserInV2(userId, tenantId, targetUserId) {
			c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "Target user is out of department scope"})
			return
		}
	}

	if err := model.RemoveUserFromTenant(targetUserId, tenantId); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Failed to remove user"})
		return
	}

	// Record audit log
	model.RecordAuditLog(tenantId, userId, ActionDeleteUser, "user", targetUserId, "", c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "User removed successfully",
	})
}

// UpdateUserRole handles PUT /api/tenant/:id/users/:userId
func UpdateUserRole(c *gin.Context) {
	tenantId, _ := strconv.Atoi(c.Param("id"))
	targetUserId, _ := strconv.Atoi(c.Param("userId"))
	userId := c.GetInt(ctxkey.Id)
	userRole := c.GetInt(ctxkey.Role)

	tenant, tErr := model.GetTenantById(tenantId)
	if tErr != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "Tenant not found"})
		return
	}
	perms := buildTenantPermissions(userId, userRole, tenantId, tenant)
	if !perms.CanSetMemberRole {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "Permission denied"})
		return
	}

	var req struct {
		Role int `json:"role" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid request"})
		return
	}
	if config.OrgMembershipV2Enabled && userRole != model.RoleRootUser {
		if !canManageTargetUserInV2(userId, tenantId, targetUserId) {
			c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "Target user is out of department scope"})
			return
		}
	}

	// Cannot change owner role
	targetRole, _ := model.GetUserRoleInTenant(targetUserId, tenantId)
	if targetRole != nil && targetRole.Role == model.RoleOwner {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Cannot change owner role"})
		return
	}
	// team_admin can only switch non-admin users between member/viewer
	if perms.IsTeamAdmin && !perms.IsDepartmentAdmin && !perms.IsOwner && userRole != model.RoleRootUser {
		if req.Role != model.RoleMember && req.Role != model.RoleViewer {
			c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "Team admin can only set member/viewer"})
			return
		}
		if targetRole != nil && (targetRole.Role == model.RoleAdmin || targetRole.Role == model.RoleOwner) {
			c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "Team admin cannot change admin/owner role"})
			return
		}
	}

	if err := model.UpdateUserRoleInTenant(targetUserId, tenantId, req.Role); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Failed to update role"})
		return
	}

	// Record audit log
	model.RecordAuditLog(tenantId, userId, ActionUpdateUser, "user", targetUserId,
		`{"role":`+strconv.Itoa(req.Role)+`}`,
		c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Role updated successfully",
	})
}

// AllocateUserQuota handles POST /api/tenant/:id/quota
func AllocateUserQuotaAPI(c *gin.Context) {
	tenantId, _ := strconv.Atoi(c.Param("id"))
	userId := c.GetInt(ctxkey.Id)
	userRole := c.GetInt(ctxkey.Role)

	tenant, tErr := model.GetTenantById(tenantId)
	if tErr != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "Tenant not found"})
		return
	}
	perms := buildTenantPermissions(userId, userRole, tenantId, tenant)
	if !perms.CanSetQuota {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "Permission denied"})
		return
	}

	var req struct {
		TargetUserId int   `json:"target_user_id" binding:"required"`
		Quota        int64 `json:"quota" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid request"})
		return
	}
	if config.OrgMembershipV2Enabled && userRole != model.RoleRootUser {
		if !canManageTargetUserInV2(userId, tenantId, req.TargetUserId) {
			c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "Target user is out of department scope"})
			return
		}
	}

	// Verify target user is in this tenant
	targetRole, err := model.GetUserRoleInTenant(req.TargetUserId, tenantId)
	if err != nil || targetRole.Status != 1 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Target user not in tenant"})
		return
	}

	// Check if tenant has enough quota
	if tenant.QuotaLimit > 0 && tenant.QuotaUsed+req.Quota > tenant.QuotaLimit {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Tenant quota limit exceeded"})
		return
	}

	if err := model.AllocateUserQuota(tenantId, req.TargetUserId, req.Quota); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Failed to allocate quota"})
		return
	}

	// Update tenant used quota
	model.DB.Model(&model.Tenant{}).Where("id = ?", tenantId).
		Update("quota_used", model.DB.Raw("quota_used + ?", req.Quota))

	// Record audit log
	model.RecordAuditLog(tenantId, userId, ActionAllocQuota, "user", req.TargetUserId,
		`{"quota":`+strconv.FormatInt(req.Quota, 10)+`}`,
		c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Quota allocated successfully",
	})
}

// GetAuditLogs handles GET /api/tenant/:id/audit
func GetAuditLogsAPI(c *gin.Context) {
	tenantId, _ := strconv.Atoi(c.Param("id"))
	userId := c.GetInt(ctxkey.Id)

	// Check permission (admin/owner in legacy mode, or department_admin/team_admin in org-v2 mode)
	var err error
	allowed := false
	if config.OrgMembershipV2Enabled {
		allowed = canManageTenantInV2(userId, tenantId)
	}
	if !allowed {
		role, roleErr := model.GetUserRoleInTenant(userId, tenantId)
		err = roleErr
		if err == nil && (role.Role == model.RoleOwner || role.Role == model.RoleAdmin) {
			allowed = true
		}
	}
	if !allowed {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "Permission denied"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	logs, err := model.GetAuditLogs(tenantId, offset, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Failed to get audit logs"})
		return
	}

	count, _ := model.CountAuditLogs(tenantId)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"logs":   logs,
			"total":  count,
			"limit":  limit,
			"offset": offset,
		},
	})
}

// GetTenantUsers handles GET /api/tenant/:id/users
func GetTenantUsersAPI(c *gin.Context) {
	tenantId, _ := strconv.Atoi(c.Param("id"))
	userId := c.GetInt(ctxkey.Id)
	userRole := c.GetInt(ctxkey.Role)

	var role *model.UserTenantRole
	var err error

	// Root user can access any tenant
	if userRole != model.RoleRootUser {
		allowed := false
		role, err = model.GetUserRoleInTenant(userId, tenantId)
		if err == nil {
			allowed = true
		}
		if config.OrgMembershipV2Enabled && !allowed {
			if canManageTenantInV2(userId, tenantId) {
				allowed = true
			}
		}
		if !allowed {
			c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "Access denied"})
			return
		}
	}

	users, err := model.GetTenantUsers(tenantId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Failed to get users"})
		return
	}

	type UserWithRole struct {
		model.User
		Role       int   `json:"role"`
		QuotaAlloc int64 `json:"quota_alloc"`
		UsedQuota  int64 `json:"used_quota"`
	}

	result := make([]*UserWithRole, 0, len(users))
	for _, u := range users {
		userRole, _ := model.GetUserRoleInTenant(u.Id, tenantId)
		alloc, _ := model.GetUserQuotaAllocation(tenantId, u.Id)

		uwr := &UserWithRole{
			User:       *u,
			Role:       0,
			QuotaAlloc: 0,
			UsedQuota:  0,
		}
		if userRole != nil {
			uwr.Role = userRole.Role
		}
		if alloc != nil {
			uwr.QuotaAlloc = alloc.Quota
			uwr.UsedQuota = alloc.UsedQuota
		}

		result = append(result, uwr)
	}

	// Record audit log for viewing users
	model.RecordAuditLog(tenantId, userId, "view_users", "tenant", tenantId, "", c.ClientIP(), c.Request.UserAgent())

	// For root user, return owner-level role
	if userRole == model.RoleRootUser {
		role = &model.UserTenantRole{
			Role:   0, // owner-level access
			Status: 1,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"users":     result,
			"user_role": role,
		},
	})
}

// LeaveTenant handles POST /api/tenant/:id/leave
func LeaveTenant(c *gin.Context) {
	tenantId, _ := strconv.Atoi(c.Param("id"))
	userId := c.GetInt(ctxkey.Id)

	role, err := model.GetUserRoleInTenant(userId, tenantId)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Not a member of this tenant"})
		return
	}

	// Owner cannot leave
	if role.Role == model.RoleOwner {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Owner cannot leave, please transfer ownership first"})
		return
	}

	if err := model.RemoveUserFromTenant(userId, tenantId); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Failed to leave tenant"})
		return
	}

	// Record audit log
	model.RecordAuditLog(tenantId, userId, "leave_tenant", "tenant", tenantId, "", c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Left tenant successfully",
	})
}
