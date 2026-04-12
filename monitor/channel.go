package monitor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/pagoda-inference/one-api/common/config"
	"github.com/pagoda-inference/one-api/common/logger"
	"github.com/pagoda-inference/one-api/common/message"
	"github.com/pagoda-inference/one-api/model"
)

// AlertConfig keys
const (
	AlertKeyChannelFailureThreshold = "AlertChannelFailureThreshold"
	AlertKeyQueueUtilizationAlert  = "AlertQueueUtilizationAlert"
	AlertKeyErrorRateAlert         = "AlertErrorRateAlert"
	AlertKeyLatencyThreshold       = "AlertLatencyThreshold"
	AlertKeyAlertEmail            = "AlertEmail"
	AlertKeyAlertWebhook          = "AlertWebhook"
	AlertKeyEnabled               = "AlertEnabled"
)

func notifyRootUser(subject string, content string) {
	// Read alert config from OptionMap
	config.OptionMapRWMutex.RLock()
	alertEmail := config.OptionMap[AlertKeyAlertEmail]
	alertWebhook := config.OptionMap[AlertKeyAlertWebhook]
	alertEnabled := config.OptionMap[AlertKeyEnabled]
	config.OptionMapRWMutex.RUnlock()

	// Check if alerts are enabled
	if alertEnabled != "true" {
		return
	}

	// Send webhook if configured
	if alertWebhook != "" {
		go sendAlertWebhook(alertWebhook, subject, content)
	}

	// Determine email recipient
	email := alertEmail
	if email == "" {
		// Fallback to root user email
		if config.RootUserEmail == "" {
			config.RootUserEmail = model.GetRootUserEmail()
		}
		email = config.RootUserEmail
	}

	if email == "" {
		logger.SysWarn("no alert email configured, skipping email notification")
		return
	}

	// Try message pusher first, then email
	if config.MessagePusherAddress != "" {
		err := message.SendMessage(subject, content, content)
		if err != nil {
			logger.SysError(fmt.Sprintf("failed to send message: %s", err.Error()))
			// Fallback to email
			err = message.SendEmail(subject, email, content)
			if err != nil {
				logger.SysError(fmt.Sprintf("failed to send email: %s", err.Error()))
			}
		}
	} else {
		// Send email directly
		err := message.SendEmail(subject, email, content)
		if err != nil {
			logger.SysError(fmt.Sprintf("failed to send email: %s", err.Error()))
		}
	}
}

// sendAlertWebhook sends an alert to the configured webhook URL
func sendAlertWebhook(webhookURL string, subject string, content string) {
	type webhookPayload struct {
		Subject string `json:"subject"`
		Content string `json:"content"`
	}
	payload := webhookPayload{
		Subject: subject,
		Content: content,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		logger.SysError(fmt.Sprintf("failed to marshal webhook payload: %s", err.Error()))
		return
	}
	resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(data))
	if err != nil {
		logger.SysError(fmt.Sprintf("failed to send webhook: %s", err.Error()))
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		logger.SysError(fmt.Sprintf("webhook returned non-success status: %d", resp.StatusCode))
	}
}

// DisableChannel disable & notify
func DisableChannel(channelId int, channelName string, reason string) {
	model.UpdateChannelStatusById(channelId, model.ChannelStatusAutoDisabled)
	logger.SysLog(fmt.Sprintf("channel #%d has been disabled: %s", channelId, reason))
	subject := fmt.Sprintf("渠道状态变更提醒")
	content := message.EmailTemplate(
		subject,
		fmt.Sprintf(`
			<p>您好！</p>
			<p>渠道「<strong>%s</strong>」（#%d）已被禁用。</p>
			<p>禁用原因：</p>
			<p style="background-color: #f8f8f8; padding: 10px; border-radius: 4px;">%s</p>
		`, channelName, channelId, reason),
	)
	notifyRootUser(subject, content)
}

func MetricDisableChannel(channelId int, successRate float64) {
	model.UpdateChannelStatusById(channelId, model.ChannelStatusAutoDisabled)
	logger.SysLog(fmt.Sprintf("channel #%d has been disabled due to low success rate: %.2f", channelId, successRate*100))
	subject := fmt.Sprintf("渠道状态变更提醒")
	content := message.EmailTemplate(
		subject,
		fmt.Sprintf(`
			<p>您好！</p>
			<p>渠道 #%d 已被系统自动禁用。</p>
			<p>禁用原因：</p>
			<p style="background-color: #f8f8f8; padding: 10px; border-radius: 4px;">该渠道在最近 %d 次调用中成功率为 <strong>%.2f%%</strong>，低于系统阈值 <strong>%.2f%%</strong>。</p>
		`, channelId, config.MetricQueueSize, successRate*100, config.MetricSuccessRateThreshold*100),
	)
	notifyRootUser(subject, content)
}

// EnableChannel enable & notify
func EnableChannel(channelId int, channelName string) {
	model.UpdateChannelStatusById(channelId, model.ChannelStatusEnabled)
	logger.SysLog(fmt.Sprintf("channel #%d has been enabled", channelId))
	subject := fmt.Sprintf("渠道状态变更提醒")
	content := message.EmailTemplate(
		subject,
		fmt.Sprintf(`
			<p>您好！</p>
			<p>渠道「<strong>%s</strong>」（#%d）已被重新启用。</p>
			<p>您现在可以继续使用该渠道了。</p>
		`, channelName, channelId),
	)
	notifyRootUser(subject, content)
}
