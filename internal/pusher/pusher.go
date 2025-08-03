package pusher

import (
	"sync"
	"time"

	"PushServer/internal/config"
	"PushServer/internal/forwarder"
	"PushServer/internal/logger"
	"PushServer/internal/model"
	"PushServer/internal/task"
)

// PushService 推送服务
type PushService struct {
	forwardService *forwarder.ForwardService
}

// NewPushService 创建推送服务
func NewPushService() *PushService {
	return &PushService{
		forwardService: forwarder.NewForwardService(),
	}
}

// ExecuteStrategy 执行推送策略
func (ps *PushService) ExecuteStrategy(taskID string, req model.PushRequest, recipient config.RecipientConfig) {
	logger.Infof("开始执行推送策略: %s, 任务ID: %s", req.Strategy, taskID)
	
	switch req.Strategy {
	case model.StrategyAll:
		ps.executeAllStrategy(taskID, req, recipient)
	case model.StrategyFailover:
		ps.executeFailoverStrategy(taskID, req, recipient)
	case model.StrategyWebhookAll:
		ps.executeWebhookAllStrategy(taskID, req, recipient)
	case model.StrategyWebhookFailover:
		ps.executeWebhookFailoverStrategy(taskID, req, recipient)
	case model.StrategyMixed:
		ps.executeMixedStrategy(taskID, req, recipient)
	default:
		task.Manager.SetTaskError(taskID, "不支持的推送策略: "+req.Strategy)
	}
}

// executeAllStrategy 执行all策略：所有渠道都发送
func (ps *PushService) executeAllStrategy(taskID string, req model.PushRequest, recipient config.RecipientConfig) {
	logger.Infof("执行all策略：向所有启用的渠道发送消息")
	
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, config.AppConfig.Queue.MaxConcurrentPerPlatform)
	
	for platformName, platform := range recipient.Platforms {
		if !platform.Enabled {
			logger.Debugf("平台 %s 未启用，跳过", platformName)
			continue
		}
		
		// 如果指定了平台，只推送到该平台
		if req.Platform != "" && req.Platform != platformName {
			logger.Debugf("指定平台 %s，跳过平台 %s", req.Platform, platformName)
			continue
		}
		
		// 向该平台的所有webhook发送
		for _, webhook := range platform.Webhooks {
			wg.Add(1)
			go func(pName string, wh config.WebhookConfig) {
				defer wg.Done()
				semaphore <- struct{}{} // 获取信号量
				defer func() { <-semaphore }() // 释放信号量
				
				result := ps.sendToWebhook(pName, wh, req)
				task.Manager.AddResult(taskID, result)
				logger.Infof("all策略推送结果: %s-%s: %s", pName, wh.Name, result.Status)
			}(platformName, webhook)
		}
	}
	
	wg.Wait()
	logger.Infof("all策略执行完成，任务ID: %s", taskID)
}

