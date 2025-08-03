package logger

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	"PushServer/internal/config"
	"github.com/sirupsen/logrus"
)

var (
	Logger      *logrus.Logger
	ErrorLogger *logrus.Logger
	InfoLogger  *logrus.Logger
	DebugLogger *logrus.Logger
)

// InitLogger 初始化日志
func InitLogger() error {
	// 创建主日志器
	Logger = logrus.New()

	// 设置日志级别
	level, err := logrus.ParseLevel(config.AppConfig.Log.Level)
	if err != nil {
		level = logrus.InfoLevel
	}
	Logger.SetLevel(level)

	// 设置日志格式
	var formatter logrus.Formatter
	if strings.ToLower(config.AppConfig.Log.Format) == "json" {
		formatter = &logrus.JSONFormatter{
			TimestampFormat: "2006-01-02 15:04:05",
		}
	} else {
		formatter = &logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02 15:04:05",
		}
	}

	if strings.ToLower(config.AppConfig.Log.Output) == "file" {
		// 确保日志目录存在
		logDir := filepath.Dir(config.AppConfig.Log.FilePath)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return err
		}

		// 创建错误日志器
		ErrorLogger = logrus.New()
		ErrorLogger.SetLevel(logrus.ErrorLevel)
		ErrorLogger.SetFormatter(formatter)
		errorFile, err := os.OpenFile(config.AppConfig.Log.ErrorFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return err
		}
		ErrorLogger.SetOutput(errorFile)

		// 创建信息日志器
		InfoLogger = logrus.New()
		InfoLogger.SetLevel(logrus.InfoLevel)
		InfoLogger.SetFormatter(formatter)
		infoFile, err := os.OpenFile(config.AppConfig.Log.InfoFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return err
		}
		InfoLogger.SetOutput(infoFile)

		// 创建调试日志器
		DebugLogger = logrus.New()
		DebugLogger.SetLevel(logrus.DebugLevel)
		DebugLogger.SetFormatter(formatter)
		debugFile, err := os.OpenFile(config.AppConfig.Log.DebugFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return err
		}
		DebugLogger.SetOutput(debugFile)

		// 主日志器输出到所有文件
		allFile, err := os.OpenFile(config.AppConfig.Log.FilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return err
		}
		Logger.SetOutput(io.MultiWriter(allFile, os.Stdout))
		Logger.SetFormatter(formatter)
	} else {
		Logger.SetOutput(os.Stdout)
		Logger.SetFormatter(formatter)
		ErrorLogger = Logger
		InfoLogger = Logger
		DebugLogger = Logger
	}

	Logger.Info("日志系统初始化完成")
	return nil
}

// Info 记录信息日志
func Info(args ...interface{}) {
	if InfoLogger != nil {
		InfoLogger.Info(args...)
	}
	Logger.Info(args...)
}

// Infof 记录格式化信息日志
func Infof(format string, args ...interface{}) {
	if InfoLogger != nil {
		InfoLogger.Infof(format, args...)
	}
	Logger.Infof(format, args...)
}

// Error 记录错误日志
func Error(args ...interface{}) {
	if ErrorLogger != nil {
		ErrorLogger.Error(args...)
	}
	Logger.Error(args...)
}

// Errorf 记录格式化错误日志
func Errorf(format string, args ...interface{}) {
	if ErrorLogger != nil {
		ErrorLogger.Errorf(format, args...)
	}
	Logger.Errorf(format, args...)
}

// Debug 记录调试日志
func Debug(args ...interface{}) {
	if DebugLogger != nil {
		DebugLogger.Debug(args...)
	}
	Logger.Debug(args...)
}

// Debugf 记录格式化调试日志
func Debugf(format string, args ...interface{}) {
	if DebugLogger != nil {
		DebugLogger.Debugf(format, args...)
	}
	Logger.Debugf(format, args...)
}

// Warn 记录警告日志
func Warn(args ...interface{}) {
	Logger.Warn(args...)
}

// Warnf 记录格式化警告日志
func Warnf(format string, args ...interface{}) {
	Logger.Warnf(format, args...)
}
