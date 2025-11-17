package api

import (
	"github.com/gin-gonic/gin"
	"github.com/pg/dts/internal/api/handler"
	"github.com/pg/dts/internal/service"
)

// SetupRoutes sets up routes
func SetupRoutes(router *gin.Engine, migrationService *service.MigrationService) {
	// New API routes (according to specification)
	dts := router.Group("/dts/api")
	{
		taskHandler := handler.NewTaskHandler(migrationService)
		tasks := dts.Group("/tasks")
		{
			tasks.POST("", taskHandler.CreateTask)                      // Create and start data synchronization task
			tasks.GET("/:task_id/status", taskHandler.GetTaskStatus)   // Query synchronization task status
			tasks.POST("/:task_id/start", taskHandler.StartTask)        // Start task
			tasks.POST("/:task_id/stop", taskHandler.StopTask)         // Stop task (task remains)
			tasks.POST("/:task_id/pause", taskHandler.PauseTask)        // Pause task
			tasks.POST("/:task_id/resume", taskHandler.ResumeTask)     // Resume task
			tasks.POST("/:task_id/switch", taskHandler.SwitchTask)     // Switchover
			tasks.DELETE("/:task_id", taskHandler.DeleteTask)          // Delete task
		}
	}
}
