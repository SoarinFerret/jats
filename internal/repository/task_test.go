package repository

import (
	"testing"
	"time"

	"github.com/soarinferret/jats/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Auto-migrate the schema
	err = db.AutoMigrate(
		&models.Task{},
		&models.Subtask{},
		&models.TimeEntry{},
		&models.Comment{},
		&models.EmailMessage{},
		&models.TaskSubscriber{},
		&models.Attachment{},
	)
	if err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	return db
}

func TestTaskRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	repo := NewTaskRepository(db)

	task := &models.Task{
		Name:        "Test Task",
		Description: "Test Description",
		Status:      models.TaskStatusOpen,
		Priority:    models.TaskPriorityMedium,
		Tags:        []string{"test", "urgent"},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	err := repo.Create(task)
	if err != nil {
		t.Errorf("Failed to create task: %v", err)
	}

	if task.ID == 0 {
		t.Error("Expected task ID to be set after creation")
	}
}

func TestTaskRepository_GetByID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewTaskRepository(db)

	// Create a task first
	task := &models.Task{
		Name:      "Test Task",
		Status:    models.TaskStatusOpen,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err := repo.Create(task)
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	// Retrieve the task
	retrieved, err := repo.GetByID(task.ID)
	if err != nil {
		t.Errorf("Failed to get task: %v", err)
	}

	if retrieved.Name != task.Name {
		t.Errorf("Expected name %s, got %s", task.Name, retrieved.Name)
	}

	if retrieved.Status != task.Status {
		t.Errorf("Expected status %s, got %s", task.Status, retrieved.Status)
	}
}

