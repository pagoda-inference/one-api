package controller

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/pagoda-inference/one-api/common/ctxkey"
	"github.com/pagoda-inference/one-api/model"
)

// GetLarkOAuthApps handles GET /api/admin/lark-apps
func GetLarkOAuthApps(c *gin.Context) {
	apps, err := model.GetAllLarkOAuthApps()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to get Lark OAuth apps: " + err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    apps,
	})
}

// GetEnabledLarkOAuthApps handles GET /api/lark-apps (public, for login page)
func GetEnabledLarkOAuthApps(c *gin.Context) {
	apps, err := model.GetLarkOAuthApps()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to get Lark OAuth apps: " + err.Error(),
		})
		return
	}
	// Convert to public format (excludes client_secret)
	publicApps := make([]*model.LarkOAuthAppPublic, len(apps))
	for i, app := range apps {
		publicApps[i] = app.ToPublic()
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    publicApps,
	})
}

// CreateLarkOAuthApp handles POST /api/admin/lark-apps
func CreateLarkOAuthApp(c *gin.Context) {
	role := c.GetInt(ctxkey.Role)
	if role < model.RoleAdminUser {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"message": "Admin access required",
		})
		return
	}

	var edit model.LarkOAuthAppEdit
	if err := c.ShouldBindJSON(&edit); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid request: " + err.Error(),
		})
		return
	}

	app := edit.ToLarkOAuthApp()
	if err := model.CreateLarkOAuthApp(app); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to create Lark OAuth app: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    app,
	})
}

// UpdateLarkOAuthApp handles PUT /api/admin/lark-apps/:id
func UpdateLarkOAuthApp(c *gin.Context) {
	role := c.GetInt(ctxkey.Role)
	if role < model.RoleAdminUser {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"message": "Admin access required",
		})
		return
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid app ID",
		})
		return
	}

	// Get existing app
	existingApp, err := model.GetLarkOAuthAppById(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	var edit model.LarkOAuthAppEdit
	if err := c.ShouldBindJSON(&edit); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid request: " + err.Error(),
		})
		return
	}

	// Update fields
	existingApp.Name = edit.Name
	existingApp.ClientId = edit.ClientId
	existingApp.ClientSecret = edit.ClientSecret
	existingApp.Enabled = edit.Enabled
	existingApp.Sort = edit.Sort

	if err := model.UpdateLarkOAuthApp(existingApp); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to update Lark OAuth app: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    existingApp,
	})
}

// DeleteLarkOAuthApp handles DELETE /api/admin/lark-apps/:id
func DeleteLarkOAuthApp(c *gin.Context) {
	role := c.GetInt(ctxkey.Role)
	if role < model.RoleAdminUser {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"message": "Admin access required",
		})
		return
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid app ID",
		})
		return
	}

	if err := model.DeleteLarkOAuthApp(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to delete Lark OAuth app: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Lark OAuth app deleted",
	})
}