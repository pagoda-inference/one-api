package model

import (
	"errors"
	"strings"

	"github.com/pagoda-inference/one-api/common"
	"gorm.io/gorm"
)

const (
	VideoTaskStatusQueued    = "queued"
	VideoTaskStatusRunning   = "running"
	VideoTaskStatusSucceeded = "succeeded"
	VideoTaskStatusFailed    = "failed"
)

type VideoGenerationTask struct {
	Id               int64  `json:"-" gorm:"primaryKey;autoIncrement"`
	TaskId           string `json:"id" gorm:"type:varchar(64);uniqueIndex;not null"`
	ProviderTaskId   string `json:"-" gorm:"type:varchar(128);index"`
	UserId           int    `json:"-" gorm:"index;not null"`
	TokenId          int    `json:"-" gorm:"index;not null"`
	ChannelId        int    `json:"-" gorm:"index;not null"`
	Model            string `json:"model" gorm:"type:varchar(128)"`
	Status           string `json:"status" gorm:"type:varchar(32);index"`
	ProviderStatus   string `json:"-" gorm:"type:varchar(32)"`
	Seed             int    `json:"seed"`
	Resolution       string `json:"resolution" gorm:"type:varchar(16)"`
	Ratio            string `json:"ratio" gorm:"type:varchar(16)"`
	Duration         int    `json:"duration"`
	FramesPerSecond  int    `json:"framespersecond"`
	VideoURL         string `json:"-" gorm:"type:text"`
	ContentJSON      string `json:"-" gorm:"type:text"`
	RequestPayload   string `json:"-" gorm:"type:text"`
	ResponsePayload  string `json:"-" gorm:"type:text"`
	ErrorMessage     string `json:"-" gorm:"type:text"`
	PreConsumedQuota int64  `json:"-" gorm:"bigint;default:0"`
	FinalQuota       int64  `json:"-" gorm:"bigint;default:0"`
	QuotaSettled     bool   `json:"-" gorm:"default:false;index"`
	CreatedTime      int64  `json:"created_at" gorm:"bigint;index"`
	UpdatedTime      int64  `json:"updated_at" gorm:"bigint;index"`
	FinishedTime     int64  `json:"-" gorm:"bigint"`
}

func (VideoGenerationTask) TableName() string {
	return "video_generation_tasks"
}

func normalizeVideoTaskStatus(status string) string {
	s := strings.ToLower(strings.TrimSpace(status))
	switch s {
	case "completed":
		return VideoTaskStatusSucceeded
	case VideoTaskStatusQueued, VideoTaskStatusRunning, VideoTaskStatusSucceeded, VideoTaskStatusFailed:
		return s
	default:
		return VideoTaskStatusQueued
	}
}

func CreateVideoTask(task *VideoGenerationTask) error {
	if task == nil {
		return errors.New("nil task")
	}
	task.Status = normalizeVideoTaskStatus(task.Status)
	task.ProviderStatus = task.Status
	return DB.Create(task).Error
}

func GetVideoTaskByTaskId(taskId string) (*VideoGenerationTask, error) {
	var task VideoGenerationTask
	err := DB.Where("task_id = ?", taskId).First(&task).Error
	if err != nil {
		return nil, err
	}
	task.Status = normalizeVideoTaskStatus(task.Status)
	return &task, nil
}

func GetVideoTaskByTaskIdAndUser(taskId string, userId int) (*VideoGenerationTask, error) {
	var task VideoGenerationTask
	err := DB.Where("task_id = ? and user_id = ?", taskId, userId).First(&task).Error
	if err != nil {
		return nil, err
	}
	task.Status = normalizeVideoTaskStatus(task.Status)
	return &task, nil
}

func ListVideoTasksByUser(userId int, pageNum int, pageSize int) ([]*VideoGenerationTask, int64, error) {
	if pageNum < 1 {
		pageNum = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	offset := (pageNum - 1) * pageSize
	var total int64
	if err := DB.Model(&VideoGenerationTask{}).Where("user_id = ?", userId).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var tasks []*VideoGenerationTask
	err := DB.Where("user_id = ?", userId).Order("id desc").Limit(pageSize).Offset(offset).Find(&tasks).Error
	if err != nil {
		return nil, 0, err
	}
	for _, t := range tasks {
		t.Status = normalizeVideoTaskStatus(t.Status)
	}
	return tasks, total, nil
}

func UpdateVideoTaskByTaskId(taskId string, updates map[string]any) error {
	if updates == nil {
		return nil
	}
	if statusRaw, ok := updates["status"]; ok {
		if status, ok := statusRaw.(string); ok {
			updates["status"] = normalizeVideoTaskStatus(status)
		}
	}
	return DB.Model(&VideoGenerationTask{}).Where("task_id = ?", taskId).Updates(updates).Error
}

func MarkVideoTaskDeleted(taskId string) error {
	return DB.Where("task_id = ?", taskId).Delete(&VideoGenerationTask{}).Error
}

func IsNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}

// GetRandomSatisfiedChannelByModelAndTypes finds an enabled channel by model with channel-type constraint.
func GetRandomSatisfiedChannelByModelAndTypes(model string, channelTypes []int) (*Channel, error) {
	if len(channelTypes) == 0 {
		return nil, errors.New("channel type is required")
	}

	ability := Ability{}
	trueVal := "1"
	if common.UsingPostgreSQL {
		trueVal = "true"
	}

	q := DB.Model(&Ability{}).
		Select("abilities.provider, abilities.model, abilities.channel_id, abilities.enabled, abilities.priority").
		Joins("JOIN channels ON channels.id = abilities.channel_id").
		Where("abilities.model = ? AND abilities.enabled = "+trueVal, model).
		Where("channels.status = ?", ChannelStatusEnabled).
		Where("channels.type IN ?", channelTypes).
		Order("abilities.priority DESC")

	var err error
	if common.UsingSQLite || common.UsingPostgreSQL {
		err = q.Order("RANDOM()").First(&ability).Error
	} else {
		err = q.Order("RAND()").First(&ability).Error
	}
	if err != nil {
		return nil, err
	}
	channel := Channel{Id: ability.ChannelId}
	err = DB.First(&channel, "id = ?", ability.ChannelId).Error
	if err != nil {
		return nil, err
	}
	return &channel, nil
}