func TestTaskRepository_GetAll(t *testing.T) {
	db := setupTestDB(t)
	repo := NewTaskRepository(db)

	// Create multiple tasks
	tasks := []*models.Task{
		{Name: "Task 1", Status: models.TaskStatusOpen, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{Name: "Task 2", Status: models.TaskStatusInProgress, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{Name: "Task 3", Status: models.TaskStatusClosed, CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}

	for _, task := range tasks {
		err := repo.Create(task)
		if err != nil {
			t.Fatalf("Failed to create task: %v", err)
		}
	}

	// Retrieve all tasks
	allTasks, err := repo.GetAll()
	if err != nil {
		t.Errorf("Failed to get all tasks: %v", err)
	}

	if len(allTasks) != 3 {
		t.Errorf("Expected 3 tasks, got %d", len(allTasks))
	}
}

func TestTaskRepository_Update(t *testing.T) {
	db := setupTestDB(t)
	repo := NewTaskRepository(db)

	// Create a task
	task := &models.Task{
		Name:      "Original Task",
		Status:    models.TaskStatusOpen,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err := repo.Create(task)
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	// Update the task
	task.Name = "Updated Task"
	task.Status = models.TaskStatusInProgress
	task.UpdatedAt = time.Now()

	err = repo.Update(task)
	if err != nil {
		t.Errorf("Failed to update task: %v", err)
	}

	// Retrieve and verify
	updated, err := repo.GetByID(task.ID)
	if err != nil {
		t.Fatalf("Failed to get updated task: %v", err)
	}

	if updated.Name != "Updated Task" {
		t.Errorf("Expected updated name 'Updated Task', got %s", updated.Name)
	}

	if updated.Status != models.TaskStatusInProgress {
		t.Errorf("Expected updated status in-progress, got %s", updated.Status)
	}
}

func TestTaskRepository_Delete(t *testing.T) {
	db := setupTestDB(t)
	repo := NewTaskRepository(db)

	// Create a task
	task := &models.Task{
		Name:      "Task to Delete",
		Status:    models.TaskStatusOpen,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err := repo.Create(task)
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	// Delete the task
	err = repo.Delete(task.ID)
	if err != nil {
		t.Errorf("Failed to delete task: %v", err)
	}

	// Try to retrieve (should fail with soft delete)
	_, err = repo.GetByID(task.ID)
	if err == nil {
		t.Error("Expected error when getting deleted task, but got none")
	}
}

func TestTaskRepository_AddTimeEntry(t *testing.T) {
	db := setupTestDB(t)
	repo := NewTaskRepository(db)

	// Create a task first
	task := &models.Task{
		Name:      "Task with Time Entry",
		Status:    models.TaskStatusOpen,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err := repo.Create(task)
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	// Add time entry
	entry := &models.TimeEntry{
		TaskID:      task.ID,
		Description: "Working on task",
		Duration:    60,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	err = repo.AddTimeEntry(entry)
	if err != nil {
		t.Errorf("Failed to add time entry: %v", err)
	}

	// Retrieve time entries
	entries, err := repo.GetTimeEntries(task.ID)
	if err != nil {
		t.Errorf("Failed to get time entries: %v", err)
	}

	if len(entries) != 1 {
		t.Errorf("Expected 1 time entry, got %d", len(entries))
	}

	if entries[0].Duration != 60 {
		t.Errorf("Expected duration 60, got %d", entries[0].Duration)
	}
}

func TestTaskRepository_AddComment(t *testing.T) {
	db := setupTestDB(t)
	repo := NewTaskRepository(db)

	// Create a task first
	task := &models.Task{
		Name:      "Task with Comment",
		Status:    models.TaskStatusOpen,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err := repo.Create(task)
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	// Add comment
	comment := &models.Comment{
		TaskID:    task.ID,
		Content:   "This is a test comment",
		IsPrivate: false,
		FromEmail: "test@example.com",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err = repo.AddComment(comment)
	if err != nil {
		t.Errorf("Failed to add comment: %v", err)
	}

	// Retrieve comments
	comments, err := repo.GetComments(task.ID)
	if err != nil {
		t.Errorf("Failed to get comments: %v", err)
	}

	if len(comments) != 1 {
		t.Errorf("Expected 1 comment, got %d", len(comments))
	}

	if comments[0].Content != "This is a test comment" {
		t.Errorf("Expected specific content, got %s", comments[0].Content)
	}
}

func TestTaskRepository_AddSubscriber(t *testing.T) {
	db := setupTestDB(t)
	repo := NewTaskRepository(db)

	// Create a task first
	task := &models.Task{
		Name:      "Task with Subscriber",
		Status:    models.TaskStatusOpen,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err := repo.Create(task)
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	// Add subscriber
	subscriber := &models.TaskSubscriber{
		TaskID:    task.ID,
		Email:     "subscriber@example.com",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err = repo.AddSubscriber(subscriber)
	if err != nil {
		t.Errorf("Failed to add subscriber: %v", err)
	}

	// Test duplicate subscriber (should not create duplicate)
	duplicate := &models.TaskSubscriber{
		TaskID:    task.ID,
		Email:     "subscriber@example.com",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err = repo.AddSubscriber(duplicate)
	if err != nil {
		t.Errorf("Failed to handle duplicate subscriber: %v", err)
	}

	// Retrieve subscribers
	subscribers, err := repo.GetSubscribers(task.ID)
	if err != nil {
		t.Errorf("Failed to get subscribers: %v", err)
	}

	if len(subscribers) != 1 {
		t.Errorf("Expected 1 subscriber, got %d", len(subscribers))
	}

	if subscribers[0].Email != "subscriber@example.com" {
		t.Errorf("Expected specific email, got %s", subscribers[0].Email)
	}
}

func TestTaskRepository_GetByEmailMessageID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewTaskRepository(db)

	// Create a task with email message ID
	messageID := "test@example.com"
	task := &models.Task{
		Name:           "Email Task",
		Status:         models.TaskStatusOpen,
		EmailMessageID: messageID,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	err := repo.Create(task)
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	// Retrieve by email message ID
	retrieved, err := repo.GetByEmailMessageID(messageID)
	if err != nil {
		t.Errorf("Failed to get task by email message ID: %v", err)
	}

	if retrieved.EmailMessageID != messageID {
		t.Errorf("Expected message ID %s, got %s", messageID, retrieved.EmailMessageID)
	}

	// Test non-existent message ID
	_, err = repo.GetByEmailMessageID("nonexistent@example.com")
	if err == nil {
		t.Error("Expected error for non-existent message ID, but got none")
	}
}
