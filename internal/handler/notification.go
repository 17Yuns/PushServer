package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"PushServer/internal/notification"
)

// GetSystemNotifications 获取系统通知列表
func GetSystemNotifications(c *gin.Context) {
	// 获取查询参数
	status := c.Query("status")    // unread, read, all
	limitStr := c.Query("limit")   // 限制数量
	offsetStr := c.Query("offset") // 偏移量

	// 解析分页参数
	limit := 50 // 默认限制50条
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 1000 {
			limit = l
		}
	}

	offset := 0 // 默认偏移量0
	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	var notifications []*notification.SystemNotification

	// 根据状态获取通知
	switch status {
	case "unread":
		notifications = notification.Manager.GetNotificationsByStatus("unread")
	case "read":
		notifications = notification.Manager.GetNotificationsByStatus("read")
	default:
		notifications = notification.Manager.GetAllNotifications()
	}

	// 应用分页
	total := len(notifications)
	start := offset
	end := offset + limit

	if start >= total {
		notifications = []*notification.SystemNotification{}
	} else {
		if end > total {
			end = total
		}
		notifications = notifications[start:end]
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "获取系统通知成功",
		"data": gin.H{
			"notifications": notifications,
			"pagination": gin.H{
				"total":  total,
				"limit":  limit,
				"offset": offset,
				"count":  len(notifications),
			},
		},
	})
}

// GetSystemNotification 获取单个系统通知
func GetSystemNotification(c *gin.Context) {
	notificationID := c.Param("id")

	if notificationID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "通知ID不能为空",
			"data":    nil,
		})
		return
	}

	notification, exists := notification.Manager.GetNotification(notificationID)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "通知不存在",
			"data": gin.H{
				"notification_id": notificationID,
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "获取系统通知成功",
		"data": gin.H{
			"notification": notification,
		},
	})
}

// MarkNotificationAsRead 标记通知为已读
func MarkNotificationAsRead(c *gin.Context) {
	notificationID := c.Param("id")

	if notificationID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "通知ID不能为空",
			"data":    nil,
		})
		return
	}

	success := notification.Manager.MarkAsRead(notificationID)
	if !success {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "通知不存在",
			"data": gin.H{
				"notification_id": notificationID,
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "标记为已读成功",
		"data": gin.H{
			"notification_id": notificationID,
		},
	})
}

// MarkAllNotificationsAsRead 标记所有通知为已读
func MarkAllNotificationsAsRead(c *gin.Context) {
	count := notification.Manager.MarkAllAsRead()

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "标记所有通知为已读成功",
		"data": gin.H{
			"marked_count": count,
		},
	})
}

// DeleteSystemNotification 删除系统通知
func DeleteSystemNotification(c *gin.Context) {
	notificationID := c.Param("id")

	if notificationID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "通知ID不能为空",
			"data":    nil,
		})
		return
	}

	success := notification.Manager.DeleteNotification(notificationID)
	if !success {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "通知不存在",
			"data": gin.H{
				"notification_id": notificationID,
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "删除通知成功",
		"data": gin.H{
			"notification_id": notificationID,
		},
	})
}

// ClearAllNotifications 清空所有系统通知
func ClearAllNotifications(c *gin.Context) {
	count := notification.Manager.ClearAllNotifications()

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "清空所有通知成功",
		"data": gin.H{
			"cleared_count": count,
		},
	})
}

// GetNotificationStatistics 获取通知统计信息
func GetNotificationStatistics(c *gin.Context) {
	stats := notification.Manager.GetStatistics()

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "获取通知统计成功",
		"data": gin.H{
			"statistics": stats,
		},
	})
}
