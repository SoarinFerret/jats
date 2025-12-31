package frontend

import (
	"html/template"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/soarinferret/jats/internal/models"
	"github.com/soarinferret/jats/internal/services"
)

// AppHandler handles main application page requests
type AppHandler struct {
	authService *services.AuthService
	templates   map[string]*template.Template
}

// NewAppHandler creates a new app handler
func NewAppHandler(authService *services.AuthService, templates map[string]*template.Template) *AppHandler {
	return &AppHandler{
		authService: authService,
		templates:   templates,
	}
}

// AppHandler serves the main application page
func (h *AppHandler) AppHandler(c *gin.Context) {
	authContext, exists := c.Get("auth")
	if !exists {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	auth := authContext.(*models.AuthContext)
	data := gin.H{
		"User": auth.User,
	}

	c.Header("Content-Type", "text/html")
	if err := h.templates["app"].Execute(c.Writer, data); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
}