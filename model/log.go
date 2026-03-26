package model

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/helper"
	"github.com/songquanpeng/one-api/common/logger"
)

type Log struct {
	Id                int    `json:"id"`
	UserId            int    `json:"user_id" gorm:"index"`
	CreatedAt         int64  `json:"created_at" gorm:"bigint;index:idx_created_at_type"`
	Type              int    `json:"type" gorm:"index:idx_created_at_type"`
	Content           string `json:"content"`
	Username          string `json:"username" gorm:"index:index_username_model_name,priority:2;default:''"`
	TokenName         string `json:"token_name" gorm:"index;default:''"`
	ModelName         string `json:"model_name" gorm:"index;index:index_username_model_name,priority:1;default:''"`
	Quota             int    `json:"quota" gorm:"default:0"`
	PromptTokens      int    `json:"prompt_tokens" gorm:"default:0"`
	CompletionTokens  int    `json:"completion_tokens" gorm:"default:0"`
	ChannelId         int    `json:"channel" gorm:"index"`
	RequestId         string `json:"request_id" gorm:"default:''"`
	ElapsedTime       int64  `json:"elapsed_time" gorm:"default:0"` // unit is ms
	IsStream          bool   `json:"is_stream" gorm:"default:false"`
	SystemPromptReset bool   `json:"system_prompt_reset" gorm:"default:false"`
}

const (
	LogTypeUnknown = iota
	LogTypeTopup
	LogTypeConsume
	LogTypeManage
	LogTypeSystem
	LogTypeTest
)

func recordLogHelper(ctx context.Context, log *Log) {
	requestId := helper.GetRequestID(ctx)
	log.RequestId = requestId
	err := LOG_DB.Create(log).Error
	if err != nil {
		logger.Error(ctx, "failed to record log: "+err.Error())
		return
	}
	logger.Infof(ctx, "record log: %+v", log)
}

func RecordLog(ctx context.Context, userId int, logType int, content string) {
	if logType == LogTypeConsume && !config.LogConsumeEnabled {
		return
	}
	log := &Log{
		UserId:    userId,
		Username:  GetUsernameById(userId),
		CreatedAt: helper.GetTimestamp(),
		Type:      logType,
		Content:   content,
	}
	recordLogHelper(ctx, log)
}

func RecordTopupLog(ctx context.Context, userId int, content string, quota int) {
	log := &Log{
		UserId:    userId,
		Username:  GetUsernameById(userId),
		CreatedAt: helper.GetTimestamp(),
		Type:      LogTypeTopup,
		Content:   content,
		Quota:     quota,
	}
	recordLogHelper(ctx, log)
}

func RecordConsumeLog(ctx context.Context, log *Log) {
	if !config.LogConsumeEnabled {
		return
	}
	log.Username = GetUsernameById(log.UserId)
	log.CreatedAt = helper.GetTimestamp()
	log.Type = LogTypeConsume
	recordLogHelper(ctx, log)
}

func RecordTestLog(ctx context.Context, log *Log) {
	log.CreatedAt = helper.GetTimestamp()
	log.Type = LogTypeTest
	recordLogHelper(ctx, log)
}

func GetAllLogs(logType int, startTimestamp int64, endTimestamp int64, modelName string, username string, tokenName string, startIdx int, num int, channel int) (logs []*Log, err error) {
	var tx *gorm.DB
	if logType == LogTypeUnknown {
		tx = LOG_DB
	} else {
		tx = LOG_DB.Where("type = ?", logType)
	}
	if modelName != "" {
		tx = tx.Where("model_name = ?", modelName)
	}
	if username != "" {
		tx = tx.Where("username = ?", username)
	}
	if tokenName != "" {
		tx = tx.Where("token_name = ?", tokenName)
	}
	if startTimestamp != 0 {
		tx = tx.Where("created_at >= ?", startTimestamp)
	}
	if endTimestamp != 0 {
		tx = tx.Where("created_at <= ?", endTimestamp)
	}
	if channel != 0 {
		tx = tx.Where("channel_id = ?", channel)
	}
	err = tx.Order("id desc").Limit(num).Offset(startIdx).Find(&logs).Error
	return logs, err
}

func GetUserLogs(userId int, logType int, startTimestamp int64, endTimestamp int64, modelName string, tokenName string, startIdx int, num int) (logs []*Log, err error) {
	var tx *gorm.DB
	if logType == LogTypeUnknown {
		tx = LOG_DB.Where("user_id = ?", userId)
	} else {
		tx = LOG_DB.Where("user_id = ? and type = ?", userId, logType)
	}
	if modelName != "" {
		tx = tx.Where("model_name = ?", modelName)
	}
	if tokenName != "" {
		tx = tx.Where("token_name = ?", tokenName)
	}
	if startTimestamp != 0 {
		tx = tx.Where("created_at >= ?", startTimestamp)
	}
	if endTimestamp != 0 {
		tx = tx.Where("created_at <= ?", endTimestamp)
	}
	err = tx.Order("id desc").Limit(num).Offset(startIdx).Omit("id").Find(&logs).Error
	return logs, err
}

