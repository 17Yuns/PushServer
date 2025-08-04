package main

import (
	"flag"
	"log"

	"PushServer/internal/config"
	"PushServer/internal/logger"
	"PushServer/internal/notification"
	"PushServer/internal/queue"
	"PushServer/internal/server"
	"PushServer/internal/task"
)

func main() {
	// 解析命令行参数
	var configPath string
	flag.StringVar(&configPath, "config", "config/config.yaml", "配置文件路径")
	flag.Parse()

	// 加载配置文件
	if err := config.LoadConfig(configPath); err != nil {
		log.Fatalf("加载配置文件失败: %v", err)
	}

	// 初始化日志
	if err := logger.InitLogger(); err != nil {
		log.Fatalf("初始化日志失败: %v", err)
	}

	logger.Info("消息推送服务启动中...")
	logger.Infof("配置文件: %s", configPath)
	logger.Infof("服务地址: %s", config.AppConfig.GetServerAddr())
	logger.Infof("运行模式: %s", config.AppConfig.Server.Mode)

	// 初始化任务管理器
	task.InitTaskManager(config.AppConfig.Task.CleanupInterval, config.AppConfig.Task.MaxAge)
	logger.Info("任务管理器初始化完成")

	// 初始化通知管理器
	notification.InitNotificationManager(1000)
	// 初始化队列
	queue.InitQueue()
	logger.Info("队列系统初始化完成")

	// 启动服务器
	srv := server.NewServer()
	if err := srv.Start(); err != nil {
		logger.Errorf("服务器启动失败: %v", err)
	}

	// 优雅关闭
	queue.PushQueue.Stop()
	task.Manager.Stop()
}
