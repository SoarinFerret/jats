package routes

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/soarinferret/jats/internal/api"
	"github.com/soarinferret/jats/internal/middleware"
	"github.com/soarinferret/jats/internal/models"
	"github.com/soarinferret/jats/internal/repository"
	"github.com/soarinferret/jats/internal/services"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type TestData struct {
	Handler     http.Handler
	TaskService *services.TaskService
	AuthService *services.AuthService
	TestUser    *models.User
	APIKey      string
}

func setupTestAPI(t *testing.T) *TestData {
	// Setup in-memory database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Auto-migrate
	err = db.AutoMigrate(
		&models.Task{},
		&models.Subtask{},
		&models.TimeEntry{},
		&models.Comment{},
		&models.TaskSubscriber{},
		&models.Attachment{},
		&models.User{},
		&models.Session{},
		&models.APIKey{},
		&models.LoginAttempt{},
	)
	if err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	// Setup services
	taskRepo := repository.NewTaskRepository(db)
	taskService := services.NewTaskService(taskRepo, nil)

	authRepo := repository.NewAuthRepository(db)
	authService := services.NewAuthService(authRepo, nil)

	// Create test user
	testUser, err := authService.RegisterUser("testuser", "test@example.com", "testpassword")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Create API key for authentication with full permissions including delete
	permissions := append(models.DefaultPermissions(), models.PermissionDeleteTasks)
	_, apiKey, err := authService.CreateAPIKey(testUser.ID, "Test API Key", permissions, nil)
	if err != nil {
		t.Fatalf("Failed to create API key: %v", err)
	}

	// Setup routes
	handler := SetupRoutes(taskService, authService)

	return &TestData{
		Handler:     handler,
		TaskService: taskService,
		AuthService: authService,
		TestUser:    testUser,
		APIKey:      apiKey,
	}
}

// addAuthHeader adds authentication header to a request
func addAuthHeader(req *http.Request, apiKey string) {
	req.Header.Set("X-API-Key", apiKey)
}

// addAuthContext adds authentication context to a request for middleware tests
func addAuthContext(req *http.Request, user *models.User) *http.Request {
	authContext := &models.AuthContext{
		User:        user,
		Permissions: models.DefaultPermissions(),
		AuthMethod:  "test",
	}

	ctx := context.WithValue(req.Context(), middleware.AuthContextKey, authContext)
	return req.WithContext(ctx)
}

// newAuthenticatedRequest creates a new HTTP request with authentication header
func newAuthenticatedRequest(method, url string, body io.Reader, apiKey string) *http.Request {
	req := httptest.NewRequest(method, url, body)
	addAuthHeader(req, apiKey)
	return req
}

func TestCreateTask(t *testing.T) {
	testData := setupTestAPI(t)

	// Create task request
	taskReq := api.TaskRequest{
		Name:        "Test Task",
		Description: "Test Description",
		Status:      models.TaskStatusOpen,
		Priority:    models.TaskPriorityMedium,
		Tags:        []string{"test", "api"},
	}

	reqBody, _ := json.Marshal(taskReq)
	req := httptest.NewRequest("POST", "/api/v1/tasks", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	addAuthHeader(req, testData.APIKey)

	w := httptest.NewRecorder()
	testData.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status %d, got %d", http.StatusCreated, w.Code)
		t.Logf("Response body: %s", w.Body.String())
	}

	var response api.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if !response.Success {
		t.Errorf("Expected success=true, got %v", response.Success)
	}

	// Verify task data content
	if response.Data == nil {
		t.Fatal("Expected task data in response")
	}

	// Convert to map for easier access
	taskData, ok := response.Data.(map[string]interface{})
	if !ok {
		t.Fatal("Expected task data to be a map")
	}

	// Verify all task fields
	if taskData["name"] != taskReq.Name {
		t.Errorf("Expected name %q, got %q", taskReq.Name, taskData["name"])
	}

	if taskData["description"] != taskReq.Description {
		t.Errorf("Expected description %q, got %q", taskReq.Description, taskData["description"])
	}

	if taskData["status"] != string(taskReq.Status) {
		t.Errorf("Expected status %q, got %q", taskReq.Status, taskData["status"])
	}

	if taskData["priority"] != string(taskReq.Priority) {
		t.Errorf("Expected priority %q, got %q", taskReq.Priority, taskData["priority"])
	}

	// Verify tags array
	tagsInterface, ok := taskData["tags"].([]interface{})
	if !ok {
		t.Error("Expected tags to be an array")
	} else {
		if len(tagsInterface) != len(taskReq.Tags) {
			t.Errorf("Expected %d tags, got %d", len(taskReq.Tags), len(tagsInterface))
		} else {
			for i, tagInterface := range tagsInterface {
				if tag, ok := tagInterface.(string); ok {
					if tag != taskReq.Tags[i] {
						t.Errorf("Expected tag %q at index %d, got %q", taskReq.Tags[i], i, tag)
					}
				}
			}
		}
	}

	// Verify ID is set (should be > 0)
	if taskID, ok := taskData["id"].(float64); !ok || taskID <= 0 {
		t.Error("Expected task ID to be set and greater than 0")
	}

	// Verify timestamps are present
	if taskData["created_at"] == nil {
		t.Error("Expected created_at timestamp to be present")
	}
	if taskData["updated_at"] == nil {
		t.Error("Expected updated_at timestamp to be present")
	}

	// Verify message
	if response.Message == "" {
		t.Error("Expected success message to be present")
	}
}