func SearchAllLogs(keyword string) (logs []*Log, err error) {
	err = LOG_DB.Where("type = ? or content LIKE ?", keyword, keyword+"%").Order("id desc").Limit(config.MaxRecentItems).Find(&logs).Error
	return logs, err
}

func SearchUserLogs(userId int, keyword string) (logs []*Log, err error) {
	err = LOG_DB.Where("user_id = ? and type = ?", userId, keyword).Order("id desc").Limit(config.MaxRecentItems).Omit("id").Find(&logs).Error
	return logs, err
}

func SumUsedQuota(logType int, startTimestamp int64, endTimestamp int64, modelName string, username string, tokenName string, channel int) (quota int64) {
	ifnull := "ifnull"
	if common.UsingPostgreSQL {
		ifnull = "COALESCE"
	}
	tx := LOG_DB.Table("logs").Select(fmt.Sprintf("%s(sum(quota),0)", ifnull))
	if username != "" {
		tx = tx.Where("username = ?", username)
	}
	if tokenName != "" {
		tx = tx.Where("token_name = ?", tokenName)
	}
	if startTimestamp != 0 {
		tx = tx.Where("created_at >= ?", startTimestamp)
	}
	if endTimestamp != 0 {
		tx = tx.Where("created_at <= ?", endTimestamp)
	}
	if modelName != "" {
		tx = tx.Where("model_name = ?", modelName)
	}
	if channel != 0 {
		tx = tx.Where("channel_id = ?", channel)
	}
	tx.Where("type = ?", LogTypeConsume).Scan(&quota)
	return quota
}

func SumUsedToken(logType int, startTimestamp int64, endTimestamp int64, modelName string, username string, tokenName string) (token int) {
	ifnull := "ifnull"
	if common.UsingPostgreSQL {
		ifnull = "COALESCE"
	}
	tx := LOG_DB.Table("logs").Select(fmt.Sprintf("%s(sum(prompt_tokens),0) + %s(sum(completion_tokens),0)", ifnull, ifnull))
	if username != "" {
		tx = tx.Where("username = ?", username)
	}
	if tokenName != "" {
		tx = tx.Where("token_name = ?", tokenName)
	}
	if startTimestamp != 0 {
		tx = tx.Where("created_at >= ?", startTimestamp)
	}
	if endTimestamp != 0 {
		tx = tx.Where("created_at <= ?", endTimestamp)
	}
	if modelName != "" {
		tx = tx.Where("model_name = ?", modelName)
	}
	tx.Where("type = ?", LogTypeConsume).Scan(&token)
	return token
}

func DeleteOldLog(targetTimestamp int64) (int64, error) {
	result := LOG_DB.Where("created_at < ?", targetTimestamp).Delete(&Log{})
	return result.RowsAffected, result.Error
}

type LogStatistic struct {
	Day              string `gorm:"column:day"`
	ModelName        string `gorm:"column:model_name"`
	RequestCount     int    `gorm:"column:request_count"`
	Quota            int    `gorm:"column:quota"`
	PromptTokens     int    `gorm:"column:prompt_tokens"`
	CompletionTokens int    `gorm:"column:completion_tokens"`
}

func SearchLogsByDayAndModel(userId, start, end int) (LogStatistics []*LogStatistic, err error) {
	groupSelect := "DATE_FORMAT(FROM_UNIXTIME(created_at), '%Y-%m-%d') as day"

	if common.UsingPostgreSQL {
		groupSelect = "TO_CHAR(date_trunc('day', to_timestamp(created_at)), 'YYYY-MM-DD') as day"
	}

	if common.UsingSQLite {
		groupSelect = "strftime('%Y-%m-%d', datetime(created_at, 'unixepoch')) as day"
	}

	err = LOG_DB.Raw(`
		SELECT `+groupSelect+`,
		model_name, count(1) as request_count,
		sum(quota) as quota,
		sum(prompt_tokens) as prompt_tokens,
		sum(completion_tokens) as completion_tokens
		FROM logs
		WHERE type=2
		AND user_id= ?
		AND created_at BETWEEN ? AND ?
		GROUP BY day, model_name
		ORDER BY day, model_name
	`, userId, start, end).Scan(&LogStatistics).Error

	return LogStatistics, err
}

