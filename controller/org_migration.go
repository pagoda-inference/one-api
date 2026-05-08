package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pagoda-inference/one-api/common/config"
	"github.com/pagoda-inference/one-api/common/ctxkey"
	"github.com/pagoda-inference/one-api/model"
)

// MigrateOrgUsers handles POST /api/admin/org/migrate-users
// body: {"dry_run": true, "limit": 100000}
func MigrateOrgUsers(c *gin.Context) {
	userRole := c.GetInt(ctxkey.Role)
	if userRole != model.RoleRootUser {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"message": "Only root can run org migration",
		})
		return
	}

	var req struct {
		DryRun bool `json:"dry_run"`
		Limit  int  `json:"limit"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		// default dry-run if body empty
		req.DryRun = true
	}
	if req.Limit <= 0 {
		req.Limit = 100000
	}

	report, err := model.MigrateUsersToDefaultOrgPools(!req.DryRun, req.Limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "org migration failed: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"dry_run": req.DryRun,
			"apply":   !req.DryRun,
			"report":  report,
		},
	})
}

// GetOrgMigrationConfig handles GET /api/admin/org/config
func GetOrgMigrationConfig(c *gin.Context) {
	userRole := c.GetInt(ctxkey.Role)
	if userRole != model.RoleRootUser {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "Only root can view org config"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"org_membership_v2_enabled":  config.OrgMembershipV2Enabled,
			"org_auto_bootstrap_enabled": config.OrgAutoBootstrapEnabled,
			"default_formal_company":     model.DefaultCompanyFormalName,
			"default_external_company":   model.DefaultCompanyExternalName,
		},
	})
}
