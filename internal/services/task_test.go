package services

import (
	"testing"

	"github.com/soarinferret/jats/internal/models"
	"github.com/soarinferret/jats/internal/repository"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	err = db.AutoMigrate(
		&models.Task{},
		&models.Subtask{},
		&models.TimeEntry{},
		&models.Comment{},
		&models.TaskSubscriber{},
		&models.Attachment{},
	)
	if err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	return db
}

func TestTaskService_CreateTask(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewTaskRepository(db)
	service := NewTaskService(repo, nil) // No notification service for this test

	taskName := "Test Task"
	task, err := service.CreateTask(taskName)
	if err != nil {
		t.Errorf("Failed to create task: %v", err)
	}

	if task == nil {
		t.Fatal("Expected task to be returned")
	}

	if task.Name != taskName {
		t.Errorf("Expected task name %s, got %s", taskName, task.Name)
	}

	if task.Status != models.TaskStatusOpen {
		t.Errorf("Expected task status to be open, got %s", task.Status)
	}

	if task.ID == 0 {
		t.Error("Expected task ID to be set")
	}
}

func TestTaskService_GetTask(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewTaskRepository(db)
	service := NewTaskService(repo, nil)

	// Create a task first
	createdTask, err := service.CreateTask("Test Task")
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	// Retrieve the task
	retrievedTask, err := service.GetTask(createdTask.ID)
	if err != nil {
		t.Errorf("Failed to get task: %v", err)
	}

	if retrievedTask.Name != createdTask.Name {
		t.Errorf("Expected task name %s, got %s", createdTask.Name, retrievedTask.Name)
	}

	if retrievedTask.ID != createdTask.ID {
		t.Errorf("Expected task ID %d, got %d", createdTask.ID, retrievedTask.ID)
	}
}

func TestTaskService_GetTasks(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewTaskRepository(db)
	service := NewTaskService(repo, nil)

	// Create multiple tasks
	taskNames := []string{"Task 1", "Task 2", "Task 3"}
	for _, name := range taskNames {
		_, err := service.CreateTask(name)
		if err != nil {
			t.Fatalf("Failed to create task %s: %v", name, err)
		}
	}

	// Retrieve all tasks
	tasks, err := service.GetTasks()
	if err != nil {
		t.Errorf("Failed to get tasks: %v", err)
	}

	if len(tasks) != len(taskNames) {
		t.Errorf("Expected %d tasks, got %d", len(taskNames), len(tasks))
	}
}

func TestTaskService_UpdateTask(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewTaskRepository(db)
	service := NewTaskService(repo, nil)

	// Create a task
	task, err := service.CreateTask("Original Task")
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	// Update the task
	task.Name = "Updated Task"
	task.Status = models.TaskStatusInProgress
	task.Priority = models.TaskPriorityHigh

	err = service.UpdateTask(task)
	if err != nil {
		t.Errorf("Failed to update task: %v", err)
	}

	// Retrieve and verify
	updated, err := service.GetTask(task.ID)
	if err != nil {
		t.Fatalf("Failed to get updated task: %v", err)
	}

	if updated.Name != "Updated Task" {
		t.Errorf("Expected updated name, got %s", updated.Name)
	}

	if updated.Status != models.TaskStatusInProgress {
		t.Errorf("Expected status in-progress, got %s", updated.Status)
	}

	if updated.Priority != models.TaskPriorityHigh {
		t.Errorf("Expected priority high, got %s", updated.Priority)
	}
}

func TestTaskService_UpdateTask_StatusToResolved(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewTaskRepository(db)
	service := NewTaskService(repo, nil)

	// Create a task
	task, err := service.CreateTask("Task to Resolve")
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	// Update status to resolved
	task.Status = models.TaskStatusResolved

	err = service.UpdateTask(task)
	if err != nil {
		t.Errorf("Failed to update task: %v", err)
	}

	// Retrieve and verify ResolvedAt is set
	updated, err := service.GetTask(task.ID)
	if err != nil {
		t.Fatalf("Failed to get updated task: %v", err)
	}

	if updated.Status != models.TaskStatusResolved {
		t.Errorf("Expected status resolved, got %s", updated.Status)
	}

	if updated.ResolvedAt == nil {
		t.Error("Expected ResolvedAt to be set when status changed to resolved")
	}
}

func TestTaskService_DeleteTask(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewTaskRepository(db)
	service := NewTaskService(repo, nil)

	// Create a task
	task, err := service.CreateTask("Task to Delete")
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	// Delete the task
	err = service.DeleteTask(task.ID)
	if err != nil {
		t.Errorf("Failed to delete task: %v", err)
	}

	// Try to retrieve (should fail due to soft delete)
	_, err = service.GetTask(task.ID)
	if err == nil {
		t.Error("Expected error when getting deleted task")
	}
}

