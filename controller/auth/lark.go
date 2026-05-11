package auth

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"

	"github.com/pagoda-inference/one-api/common/config"
	"github.com/pagoda-inference/one-api/common/logger"
	"github.com/pagoda-inference/one-api/controller"
	"github.com/pagoda-inference/one-api/model"
)

type LarkOAuthResponse struct {
	Code        int    `json:"code"`
	Message     string `json:"message"`
	Msg         string `json:"msg"`
	AccessToken string `json:"access_token"`
}

type LarkUser struct {
	Name      string `json:"name"`
	OpenID    string `json:"open_id"`
	UserID    string `json:"user_id"`
	Openid    string `json:"openid"`
	Email     string `json:"email"`
	AvatarUrl string `json:"avatar_url"`
	AvatarURL string `json:"avatar_url_72"`
}

// LarkUserInfoResponse wraps the user info in a "data" field
type LarkUserInfoResponse struct {
	Code    int      `json:"code"`
	Message string   `json:"message"`
	Msg     string   `json:"msg"`
	Data    LarkUser `json:"data"`
}

func larkErrMsg(message, msg string) string {
	if message != "" {
		return message
	}
	if msg != "" {
		return msg
	}
	return "unknown lark error"
}

// getLarkUserDetail fetches email/avatar/department from Contact V3 API
func getLarkUserDetail(accessToken string, openId string) (email, avatarUrl, departmentName string, err error) {
	if openId == "" {
		return "", "", "", errors.New("lark open id is empty")
	}

	// First convert open_id to user_id
	convertReq := map[string]interface{}{
		"open_ids": []string{openId},
	}
	convertBody, err := json.Marshal(convertReq)
	if err != nil {
		return "", "", "", err
	}
	req, err := http.NewRequest("POST", "https://open.feishu.cn/open-apis/contact/v3/users/batch_get_id", bytes.NewBuffer(convertBody))
	if err != nil {
		return "", "", "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	client := http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", "", "", fmt.Errorf("lark contact convert http status %d", resp.StatusCode)
	}
	var convertResp struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Msg     string `json:"msg"`
		Data    struct {
			UserList []struct {
				UserID string `json:"user_id"`
				OpenID string `json:"open_id"`
			} `json:"user_list"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &convertResp); err != nil {
		return "", "", "", err
	}
	if convertResp.Code != 0 {
		return "", "", "", fmt.Errorf("lark contact convert failed: %s", larkErrMsg(convertResp.Message, convertResp.Msg))
	}
	if len(convertResp.Data.UserList) == 0 {
		return "", "", "", errors.New("failed to convert open_id to user_id")
	}
	userId := convertResp.Data.UserList[0].UserID

	// Then get user detail with email/avatar/department ids
	req2, err := http.NewRequest("GET", fmt.Sprintf("https://open.feishu.cn/open-apis/contact/v3/users/%s?user_id_type=user_id", userId), nil)
	if err != nil {
		return "", "", "", err
	}
	req2.Header.Set("Authorization", "Bearer "+accessToken)
	resp2, err := client.Do(req2)
	if err != nil {
		return "", "", "", err
	}
	defer resp2.Body.Close()
	body2, err := io.ReadAll(resp2.Body)
	if err != nil {
		return "", "", "", err
	}
	if resp2.StatusCode != http.StatusOK {
		return "", "", "", fmt.Errorf("lark contact detail http status %d", resp2.StatusCode)
	}
	var detailResp struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Msg     string `json:"msg"`
		Data    struct {
			User struct {
				Email  string `json:"email"`
				Avatar struct {
					AvatarOrigin string `json:"avatar_origin"`
					Avatar72     string `json:"avatar_72"`
				} `json:"avatar"`
				DepartmentIDs     []string `json:"department_ids"`
				OpenDepartmentIDs []string `json:"open_department_ids"`
			} `json:"user"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body2, &detailResp); err != nil {
		return "", "", "", err
	}
	if detailResp.Code != 0 {
		return "", "", "", fmt.Errorf("lark contact detail failed: %s", larkErrMsg(detailResp.Message, detailResp.Msg))
	}

	departmentName = ""
	fetchDept := func(deptID string, idType string) string {
		req3, reqErr := http.NewRequest("GET", fmt.Sprintf("https://open.feishu.cn/open-apis/contact/v3/departments/%s?department_id_type=%s&user_id_type=open_id", deptID, idType), nil)
		if reqErr != nil {
			return ""
		}
		req3.Header.Set("Authorization", "Bearer "+accessToken)
		resp3, doErr := client.Do(req3)
		if doErr != nil {
			return ""
		}
		body3, _ := io.ReadAll(resp3.Body)
		_ = resp3.Body.Close()
		if resp3.StatusCode != http.StatusOK {
			return ""
		}
		var deptResp struct {
			Code int `json:"code"`
			Data struct {
				Department struct {
					Name string `json:"name"`
					I18N struct {
						ZhCN string `json:"zh_cn"`
						EnUS string `json:"en_us"`
					} `json:"i18n_name"`
				} `json:"department"`
			} `json:"data"`
		}
		if json.Unmarshal(body3, &deptResp) != nil || deptResp.Code != 0 {
			return ""
		}
		if deptResp.Data.Department.I18N.ZhCN != "" {
			return deptResp.Data.Department.I18N.ZhCN
		}
		if deptResp.Data.Department.Name != "" {
			return deptResp.Data.Department.Name
		}
		return deptResp.Data.Department.I18N.EnUS
	}
	if len(detailResp.Data.User.DepartmentIDs) > 0 {
		deptID := detailResp.Data.User.DepartmentIDs[0]
		departmentName = fetchDept(deptID, "department_id")
		if departmentName == "" {
			departmentName = fetchDept(deptID, "open_department_id")
		}
	} else if len(detailResp.Data.User.OpenDepartmentIDs) > 0 {
		deptID := detailResp.Data.User.OpenDepartmentIDs[0]
		departmentName = fetchDept(deptID, "open_department_id")
		if departmentName == "" {
			departmentName = fetchDept(deptID, "department_id")
		}
	}
	avatar := detailResp.Data.User.Avatar.AvatarOrigin
	if avatar == "" {
		avatar = detailResp.Data.User.Avatar.Avatar72
	}
	return detailResp.Data.User.Email, avatar, departmentName, nil
}

