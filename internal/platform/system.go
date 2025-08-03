package platform

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"PushServer/internal/config"
	"PushServer/internal/logger"
	"PushServer/internal/model"
)

// SystemPlatform 系统通知平台
type SystemPlatform struct{}

// NewSystemPlatform 创建系统通知平台实例
func NewSystemPlatform() *SystemPlatform {
	return &SystemPlatform{}
}

// Send 发送系统通知
func (s *SystemPlatform) Send(webhook config.WebhookConfig, req model.PushRequest) PlatformResult {
	logger.Infof("开始发送系统通知: %s, 类型: %s, 样式: %s", webhook.Name, req.Type, req.Style)

	// 根据配置的通知方式发送
	switch webhook.URL {
	case "syslog":
		return s.sendToSyslog(webhook, req)
	case "file":
		return s.sendToFile(webhook, req)
	case "console":
		return s.sendToConsole(webhook, req)
	case "http":
		return s.sendToHTTP(webhook, req)
	default:
		return PlatformResult{
			Platform:  "system",
			Webhook:   webhook.Name,
			Status:    "failed",
			Message:   "不支持的系统通知类型: " + webhook.URL,
			Timestamp: time.Now(),
		}
	}
}

// sendToSyslog 发送到系统日志
func (s *SystemPlatform) sendToSyslog(webhook config.WebhookConfig, req model.PushRequest) PlatformResult {
	// 构建日志消息
	logMessage := s.buildLogMessage(req)

	// 根据消息类型选择日志级别
	switch req.Type {
	case model.TypeError:
		logger.Errorf("[系统通知] %s", logMessage)
	case model.TypeWarning:
		logger.Warnf("[系统通知] %s", logMessage)
	case model.TypeInfo:
		logger.Infof("[系统通知] %s", logMessage)
	default:
		logger.Infof("[系统通知] %s", logMessage)
	}

	return PlatformResult{
		Platform:  "system",
		Webhook:   webhook.Name,
		Status:    "success",
		Message:   "系统通知已写入日志",
		Timestamp: time.Now(),
	}
}

// sendToFile 发送到文件
func (s *SystemPlatform) sendToFile(webhook config.WebhookConfig, req model.PushRequest) PlatformResult {
	// 确保通知目录存在
	notifyDir := "notifications"
	if err := os.MkdirAll(notifyDir, 0755); err != nil {
		return PlatformResult{
			Platform:  "system",
			Webhook:   webhook.Name,
			Status:    "failed",
			Message:   fmt.Sprintf("创建通知目录失败: %v", err),
			Timestamp: time.Now(),
		}
	}

	// 生成文件名
	filename := fmt.Sprintf("system_notify_%s.txt", time.Now().Format("20060102_150405"))
	filepath := filepath.Join(notifyDir, filename)

	// 构建文件内容
	content := s.buildFileContent(req)

	// 写入文件
	if err := os.WriteFile(filepath, []byte(content), 0644); err != nil {
		return PlatformResult{
			Platform:  "system",
			Webhook:   webhook.Name,
			Status:    "failed",
			Message:   fmt.Sprintf("写入通知文件失败: %v", err),
			Timestamp: time.Now(),
		}
	}

	logger.Infof("系统通知发送成功: %s (文件: %s)", webhook.Name, filepath)
	return PlatformResult{
		Platform:  "system",
		Webhook:   webhook.Name,
		Status:    "success",
		Message:   fmt.Sprintf("系统通知已保存到文件: %s", filepath),
		Timestamp: time.Now(),
	}
}

// sendToConsole 发送到控制台
func (s *SystemPlatform) sendToConsole(webhook config.WebhookConfig, req model.PushRequest) PlatformResult {
	// 构建控制台消息
	consoleMessage := s.buildConsoleMessage(req)

	// 输出到控制台
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("🚨 系统通知 🚨")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println(consoleMessage)
	fmt.Println(strings.Repeat("=", 60))

	logger.Infof("系统通知发送成功: %s (控制台)", webhook.Name)
	return PlatformResult{
		Platform:  "system",
		Webhook:   webhook.Name,
		Status:    "success",
		Message:   "系统通知已输出到控制台",
		Timestamp: time.Now(),
	}
}

