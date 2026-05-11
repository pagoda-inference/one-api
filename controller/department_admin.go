package controller

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/pagoda-inference/one-api/common/ctxkey"
	"github.com/pagoda-inference/one-api/model"
)

// GetDepartmentMembers handles GET /api/department/:id/members (root only)
func GetDepartmentMembers(c *gin.Context) {
	userRole := c.GetInt(ctxkey.Role)
	if userRole != model.RoleRootUser {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "Only root can view department members"})
		return
	}

	departmentId, _ := strconv.Atoi(c.Param("id"))
	if departmentId <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid department id"})
		return
	}

	dept, err := model.GetDepartmentById(departmentId)
	if err != nil || dept == nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "Department not found"})
		return
	}

	users, err := model.GetAllUsersByDepartment(departmentId, 0, 2000, "id", "")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Failed to get department users"})
		return
	}

	memberships, _ := model.ListDepartmentMemberships(departmentId)
	roleByUser := make(map[int]string, len(memberships))
	for _, m := range memberships {
		roleByUser[m.UserId] = m.Role
	}

	type DepartmentMember struct {
		Id               int    `json:"id"`
		Username         string `json:"username"`
		DisplayName      string `json:"display_name"`
		Email            string `json:"email"`
		Role             string `json:"role"`
		IsDepartmentAdmin bool  `json:"is_department_admin"`
	}

	result := make([]*DepartmentMember, 0, len(users))
	for _, u := range users {
		role := roleByUser[u.Id]
		if role == "" {
			role = model.OrgRoleMember
		}
		result = append(result, &DepartmentMember{
			Id:               u.Id,
			Username:         u.Username,
			DisplayName:      u.DisplayName,
			Email:            u.Email,
			Role:             role,
			IsDepartmentAdmin: role == model.OrgRoleDepartmentAdmin,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"department_id":   departmentId,
			"department_name": dept.Name,
			"members":         result,
		},
	})
}

// UpdateDepartmentMemberRole handles PUT /api/department/:id/members/:userId (root only)
func UpdateDepartmentMemberRole(c *gin.Context) {
	userRole := c.GetInt(ctxkey.Role)
	if userRole != model.RoleRootUser {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "Only root can update department member role"})
		return
	}

	departmentId, _ := strconv.Atoi(c.Param("id"))
	targetUserId, _ := strconv.Atoi(c.Param("userId"))
	if departmentId <= 0 || targetUserId <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid id"})
		return
	}

	var req struct {
		IsDepartmentAdmin bool `json:"is_department_admin"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid request"})
		return
	}

	dept, err := model.GetDepartmentById(departmentId)
	if err != nil || dept == nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "Department not found"})
		return
	}

	user, err := model.GetUserById(targetUserId, false)
	if err != nil || user == nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "User not found"})
		return
	}
	if user.DepartmentId != departmentId {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "User is not in this department"})
		return
	}

	role := model.OrgRoleMember
	if req.IsDepartmentAdmin {
		role = model.OrgRoleDepartmentAdmin
	}
	if err = model.UpsertDepartmentMembershipRole(targetUserId, dept.CompanyId, departmentId, role, "manual"); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Failed to update membership"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"user_id":             targetUserId,
			"department_id":       departmentId,
			"is_department_admin": req.IsDepartmentAdmin,
		},
	})
}
