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
	"github.com/pagoda-inference/one-api/common/config"
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
			Email             string   `json:"email"`
			EnterpriseEmail   string   `json:"enterprise_email"`
			DepartmentIDs     []string `json:"department_ids"`
			OpenDepartmentIDs []string `json:"open_department_ids"`
			Avatar            struct {
				AvatarOrigin string `json:"avatar_origin"`
				Avatar72     string `json:"avatar_72"`
			} `json:"avatar"`
		} `json:"user"`
	} `json:"data"`
}

type larkDirectoryDepartmentsFilterResp struct {
	Code    int    `json:"code"`
	Msg     string `json:"msg"`
	Message string `json:"message"`
	Data    struct {
		Items []struct {
			Name             string `json:"name"`
			DepartmentID     string `json:"department_id"`
			OpenDepartmentID string `json:"open_department_id"`
			I18N             struct {
				ZhCN string `json:"zh_cn"`
				EnUS string `json:"en_us"`
			} `json:"i18n_name"`
		} `json:"items"`
		PageResponse struct {
			HasMore   bool   `json:"has_more"`
			PageToken string `json:"page_token"`
		} `json:"page_response"`
	} `json:"data"`
}

type debugLarkDepartmentReq struct {
	AppID        int    `json:"app_id"`
	DepartmentID string `json:"department_id"`
}

func larkRespMsg(msg, message string) string {
	if msg != "" {
		return msg
	}
	return message
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func getLarkDepartmentNameByBatch(client *http.Client, tenantAccessToken, departmentID, idType string) (string, error) {
	departmentID = strings.TrimSpace(departmentID)
	if departmentID == "" {
		return "", nil
	}
	batchURL := fmt.Sprintf("https://open.feishu.cn/open-apis/contact/v3/departments/batch?department_ids=%s&department_id_type=%s&user_id_type=open_id&department_fields=name,i18n_name,department_id,open_department_id", departmentID, idType)
	req, err := http.NewRequest("GET", batchURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+tenantAccessToken)
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
		return "", fmt.Errorf("department batch http status %d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var batchResp struct {
		Code    int    `json:"code"`
		Msg     string `json:"msg"`
		Message string `json:"message"`
		Data    struct {
			Items []struct {
				Name string `json:"name"`
				I18N struct {
					ZhCN string `json:"zh_cn"`
					EnUS string `json:"en_us"`
				} `json:"i18n_name"`
			} `json:"items"`
		} `json:"data"`
	}
	if err = json.Unmarshal(body, &batchResp); err != nil {
		return "", err
	}
	if batchResp.Code != 0 {
		return "", fmt.Errorf("department batch failed: %s", larkRespMsg(batchResp.Msg, batchResp.Message))
	}
	if len(batchResp.Data.Items) == 0 {
		return "", nil
	}
	name := strings.TrimSpace(batchResp.Data.Items[0].I18N.ZhCN)
	if name == "" {
		name = strings.TrimSpace(batchResp.Data.Items[0].Name)
	}
	if name == "" {
		name = strings.TrimSpace(batchResp.Data.Items[0].I18N.EnUS)
	}
	return name, nil
}

func getLarkDepartmentNameByMGet(client *http.Client, tenantAccessToken, departmentID string) (string, string, error) {
	departmentID = strings.TrimSpace(departmentID)
	if departmentID == "" {
		return "", "empty_department_id", nil
	}
	reqBody := map[string]any{
		"department_ids":  []string{departmentID},
		"required_fields": []string{"name"},
	}
	payload, err := json.Marshal(reqBody)
	if err != nil {
		return "", "", err
	}
	url := "https://open.feishu.cn/open-apis/directory/v1/departments/mget?department_id_type=open_department_id&employee_id_type=open_id"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Authorization", "Bearer "+tenantAccessToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return "", "", err
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return "", "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Sprintf("status=%d body=%s", resp.StatusCode, truncate(strings.TrimSpace(string(body)), 220)), nil
	}
	var mgetResp struct {
		Code    int    `json:"code"`
		Msg     string `json:"msg"`
		Message string `json:"message"`
		Data    struct {
			Departments []struct {
				DepartmentID string `json:"department_id"`
				Name         struct {
					DefaultValue string `json:"default_value"`
					I18NValue    struct {
						ZhCN string `json:"zh_cn"`
						EnUS string `json:"en_us"`
						JaJP string `json:"ja_jp"`
					} `json:"i18n_value"`
				} `json:"name"`
			} `json:"departments"`
		} `json:"data"`
	}
	if err = json.Unmarshal(body, &mgetResp); err != nil {
		return "", "", err
	}
	if mgetResp.Code != 0 {
		return "", fmt.Sprintf("code=%d msg=%s", mgetResp.Code, truncate(larkRespMsg(mgetResp.Msg, mgetResp.Message), 120)), nil
	}
	if len(mgetResp.Data.Departments) == 0 {
		return "", "code=0 departments=0", nil
	}
	name := strings.TrimSpace(mgetResp.Data.Departments[0].Name.I18NValue.ZhCN)
	if name == "" {
		name = strings.TrimSpace(mgetResp.Data.Departments[0].Name.DefaultValue)
	}
	if name == "" {
		name = strings.TrimSpace(mgetResp.Data.Departments[0].Name.I18NValue.EnUS)
	}
	if name == "" {
		name = strings.TrimSpace(mgetResp.Data.Departments[0].Name.I18NValue.JaJP)
	}
	return name, fmt.Sprintf("code=0 departments=%d name=%q", len(mgetResp.Data.Departments), truncate(name, 60)), nil
}

