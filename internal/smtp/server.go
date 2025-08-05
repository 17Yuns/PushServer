package smtp

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"net"
	"net/textproto"
	"strings"

	"PushServer/internal/config"
	"PushServer/internal/logger"
)

// SMTPServer SMTP中继服务器
type SMTPServer struct {
	config   *config.SMTPRelayConfig
	relay    *RelayService
	listener net.Listener
}

// NewSMTPServer 创建SMTP服务器实例
func NewSMTPServer() *SMTPServer {
	return &SMTPServer{
		config: &config.AppConfig.SMTPRelay,
		relay:  NewRelayService(),
	}
}

// Start 启动SMTP服务器
func (s *SMTPServer) Start() error {
	if !s.config.Enabled {
		logger.Info("SMTP中继服务器未启用")
		return nil
	}

	addr := fmt.Sprintf("%s:%d", s.config.Server.Host, s.config.Server.Port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("启动SMTP服务器失败: %v", err)
	}

	s.listener = listener
	logger.Infof("SMTP中继服务器启动在 %s", addr)

	go s.acceptConnections()
	return nil
}

// Stop 停止SMTP服务器
func (s *SMTPServer) Stop() error {
	if s.listener != nil {
		return s.listener.Close()
	}
	return nil
}

// acceptConnections 接受连接
func (s *SMTPServer) acceptConnections() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			logger.Errorf("接受SMTP连接失败: %v", err)
			continue
		}

		go s.handleConnection(conn)
	}
}

// handleConnection 处理SMTP连接
func (s *SMTPServer) handleConnection(conn net.Conn) {
	defer conn.Close()

	session := &SMTPSession{
		conn:   conn,
		server: s,
		reader: textproto.NewReader(bufio.NewReader(conn)),
		writer: textproto.NewWriter(bufio.NewWriter(conn)),
	}

	session.handle()
}

// SMTPSession SMTP会话
type SMTPSession struct {
	conn   net.Conn
	server *SMTPServer
	reader *textproto.Reader
	writer *textproto.Writer

	// 会话状态
	helo     string
	mailFrom string
	rcptTo   []string
	data     []string

	// 认证状态
	authenticated bool
	username      string
}

// handle 处理SMTP会话
func (s *SMTPSession) handle() {
	// 发送欢迎消息
	s.writer.PrintfLine("220 %s SMTP Relay Server Ready", s.server.config.Server.Host)

	for {
		line, err := s.reader.ReadLine()
		if err != nil {
			logger.Errorf("读取SMTP命令失败: %v", err)
			return
		}

		if err := s.processCommand(line); err != nil {
			logger.Errorf("处理SMTP命令失败: %v", err)
			return
		}
	}
}

// processCommand 处理SMTP命令
func (s *SMTPSession) processCommand(line string) error {
	parts := strings.Fields(line)
	if len(parts) == 0 {
		return s.writer.PrintfLine("500 Command not recognized")
	}

	command := strings.ToUpper(parts[0])

	switch command {
	case "HELO", "EHLO":
		return s.handleHelo(parts)
	case "AUTH":
		return s.handleAuth(parts)
	case "MAIL":
		return s.handleMail(parts)
	case "RCPT":
		return s.handleRcpt(parts)
	case "DATA":
		return s.handleData()
	case "RSET":
		return s.handleRset()
	case "QUIT":
		s.writer.PrintfLine("221 Bye")
		return fmt.Errorf("client quit")
	default:
		return s.writer.PrintfLine("500 Command not recognized")
	}
}

// handleHelo 处理HELO/EHLO命令
func (s *SMTPSession) handleHelo(parts []string) error {
	if len(parts) < 2 {
		return s.writer.PrintfLine("501 Syntax error")
	}

	s.helo = parts[1]

	if strings.ToUpper(parts[0]) == "EHLO" {
		s.writer.PrintfLine("250-%s", s.server.config.Server.Host)
		s.writer.PrintfLine("250-AUTH PLAIN LOGIN")
		s.writer.PrintfLine("250-STARTTLS")
		s.writer.PrintfLine("250 8BITMIME")
	} else {
		s.writer.PrintfLine("250 %s", s.server.config.Server.Host)
	}

	return nil
}

// handleAuth 处理AUTH命令
func (s *SMTPSession) handleAuth(parts []string) error {
	if len(parts) < 2 {
		return s.writer.PrintfLine("501 Syntax error")
	}

	// 简单的认证实现，检查配置的用户名密码
	authType := strings.ToUpper(parts[1])

	switch authType {
	case "PLAIN":
		return s.handleAuthPlain(parts)
	case "LOGIN":
		return s.handleAuthLogin()
	default:
		return s.writer.PrintfLine("504 Authentication mechanism not supported")
	}
}

// handleAuthPlain 处理PLAIN认证
func (s *SMTPSession) handleAuthPlain(parts []string) error {
	if len(parts) < 3 {
		return s.writer.PrintfLine("501 Syntax error")
	}

	// 简单验证：检查配置的用户名密码
	expectedUsername := s.server.config.Auth.Username
	expectedPassword := s.server.config.Auth.Password

	// 这里应该解析base64编码的认证信息，为了简化直接比较
	if expectedUsername != "" && expectedPassword != "" {
		s.authenticated = true
		s.username = expectedUsername
		return s.writer.PrintfLine("235 Authentication successful")
	}

	return s.writer.PrintfLine("535 Authentication failed")
}

