package controller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pagoda-inference/one-api/common/ctxkey"
	"github.com/pagoda-inference/one-api/common/logger"
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

type larkTenantTokenResp struct {
	Code              int    `json:"code"`
	Msg               string `json:"msg"`
	Message           string `json:"message"`
	TenantAccessToken string `json:"tenant_access_token"`
}

type larkContactConvertResp struct {
	Code    int    `json:"code"`
	Msg     string `json:"msg"`
	Message string `json:"message"`
	Data    struct {
		UserList []struct {
			UserID string `json:"user_id"`
		} `json:"user_list"`
	} `json:"data"`
}

type larkContactUserDetailResp struct {
	Code    int    `json:"code"`
	Msg     string `json:"msg"`
	Message string `json:"message"`
	Data    struct {
		User struct {
			Email           string `json:"email"`
			EnterpriseEmail string `json:"enterprise_email"`
			Avatar          struct {
				AvatarOrigin string `json:"avatar_origin"`
				Avatar72     string `json:"avatar_72"`
			} `json:"avatar"`
		} `json:"user"`
	} `json:"data"`
}

func larkRespMsg(msg, message string) string {
	if msg != "" {
		return msg
	}
	return message
}

func getLarkTenantAccessToken(app *model.LarkOAuthApp) (string, error) {
	reqBody, err := json.Marshal(map[string]string{
		"app_id":     app.ClientId,
		"app_secret": app.ClientSecret,
	})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequest("POST", "https://open.feishu.cn/open-apis/auth/v3/tenant_access_token/internal", bytes.NewBuffer(reqBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("tenant token http status %d", resp.StatusCode)
	}
	var tokenResp larkTenantTokenResp
	if err = json.Unmarshal(body, &tokenResp); err != nil {
		return "", err
	}
	if tokenResp.Code != 0 {
		return "", fmt.Errorf("tenant token failed: %s", larkRespMsg(tokenResp.Msg, tokenResp.Message))
	}
	if tokenResp.TenantAccessToken == "" {
		return "", fmt.Errorf("tenant token empty")
	}
	return tokenResp.TenantAccessToken, nil
}

func getLarkUserDetailByTenantToken(tenantAccessToken, openID string) (string, string, error) {
	convertReq, err := json.Marshal(map[string]any{
		"open_ids": []string{openID},
	})
	if err != nil {
		return "", "", err
	}
	client := &http.Client{Timeout: 8 * time.Second}
	req, err := http.NewRequest("POST", "https://open.feishu.cn/open-apis/contact/v3/users/batch_get_id", bytes.NewBuffer(convertReq))
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tenantAccessToken)
	resp, err := client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("contact convert http status %d", resp.StatusCode)
	}
	var convertResp larkContactConvertResp
	if err = json.Unmarshal(body, &convertResp); err != nil {
		return "", "", err
	}
	if convertResp.Code != 0 {
		return "", "", fmt.Errorf("contact convert failed: %s", larkRespMsg(convertResp.Msg, convertResp.Message))
	}
	if len(convertResp.Data.UserList) == 0 || convertResp.Data.UserList[0].UserID == "" {
		return "", "", fmt.Errorf("contact convert empty user_id")
	}
	userID := convertResp.Data.UserList[0].UserID

	detailURL := fmt.Sprintf("https://open.feishu.cn/open-apis/contact/v3/users/%s?user_id_type=user_id", userID)
	req2, err := http.NewRequest("GET", detailURL, nil)
	if err != nil {
		return "", "", err
	}
	req2.Header.Set("Authorization", "Bearer "+tenantAccessToken)
	resp2, err := client.Do(req2)
	if err != nil {
		return "", "", err
	}
	defer resp2.Body.Close()
	body2, err := io.ReadAll(resp2.Body)
	if err != nil {
		return "", "", err
	}
	if resp2.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("contact detail http status %d", resp2.StatusCode)
	}
	var detailResp larkContactUserDetailResp
	if err = json.Unmarshal(body2, &detailResp); err != nil {
		return "", "", err
	}
	if detailResp.Code != 0 {
		return "", "", fmt.Errorf("contact detail failed: %s", larkRespMsg(detailResp.Msg, detailResp.Message))
	}
	email := strings.TrimSpace(detailResp.Data.User.Email)
	if email == "" {
		email = strings.TrimSpace(detailResp.Data.User.EnterpriseEmail)
	}
	avatar := strings.TrimSpace(detailResp.Data.User.Avatar.AvatarOrigin)
	if avatar == "" {
		avatar = strings.TrimSpace(detailResp.Data.User.Avatar.Avatar72)
	}
	return email, avatar, nil
}