// TokenUsageStatistic represents usage statistics by token
type TokenUsageStatistic struct {
	TokenId          int    `gorm:"column:token_id" json:"token_id"`
	TokenName        string `gorm:"column:token_name" json:"token_name"`
	RequestCount     int    `gorm:"column:request_count" json:"request_count"`
	Quota            int    `gorm:"column:quota" json:"quota"`
	PromptTokens     int    `gorm:"column:prompt_tokens" json:"prompt_tokens"`
	CompletionTokens int    `gorm:"column:completion_tokens" json:"completion_tokens"`
}

// ModelUsageStatistic represents usage statistics by model
type ModelUsageStatistic struct {
	ModelName        string `gorm:"column:model_name" json:"model_name"`
	RequestCount     int    `gorm:"column:request_count" json:"request_count"`
	Quota            int    `gorm:"column:quota" json:"quota"`
	PromptTokens     int    `gorm:"column:prompt_tokens" json:"prompt_tokens"`
	CompletionTokens int    `gorm:"column:completion_tokens" json:"completion_tokens"`
}

// ChannelUsageStatistic represents usage statistics by channel
type ChannelUsageStatistic struct {
	ChannelId        int    `gorm:"column:channel_id" json:"channel_id"`
	ChannelName      string `gorm:"column:channel_name" json:"channel_name"`
	RequestCount     int    `gorm:"column:request_count" json:"request_count"`
	Quota            int    `gorm:"column:quota" json:"quota"`
	PromptTokens     int    `gorm:"column:prompt_tokens" json:"prompt_tokens"`
	CompletionTokens int    `gorm:"column:completion_tokens" json:"completion_tokens"`
}

// GetTokenUsageStatistics returns usage statistics grouped by token
func GetTokenUsageStatistics(userId int, startTimestamp, endTimestamp int64) ([]*TokenUsageStatistic, error) {
	var stats []*TokenUsageStatistic

	query := LOG_DB.Table("logs").
		Select(`
			token_name,
			COUNT(1) as request_count,
			SUM(quota) as quota,
			SUM(prompt_tokens) as prompt_tokens,
			SUM(completion_tokens) as completion_tokens
		`).
		Where("type = ?", LogTypeConsume).
		Group("token_name").
		Order("request_count DESC")

	if userId > 0 {
		query = query.Where("user_id = ?", userId)
	}
	if startTimestamp > 0 {
		query = query.Where("created_at >= ?", startTimestamp)
	}
	if endTimestamp > 0 {
		query = query.Where("created_at <= ?", endTimestamp)
	}

	err := query.Scan(&stats).Error
	return stats, err
}

// GetModelUsageStatistics returns usage statistics grouped by model
func GetModelUsageStatistics(userId int, startTimestamp, endTimestamp int64) ([]*ModelUsageStatistic, error) {
	var stats []*ModelUsageStatistic

	query := LOG_DB.Table("logs").
		Select(`
			model_name,
			COUNT(1) as request_count,
			SUM(quota) as quota,
			SUM(prompt_tokens) as prompt_tokens,
			SUM(completion_tokens) as completion_tokens
		`).
		Where("type = ?", LogTypeConsume).
		Where("model_name != ''").
		Group("model_name").
		Order("request_count DESC")

	if userId > 0 {
		query = query.Where("user_id = ?", userId)
	}
	if startTimestamp > 0 {
		query = query.Where("created_at >= ?", startTimestamp)
	}
	if endTimestamp > 0 {
		query = query.Where("created_at <= ?", endTimestamp)
	}

	err := query.Scan(&stats).Error
	return stats, err
}

// GetChannelUsageStatistics returns usage statistics grouped by channel
func GetChannelUsageStatistics(userId int, startTimestamp, endTimestamp int64) ([]*ChannelUsageStatistic, error) {
	var stats []*ChannelUsageStatistic

	query := LOG_DB.Table("logs").
		Select(`
			channel_id,
			COUNT(1) as request_count,
			SUM(quota) as quota,
			SUM(prompt_tokens) as prompt_tokens,
			SUM(completion_tokens) as completion_tokens
		`).
		Where("type = ?", LogTypeConsume).
		Where("channel_id > 0").
		Group("channel_id").
		Order("request_count DESC")

	if userId > 0 {
		query = query.Where("user_id = ?", userId)
	}
	if startTimestamp > 0 {
		query = query.Where("created_at >= ?", startTimestamp)
	}
	if endTimestamp > 0 {
		query = query.Where("created_at <= ?", endTimestamp)
	}

	err := query.Scan(&stats).Error
	return stats, err
}

// HourlyUsageStatistic represents usage statistics by hour
type HourlyUsageStatistic struct {
	Hour             string `gorm:"column:hour" json:"hour"`
	RequestCount     int    `gorm:"column:request_count" json:"request_count"`
	Quota            int    `gorm:"column:quota" json:"quota"`
	PromptTokens     int    `gorm:"column:prompt_tokens" json:"prompt_tokens"`
	CompletionTokens int    `gorm:"column:completion_tokens" json:"completion_tokens"`
}

