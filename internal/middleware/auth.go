package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/soarinferret/jats/internal/common"
	"github.com/soarinferret/jats/internal/models"
	"github.com/soarinferret/jats/internal/services"
)

// AuthContextKey is used to store auth context in request context
type contextKey string

const AuthContextKey contextKey = "auth_context"

// AuthMiddleware provides authentication middleware
type AuthMiddleware struct {
	authService *services.AuthService
}

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware(authService *services.AuthService) *AuthMiddleware {
	return &AuthMiddleware{
		authService: authService,
	}
}

// RequireAuth middleware that requires authentication
func (m *AuthMiddleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authContext, err := m.authenticate(r)
		if err != nil {
			common.SendErrorResponse(w, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", err.Error(), nil)
			return
		}
		
		if authContext == nil || !authContext.IsAuthenticated() {
			common.SendErrorResponse(w, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication required", nil)
			return
		}
		
		// Add auth context to request context
		ctx := context.WithValue(r.Context(), AuthContextKey, authContext)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequirePermission middleware that requires a specific permission
func (m *AuthMiddleware) RequirePermission(permission string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authContext, err := m.authenticate(r)
			if err != nil {
				common.SendErrorResponse(w, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", err.Error(), nil)
				return
			}
			
			if authContext == nil || !authContext.IsAuthenticated() {
				common.SendErrorResponse(w, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication required", nil)
				return
			}
			
			if !authContext.HasPermission(permission) {
				common.SendErrorResponse(w, http.StatusForbidden, "INSUFFICIENT_PERMISSIONS", 
					"Insufficient permissions for this operation", map[string]interface{}{
						"required_permission": permission,
						"user_permissions":    authContext.Permissions,
					})
				return
			}
			
			// Add auth context to request context
			ctx := context.WithValue(r.Context(), AuthContextKey, authContext)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// OptionalAuth middleware that adds auth context if available but doesn't require it
func (m *AuthMiddleware) OptionalAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authContext, _ := m.authenticate(r)
		
		// Add auth context to request context (may be nil)
		ctx := context.WithValue(r.Context(), AuthContextKey, authContext)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// authenticate tries to authenticate the request using session or API key
func (m *AuthMiddleware) authenticate(r *http.Request) (*models.AuthContext, error) {
	// Try session authentication first
	if sessionToken := m.extractSessionToken(r); sessionToken != "" {
		return m.authService.ValidateSession(sessionToken)
	}
	
	// Try API key authentication
	if apiKey := m.extractAPIKey(r); apiKey != "" {
		return m.authService.ValidateAPIKey(apiKey)
	}
	
	return nil, nil
}

// extractSessionToken extracts session token from cookie or header
func (m *AuthMiddleware) extractSessionToken(r *http.Request) string {
	// Try cookie first
	if cookie, err := r.Cookie("session_token"); err == nil {
		return cookie.Value
	}
	
	// Try Authorization header with Bearer token
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		token := strings.TrimPrefix(auth, "Bearer ")
		// Simple heuristic: session tokens are typically shorter than API keys
		// API keys are 44 chars (base64 32 bytes), session tokens are 32 chars
		if len(token) < 64 {
			return token
		}
	}
	
	return ""
}

// extractAPIKey extracts API key from header
func (m *AuthMiddleware) extractAPIKey(r *http.Request) string {
	// Try X-API-Key header
	if apiKey := r.Header.Get("X-API-Key"); apiKey != "" {
		return apiKey
	}
	
	// Try Authorization header with Bearer token (longer tokens are likely API keys)
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		token := strings.TrimPrefix(auth, "Bearer ")
		// Simple heuristic: API keys are typically longer than session tokens
		// API keys are 44 chars (base64 32 bytes), session tokens are 32 chars
		if len(token) >= 64 {
			return token
		}
	}
	
	return ""
}

// GetAuthContext retrieves the auth context from the request context
func GetAuthContext(r *http.Request) *models.AuthContext {
	if authContext, ok := r.Context().Value(AuthContextKey).(*models.AuthContext); ok {
		return authContext
	}
	return nil
}

// GetCurrentUser retrieves the current user from the request context
func GetCurrentUser(r *http.Request) *models.User {
	if authContext := GetAuthContext(r); authContext != nil {
		return authContext.User
	}
	return nil
}

// HasPermission checks if the current user has a specific permission
func HasPermission(r *http.Request, permission string) bool {
	if authContext := GetAuthContext(r); authContext != nil {
		return authContext.HasPermission(permission)
	}
	return false
}

// IsAuthenticated checks if the request is authenticated
func IsAuthenticated(r *http.Request) bool {
	if authContext := GetAuthContext(r); authContext != nil {
		return authContext.IsAuthenticated()
	}
	return false
}

// CORS middleware to handle CORS headers
func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		
		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

// SecurityHeaders middleware adds security headers
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Security headers
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Content-Security-Policy", "default-src 'self'")
		
		next.ServeHTTP(w, r)
	})
}

// RateLimiting middleware (basic implementation)
func RateLimiting(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Basic rate limiting could be implemented here
		// For now, just pass through
		next.ServeHTTP(w, r)
	})
}