func TestGetTasks(t *testing.T) {
	testData := setupTestAPI(t)

	// Create multiple test tasks
	expectedTasks := []struct {
		name   string
		status models.TaskStatus
		tags   []string
	}{
		{"Test Task 1", models.TaskStatusOpen, []string{"urgent", "test"}},
		{"Test Task 2", models.TaskStatusInProgress, []string{"normal", "feature"}},
		{"Test Task 3", models.TaskStatusResolved, []string{"urgent", "bugfix"}},
	}

	createdTasks := make([]*models.Task, len(expectedTasks))
	for i, expected := range expectedTasks {
		task, err := testData.TaskService.CreateTask(expected.name)
		if err != nil {
			t.Fatalf("Failed to create test task %d: %v", i+1, err)
		}
		task.Status = expected.status
		task.Tags = expected.tags
		if err := testData.TaskService.UpdateTask(task); err != nil {
			t.Fatalf("Failed to update test task %d: %v", i+1, err)
		}
		createdTasks[i] = task
	}

	req := httptest.NewRequest("GET", "/api/v1/tasks", nil)
	addAuthHeader(req, testData.APIKey)
	w := httptest.NewRecorder()
	testData.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		t.Logf("Response body: %s", w.Body.String())
	}

	var response api.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if !response.Success {
		t.Errorf("Expected success=true, got %v", response.Success)
	}

	// Verify response structure for paginated data
	paginatedData, ok := response.Data.(map[string]interface{})
	if !ok {
		t.Fatal("Expected paginated response structure")
	}

	// Check pagination metadata
	pagination, ok := paginatedData["pagination"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected pagination metadata")
	}

	totalItems := int(pagination["total"].(float64))
	if totalItems != len(expectedTasks) {
		t.Errorf("Expected %d total items, got %d", len(expectedTasks), totalItems)
	}

	// Check items array
	itemsInterface, ok := paginatedData["items"].([]interface{})
	if !ok {
		t.Fatal("Expected items to be an array")
	}

	if len(itemsInterface) != len(expectedTasks) {
		t.Errorf("Expected %d tasks in response, got %d", len(expectedTasks), len(itemsInterface))
	}

	// Verify each task content
	for i, itemInterface := range itemsInterface {
		taskData, ok := itemInterface.(map[string]interface{})
		if !ok {
			t.Errorf("Expected task %d to be a map", i)
			continue
		}

		// Find corresponding created task
		var expectedTask *models.Task
		taskName := taskData["name"].(string)
		for _, created := range createdTasks {
			if created.Name == taskName {
				expectedTask = created
				break
			}
		}

		if expectedTask == nil {
			t.Errorf("Could not find expected task for %q", taskName)
			continue
		}

		// Verify task data matches what we created
		if taskData["status"] != string(expectedTask.Status) {
			t.Errorf("Task %q: expected status %q, got %q", taskName, expectedTask.Status, taskData["status"])
		}

		// Verify tags
		tagsInterface, ok := taskData["tags"].([]interface{})
		if !ok {
			t.Errorf("Task %q: expected tags to be an array", taskName)
		} else if len(tagsInterface) != len(expectedTask.Tags) {
			t.Errorf("Task %q: expected %d tags, got %d", taskName, len(expectedTask.Tags), len(tagsInterface))
		}
	}
}

