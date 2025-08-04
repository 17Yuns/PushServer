package pusher

import (
	"fmt"
	"sync"
	"time"

	"PushServer/internal/config"
	"PushServer/internal/logger"
	"PushServer/internal/model"
	"PushServer/internal/platform"
	"PushServer/internal/task"
)

// PushService 推送服务
type PushService struct {
	platformManager *platform.PlatformManager
}

// NewPushService 创建推送服务
func NewPushService() *PushService {
	return &PushService{
		platformManager: platform.NewPlatformManager(),
	}
}

// ExecuteStrategy 执行推送策略
func (ps *PushService) ExecuteStrategy(taskID string, req model.PushRequest, recipient config.RecipientConfig) {
	// 如果指定了平台，直接忽略策略，只在该平台内推送直到成功
	if req.Platform != "" {
		logger.Infof("指定平台推送: %s, 任务ID: %s (忽略策略: %s)", req.Platform, taskID, req.Strategy)
		ps.executePlatformOnlyStrategy(taskID, req, recipient)
		return
	}

	logger.Infof("开始执行推送策略: %s, 任务ID: %s", req.Strategy, taskID)

	switch req.Strategy {
	case model.StrategyAll:
		ps.executeAllStrategy(taskID, req, recipient)
	case model.StrategyFailover:
		ps.executeFailoverStrategy(taskID, req, recipient)
	case model.StrategyWebhookFailover:
		ps.executeWebhookFailoverStrategy(taskID, req, recipient)
	case model.StrategyMixed:
		ps.executeMixedStrategy(taskID, req, recipient)
	default:
		task.Manager.SetTaskError(taskID, "不支持的推送策略: "+req.Strategy)
	}
}

// executePlatformOnlyStrategy 执行指定平台推送：忽略策略，只在指定平台内推送直到成功
func (ps *PushService) executePlatformOnlyStrategy(taskID string, req model.PushRequest, recipient config.RecipientConfig) {
	logger.Infof("执行指定平台推送: %s，只要有一个地址成功即可", req.Platform)

	platformConfig, exists := recipient.Platforms[req.Platform]
	if !exists {
		task.Manager.SetTaskError(taskID, "指定的平台不存在: "+req.Platform)
		return
	}

	if !platformConfig.Enabled {
		task.Manager.SetTaskError(taskID, "指定的平台未启用: "+req.Platform)
		return
	}

	// 根据平台类型处理不同的配置
	if req.Platform == "email" {
		// 邮件平台使用recipients配置
		for _, recipient := range platformConfig.Recipients {
			webhook := config.WebhookConfig{
				URL:    recipient.Email,
				Secret: "",
				Name:   recipient.Name,
			}
			result := ps.sendToWebhook(req.Platform, webhook, req)
			task.Manager.AddResult(taskID, result)
			logger.Infof("指定平台推送结果: %s-%s: %s", req.Platform, recipient.Name, result.Status)

			// 只要有一个成功就停止
			if result.Status == "success" {
				logger.Infof("指定平台 %s 推送成功，任务完成，任务ID: %s", req.Platform, taskID)
				return
			}
		}
	} else if req.Platform == "system" {
		// 系统通知平台使用notifications配置
		for _, notification := range platformConfig.Notifications {
			webhook := config.WebhookConfig{
				URL:    notification.Type,
				Secret: "",
				Name:   notification.Name,
			}
			result := ps.sendToWebhook(req.Platform, webhook, req)
			task.Manager.AddResult(taskID, result)
			logger.Infof("指定平台推送结果: %s-%s: %s", req.Platform, notification.Name, result.Status)

			// 只要有一个成功就停止
			if result.Status == "success" {
				logger.Infof("指定平台 %s 推送成功，任务完成，任务ID: %s", req.Platform, taskID)
				return
			}
		}
	} else {
		// 其他平台使用webhooks配置
		for _, webhook := range platformConfig.Webhooks {
			result := ps.sendToWebhook(req.Platform, webhook, req)
			task.Manager.AddResult(taskID, result)
			logger.Infof("指定平台推送结果: %s-%s: %s", req.Platform, webhook.Name, result.Status)

			// 只要有一个成功就停止
			if result.Status == "success" {
				logger.Infof("指定平台 %s 推送成功，任务完成，任务ID: %s", req.Platform, taskID)
				return
			}
		}
	}

	logger.Warnf("指定平台 %s 所有地址都推送失败，任务ID: %s", req.Platform, taskID)

	// 触发系统通知作为最后防线
	ps.triggerSystemNotification(taskID, req, fmt.Sprintf("指定平台 %s 推送失败", req.Platform))
}

