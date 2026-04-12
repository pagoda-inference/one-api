package controller

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/pagoda-inference/one-api/common/config"
	"github.com/pagoda-inference/one-api/common/ctxkey"
	"github.com/pagoda-inference/one-api/common/message"
	"github.com/pagoda-inference/one-api/model"
)

// GetUserNotifications 获取当前用户的通知列表
func GetUserNotifications(c *gin.Context) {
	userId := c.GetInt(ctxkey.Id)
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	notifications, err := model.GetUserNotifications(userId, limit, offset)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    notifications,
	})
}

// GetUnreadNotificationCount 获取未读通知数量
func GetUnreadNotificationCount(c *gin.Context) {
	userId := c.GetInt(ctxkey.Id)

	count, err := model.GetUnreadNotificationCount(userId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    count,
	})
}

// MarkNotificationAsRead 标记通知为已读
func MarkNotificationAsRead(c *gin.Context) {
	userId := c.GetInt(ctxkey.Id)
	notificationId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "无效的通知ID",
		})
		return
	}

	err = model.MarkNotificationAsRead(notificationId, userId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
	})
}

// MarkAllNotificationsAsRead 标记所有通知为已读
func MarkAllNotificationsAsRead(c *gin.Context) {
	userId := c.GetInt(ctxkey.Id)

	err := model.MarkAllNotificationsAsRead(userId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
	})
}

// CreateNotification 创建通知（仅管理员）
func CreateNotification(c *gin.Context) {
	type CreateReq struct {
		Title   string `json:"title" binding:"required"`
		Content string `json:"content" binding:"required"`
		Type    string `json:"type"`
	}

	var req CreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "参数错误",
		})
		return
	}

	notificationType := "system"
	if req.Type != "" {
		notificationType = req.Type
	}

	// 创建广播通知
	err := model.CreateBroadcastNotification(req.Title, req.Content, notificationType)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	// 发送邮件通知给设置了邮箱的用户
	go sendNotificationEmails(req.Title, req.Content)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "通知已创建并发送给所有用户",
	})
}

// sendNotificationEmails 异步发送通知邮件
func sendNotificationEmails(title, content string) {
	// 获取所有设置了邮箱的用户
	users, err := model.GetUsersWithEmail()
	if err != nil {
		return
	}

	emailContent := message.EmailTemplate(
		title,
		fmt.Sprintf(`<p>系统通知：%s</p><p>%s</p>`, title, content),
	)

	for _, user := range users {
		if user.Email != "" {
			err := message.SendEmail(title, user.Email, emailContent)
			if err != nil {
				continue
			}
		}
	}

	// 同时发送到告警邮箱
	if config.RootUserEmail != "" {
		message.SendEmail(title, config.RootUserEmail, emailContent)
	}
}

// GetAllNotifications 获取所有通知（仅管理员）
func GetAllNotifications(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	notifications, err := model.GetAllNotifications(limit, offset)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    notifications,
	})
}

// DeleteNotification 删除通知（仅管理员）
func DeleteNotification(c *gin.Context) {
	notificationId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "无效的通知ID",
		})
		return
	}

	err = model.DeleteNotification(notificationId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
	})
}