func TestGetTask(t *testing.T) {
	testData := setupTestAPI(t)

	// Create a test task with specific data
	originalTask, err := testData.TaskService.CreateTask("Specific Test Task")
	if err != nil {
		t.Fatalf("Failed to create test task: %v", err)
	}

	// Update task with additional details
	originalTask.Description = "Detailed task description"
	originalTask.Status = models.TaskStatusInProgress
	originalTask.Priority = models.TaskPriorityHigh
	originalTask.Tags = []string{"important", "feature", "client1"}

	if err := testData.TaskService.UpdateTask(originalTask); err != nil {
		t.Fatalf("Failed to update test task: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/v1/tasks/1", nil)
	req.SetPathValue("id", "1")
	addAuthHeader(req, testData.APIKey)

	w := httptest.NewRecorder()
	testData.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		t.Logf("Response body: %s", w.Body.String())
	}

	var response api.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if !response.Success {
		t.Errorf("Expected success=true, got %v", response.Success)
	}

	// Verify complete task content
	if response.Data == nil {
		t.Fatal("Expected task data in response")
	}

	taskData, ok := response.Data.(map[string]interface{})
	if !ok {
		t.Fatal("Expected task data to be a map")
	}

	// Verify all fields match
	if taskData["id"].(float64) != float64(originalTask.ID) {
		t.Errorf("Expected task ID %d, got %v", originalTask.ID, taskData["id"])
	}

	if taskData["name"] != originalTask.Name {
		t.Errorf("Expected name %q, got %q", originalTask.Name, taskData["name"])
	}

	if taskData["description"] != originalTask.Description {
		t.Errorf("Expected description %q, got %q", originalTask.Description, taskData["description"])
	}

	if taskData["status"] != string(originalTask.Status) {
		t.Errorf("Expected status %q, got %q", originalTask.Status, taskData["status"])
	}

	if taskData["priority"] != string(originalTask.Priority) {
		t.Errorf("Expected priority %q, got %q", originalTask.Priority, taskData["priority"])
	}

	// Verify tags array content exactly
	tagsInterface, ok := taskData["tags"].([]interface{})
	if !ok {
		t.Fatal("Expected tags to be an array")
	}

	if len(tagsInterface) != len(originalTask.Tags) {
		t.Errorf("Expected %d tags, got %d", len(originalTask.Tags), len(tagsInterface))
	}

	// Create a map to check tag existence (order might vary)
	expectedTags := make(map[string]bool)
	for _, tag := range originalTask.Tags {
		expectedTags[tag] = true
	}

	for _, tagInterface := range tagsInterface {
		if tag, ok := tagInterface.(string); ok {
			if !expectedTags[tag] {
				t.Errorf("Unexpected tag %q in response", tag)
			}
			delete(expectedTags, tag)
		}
	}

	// Check if any expected tags were missing
	for missingTag := range expectedTags {
		t.Errorf("Expected tag %q was missing from response", missingTag)
	}

	// Verify timestamps
	if taskData["created_at"] == nil {
		t.Error("Expected created_at timestamp")
	}
	if taskData["updated_at"] == nil {
		t.Error("Expected updated_at timestamp")
	}
}

