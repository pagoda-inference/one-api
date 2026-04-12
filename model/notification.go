package model

import (
	"github.com/pagoda-inference/one-api/common/helper"
)

// Notification represents a system notification
type Notification struct {
	Id        int    `json:"id" gorm:"primaryKey"`
	UserId    int    `json:"user_id" gorm:"index"`           // 0 means broadcast to all users
	Title     string `json:"title" gorm:"size:255"`
	Content   string `json:"content" gorm:"type:text"`
	Type      string `json:"type" gorm:"size:32;default:system"` // system, alert
	IsRead    bool   `json:"is_read" gorm:"default:false"`
	ReadAt    *int64 `json:"read_at" gorm:"bigint"`
	CreatedAt int64  `json:"created_at" gorm:"bigint"`
}

func (Notification) TableName() string {
	return "notifications"
}

// Create creates a new notification
func (n *Notification) Create() error {
	if n.CreatedAt == 0 {
		n.CreatedAt = helper.GetTimestamp()
	}
	return DB.Create(n).Error
}

// CreateBroadcast creates a broadcast notification (UserId = 0)
func CreateBroadcastNotification(title, content, notificationType string) error {
	n := &Notification{
		UserId:    0, // broadcast
		Title:     title,
		Content:   content,
		Type:      notificationType,
		IsRead:    false,
		CreatedAt: helper.GetTimestamp(),
	}
	return n.Create()
}

// GetUserNotifications gets all notifications for a user (including broadcasts)
func GetUserNotifications(userId int, limit int, offset int) ([]*Notification, error) {
	var notifications []*Notification
	query := DB.Where("user_id = ? OR user_id = 0", userId).Order("created_at DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}
	err := query.Find(&notifications).Error
	return notifications, err
}

// GetUnreadCount gets unread notification count for a user
func GetUnreadNotificationCount(userId int) (int64, error) {
	var count int64
	err := DB.Model(&Notification{}).
		Where("(user_id = ? OR user_id = 0) AND is_read = ?", userId, false).
		Count(&count).Error
	return count, err
}

// MarkAsRead marks a notification as read
func MarkNotificationAsRead(notificationId int, userId int) error {
	timestamp := helper.GetTimestamp()
	return DB.Model(&Notification{}).
		Where("id = ? AND (user_id = ? OR user_id = 0)", notificationId, userId).
		Updates(map[string]interface{}{"is_read": true, "read_at": timestamp}).Error
}

// MarkAllAsRead marks all notifications as read for a user
func MarkAllNotificationsAsRead(userId int) error {
	timestamp := helper.GetTimestamp()
	return DB.Model(&Notification{}).
		Where("(user_id = ? OR user_id = 0) AND is_read = ?", userId, false).
		Updates(map[string]interface{}{"is_read": true, "read_at": timestamp}).Error
}

// GetAllNotifications gets all notifications (for admin)
func GetAllNotifications(limit int, offset int) ([]*Notification, error) {
	var notifications []*Notification
	query := DB.Order("created_at DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}
	err := query.Find(&notifications).Error
	return notifications, err
}

// DeleteNotification deletes a notification
func DeleteNotification(notificationId int) error {
	return DB.Delete(&Notification{}, "id = ?", notificationId).Error
}