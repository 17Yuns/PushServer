package notification

import (
	"sync"
	"time"

	"PushServer/internal/model"
)

// SystemNotification 系统通知结构
type SystemNotification struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Message   string    `json:"message"`
	Type      string    `json:"type"`
	Strategy  string    `json:"strategy"`
	Style     string    `json:"style"`
	Source    string    `json:"source"`
	TaskID    string    `json:"task_id"`
	Reason    string    `json:"reason"`
	CreatedAt time.Time `json:"created_at"`
	Status    string    `json:"status"` // unread, read
}

// NotificationManager 系统通知管理器
type NotificationManager struct {
	notifications map[string]*SystemNotification
	mutex         sync.RWMutex
	maxSize       int
}

var Manager *NotificationManager

// InitNotificationManager 初始化通知管理器
func InitNotificationManager(maxSize int) {
	Manager = &NotificationManager{
		notifications: make(map[string]*SystemNotification),
		maxSize:       maxSize,
	}
}

// AddNotification 添加系统通知
func (nm *NotificationManager) AddNotification(taskID string, req model.PushRequest, reason string) string {
	nm.mutex.Lock()
	defer nm.mutex.Unlock()

	// 生成通知ID
	notificationID := generateNotificationID()

	notification := &SystemNotification{
		ID:        notificationID,
		Title:     req.Content.Title,
		Message:   req.Content.Msg,
		Type:      req.Type,
		Strategy:  req.Strategy,
		Style:     req.Style,
		Source:    "PushServer-SystemNotification",
		TaskID:    taskID,
		Reason:    reason,
		CreatedAt: time.Now(),
		Status:    "unread",
	}

	// 如果超过最大数量，删除最旧的通知
	if len(nm.notifications) >= nm.maxSize {
		nm.removeOldestNotification()
	}

	nm.notifications[notificationID] = notification
	return notificationID
}

// GetNotification 获取单个通知
func (nm *NotificationManager) GetNotification(id string) (*SystemNotification, bool) {
	nm.mutex.RLock()
	defer nm.mutex.RUnlock()

	notification, exists := nm.notifications[id]
	return notification, exists
}

// GetAllNotifications 获取所有通知
func (nm *NotificationManager) GetAllNotifications() []*SystemNotification {
	nm.mutex.RLock()
	defer nm.mutex.RUnlock()

	notifications := make([]*SystemNotification, 0, len(nm.notifications))
	for _, notification := range nm.notifications {
		notifications = append(notifications, notification)
	}

	// 按创建时间倒序排列
	for i := 0; i < len(notifications)-1; i++ {
		for j := i + 1; j < len(notifications); j++ {
			if notifications[i].CreatedAt.Before(notifications[j].CreatedAt) {
				notifications[i], notifications[j] = notifications[j], notifications[i]
			}
		}
	}

	return notifications
}

// GetNotificationsByStatus 根据状态获取通知
func (nm *NotificationManager) GetNotificationsByStatus(status string) []*SystemNotification {
	nm.mutex.RLock()
	defer nm.mutex.RUnlock()

	var notifications []*SystemNotification
	for _, notification := range nm.notifications {
		if notification.Status == status {
			notifications = append(notifications, notification)
		}
	}

	// 按创建时间倒序排列
	for i := 0; i < len(notifications)-1; i++ {
		for j := i + 1; j < len(notifications); j++ {
			if notifications[i].CreatedAt.Before(notifications[j].CreatedAt) {
				notifications[i], notifications[j] = notifications[j], notifications[i]
			}
		}
	}

	return notifications
}

// MarkAsRead 标记通知为已读
func (nm *NotificationManager) MarkAsRead(id string) bool {
	nm.mutex.Lock()
	defer nm.mutex.Unlock()

	if notification, exists := nm.notifications[id]; exists {
		notification.Status = "read"
		return true
	}
	return false
}

// MarkAllAsRead 标记所有通知为已读
func (nm *NotificationManager) MarkAllAsRead() int {
	nm.mutex.Lock()
	defer nm.mutex.Unlock()

	count := 0
	for _, notification := range nm.notifications {
		if notification.Status == "unread" {
			notification.Status = "read"
			count++
		}
	}
	return count
}

// DeleteNotification 删除通知
func (nm *NotificationManager) DeleteNotification(id string) bool {
	nm.mutex.Lock()
	defer nm.mutex.Unlock()

	if _, exists := nm.notifications[id]; exists {
		delete(nm.notifications, id)
		return true
	}
	return false
}

// ClearAllNotifications 清空所有通知
func (nm *NotificationManager) ClearAllNotifications() int {
	nm.mutex.Lock()
	defer nm.mutex.Unlock()

	count := len(nm.notifications)
	nm.notifications = make(map[string]*SystemNotification)
	return count
}

// GetUnreadCount 获取未读通知数量
func (nm *NotificationManager) GetUnreadCount() int {
	nm.mutex.RLock()
	defer nm.mutex.RUnlock()

	count := 0
	for _, notification := range nm.notifications {
		if notification.Status == "unread" {
			count++
		}
	}
	return count
}

// GetStatistics 获取通知统计信息
func (nm *NotificationManager) GetStatistics() map[string]interface{} {
	nm.mutex.RLock()
	defer nm.mutex.RUnlock()

	stats := map[string]interface{}{
		"total":   len(nm.notifications),
		"unread":  0,
		"read":    0,
		"by_type": make(map[string]int),
	}

	for _, notification := range nm.notifications {
		if notification.Status == "unread" {
			stats["unread"] = stats["unread"].(int) + 1
		} else {
			stats["read"] = stats["read"].(int) + 1
		}

		byType := stats["by_type"].(map[string]int)
		byType[notification.Type]++
	}

	return stats
}

// removeOldestNotification 删除最旧的通知
func (nm *NotificationManager) removeOldestNotification() {
	var oldestID string
	var oldestTime time.Time

	for id, notification := range nm.notifications {
		if oldestID == "" || notification.CreatedAt.Before(oldestTime) {
			oldestID = id
			oldestTime = notification.CreatedAt
		}
	}

	if oldestID != "" {
		delete(nm.notifications, oldestID)
	}
}

// generateNotificationID 生成通知ID
func generateNotificationID() string {
	return "notify_" + time.Now().Format("20060102150405") + "_" + generateRandomString(6)
}

// generateRandomString 生成随机字符串
func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(result)
}