func TestCreateTaskValidation(t *testing.T) {
	testData := setupTestAPI(t)

	testCases := []struct {
		name           string
		request        api.TaskRequest
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "Empty name",
			request:        api.TaskRequest{Name: ""},
			expectedStatus: http.StatusUnprocessableEntity,
			expectedError:  "VALIDATION_ERROR",
		},
		{
			name:           "Whitespace only name",
			request:        api.TaskRequest{Name: "   "},
			expectedStatus: http.StatusUnprocessableEntity,
			expectedError:  "VALIDATION_ERROR",
		},
		{
			name:           "Invalid status",
			request:        api.TaskRequest{Name: "Valid Task", Status: "invalid-status"},
			expectedStatus: http.StatusUnprocessableEntity,
			expectedError:  "VALIDATION_ERROR",
		},
		{
			name:           "Invalid priority",
			request:        api.TaskRequest{Name: "Valid Task", Priority: "invalid-priority"},
			expectedStatus: http.StatusUnprocessableEntity,
			expectedError:  "VALIDATION_ERROR",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reqBody, _ := json.Marshal(tc.request)
			req := httptest.NewRequest("POST", "/api/v1/tasks", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			addAuthHeader(req, testData.APIKey)

			w := httptest.NewRecorder()
			testData.Handler.ServeHTTP(w, req)

			if w.Code != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d", tc.expectedStatus, w.Code)
				t.Logf("Response body: %s", w.Body.String())
			}

			var response api.APIResponse
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("Failed to unmarshal response: %v", err)
			}

			if response.Success {
				t.Error("Expected success=false for validation error")
			}

			if response.Error == nil {
				t.Fatal("Expected error details in response")
			}

			if response.Error.Code != tc.expectedError {
				t.Errorf("Expected error code %q, got %q", tc.expectedError, response.Error.Code)
			}

			if response.Error.Message == "" {
				t.Error("Expected error message to be present")
			}

			// Verify error details contain validation information
			if response.Error.Details == nil {
				t.Error("Expected validation details to be present")
			}
		})
	}
}

