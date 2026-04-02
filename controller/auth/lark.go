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
	AccessToken string `json:"access_token"`
}

type LarkUser struct {
	Name   string `json:"name"`
	OpenID string `json:"open_id"`
	UserID string `json:"user_id"`
	Openid string `json:"openid"`
}

// LarkUserInfoResponse wraps the user info in a "data" field
type LarkUserInfoResponse struct {
	Code int      `json:"code"`
	Data LarkUser `json:"data"`
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

func getLarkUserInfoByCode(code string) (*LarkUser, error) {
	if code == "" {
		return nil, errors.New("无效的参数")
	}
	values := map[string]string{
		"client_id":     config.LarkClientId,
		"client_secret": config.LarkClientSecret,
		"code":          code,
		"grant_type":    "authorization_code",
		"redirect_uri":  fmt.Sprintf("%s/oauth/lark", config.ServerAddress),
	}
	jsonData, err := json.Marshal(values)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", "https://open.feishu.cn/open-apis/authen/v2/oauth/token", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	client := http.Client{
		Timeout: 5 * time.Second,
	}
	res, err := client.Do(req)
	if err != nil {
		logger.SysLog(err.Error())
		return nil, errors.New("无法连接至飞书服务器，请稍后重试！")
	}
	defer res.Body.Close()
	var oAuthResponse LarkOAuthResponse
	tokenBody, _ := io.ReadAll(res.Body)
	logger.SysLogf("Lark token response: %s", string(tokenBody))
	err = json.Unmarshal(tokenBody, &oAuthResponse)
	if err != nil {
		return nil, err
	}
	if oAuthResponse.AccessToken == "" {
		return nil, errors.New("飞书返回的 access_token 为空")
	}
	req, err = http.NewRequest("GET", fmt.Sprintf("https://open.feishu.cn/open-apis/authen/v1/user_info?access_token=%s", oAuthResponse.AccessToken), nil)
	if err != nil {
		return nil, err
	}
	res2, err := client.Do(req)
	if err != nil {
		logger.SysLog(err.Error())
		return nil, errors.New("无法连接至飞书服务器，请稍后重试！")
	}
	var larkUserResp LarkUserInfoResponse
	body, _ := io.ReadAll(res2.Body)
	logger.SysLogf("Lark user info response body: %s", string(body))
	err = json.Unmarshal(body, &larkUserResp)
	if err != nil {
		logger.SysLogf("Lark user info unmarshal error: %v", err)
		return nil, err
	}
	larkUser := larkUserResp.Data
	logger.SysLogf("Lark user info parsed: name=%s, openid=%s, user_id=%s, open_id=%s", larkUser.Name, larkUser.Openid, larkUser.UserID, larkUser.OpenID)
	return &larkUser, nil
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
	larkUser, err := getLarkUserInfoByCode(code)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
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

			if err := user.Insert(ctx, 0); err != nil {
				c.JSON(http.StatusOK, gin.H{
					"success": false,
					"message": err.Error(),
				})
				return
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
	larkUser, err := getLarkUserInfoByCode(code)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
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
	err = user.Update(false)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "bind",
	})
	return
}
