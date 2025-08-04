package router

import (
	"PushServer/internal/handler"
	"PushServer/internal/middleware"
	"github.com/gin-gonic/gin"
)

// SetupRouter 设置路由
func SetupRouter() *gin.Engine {
	r := gin.New()

	// 添加中间件
	r.Use(middleware.Logger())
	r.Use(middleware.Recovery())
	r.Use(middleware.CORS())

	// 健康检查
	r.GET("/health", handler.HealthCheck)

	// API路由组
	api := r.Group("/api/v1")
	{
		// 消息推送接口
		api.POST("/push", handler.PushMessage)
		
		// 任务状态查询接口
		api.GET("/task/:id", handler.GetTaskStatus)
		
		// 系统通知接口
		notifications := api.Group("/notifications")
		{
			notifications.GET("", handler.GetSystemNotifications)           // 获取通知列表
			notifications.GET("/:id", handler.GetSystemNotification)        // 获取单个通知
			notifications.PUT("/:id/read", handler.MarkNotificationAsRead)  // 标记为已读
			notifications.PUT("/read-all", handler.MarkAllNotificationsAsRead) // 标记所有为已读
			notifications.DELETE("/:id", handler.DeleteSystemNotification)  // 删除通知
			notifications.DELETE("", handler.ClearAllNotifications)         // 清空所有通知
			notifications.GET("/statistics", handler.GetNotificationStatistics) // 获取统计信息
		}
	}

	return r
}