// getLarkID returns the actual Lark/OpenID, checking multiple possible field names
func (u *LarkUser) getLarkID() string {
	if u.OpenID != "" {
		return u.OpenID
	}
	if u.UserID != "" {
		return u.UserID
	}
	if u.Openid != "" {
		return u.Openid
	}
	return ""
}

func getLarkUserInfoByCode(code string, appId string) (*LarkUser, string, error) {
	if code == "" {
		return nil, "", errors.New("无效的参数")
	}

	var clientId, clientSecret string
	var redirectUri string

	if appId != "" {
		// Multi-app mode: look up app from database
		id, err := strconv.Atoi(appId)
		if err != nil {
			return nil, "", errors.New("无效的飞书应用ID")
		}
		app, err := model.GetLarkOAuthAppById(id)
		if err != nil {
			return nil, "", errors.New("飞书应用不存在或已禁用")
		}
		if !app.Enabled {
			return nil, "", errors.New("飞书应用已禁用")
		}
		clientId = app.ClientId
		clientSecret = app.ClientSecret
		redirectUri = fmt.Sprintf("%s/oauth/lark?app_id=%s", config.ServerAddress, appId)
	} else {
		// Legacy single-app mode: use config
		clientId = config.LarkClientId
		clientSecret = config.LarkClientSecret
		redirectUri = fmt.Sprintf("%s/oauth/lark", config.ServerAddress)
	}

	values := map[string]string{
		"client_id":     clientId,
		"client_secret": clientSecret,
		"code":          code,
		"grant_type":    "authorization_code",
		"redirect_uri":  redirectUri,
	}
	jsonData, err := json.Marshal(values)
	if err != nil {
		return nil, "", err
	}
	req, err := http.NewRequest("POST", "https://open.feishu.cn/open-apis/authen/v2/oauth/token", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	client := http.Client{
		Timeout: 5 * time.Second,
	}
	res, err := client.Do(req)
	if err != nil {
		logger.SysLog(err.Error())
		return nil, "", errors.New("无法连接至飞书服务器，请稍后重试！")
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("飞书 token 请求失败，HTTP %d", res.StatusCode)
	}
	var oAuthResponse LarkOAuthResponse
	tokenBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, "", err
	}
	err = json.Unmarshal(tokenBody, &oAuthResponse)
	if err != nil {
		return nil, "", err
	}
	if oAuthResponse.Code != 0 {
		return nil, "", fmt.Errorf("飞书 token 请求失败: %s", larkErrMsg(oAuthResponse.Message, oAuthResponse.Msg))
	}
	if oAuthResponse.AccessToken == "" {
		return nil, "", errors.New("飞书返回的 access_token 为空")
	}
	req, err = http.NewRequest("GET", fmt.Sprintf("https://open.feishu.cn/open-apis/authen/v1/user_info?access_token=%s", oAuthResponse.AccessToken), nil)
	if err != nil {
		return nil, "", err
	}
	res2, err := client.Do(req)
	if err != nil {
		logger.SysLog(err.Error())
		return nil, "", errors.New("无法连接至飞书服务器，请稍后重试！")
	}
	defer res2.Body.Close()
	if res2.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("飞书用户信息请求失败，HTTP %d", res2.StatusCode)
	}
	var larkUserResp LarkUserInfoResponse
	body, err := io.ReadAll(res2.Body)
	if err != nil {
		return nil, "", err
	}
	err = json.Unmarshal(body, &larkUserResp)
	if err != nil {
		logger.SysLogf("Lark user info unmarshal error: %v", err)
		return nil, "", err
	}
	if larkUserResp.Code != 0 {
		return nil, "", fmt.Errorf("飞书用户信息请求失败: %s", larkErrMsg(larkUserResp.Message, larkUserResp.Msg))
	}
	larkUser := larkUserResp.Data
	logger.SysLogf("Lark user info parsed: name=%s, openid=%s, user_id=%s, open_id=%s", larkUser.Name, larkUser.Openid, larkUser.UserID, larkUser.OpenID)
	return &larkUser, oAuthResponse.AccessToken, nil
}

