package smtp

import (
	"crypto/tls"
	"fmt"
	"math/rand"
	"net/smtp"
	"strings"
	"time"

	"PushServer/internal/config"
	"PushServer/internal/logger"
	"PushServer/internal/model"
	"PushServer/internal/notification"
)

// RelayService SMTP中继服务
type RelayService struct {
	config *config.SMTPRelayConfig
}

// NewRelayService 创建SMTP中继服务实例
func NewRelayService() *RelayService {
	return &RelayService{
		config: &config.AppConfig.SMTPRelay,
	}
}

// EmailMessage 邮件消息结构
type EmailMessage struct {
	To      []string
	Subject string
	Body    string
	IsHTML  bool
}

// SendEmail 发送邮件（通过SMTP中继）
func (rs *RelayService) SendEmail(msg EmailMessage) error {
	if !rs.config.Enabled {
		return fmt.Errorf("SMTP中继功能未启用")
	}

	// 获取可用的SMTP账户
	availableAccounts := rs.getAvailableAccounts()
	if len(availableAccounts) == 0 {
		return fmt.Errorf("没有可用的SMTP账户")
	}

	// 随机打乱账户顺序
	rs.shuffleAccounts(availableAccounts)

	var lastErr error
	maxRetries := rs.config.MaxRetries
	if maxRetries <= 0 {
		maxRetries = len(availableAccounts)
	}

	// 尝试发送邮件
	for i, account := range availableAccounts {
		if i >= maxRetries {
			break
		}

		logger.Infof("尝试使用SMTP账户发送邮件: %s (%s)", account.Name, account.Host)

		err := rs.sendEmailWithAccount(account, msg)
		if err == nil {
			logger.Infof("邮件发送成功，使用账户: %s", account.Name)
			return nil
		}

		logger.Warnf("SMTP账户 %s 发送失败: %v", account.Name, err)
		lastErr = err
	}

	// 所有账户都失败，触发系统通知
	rs.triggerSystemNotification(msg, lastErr)

	return fmt.Errorf("所有SMTP账户都发送失败，最后错误: %v", lastErr)
}

// sendEmailWithAccount 使用指定账户发送邮件
func (rs *RelayService) sendEmailWithAccount(account config.SMTPAccountConfig, msg EmailMessage) error {
	// 构建邮件内容
	emailContent := rs.buildEmailContent(account.From, msg)

	// 建立SMTP连接
	addr := fmt.Sprintf("%s:%d", account.Host, account.Port)

	// 创建TLS配置
	tlsConfig := &tls.Config{
		ServerName: account.Host,
	}

	// 连接到SMTP服务器
	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		// 尝试非TLS连接
		client, err := smtp.Dial(addr)
		if err != nil {
			return fmt.Errorf("连接SMTP服务器失败: %v", err)
		}
		defer client.Close()

		// 尝试启用STARTTLS
		if ok, _ := client.Extension("STARTTLS"); ok {
			if err = client.StartTLS(tlsConfig); err != nil {
				return fmt.Errorf("启用STARTTLS失败: %v", err)
			}
		}

		return rs.sendEmailWithClient(client, account, msg.To, emailContent)
	}
	defer conn.Close()

	// 使用TLS连接创建SMTP客户端
	client, err := smtp.NewClient(conn, account.Host)
	if err != nil {
		return fmt.Errorf("创建SMTP客户端失败: %v", err)
	}
	defer client.Close()

	return rs.sendEmailWithClient(client, account, msg.To, emailContent)
}