func TestTaskFiltering(t *testing.T) {
	testData := setupTestAPI(t)

	// Create test tasks with different statuses, priorities, and tags
	testTasks := []struct {
		name     string
		status   models.TaskStatus
		priority models.TaskPriority
		tags     []string
	}{
		{"Open Urgent Task", models.TaskStatusOpen, models.TaskPriorityHigh, []string{"urgent", "client1"}},
		{"In Progress Task", models.TaskStatusInProgress, models.TaskPriorityMedium, []string{"client1", "feature"}},
		{"Resolved Task", models.TaskStatusResolved, models.TaskPriorityLow, []string{"client2", "bugfix"}},
		{"Another Open Task", models.TaskStatusOpen, models.TaskPriorityMedium, []string{"client2"}},
	}

	createdTasks := make([]*models.Task, len(testTasks))
	for i, taskData := range testTasks {
		task, err := testData.TaskService.CreateTask(taskData.name)
		if err != nil {
			t.Fatalf("Failed to create task %d: %v", i+1, err)
		}
		task.Status = taskData.status
		task.Priority = taskData.priority
		task.Tags = taskData.tags
		if err := testData.TaskService.UpdateTask(task); err != nil {
			t.Fatalf("Failed to update task %d: %v", i+1, err)
		}
		createdTasks[i] = task
	}

	t.Run("Filter by status", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/tasks?status=open", nil)
		addAuthHeader(req, testData.APIKey)
		w := httptest.NewRecorder()
		testData.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		var response api.APIResponse
		json.Unmarshal(w.Body.Bytes(), &response)

		paginatedData := response.Data.(map[string]interface{})
		items := paginatedData["items"].([]interface{})

		// Should only return "open" status tasks
		expectedOpenTasks := 2 // "Open Urgent Task" and "Another Open Task"
		if len(items) != expectedOpenTasks {
			t.Errorf("Expected %d open tasks, got %d", expectedOpenTasks, len(items))
		}

		// Verify all returned tasks have "open" status
		for _, item := range items {
			taskData := item.(map[string]interface{})
			if taskData["status"] != string(models.TaskStatusOpen) {
				t.Errorf("Expected task status 'open', got %q", taskData["status"])
			}
		}
	})

	t.Run("Filter by tag", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/tasks?tags=client1", nil)
		addAuthHeader(req, testData.APIKey)
		w := httptest.NewRecorder()
		testData.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		var response api.APIResponse
		json.Unmarshal(w.Body.Bytes(), &response)

		paginatedData := response.Data.(map[string]interface{})
		items := paginatedData["items"].([]interface{})

		// Should return tasks with "client1" tag
		expectedClient1Tasks := 2 // "Open Urgent Task" and "In Progress Task"
		if len(items) != expectedClient1Tasks {
			t.Errorf("Expected %d client1 tasks, got %d", expectedClient1Tasks, len(items))
		}

		// Verify all returned tasks have "client1" tag
		for _, item := range items {
			taskData := item.(map[string]interface{})
			tags := taskData["tags"].([]interface{})

			hasClient1Tag := false
			for _, tag := range tags {
				if tag.(string) == "client1" {
					hasClient1Tag = true
					break
				}
			}
			if !hasClient1Tag {
				t.Errorf("Task %q should have 'client1' tag", taskData["name"])
			}
		}
	})

	t.Run("Filter by priority", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/tasks?priority=high", nil)
		addAuthHeader(req, testData.APIKey)
		w := httptest.NewRecorder()
		testData.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		var response api.APIResponse
		json.Unmarshal(w.Body.Bytes(), &response)

		paginatedData := response.Data.(map[string]interface{})
		items := paginatedData["items"].([]interface{})

		// Should return only high priority tasks
		expectedHighPriorityTasks := 1 // "Open Urgent Task"
		if len(items) != expectedHighPriorityTasks {
			t.Errorf("Expected %d high priority tasks, got %d", expectedHighPriorityTasks, len(items))
		}

		for _, item := range items {
			taskData := item.(map[string]interface{})
			if taskData["priority"] != string(models.TaskPriorityHigh) {
				t.Errorf("Expected task priority 'high', got %q", taskData["priority"])
			}
		}
	})

	t.Run("Combined filters", func(t *testing.T) {
		// Filter by status=open AND priority=medium
		req := httptest.NewRequest("GET", "/api/v1/tasks?status=open&priority=medium", nil)
		addAuthHeader(req, testData.APIKey)
		w := httptest.NewRecorder()
		testData.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		var response api.APIResponse
		json.Unmarshal(w.Body.Bytes(), &response)

		paginatedData := response.Data.(map[string]interface{})
		items := paginatedData["items"].([]interface{})

		// Should return only "Another Open Task" (open + medium priority)
		expectedTasks := 1
		if len(items) != expectedTasks {
			t.Errorf("Expected %d tasks matching open+medium, got %d", expectedTasks, len(items))
		}

		for _, item := range items {
			taskData := item.(map[string]interface{})
			if taskData["status"] != string(models.TaskStatusOpen) {
				t.Errorf("Expected task status 'open', got %q", taskData["status"])
			}
			if taskData["priority"] != string(models.TaskPriorityMedium) {
				t.Errorf("Expected task priority 'medium', got %q", taskData["priority"])
			}
		}
	})

	t.Run("Search filter", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/tasks?search=urgent", nil)
		addAuthHeader(req, testData.APIKey)
		w := httptest.NewRecorder()
		testData.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		var response api.APIResponse
		json.Unmarshal(w.Body.Bytes(), &response)

		paginatedData := response.Data.(map[string]interface{})
		items := paginatedData["items"].([]interface{})

		// Should find "Open Urgent Task"
		expectedTasks := 1
		if len(items) != expectedTasks {
			t.Errorf("Expected %d tasks matching 'urgent', got %d", expectedTasks, len(items))
		}

		if len(items) > 0 {
			taskData := items[0].(map[string]interface{})
			taskName := taskData["name"].(string)
			if taskName != "Open Urgent Task" {
				t.Errorf("Expected to find 'Open Urgent Task', got %q", taskName)
			}
		}
	})
}

