package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pg/dts/internal/api"
	"github.com/pg/dts/internal/config"
	"github.com/pg/dts/internal/logger"
	"github.com/pg/dts/internal/model"
	"github.com/pg/dts/internal/service"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var (
	// 版本信息（编译时注入）
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

func main() {
	// 加载配置（支持命令行参数，会解析所有 flag）
	cfg, flags, err := config.LoadWithFlags("configs/config.yaml")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		config.PrintUsage()
		os.Exit(1)
	}

	// 检查版本参数
	if flags.ShowVersion {
		fmt.Printf("Version: %s\n", Version)
		fmt.Printf("Build Time: %s\n", BuildTime)
		fmt.Printf("Git Commit: %s\n", GitCommit)
		os.Exit(0)
	}

	// 初始化日志
	if err := logger.Init(&cfg.Log); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	log := logger.GetLogger()
	log.WithFields(logrus.Fields{
		"version":   Version,
		"buildTime": BuildTime,
		"gitCommit": GitCommit,
	}).Info("Starting DTS server")

	// 连接元数据库（用于存储任务信息）
	log.WithFields(logrus.Fields{
		"host": cfg.Database.Host,
		"port": cfg.Database.Port,
		"db":   cfg.Database.DBName,
	}).Info("Connecting to metadata database")

	db, err := gorm.Open(postgres.Open(cfg.Database.DSN()), &gorm.Config{})
	if err != nil {
		log.WithError(err).Fatal("Failed to connect to metadata database")
	}

	// 自动迁移表结构
	log.Info("Running database migrations")
	if err := db.AutoMigrate(&model.MigrationTask{}); err != nil {
		log.WithError(err).Fatal("Failed to migrate database")
	}
	log.Info("Database migrations completed")

	// 创建服务
	migrationService := service.NewMigrationService(db)

	// 设置 Gin 模式
	if log.GetLevel() == logrus.DebugLevel {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// 设置 Gin 路由
	router := gin.New()

	// 添加中间件
	router.Use(ginLogger(log))
	router.Use(gin.Recovery())

	// 设置路由
	api.SetupRoutes(router, migrationService)

	// 启动定期清理已完成任务的 goroutine
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			if errors := migrationService.GetTaskManager().CleanupCompletedTasks(); len(errors) > 0 {
				log.WithField("errors", errors).Warn("Errors during task cleanup")
			}
		}
	}()

	// 启动服务器
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	log.WithFields(logrus.Fields{
		"host": cfg.Server.Host,
		"port": cfg.Server.Port,
		"addr": addr,
	}).Info("Starting HTTP server")

	server := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.WithError(err).Fatal("Failed to start server")
	}
}

// ginLogger 自定义 Gin 日志中间件
func ginLogger(log *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// 处理请求
		c.Next()

		// 记录日志
		latency := time.Since(start)
		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := c.Writer.Status()
		errorMessage := c.Errors.ByType(gin.ErrorTypePrivate).String()

		entry := log.WithFields(logrus.Fields{
			"status":     statusCode,
			"method":     method,
			"path":       path,
			"ip":         clientIP,
			"latency":    latency,
			"user_agent": c.Request.UserAgent(),
		})

		if raw != "" {
			entry = entry.WithField("query", raw)
		}

		if errorMessage != "" {
			entry = entry.WithField("error", errorMessage)
		}

		if statusCode >= 500 {
			entry.Error("HTTP request failed")
		} else if statusCode >= 400 {
			entry.Warn("HTTP request warning")
		} else {
			entry.Info("HTTP request")
		}
	}
}