func getLarkDepartmentDebugByMGet(client *http.Client, tenantAccessToken, departmentID string) string {
	name, meta, err := getLarkDepartmentNameByMGet(client, tenantAccessToken, departmentID)
	if err != nil {
		return "err:" + truncate(err.Error(), 180)
	}
	if name != "" {
		return "name=" + truncate(name, 80) + " " + meta
	}
	return meta
}

func getLarkDepartmentDebugByBatch(client *http.Client, tenantAccessToken, departmentID, idType string) string {
	departmentID = strings.TrimSpace(departmentID)
	if departmentID == "" {
		return "empty_department_id"
	}
	batchURL := fmt.Sprintf("https://open.feishu.cn/open-apis/contact/v3/departments/batch?department_ids=%s&department_id_type=%s&user_id_type=open_id&department_fields=name,i18n_name,department_id,open_department_id", departmentID, idType)
	req, err := http.NewRequest("GET", batchURL, nil)
	if err != nil {
		return "req_err:" + truncate(err.Error(), 120)
	}
	req.Header.Set("Authorization", "Bearer "+tenantAccessToken)
	resp, err := client.Do(req)
	if err != nil {
		return "http_err:" + truncate(err.Error(), 120)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Sprintf("status=%d body=%s", resp.StatusCode, truncate(strings.TrimSpace(string(body)), 180))
	}
	return "status=200 body=" + truncate(strings.TrimSpace(string(body)), 180)
}

func getLarkDepartmentNameByDirectoryFilterID(client *http.Client, tenantAccessToken, departmentID string) (string, string, error) {
	queries := []struct {
		field string
		dType string
	}{
		{field: "open_department_id", dType: "open_department_id"},
		{field: "department_id", dType: "department_id"},
	}
	for _, q := range queries {
		reqBody := map[string]any{
			"filter": map[string]any{
				"conditions": []map[string]string{
					{
						"field":    q.field,
						"operator": "eq",
						"value":    fmt.Sprintf("\"%s\"", departmentID),
					},
				},
			},
			"required_fields": []string{"name", "i18n_name", "department_id", "open_department_id"},
			"page_request": map[string]any{
				"page_size": 10,
			},
		}
		payload, err := json.Marshal(reqBody)
		if err != nil {
			return "", "", err
		}
		url := fmt.Sprintf("https://open.feishu.cn/open-apis/directory/v1/departments/filter?department_id_type=%s&employee_id_type=open_id", q.dType)
		req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
		if err != nil {
			return "", "", err
		}
		req.Header.Set("Authorization", "Bearer "+tenantAccessToken)
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			continue
		}
		var filterResp larkDirectoryDepartmentsFilterResp
		if err = json.Unmarshal(body, &filterResp); err != nil {
			continue
		}
		if filterResp.Code != 0 || len(filterResp.Data.Items) == 0 {
			continue
		}
		for _, d := range filterResp.Data.Items {
			name := strings.TrimSpace(d.I18N.ZhCN)
			if name == "" {
				name = strings.TrimSpace(d.Name)
			}
			if name == "" {
				name = strings.TrimSpace(d.I18N.EnUS)
			}
			if name != "" {
				return name, fmt.Sprintf("query_field=%s id_type=%s", q.field, q.dType), nil
			}
		}
	}
	return "", "no_name_from_directory_filter_by_id", nil
}