func LarkOAuth(c *gin.Context) {
	ctx := c.Request.Context()
	session := sessions.Default(c)
	state := c.Query("state")
	if state == "" || session.Get("oauth_state") == nil || state != session.Get("oauth_state").(string) {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"message": "state is empty or not same",
		})
		return
	}
	username := session.Get("username")
	if username != nil {
		LarkBind(c)
		return
	}
	code := c.Query("code")
	appId := c.Query("app_id") // Get app_id from query parameter
	orgSource := "lark_formal"
	if appId != "" {
		if id, convErr := strconv.Atoi(appId); convErr == nil {
			if app, appErr := model.GetLarkOAuthAppById(id); appErr == nil && app != nil {
				switch app.ClientId {
				case "cli_a95e1738cd391bda":
					orgSource = "lark_external"
				case "cli_a94c9bd14ef95bd2":
					orgSource = "lark_formal"
				default:
					orgSource = "lark_formal"
				}
			}
		}
	}
	larkUser, accessToken, err := getLarkUserInfoByCode(code, appId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	// Fetch email and avatar from Lark Contact API; fallback to authen fields when unavailable
	larkEmail, larkAvatar, larkDepartmentName, _ := getLarkUserDetail(accessToken, larkUser.getLarkID())
	// Fallback to robust resolver chain (same as sync-users) when department is missing.
	if strings.TrimSpace(larkDepartmentName) == "" && accessToken != "" && larkUser.getLarkID() != "" {
		if _, _, deptName, _, _, _, detailErr := controller.GetLarkUserDetailByOpenIDForAuth(accessToken, larkUser.getLarkID()); detailErr == nil {
			larkDepartmentName = strings.TrimSpace(deptName)
		}
	}
	if larkEmail == "" {
		larkEmail = larkUser.Email
	}
	if larkAvatar == "" {
		larkAvatar = larkUser.AvatarUrl
	}
	if larkAvatar == "" {
		larkAvatar = larkUser.AvatarURL
	}

	user := model.User{
		LarkId: larkUser.getLarkID(),
	}
	if model.IsLarkIdAlreadyTaken(user.LarkId) {
		err := user.FillUserByLarkId()
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return
		}
		// Update email and avatar for existing users
		if larkEmail != "" {
			user.Email = larkEmail
		}
		if larkAvatar != "" {
			user.AvatarUrl = larkAvatar
		}
		if err := user.Update(false); err != nil {
			logger.SysLogf("LarkOAuth: failed to update user email/avatar, user_id=%d, err=%v", user.Id, err)
		}
		if config.OrgMembershipV2Enabled {
			_ = model.ResolveAndUpsertUserOrg(&user, orgSource)
			_ = model.ResolveAndUpsertUserDepartmentByName(&user, larkDepartmentName, orgSource)
		}
	} else {
		if config.RegisterEnabled {
			user.Username = "lark_" + strconv.Itoa(model.GetMaxUserId()+1)
			if larkUser.Name != "" {
				user.DisplayName = larkUser.Name
			} else {
				user.DisplayName = "Lark User"
			}
			user.Role = model.RoleCommonUser
			user.Status = model.UserStatusEnabled
			user.Email = larkEmail
			user.AvatarUrl = larkAvatar

			if err := user.Insert(ctx, 0); err != nil {
				c.JSON(http.StatusOK, gin.H{
					"success": false,
					"message": err.Error(),
				})
				return
			}
			if config.OrgMembershipV2Enabled {
				_ = model.ResolveAndUpsertUserOrg(&user, orgSource)
				_ = model.ResolveAndUpsertUserDepartmentByName(&user, larkDepartmentName, orgSource)
			}
		} else {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "管理员关闭了新用户注册",
			})
			return
		}
	}

	if user.Status != model.UserStatusEnabled {
		c.JSON(http.StatusOK, gin.H{
			"message": "用户已被封禁",
			"success": false,
		})
		return
	}
	controller.SetupLogin(&user, c)
}

