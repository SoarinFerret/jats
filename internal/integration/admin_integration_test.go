package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/soarinferret/jats/internal/models"
	"github.com/soarinferret/jats/internal/repository"
	"github.com/soarinferret/jats/internal/routes"
	"github.com/soarinferret/jats/internal/services"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// IntegrationTestSuite provides a complete test environment
type IntegrationTestSuite struct {
	db           *gorm.DB
	server       *httptest.Server
	authService  *services.AuthService
	taskService  *services.TaskService
	authRepo     *repository.AuthRepository
	adminUser    *models.User
	regularUser  *models.User
	adminToken   string
	regularToken string
}

func setupIntegrationTest() (*IntegrationTestSuite, error) {
	// Setup in-memory database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Migrate schema
	err = db.AutoMigrate(
		&models.User{},
		&models.Session{},
		&models.APIKey{},
		&models.LoginAttempt{},
		&models.Task{},
		&models.Subtask{},
		&models.TimeEntry{},
		&models.Comment{},
		&models.EmailMessage{},
		&models.TaskSubscriber{},
		&models.Attachment{},
		&models.SavedQuery{},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	// Setup repositories and services
	taskRepo := repository.NewTaskRepository(db)
	authRepo := repository.NewAuthRepository(db)
	taskService := services.NewTaskService(taskRepo, nil)
	authService := services.NewAuthService(authRepo, nil)

	// Setup test server
	handler := routes.SetupRoutes(taskService, authService, authRepo)
	server := httptest.NewServer(handler)

	suite := &IntegrationTestSuite{
		db:          db,
		server:      server,
		authService: authService,
		taskService: taskService,
		authRepo:    authRepo,
	}

	// Create test users
	if err := suite.createTestUsers(); err != nil {
		return nil, fmt.Errorf("failed to create test users: %w", err)
	}

	return suite, nil
}

func (suite *IntegrationTestSuite) createTestUsers() error {
	// Create admin user
	adminUser, err := suite.authService.RegisterUser("admin", "admin@test.com", "adminpass123")
	if err != nil {
		return err
	}
	suite.adminUser = adminUser

	// Create an admin API key with admin permissions for the admin user
	_, adminAPIKey, err := suite.authService.CreateAPIKey(adminUser.ID, "Admin API Key", models.AdminPermissions(), nil)
	if err != nil {
		return err
	}
	suite.adminToken = adminAPIKey

	// Create regular user  
	regularUser, err := suite.authService.RegisterUser("regular", "regular@test.com", "regularpass123")
	if err != nil {
		return err
	}
	suite.regularUser = regularUser

	// Create a regular API key with default permissions for the regular user
	_, regularAPIKey, err := suite.authService.CreateAPIKey(regularUser.ID, "Regular API Key", models.DefaultPermissions(), nil)
	if err != nil {
		return err
	}
	suite.regularToken = regularAPIKey

	return nil
}

func (suite *IntegrationTestSuite) loginUser(username, password string) (string, error) {
	loginReq := map[string]string{
		"username": username,
		"password": password,
	}

	body, _ := json.Marshal(loginReq)
	resp, err := http.Post(suite.server.URL+"/api/v1/auth/login", "application/json", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("login failed with status %d", resp.StatusCode)
	}

	// Extract session token from cookies
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "session_token" {
			return cookie.Value, nil
		}
	}

	return "", fmt.Errorf("no session token found in response")
}

