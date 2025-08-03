package server

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"PushServer/internal/config"
	"PushServer/internal/logger"
	"PushServer/internal/router"
	"github.com/gin-gonic/gin"
)

type Server struct {
	httpServer *http.Server
}

// NewServer 创建新的服务器实例
func NewServer() *Server {
	return &Server{}
}

// Start 启动服务器
func (s *Server) Start() error {
	// 设置Gin模式
	gin.SetMode(config.AppConfig.Server.Mode)

	// 创建路由
	r := router.SetupRouter()

	// 创建HTTP服务器
	s.httpServer = &http.Server{
		Addr:    config.AppConfig.GetServerAddr(),
		Handler: r,
	}

	// 启动服务器的goroutine
	go func() {
		logger.Infof("服务器启动在 %s", config.AppConfig.GetServerAddr())
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Errorf("服务器启动失败: %v", err)
		}
	}()

	// 等待中断信号以优雅地关闭服务器
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("正在关闭服务器...")

	// 优雅关闭服务器，等待5秒钟完成现有请求
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.httpServer.Shutdown(ctx); err != nil {
		logger.Errorf("服务器强制关闭: %v", err)
		return err
	}

	logger.Info("服务器已关闭")
	return nil
}

// Stop 停止服务器
func (s *Server) Stop() error {
	if s.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}