// executeFailoverStrategy 执行failover策略：渠道间故障转移
func (ps *PushService) executeFailoverStrategy(taskID string, req model.PushRequest, recipient config.RecipientConfig) {
	logger.Infof("执行failover策略：渠道间故障转移")
	
	for platformName, platform := range recipient.Platforms {
		if !platform.Enabled {
			logger.Debugf("平台 %s 未启用，跳过", platformName)
			continue
		}
		
		// 如果指定了平台，只推送到该平台
		if req.Platform != "" && req.Platform != platformName {
			logger.Debugf("指定平台 %s，跳过平台 %s", req.Platform, platformName)
			continue
		}
		
		// 尝试该平台的第一个webhook
		if len(platform.Webhooks) > 0 {
			webhook := platform.Webhooks[0]
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
	
	logger.Infof("failover策略执行完成，任务ID: %s", taskID)
}

// executeWebhookAllStrategy 执行webhook_all策略：每个渠道内所有webhook都发送
func (ps *PushService) executeWebhookAllStrategy(taskID string, req model.PushRequest, recipient config.RecipientConfig) {
	logger.Infof("执行webhook_all策略：每个渠道内所有webhook都发送")
	
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, config.AppConfig.Queue.MaxConcurrentPerPlatform)
	
	for platformName, platform := range recipient.Platforms {
		if !platform.Enabled {
			logger.Debugf("平台 %s 未启用，跳过", platformName)
			continue
		}
		
		// 如果指定了平台，只推送到该平台
		if req.Platform != "" && req.Platform != platformName {
			logger.Debugf("指定平台 %s，跳过平台 %s", req.Platform, platformName)
			continue
		}
		
		// 向该平台的所有webhook发送
		for _, webhook := range platform.Webhooks {
			wg.Add(1)
			go func(pName string, wh config.WebhookConfig) {
				defer wg.Done()
				semaphore <- struct{}{} // 获取信号量
				defer func() { <-semaphore }() // 释放信号量
				
				result := ps.sendToWebhook(pName, wh, req)
				task.Manager.AddResult(taskID, result)
				logger.Infof("webhook_all策略推送结果: %s-%s: %s", pName, wh.Name, result.Status)
			}(platformName, webhook)
		}
	}
	
	wg.Wait()
	logger.Infof("webhook_all策略执行完成，任务ID: %s", taskID)
}

// executeWebhookFailoverStrategy 执行webhook_failover策略：每个渠道内webhook故障转移
func (ps *PushService) executeWebhookFailoverStrategy(taskID string, req model.PushRequest, recipient config.RecipientConfig) {
	logger.Infof("执行webhook_failover策略：每个渠道内webhook故障转移")
	
	for platformName, platform := range recipient.Platforms {
		if !platform.Enabled {
			logger.Debugf("平台 %s 未启用，跳过", platformName)
			continue
		}
		
		// 如果指定了平台，只推送到该平台
		if req.Platform != "" && req.Platform != platformName {
			logger.Debugf("指定平台 %s，跳过平台 %s", req.Platform, platformName)
			continue
		}
		
		// 在当前平台内尝试webhook故障转移
		platformSuccess := false
		for _, webhook := range platform.Webhooks {
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
		
		if !platformSuccess {
			logger.Warnf("平台 %s 所有webhook都失败", platformName)
		}
	}
	
	logger.Infof("webhook_failover策略执行完成，任务ID: %s", taskID)
}

// executeMixedStrategy 执行mixed策略：渠道间故障转移，渠道内webhook全发送
func (ps *PushService) executeMixedStrategy(taskID string, req model.PushRequest, recipient config.RecipientConfig) {
	logger.Infof("执行mixed策略：渠道间故障转移，渠道内webhook全发送")
	
	for platformName, platform := range recipient.Platforms {
		if !platform.Enabled {
			logger.Debugf("平台 %s 未启用，跳过", platformName)
			continue
		}
		
		// 如果指定了平台，只推送到该平台
		if req.Platform != "" && req.Platform != platformName {
			logger.Debugf("指定平台 %s，跳过平台 %s", req.Platform, platformName)
			continue
		}
		
		platformSuccess := false
		var wg sync.WaitGroup
		semaphore := make(chan struct{}, config.AppConfig.Queue.MaxConcurrentPerPlatform)
		successChan := make(chan bool, len(platform.Webhooks))
		
		// 向当前平台的所有webhook并发发送
		for _, webhook := range platform.Webhooks {
			wg.Add(1)
			go func(wh config.WebhookConfig) {
				defer wg.Done()
				semaphore <- struct{}{} // 获取信号量
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
		
		// 如果当前平台有成功的，停止尝试其他平台
		if platformSuccess {
			logger.Infof("mixed策略平台 %s 成功，停止尝试其他平台，任务ID: %s", platformName, taskID)
			return
		}
		
		logger.Warnf("mixed策略平台 %s 所有webhook都失败，尝试下一个平台", platformName)
	}
	
	logger.Infof("mixed策略执行完成，任务ID: %s", taskID)
}

// sendToWebhook 发送到webhook
func (ps *PushService) sendToWebhook(platform string, webhook config.WebhookConfig, req model.PushRequest) task.PushResult {
	logger.Infof("开始发送消息到 %s - %s: %s", platform, webhook.Name, req.Content.Title)
	
	var forwardResult forwarder.ForwardResult
	
	// 根据平台选择对应的转发服务
	switch platform {
	case "feishu":
		forwardResult = ps.forwardService.ForwardToFeishu(webhook, req)
	case "dingtalk":
		forwardResult = ps.forwardService.ForwardToDingtalk(webhook, req)
	case "wechat":
		forwardResult = ps.forwardService.ForwardToWechat(webhook, req)
	default:
		logger.Errorf("不支持的平台: %s", platform)
		forwardResult = forwarder.ForwardResult{
			Platform:  platform,
			Webhook:   webhook.Name,
			Status:    "failed",
			Message:   "不支持的平台: " + platform,
			Timestamp: time.Now(),
		}
	}
	
	// 转换为任务结果格式
	result := task.PushResult{
		Platform:  forwardResult.Platform,
		Webhook:   forwardResult.Webhook,
		Status:    forwardResult.Status,
		Message:   forwardResult.Message,
		Timestamp: forwardResult.Timestamp,
	}
	
	return result
}