// executeAllStrategy 执行all策略：所有渠道都发送
func (ps *PushService) executeAllStrategy(taskID string, req model.PushRequest, recipient config.RecipientConfig) {
	logger.Infof("执行all策略：向所有启用的渠道发送消息")

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, config.AppConfig.Queue.MaxConcurrentPerPlatform)

	for platformName, platformConfig := range recipient.Platforms {
		if !platformConfig.Enabled {
			logger.Debugf("平台 %s 未启用，跳过", platformName)
			continue
		}

		// 根据平台类型处理不同的配置
		if platformName == "email" {
			// 邮件平台使用recipients配置
			for _, recipient := range platformConfig.Recipients {
				wg.Add(1)
				go func(pName string, rec config.EmailRecipientConfig) {
					defer wg.Done()
					semaphore <- struct{}{}        // 获取信号量
					defer func() { <-semaphore }() // 释放信号量

					// 将邮件收件人转换为webhook格式以兼容现有接口
					webhook := config.WebhookConfig{
						URL:    rec.Email,
						Secret: "",
						Name:   rec.Name,
					}
					result := ps.sendToWebhook(pName, webhook, req)
					task.Manager.AddResult(taskID, result)
					logger.Infof("all策略推送结果: %s-%s: %s", pName, rec.Name, result.Status)
				}(platformName, recipient)
			}
		} else if platformName == "system" {
			// 系统通知平台使用notifications配置
			for _, notification := range platformConfig.Notifications {
				wg.Add(1)
				go func(pName string, notif config.SystemNotificationConfig) {
					defer wg.Done()
					semaphore <- struct{}{}        // 获取信号量
					defer func() { <-semaphore }() // 释放信号量

					webhook := config.WebhookConfig{
						URL:    notif.Type,
						Secret: "",
						Name:   notif.Name,
					}
					result := ps.sendToWebhook(pName, webhook, req)
					task.Manager.AddResult(taskID, result)
					logger.Infof("all策略推送结果: %s-%s: %s", pName, notif.Name, result.Status)
				}(platformName, notification)
			}
		} else {
			// 其他平台使用webhooks配置
			for _, webhook := range platformConfig.Webhooks {
				wg.Add(1)
				go func(pName string, wh config.WebhookConfig) {
					defer wg.Done()
					semaphore <- struct{}{}        // 获取信号量
					defer func() { <-semaphore }() // 释放信号量

					result := ps.sendToWebhook(pName, wh, req)
					task.Manager.AddResult(taskID, result)
					logger.Infof("all策略推送结果: %s-%s: %s", pName, wh.Name, result.Status)
				}(platformName, webhook)
			}
		}
	}

	wg.Wait()
	logger.Infof("all策略执行完成，任务ID: %s", taskID)

	// 检查是否有成功的推送，如果全部失败则触发系统通知
	ps.checkAndTriggerSystemNotification(taskID, req, "all策略所有渠道推送失败")
}

// executeFailoverStrategy 执行failover策略：渠道间故障转移
func (ps *PushService) executeFailoverStrategy(taskID string, req model.PushRequest, recipient config.RecipientConfig) {
	logger.Infof("执行failover策略：渠道间故障转移")

	for platformName, platformConfig := range recipient.Platforms {
		if !platformConfig.Enabled {
			logger.Debugf("平台 %s 未启用，跳过", platformName)
			continue
		}

		// 根据平台类型处理不同的配置
		if platformName == "email" {
			// 邮件平台使用recipients配置
			if len(platformConfig.Recipients) > 0 {
				recipient := platformConfig.Recipients[0]
				webhook := config.WebhookConfig{
					URL:    recipient.Email,
					Secret: "",
					Name:   recipient.Name,
				}
				result := ps.sendToWebhook(platformName, webhook, req)
				task.Manager.AddResult(taskID, result)
				logger.Infof("failover策略推送结果: %s-%s: %s", platformName, recipient.Name, result.Status)

				// 如果成功，停止尝试其他平台
				if result.Status == "success" {
					logger.Infof("failover策略成功，停止尝试其他平台，任务ID: %s", taskID)
					return
				}
			}
		} else {
			// 其他平台使用webhooks配置
			if len(platformConfig.Webhooks) > 0 {
				webhook := platformConfig.Webhooks[0]
				result := ps.sendToWebhook(platformName, webhook, req)
				task.Manager.AddResult(taskID, result)
				logger.Infof("failover策略推送结果: %s-%s: %s", platformName, webhook.Name, result.Status)

				// 如果成功，停止尝试其他平台
				if result.Status == "success" {
					logger.Infof("failover策略成功，停止尝试其他平台，任务ID: %s", taskID)
					return
				}
			}
		}
	}

	logger.Infof("failover策略执行完成，任务ID: %s", taskID)

	// 检查是否有成功的推送，如果全部失败则触发系统通知
	ps.checkAndTriggerSystemNotification(taskID, req, "failover策略所有渠道推送失败")
}