func TestTimeEntryEndpoints(t *testing.T) {
	testData := setupTestAPI(t)

	// Create a test task first
	task, err := testData.TaskService.CreateTask("Test Task for Time Tracking")
	if err != nil {
		t.Fatalf("Failed to create test task: %v", err)
	}

	// Test creating time entry
	timeReq := api.TimeEntryRequest{
		Description: "Working on task implementation",
		Duration:    120, // 2 hours
	}

	reqBody, _ := json.Marshal(timeReq)
	req := httptest.NewRequest("POST", "/api/v1/tasks/1/time", bytes.NewBuffer(reqBody))
	req.SetPathValue("id", "1")
	req.Header.Set("Content-Type", "application/json")
	addAuthHeader(req, testData.APIKey)

	w := httptest.NewRecorder()
	testData.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status %d, got %d", http.StatusCreated, w.Code)
		t.Logf("Response body: %s", w.Body.String())
	}

	// Verify time entry creation response
	var response api.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if !response.Success {
		t.Error("Expected success=true")
	}

	timeEntryData, ok := response.Data.(map[string]interface{})
	if !ok {
		t.Fatal("Expected time entry data to be a map")
	}

	// Verify time entry content
	if timeEntryData["description"] != timeReq.Description {
		t.Errorf("Expected description %q, got %q", timeReq.Description, timeEntryData["description"])
	}

	if int(timeEntryData["duration"].(float64)) != timeReq.Duration {
		t.Errorf("Expected duration %d, got %v", timeReq.Duration, timeEntryData["duration"])
	}

	if int(timeEntryData["task_id"].(float64)) != int(task.ID) {
		t.Errorf("Expected task_id %d, got %v", task.ID, timeEntryData["task_id"])
	}

	// Verify timestamps are set
	if timeEntryData["created_at"] == nil {
		t.Error("Expected created_at timestamp")
	}
	if timeEntryData["updated_at"] == nil {
		t.Error("Expected updated_at timestamp")
	}

	// Test getting time entries
	req = httptest.NewRequest("GET", "/api/v1/tasks/1/time", nil)
	req.SetPathValue("id", "1")
	addAuthHeader(req, testData.APIKey)

	w = httptest.NewRecorder()
	testData.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	// For now, the API returns empty array since repository method isn't fully implemented
	// But we can verify the structure
	var getResponse api.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &getResponse); err != nil {
		t.Fatalf("Failed to unmarshal get response: %v", err)
	}

	if !getResponse.Success {
		t.Error("Expected success=true for get time entries")
	}

	// Test validation for time entries
	t.Run("Invalid time entry validation", func(t *testing.T) {
		invalidTimeReq := api.TimeEntryRequest{
			Description: "Valid description",
			Duration:    -30, // Invalid negative duration
		}

		reqBody, _ := json.Marshal(invalidTimeReq)
		req := httptest.NewRequest("POST", "/api/v1/tasks/1/time", bytes.NewBuffer(reqBody))
		req.SetPathValue("id", "1")
		req.Header.Set("Content-Type", "application/json")
		addAuthHeader(req, testData.APIKey)

		w := httptest.NewRecorder()
		testData.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusUnprocessableEntity {
			t.Errorf("Expected status %d for invalid duration, got %d", http.StatusUnprocessableEntity, w.Code)
		}

		var errorResponse api.APIResponse
		json.Unmarshal(w.Body.Bytes(), &errorResponse)

		if errorResponse.Success {
			t.Error("Expected success=false for validation error")
		}
		if errorResponse.Error == nil {
			t.Error("Expected error details")
		}
	})

	// Test time entry for non-existent task
	t.Run("Time entry for non-existent task", func(t *testing.T) {
		reqBody, _ := json.Marshal(timeReq)
		req := httptest.NewRequest("POST", "/api/v1/tasks/999/time", bytes.NewBuffer(reqBody))
		req.SetPathValue("id", "999")
		req.Header.Set("Content-Type", "application/json")
		addAuthHeader(req, testData.APIKey)

		w := httptest.NewRecorder()
		testData.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status %d for non-existent task, got %d", http.StatusNotFound, w.Code)
		}

		var errorResponse api.APIResponse
		json.Unmarshal(w.Body.Bytes(), &errorResponse)

		if errorResponse.Success {
			t.Error("Expected success=false for non-existent task")
		}
	})
}

