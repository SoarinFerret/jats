package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/soarinferret/jats/internal/middleware"
	"github.com/soarinferret/jats/internal/models"
	"github.com/soarinferret/jats/internal/repository"
	"github.com/soarinferret/jats/internal/services"
)

// GinAdminHandlers handles admin-related HTTP endpoints for Gin router
type GinAdminHandlers struct {
	authService *services.AuthService
	authRepo    *repository.AuthRepository
}

// NewGinAdminHandlers creates a new gin admin handlers instance
func NewGinAdminHandlers(authService *services.AuthService, authRepo *repository.AuthRepository) *GinAdminHandlers {
	return &GinAdminHandlers{
		authService: authService,
		authRepo:    authRepo,
	}
}

// checkAdminPermission checks if the current user has admin permissions
func (h *GinAdminHandlers) checkAdminPermission(c *gin.Context) (*models.AuthContext, bool) {
	authContextRaw, exists := c.Get(middleware.AuthContextKey)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error": map[string]interface{}{
				"code":    "UNAUTHORIZED",
				"message": "Authentication required",
			},
		})
		return nil, false
	}

	authContext, ok := authContextRaw.(*models.AuthContext)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error": map[string]interface{}{
				"code":    "INVALID_AUTH_CONTEXT",
				"message": "Invalid authentication context",
			},
		})
		return nil, false
	}

	if !authContext.HasPermission(models.PermissionAdmin) {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error": map[string]interface{}{
				"code":    "INSUFFICIENT_PERMISSIONS",
				"message": "Admin permissions required",
			},
		})
		return nil, false
	}

	return authContext, true
}