func (suite *IntegrationTestSuite) makeAuthenticatedRequest(method, path string, body []byte, apiKey string) (*http.Response, error) {
	req, err := http.NewRequest(method, suite.server.URL+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{}
	return client.Do(req)
}

func (suite *IntegrationTestSuite) Close() {
	suite.server.Close()
}

func TestAdminUserManagementIntegration(t *testing.T) {
	suite, err := setupIntegrationTest()
	if err != nil {
		t.Fatalf("Failed to setup integration test: %v", err)
	}
	defer suite.Close()

	t.Run("Complete user management workflow", func(t *testing.T) {
		// 1. Admin creates a new user
		createReq := map[string]interface{}{
			"username":  "newuser",
			"email":     "newuser@test.com",
			"password":  "newpass123",
			"is_active": true,
		}

		createBody, _ := json.Marshal(createReq)
		resp, err := suite.makeAuthenticatedRequest("POST", "/api/v1/admin/users", createBody, suite.adminToken)
		if err != nil {
			t.Fatalf("Failed to create user: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			t.Errorf("Expected status 201, got %d", resp.StatusCode)
		}

		var createResponse map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&createResponse); err != nil {
			t.Fatalf("Failed to decode create response: %v", err)
		}

		if !createResponse["success"].(bool) {
			t.Error("Create user should be successful")
		}

		userData := createResponse["data"].(map[string]interface{})
		newUserID := userData["id"].(float64)

		// 2. Admin lists all users and verifies new user is included
		resp, err = suite.makeAuthenticatedRequest("GET", "/api/v1/admin/users", nil, suite.adminToken)
		if err != nil {
			t.Fatalf("Failed to list users: %v", err)
		}
		defer resp.Body.Close()

		var listResponse map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&listResponse); err != nil {
			t.Fatalf("Failed to decode list response: %v", err)
		}

		users := listResponse["data"].([]interface{})
		if len(users) != 3 { // admin + regular + newuser
			t.Errorf("Expected 3 users, got %d", len(users))
		}

		// Verify new user is in the list
		found := false
		for _, user := range users {
			userMap := user.(map[string]interface{})
			if userMap["username"] == "newuser" {
				found = true
				break
			}
		}
		if !found {
			t.Error("New user should be in user list")
		}

		// 3. Admin gets specific user details
		userPath := fmt.Sprintf("/api/v1/admin/users/%.0f", newUserID)
		resp, err = suite.makeAuthenticatedRequest("GET", userPath, nil, suite.adminToken)
		if err != nil {
			t.Fatalf("Failed to get user details: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200 for user details, got %d", resp.StatusCode)
		}

		// 4. Admin updates user email
		updateReq := map[string]interface{}{
			"email": "updated@test.com",
		}

		updateBody, _ := json.Marshal(updateReq)
		resp, err = suite.makeAuthenticatedRequest("PUT", userPath, updateBody, suite.adminToken)
		if err != nil {
			t.Fatalf("Failed to update user: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200 for user update, got %d", resp.StatusCode)
		}

		// 5. Verify user was updated
		resp, err = suite.makeAuthenticatedRequest("GET", userPath, nil, suite.adminToken)
		if err != nil {
			t.Fatalf("Failed to get updated user: %v", err)
		}
		defer resp.Body.Close()

		var getUserResponse map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&getUserResponse); err != nil {
			t.Fatalf("Failed to decode get user response: %v", err)
		}

		updatedUserData := getUserResponse["data"].(map[string]interface{})
		if updatedUserData["email"] != "updated@test.com" {
			t.Errorf("Expected email to be updated to 'updated@test.com', got %v", updatedUserData["email"])
		}

		// 6. Admin resets user password
		resetReq := map[string]interface{}{
			"new_password": "resetpass123",
		}

		resetBody, _ := json.Marshal(resetReq)
		resetPath := fmt.Sprintf("/api/v1/admin/users/%.0f/reset-password", newUserID)
		resp, err = suite.makeAuthenticatedRequest("POST", resetPath, resetBody, suite.adminToken)
		if err != nil {
			t.Fatalf("Failed to reset password: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200 for password reset, got %d", resp.StatusCode)
		}

		// 7. Admin deletes the user
		resp, err = suite.makeAuthenticatedRequest("DELETE", userPath, nil, suite.adminToken)
		if err != nil {
			t.Fatalf("Failed to delete user: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200 for user deletion, got %d", resp.StatusCode)
		}

		// 8. Verify user was deleted
		resp, err = suite.makeAuthenticatedRequest("GET", userPath, nil, suite.adminToken)
		if err != nil {
			t.Fatalf("Failed to check deleted user: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("Expected status 404 for deleted user, got %d", resp.StatusCode)
		}
	})

	t.Run("Regular user cannot access admin endpoints", func(t *testing.T) {
		// Try to list users as regular user
		resp, err := suite.makeAuthenticatedRequest("GET", "/api/v1/admin/users", nil, suite.regularToken)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusForbidden {
			t.Errorf("Expected status 403 for regular user accessing admin endpoint, got %d", resp.StatusCode)
		}

		// Try to create user as regular user
		createReq := map[string]interface{}{
			"username": "unauthorizeduser",
			"email":    "unauthorized@test.com",
			"password": "pass123",
		}

		createBody, _ := json.Marshal(createReq)
		resp, err = suite.makeAuthenticatedRequest("POST", "/api/v1/admin/users", createBody, suite.regularToken)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusForbidden {
			t.Errorf("Expected status 403 for regular user creating user, got %d", resp.StatusCode)
		}
	})

	t.Run("Unauthenticated requests are rejected", func(t *testing.T) {
		req, _ := http.NewRequest("GET", suite.server.URL+"/api/v1/admin/users", nil)
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("Expected status 401 for unauthenticated request, got %d", resp.StatusCode)
		}
	})

	t.Run("Admin cannot delete themselves", func(t *testing.T) {
		// Try to delete admin user using admin token
		adminPath := fmt.Sprintf("/api/v1/admin/users/%d", suite.adminUser.ID)
		resp, err := suite.makeAuthenticatedRequest("DELETE", adminPath, nil, suite.adminToken)
		if err != nil {
			t.Fatalf("Failed to make delete request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Expected status 400 for admin deleting themselves, got %d", resp.StatusCode)
		}
	})

	t.Run("Error handling for invalid data", func(t *testing.T) {
		// Try to create user with missing required fields
		createReq := map[string]interface{}{
			"username": "", // Empty username
			"email":    "invalid@test.com",
			"password": "pass123",
		}

		createBody, _ := json.Marshal(createReq)
		resp, err := suite.makeAuthenticatedRequest("POST", "/api/v1/admin/users", createBody, suite.adminToken)
		if err != nil {
			t.Fatalf("Failed to create user with invalid data: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusCreated {
			t.Error("Should not be able to create user with empty username")
		}

		// Try to get non-existent user
		resp, err = suite.makeAuthenticatedRequest("GET", "/api/v1/admin/users/99999", nil, suite.adminToken)
		if err != nil {
			t.Fatalf("Failed to get non-existent user: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("Expected status 404 for non-existent user, got %d", resp.StatusCode)
		}
	})
}

func TestAdminPermissionModelIntegration(t *testing.T) {
	suite, err := setupIntegrationTest()
	if err != nil {
		t.Fatalf("Failed to setup integration test: %v", err)
	}
	defer suite.Close()

	t.Run("Permission model consistency", func(t *testing.T) {
		// Test that admin permissions include all required permissions
		adminPerms := models.AdminPermissions()
		expectedPerms := []string{
			models.PermissionReadTasks,
			models.PermissionWriteTasks,
			models.PermissionDeleteTasks,
			models.PermissionReadTime,
			models.PermissionWriteTime,
			models.PermissionAdmin,
		}

		for _, expectedPerm := range expectedPerms {
			found := false
			for _, adminPerm := range adminPerms {
				if adminPerm == expectedPerm {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Admin permissions should include %s", expectedPerm)
			}
		}

		// Test that default permissions don't include admin
		defaultPerms := models.DefaultPermissions()
		for _, perm := range defaultPerms {
			if perm == models.PermissionAdmin {
				t.Error("Default permissions should not include admin permission")
			}
		}
	})
}

// Performance test for admin operations
func TestAdminOperationsPerformance(t *testing.T) {
	suite, err := setupIntegrationTest()
	if err != nil {
		t.Fatalf("Failed to setup integration test: %v", err)
	}
	defer suite.Close()

	t.Run("Bulk user operations performance", func(t *testing.T) {
		start := time.Now()

		// Create 10 users
		for i := 0; i < 10; i++ {
			createReq := map[string]interface{}{
				"username":  fmt.Sprintf("bulkuser%d", i),
				"email":     fmt.Sprintf("bulk%d@test.com", i),
				"password":  "bulkpass123",
				"is_active": true,
			}

			createBody, _ := json.Marshal(createReq)
			resp, err := suite.makeAuthenticatedRequest("POST", "/api/v1/admin/users", createBody, suite.adminToken)
			if err != nil {
				t.Fatalf("Failed to create bulk user %d: %v", i, err)
			}
			resp.Body.Close()
		}

		creationTime := time.Since(start)

		// List all users
		start = time.Now()
		resp, err := suite.makeAuthenticatedRequest("GET", "/api/v1/admin/users", nil, suite.adminToken)
		if err != nil {
			t.Fatalf("Failed to list users: %v", err)
		}
		resp.Body.Close()

		listTime := time.Since(start)

		t.Logf("Created 10 users in %v", creationTime)
		t.Logf("Listed all users in %v", listTime)

		// Performance thresholds (adjust based on requirements)
		if creationTime > 5*time.Second {
			t.Errorf("User creation took too long: %v", creationTime)
		}
		if listTime > 100*time.Millisecond {
			t.Errorf("User listing took too long: %v", listTime)
		}
	})
}