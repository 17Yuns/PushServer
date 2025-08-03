package platform

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"PushServer/internal/config"
	"PushServer/internal/logger"
	"PushServer/internal/model"
)

// DingtalkPlatform 钉钉平台
type DingtalkPlatform struct{}

// NewDingtalkPlatform 创建钉钉平台实例
func NewDingtalkPlatform() *DingtalkPlatform {
	return &DingtalkPlatform{}
}

// Send 发送消息到钉钉
func (d *DingtalkPlatform) Send(webhook config.WebhookConfig, req model.PushRequest) PlatformResult {
	logger.Infof("开始转发到钉钉: %s, 类型: %s, 样式: %s", webhook.Name, req.Type, req.Style)

	var payload interface{}

	// 根据样式选择消息格式
	if req.Style == model.StyleCard {
		payload = d.buildCardMessage(req)
	} else {
		payload = d.buildTextMessage(req)
	}

	// 发送HTTP请求
	result := d.sendHTTPRequest(webhook, payload)

	if result.Status == "success" {
		logger.Infof("钉钉转发成功: %s", webhook.Name)
	} else {
		logger.Errorf("钉钉转发失败: %s, 错误: %s", webhook.Name, result.Message)
	}

	return result
}

// buildTextMessage 构建文本消息
func (d *DingtalkPlatform) buildTextMessage(req model.PushRequest) map[string]interface{} {
	// 根据消息类型添加图标
	var icon string
	switch req.Type {
	case model.TypeError:
		icon = "❌"
	case model.TypeWarning:
		icon = "⚠️"
	case model.TypeInfo:
		icon = "ℹ️"
	default:
		icon = "📢"
	}

	text := fmt.Sprintf("%s %s\n%s", icon, req.Content.Title, req.Content.Msg)

	return map[string]interface{}{
		"msgtype": "text",
		"text": map[string]interface{}{
			"content": text,
		},
	}
}

// buildCardMessage 构建卡片消息（钉钉的ActionCard）
func (d *DingtalkPlatform) buildCardMessage(req model.PushRequest) map[string]interface{} {
	// 根据消息类型设置图标
	var icon string
	switch req.Type {
	case model.TypeError:
		icon = "❌"
	case model.TypeWarning:
		icon = "⚠️"
	case model.TypeInfo:
		icon = "ℹ️"
	default:
		icon = "📢"
	}

	// 构建Markdown格式的内容
	markdown := fmt.Sprintf(`
## %s %s

%s

---

**发送时间:** %s
`, icon, req.Content.Title, req.Content.Msg, time.Now().Format("2006-01-02 15:04:05"))

	return map[string]interface{}{
		"msgtype": "actionCard",
		"actionCard": map[string]interface{}{
			"title":          fmt.Sprintf("%s %s", icon, req.Content.Title),
			"text":           markdown,
			"hideAvatar":     "0",
			"btnOrientation": "0",
		},
	}
}

// sendHTTPRequest 发送HTTP请求
func (d *DingtalkPlatform) sendHTTPRequest(webhook config.WebhookConfig, payload interface{}) PlatformResult {
	result := PlatformResult{
		Platform:  "dingtalk",
		Webhook:   webhook.Name,
		Timestamp: time.Now(),
	}

	// 构建请求URL（包含签名）
	requestURL := webhook.URL
	if webhook.Secret != "" {
		timestamp := time.Now().UnixNano() / 1e6
		sign := d.generateSign(timestamp, webhook.Secret)
		requestURL = fmt.Sprintf("%s&timestamp=%d&sign=%s", webhook.URL, timestamp, url.QueryEscape(sign))
	}

	// 序列化payload
	jsonData, err := json.Marshal(payload)
	if err != nil {
		result.Status = "failed"
		result.Message = fmt.Sprintf("JSON序列化失败: %v", err)
		return result
	}

	// 创建HTTP请求
	req, err := http.NewRequest("POST", requestURL, bytes.NewBuffer(jsonData))
	if err != nil {
		result.Status = "failed"
		result.Message = fmt.Sprintf("创建HTTP请求失败: %v", err)
		return result
	}

	req.Header.Set("Content-Type", "application/json")

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

	// 检查钉钉API响应
	if errCode, ok := response["errcode"].(float64); ok && errCode == 0 {
		result.Status = "success"
		result.Message = "钉钉消息发送成功"
	} else {
		result.Status = "failed"
		if errMsg, ok := response["errmsg"].(string); ok {
			result.Message = fmt.Sprintf("钉钉API错误: %s", errMsg)
		} else {
			result.Message = "钉钉API返回未知错误"
		}
	}

	return result
}

// generateSign 生成钉钉签名
func (d *DingtalkPlatform) generateSign(timestamp int64, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(fmt.Sprintf("%v\n%s", timestamp, secret)))
	return url.QueryEscape(base64.StdEncoding.EncodeToString(h.Sum(nil)))
}

// GetName 获取平台名称
func (d *DingtalkPlatform) GetName() string {
	return "dingtalk"
}
