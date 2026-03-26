package controller

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/pagoda-inference/one-api/common/ctxkey"
	"github.com/pagoda-inference/one-api/common/helper"
	"github.com/pagoda-inference/one-api/model"
)

// TenantConstants
const (
	ActionCreateUser   = "create_user"
	ActionDeleteUser   = "delete_user"
	ActionUpdateUser   = "update_user"
	ActionAllocQuota   = "allocate_quota"
	ActionCreateChannel = "create_channel"
	ActionDeleteChannel = "delete_channel"
	ActionUpdateChannel = "update_channel"
	ActionCreateToken  = "create_token"
	ActionDeleteToken  = "delete_token"
	ActionLogin        = "login"
	ActionLogout       = "logout"
)

// CreateTenant handles POST /api/tenant
func CreateTenant(c *gin.Context) {
	var req struct {
		Name string `json:"name" binding:"required"`
		Code string `json:"code" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid request"})
		return
	}

	userId := c.GetInt(ctxkey.UserId)

	tenant := &model.Tenant{
		Name:    req.Name,
		Code:    req.Code,
		Status:  model.TenantStatusActive,
		OwnerId: userId,
	}

	if err := model.CreateTenant(tenant); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Failed to create tenant"})
		return
	}

	// Add creator as owner
	if err := model.AddUserToTenant(userId, tenant.Id, model.RoleOwner, 0); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Failed to add user to tenant"})
		return
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
	userId := c.GetInt(ctxkey.UserId)

	roles, err := model.GetUserTenants(userId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Failed to get tenants"})
		return
	}

	type TenantInfo struct {
		model.Tenant
		Role        int   `json:"role"`
		QuotaAlloc  int64 `json:"quota_alloc"`
		UsedQuota   int64 `json:"used_quota"`
		UserCount   int64 `json:"user_count"`
	}

	tenants := make([]*TenantInfo, 0, len(roles))
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
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    tenants,
	})
}

// GetTenant handles GET /api/tenant/:id
func GetTenant(c *gin.Context) {
	tenantId, _ := strconv.Atoi(c.Param("id"))
	userId := c.GetInt(ctxkey.UserId)

	// Check if user has access to this tenant
	role, err := model.GetUserRoleInTenant(userId, tenantId)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "Access denied"})
		return
	}

	tenant, err := model.GetTenantById(tenantId)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "Tenant not found"})
		return
	}

	// Get quota allocations
	allocs, _ := model.GetAllQuotaAllocations(tenantId)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"tenant":            tenant,
			"user_role":         role,
			"quota_allocations": allocs,
		},
	})
}

// UpdateTenant handles PUT /api/tenant/:id
func UpdateTenant(c *gin.Context) {
	tenantId, _ := strconv.Atoi(c.Param("id"))
	userId := c.GetInt(ctxkey.UserId)

	// Check if user has permission (owner only)
	role, err := model.GetUserRoleInTenant(userId, tenantId)
	if err != nil || role.Role != model.RoleOwner {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "Only owner can update tenant"})
		return
	}

	var req struct {
		Name       string `json:"name"`
		Settings   string `json:"settings"`
		MaxUsers   int    `json:"max_users"`
		MaxChannels int   `json:"max_channels"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid request"})
		return
	}

	tenant, err := model.GetTenantById(tenantId)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "Tenant not found"})
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
	if req.MaxChannels > 0 {
		tenant.MaxChannels = req.MaxChannels
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
	userId := c.GetInt(ctxkey.UserId)

	// Check permission (admin or owner)
	role, err := model.GetUserRoleInTenant(userId, tenantId)
	if err != nil || (role.Role != model.RoleOwner && role.Role != model.RoleAdmin) {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "Permission denied"})
		return
	}

	var req struct {
		UserId   int    `json:"user_id" binding:"required"`
		Role     int    `json:"role" binding:"required"`
		Quota    int64  `json:"quota"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid request"})
		return
	}

	// Validate role
	if req.Role < model.RoleAdmin || req.Role > model.RoleViewer {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid role"})
		return
	}

	// Check user count limit
	count, _ := model.CountTenantUsers(tenantId)
	tenant, _ := model.GetTenantById(tenantId)
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
	userId := c.GetInt(ctxkey.UserId)

	// Check permission (admin or owner)
	role, err := model.GetUserRoleInTenant(userId, tenantId)
	if err != nil || (role.Role != model.RoleOwner && role.Role != model.RoleAdmin) {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "Permission denied"})
		return
	}

	// Cannot remove owner
	targetRole, _ := model.GetUserRoleInTenant(targetUserId, tenantId)
	if targetRole.Role == model.RoleOwner {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Cannot remove owner"})
		return
	}

	// Cannot remove self if not owner
	if targetUserId == userId && role.Role != model.RoleOwner {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Cannot remove yourself"})
		return
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
	userId := c.GetInt(ctxkey.UserId)

	// Check permission (owner only)
	role, err := model.GetUserRoleInTenant(userId, tenantId)
	if err != nil || role.Role != model.RoleOwner {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "Only owner can update roles"})
		return
	}

	var req struct {
		Role int `json:"role" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid request"})
		return
	}

	// Cannot change owner role
	if targetRole, _ := model.GetUserRoleInTenant(targetUserId, tenantId); targetRole.Role == model.RoleOwner {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Cannot change owner role"})
		return
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
	userId := c.GetInt(ctxkey.UserId)

	// Check permission (admin or owner)
	role, err := model.GetUserRoleInTenant(userId, tenantId)
	if err != nil || (role.Role != model.RoleOwner && role.Role != model.RoleAdmin) {
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

	// Verify target user is in this tenant
	targetRole, err := model.GetUserRoleInTenant(req.TargetUserId, tenantId)
	if err != nil || targetRole.Status != 1 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Target user not in tenant"})
		return
	}

	// Check if tenant has enough quota
	tenant, _ := model.GetTenantById(tenantId)
	if tenant.QuotaLimit > 0 && tenant.QuotaUsed+req.Quota > tenant.QuotaLimit {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Tenant quota limit exceeded"})
		return
	}

	if err := model.AllocateUserQuota(tenantId, req.TargetUserId, req.Quota); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Failed to allocate quota"})
		return
	}

	// Update tenant used quota
	model.DB().Model(&model.Tenant{}).Where("id = ?", tenantId).
		Update("quota_used", model.DB().Raw("quota_used + ?", req.Quota))

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
	userId := c.GetInt(ctxkey.UserId)

	// Check permission (admin or owner)
	role, err := model.GetUserRoleInTenant(userId, tenantId)
	if err != nil || (role.Role != model.RoleOwner && role.Role != model.RoleAdmin) {
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
	userId := c.GetInt(ctxkey.UserId)

	// Check if user has access
	role, err := model.GetUserRoleInTenant(userId, tenantId)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "Access denied"})
		return
	}

	users, err := model.GetTenantUsers(tenantId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Failed to get users"})
		return
	}

	type UserWithRole struct {
		model.User
		Role      int   `json:"role"`
		QuotaAlloc int64 `json:"quota_alloc"`
		UsedQuota  int64 `json:"used_quota"`
	}

	result := make([]*UserWithRole, 0, len(users))
	for _, u := range users {
		userRole, _ := model.GetUserRoleInTenant(u.Id, tenantId)
		alloc, _ := model.GetUserQuotaAllocation(tenantId, u.Id)

		uwr := &UserWithRole{
			User:      *u,
			Role:      userRole.Role,
			QuotaAlloc: 0,
			UsedQuota:  0,
		}
		if alloc != nil {
			uwr.QuotaAlloc = alloc.Quota
			uwr.UsedQuota = alloc.UsedQuota
		}

		result = append(result, uwr)
	}

	// Record audit log for viewing users
	model.RecordAuditLog(tenantId, userId, "view_users", "tenant", tenantId, "", c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"users":      result,
			"user_role":  role,
		},
	})
}

// LeaveTenant handles POST /api/tenant/:id/leave
func LeaveTenant(c *gin.Context) {
	tenantId, _ := strconv.Atoi(c.Param("id"))
	userId := c.GetInt(ctxkey.UserId)

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