func getLarkDepartmentDebugByDirectoryFilterID(client *http.Client, tenantAccessToken, departmentID string) map[string]string {
	result := map[string]string{}
	queries := []struct {
		key   string
		field string
		dType string
	}{
		{key: "open_department_id", field: "open_department_id", dType: "open_department_id"},
		{key: "department_id", field: "department_id", dType: "department_id"},
	}
	for _, q := range queries {
		reqBody := map[string]any{
			"filter": map[string]any{
				"conditions": []map[string]string{
					{
						"field":    q.field,
						"operator": "eq",
						"value":    fmt.Sprintf("\"%s\"", departmentID),
					},
				},
			},
			"required_fields": []string{"name", "i18n_name", "department_id", "open_department_id"},
			"page_request": map[string]any{
				"page_size": 10,
			},
		}
		payload, err := json.Marshal(reqBody)
		if err != nil {
			result[q.key] = "marshal_err:" + truncate(err.Error(), 120)
			continue
		}
		url := fmt.Sprintf("https://open.feishu.cn/open-apis/directory/v1/departments/filter?department_id_type=%s&employee_id_type=open_id", q.dType)
		req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
		if err != nil {
			result[q.key] = "req_err:" + truncate(err.Error(), 120)
			continue
		}
		req.Header.Set("Authorization", "Bearer "+tenantAccessToken)
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			result[q.key] = "http_err:" + truncate(err.Error(), 120)
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		result[q.key] = fmt.Sprintf("status=%d body=%s", resp.StatusCode, truncate(strings.TrimSpace(string(body)), 220))
	}
	return result
}

func getLarkDepartmentNameMapByDirectoryFilter(client *http.Client, tenantAccessToken string) (map[string]string, error) {
	type queueItem struct {
		parentID string
	}
	deptMap := make(map[string]string, 256)
	visitedParent := make(map[string]bool, 64)
	queue := []queueItem{{parentID: "0"}}
	for len(queue) > 0 {
		item := queue[0]
		queue = queue[1:]
		if visitedParent[item.parentID] {
			continue
		}
		visitedParent[item.parentID] = true

		pageToken := ""
		for {
			reqBody := map[string]any{
				"filter": map[string]any{
					"conditions": []map[string]string{
						{
							"field":    "parent_department_id",
							"operator": "eq",
							"value":    fmt.Sprintf("\"%s\"", item.parentID),
						},
					},
				},
				"required_fields": []string{"name", "i18n_name", "department_id", "open_department_id"},
				"page_request": map[string]any{
					"page_size":  100,
					"page_token": pageToken,
				},
			}
			payload, err := json.Marshal(reqBody)
			if err != nil {
				return nil, err
			}
			req, err := http.NewRequest("POST", "https://open.feishu.cn/open-apis/directory/v1/departments/filter?department_id_type=open_department_id&employee_id_type=open_id", bytes.NewBuffer(payload))
			if err != nil {
				return nil, err
			}
			req.Header.Set("Authorization", "Bearer "+tenantAccessToken)
			req.Header.Set("Content-Type", "application/json")
			resp, err := client.Do(req)
			if err != nil {
				return nil, err
			}
			body, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				return nil, err
			}
			if resp.StatusCode != http.StatusOK {
				return nil, fmt.Errorf("directory departments filter http status %d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
			}
			var filterResp larkDirectoryDepartmentsFilterResp
			if err = json.Unmarshal(body, &filterResp); err != nil {
				return nil, err
			}
			if filterResp.Code != 0 {
				return nil, fmt.Errorf("directory departments filter failed: %s", larkRespMsg(filterResp.Msg, filterResp.Message))
			}

			for _, d := range filterResp.Data.Items {
				name := strings.TrimSpace(d.I18N.ZhCN)
				if name == "" {
					name = strings.TrimSpace(d.Name)
				}
				if name == "" {
					name = strings.TrimSpace(d.I18N.EnUS)
				}
				if strings.TrimSpace(d.DepartmentID) != "" && name != "" {
					deptMap[strings.TrimSpace(d.DepartmentID)] = name
				}
				if strings.TrimSpace(d.OpenDepartmentID) != "" && name != "" {
					deptMap[strings.TrimSpace(d.OpenDepartmentID)] = name
					queue = append(queue, queueItem{parentID: strings.TrimSpace(d.OpenDepartmentID)})
				}
			}

			if !filterResp.Data.PageResponse.HasMore || strings.TrimSpace(filterResp.Data.PageResponse.PageToken) == "" {
				break
			}
			pageToken = strings.TrimSpace(filterResp.Data.PageResponse.PageToken)
		}
	}
	return deptMap, nil
}

