package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/soarinferret/jats/internal/auth"
	"github.com/soarinferret/jats/internal/middleware"
	"github.com/soarinferret/jats/internal/models"
	"github.com/soarinferret/jats/internal/repository"
	"github.com/soarinferret/jats/internal/services"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupAuthTestDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("Failed to connect to test database")
	}

	// Migrate the schema
	err = db.AutoMigrate(
		&models.Task{},
		&models.TimeEntry{},
		&models.Comment{},
		&models.Subtask{},
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

func setupAuthTestServices() (*services.TaskService, *services.AuthService) {
	db := setupAuthTestDB()
	
	taskRepo := repository.NewTaskRepository(db)
	taskService := services.NewTaskService(taskRepo, nil)
	
	authRepo := repository.NewAuthRepository(db)
	authService := services.NewAuthService(authRepo, nil)
	
	return taskService, authService
}

func TestUserRegistration(t *testing.T) {
	taskService, authService := setupAuthTestServices()
	authHandlers := NewAuthHandlers(authService)

	t.Run("Successful registration", func(t *testing.T) {
		reqBody := RegisterRequest{
			Username: "testuser",
			Email:    "test@example.com",
			Password: "securepassword123",
		}
		
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewReader(body))
		w := httptest.NewRecorder()
		
		authHandlers.Register(w, req)
		
		if w.Code != http.StatusCreated {
			t.Errorf("Expected status %d, got %d", http.StatusCreated, w.Code)
		}
		
		var response APIResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}
		
		if !response.Success {
			t.Errorf("Expected successful response")
		}
		
		userData, ok := response.Data.(map[string]interface{})
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
			t.Error("Hashed password should not be returned")
		}
		
		if _, exists := userData["totp_secret"]; exists {
			t.Error("TOTP secret should not be returned")
		}
	})

	t.Run("Registration validation errors", func(t *testing.T) {
		testCases := []struct {
			name           string
			request        RegisterRequest
			expectedStatus int
			expectedError  string
		}{
			{"Empty username", RegisterRequest{Username: "", Email: "test@example.com", Password: "pass"}, http.StatusInternalServerError, "username is required"},
			{"Empty email", RegisterRequest{Username: "user", Email: "", Password: "pass"}, http.StatusInternalServerError, "email is required"},
			{"Empty password", RegisterRequest{Username: "user", Email: "test@example.com", Password: ""}, http.StatusInternalServerError, "password is required"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				body, _ := json.Marshal(tc.request)
				req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewReader(body))
				w := httptest.NewRecorder()
				
				authHandlers.Register(w, req)
				
				if w.Code != tc.expectedStatus {
					t.Errorf("Expected status %d, got %d", tc.expectedStatus, w.Code)
				}
			})
		}
	})

	t.Run("Duplicate user registration", func(t *testing.T) {
		// First registration
		reqBody := RegisterRequest{
			Username: "duplicate",
			Email:    "duplicate@example.com",
			Password: "password123",
		}
		
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewReader(body))
		w := httptest.NewRecorder()
		
		authHandlers.Register(w, req)
		
		if w.Code != http.StatusCreated {
			t.Errorf("First registration failed: %d", w.Code)
		}
		
		// Second registration with same username
		req = httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewReader(body))
		w = httptest.NewRecorder()
		
		authHandlers.Register(w, req)
		
		if w.Code != http.StatusConflict {
			t.Errorf("Expected status %d, got %d", http.StatusConflict, w.Code)
		}
	})

	_ = taskService // Use taskService to avoid unused variable warning
}

