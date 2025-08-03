package platform

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"PushServer/internal/config"
	"PushServer/internal/logger"
	"PushServer/internal/model"
)

// WechatPlatform 企业微信平台
type WechatPlatform struct{}

// NewWechatPlatform 创建企业微信平台实例
func NewWechatPlatform() *WechatPlatform {
	return &WechatPlatform{}
}

// Send 发送消息到企业微信
func (w *WechatPlatform) Send(webhook config.WebhookConfig, req model.PushRequest) PlatformResult {
	logger.Infof("开始转发到企业微信: %s, 类型: %s, 样式: %s", webhook.Name, req.Type, req.Style)

	var payload interface{}

	// 根据样式选择消息格式
	if req.Style == model.StyleCard {
		payload = w.buildCardMessage(req)
	} else {
		payload = w.buildTextMessage(req)
	}

	// 发送HTTP请求
	result := w.sendHTTPRequest(webhook, payload)

	if result.Status == "success" {
		logger.Infof("企业微信转发成功: %s", webhook.Name)
	} else {
		logger.Errorf("企业微信转发失败: %s, 错误: %s", webhook.Name, result.Message)
	}

	return result
}

// buildTextMessage 构建文本消息
func (w *WechatPlatform) buildTextMessage(req model.PushRequest) map[string]interface{} {
	// 根据消息类型添加图标
	var icon string
	switch req.Type {
	case model.TypeError:
		icon = "🚨"
	case model.TypeWarning:
		icon = "⚠️"
	case model.TypeInfo:
		icon = "📋"
	default:
		icon = "💬"
	}

	content := fmt.Sprintf("%s %s\n%s", icon, req.Content.Title, req.Content.Msg)

	return map[string]interface{}{
		"msgtype": "text",
		"text": map[string]interface{}{
			"content": content,
		},
	}
}

// buildCardMessage 构建卡片消息（企业微信的模板卡片）
func (w *WechatPlatform) buildCardMessage(req model.PushRequest) map[string]interface{} {
	// 根据消息类型设置图标
	var icon string
	switch req.Type {
	case model.TypeError:
		icon = "🚨"
	case model.TypeWarning:
		icon = "⚠️"
	case model.TypeInfo:
		icon = "📋"
	default:
		icon = "💬"
	}

	// 构建Markdown格式的内容
	content := fmt.Sprintf(`## %s %s

%s

---
**发送时间:** %s`, icon, req.Content.Title, req.Content.Msg, time.Now().Format("2006-01-02 15:04:05"))

	return map[string]interface{}{
		"msgtype": "markdown",
		"markdown": map[string]interface{}{
			"content": content,
		},
	}
}

// sendHTTPRequest 发送HTTP请求
func (w *WechatPlatform) sendHTTPRequest(webhook config.WebhookConfig, payload interface{}) PlatformResult {
	result := PlatformResult{
		Platform:  "wechat",
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
		// 企业微信的签名验证
		signature := w.generateSign(string(jsonData), webhook.Secret)
		req.Header.Set("X-Signature", signature)
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

	// 解析响应
	var response map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		result.Status = "failed"
		result.Message = fmt.Sprintf("解析响应失败: %v", err)
		return result
	}

	// 检查企业微信API响应
	if errCode, ok := response["errcode"].(float64); ok && errCode == 0 {
		result.Status = "success"
		result.Message = "企业微信消息发送成功"
	} else {
		result.Status = "failed"
		if errMsg, ok := response["errmsg"].(string); ok {
			result.Message = fmt.Sprintf("企业微信API错误: %s", errMsg)
		} else {
			result.Message = "企业微信API返回未知错误"
		}
	}

	return result
}

// generateSign 生成企业微信签名
func (w *WechatPlatform) generateSign(data, secret string) string {
	h := md5.New()
	h.Write([]byte(data + secret))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// GetName 获取平台名称
func (w *WechatPlatform) GetName() string {
	return "wechat"
}
