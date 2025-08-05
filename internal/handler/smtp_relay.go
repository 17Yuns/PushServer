package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	smtpRelay "PushServer/internal/smtp"
)

// GetSMTPRelayStatistics 获取SMTP中继统计信息
func GetSMTPRelayStatistics(c *gin.Context) {
	relayService := smtpRelay.NewRelayService()
	stats := relayService.GetStatistics()

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "获取SMTP中继统计成功",
		"data": gin.H{
			"statistics": stats,
		},
	})
}

// GetSMTPRelayStatus 获取SMTP中继状态
func GetSMTPRelayStatus(c *gin.Context) {
	relayService := smtpRelay.NewRelayService()

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "获取SMTP中继状态成功",
		"data": gin.H{
			"enabled": relayService.IsEnabled(),
			"status":  "运行中",
		},
	})
}
