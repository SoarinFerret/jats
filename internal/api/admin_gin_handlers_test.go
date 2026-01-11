package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/soarinferret/jats/internal/auth"
	"github.com/soarinferret/jats/internal/middleware"
	"github.com/soarinferret/jats/internal/models"
	"github.com/soarinferret/jats/internal/repository"
	"github.com/soarinferret/jats/internal/services"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupGinAdminTestDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("Failed to connect to test database")
	}

	err = db.AutoMigrate(
		&models.User{},
		&models.Session{},
		&models.APIKey{},
		&models.LoginAttempt{},
	)
	if err != nil {
		panic("Failed to migrate test database")
	}

	return db
}

func setupGinAdminTestServices() (*services.AuthService, *repository.AuthRepository) {
	db := setupGinAdminTestDB()
	authRepo := repository.NewAuthRepository(db)
	authService := services.NewAuthService(authRepo, nil)
	return authService, authRepo
}

func createGinTestUser(authService *services.AuthService, username, email string, isAdmin bool) *models.User {
	user, err := authService.RegisterUser(username, email, "testpassword123")
	if err != nil {
		panic(fmt.Sprintf("Failed to create test user: %v", err))
	}
	return user
}

func createGinTestAuthContext(user *models.User, isAdmin bool) *models.AuthContext {
	permissions := models.DefaultPermissions()
	if isAdmin {
		permissions = models.AdminPermissions()
	}
	
	return &models.AuthContext{
		User:        user,
		Permissions: permissions,
		AuthMethod:  "session",
	}
}

func setupGinTestRouter(adminHandlers *GinAdminHandlers, authContext *models.AuthContext) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	
	// Add middleware that sets the auth context
	router.Use(func(c *gin.Context) {
		c.Set(middleware.AuthContextKey, authContext)
		c.Next()
	})
	
	admin := router.Group("/api/v1/admin")
	{
		admin.GET("/users", adminHandlers.GetAllUsers)
		admin.POST("/users", adminHandlers.CreateUser)
		admin.GET("/users/:id", adminHandlers.GetUser)
		admin.PUT("/users/:id", adminHandlers.UpdateUser)
		admin.DELETE("/users/:id", adminHandlers.DeleteUser)
		admin.POST("/users/:id/reset-password", adminHandlers.ResetUserPassword)
	}
	
	return router
}

func TestGinAdminHandlers_CreateUser(t *testing.T) {
	authService, authRepo := setupGinAdminTestServices()
	adminHandlers := NewGinAdminHandlers(authService, authRepo)

	t.Run("Successful user creation", func(t *testing.T) {
		adminUser := createGinTestUser(authService, "admin", "admin@test.com", true)
		authContext := createGinTestAuthContext(adminUser, true)
		router := setupGinTestRouter(adminHandlers, authContext)

		reqBody := CreateUserRequest{
			Username: "newuser",
			Email:    "newuser@test.com",
			Password: "password123",
			IsActive: true,
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/v1/admin/users", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("Expected status %d, got %d. Body: %s", http.StatusCreated, w.Code, w.Body.String())
		}

		var response map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if !response["success"].(bool) {
			t.Errorf("Expected successful response, got error: %v", response["error"])
		}

		userData, ok := response["data"].(map[string]interface{})
		if !ok {
			t.Fatal("Expected data to be a map")
		}

		if userData["username"] != reqBody.Username {
			t.Errorf("Expected username %q, got %q", reqBody.Username, userData["username"])
		}

		if userData["email"] != reqBody.Email {
			t.Errorf("Expected email %q, got %q", reqBody.Email, userData["email"])
		}

		// Verify sensitive fields are not returned
		if _, exists := userData["hashed_password"]; exists {
			t.Error("Password should not be returned in response")
		}
	})

	t.Run("Access denied for non-admin", func(t *testing.T) {
		regularUser := createGinTestUser(authService, "regular", "regular@test.com", false)
		authContext := createGinTestAuthContext(regularUser, false)
		router := setupGinTestRouter(adminHandlers, authContext)

		reqBody := CreateUserRequest{
			Username: "newuser2",
			Email:    "newuser2@test.com",
			Password: "password123",
			IsActive: true,
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/v1/admin/users", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("Expected status %d, got %d", http.StatusForbidden, w.Code)
		}
	})

	t.Run("Duplicate username error", func(t *testing.T) {
		adminUser := createGinTestUser(authService, "admin2", "admin2@test.com", true)
		authContext := createGinTestAuthContext(adminUser, true)
		router := setupGinTestRouter(adminHandlers, authContext)

		// Create first user
		createGinTestUser(authService, "duplicate", "duplicate@test.com", false)

		reqBody := CreateUserRequest{
			Username: "duplicate", // Same username
			Email:    "different@test.com",
			Password: "password123",
			IsActive: true,
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/v1/admin/users", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusConflict {
			t.Errorf("Expected status %d, got %d", http.StatusConflict, w.Code)
		}
	})
}