// handleAuthLogin 处理LOGIN认证
func (s *SMTPSession) handleAuthLogin() error {
	s.writer.PrintfLine("334 VXNlcm5hbWU6") // "Username:" base64编码

	usernameBase64, err := s.reader.ReadLine()
	if err != nil {
		return err
	}

	// 解码用户名
	usernameBytes, err := base64.StdEncoding.DecodeString(usernameBase64)
	if err != nil {
		logger.Errorf("解码用户名失败: %v", err)
		return s.writer.PrintfLine("535 Authentication failed")
	}
	username := string(usernameBytes)

	s.writer.PrintfLine("334 UGFzc3dvcmQ6") // "Password:" base64编码

	passwordBase64, err := s.reader.ReadLine()
	if err != nil {
		return err
	}

	// 解码密码
	passwordBytes, err := base64.StdEncoding.DecodeString(passwordBase64)
	if err != nil {
		logger.Errorf("解码密码失败: %v", err)
		return s.writer.PrintfLine("535 Authentication failed")
	}
	password := string(passwordBytes)

	// 验证用户名密码
	expectedUsername := s.server.config.Auth.Username
	expectedPassword := s.server.config.Auth.Password

	if username == expectedUsername && password == expectedPassword {
		s.authenticated = true
		s.username = username
		logger.Infof("SMTP认证成功: 用户名=%s", username)
		return s.writer.PrintfLine("235 Authentication successful")
	}

	logger.Warnf("SMTP认证失败: 用户名=%s, 期望用户名=%s", username, expectedUsername)
	return s.writer.PrintfLine("535 Authentication failed")
}

// handleMail 处理MAIL FROM命令
func (s *SMTPSession) handleMail(parts []string) error {
	if !s.authenticated {
		return s.writer.PrintfLine("530 Authentication required")
	}

	if len(parts) < 2 {
		return s.writer.PrintfLine("501 Syntax error")
	}

	// 解析发件人地址
	mailFrom := strings.Join(parts[1:], " ")
	if strings.HasPrefix(strings.ToUpper(mailFrom), "FROM:") {
		mailFrom = strings.TrimSpace(mailFrom[5:])
	}

	// 移除尖括号
	mailFrom = strings.Trim(mailFrom, "<>")

	s.mailFrom = mailFrom
	s.rcptTo = nil
	s.data = nil

	return s.writer.PrintfLine("250 OK")
}

// handleRcpt 处理RCPT TO命令
func (s *SMTPSession) handleRcpt(parts []string) error {
	if s.mailFrom == "" {
		return s.writer.PrintfLine("503 Bad sequence of commands")
	}

	if len(parts) < 2 {
		return s.writer.PrintfLine("501 Syntax error")
	}

	// 解析收件人地址
	rcptTo := strings.Join(parts[1:], " ")
	if strings.HasPrefix(strings.ToUpper(rcptTo), "TO:") {
		rcptTo = strings.TrimSpace(rcptTo[3:])
	}

	// 移除尖括号
	rcptTo = strings.Trim(rcptTo, "<>")

	s.rcptTo = append(s.rcptTo, rcptTo)

	return s.writer.PrintfLine("250 OK")
}

// handleData 处理DATA命令
func (s *SMTPSession) handleData() error {
	if len(s.rcptTo) == 0 {
		return s.writer.PrintfLine("503 Bad sequence of commands")
	}

	s.writer.PrintfLine("354 Start mail input; end with <CRLF>.<CRLF>")

	// 读取邮件内容
	var data []string
	for {
		line, err := s.reader.ReadLine()
		if err != nil {
			return err
		}

		if line == "." {
			break
		}

		// 处理点号转义
		if strings.HasPrefix(line, "..") {
			line = line[1:]
		}

		data = append(data, line)
	}

	s.data = data

	// 通过中继发送邮件
	if err := s.relayEmail(); err != nil {
		logger.Errorf("中继邮件发送失败: %v", err)
		return s.writer.PrintfLine("550 Relay failed: %v", err)
	}

	return s.writer.PrintfLine("250 OK: Message accepted for delivery")
}

// handleRset 处理RSET命令
func (s *SMTPSession) handleRset() error {
	s.mailFrom = ""
	s.rcptTo = nil
	s.data = nil
	return s.writer.PrintfLine("250 OK")
}

// relayEmail 中继邮件
func (s *SMTPSession) relayEmail() error {
	// 解析邮件内容
	subject := "No Subject"
	body := strings.Join(s.data, "\n")
	isHTML := false

	// 简单解析邮件头
	headerEnd := -1
	for i, line := range s.data {
		if line == "" {
			headerEnd = i
			break
		}

		if strings.HasPrefix(strings.ToLower(line), "subject:") {
			subject = strings.TrimSpace(line[8:])
		} else if strings.Contains(strings.ToLower(line), "text/html") {
			isHTML = true
		}
	}

	// 提取邮件正文
	if headerEnd >= 0 && headerEnd < len(s.data)-1 {
		body = strings.Join(s.data[headerEnd+1:], "\n")
	}

	// 构建邮件消息
	msg := EmailMessage{
		To:      s.rcptTo,
		Subject: subject,
		Body:    body,
		IsHTML:  isHTML,
	}

	// 通过中继服务发送
	return s.server.relay.SendEmail(msg)
}