// executeWebhookFailoverStrategy 执行webhook_failover策略：每个渠道内webhook故障转移
func (ps *PushService) executeWebhookFailoverStrategy(taskID string, req model.PushRequest, recipient config.RecipientConfig) {
	logger.Infof("执行webhook_failover策略：每个渠道内webhook故障转移")

	for platformName, platformConfig := range recipient.Platforms {
		if !platformConfig.Enabled {
			logger.Debugf("平台 %s 未启用，跳过", platformName)
			continue
		}

		// 根据平台类型处理不同的配置
		platformSuccess := false
		if platformName == "email" {
			// 邮件平台使用recipients配置
			for _, recipient := range platformConfig.Recipients {
				webhook := config.WebhookConfig{
					URL:    recipient.Email,
					Secret: "",
					Name:   recipient.Name,
				}
				result := ps.sendToWebhook(platformName, webhook, req)
				task.Manager.AddResult(taskID, result)
				logger.Infof("webhook_failover策略推送结果: %s-%s: %s", platformName, recipient.Name, result.Status)

				// 如果成功，停止尝试当前平台的其他收件人
				if result.Status == "success" {
					platformSuccess = true
					logger.Infof("平台 %s webhook故障转移成功", platformName)
					break
				}
			}
		} else {
			// 其他平台使用webhooks配置
			for _, webhook := range platformConfig.Webhooks {
				result := ps.sendToWebhook(platformName, webhook, req)
				task.Manager.AddResult(taskID, result)
				logger.Infof("webhook_failover策略推送结果: %s-%s: %s", platformName, webhook.Name, result.Status)

				// 如果成功，停止尝试当前平台的其他webhook
				if result.Status == "success" {
					platformSuccess = true
					logger.Infof("平台 %s webhook故障转移成功", platformName)
					break
				}
			}
		}

		if !platformSuccess {
			logger.Warnf("平台 %s 所有webhook都失败", platformName)
		}
	}

	logger.Infof("webhook_failover策略执行完成，任务ID: %s", taskID)

	// 检查是否有成功的推送，如果全部失败则触发系统通知
	ps.checkAndTriggerSystemNotification(taskID, req, "webhook_failover策略所有渠道推送失败")
}

// executeMixedStrategy 执行mixed策略：渠道间故障转移，渠道内webhook全发送
func (ps *PushService) executeMixedStrategy(taskID string, req model.PushRequest, recipient config.RecipientConfig) {
	logger.Infof("执行mixed策略：渠道间故障转移，渠道内webhook全发送")

	for platformName, platformConfig := range recipient.Platforms {
		if !platformConfig.Enabled {
			logger.Debugf("平台 %s 未启用，跳过", platformName)
			continue
		}

		platformSuccess := false
		var wg sync.WaitGroup
		semaphore := make(chan struct{}, config.AppConfig.Queue.MaxConcurrentPerPlatform)

		// 根据平台类型处理不同的配置
		if platformName == "email" {
			// 邮件平台使用recipients配置
			successChan := make(chan bool, len(platformConfig.Recipients))

			for _, recipient := range platformConfig.Recipients {
				wg.Add(1)
				go func(rec config.EmailRecipientConfig) {
					defer wg.Done()
					semaphore <- struct{}{}        // 获取信号量
					defer func() { <-semaphore }() // 释放信号量

					webhook := config.WebhookConfig{
						URL:    rec.Email,
						Secret: "",
						Name:   rec.Name,
					}
					result := ps.sendToWebhook(platformName, webhook, req)
					task.Manager.AddResult(taskID, result)
					logger.Infof("mixed策略推送结果: %s-%s: %s", platformName, rec.Name, result.Status)

					if result.Status == "success" {
						select {
						case successChan <- true:
						default:
						}
					}
				}(recipient)
			}

			wg.Wait()
			close(successChan)

			// 检查是否有成功的
			select {
			case <-successChan:
				platformSuccess = true
			default:
			}
		} else {
			// 其他平台使用webhooks配置
			successChan := make(chan bool, len(platformConfig.Webhooks))

			for _, webhook := range platformConfig.Webhooks {
				wg.Add(1)
				go func(wh config.WebhookConfig) {
					defer wg.Done()
					semaphore <- struct{}{}        // 获取信号量
					defer func() { <-semaphore }() // 释放信号量

					result := ps.sendToWebhook(platformName, wh, req)
					task.Manager.AddResult(taskID, result)
					logger.Infof("mixed策略推送结果: %s-%s: %s", platformName, wh.Name, result.Status)

					if result.Status == "success" {
						select {
						case successChan <- true:
						default:
						}
					}
				}(webhook)
			}

			wg.Wait()
			close(successChan)

			// 检查是否有成功的
			select {
			case <-successChan:
				platformSuccess = true
			default:
			}
		}

		// 如果当前平台有成功的，停止尝试其他平台
		if platformSuccess {
			logger.Infof("mixed策略平台 %s 成功，停止尝试其他平台，任务ID: %s", platformName, taskID)
			return
		}

		logger.Warnf("mixed策略平台 %s 所有webhook都失败，尝试下一个平台", platformName)
	}

	logger.Infof("mixed策略执行完成，任务ID: %s", taskID)

	// 检查是否有成功的推送，如果全部失败则触发系统通知
	ps.checkAndTriggerSystemNotification(taskID, req, "mixed策略所有渠道推送失败")
}