// CreateUserRequest represents an admin user creation request
type CreateUserRequest struct {
	Username string `json:"username" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
	IsActive bool   `json:"is_active"`
}

// UpdateUserRequest represents an admin user update request
type UpdateUserRequest struct {
	Username *string `json:"username,omitempty"`
	Email    *string `json:"email,omitempty"`
	IsActive *bool   `json:"is_active,omitempty"`
}

// CreateUser creates a new user (admin only)
func (h *GinAdminHandlers) CreateUser(c *gin.Context) {
	// Check admin permissions
	if _, ok := h.checkAdminPermission(c); !ok {
		return
	}

	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": map[string]interface{}{
				"code":    "INVALID_REQUEST",
				"message": "Invalid request body",
				"details": err.Error(),
			},
		})
		return
	}

	// Set default active status if not provided
	if !req.IsActive && req.Password != "" {
		req.IsActive = true
	}

	user, err := h.authService.RegisterUser(req.Username, req.Email, req.Password)
	if err != nil {
		if err == services.ErrUserExists {
			c.JSON(http.StatusConflict, gin.H{
				"success": false,
				"error": map[string]interface{}{
					"code":    "USER_EXISTS",
					"message": "User already exists",
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": map[string]interface{}{
				"code":    "USER_CREATION_FAILED",
				"message": err.Error(),
			},
		})
		return
	}

	// Update active status if different from default
	if !req.IsActive {
		user.IsActive = req.IsActive
		if err := h.authRepo.UpdateUser(user); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error": map[string]interface{}{
					"code":    "USER_UPDATE_FAILED",
					"message": err.Error(),
				},
			})
			return
		}
	}

	// Remove sensitive fields before returning
	user.HashedPassword = ""
	user.TOTPSecret = ""

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    user,
		"message": "User created successfully",
	})
}

// GetAllUsers returns all users (admin only)
func (h *GinAdminHandlers) GetAllUsers(c *gin.Context) {
	// Check admin permissions
	if _, ok := h.checkAdminPermission(c); !ok {
		return
	}

	users, err := h.authRepo.GetAllUsers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": map[string]interface{}{
				"code":    "FAILED_TO_GET_USERS",
				"message": err.Error(),
			},
		})
		return
	}

	// Remove sensitive fields
	for i := range users {
		users[i].HashedPassword = ""
		users[i].TOTPSecret = ""
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    users,
		"message": "Users retrieved successfully",
	})
}

// GetUser returns a specific user (admin only)
func (h *GinAdminHandlers) GetUser(c *gin.Context) {
	// Check admin permissions
	if _, ok := h.checkAdminPermission(c); !ok {
		return
	}

	userIDStr := c.Param("id")
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": map[string]interface{}{
				"code":    "INVALID_USER_ID",
				"message": "Invalid user ID",
			},
		})
		return
	}

	user, err := h.authRepo.GetUserByID(uint(userID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": map[string]interface{}{
				"code":    "FAILED_TO_GET_USER",
				"message": err.Error(),
			},
		})
		return
	}
	if user == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error": map[string]interface{}{
				"code":    "USER_NOT_FOUND",
				"message": "User not found",
			},
		})
		return
	}

	// Remove sensitive fields
	user.HashedPassword = ""
	user.TOTPSecret = ""

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    user,
		"message": "User retrieved successfully",
	})
}

// UpdateUser updates a user (admin only)
func (h *GinAdminHandlers) UpdateUser(c *gin.Context) {
	// Check admin permissions
	if _, ok := h.checkAdminPermission(c); !ok {
		return
	}

	userIDStr := c.Param("id")
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": map[string]interface{}{
				"code":    "INVALID_USER_ID",
				"message": "Invalid user ID",
			},
		})
		return
	}

	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": map[string]interface{}{
				"code":    "INVALID_REQUEST",
				"message": "Invalid request body",
			},
		})
		return
	}

	user, err := h.authRepo.GetUserByID(uint(userID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": map[string]interface{}{
				"code":    "FAILED_TO_GET_USER",
				"message": err.Error(),
			},
		})
		return
	}
	if user == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error": map[string]interface{}{
				"code":    "USER_NOT_FOUND",
				"message": "User not found",
			},
		})
		return
	}

	// Update fields if provided
	if req.Username != nil {
		user.Username = *req.Username
	}
	if req.Email != nil {
		user.Email = *req.Email
	}
	if req.IsActive != nil {
		user.IsActive = *req.IsActive
	}

	if err := h.authRepo.UpdateUser(user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": map[string]interface{}{
				"code":    "USER_UPDATE_FAILED",
				"message": err.Error(),
			},
		})
		return
	}

	// Remove sensitive fields
	user.HashedPassword = ""
	user.TOTPSecret = ""

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    user,
		"message": "User updated successfully",
	})
}

// DeleteUser deletes a user (admin only)
func (h *GinAdminHandlers) DeleteUser(c *gin.Context) {
	// Check admin permissions
	authContext, ok := h.checkAdminPermission(c)
	if !ok {
		return
	}

	userIDStr := c.Param("id")
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": map[string]interface{}{
				"code":    "INVALID_USER_ID",
				"message": "Invalid user ID",
			},
		})
		return
	}

	// Check if user exists
	user, err := h.authRepo.GetUserByID(uint(userID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": map[string]interface{}{
				"code":    "FAILED_TO_GET_USER",
				"message": err.Error(),
			},
		})
		return
	}
	if user == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error": map[string]interface{}{
				"code":    "USER_NOT_FOUND",
				"message": "User not found",
			},
		})
		return
	}

	// Prevent self-deletion
	if authContext.User.ID == uint(userID) {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": map[string]interface{}{
				"code":    "CANNOT_DELETE_SELF",
				"message": "Cannot delete your own user account",
			},
		})
		return
	}

	if err := h.authRepo.DeleteUser(uint(userID)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": map[string]interface{}{
				"code":    "USER_DELETION_FAILED",
				"message": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    nil,
		"message": "User deleted successfully",
	})
}

// ResetUserPassword resets a user's password (admin only)
func (h *GinAdminHandlers) ResetUserPassword(c *gin.Context) {
	// Check admin permissions
	if _, ok := h.checkAdminPermission(c); !ok {
		return
	}

	userIDStr := c.Param("id")
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": map[string]interface{}{
				"code":    "INVALID_USER_ID",
				"message": "Invalid user ID",
			},
		})
		return
	}

	var req struct {
		NewPassword string `json:"new_password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": map[string]interface{}{
				"code":    "INVALID_REQUEST",
				"message": "Invalid request body",
			},
		})
		return
	}

	user, err := h.authRepo.GetUserByID(uint(userID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": map[string]interface{}{
				"code":    "FAILED_TO_GET_USER",
				"message": err.Error(),
			},
		})
		return
	}
	if user == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error": map[string]interface{}{
				"code":    "USER_NOT_FOUND",
				"message": "User not found",
			},
		})
		return
	}

	if err := h.authService.ResetPassword(user.Username, req.NewPassword); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": map[string]interface{}{
				"code":    "PASSWORD_RESET_FAILED",
				"message": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    nil,
		"message": "Password reset successfully",
	})
}