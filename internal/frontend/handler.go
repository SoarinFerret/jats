package frontend

import (
	"html/template"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/soarinferret/jats/internal/services"
)

// Handler coordinates all frontend request handling
type Handler struct {
	authService *services.AuthService
	taskService *services.TaskService
	templates   map[string]*template.Template

	// Sub-handlers for different areas
	Auth        *AuthHandler
	Tasks       *TaskHandler
	Saved       *SavedQueryHandler
	App         *AppHandler
	Attachments *AttachmentHandler
	Reports     *ReportHandler
}

// NewHandler creates a new frontend handler with all sub-handlers
func NewHandler(authService *services.AuthService, taskService *services.TaskService) *Handler {
	h := &Handler{
		authService: authService,
		taskService: taskService,
		templates:   make(map[string]*template.Template),
	}

	// Initialize sub-handlers (they share the same templates map)
	h.Auth = NewAuthHandler(authService, h.templates)
	h.Tasks = NewTaskHandler(taskService, h.templates)
	h.Saved = NewSavedQueryHandler(taskService, h.templates)
	h.App = NewAppHandler(authService, h.templates)
	h.Attachments = NewAttachmentHandler(taskService, "./attachments")
	h.Reports = NewReportHandler(taskService, h.templates)

	return h
}

// LoadTemplates loads all HTML templates
func (h *Handler) LoadTemplates(templatesDir string) error {
	// Load login template
	loginTmpl, err := template.ParseFiles(filepath.Join(templatesDir, "login.html"))
	if err != nil {
		return err
	}
	h.templates["login"] = loginTmpl

	// Load app template
	appTmpl, err := template.ParseFiles(filepath.Join(templatesDir, "app.html"))
	if err != nil {
		return err
	}
	h.templates["app"] = appTmpl

	// Load tasks template
	tasksTmpl, err := template.ParseFiles(filepath.Join(templatesDir, "tasks.html"))
	if err != nil {
		return err
	}
	h.templates["tasks"] = tasksTmpl

	return nil
}

// SavedQueryTasksHandler coordinates between saved query and task handlers
func (h *Handler) SavedQueryTasksHandler(c *gin.Context) {
	h.Saved.SavedQueryTasksHandler(c, h.Tasks)
}

// getSessionToken extracts session token from cookie - shared utility
func getSessionToken(c interface{}) string {
	// This will be implemented based on your gin context interface
	return ""
}