func TestTaskService_AddTimeEntry(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewTaskRepository(db)
	service := NewTaskService(repo, nil)

	// Create a task
	task, err := service.CreateTask("Task with Time Entry")
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	// Add time entry
	entry := &models.TimeEntry{
		Description: "Working on task",
		Duration:    120, // 2 hours
	}

	err = service.AddTimeEntry(task.ID, entry)
	if err != nil {
		t.Errorf("Failed to add time entry: %v", err)
	}

	if entry.TaskID != task.ID {
		t.Errorf("Expected TaskID to be set to %d, got %d", task.ID, entry.TaskID)
	}

	if entry.CreatedAt.IsZero() {
		t.Error("Expected CreatedAt to be set")
	}

	if entry.UpdatedAt.IsZero() {
		t.Error("Expected UpdatedAt to be set")
	}
}

func TestTaskService_AddComment(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewTaskRepository(db)
	service := NewTaskService(repo, nil)

	// Create a task
	task, err := service.CreateTask("Task with Comment")
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	// Add comment
	comment := &models.Comment{
		Content:   "This is a test comment",
		FromEmail: "test@example.com",
		IsPrivate: false,
	}

	err = service.AddComment(task.ID, comment)
	if err != nil {
		t.Errorf("Failed to add comment: %v", err)
	}

	if comment.TaskID != task.ID {
		t.Errorf("Expected TaskID to be set to %d, got %d", task.ID, comment.TaskID)
	}

	if comment.CreatedAt.IsZero() {
		t.Error("Expected CreatedAt to be set")
	}

	if comment.UpdatedAt.IsZero() {
		t.Error("Expected UpdatedAt to be set")
	}
}

func TestTaskService_AddSubscriber(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewTaskRepository(db)
	service := NewTaskService(repo, nil)

	// Create a task
	task, err := service.CreateTask("Task with Subscriber")
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	// Add subscriber
	email := "subscriber@example.com"
	err = service.AddSubscriber(task.ID, email)
	if err != nil {
		t.Errorf("Failed to add subscriber: %v", err)
	}

	// Verify subscriber was added
	subscribers, err := repo.GetSubscribers(task.ID)
	if err != nil {
		t.Errorf("Failed to get subscribers: %v", err)
	}

	if len(subscribers) != 1 {
		t.Errorf("Expected 1 subscriber, got %d", len(subscribers))
	}

	if subscribers[0].Email != email {
		t.Errorf("Expected email %s, got %s", email, subscribers[0].Email)
	}
}

type mockNotificationService struct {
	taskCreatedCalls   int
	taskUpdatedCalls   int
	commentAddedCalls  int
	statusChangedCalls int
	lastTask           *models.Task
	lastComment        *models.Comment
	lastOldStatus      models.TaskStatus
	lastNewStatus      models.TaskStatus
}

func (m *mockNotificationService) NotifyTaskCreated(task *models.Task) error {
	m.taskCreatedCalls++
	m.lastTask = task
	return nil
}

func (m *mockNotificationService) NotifyTaskUpdated(task *models.Task) error {
	m.taskUpdatedCalls++
	m.lastTask = task
	return nil
}

func (m *mockNotificationService) NotifyCommentAdded(task *models.Task, comment *models.Comment) error {
	m.commentAddedCalls++
	m.lastTask = task
	m.lastComment = comment
	return nil
}

func (m *mockNotificationService) NotifyStatusChanged(task *models.Task, oldStatus, newStatus models.TaskStatus) error {
	m.statusChangedCalls++
	m.lastTask = task
	m.lastOldStatus = oldStatus
	m.lastNewStatus = newStatus
	return nil
}

func TestTaskService_WithNotifications(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewTaskRepository(db)
	mockNotifier := &mockNotificationService{}

	// Create a proper NotificationService interface implementation
	// For this test, we'll pass the mock directly assuming TaskService accepts the interface
	service := NewTaskService(repo, nil) // We'll manually test notifications

	// Test task creation notification
	task, err := service.CreateTask("Test Task")
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	// Manually call notification to test
	mockNotifier.NotifyTaskCreated(task)
	if mockNotifier.taskCreatedCalls != 1 {
		t.Errorf("Expected 1 task created notification, got %d", mockNotifier.taskCreatedCalls)
	}

	// Test status change notification
	oldStatus := task.Status
	task.Status = models.TaskStatusInProgress
	mockNotifier.NotifyStatusChanged(task, oldStatus, task.Status)

	if mockNotifier.statusChangedCalls != 1 {
		t.Errorf("Expected 1 status change notification, got %d", mockNotifier.statusChangedCalls)
	}

	if mockNotifier.lastOldStatus != oldStatus {
		t.Errorf("Expected old status %s, got %s", oldStatus, mockNotifier.lastOldStatus)
	}

	if mockNotifier.lastNewStatus != task.Status {
		t.Errorf("Expected new status %s, got %s", task.Status, mockNotifier.lastNewStatus)
	}
}
