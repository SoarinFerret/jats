package frontend

import (
	"html/template"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/soarinferret/jats/internal/services"
)

// AuthHandler handles authentication-related frontend requests
type AuthHandler struct {
	authService *services.AuthService
	templates   map[string]*template.Template
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(authService *services.AuthService, templates map[string]*template.Template) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		templates:   templates,
	}
}

// LoginPageHandler serves the login page
func (h *AuthHandler) LoginPageHandler(c *gin.Context) {
	// Check if user is already logged in
	if sessionToken := h.getSessionToken(c); sessionToken != "" {
		if _, err := h.authService.ValidateSession(sessionToken); err == nil {
			c.Redirect(http.StatusFound, "/")
			return
		}
	}

	c.Header("Content-Type", "text/html")
	if err := h.templates["login"].Execute(c.Writer, nil); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
}

// LoginHandler handles login form submission
func (h *AuthHandler) LoginHandler(c *gin.Context) {
	username := c.PostForm("username")
	password := c.PostForm("password")
	totpCode := c.PostForm("totp_code")

	if username == "" || password == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": map[string]string{
				"message": "Username and password are required",
			},
		})
		return
	}

	req := &services.LoginRequest{
		Username:  username,
		Password:  password,
		TOTPCode:  totpCode,
		UserAgent: c.GetHeader("User-Agent"),
		IPAddress: c.ClientIP(),
	}

	result, err := h.authService.Login(req)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error": map[string]string{
				"message": err.Error(),
			},
		})
		return
	}

	if result.RequiresTOTP {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error": map[string]string{
				"message": "TOTP verification required",
			},
		})
		return
	}

	// Set session cookie
	c.SetCookie("session_token", result.Session.Token, int(24*time.Hour.Seconds()), "/", "", false, true)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Login successful",
	})
}

// LogoutHandler handles logout
func (h *AuthHandler) LogoutHandler(c *gin.Context) {
	sessionToken := h.getSessionToken(c)
	if sessionToken != "" {
		h.authService.Logout(sessionToken)
	}

	// Clear session cookie
	c.SetCookie("session_token", "", -1, "/", "", false, true)
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// getSessionToken extracts session token from cookie
func (h *AuthHandler) getSessionToken(c *gin.Context) string {
	token, err := c.Cookie("session_token")
	if err != nil {
		return ""
	}
	return token
}