func getLarkUserDetailByUserID(tenantAccessToken, userID string) (string, string, error) {
	client := &http.Client{Timeout: 8 * time.Second}
	detailURL := fmt.Sprintf("https://open.feishu.cn/open-apis/contact/v3/users/%s?user_id_type=user_id", userID)
	req, err := http.NewRequest("GET", detailURL, nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Authorization", "Bearer "+tenantAccessToken)
	resp, err := client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("contact detail by user_id http status %d", resp.StatusCode)
	}
	var detailResp larkContactUserDetailResp
	if err = json.Unmarshal(body, &detailResp); err != nil {
		return "", "", err
	}
	if detailResp.Code != 0 {
		return "", "", fmt.Errorf("contact detail by user_id failed: %s", larkRespMsg(detailResp.Msg, detailResp.Message))
	}
	email := strings.TrimSpace(detailResp.Data.User.Email)
	if email == "" {
		email = strings.TrimSpace(detailResp.Data.User.EnterpriseEmail)
	}
	avatar := strings.TrimSpace(detailResp.Data.User.Avatar.AvatarOrigin)
	if avatar == "" {
		avatar = strings.TrimSpace(detailResp.Data.User.Avatar.Avatar72)
	}
	return email, avatar, nil
}

// SyncLarkUsersProfile handles POST /api/admin/lark-apps/sync-users
// It backfills email/avatar for existing users who already bound lark_id.
func SyncLarkUsersProfile(c *gin.Context) {
	role := c.GetInt(ctxkey.Role)
	if role < model.RoleAdminUser {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"message": "Admin access required",
		})
		return
	}

	apps, err := model.GetLarkOAuthApps()
	if err != nil || len(apps) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "No enabled Lark OAuth app found",
		})
		return
	}
	users, err := model.GetUsersWithLarkID()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to load lark-bound users: " + err.Error(),
		})
		return
	}

	total := len(users)
	updated := 0
	failed := 0
	skipped := 0
	errorsList := make([]string, 0)
	failedExamples := make([]string, 0)

	tokenByAppID := make(map[int]string, len(apps))
	for _, app := range apps {
		tenantToken, tokenErr := getLarkTenantAccessToken(app)
		if tokenErr != nil {
			logger.SysLogf("SyncLarkUsersProfile: app=%d token error=%v", app.Id, tokenErr)
			errorsList = append(errorsList, fmt.Sprintf("app %s token error: %v", app.Name, tokenErr))
			continue
		}
		tokenByAppID[app.Id] = tenantToken
	}

	if len(tokenByAppID) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "No available lark tenant token",
			"data": gin.H{
				"total":           total,
				"updated":         0,
				"failed":          total,
				"skipped":         0,
				"errors":          errorsList,
				"failed_examples": failedExamples,
			},
		})
		return
	}

	for _, u := range users {
		larkID := strings.TrimSpace(u.LarkId)
		if larkID == "" {
			skipped++
			continue
		}

		var email string
		var avatar string
		var detailErr error

		found := false
		for _, app := range apps {
			tenantToken, ok := tokenByAppID[app.Id]
			if !ok || tenantToken == "" {
				continue
			}

			// Try as open_id first.
			email, avatar, detailErr = getLarkUserDetailByTenantToken(tenantToken, larkID)
			if detailErr == nil && (email != "" || avatar != "") {
				found = true
				break
			}

			// Fallback: some legacy users may store user_id in lark_id.
			email, avatar, detailErr = getLarkUserDetailByUserID(tenantToken, larkID)
			if detailErr == nil && (email != "" || avatar != "") {
				found = true
				break
			}
		}

		if !found {
			failed++
			logger.SysLogf("SyncLarkUsersProfile: user=%d lark_id=%s all apps failed, last_err=%v", u.Id, larkID, detailErr)
			if len(failedExamples) < 20 {
				failedExamples = append(failedExamples, fmt.Sprintf("user=%d lark_id=%s err=%v", u.Id, larkID, detailErr))
			}
			continue
		}

		newEmail := strings.TrimSpace(email)
		newAvatar := strings.TrimSpace(avatar)

		needUpdate := false
		if newEmail != "" && newEmail != u.Email {
			u.Email = newEmail
			needUpdate = true
		}
		if newAvatar != "" && newAvatar != u.AvatarUrl {
			u.AvatarUrl = newAvatar
			needUpdate = true
		}

		if !needUpdate {
			skipped++
			continue
		}
		if err = u.Update(false); err != nil {
			failed++
			logger.SysLogf("SyncLarkUsersProfile: update user=%d failed=%v", u.Id, err)
			continue
		}
		updated++
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"total":           total,
			"updated":         updated,
			"failed":          failed,
			"skipped":         skipped,
			"errors":          errorsList,
			"failed_examples": failedExamples,
		},
	})
}
