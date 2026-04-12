package controller

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"github.com/pagoda-inference/one-api/common"
	"github.com/pagoda-inference/one-api/common/ctxkey"
	"github.com/pagoda-inference/one-api/model"
)

// GetSignInRecords handles GET /api/user/signin/records
func GetSignInRecords(c *gin.Context) {
	userId := c.GetInt(ctxkey.Id)
	if userId == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "未登录"})
		return
	}

	// Get date range, default to last 30 days
	endDate := time.Now().Format("2006-01-02")
	startDate := time.Now().AddDate(0, 0, -30).Format("2006-01-02")

	records, err := model.GetSignInRecords(userId, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "获取签到记录失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    records,
	})
}

// SignIn handles POST /api/user/signin
func SignIn(c *gin.Context) {
	userId := c.GetInt(ctxkey.Id)
	if userId == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "未登录"})
		return
	}

	today := model.TodayDate()
	reward := model.GetTodaySignInReward()

	// Check if already signed in today
	exists, err := model.SignInExists(userId, today)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "签到失败"})
		return
	}
	if exists {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "今日已签到",
		})
		return
	}

	// Check user's current quota - if -1 (unlimited), don't add quota
	user, err := model.GetUserById(userId, false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "签到失败"})
		return
	}

	// Create sign-in record
	signIn := &model.SignIn{
		UserId:    userId,
		Date:      today,
		Quota:     reward,
		CreatedAt: time.Now().Unix(),
	}

	// Only add quota if user quota is not -1 (unlimited)
	addQuota := user.Quota >= 0

	err = model.DB.Transaction(func(tx *gorm.DB) error {
		// Create sign-in record
		if err := tx.Create(signIn).Error; err != nil {
			return err
		}
		// Add quota to user only if quota is not unlimited (-1)
		if addQuota {
			if err := tx.Model(&model.User{}).Where("id = ?", userId).Update("quota", gorm.Expr("quota + ?", reward)).Error; err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "签到失败: " + err.Error()})
		return
	}

	// Record log
	model.RecordLog(c.Request.Context(), userId, model.LogTypeTopup, "签到奖励 "+common.LogQuota(reward))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "签到成功",
		"data": gin.H{
			"quota":     reward,
			"addQuota":  addQuota,
		},
	})
}