func TestTagEndpoints(t *testing.T) {
	testData := setupTestAPI(t)

	// Create test tasks with tags
	task1, _ := testData.TaskService.CreateTask("Task 1")
	task1.Tags = []string{"urgent", "client1"}
	testData.TaskService.UpdateTask(task1)

	task2, _ := testData.TaskService.CreateTask("Task 2")
	task2.Tags = []string{"client1", "feature"}
	testData.TaskService.UpdateTask(task2)

	// Test getting all tags
	req := httptest.NewRequest("GET", "/api/v1/tags", nil)
	addAuthHeader(req, testData.APIKey)
	w := httptest.NewRecorder()
	testData.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response api.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if !response.Success {
		t.Errorf("Expected success=true, got %v", response.Success)
	}

	// Test getting tasks by tag
	req = httptest.NewRequest("GET", "/api/v1/tags/client1/tasks", nil)
	req.SetPathValue("tag", "client1")
	addAuthHeader(req, testData.APIKey)

	w = httptest.NewRecorder()
	testData.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestKanbanEndpoint(t *testing.T) {
	testData := setupTestAPI(t)

	// Create test tasks with different statuses
	testTasks := []struct {
		name   string
		status models.TaskStatus
	}{
		{"Open Task 1", models.TaskStatusOpen},
		{"Open Task 2", models.TaskStatusOpen},
		{"In Progress Task", models.TaskStatusInProgress},
		{"Resolved Task", models.TaskStatusResolved},
	}

	for _, taskData := range testTasks {
		task, _ := testData.TaskService.CreateTask(taskData.name)
		task.Status = taskData.status
		testData.TaskService.UpdateTask(task)
	}

	// Test kanban endpoint
	req := httptest.NewRequest("GET", "/api/v1/kanban", nil)
	addAuthHeader(req, testData.APIKey)
	w := httptest.NewRecorder()
	testData.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		t.Logf("Response body: %s", w.Body.String())
	}

	var response api.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if !response.Success {
		t.Errorf("Expected success=true, got %v", response.Success)
	}

	// Verify kanban structure
	kanbanData, ok := response.Data.(map[string]interface{})
	if !ok {
		t.Fatal("Expected kanban data to be a map")
	}

	// Verify columns exist
	columns, ok := kanbanData["columns"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected columns to be a map")
	}

	// Verify statistics
	statistics, ok := kanbanData["statistics"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected statistics to be a map")
	}

	// Check expected column counts
	expectedCounts := map[string]int{
		"open":        2,
		"in-progress": 1,
		"resolved":    1,
		"closed":      0,
	}

	for status, expectedCount := range expectedCounts {
		column, exists := columns[status]
		if !exists {
			t.Errorf("Expected column %q to exist", status)
			continue
		}

		columnTasks, ok := column.([]interface{})
		if !ok {
			t.Errorf("Expected column %q to be an array", status)
			continue
		}

		if len(columnTasks) != expectedCount {
			t.Errorf("Column %q: expected %d tasks, got %d", status, expectedCount, len(columnTasks))
		}

		// Verify statistics match column counts
		statCount := int(statistics[status].(float64))
		if statCount != expectedCount {
			t.Errorf("Statistics %q: expected %d, got %d", status, expectedCount, statCount)
		}
	}

	// Verify total count
	totalStat := int(statistics["total"].(float64))
	expectedTotal := 4
	if totalStat != expectedTotal {
		t.Errorf("Expected total %d, got %d", expectedTotal, totalStat)
	}
}