// sendEmailWithClient 使用SMTP客户端发送邮件
func (rs *RelayService) sendEmailWithClient(client *smtp.Client, account config.SMTPAccountConfig, to []string, content string) error {
	// 身份验证
	if account.Username != "" && account.Password != "" {
		auth := smtp.PlainAuth("", account.Username, account.Password, account.Host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP身份验证失败: %v", err)
		}
	}

	// 设置发件人
	if err := client.Mail(account.From); err != nil {
		return fmt.Errorf("设置发件人失败: %v", err)
	}

	// 设置收件人
	for _, recipient := range to {
		if err := client.Rcpt(recipient); err != nil {
			return fmt.Errorf("设置收件人失败 (%s): %v", recipient, err)
		}
	}

	// 发送邮件内容
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("开始发送邮件数据失败: %v", err)
	}
	defer writer.Close()

	if _, err := writer.Write([]byte(content)); err != nil {
		return fmt.Errorf("写入邮件内容失败: %v", err)
	}

	return nil
}

// buildEmailContent 构建邮件内容
func (rs *RelayService) buildEmailContent(from string, msg EmailMessage) string {
	var contentType string
	if msg.IsHTML {
		contentType = "text/html; charset=UTF-8"
	} else {
		contentType = "text/plain; charset=UTF-8"
	}

	content := fmt.Sprintf("From: %s\r\n", from)
	content += fmt.Sprintf("To: %s\r\n", strings.Join(msg.To, ", "))
	content += fmt.Sprintf("Subject: %s\r\n", msg.Subject)
	content += fmt.Sprintf("Content-Type: %s\r\n", contentType)
	content += fmt.Sprintf("Date: %s\r\n", time.Now().Format(time.RFC1123Z))
	content += "\r\n"
	content += msg.Body

	return content
}

// getAvailableAccounts 获取可用的SMTP账户
func (rs *RelayService) getAvailableAccounts() []config.SMTPAccountConfig {
	var available []config.SMTPAccountConfig
	for _, account := range rs.config.Accounts {
		if account.Enabled && account.Host != "" && account.From != "" {
			available = append(available, account)
		}
	}
	return available
}

// shuffleAccounts 随机打乱账户顺序
func (rs *RelayService) shuffleAccounts(accounts []config.SMTPAccountConfig) {
	rand.Seed(time.Now().UnixNano())
	for i := len(accounts) - 1; i > 0; i-- {
		j := rand.Intn(i + 1)
		accounts[i], accounts[j] = accounts[j], accounts[i]
	}
}

// triggerSystemNotification 触发系统通知
func (rs *RelayService) triggerSystemNotification(msg EmailMessage, err error) {
	// 构建系统通知内容
	title := "SMTP中继发送失败"
	message := fmt.Sprintf("所有SMTP账户都无法发送邮件\n\n原始邮件信息:\n收件人: %s\n主题: %s\n\n错误信息: %v",
		strings.Join(msg.To, ", "),
		msg.Subject,
		err,
	)

	// 创建推送请求（用于系统通知）
	pushReq := model.PushRequest{
		RecipientAlias: "smtp_relay_error",
		Type:           model.TypeError,
		Strategy:       "system",
		Style:          model.StyleCard,
		Content: model.MessageContent{
			Title: title,
			Msg:   message,
		},
	}

	// 添加到系统通知
	notificationID := notification.Manager.AddNotification("", pushReq, "SMTP中继发送失败")
	logger.Errorf("SMTP中继发送失败，已触发系统通知: %s", notificationID)
}

// GetStatistics 获取SMTP中继统计信息
func (rs *RelayService) GetStatistics() map[string]interface{} {
	stats := map[string]interface{}{
		"enabled":            rs.config.Enabled,
		"total_accounts":     len(rs.config.Accounts),
		"available_accounts": len(rs.getAvailableAccounts()),
		"max_retries":        rs.config.MaxRetries,
	}

	// 账户详情
	var accountStats []map[string]interface{}
	for _, account := range rs.config.Accounts {
		accountStats = append(accountStats, map[string]interface{}{
			"name":    account.Name,
			"host":    account.Host,
			"port":    account.Port,
			"from":    account.From,
			"enabled": account.Enabled,
		})
	}
	stats["accounts"] = accountStats

	return stats
}

// IsEnabled 检查SMTP中继是否启用
func (rs *RelayService) IsEnabled() bool {
	return rs.config.Enabled && len(rs.getAvailableAccounts()) > 0
}
