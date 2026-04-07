package controller

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/pagoda-inference/one-api/common/ctxkey"
	"github.com/pagoda-inference/one-api/model"
)

// CreateDepartment handles POST /api/company/:id/department
func CreateDepartment(c *gin.Context) {
	companyId, _ := strconv.Atoi(c.Param("id"))
	userId := c.GetInt(ctxkey.UserId)
	userRole := c.GetInt(ctxkey.Role)

	// Only root can create department
	if userRole != model.RoleRootUser {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "Only root can create department"})
		return
	}

	var req struct {
		Name       string `json:"name" binding:"required"`
		Code      string `json:"code"`
		Description string `json:"description"`
		QuotaLimit int64  `json:"quota_limit"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid request"})
		return
	}

	// Verify company exists
	_, err := model.GetCompanyById(companyId)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "Company not found"})
		return
	}

	department := &model.Department{
		CompanyId:   companyId,
		Name:        req.Name,
		Code:       req.Code,
		Description: req.Description,
		QuotaLimit:  req.QuotaLimit,
		Status:      model.DepartmentStatusActive,
	}

	if err := model.CreateDepartment(department); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Failed to create department: " + err.Error()})
		return
	}

	// Record audit log
	model.RecordAuditLog(companyId, userId, ActionCreateDepartment, "department", department.Id, "", c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    department,
	})
}

// GetDepartments handles GET /api/company/:id/departments
func GetDepartments(c *gin.Context) {
	companyId, _ := strconv.Atoi(c.Param("id"))
	userId := c.GetInt(ctxkey.UserId)
	userRole := c.GetInt(ctxkey.Role)

	// Only root can view departments
	if userRole != model.RoleRootUser {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "Only root can view departments"})
		return
	}

	departments, err := model.GetDepartmentsByCompany(companyId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Failed to get departments"})
		return
	}

	type DepartmentWithTeams struct {
		model.Department
		TeamCount int `json:"team_count"`
	}

	result := make([]*DepartmentWithTeams, len(departments))
	for i, dept := range departments {
		teamCount, _ := model.CountDepartmentTeams(dept.Id)
		result[i] = &DepartmentWithTeams{
			Department: *dept,
			TeamCount:  int(teamCount),
		}
	}

	// Record audit log
	model.RecordAuditLog(companyId, userId, "view_departments", "department", 0, "", c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}

// GetDepartment handles GET /api/department/:id
func GetDepartment(c *gin.Context) {
	departmentId, _ := strconv.Atoi(c.Param("id"))
	userRole := c.GetInt(ctxkey.Role)

	// Only root can view department details
	if userRole != model.RoleRootUser {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "Only root can view department details"})
		return
	}

	department, err := model.GetDepartmentById(departmentId)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "Department not found"})
		return
	}

	// Get teams in this department
	teams, _ := model.GetDepartmentTeams(departmentId)

	type DepartmentDetail struct {
		model.Department
		Teams []*model.Tenant `json:"teams"`
	}

	detail := &DepartmentDetail{
		Department: *department,
		Teams:      teams,
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    detail,
	})
}

// UpdateDepartment handles PUT /api/department/:id
func UpdateDepartment(c *gin.Context) {
	departmentId, _ := strconv.Atoi(c.Param("id"))
	userId := c.GetInt(ctxkey.UserId)
	userRole := c.GetInt(ctxkey.Role)

	// Only root can update department
	if userRole != model.RoleRootUser {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "Only root can update department"})
		return
	}

	var req struct {
		Name        string `json:"name"`
		Code       string `json:"code"`
		Description string `json:"description"`
		QuotaLimit int64  `json:"quota_limit"`
		Status      int    `json:"status"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid request"})
		return
	}

	department, err := model.GetDepartmentById(departmentId)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "Department not found"})
		return
	}

	if req.Name != "" {
		department.Name = req.Name
	}
	if req.Code != "" {
		department.Code = req.Code
	}
	if req.Description != "" {
		department.Description = req.Description
	}
	if req.QuotaLimit > 0 {
		department.QuotaLimit = req.QuotaLimit
	}
	if req.Status > 0 {
		department.Status = req.Status
	}

	if err := model.UpdateDepartment(department); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Failed to update department"})
		return
	}

	// Record audit log
	model.RecordAuditLog(department.CompanyId, userId, "update_department", "department", departmentId, "", c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    department,
	})
}

// DeleteDepartment handles DELETE /api/department/:id
func DeleteDepartment(c *gin.Context) {
	departmentId, _ := strconv.Atoi(c.Param("id"))
	userId := c.GetInt(ctxkey.UserId)
	userRole := c.GetInt(ctxkey.Role)

	// Only root can delete department
	if userRole != model.RoleRootUser {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "Only root can delete department"})
		return
	}

	department, err := model.GetDepartmentById(departmentId)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "Department not found"})
		return
	}

	if err := model.DeleteDepartment(departmentId); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Failed to delete department"})
		return
	}

	// Record audit log
	model.RecordAuditLog(department.CompanyId, userId, "delete_department", "department", departmentId, "", c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Department deleted successfully",
	})
}