func getLarkDepartmentNameMapByContactList(client *http.Client, tenantAccessToken string) (map[string]string, error) {
	deptMap := make(map[string]string, 256)
	pageToken := ""
	for {
		listURL := "https://open.feishu.cn/open-apis/contact/v3/departments?department_id=0&department_id_type=department_id&user_id_type=open_id&page_size=50&fetch_child=true&department_fields=name,i18n_name,department_id,open_department_id"
		if pageToken != "" {
			listURL += "&page_token=" + pageToken
		}
		req, err := http.NewRequest("GET", listURL, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+tenantAccessToken)
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, err
		}
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("contact departments list http status %d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
		}
		var listResp struct {
			Code    int    `json:"code"`
			Msg     string `json:"msg"`
			Message string `json:"message"`
			Data    struct {
				Items []struct {
					Name             string `json:"name"`
					DepartmentID     string `json:"department_id"`
					OpenDepartmentID string `json:"open_department_id"`
					I18N             struct {
						ZhCN string `json:"zh_cn"`
						EnUS string `json:"en_us"`
					} `json:"i18n_name"`
				} `json:"items"`
				HasMore   bool   `json:"has_more"`
				PageToken string `json:"page_token"`
			} `json:"data"`
		}
		if err = json.Unmarshal(body, &listResp); err != nil {
			return nil, err
		}
		if listResp.Code != 0 {
			return nil, fmt.Errorf("contact departments list failed: %s", larkRespMsg(listResp.Msg, listResp.Message))
		}
		for _, d := range listResp.Data.Items {
			name := strings.TrimSpace(d.I18N.ZhCN)
			if name == "" {
				name = strings.TrimSpace(d.Name)
			}
			if name == "" {
				name = strings.TrimSpace(d.I18N.EnUS)
			}
			if name == "" {
				continue
			}
			if strings.TrimSpace(d.DepartmentID) != "" {
				deptMap[strings.TrimSpace(d.DepartmentID)] = name
			}
			if strings.TrimSpace(d.OpenDepartmentID) != "" {
				deptMap[strings.TrimSpace(d.OpenDepartmentID)] = name
			}
		}
		if !listResp.Data.HasMore || strings.TrimSpace(listResp.Data.PageToken) == "" {
			break
		}
		pageToken = strings.TrimSpace(listResp.Data.PageToken)
	}
	return deptMap, nil
}

