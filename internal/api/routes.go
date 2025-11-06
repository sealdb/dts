package api

import (
	"github.com/gin-gonic/gin"
	"github.com/pg/dts/internal/api/handler"
	"github.com/pg/dts/internal/service"
)

// SetupRoutes 设置路由
func SetupRoutes(router *gin.Engine, migrationService *service.MigrationService) {
	// 新的 API 路由（按照规范）
	rdscheduler := router.Group("/rdscheduler/api")
	{
		taskHandler := handler.NewTaskHandler(migrationService)
		tasks := rdscheduler.Group("/tasks")
		{
			tasks.POST("", taskHandler.CreateTask)                    // 启动数据同步任务
			tasks.GET("/:task_id", taskHandler.GetTaskStatus)         // 查询同步任务状态
			tasks.POST("/:task_id/switch", taskHandler.SwitchTask)    // 切流
			tasks.DELETE("/:task_id", taskHandler.DeleteTask)         // 结束任务
		}
	}

	// 保留旧的 API 路由（用于兼容或内部管理）
	api := router.Group("/api/v1")
	{
		// 健康检查
		healthHandler := handler.NewHealthHandler()
		api.GET("/health", healthHandler.Check)

		// 迁移任务（内部管理接口）
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