func TestUserLogin(t *testing.T) {
	taskService, authService := setupAuthTestServices()
	authHandlers := NewAuthHandlers(authService)
	
	// Create test user using the service
	_, err := authService.RegisterUser("logintest", "logintest@example.com", "testpassword")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	t.Run("Successful login", func(t *testing.T) {
		reqBody := LoginRequest{
			Username: "logintest",
			Password: "testpassword",
		}
		
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(body))
		w := httptest.NewRecorder()
		
		authHandlers.Login(w, req)
		
		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}
		
		var response APIResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}
		
		if !response.Success {
			t.Errorf("Expected successful response")
		}
		
		data, ok := response.Data.(map[string]interface{})
		if !ok {
			t.Fatal("Expected data to be a map")
		}
		
		// Verify user data is returned
		if _, exists := data["user"]; !exists {
			t.Error("Expected user data in response")
		}
		
		// Verify session data is returned
		if _, exists := data["session"]; !exists {
			t.Error("Expected session data in response")
		}
		
		// Verify session cookie is set
		cookies := w.Result().Cookies()
		var sessionCookie *http.Cookie
		for _, cookie := range cookies {
			if cookie.Name == "session_token" {
				sessionCookie = cookie
				break
			}
		}
		
		if sessionCookie == nil {
			t.Error("Expected session cookie to be set")
		} else if sessionCookie.Value == "" {
			t.Error("Expected session cookie to have a value")
		}
	})

	t.Run("Invalid credentials", func(t *testing.T) {
		testCases := []struct {
			name     string
			username string
			password string
		}{
			{"Wrong username", "wronguser", "testpassword"},
			{"Wrong password", "logintest", "wrongpassword"},
			{"Empty username", "", "testpassword"},
			{"Empty password", "logintest", ""},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				reqBody := LoginRequest{
					Username: tc.username,
					Password: tc.password,
				}
				
				body, _ := json.Marshal(reqBody)
				req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(body))
				w := httptest.NewRecorder()
				
				authHandlers.Login(w, req)
				
				if w.Code != http.StatusUnauthorized {
					t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
				}
			})
		}
	})

	_ = taskService // Use taskService to avoid unused variable warning
}

func TestTOTPSetup(t *testing.T) {
	taskService, authService := setupAuthTestServices()
	authHandlers := NewAuthHandlers(authService)

	// Create test user using the service
	testUser, err := authService.RegisterUser("totptest", "totptest@example.com", "testpassword")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	t.Run("TOTP setup", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/auth/totp/setup", nil)
		req = req.WithContext(setTestAuthContext(req, testUser))
		w := httptest.NewRecorder()
		
		authHandlers.SetupTOTP(w, req)
		
		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}
		
		var response APIResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}
		
		if !response.Success {
			t.Errorf("Expected successful response")
		}
		
		data, ok := response.Data.(map[string]interface{})
		if !ok {
			t.Fatal("Expected data to be a map")
		}
		
		// Verify TOTP setup fields
		if _, exists := data["secret"]; !exists {
			t.Error("Expected secret in response")
		}
		
		if _, exists := data["provisioning_uri"]; !exists {
			t.Error("Expected provisioning_uri in response")
		}
		
		if _, exists := data["qr_code_url"]; !exists {
			t.Error("Expected qr_code_url in response")
		}
	})

	t.Run("TOTP enable", func(t *testing.T) {
		// First setup TOTP
		secret, _, err := authService.SetupTOTP(testUser.ID)
		if err != nil {
			t.Fatalf("Failed to setup TOTP: %v", err)
		}
		
		// Generate valid TOTP code
		totpCode, err := auth.GenerateTOTPCode(secret)
		if err != nil {
			t.Fatalf("Failed to generate TOTP code: %v", err)
		}
		
		reqBody := map[string]string{
			"totp_code": totpCode,
		}
		
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/v1/auth/totp/enable", bytes.NewReader(body))
		req = req.WithContext(setTestAuthContext(req, testUser))
		w := httptest.NewRecorder()
		
		authHandlers.EnableTOTP(w, req)
		
		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}
		
		var response APIResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}
		
		if !response.Success {
			t.Errorf("Expected successful response")
		}
	})

	_ = taskService // Use taskService to avoid unused variable warning
}