func TestGinAdminHandlers_GetAllUsers(t *testing.T) {
	authService, authRepo := setupGinAdminTestServices()
	adminHandlers := NewGinAdminHandlers(authService, authRepo)

	t.Run("Successful user list retrieval", func(t *testing.T) {
		adminUser := createGinTestUser(authService, "admin", "admin@test.com", true)
		createGinTestUser(authService, "user1", "user1@test.com", false)
		createGinTestUser(authService, "user2", "user2@test.com", false)

		authContext := createGinTestAuthContext(adminUser, true)
		router := setupGinTestRouter(adminHandlers, authContext)

		req := httptest.NewRequest("GET", "/api/v1/admin/users", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		var response map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if !response["success"].(bool) {
			t.Errorf("Expected successful response")
		}

		users, ok := response["data"].([]interface{})
		if !ok {
			t.Fatal("Expected data to be a slice")
		}

		if len(users) != 3 { // admin + 2 regular users
			t.Errorf("Expected 3 users, got %d", len(users))
		}

		// Check that sensitive fields are not returned
		for _, userInterface := range users {
			user := userInterface.(map[string]interface{})
			if _, exists := user["hashed_password"]; exists {
				t.Error("Password should not be returned in response")
			}
			if _, exists := user["totp_secret"]; exists {
				t.Error("TOTP secret should not be returned in response")
			}
		}
	})

	t.Run("Access denied for non-admin", func(t *testing.T) {
		regularUser := createGinTestUser(authService, "regular", "regular@test.com", false)
		authContext := createGinTestAuthContext(regularUser, false)
		router := setupGinTestRouter(adminHandlers, authContext)

		req := httptest.NewRequest("GET", "/api/v1/admin/users", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("Expected status %d, got %d", http.StatusForbidden, w.Code)
		}
	})
}

func TestGinAdminHandlers_GetUser(t *testing.T) {
	authService, authRepo := setupGinAdminTestServices()
	adminHandlers := NewGinAdminHandlers(authService, authRepo)

	t.Run("Successful user retrieval", func(t *testing.T) {
		adminUser := createGinTestUser(authService, "admin", "admin@test.com", true)
		targetUser := createGinTestUser(authService, "target", "target@test.com", false)
		authContext := createGinTestAuthContext(adminUser, true)
		router := setupGinTestRouter(adminHandlers, authContext)

		req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/admin/users/%d", targetUser.ID), nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
		}

		var response map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if !response["success"].(bool) {
			t.Errorf("Expected successful response")
		}

		userData, ok := response["data"].(map[string]interface{})
		if !ok {
			t.Fatal("Expected data to be a map")
		}

		if userData["username"] != targetUser.Username {
			t.Errorf("Expected username %q, got %q", targetUser.Username, userData["username"])
		}
	})

	t.Run("User not found", func(t *testing.T) {
		adminUser := createGinTestUser(authService, "admin3", "admin3@test.com", true)
		authContext := createGinTestAuthContext(adminUser, true)
		router := setupGinTestRouter(adminHandlers, authContext)

		req := httptest.NewRequest("GET", "/api/v1/admin/users/999", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
		}
	})
}

