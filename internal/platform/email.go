package platform

import (
	"fmt"
	"net/smtp"
	"time"

	"PushServer/internal/config"
	"PushServer/internal/logger"
	"PushServer/internal/model"
)

// EmailPlatform 邮件平台
type EmailPlatform struct{}

// NewEmailPlatform 创建邮件平台实例
func NewEmailPlatform() *EmailPlatform {
	return &EmailPlatform{}
}

// Send 发送邮件
func (e *EmailPlatform) Send(webhook config.WebhookConfig, req model.PushRequest) PlatformResult {
	logger.Infof("开始发送邮件: %s, 类型: %s, 样式: %s", webhook.Name, req.Type, req.Style)

	// 检查是否配置了SMTP服务器
	if config.AppConfig.Email.SMTPHost == "" {
		return PlatformResult{
			Platform:  "email",
			Webhook:   webhook.Name,
			Status:    "failed",
			Message:   "未配置SMTP服务器",
			Timestamp: time.Now(),
		}
	}

	// 使用SMTP发送邮件
	return e.sendViaSMTP(webhook, req)
}

// sendViaSMTP 通过SMTP发送邮件
func (e *EmailPlatform) sendViaSMTP(webhook config.WebhookConfig, req model.PushRequest) PlatformResult {
	result := PlatformResult{
		Platform:  "email",
		Webhook:   webhook.Name,
		Timestamp: time.Now(),
	}

	// 构建邮件内容
	subject, body := e.buildEmailContent(req)

	// 设置邮件头
	headers := make(map[string]string)
	headers["From"] = config.AppConfig.Email.From
	headers["To"] = webhook.URL // 在邮件场景下，URL字段存储收件人邮箱
	headers["Subject"] = subject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/html; charset=UTF-8"

	// 构建完整邮件
	message := ""
	for k, v := range headers {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	message += "\r\n" + body

	// SMTP认证
	auth := smtp.PlainAuth("", config.AppConfig.Email.Username, config.AppConfig.Email.Password, config.AppConfig.Email.SMTPHost)

	// 发送邮件
	addr := fmt.Sprintf("%s:%d", config.AppConfig.Email.SMTPHost, config.AppConfig.Email.SMTPPort)
	err := smtp.SendMail(addr, auth, config.AppConfig.Email.From, []string{webhook.URL}, []byte(message))

	if err != nil {
		result.Status = "failed"
		result.Message = fmt.Sprintf("SMTP发送失败: %v", err)
		logger.Errorf("邮件发送失败: %s, 错误: %v", webhook.Name, err)
	} else {
		result.Status = "success"
		result.Message = "邮件发送成功"
		logger.Infof("邮件发送成功: %s", webhook.Name)
	}

	return result
}

// buildEmailContent 构建邮件内容
func (e *EmailPlatform) buildEmailContent(req model.PushRequest) (string, string) {
	// 根据消息类型设置主题前缀和样式
	var prefix, color, icon string
	switch req.Type {
	case model.TypeError:
		prefix = "[错误]"
		color = "#dc3545"
		icon = "🔴"
	case model.TypeWarning:
		prefix = "[警告]"
		color = "#ffc107"
		icon = "🟡"
	case model.TypeInfo:
		prefix = "[信息]"
		color = "#17a2b8"
		icon = "🔵"
	default:
		prefix = "[通知]"
		color = "#6c757d"
		icon = "ℹ️"
	}

	subject := fmt.Sprintf("%s %s", prefix, req.Content.Title)

	// 根据样式构建邮件正文
	var body string
	if req.Style == model.StyleCard {
		body = e.buildHTMLCard(req, color, icon)
	} else {
		body = e.buildHTMLText(req, color, icon)
	}

	return subject, body
}

// buildHTMLText 构建HTML文本邮件
func (e *EmailPlatform) buildHTMLText(req model.PushRequest, color, icon string) string {
	return fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>%s</title>
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
    <div style="max-width: 600px; margin: 0 auto; padding: 20px;">
        <h2 style="color: %s; border-left: 4px solid %s; padding-left: 10px;">
            %s %s
        </h2>
        <div style="background-color: #f8f9fa; padding: 15px; border-radius: 5px; margin: 20px 0;">
            <p style="margin: 0; white-space: pre-wrap;">%s</p>
        </div>
        <hr style="border: none; border-top: 1px solid #eee; margin: 20px 0;">
        <p style="color: #666; font-size: 12px; margin: 0;">
            发送时间: %s<br>
            消息类型: %s
        </p>
    </div>
</body>
</html>`,
		req.Content.Title,
		color, color,
		icon, req.Content.Title,
		req.Content.Msg,
		time.Now().Format("2006-01-02 15:04:05"),
		req.Type,
	)
}

// buildHTMLCard 构建HTML卡片邮件
func (e *EmailPlatform) buildHTMLCard(req model.PushRequest, color, icon string) string {
	return fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>%s</title>
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333; background-color: #f5f5f5; padding: 20px;">
    <div style="max-width: 600px; margin: 0 auto; background-color: white; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); overflow: hidden;">
        <!-- 头部 -->
        <div style="background-color: %s; color: white; padding: 20px; text-align: center;">
            <h1 style="margin: 0; font-size: 24px;">%s %s</h1>
        </div>
        
        <!-- 内容区域 -->
        <div style="padding: 30px;">
            <div style="background-color: #f8f9fa; border-left: 4px solid %s; padding: 20px; margin: 20px 0; border-radius: 4px;">
                <h3 style="margin-top: 0; color: %s;">消息内容</h3>
                <p style="margin-bottom: 0; white-space: pre-wrap; font-size: 16px;">%s</p>
            </div>
            
            <!-- 信息栏 -->
            <div style="background-color: #e9ecef; padding: 15px; border-radius: 4px; margin-top: 20px;">
                <table style="width: 100%%; border-collapse: collapse;">
                    <tr>
                        <td style="padding: 5px 0; font-weight: bold; color: #495057;">发送时间:</td>
                        <td style="padding: 5px 0; color: #6c757d;">%s</td>
                    </tr>
                    <tr>
                        <td style="padding: 5px 0; font-weight: bold; color: #495057;">消息类型:</td>
                        <td style="padding: 5px 0; color: #6c757d;">%s</td>
                    </tr>
                    <tr>
                        <td style="padding: 5px 0; font-weight: bold; color: #495057;">推送策略:</td>
                        <td style="padding: 5px 0; color: #6c757d;">%s</td>
                    </tr>
                </table>
            </div>
        </div>
        
        <!-- 底部 -->
        <div style="background-color: #f8f9fa; padding: 15px; text-align: center; border-top: 1px solid #dee2e6;">
            <p style="margin: 0; color: #6c757d; font-size: 12px;">
                此邮件由消息推送服务自动发送，请勿回复
            </p>
        </div>
    </div>
</body>
</html>`,
		req.Content.Title,
		color,
		icon, req.Content.Title,
		color,
		color, req.Content.Msg,
		time.Now().Format("2006-01-02 15:04:05"),
		req.Type,
		req.Strategy,
	)
}

// GetName 获取平台名称
func (e *EmailPlatform) GetName() string {
	return "email"
}
