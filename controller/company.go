package controller

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/pagoda-inference/one-api/common/ctxkey"
	"github.com/pagoda-inference/one-api/model"
)

// CompanyConstants
const (
	ActionCreateCompany   = "create_company"
	ActionUpdateCompany   = "update_company"
	ActionDeleteCompany   = "delete_company"
	ActionCreateDepartment = "create_department"
)

// CreateCompany handles POST /api/company
func CreateCompany(c *gin.Context) {
	var req struct {
		Name       string `json:"name" binding:"required"`
		Code      string `json:"code" binding:"required"`
		LogoUrl   string `json:"logo_url"`
		Description string `json:"description"`
		QuotaLimit int64  `json:"quota_limit"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid request"})
		return
	}

	userId := c.GetInt(ctxkey.UserId)

	company := &model.Company{
		Name:        req.Name,
		Code:       req.Code,
		LogoUrl:    req.LogoUrl,
		Description: req.Description,
		QuotaLimit: req.QuotaLimit,
		Status:     model.CompanyStatusActive,
	}

	if err := model.CreateCompany(company); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Failed to create company: " + err.Error()})
		return
	}

	// Record audit log
	model.RecordAuditLog(0, userId, ActionCreateCompany, "company", company.Id, "", c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    company,
	})
}

// GetAllCompanies handles GET /api/company
func GetAllCompanies(c *gin.Context) {
	userId := c.GetInt(ctxkey.UserId)
	userRole := c.GetInt(ctxkey.Role)

	// Only root can see all companies
	if userRole != model.RoleRootUser {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "Only root can view all companies"})
		return
	}

	companies, err := model.GetAllCompanies()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Failed to get companies"})
		return
	}

	// Return empty array if no companies exist (but user is root, so they can create one)
	if companies == nil {
		companies = []*model.Company{}
	}

	type CompanyWithDepts struct {
		model.Company
		DepartmentCount int `json:"department_count"`
		TeamCount       int `json:"team_count"`
	}

	result := make([]*CompanyWithDepts, len(companies))
	for i, comp := range companies {
		deptCount, _ := model.CountCompanyDepartments(comp.Id)
		result[i] = &CompanyWithDepts{
			Company:          *comp,
			DepartmentCount: int(deptCount),
		}
	}

	// Record audit log
	model.RecordAuditLog(0, userId, "view_companies", "company", 0, "", c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}

// GetCompany handles GET /api/company/:id
func GetCompany(c *gin.Context) {
	companyId, _ := strconv.Atoi(c.Param("id"))
	userRole := c.GetInt(ctxkey.Role)

	// Only root can view company details
	if userRole != model.RoleRootUser {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "Only root can view company details"})
		return
	}

	company, err := model.GetCompanyById(companyId)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "Company not found"})
		return
	}

	// Get departments
	departments, _ := model.GetCompanyDepartments(companyId)

	type CompanyDetail struct {
		model.Company
		Departments []*model.Department `json:"departments"`
	}

	detail := &CompanyDetail{
		Company:     *company,
		Departments: departments,
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    detail,
	})
}

// UpdateCompany handles PUT /api/company/:id
func UpdateCompany(c *gin.Context) {
	companyId, _ := strconv.Atoi(c.Param("id"))
	userId := c.GetInt(ctxkey.UserId)
	userRole := c.GetInt(ctxkey.Role)

	// Only root can update company
	if userRole != model.RoleRootUser {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "Only root can update company"})
		return
	}

	var req struct {
		Name        string `json:"name"`
		LogoUrl    string `json:"logo_url"`
		Description string `json:"description"`
		QuotaLimit int64  `json:"quota_limit"`
		Status      int    `json:"status"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid request"})
		return
	}

	company, err := model.GetCompanyById(companyId)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "Company not found"})
		return
	}

	if req.Name != "" {
		company.Name = req.Name
	}
	if req.LogoUrl != "" {
		company.LogoUrl = req.LogoUrl
	}
	if req.Description != "" {
		company.Description = req.Description
	}
	if req.QuotaLimit > 0 {
		company.QuotaLimit = req.QuotaLimit
	}
	if req.Status > 0 {
		company.Status = req.Status
	}

	if err := model.UpdateCompany(company); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Failed to update company"})
		return
	}

	// Record audit log
	model.RecordAuditLog(0, userId, ActionUpdateCompany, "company", companyId, "", c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    company,
	})
}

// DeleteCompany handles DELETE /api/company/:id
func DeleteCompany(c *gin.Context) {
	companyId, _ := strconv.Atoi(c.Param("id"))
	userId := c.GetInt(ctxkey.UserId)
	userRole := c.GetInt(ctxkey.Role)

	// Only root can delete company
	if userRole != model.RoleRootUser {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "Only root can delete company"})
		return
	}

	if err := model.DeleteCompany(companyId); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Failed to delete company"})
		return
	}

	// Record audit log
	model.RecordAuditLog(0, userId, ActionDeleteCompany, "company", companyId, "", c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Company deleted successfully",
	})
}
