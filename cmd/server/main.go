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
	gormlogger "gorm.io/gorm/logger"
)

var (
	// Version information (injected at build time)
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

func main() {
	// Load configuration (supports command line arguments, will parse all flags)
	cfg, flags, err := config.LoadWithFlags("configs/config.yaml")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		config.PrintUsage()
		os.Exit(1)
	}

	// Check version flag
	if flags.ShowVersion {
		fmt.Printf("Version: %s\n", Version)
		fmt.Printf("Build Time: %s\n", BuildTime)
		fmt.Printf("Git Commit: %s\n", GitCommit)
		os.Exit(0)
	}

	// Initialize logger
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

	// Connect to metadata database (for storing task information)
	log.WithFields(logrus.Fields{
		"host": cfg.Database.Host,
		"port": cfg.Database.Port,
		"db":   cfg.Database.DBName,
	}).Info("Connecting to metadata database")

	db, err := gorm.Open(postgres.Open(cfg.Database.DSN()), &gorm.Config{
		PrepareStmt: false, // Disable prepared statements to avoid "insufficient arguments" error
		Logger:      gormlogger.Default.LogMode(gormlogger.Info), // TODO: need to delete
	})
	if err != nil {
		log.WithError(err).Fatal("Failed to connect to metadata database")
	}

	// Auto migrate table structure (ensure tables exist)
	log.Info("Initializing database schema")
	migrator := db.Migrator()
	if !migrator.HasTable(&model.MigrationTask{}) {
		if err := migrator.CreateTable(&model.MigrationTask{}); err != nil {
			log.WithError(err).Fatal("Failed to create migration_tasks table")
		}
		log.Info("Created migration_tasks table")
	} else {
		// Table exists, use Migrator().AutoMigrate() to update schema if needed
		// This should avoid triggering AfterFind hook during schema queries
		if err := migrator.AutoMigrate(&model.MigrationTask{}); err != nil {
			log.WithError(err).WithField("error_type", fmt.Sprintf("%T", err)).Fatal("Failed to migrate database schema")
		}
		log.Info("Updated migration_tasks table schema")
	}
	log.Info("Database schema initialized")

	// Create service
	migrationService := service.NewMigrationService(db)

	// Set Gin mode
	if log.GetLevel() == logrus.DebugLevel {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// Set up Gin routes
	router := gin.New()

	// Add middleware
	router.Use(ginLogger(log))
	router.Use(gin.Recovery())

	// Set up routes
	api.SetupRoutes(router, migrationService)

	// Start goroutine to periodically clean up completed tasks
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			if errors := migrationService.GetTaskManager().CleanupCompletedTasks(); len(errors) > 0 {
				log.WithField("errors", errors).Warn("Errors during task cleanup")
			}
		}
	}()

	// Start server
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

// ginLogger is a custom Gin logging middleware
func ginLogger(log *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Log request
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