// sendToWebhook 发送到webhook
func (ps *PushService) sendToWebhook(platformName string, webhook config.WebhookConfig, req model.PushRequest) task.PushResult {
	logger.Infof("开始发送消息到 %s - %s: %s", platformName, webhook.Name, req.Content.Title)

	// 根据平台选择对应的转发服务
	var result platform.PlatformResult
	switch platformName {
	case "feishu":
		result = ps.platformManager.ForwardToFeishu(webhook, req)
	case "dingtalk":
		result = ps.platformManager.ForwardToDingtalk(webhook, req)
	case "wechat":
		result = ps.platformManager.ForwardToWorkWechat(webhook, req)
	case "email":
		result = ps.platformManager.ForwardToEmail(webhook, req)
	case "system":
		result = ps.platformManager.ForwardToSystem(webhook, req)
	default:
		result = platform.PlatformResult{
			Platform:  platformName,
			Webhook:   webhook.Name,
			Status:    "failed",
			Message:   "不支持的平台: " + platformName,
			Timestamp: time.Now(),
		}
	}

	// 转换为任务结果格式
	taskResult := task.PushResult{
		Platform:  result.Platform,
		Webhook:   result.Webhook,
		Status:    result.Status,
		Message:   result.Message,
		Timestamp: result.Timestamp,
	}

	return taskResult
}

// checkAndTriggerSystemNotification 检查推送结果并触发系统通知
func (ps *PushService) checkAndTriggerSystemNotification(taskID string, req model.PushRequest, reason string) {
	// 获取任务结果
	taskInfo, exists := task.Manager.GetTask(taskID)
	if !exists || taskInfo == nil {
		logger.Warnf("无法获取任务信息，跳过系统通知检查，任务ID: %s", taskID)
		return
	}

	// 检查是否有成功的推送
	hasSuccess := false
	for _, result := range taskInfo.Results {
		if result.Status == "success" {
			hasSuccess = true
			break
		}
	}

	// 如果没有成功的推送，触发系统通知
	if !hasSuccess {
		logger.Warnf("所有推送都失败，触发系统通知: %s", reason)
		ps.triggerSystemNotification(taskID, req, reason)
	}
}

// triggerSystemNotification 触发系统通知
func (ps *PushService) triggerSystemNotification(taskID string, req model.PushRequest, reason string) {
	logger.Infof("触发系统通知作为最后防线，任务ID: %s, 原因: %s", taskID, reason)

	// 检查是否启用了系统通知
	if !config.AppConfig.System.Enabled {
		logger.Infof("系统通知未启用，跳过系统通知")
		return
	}

	// 构建系统通知内容
	systemReq := model.PushRequest{
		Type:     model.TypeError, // 系统通知默认为错误类型
		Style:    model.StyleText,
		Strategy: "system",
		Platform: "system",
		Content: model.MessageContent{
			Title: fmt.Sprintf("🚨 推送系统故障通知 - %s", reason),
			Msg: fmt.Sprintf(`原始消息推送失败，触发系统通知：

原始标题: %s
原始内容: %s
失败原因: %s
任务ID: %s
失败时间: %s

请检查推送服务配置和网络连接状态。`,
				req.Content.Title,
				req.Content.Msg,
				reason,
				taskID,
				time.Now().Format("2006-01-02 15:04:05"),
			),
		},
	}

	// 发送系统通知到所有配置的系统通知方式
	for _, notifyConfig := range config.AppConfig.System.Notifications {
		webhook := config.WebhookConfig{
			URL:    notifyConfig.Type,
			Secret: "",
			Name:   notifyConfig.Name,
		}

		result := ps.platformManager.ForwardToSystem(webhook, systemReq)
		task.Manager.AddResult(taskID, task.PushResult{
			Platform:  result.Platform,
			Webhook:   result.Webhook,
			Status:    result.Status,
			Message:   result.Message,
			Timestamp: result.Timestamp,
		})

		logger.Infof("系统通知发送结果: %s-%s: %s", result.Platform, result.Webhook, result.Status)
	}
}
