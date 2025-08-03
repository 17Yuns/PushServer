package handler

import (
	"net/http"

	"PushServer/internal/config"
	"PushServer/internal/logger"
	"PushServer/internal/model"
	"PushServer/internal/queue"
	"PushServer/internal/task"
	"github.com/gin-gonic/gin"
)

// Response 统一响应结构
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// HealthCheck 健康检查
func HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "服务运行正常",
		Data: gin.H{
			"status": "healthy",
		},
	})
}

// PushMessage 推送消息
func PushMessage(c *gin.Context) {
	var req model.PushRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Errorf("参数绑定失败: %v", err)
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "请求参数错误: " + err.Error(),
		})
		return
	}

	// 设置默认值
	req.SetDefaults()

	// 验证参数
	if err := req.Validate(); err != nil {
		logger.Errorf("参数验证失败: %v", err)
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: err.Error(),
		})
		return
	}

	// 检查接收者是否存在
	recipient, exists := config.AppConfig.GetRecipient(req.RecipientAlias)
	if !exists {
		logger.Errorf("接收者不存在: %s", req.RecipientAlias)
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "接收者不存在: " + req.RecipientAlias,
		})
		return
	}

	logger.Infof("收到推送请求: 接收者=%s, 类型=%s, 策略=%s, 标题=%s",
		req.RecipientAlias, req.Type, req.Strategy, req.Content.Title)

	// 创建任务
	newTask := task.Manager.CreateTask(req)

	// 添加到队列
	job := queue.PushJob{
		TaskID:  newTask.ID,
		Request: req,
	}

	if err := queue.PushQueue.AddJob(job); err != nil {
		logger.Errorf("添加任务到队列失败: %v", err)
		task.Manager.SetTaskError(newTask.ID, "队列已满，请稍后重试")
		c.JSON(http.StatusServiceUnavailable, Response{
			Code:    503,
			Message: "服务繁忙，请稍后重试",
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "消息推送任务已创建",
		Data: gin.H{
			"task_id":   newTask.ID,
			"recipient": recipient.Name,
			"type":      req.Type,
			"strategy":  req.Strategy,
			"style":     req.Style,
			"title":     req.Content.Title,
		},
	})
}

// GetTaskStatus 获取任务状态
func GetTaskStatus(c *gin.Context) {
	taskID := c.Param("id")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "任务ID不能为空",
		})
		return
	}

	taskInfo, exists := task.Manager.GetTask(taskID)
	if !exists {
		c.JSON(http.StatusNotFound, Response{
			Code:    404,
			Message: "任务不存在",
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "获取任务状态成功",
		Data:    taskInfo,
	})
}
