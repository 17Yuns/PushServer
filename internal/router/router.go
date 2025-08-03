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
	}

	return r
}