// GetHourlyUsageStatistics returns usage statistics grouped by hour
func GetHourlyUsageStatistics(userId int, startTimestamp, endTimestamp int64) ([]*HourlyUsageStatistic, error) {
	var stats []*HourlyUsageStatistic

	hourSelect := "DATE_FORMAT(FROM_UNIXTIME(created_at), '%Y-%m-%d %H:00') as hour"
	if common.UsingPostgreSQL {
		hourSelect = "TO_CHAR(date_trunc('hour', to_timestamp(created_at)), 'YYYY-MM-DD HH24:00') as hour"
	}
	if common.UsingSQLite {
		hourSelect = "strftime('%Y-%m-%d %H:00', datetime(created_at, 'unixepoch')) as hour"
	}

	query := LOG_DB.Raw(`
		SELECT `+hourSelect+`,
		COUNT(1) as request_count,
		SUM(quota) as quota,
		SUM(prompt_tokens) as prompt_tokens,
		SUM(completion_tokens) as completion_tokens
		FROM logs
		WHERE type = ?
	`, LogTypeConsume)

	if userId > 0 {
		query = LOG_DB.Raw(`
			SELECT `+hourSelect+`,
			COUNT(1) as request_count,
			SUM(quota) as quota,
			SUM(prompt_tokens) as prompt_tokens,
			SUM(completion_tokens) as completion_tokens
			FROM logs
			WHERE type = ? AND user_id = ?
		`, LogTypeConsume, userId)
	}

	if startTimestamp > 0 || endTimestamp > 0 {
		var args []interface{}
		sql := `
			SELECT `+hourSelect+`,
			COUNT(1) as request_count,
			SUM(quota) as quota,
			SUM(prompt_tokens) as prompt_tokens,
			SUM(completion_tokens) as completion_tokens
			FROM logs
			WHERE type = ?
		`
		args = append(args, LogTypeConsume)

		if userId > 0 {
			sql += " AND user_id = ?"
			args = append(args, userId)
		}
		if startTimestamp > 0 {
			sql += " AND created_at >= ?"
			args = append(args, startTimestamp)
		}
		if endTimestamp > 0 {
			sql += " AND created_at <= ?"
			args = append(args, endTimestamp)
		}
		sql += " GROUP BY hour ORDER BY hour"
		query = LOG_DB.Raw(sql, args...)
	} else {
		sql := `
			SELECT `+hourSelect+`,
			COUNT(1) as request_count,
			SUM(quota) as quota,
			SUM(prompt_tokens) as prompt_tokens,
			SUM(completion_tokens) as completion_tokens
			FROM logs
			WHERE type = ?
		`
		if userId > 0 {
			sql += " AND user_id = ?"
			query = LOG_DB.Raw(sql+" GROUP BY hour ORDER BY hour", LogTypeConsume, userId)
		} else {
			query = LOG_DB.Raw(sql+" GROUP BY hour ORDER BY hour", LogTypeConsume)
		}
	}

	err := query.Scan(&stats).Error
	return stats, err
}

// GetUserUsageSummary returns a summary of usage for a user
func GetUserUsageSummary(userId int, startTimestamp, endTimestamp int64) (map[string]interface{}, error) {
	var totalQuota int64
	var totalPromptTokens int64
	var totalCompletionTokens int64
	var totalRequests int64

	query := LOG_DB.Table("logs").
		Where("type = ?", LogTypeConsume)

	if userId > 0 {
		query = query.Where("user_id = ?", userId)
	}
	if startTimestamp > 0 {
		query = query.Where("created_at >= ?", startTimestamp)
	}
	if endTimestamp > 0 {
		query = query.Where("created_at <= ?", endTimestamp)
	}

	err := query.Select(`
		COUNT(1) as total_requests,
		COALESCE(SUM(quota), 0) as total_quota,
		COALESCE(SUM(prompt_tokens), 0) as total_prompt_tokens,
		COALESCE(SUM(completion_tokens), 0) as total_completion_tokens
	`).Scan(map[string]interface{}{
		"total_requests":          &totalRequests,
		"total_quota":             &totalQuota,
		"total_prompt_tokens":     &totalPromptTokens,
		"total_completion_tokens": &totalCompletionTokens,
	}).Error

	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"total_requests":          totalRequests,
		"total_quota":             totalQuota,
		"total_prompt_tokens":     totalPromptTokens,
		"total_completion_tokens": totalCompletionTokens,
		"total_tokens":            totalPromptTokens + totalCompletionTokens,
	}, nil
}
