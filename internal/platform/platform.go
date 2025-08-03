package platform

import (
	"time"

	"PushServer/internal/config"
	"PushServer/internal/model"
)

// Platform 平台接口
type Platform interface {
	// Send 发送消息到平台
	Send(webhook config.WebhookConfig, req model.PushRequest) PlatformResult
	// GetName 获取平台名称
	GetName() string
}

// PlatformResult 平台推送结果
type PlatformResult struct {
	Platform  string    `json:"platform"`  // 平台名称
	Webhook   string    `json:"webhook"`   // Webhook名称
	Status    string    `json:"status"`    // 状态: success, failed
	Message   string    `json:"message"`   // 结果消息
	Timestamp time.Time `json:"timestamp"` // 时间戳
}

// PlatformManager 平台管理器
type PlatformManager struct {
	platforms map[string]Platform
}

// NewPlatformManager 创建平台管理器
func NewPlatformManager() *PlatformManager {
	manager := &PlatformManager{
		platforms: make(map[string]Platform),
	}

	// 注册所有平台
	manager.RegisterPlatform(NewFeishuPlatform())
	manager.RegisterPlatform(NewDingtalkPlatform())
	manager.RegisterPlatform(NewWechatPlatform())

	return manager
}

// RegisterPlatform 注册平台
func (pm *PlatformManager) RegisterPlatform(platform Platform) {
	pm.platforms[platform.GetName()] = platform
}

// GetPlatform 获取平台实例
func (pm *PlatformManager) GetPlatform(name string) (Platform, bool) {
	platform, exists := pm.platforms[name]
	return platform, exists
}

// GetAllPlatforms 获取所有平台
func (pm *PlatformManager) GetAllPlatforms() map[string]Platform {
	return pm.platforms
}

// Send 发送消息到指定平台
func (pm *PlatformManager) Send(platformName string, webhook config.WebhookConfig, req model.PushRequest) PlatformResult {
	platform, exists := pm.GetPlatform(platformName)
	if !exists {
		return PlatformResult{
			Platform:  platformName,
			Webhook:   webhook.Name,
			Status:    "failed",
			Message:   "不支持的平台: " + platformName,
			Timestamp: time.Now(),
		}
	}

	return platform.Send(webhook, req)
}

// ForwardToFeishu 转发到飞书
func (pm *PlatformManager) ForwardToFeishu(webhook config.WebhookConfig, req model.PushRequest) PlatformResult {
	return pm.Send("feishu", webhook, req)
}

// ForwardToDingtalk 转发到钉钉
func (pm *PlatformManager) ForwardToDingtalk(webhook config.WebhookConfig, req model.PushRequest) PlatformResult {
	return pm.Send("dingtalk", webhook, req)
}

// ForwardToWechat 转发到企业微信
func (pm *PlatformManager) ForwardToWechat(webhook config.WebhookConfig, req model.PushRequest) PlatformResult {
	return pm.Send("wechat", webhook, req)
}