func TestAPIKeyManagement(t *testing.T) {
	taskService, authService := setupAuthTestServices()
	authHandlers := NewAuthHandlers(authService)

	// Create test user using the service
	testUser, err := authService.RegisterUser("apikeytest", "apikeytest@example.com", "testpassword")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	t.Run("Create API key", func(t *testing.T) {
		reqBody := APIKeyRequest{
			Name:        "Test API Key",
			Permissions: []string{"tasks:read", "tasks:write"},
		}
		
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/v1/auth/api-keys", bytes.NewReader(body))
		req = req.WithContext(setTestAuthContext(req, testUser))
		w := httptest.NewRecorder()
		
		authHandlers.CreateAPIKey(w, req)
		
		if w.Code != http.StatusCreated {
			t.Errorf("Expected status %d, got %d", http.StatusCreated, w.Code)
		}
		
		var response APIResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}
		
		if !response.Success {
			t.Errorf("Expected successful response")
		}
		
		data, ok := response.Data.(map[string]interface{})
		if !ok {
			t.Fatal("Expected data to be a map")
		}
		
		// Verify API key data
		if _, exists := data["api_key"]; !exists {
			t.Error("Expected api_key in response")
		}
		
		if _, exists := data["key"]; !exists {
			t.Error("Expected key in response")
		}
		
		// Verify key is not empty
		key, ok := data["key"].(string)
		if !ok || key == "" {
			t.Error("Expected non-empty key string")
		}
	})

	t.Run("List API keys", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/auth/api-keys", nil)
		req = req.WithContext(setTestAuthContext(req, testUser))
		w := httptest.NewRecorder()
		
		authHandlers.GetAPIKeys(w, req)
		
		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}
		
		var response APIResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}
		
		if !response.Success {
			t.Errorf("Expected successful response")
		}
		
		// Verify data is an array
		if _, ok := response.Data.([]interface{}); !ok {
			t.Error("Expected data to be an array")
		}
	})

	_ = taskService // Use taskService to avoid unused variable warning
}

func TestSessionManagement(t *testing.T) {
	taskService, authService := setupAuthTestServices()
	authHandlers := NewAuthHandlers(authService)

	// Create test user using the service
	testUser, err := authService.RegisterUser("sessiontest", "sessiontest@example.com", "testpassword")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	t.Run("Get user sessions", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/auth/sessions", nil)
		req = req.WithContext(setTestAuthContext(req, testUser))
		w := httptest.NewRecorder()
		
		authHandlers.GetSessions(w, req)
		
		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}
		
		var response APIResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}
		
		if !response.Success {
			t.Errorf("Expected successful response")
		}
		
		// Verify data is an array
		if _, ok := response.Data.([]interface{}); !ok {
			t.Error("Expected data to be an array")
		}
	})

	t.Run("Logout", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/auth/logout", nil)
		w := httptest.NewRecorder()
		
		authHandlers.Logout(w, req)
		
		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}
		
		// Verify session cookie is cleared
		cookies := w.Result().Cookies()
		var sessionCookie *http.Cookie
		for _, cookie := range cookies {
			if cookie.Name == "session_token" {
				sessionCookie = cookie
				break
			}
		}
		
		if sessionCookie == nil {
			t.Error("Expected session cookie to be set")
		} else if !sessionCookie.Expires.Before(time.Now()) {
			t.Error("Expected session cookie to be expired")
		}
	})

	t.Run("Logout all sessions", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/v1/auth/sessions/all", nil)
		req = req.WithContext(setTestAuthContext(req, testUser))
		w := httptest.NewRecorder()
		
		authHandlers.LogoutAll(w, req)
		
		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}
		
		var response APIResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}
		
		if !response.Success {
			t.Errorf("Expected successful response")
		}
	})

	_ = taskService // Use taskService to avoid unused variable warning
}

// Helper function to set test auth context
func setTestAuthContext(req *http.Request, user *models.User) context.Context {
	authContext := &models.AuthContext{
		User:        user,
		Permissions: models.DefaultPermissions(),
		AuthMethod:  "session",
	}
	
	return context.WithValue(req.Context(), middleware.AuthContextKey, authContext)
}