// Logging middleware for request logging
func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Basic request logging could be implemented here
		// For now, just pass through
		next.ServeHTTP(w, r)
	})
}

// GinAuthMiddleware provides Gin-compatible authentication middleware
type GinAuthMiddleware struct {
	authService *services.AuthService
}

// NewGinAuthMiddleware creates a new Gin authentication middleware
func NewGinAuthMiddleware(authService *services.AuthService) *GinAuthMiddleware {
	return &GinAuthMiddleware{
		authService: authService,
	}
}

// RequireAuth Gin middleware that requires authentication
func (m *GinAuthMiddleware) RequireAuth() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		authContext, err := m.authenticateGin(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": map[string]string{
					"code":    "AUTHENTICATION_REQUIRED",
					"message": err.Error(),
				},
			})
			c.Abort()
			return
		}
		
		if authContext == nil || !authContext.IsAuthenticated() {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": map[string]string{
					"code":    "AUTHENTICATION_REQUIRED",
					"message": "Authentication required",
				},
			})
			c.Abort()
			return
		}
		
		// Add auth context to Gin context
		c.Set(string(AuthContextKey), authContext)
		c.Next()
	})
}

// RequirePermission Gin middleware that requires a specific permission
func (m *GinAuthMiddleware) RequirePermission(permission string) gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		authContext, err := m.authenticateGin(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": map[string]string{
					"code":    "AUTHENTICATION_REQUIRED",
					"message": err.Error(),
				},
			})
			c.Abort()
			return
		}
		
		if authContext == nil || !authContext.IsAuthenticated() {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": map[string]string{
					"code":    "AUTHENTICATION_REQUIRED",
					"message": "Authentication required",
				},
			})
			c.Abort()
			return
		}
		
		if !authContext.HasPermission(permission) {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"error": map[string]interface{}{
					"code":                "INSUFFICIENT_PERMISSIONS",
					"message":             "Insufficient permissions for this operation",
					"required_permission": permission,
					"user_permissions":    authContext.Permissions,
				},
			})
			c.Abort()
			return
		}
		
		// Add auth context to Gin context
		c.Set(string(AuthContextKey), authContext)
		c.Next()
	})
}

// authenticateGin tries to authenticate the Gin request using session or API key
func (m *GinAuthMiddleware) authenticateGin(c *gin.Context) (*models.AuthContext, error) {
	// Try session authentication first
	if sessionToken := m.extractSessionTokenGin(c); sessionToken != "" {
		return m.authService.ValidateSession(sessionToken)
	}
	
	// Try API key authentication
	if apiKey := m.extractAPIKeyGin(c); apiKey != "" {
		return m.authService.ValidateAPIKey(apiKey)
	}
	
	return nil, nil
}

// extractSessionTokenGin extracts session token from cookie or header in Gin
func (m *GinAuthMiddleware) extractSessionTokenGin(c *gin.Context) string {
	// Try cookie first
	if sessionToken, err := c.Cookie("session_token"); err == nil {
		return sessionToken
	}
	
	// Try Authorization header with Bearer token
	auth := c.GetHeader("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		token := strings.TrimPrefix(auth, "Bearer ")
		// Simple heuristic: session tokens are typically shorter than API keys
		// API keys are 44 chars (base64 32 bytes), session tokens are 32 chars
		if len(token) < 64 {
			return token
		}
	}
	
	return ""
}

// extractAPIKeyGin extracts API key from header in Gin
func (m *GinAuthMiddleware) extractAPIKeyGin(c *gin.Context) string {
	// Try X-API-Key header
	if apiKey := c.GetHeader("X-API-Key"); apiKey != "" {
		return apiKey
	}
	
	// Try Authorization header with Bearer token (longer tokens are likely API keys)
	auth := c.GetHeader("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		token := strings.TrimPrefix(auth, "Bearer ")
		// Simple heuristic: API keys are typically longer than session tokens
		// API keys are 44 chars (base64 32 bytes), session tokens are 32 chars
		if len(token) >= 64 {
			return token
		}
	}
	
	return ""
}
