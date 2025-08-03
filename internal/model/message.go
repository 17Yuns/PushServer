package model

import "fmt"

// PushRequest 推送请求结构
type PushRequest struct {
	RecipientAlias string      `json:"recipient_alias" binding:"required"` // 接收者别名
	Type           string      `json:"type"`                               // 消息类型: error, warning, info
	Platform       string      `json:"platform"`                          // 指定平台(可选)
	Strategy       string      `json:"strategy"`                          // 发送策略
	Style          string      `json:"style"`                             // 消息样式: text, card
	Content        MessageContent `json:"content" binding:"required"`     // 消息内容
}

// MessageContent 消息内容
type MessageContent struct {
	Title string `json:"title" binding:"required"` // 消息标题
	Msg   string `json:"msg" binding:"required"`   // 消息内容
}

// 消息类型常量
const (
	TypeInfo    = "info"
	TypeWarning = "warning"
	TypeError   = "error"
)

// 发送策略常量
const (
	StrategyAll             = "all"             // 所有渠道都发送
	StrategyFailover        = "failover"        // 渠道间故障转移
	StrategyWebhookAll      = "webhook_all"     // 每个渠道内所有webhook都发送
	StrategyWebhookFailover = "webhook_failover" // 每个渠道内webhook故障转移
	StrategyMixed           = "mixed"           // 渠道间故障转移，渠道内webhook全发送
)

// 消息样式常量
const (
	StyleText = "text"
	StyleCard = "card"
)

// SetDefaults 设置默认值
func (r *PushRequest) SetDefaults() {
	if r.Type == "" {
		r.Type = TypeInfo
	}
	if r.Strategy == "" {
		r.Strategy = StrategyFailover
	}
	if r.Style == "" {
		r.Style = StyleText
	}
}

// Validate 验证请求参数
func (r *PushRequest) Validate() error {
	// 验证消息类型
	if r.Type != TypeInfo && r.Type != TypeWarning && r.Type != TypeError {
		return fmt.Errorf("无效的消息类型: %s，只支持 info, warning, error", r.Type)
	}
	
	// 验证发送策略
	validStrategies := []string{StrategyAll, StrategyFailover, StrategyWebhookAll, StrategyWebhookFailover, StrategyMixed}
	valid := false
	for _, strategy := range validStrategies {
		if r.Strategy == strategy {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("无效的发送策略: %s", r.Strategy)
	}
	
	// 验证消息样式
	if r.Style != StyleText && r.Style != StyleCard {
		return fmt.Errorf("无效的消息样式: %s，只支持 text, card", r.Style)
	}
	
	return nil
}