// sendToHTTP 发送HTTP通知
func (s *SystemPlatform) sendToHTTP(webhook config.WebhookConfig, req model.PushRequest) PlatformResult {
	// 从配置中获取HTTP URL
	var httpURL string
	for _, notifyConfig := range config.AppConfig.System.Notifications {
		if notifyConfig.Type == "http" && notifyConfig.Name == webhook.Name {
			httpURL = notifyConfig.URL
			break
		}
	}

	if httpURL == "" {
		return PlatformResult{
			Platform:  "system",
			Webhook:   webhook.Name,
			Status:    "failed",
			Message:   "HTTP通知URL未配置",
			Timestamp: time.Now(),
		}
	}

	// 构建HTTP请求数据
	data := s.buildHTTPData(req)
	jsonData, err := json.Marshal(data)
	if err != nil {
		return PlatformResult{
			Platform:  "system",
			Webhook:   webhook.Name,
			Status:    "failed",
			Message:   "构建HTTP请求数据失败: " + err.Error(),
			Timestamp: time.Now(),
		}
	}

	// 发送HTTP请求
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Post(httpURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return PlatformResult{
			Platform:  "system",
			Webhook:   webhook.Name,
			Status:    "failed",
			Message:   "HTTP请求发送失败: " + err.Error(),
			Timestamp: time.Now(),
		}
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		logger.Infof("系统通知发送成功: %s (HTTP: %s)", webhook.Name, httpURL)
		return PlatformResult{
			Platform:  "system",
			Webhook:   webhook.Name,
			Status:    "success",
			Message:   fmt.Sprintf("HTTP通知发送成功，状态码: %d", resp.StatusCode),
			Timestamp: time.Now(),
		}
	} else {
		logger.Errorf("系统通知发送失败: %s (HTTP: %s, 状态码: %d)", webhook.Name, httpURL, resp.StatusCode)
		return PlatformResult{
			Platform:  "system",
			Webhook:   webhook.Name,
			Status:    "failed",
			Message:   fmt.Sprintf("HTTP通知发送失败，状态码: %d", resp.StatusCode),
			Timestamp: time.Now(),
		}
	}
}

// buildLogMessage 构建日志消息
func (s *SystemPlatform) buildLogMessage(req model.PushRequest) string {
	return fmt.Sprintf("标题: %s | 内容: %s | 类型: %s | 策略: %s | 时间: %s",
		req.Content.Title,
		req.Content.Msg,
		req.Type,
		req.Strategy,
		time.Now().Format("2006-01-02 15:04:05"),
	)
}

// buildFileContent 构建文件内容
func (s *SystemPlatform) buildFileContent(req model.PushRequest) string {
	var typeIcon string
	switch req.Type {
	case model.TypeError:
		typeIcon = "🔴 错误"
	case model.TypeWarning:
		typeIcon = "🟡 警告"
	case model.TypeInfo:
		typeIcon = "🔵 信息"
	default:
		typeIcon = "ℹ️ 通知"
	}

	return fmt.Sprintf(`系统通知记录
==========================================

%s %s

内容详情:
%s

推送信息:
- 消息类型: %s
- 推送策略: %s
- 消息样式: %s
- 通知时间: %s

==========================================
此通知由PushServer系统自动生成
`,
		typeIcon, req.Content.Title,
		req.Content.Msg,
		req.Type,
		req.Strategy,
		req.Style,
		time.Now().Format("2006-01-02 15:04:05"),
	)
}

// buildConsoleMessage 构建控制台消息
func (s *SystemPlatform) buildConsoleMessage(req model.PushRequest) string {
	var typeIcon string
	switch req.Type {
	case model.TypeError:
		typeIcon = "🔴 错误"
	case model.TypeWarning:
		typeIcon = "🟡 警告"
	case model.TypeInfo:
		typeIcon = "🔵 信息"
	default:
		typeIcon = "ℹ️ 通知"
	}

	return fmt.Sprintf(`%s %s

内容: %s

类型: %s | 策略: %s | 样式: %s
时间: %s`,
		typeIcon, req.Content.Title,
		req.Content.Msg,
		req.Type,
		req.Strategy,
		req.Style,
		time.Now().Format("2006-01-02 15:04:05"),
	)
}

// HTTPNotificationData HTTP通知数据结构
type HTTPNotificationData struct {
	Title     string `json:"title"`
	Message   string `json:"message"`
	Type      string `json:"type"`
	Strategy  string `json:"strategy"`
	Style     string `json:"style"`
	Timestamp string `json:"timestamp"`
	Source    string `json:"source"`
}

// buildHTTPData 构建HTTP请求数据
func (s *SystemPlatform) buildHTTPData(req model.PushRequest) HTTPNotificationData {
	return HTTPNotificationData{
		Title:     req.Content.Title,
		Message:   req.Content.Msg,
		Type:      req.Type,
		Strategy:  req.Strategy,
		Style:     req.Style,
		Timestamp: time.Now().Format("2006-01-02 15:04:05"),
		Source:    "PushServer-SystemNotification",
	}
}

// GetName 获取平台名称
func (s *SystemPlatform) GetName() string {
	return "system"
}