func LarkBind(c *gin.Context) {
	code := c.Query("code")
	appId := c.Query("app_id") // Get app_id from query parameter
	orgSource := "lark_formal"
	if appId != "" {
		if id, convErr := strconv.Atoi(appId); convErr == nil {
			if app, appErr := model.GetLarkOAuthAppById(id); appErr == nil && app != nil {
				switch app.ClientId {
				case "cli_a95e1738cd391bda":
					orgSource = "lark_external"
				case "cli_a94c9bd14ef95bd2":
					orgSource = "lark_formal"
				default:
					orgSource = "lark_formal"
				}
			}
		}
	}
	larkUser, accessToken, err := getLarkUserInfoByCode(code, appId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	// Fetch email and avatar from Lark Contact API; fallback to authen fields when unavailable
	larkEmail, larkAvatar, larkDepartmentName, _ := getLarkUserDetail(accessToken, larkUser.getLarkID())
	if strings.TrimSpace(larkDepartmentName) == "" && accessToken != "" && larkUser.getLarkID() != "" {
		if _, _, deptName, _, _, _, detailErr := controller.GetLarkUserDetailByOpenIDForAuth(accessToken, larkUser.getLarkID()); detailErr == nil {
			larkDepartmentName = strings.TrimSpace(deptName)
		}
	}
	if larkEmail == "" {
		larkEmail = larkUser.Email
	}
	if larkAvatar == "" {
		larkAvatar = larkUser.AvatarUrl
	}
	if larkAvatar == "" {
		larkAvatar = larkUser.AvatarURL
	}

	user := model.User{
		LarkId: larkUser.getLarkID(),
	}
	if model.IsLarkIdAlreadyTaken(user.LarkId) {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "该飞书账户已被绑定",
		})
		return
	}
	session := sessions.Default(c)
	id := session.Get("id")
	// id := c.GetInt("id")  // critical bug!
	user.Id = id.(int)
	err = user.FillUserById()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	user.LarkId = larkUser.getLarkID()
	if larkEmail != "" {
		user.Email = larkEmail
	}
	if larkAvatar != "" {
		user.AvatarUrl = larkAvatar
	}
	err = user.Update(false)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	if config.OrgMembershipV2Enabled {
		_ = model.ResolveAndUpsertUserOrg(&user, orgSource)
		_ = model.ResolveAndUpsertUserDepartmentByName(&user, larkDepartmentName, orgSource)
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "bind",
	})
	return
}
