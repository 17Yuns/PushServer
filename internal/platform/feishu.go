package platform

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"PushServer/internal/config"
	"PushServer/internal/logger"
	"PushServer/internal/model"
)

// FeishuPlatform 飞书平台
type FeishuPlatform struct{}

// NewFeishuPlatform 创建飞书平台实例
func NewFeishuPlatform() *FeishuPlatform {
	return &FeishuPlatform{}
}

// Send 发送消息到飞书
func (f *FeishuPlatform) Send(webhook config.WebhookConfig, req model.PushRequest) PlatformResult {
	logger.Infof("开始转发到飞书: %s, 类型: %s, 样式: %s", webhook.Name, req.Type, req.Style)

	var payload interface{}

	// 根据样式选择消息格式
	if req.Style == model.StyleCard {
		payload = f.buildCardMessage(req)
	} else {
		payload = f.buildTextMessage(req)
	}

	// 发送HTTP请求
	result := f.sendHTTPRequest(webhook, payload)

	if result.Status == "success" {
		logger.Infof("飞书转发成功: %s", webhook.Name)
	} else {
		logger.Errorf("飞书转发失败: %s, 错误: %s", webhook.Name, result.Message)
	}

	return result
}

// buildTextMessage 构建文本消息
func (f *FeishuPlatform) buildTextMessage(req model.PushRequest) map[string]interface{} {
	// 根据消息类型添加图标
	var icon string
	switch req.Type {
	case model.TypeError:
		icon = "🔴"
	case model.TypeWarning:
		icon = "🟡"
	case model.TypeInfo:
		icon = "🔵"
	default:
		icon = "ℹ️"
	}

	text := fmt.Sprintf("%s %s\n%s", icon, req.Content.Title, req.Content.Msg)

	return map[string]interface{}{
		"msg_type": "text",
		"content": map[string]interface{}{
			"text": text,
		},
	}
}

// buildCardMessage 构建卡片消息
func (f *FeishuPlatform) buildCardMessage(req model.PushRequest) map[string]interface{} {
	// 根据消息类型设置颜色
	var color string
	var icon string
	switch req.Type {
	case model.TypeError:
		color = "red"
		icon = "🔴"
	case model.TypeWarning:
		color = "orange"
		icon = "🟡"
	case model.TypeInfo:
		color = "blue"
		icon = "🔵"
	default:
		color = "grey"
		icon = "ℹ️"
	}

	card := map[string]interface{}{
		"config": map[string]interface{}{
			"wide_screen_mode": true,
		},
		"elements": []map[string]interface{}{
			{
				"tag": "div",
				"text": map[string]interface{}{
					"content": fmt.Sprintf("**%s %s**", icon, req.Content.Title),
					"tag":     "lark_md",
				},
			},
			{
				"tag": "div",
				"text": map[string]interface{}{
					"content": req.Content.Msg,
					"tag":     "lark_md",
				},
			},
			{
				"tag": "hr",
			},
			{
				"tag": "div",
				"text": map[string]interface{}{
					"content": fmt.Sprintf("**发送时间:** %s", time.Now().Format("2006-01-02 15:04:05")),
					"tag":     "lark_md",
				},
			},
		},
		"header": map[string]interface{}{
			"template": color,
			"title": map[string]interface{}{
				"content": fmt.Sprintf("%s %s", icon, req.Content.Title),
				"tag":     "plain_text",
			},
		},
	}

	return map[string]interface{}{
		"msg_type": "interactive",
		"card":     card,
	}
}

// sendHTTPRequest 发送HTTP请求
func (f *FeishuPlatform) sendHTTPRequest(webhook config.WebhookConfig, payload interface{}) PlatformResult {
	result := PlatformResult{
		Platform:  "feishu",
		Webhook:   webhook.Name,
		Timestamp: time.Now(),
	}

	// 序列化payload
	jsonData, err := json.Marshal(payload)
	if err != nil {
		result.Status = "failed"
		result.Message = fmt.Sprintf("JSON序列化失败: %v", err)
		return result
	}

	// 创建HTTP请求
	req, err := http.NewRequest("POST", webhook.URL, bytes.NewBuffer(jsonData))
	if err != nil {
		result.Status = "failed"
		result.Message = fmt.Sprintf("创建HTTP请求失败: %v", err)
		return result
	}

	req.Header.Set("Content-Type", "application/json")

	// 如果有签名，添加签名头
	if webhook.Secret != "" {
		// 这里可以添加飞书的签名验证逻辑
		// req.Header.Set("X-Lark-Signature", signature)
	}

	// 发送请求
	client := &http.Client{
		Timeout: time.Duration(config.AppConfig.Queue.Timeout) * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		result.Status = "failed"
		result.Message = fmt.Sprintf("HTTP请求失败: %v", err)
		return result
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode == http.StatusOK {
		result.Status = "success"
		result.Message = "飞书消息发送成功"
	} else {
		result.Status = "failed"
		result.Message = fmt.Sprintf("飞书API返回错误状态码: %d", resp.StatusCode)
	}

	return result
}

// GetName 获取平台名称
func (f *FeishuPlatform) GetName() string {
	return "feishu"
}