func getLarkDepartmentRawByContactList(client *http.Client, tenantAccessToken string) string {
	listURL := "https://open.feishu.cn/open-apis/contact/v3/departments?department_id=0&department_id_type=department_id&user_id_type=open_id&page_size=50&fetch_child=true&department_fields=name,i18n_name,department_id,open_department_id"
	req, err := http.NewRequest("GET", listURL, nil)
	if err != nil {
		return "req_err:" + truncate(err.Error(), 160)
	}
	req.Header.Set("Authorization", "Bearer "+tenantAccessToken)
	resp, err := client.Do(req)
	if err != nil {
		return "http_err:" + truncate(err.Error(), 160)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	return fmt.Sprintf("status=%d body=%s", resp.StatusCode, truncate(strings.TrimSpace(string(body)), 1200))
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

func getLarkDepartmentNameByID(client *http.Client, tenantAccessToken string, departmentID string) (string, string, error) {
	if strings.TrimSpace(departmentID) == "" {
		return "", "empty_department_id", nil
	}
	if name, meta, err := getLarkDepartmentNameByMGet(client, tenantAccessToken, departmentID); err == nil && name != "" {
		return name, "hit=directory_mget " + meta, nil
	}

	if name, err := getLarkDepartmentNameByBatch(client, tenantAccessToken, departmentID, "open_department_id"); err == nil && name != "" {
		return name, "hit=batch_open", nil
	}
	if name, err := getLarkDepartmentNameByBatch(client, tenantAccessToken, departmentID, "department_id"); err == nil && name != "" {
		return name, "hit=batch_department", nil
	}
	if name, meta, err := getLarkDepartmentNameByDirectoryFilterID(client, tenantAccessToken, departmentID); err == nil && name != "" {
		return name, "hit=directory_filter_by_id " + meta, nil
	}
	deptMap, err := getLarkDepartmentNameMapByContactList(client, tenantAccessToken)
	if err == nil && len(deptMap) > 0 {
		if name := strings.TrimSpace(deptMap[strings.TrimSpace(departmentID)]); name != "" {
			return name, fmt.Sprintf("hit=contact_list map=%d", len(deptMap)), nil
		}
	}
	deptMap, err = getLarkDepartmentNameMapByDirectoryFilter(client, tenantAccessToken)
	if err == nil && len(deptMap) > 0 {
		if name := strings.TrimSpace(deptMap[strings.TrimSpace(departmentID)]); name != "" {
			return name, fmt.Sprintf("hit=directory_filter map=%d", len(deptMap)), nil
		}
	}
	debug := fmt.Sprintf("miss dept_id=%s; batch_open=%s; batch_department=%s", departmentID,
		getLarkDepartmentDebugByBatch(client, tenantAccessToken, departmentID, "open_department_id"),
		getLarkDepartmentDebugByBatch(client, tenantAccessToken, departmentID, "department_id"))
	return "", truncate(debug, 600), nil
}

func extractLarkContactUser(detailResp *larkContactUserDetailResp, client *http.Client, tenantAccessToken string) (string, string, string, string, string, string, error) {
	email := strings.TrimSpace(detailResp.Data.User.Email)
	if email == "" {
		email = strings.TrimSpace(detailResp.Data.User.EnterpriseEmail)
	}
	avatar := strings.TrimSpace(detailResp.Data.User.Avatar.AvatarOrigin)
	if avatar == "" {
		avatar = strings.TrimSpace(detailResp.Data.User.Avatar.Avatar72)
	}
	departmentName := ""
	departmentID := ""
	departmentSource := "none"
	if len(detailResp.Data.User.DepartmentIDs) > 0 {
		departmentID = detailResp.Data.User.DepartmentIDs[0]
		departmentSource = "department_ids"
	} else if len(detailResp.Data.User.OpenDepartmentIDs) > 0 {
		departmentID = detailResp.Data.User.OpenDepartmentIDs[0]
		departmentSource = "open_department_ids"
	}
	if departmentID != "" {
		deptName, debugInfo, err := getLarkDepartmentNameByID(client, tenantAccessToken, departmentID)
		if err == nil {
			departmentName = deptName
			return email, avatar, departmentName, departmentSource, departmentID, debugInfo, nil
		}
	}
	return email, avatar, departmentName, departmentSource, departmentID, "no_department_id", nil
}

func getLarkUserDetailByTenantToken(tenantAccessToken, openID string) (string, string, string, string, string, string, error) {
	convertReq, err := json.Marshal(map[string]any{
		"open_ids": []string{openID},
	})
	if err != nil {
		return "", "", "", "none", "", "", err
	}
	client := &http.Client{Timeout: 8 * time.Second}
	req, err := http.NewRequest("POST", "https://open.feishu.cn/open-apis/contact/v3/users/batch_get_id", bytes.NewBuffer(convertReq))
	if err != nil {
		return "", "", "", "none", "", "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tenantAccessToken)
	resp, err := client.Do(req)
	if err != nil {
		return "", "", "", "none", "", "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", "", "none", "", "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", "", "", "none", "", "", fmt.Errorf("contact convert http status %d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var convertResp larkContactConvertResp
	if err = json.Unmarshal(body, &convertResp); err != nil {
		return "", "", "", "none", "", "", err
	}
	if convertResp.Code != 0 {
		return "", "", "", "none", "", "", fmt.Errorf("contact convert failed: %s", larkRespMsg(convertResp.Msg, convertResp.Message))
	}
	if len(convertResp.Data.UserList) == 0 || convertResp.Data.UserList[0].UserID == "" {
		return "", "", "", "none", "", "", fmt.Errorf("contact convert empty user_id")
	}
	userID := convertResp.Data.UserList[0].UserID

	detailURL := fmt.Sprintf("https://open.feishu.cn/open-apis/contact/v3/users/%s?user_id_type=user_id", userID)
	req2, err := http.NewRequest("GET", detailURL, nil)
	if err != nil {
		return "", "", "", "none", "", "", err
	}
	req2.Header.Set("Authorization", "Bearer "+tenantAccessToken)
	resp2, err := client.Do(req2)
	if err != nil {
		return "", "", "", "none", "", "", err
	}
	defer resp2.Body.Close()
	body2, err := io.ReadAll(resp2.Body)
	if err != nil {
		return "", "", "", "none", "", "", err
	}
	if resp2.StatusCode != http.StatusOK {
		return "", "", "", "none", "", "", fmt.Errorf("contact detail http status %d body=%s", resp2.StatusCode, strings.TrimSpace(string(body2)))
	}
	var detailResp larkContactUserDetailResp
	if err = json.Unmarshal(body2, &detailResp); err != nil {
		return "", "", "", "none", "", "", err
	}
	if detailResp.Code != 0 {
		return "", "", "", "none", "", "", fmt.Errorf("contact detail failed: %s", larkRespMsg(detailResp.Msg, detailResp.Message))
	}
	return extractLarkContactUser(&detailResp, client, tenantAccessToken)
}

func getLarkUserDetailByUserID(tenantAccessToken, userID string) (string, string, string, string, string, string, error) {
	client := &http.Client{Timeout: 8 * time.Second}
	detailURL := fmt.Sprintf("https://open.feishu.cn/open-apis/contact/v3/users/%s?user_id_type=user_id", userID)
	req, err := http.NewRequest("GET", detailURL, nil)
	if err != nil {
		return "", "", "", "none", "", "", err
	}
	req.Header.Set("Authorization", "Bearer "+tenantAccessToken)
	resp, err := client.Do(req)
	if err != nil {
		return "", "", "", "none", "", "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", "", "none", "", "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", "", "", "none", "", "", fmt.Errorf("contact detail by user_id http status %d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var detailResp larkContactUserDetailResp
	if err = json.Unmarshal(body, &detailResp); err != nil {
		return "", "", "", "none", "", "", err
	}
	if detailResp.Code != 0 {
		return "", "", "", "none", "", "", fmt.Errorf("contact detail by user_id failed: %s", larkRespMsg(detailResp.Msg, detailResp.Message))
	}
	return extractLarkContactUser(&detailResp, client, tenantAccessToken)
}

func getLarkUserDetailByOpenID(tenantAccessToken, openID string) (string, string, string, string, string, string, error) {
	client := &http.Client{Timeout: 8 * time.Second}
	detailURL := fmt.Sprintf("https://open.feishu.cn/open-apis/contact/v3/users/%s?user_id_type=open_id", openID)
	req, err := http.NewRequest("GET", detailURL, nil)
	if err != nil {
		return "", "", "", "none", "", "", err
	}
	req.Header.Set("Authorization", "Bearer "+tenantAccessToken)
	resp, err := client.Do(req)
	if err != nil {
		return "", "", "", "none", "", "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", "", "none", "", "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", "", "", "none", "", "", fmt.Errorf("contact detail by open_id http status %d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var detailResp larkContactUserDetailResp
	if err = json.Unmarshal(body, &detailResp); err != nil {
		return "", "", "", "none", "", "", err
	}
	if detailResp.Code != 0 {
		return "", "", "", "none", "", "", fmt.Errorf("contact detail by open_id failed: %s", larkRespMsg(detailResp.Msg, detailResp.Message))
	}
	return extractLarkContactUser(&detailResp, client, tenantAccessToken)
}

// GetLarkUserDetailByOpenIDForAuth exposes the robust department resolving chain
// used by sync-users, so OAuth login path can reuse the same logic.
func GetLarkUserDetailByOpenIDForAuth(tenantAccessToken, openID string) (string, string, string, string, string, string, error) {
	return getLarkUserDetailByOpenID(tenantAccessToken, openID)
}

// DebugLarkDepartmentResolve handles POST /api/admin/lark-apps/debug-department
func DebugLarkDepartmentResolve(c *gin.Context) {
	role := c.GetInt(ctxkey.Role)
	if role < model.RoleAdminUser {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "Admin access required"})
		return
	}
	var req debugLarkDepartmentReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid request: " + err.Error()})
		return
	}
	departmentID := strings.TrimSpace(req.DepartmentID)
	if departmentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "department_id is required"})
		return
	}

	var app *model.LarkOAuthApp
	var err error
	if req.AppID > 0 {
		app, err = model.GetLarkOAuthAppById(req.AppID)
		if err != nil || app == nil {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "lark app not found"})
			return
		}
	} else {
		apps, e := model.GetAllLarkOAuthApps()
		if e != nil || len(apps) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "no lark app available"})
			return
		}
		app = apps[0]
	}

	tenantToken, err := getLarkTenantAccessToken(app)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "get tenant token failed: " + err.Error()})
		return
	}

	client := &http.Client{Timeout: 8 * time.Second}
	batchOpenName, _ := getLarkDepartmentNameByBatch(client, tenantToken, departmentID, "open_department_id")
	batchDeptName, _ := getLarkDepartmentNameByBatch(client, tenantToken, departmentID, "department_id")
	directoryByIDName, directoryByIDMeta, _ := getLarkDepartmentNameByDirectoryFilterID(client, tenantToken, departmentID)
	contactMap, contactErr := getLarkDepartmentNameMapByContactList(client, tenantToken)
	directoryMap, directoryErr := getLarkDepartmentNameMapByDirectoryFilter(client, tenantToken)

	contactMapName := ""
	if contactErr == nil {
		contactMapName = strings.TrimSpace(contactMap[departmentID])
	}
	directoryMapName := ""
	if directoryErr == nil {
		directoryMapName = strings.TrimSpace(directoryMap[departmentID])
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"app_id":               app.Id,
			"app_name":             app.Name,
			"department_id":        departmentID,
			"mget_name":            func() string { n, _, _ := getLarkDepartmentNameByMGet(client, tenantToken, departmentID); return n }(),
			"mget_raw":             getLarkDepartmentDebugByMGet(client, tenantToken, departmentID),
			"batch_open_name":      batchOpenName,
			"batch_department_name": batchDeptName,
			"batch_open_raw":       getLarkDepartmentDebugByBatch(client, tenantToken, departmentID, "open_department_id"),
			"batch_department_raw": getLarkDepartmentDebugByBatch(client, tenantToken, departmentID, "department_id"),
			"directory_by_id_name": directoryByIDName,
			"directory_by_id_meta": directoryByIDMeta,
			"directory_by_id_raw":  getLarkDepartmentDebugByDirectoryFilterID(client, tenantToken, departmentID),
			"contact_map_name":     contactMapName,
			"contact_map_size":     len(contactMap),
			"contact_map_err":      errString(contactErr),
			"contact_list_raw":     getLarkDepartmentRawByContactList(client, tenantToken),
			"directory_map_name":   directoryMapName,
			"directory_map_size":   len(directoryMap),
			"directory_map_err":    errString(directoryErr),
		},
	})
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
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

	// Use all apps (including disabled) for historical user backfill,
	// because lark open_id is app-scoped.
	apps, err := model.GetAllLarkOAuthApps()
	if err != nil || len(apps) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "No Lark OAuth app found",
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
	orgUpdated := 0
	departmentResolved := 0
	departmentDebugHits := map[string]int{
		"batch_open":       0,
		"batch_department": 0,
		"contact_list":     0,
		"directory_filter": 0,
		"none":             0,
	}
	deptKeyMissExamples := make([]string, 0, 10)
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
		var departmentName string
		var departmentSource string
		var departmentID string
		var departmentDebug string
		var detailErr error

		found := false
		for _, app := range apps {
			tenantToken, ok := tokenByAppID[app.Id]
			if !ok || tenantToken == "" {
				continue
			}

			if strings.HasPrefix(larkID, "ou_") {
				// Normal case: lark_id stores open_id (ou_xxx).
				// Do NOT fall back to convert endpoint here, otherwise the real open_id error gets masked.
				email, avatar, departmentName, departmentSource, departmentID, departmentDebug, detailErr = getLarkUserDetailByOpenID(tenantToken, larkID)
				if detailErr == nil && (email != "" || avatar != "") {
					found = true
					break
				}
				continue
			}

			// Legacy/non-standard case: may store user_id in lark_id.
			email, avatar, departmentName, departmentSource, departmentID, departmentDebug, detailErr = getLarkUserDetailByUserID(tenantToken, larkID)
			if detailErr == nil && (email != "" || avatar != "") {
				found = true
				break
			}

			// Fallback: try as open_id as last resort.
			email, avatar, departmentName, departmentSource, departmentID, departmentDebug, detailErr = getLarkUserDetailByOpenID(tenantToken, larkID)
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

		if needUpdate {
			if err = u.Update(false); err != nil {
				failed++
				logger.SysLogf("SyncLarkUsersProfile: update user=%d failed=%v", u.Id, err)
				continue
			}
			updated++
		}

		// Force org re-classification for all lark users: formal pool + real lark department.
		if config.OrgMembershipV2Enabled {
			u.CompanyId = 0
			u.DepartmentId = 0
			if err = model.ResolveAndUpsertUserOrg(u, "lark"); err == nil {
				if strings.TrimSpace(departmentName) != "" {
					departmentResolved++
				}
				if strings.Contains(departmentDebug, "hit=batch_open") {
					departmentDebugHits["batch_open"]++
				} else if strings.Contains(departmentDebug, "hit=batch_department") {
					departmentDebugHits["batch_department"]++
				} else if strings.Contains(departmentDebug, "hit=contact_list") {
					departmentDebugHits["contact_list"]++
				} else if strings.Contains(departmentDebug, "hit=directory_filter") {
					departmentDebugHits["directory_filter"]++
				} else {
					departmentDebugHits["none"]++
					if len(deptKeyMissExamples) < 10 {
						deptKeyMissExamples = append(deptKeyMissExamples, fmt.Sprintf("user=%d source=%s dept_id=%s debug=%s", u.Id, departmentSource, departmentID, truncate(departmentDebug, 220)))
					}
				}
				_ = model.ResolveAndUpsertUserDepartmentByName(u, departmentName, "lark")
				orgUpdated++
			}
		}
		if !needUpdate {
			skipped++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"total":                    total,
			"updated":                  updated,
			"org_updated":              orgUpdated,
			"department_resolved":      departmentResolved,
			"department_debug_hits":    departmentDebugHits,
			"department_miss_examples": deptKeyMissExamples,
			"failed":                   failed,
			"skipped":                  skipped,
			"errors":                   errorsList,
			"failed_examples":          failedExamples,
		},
	})
}