func TestGinAdminHandlers_UpdateUser(t *testing.T) {
	authService, authRepo := setupGinAdminTestServices()
	adminHandlers := NewGinAdminHandlers(authService, authRepo)

	t.Run("Successful user update", func(t *testing.T) {
		adminUser := createGinTestUser(authService, "admin", "admin@test.com", true)
		targetUser := createGinTestUser(authService, "target", "target@test.com", false)
		authContext := createGinTestAuthContext(adminUser, true)
		router := setupGinTestRouter(adminHandlers, authContext)

		newEmail := "updated@test.com"
		isActive := false
		reqBody := UpdateUserRequest{
			Email:    &newEmail,
			IsActive: &isActive,
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("PUT", fmt.Sprintf("/api/v1/admin/users/%d", targetUser.ID), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
		}

		var response map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if !response["success"].(bool) {
			t.Errorf("Expected successful response")
		}

		userData, ok := response["data"].(map[string]interface{})
		if !ok {
			t.Fatal("Expected data to be a map")
		}

		if userData["email"] != newEmail {
			t.Errorf("Expected email %q, got %q", newEmail, userData["email"])
		}

		if userData["is_active"] != isActive {
			t.Errorf("Expected is_active %t, got %v", isActive, userData["is_active"])
		}
	})
}

func TestGinAdminHandlers_DeleteUser(t *testing.T) {
	authService, authRepo := setupGinAdminTestServices()
	adminHandlers := NewGinAdminHandlers(authService, authRepo)

	t.Run("Successful user deletion", func(t *testing.T) {
		adminUser := createGinTestUser(authService, "admin", "admin@test.com", true)
		targetUser := createGinTestUser(authService, "target", "target@test.com", false)
		authContext := createGinTestAuthContext(adminUser, true)
		router := setupGinTestRouter(adminHandlers, authContext)

		req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/v1/admin/users/%d", targetUser.ID), nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
		}

		// Verify user was actually deleted
		deletedUser, err := authRepo.GetUserByID(targetUser.ID)
		if err == nil && deletedUser != nil {
			t.Error("User should have been deleted")
		}
	})

	t.Run("Cannot delete self", func(t *testing.T) {
		adminUser := createGinTestUser(authService, "admin2", "admin2@test.com", true)
		authContext := createGinTestAuthContext(adminUser, true)
		router := setupGinTestRouter(adminHandlers, authContext)

		req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/v1/admin/users/%d", adminUser.ID), nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})
}

func TestGinAdminHandlers_ResetUserPassword(t *testing.T) {
	authService, authRepo := setupGinAdminTestServices()
	adminHandlers := NewGinAdminHandlers(authService, authRepo)

	t.Run("Successful password reset", func(t *testing.T) {
		adminUser := createGinTestUser(authService, "admin", "admin@test.com", true)
		targetUser := createGinTestUser(authService, "target", "target@test.com", false)
		authContext := createGinTestAuthContext(adminUser, true)
		router := setupGinTestRouter(adminHandlers, authContext)

		reqBody := map[string]string{
			"new_password": "newpassword123",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/admin/users/%d/reset-password", targetUser.ID), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
		}

		// Verify password was actually changed by attempting to login with new password
		updatedUser, err := authRepo.GetUserByID(targetUser.ID)
		if err != nil {
			t.Fatalf("Failed to get updated user: %v", err)
		}

		// Verify new password works
		valid, err := auth.VerifyPassword("newpassword123", updatedUser.HashedPassword)
		if err != nil {
			t.Fatalf("Failed to verify password: %v", err)
		}
		if !valid {
			t.Error("New password should be valid")
		}

		// Verify old password doesn't work
		valid, err = auth.VerifyPassword("testpassword123", updatedUser.HashedPassword)
		if err != nil {
			t.Fatalf("Failed to verify old password: %v", err)
		}
		if valid {
			t.Error("Old password should no longer be valid")
		}
	})
}