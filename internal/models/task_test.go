package models

import (
	"testing"
	"time"
)

func TestTaskStatus(t *testing.T) {
	tests := []struct {
		name   string
		status TaskStatus
		valid  bool
	}{
		{"Open status", TaskStatusOpen, true},
		{"In progress status", TaskStatusInProgress, true},
		{"Resolved status", TaskStatusResolved, true},
		{"Closed status", TaskStatusClosed, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.status) == "" && tt.valid {
				t.Errorf("Expected valid status but got empty string")
			}
		})
	}
}

func TestTaskPriority(t *testing.T) {
	tests := []struct {
		name     string
		priority TaskPriority
		valid    bool
	}{
		{"Low priority", TaskPriorityLow, true},
		{"Medium priority", TaskPriorityMedium, true},
		{"High priority", TaskPriorityHigh, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.priority) == "" && tt.valid {
				t.Errorf("Expected valid priority but got empty string")
			}
		})
	}
}

func TestTaskCreation(t *testing.T) {
	now := time.Now()
	task := Task{
		ID:          1,
		Name:        "Test task",
		Description: "Test description",
		Status:      TaskStatusOpen,
		Priority:    TaskPriorityMedium,
		Tags:        []string{"test", "urgent"},
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if task.ID != 1 {
		t.Errorf("Expected ID 1, got %d", task.ID)
	}

	if task.Name != "Test task" {
		t.Errorf("Expected name 'Test task', got %s", task.Name)
	}

	if task.Status != TaskStatusOpen {
		t.Errorf("Expected status open, got %s", task.Status)
	}

	if task.Priority != TaskPriorityMedium {
		t.Errorf("Expected priority medium, got %s", task.Priority)
	}

	if len(task.Tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(task.Tags))
	}
}

func TestSubtaskCreation(t *testing.T) {
	subtask := Subtask{
		ID:        1,
		TaskID:    1,
		Name:      "Test subtask",
		Completed: false,
	}

	if subtask.ID != 1 {
		t.Errorf("Expected ID 1, got %d", subtask.ID)
	}

	if subtask.TaskID != 1 {
		t.Errorf("Expected TaskID 1, got %d", subtask.TaskID)
	}

	if subtask.Name != "Test subtask" {
		t.Errorf("Expected name 'Test subtask', got %s", subtask.Name)
	}

	if subtask.Completed != false {
		t.Errorf("Expected completed false, got %t", subtask.Completed)
	}
}

func TestTimeEntryCreation(t *testing.T) {
	now := time.Now()
	entry := TimeEntry{
		ID:          1,
		TaskID:      1,
		Description: "Working on task",
		Duration:    120, // 2 hours in minutes
		CreatedAt:   now,
	}

	if entry.ID != 1 {
		t.Errorf("Expected ID 1, got %d", entry.ID)
	}

	if entry.TaskID != 1 {
		t.Errorf("Expected TaskID 1, got %d", entry.TaskID)
	}

	if entry.Duration != 120 {
		t.Errorf("Expected duration 120, got %d", entry.Duration)
	}
}

func TestCommentCreation(t *testing.T) {
	now := time.Now()
	comment := Comment{
		ID:        1,
		TaskID:    1,
		Content:   "This is a test comment",
		IsPrivate: false,
		FromEmail: "test@example.com",
		CreatedAt: now,
	}

	if comment.ID != 1 {
		t.Errorf("Expected ID 1, got %d", comment.ID)
	}

	if comment.TaskID != 1 {
		t.Errorf("Expected TaskID 1, got %d", comment.TaskID)
	}

	if comment.Content != "This is a test comment" {
		t.Errorf("Expected specific content, got %s", comment.Content)
	}

	if comment.IsPrivate != false {
		t.Errorf("Expected IsPrivate false, got %t", comment.IsPrivate)
	}
}

func TestAttachmentCreation(t *testing.T) {
	now := time.Now()
	taskID := uint(1)

	attachment := Attachment{
		ID:           1,
		TaskID:       &taskID,
		FileName:     "20240101-abc123.pdf",
		OriginalName: "document.pdf",
		ContentType:  "application/pdf",
		FilePath:     "/uploads/20240101-abc123.pdf",
		CreatedAt:    now,
	}

	if attachment.ID != 1 {
		t.Errorf("Expected ID 1, got %d", attachment.ID)
	}

	if attachment.TaskID == nil || *attachment.TaskID != 1 {
		t.Errorf("Expected TaskID 1, got %v", attachment.TaskID)
	}

	if attachment.FileName != "20240101-abc123.pdf" {
		t.Errorf("Expected specific filename, got %s", attachment.FileName)
	}

	if attachment.OriginalName != "document.pdf" {
		t.Errorf("Expected original name 'document.pdf', got %s", attachment.OriginalName)
	}

	if attachment.ContentType != "application/pdf" {
		t.Errorf("Expected content type 'application/pdf', got %s", attachment.ContentType)
	}
}
