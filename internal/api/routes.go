package api

import (
	"github.com/gin-gonic/gin"
	"github.com/pg/dts/internal/api/handler"
	"github.com/pg/dts/internal/service"
)

// SetupRoutes sets up routes
func SetupRoutes(router *gin.Engine, migrationService *service.MigrationService) {
	// New API routes (according to specification)
	rdscheduler := router.Group("/rdscheduler/api")
	{
		taskHandler := handler.NewTaskHandler(migrationService)
		tasks := rdscheduler.Group("/tasks")
		{
			tasks.POST("", taskHandler.CreateTask)                 // Start data synchronization task
			tasks.GET("/:task_id", taskHandler.GetTaskStatus)      // Query synchronization task status
			tasks.POST("/:task_id/switch", taskHandler.SwitchTask) // Switchover
			tasks.DELETE("/:task_id", taskHandler.DeleteTask)      // End task
		}
	}

	// Keep old API routes (for compatibility or internal management)
	api := router.Group("/api/v1")
	{
		// Health check
		healthHandler := handler.NewHealthHandler()
		api.GET("/health", healthHandler.Check)

		// Migration tasks (internal management interface)
		migrationHandler := handler.NewMigrationHandler(migrationService)
		migrations := api.Group("/migrations")
		{
			migrations.POST("", migrationHandler.CreateTask)
			migrations.GET("", migrationHandler.ListTasks)
			migrations.GET("/:id", migrationHandler.GetTask)
			migrations.GET("/:id/status", migrationHandler.GetTaskStatus)
			migrations.POST("/:id/start", migrationHandler.StartTask)
			migrations.POST("/:id/pause", migrationHandler.PauseTask)
			migrations.POST("/:id/resume", migrationHandler.ResumeTask)
			migrations.POST("/:id/cancel", migrationHandler.CancelTask)
		}
	}
}
