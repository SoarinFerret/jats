package routes

import (
	"net/http"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/soarinferret/jats/internal/api"
	"github.com/soarinferret/jats/internal/frontend"
	"github.com/soarinferret/jats/internal/middleware"
	"github.com/soarinferret/jats/internal/models"
	"github.com/soarinferret/jats/internal/services"
)

func SetupRoutes(taskService *services.TaskService, authService *services.AuthService) http.Handler {
	// Set Gin mode
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	// Add Gin middleware
	router.Use(gin.Recovery())
	router.Use(func(c *gin.Context) {
		// Add CORS headers
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// Initialize middleware
	authMiddleware := middleware.NewGinAuthMiddleware(authService)

	// Initialize API handlers
	taskHandlers := api.NewTaskHandlers(taskService)
	timeHandlers := api.NewTimeHandlers(taskService)
	commentHandlers := api.NewCommentHandlers(taskService)
	subtaskHandlers := api.NewSubtaskHandlers(taskService)
	tagHandlers := api.NewTagHandlers(taskService)
	searchHandlers := api.NewSearchHandlers(taskService)
	savedQueryHandlers := api.NewSavedQueryHandlers(taskService)
	authHandlers := api.NewAuthHandlers(authService)

	// Initialize frontend handlers
	frontendHandler := frontend.NewHandler(authService, taskService)

	// Load frontend templates (skip in tests)
	templatesDir := filepath.Join("frontend", "templates")
	if err := frontendHandler.LoadTemplates(templatesDir); err != nil {
		// During tests, templates might not exist - we'll handle this in the handlers
		// For now, just log the error and continue
	}

	// Frontend routes (public)
	router.GET("/login", frontendHandler.Auth.LoginPageHandler)
	router.POST("/login", frontendHandler.Auth.LoginHandler)
	router.POST("/logout", frontendHandler.Auth.LogoutHandler)

	// Frontend routes (protected)
	router.GET("/", authMiddleware.RequireAuth(), frontendHandler.App.AppHandler)

	// App routes (protected)
	appRoutes := router.Group("/app", authMiddleware.RequireAuth())
	{
		appRoutes.GET("/tasks", frontendHandler.Tasks.TaskListHandler)
		appRoutes.GET("/tasks/new", frontendHandler.Tasks.NewTaskFormHandler)
		appRoutes.POST("/tasks", frontendHandler.Tasks.CreateTaskHandler)
		appRoutes.POST("/tasks/:id/toggle-complete", frontendHandler.Tasks.TaskToggleCompleteHandler)
		appRoutes.GET("/tasks/:id/detail", frontendHandler.Tasks.TaskDetailHandler)
		appRoutes.GET("/tasks/:id/subtasks", frontendHandler.Tasks.TaskSubtasksHandler)

		// Saved queries frontend routes
		appRoutes.GET("/saved-queries", frontendHandler.Saved.SavedQueriesListHandler)
		appRoutes.GET("/saved-queries/new", frontendHandler.Saved.NewSavedQueryFormHandler)
		appRoutes.POST("/saved-queries", frontendHandler.Saved.CreateSavedQueryHandler)
		appRoutes.GET("/saved-queries/:id/tasks", frontendHandler.SavedQueryTasksHandler)

		// Comment routes
		appRoutes.POST("/tasks/:id/comments", frontendHandler.Tasks.AddTaskCommentHandler)

		// Subtask routes
		appRoutes.POST("/tasks/:id/subtasks", frontendHandler.Tasks.AddSubtaskHandler)
		appRoutes.POST("/tasks/:id/subtasks/:subtaskId/toggle", frontendHandler.Tasks.ToggleSubtaskHandler)
		appRoutes.DELETE("/tasks/:id/subtasks/:subtaskId", frontendHandler.Tasks.DeleteSubtaskHandler)

		// Attachment routes
		appRoutes.GET("/attachments/:id", frontendHandler.Attachments.ServeAttachment)
	}

	// API routes
	api := router.Group("/api/v1")
	{
		// Public authentication endpoints
		auth := api.Group("/auth")
		{
			auth.POST("/register", gin.WrapF(authHandlers.Register))
			auth.POST("/login", gin.WrapF(authHandlers.Login))
			auth.POST("/logout", gin.WrapF(authHandlers.Logout))
		}

		// Protected authentication endpoints
		authProtected := api.Group("/auth", authMiddleware.RequireAuth())
		{
			authProtected.GET("/profile", gin.WrapF(authHandlers.GetProfile))
			authProtected.POST("/totp/setup", gin.WrapF(authHandlers.SetupTOTP))
			authProtected.POST("/totp/enable", gin.WrapF(authHandlers.EnableTOTP))
			authProtected.DELETE("/totp/disable", gin.WrapF(authHandlers.DisableTOTP))
			authProtected.POST("/api-keys", gin.WrapF(authHandlers.CreateAPIKey))
			authProtected.GET("/api-keys", gin.WrapF(authHandlers.GetAPIKeys))
			authProtected.DELETE("/api-keys", gin.WrapF(authHandlers.DeleteAPIKey))
			authProtected.GET("/sessions", gin.WrapF(authHandlers.GetSessions))
			authProtected.DELETE("/sessions/all", gin.WrapF(authHandlers.LogoutAll))
		}

		// Task endpoints
		tasks := api.Group("/tasks", authMiddleware.RequirePermission(models.PermissionReadTasks))
		{
			tasks.GET("", gin.WrapF(taskHandlers.GetTasks))
			tasks.POST("", authMiddleware.RequirePermission(models.PermissionWriteTasks), gin.WrapF(taskHandlers.CreateTask))
			tasks.GET("/:id", gin.WrapF(taskHandlers.GetTask))
			tasks.PUT("/:id", authMiddleware.RequirePermission(models.PermissionWriteTasks), gin.WrapF(taskHandlers.UpdateTask))
			tasks.PATCH("/:id", authMiddleware.RequirePermission(models.PermissionWriteTasks), gin.WrapF(taskHandlers.PartialUpdateTask))
			tasks.DELETE("/:id", authMiddleware.RequirePermission(models.PermissionDeleteTasks), gin.WrapF(taskHandlers.DeleteTask))

			// Time tracking endpoints
			tasks.GET("/:id/time", authMiddleware.RequirePermission(models.PermissionReadTime), gin.WrapF(timeHandlers.GetTimeEntries))
			tasks.POST("/:id/time", authMiddleware.RequirePermission(models.PermissionWriteTime), gin.WrapF(timeHandlers.CreateTimeEntry))
			tasks.PUT("/:id/time/:timeId", authMiddleware.RequirePermission(models.PermissionWriteTime), gin.WrapF(timeHandlers.UpdateTimeEntry))
			tasks.DELETE("/:id/time/:timeId", authMiddleware.RequirePermission(models.PermissionWriteTime), gin.WrapF(timeHandlers.DeleteTimeEntry))

			// Comment endpoints
			tasks.GET("/:id/comments", gin.WrapF(commentHandlers.GetComments))
			tasks.POST("/:id/comments", authMiddleware.RequirePermission(models.PermissionWriteTasks), gin.WrapF(commentHandlers.CreateComment))
			tasks.PUT("/:id/comments/:commentId", authMiddleware.RequirePermission(models.PermissionWriteTasks), gin.WrapF(commentHandlers.UpdateComment))
			tasks.DELETE("/:id/comments/:commentId", authMiddleware.RequirePermission(models.PermissionWriteTasks), gin.WrapF(commentHandlers.DeleteComment))

			// Subtask endpoints
			tasks.GET("/:id/subtasks", gin.WrapF(subtaskHandlers.GetSubtasks))
			tasks.POST("/:id/subtasks", authMiddleware.RequirePermission(models.PermissionWriteTasks), gin.WrapF(subtaskHandlers.CreateSubtask))
			tasks.PUT("/:id/subtasks/:subtaskId", authMiddleware.RequirePermission(models.PermissionWriteTasks), gin.WrapF(subtaskHandlers.UpdateSubtask))
			tasks.PATCH("/:id/subtasks/:subtaskId/toggle", authMiddleware.RequirePermission(models.PermissionWriteTasks), gin.WrapF(subtaskHandlers.ToggleSubtask))
			tasks.DELETE("/:id/subtasks/:subtaskId", authMiddleware.RequirePermission(models.PermissionWriteTasks), gin.WrapF(subtaskHandlers.DeleteSubtask))

			// Tag endpoints for specific tasks
			tasks.POST("/:id/tags", authMiddleware.RequirePermission(models.PermissionWriteTasks), gin.WrapF(tagHandlers.AddTaskTags))
			tasks.DELETE("/:id/tags/:tag", authMiddleware.RequirePermission(models.PermissionWriteTasks), gin.WrapF(tagHandlers.RemoveTaskTag))
		}

		// Saved query endpoints
		savedQueries := api.Group("/saved-queries", authMiddleware.RequirePermission(models.PermissionReadTasks))
		{
			savedQueries.GET("", gin.WrapF(savedQueryHandlers.GetSavedQueries))
			savedQueries.POST("", authMiddleware.RequirePermission(models.PermissionWriteTasks), gin.WrapF(savedQueryHandlers.CreateSavedQuery))
			savedQueries.GET("/:id", gin.WrapF(savedQueryHandlers.GetSavedQuery))
			savedQueries.PUT("/:id", authMiddleware.RequirePermission(models.PermissionWriteTasks), gin.WrapF(savedQueryHandlers.UpdateSavedQuery))
			savedQueries.DELETE("/:id", authMiddleware.RequirePermission(models.PermissionWriteTasks), gin.WrapF(savedQueryHandlers.DeleteSavedQuery))
			savedQueries.GET("/:id/tasks", gin.WrapF(savedQueryHandlers.GetTasksBySavedQuery))
		}

		// General endpoints
		api.GET("/time", authMiddleware.RequirePermission(models.PermissionReadTime), gin.WrapF(timeHandlers.GetAllTimeEntries))
		api.GET("/tags", authMiddleware.RequirePermission(models.PermissionReadTasks), gin.WrapF(tagHandlers.GetTags))
		api.GET("/tags/:tag/tasks", authMiddleware.RequirePermission(models.PermissionReadTasks), gin.WrapF(tagHandlers.GetTasksByTag))
		api.GET("/search", authMiddleware.RequirePermission(models.PermissionReadTasks), gin.WrapF(searchHandlers.Search))
		api.GET("/kanban", authMiddleware.RequirePermission(models.PermissionReadTasks), gin.WrapF(searchHandlers.GetKanban))
		api.GET("/kanban/:tag", authMiddleware.RequirePermission(models.PermissionReadTasks), gin.WrapF(searchHandlers.GetKanbanByTag))
	}

	return router
}