func TestCRUDOperationsConsistency(t *testing.T) {
	testData := setupTestAPI(t)

	// Test full CRUD cycle with content verification
	originalTask := api.TaskRequest{
		Name:        "CRUD Test Task",
		Description: "Testing complete CRUD operations",
		Status:      models.TaskStatusOpen,
		Priority:    models.TaskPriorityMedium,
		Tags:        []string{"test", "crud", "api"},
	}

	// 1. CREATE - Verify creation content
	reqBody, _ := json.Marshal(originalTask)
	req := httptest.NewRequest("POST", "/api/v1/tasks", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	addAuthHeader(req, testData.APIKey)

	w := httptest.NewRecorder()
	testData.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("CREATE failed with status %d", w.Code)
	}

	var createResponse api.APIResponse
	json.Unmarshal(w.Body.Bytes(), &createResponse)

	createdTask := createResponse.Data.(map[string]interface{})
	taskID := int(createdTask["id"].(float64))

	// Verify created content matches request
	if createdTask["name"] != originalTask.Name {
		t.Errorf("CREATE: name mismatch")
	}
	if createdTask["description"] != originalTask.Description {
		t.Errorf("CREATE: description mismatch")
	}

	// 2. READ - Verify retrieval content
	req = httptest.NewRequest("GET", "/api/v1/tasks/1", nil)
	req.SetPathValue("id", "1")
	addAuthHeader(req, testData.APIKey)

	w = httptest.NewRecorder()
	testData.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("READ failed with status %d", w.Code)
	}

	var readResponse api.APIResponse
	json.Unmarshal(w.Body.Bytes(), &readResponse)

	readTask := readResponse.Data.(map[string]interface{})

	// Verify read content matches created content
	if readTask["name"] != createdTask["name"] {
		t.Errorf("READ: name doesn't match created task")
	}
	if readTask["id"] != createdTask["id"] {
		t.Errorf("READ: ID doesn't match created task")
	}

	// 3. UPDATE - Verify update content
	updateTask := api.TaskRequest{
		Name:        "Updated CRUD Test Task",
		Description: "Updated description for testing",
		Status:      models.TaskStatusInProgress,
		Priority:    models.TaskPriorityHigh,
		Tags:        []string{"updated", "test"},
	}

	reqBody, _ = json.Marshal(updateTask)
	req = httptest.NewRequest("PUT", "/api/v1/tasks/1", bytes.NewBuffer(reqBody))
	req.SetPathValue("id", "1")
	req.Header.Set("Content-Type", "application/json")
	addAuthHeader(req, testData.APIKey)

	w = httptest.NewRecorder()
	testData.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("UPDATE failed with status %d", w.Code)
	}

	var updateResponse api.APIResponse
	json.Unmarshal(w.Body.Bytes(), &updateResponse)

	updatedTask := updateResponse.Data.(map[string]interface{})

	// Verify update content
	if updatedTask["name"] != updateTask.Name {
		t.Errorf("UPDATE: name not updated correctly")
	}
	if updatedTask["status"] != string(updateTask.Status) {
		t.Errorf("UPDATE: status not updated correctly")
	}
	if int(updatedTask["id"].(float64)) != taskID {
		t.Errorf("UPDATE: ID should remain the same")
	}

	// 4. READ again - Verify persistence
	req = httptest.NewRequest("GET", "/api/v1/tasks/1", nil)
	req.SetPathValue("id", "1")
	addAuthHeader(req, testData.APIKey)

	w = httptest.NewRecorder()
	testData.Handler.ServeHTTP(w, req)

	var readAfterUpdateResponse api.APIResponse
	json.Unmarshal(w.Body.Bytes(), &readAfterUpdateResponse)

	persistedTask := readAfterUpdateResponse.Data.(map[string]interface{})

	// Verify changes persisted
	if persistedTask["name"] != updateTask.Name {
		t.Errorf("PERSISTENCE: updated name not persisted")
	}
	if persistedTask["status"] != string(updateTask.Status) {
		t.Errorf("PERSISTENCE: updated status not persisted")
	}

	// 5. DELETE - Verify deletion
	req = httptest.NewRequest("DELETE", "/api/v1/tasks/1", nil)
	req.SetPathValue("id", "1")
	addAuthHeader(req, testData.APIKey)

	w = httptest.NewRecorder()
	testData.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("DELETE failed with status %d", w.Code)
	}

	// 6. READ after DELETE - Verify task no longer accessible
	req = httptest.NewRequest("GET", "/api/v1/tasks/1", nil)
	req.SetPathValue("id", "1")
	addAuthHeader(req, testData.APIKey)

	w = httptest.NewRecorder()
	testData.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected 404 after DELETE, got %d", w.Code)
	}

	var deleteVerifyResponse api.APIResponse
	json.Unmarshal(w.Body.Bytes(), &deleteVerifyResponse)

	if deleteVerifyResponse.Success {
		t.Error("Expected success=false when accessing deleted task")
	}
}
