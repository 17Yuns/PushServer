package forwarder

import (
	"PushServer/internal/config"
	"PushServer/internal/model"
	"PushServer/internal/platform"
)

// ForwardService 转发服务
type ForwardService struct {
	platformManager *platform.PlatformManager
}

// NewForwardService 创建转发服务
func NewForwardService() *ForwardService {
	return &ForwardService{
		platformManager: platform.NewPlatformManager(),
	}
}

// ForwardResult 转发结果 (使用platform.PlatformResult)
type ForwardResult = platform.PlatformResult

// ForwardToFeishu 转发到飞书
func (fs *ForwardService) ForwardToFeishu(webhook config.WebhookConfig, req model.PushRequest) ForwardResult {
	return fs.platformManager.ForwardToFeishu(webhook, req)
}

// ForwardToDingtalk 转发到钉钉
func (fs *ForwardService) ForwardToDingtalk(webhook config.WebhookConfig, req model.PushRequest) ForwardResult {
	return fs.platformManager.ForwardToDingtalk(webhook, req)
}

// ForwardToWechat 转发到企业微信
func (fs *ForwardService) ForwardToWechat(webhook config.WebhookConfig, req model.PushRequest) ForwardResult {
	return fs.platformManager.ForwardToWechat(webhook, req)
}
