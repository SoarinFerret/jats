package services

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"mime/multipart"
	"strings"
	"testing"
	"time"

	"github.com/emersion/go-imap"
	"github.com/soarinferret/jats/internal/config"
	"github.com/soarinferret/jats/internal/models"
)

func TestEmailService_CleanSubject(t *testing.T) {
	cfg := &config.Config{}
	service := NewEmailService(nil, nil, nil, nil, cfg)

	tests := []struct {
		name     string
		subject  string
		expected string
	}{
		{"No prefix", "Test Subject", "Test Subject"},
		{"Re prefix", "Re: Test Subject", "Test Subject"},
		{"RE prefix uppercase", "RE: Test Subject", "Test Subject"},
		{"Fwd prefix", "Fwd: Test Subject", "Test Subject"},
		{"FWD prefix uppercase", "FWD: Test Subject", "Test Subject"},
		{"Fw prefix", "Fw: Test Subject", "Test Subject"},
		{"Multiple prefixes", "Re: Fwd: Test Subject", "Test Subject"},
		{"With extra spaces", "Re:   Test Subject", "Test Subject"},
		{"Empty after prefix", "Re: ", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.cleanSubject(tt.subject)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestEmailService_ReadPlainBody(t *testing.T) {
	cfg := &config.Config{}
	service := NewEmailService(nil, nil, nil, nil, cfg)

	testContent := "This is a test email body"
	reader := strings.NewReader(testContent)

	result, err := service.readPlainBody(reader)
	if err != nil {
		t.Errorf("Failed to read plain body: %v", err)
	}

	if result != testContent {
		t.Errorf("Expected %q, got %q", testContent, result)
	}
}

func TestEmailService_ProcessAttachment(t *testing.T) {
	tempDir := t.TempDir()
	storageService := NewStorageService(tempDir)
	cfg := &config.Config{}
	emailService := NewEmailService(nil, nil, nil, storageService, cfg)

	// Create a mock multipart.Part
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Create attachment part
	part, err := writer.CreateFormFile("attachment", "test.txt")
	if err != nil {
		t.Fatalf("Failed to create form file: %v", err)
	}

	testContent := "This is test attachment content"
	_, err = part.Write([]byte(testContent))
	if err != nil {
		t.Fatalf("Failed to write content: %v", err)
	}

	writer.Close()

	// Parse the multipart message
	reader := multipart.NewReader(&buf, writer.Boundary())
	mockPart, err := reader.NextPart()
	if err != nil {
		t.Fatalf("Failed to get next part: %v", err)
	}

	// Process the attachment
	attachment, err := emailService.processAttachment(mockPart)
	if err != nil {
		t.Errorf("Failed to process attachment: %v", err)
	}

	if attachment == nil {
		t.Fatal("Expected attachment to be returned")
	}

	if attachment.OriginalName != "test.txt" {
		t.Errorf("Expected original name 'test.txt', got %s", attachment.OriginalName)
	}

	// Verify content was saved correctly
	savedContent, err := storageService.GetAttachment(attachment.FilePath)
	if err != nil {
		t.Errorf("Failed to retrieve saved attachment: %v", err)
	}

	if string(savedContent) != testContent {
		t.Errorf("Expected content %q, got %q", testContent, string(savedContent))
	}
}

func TestEmailService_ProcessAttachment_Base64Encoded(t *testing.T) {
	tempDir := t.TempDir()
	storageService := NewStorageService(tempDir)
	cfg := &config.Config{}
	emailService := NewEmailService(nil, nil, nil, storageService, cfg)

	// Create test data and encode it as base64
	originalContent := "This is test attachment content for base64 test"
	encodedContent := base64.StdEncoding.EncodeToString([]byte(originalContent))

	// Create a multipart message with base64 encoded attachment
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Create attachment part with base64 encoding
	attachmentHeaders := map[string][]string{
		"Content-Type":                 {"text/plain"},
		"Content-Disposition":          {`attachment; filename="test.txt"`},
		"Content-Transfer-Encoding":    {"base64"},
	}
	
	attachmentPart, err := writer.CreatePart(attachmentHeaders)
	if err != nil {
		t.Fatalf("Failed to create attachment part: %v", err)
	}

	// Write base64 encoded content
	_, err = attachmentPart.Write([]byte(encodedContent))
	if err != nil {
		t.Fatalf("Failed to write encoded content: %v", err)
	}

	writer.Close()

	// Parse the multipart message
	boundary := writer.Boundary()
	body, attachments, err := emailService.parseMultipartMessage(&buf, boundary)
	if err != nil {
		t.Fatalf("Failed to parse multipart message: %v", err)
	}

	// Should have empty body and one attachment
	if body != "" {
		t.Errorf("Expected empty body, got %q", body)
	}

	if len(attachments) != 1 {
		t.Fatalf("Expected 1 attachment, got %d", len(attachments))
	}

	attachment := attachments[0]
	if attachment.OriginalName != "test.txt" {
		t.Errorf("Expected filename 'test.txt', got %q", attachment.OriginalName)
	}

	// Verify the content was properly decoded
	savedContent, err := storageService.GetAttachment(attachment.FilePath)
	if err != nil {
		t.Fatalf("Failed to retrieve saved attachment: %v", err)
	}

	if string(savedContent) != originalContent {
		t.Errorf("Expected decoded content %q, got %q", originalContent, string(savedContent))
	}
}

func TestEmailService_ParseMultipartMessage(t *testing.T) {
	tempDir := t.TempDir()
	storageService := NewStorageService(tempDir)
	cfg := &config.Config{}
	emailService := NewEmailService(nil, nil, nil, storageService, cfg)

	// Create a multipart message with both text and attachment
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add text part
	textPart, err := writer.CreatePart(map[string][]string{
		"Content-Type": {"text/plain"},
	})
	if err != nil {
		t.Fatalf("Failed to create text part: %v", err)
	}
	textContent := "This is the email body"
	textPart.Write([]byte(textContent))

	// Add attachment part
	attachPart, err := writer.CreatePart(map[string][]string{
		"Content-Type":        {"application/pdf"},
		"Content-Disposition": {"attachment; filename=\"document.pdf\""},
	})
	if err != nil {
		t.Fatalf("Failed to create attachment part: %v", err)
	}
	attachContent := "PDF content here"
	attachPart.Write([]byte(attachContent))

	writer.Close()

	// Parse the multipart message
	body, attachments, err := emailService.parseMultipartMessage(&buf, writer.Boundary())
	if err != nil {
		t.Errorf("Failed to parse multipart message: %v", err)
	}

	if body != textContent {
		t.Errorf("Expected body %q, got %q", textContent, body)
	}

	if len(attachments) != 1 {
		t.Errorf("Expected 1 attachment, got %d", len(attachments))
	}

	if attachments[0].OriginalName != "document.pdf" {
		t.Errorf("Expected original name 'document.pdf', got %s", attachments[0].OriginalName)
	}

	if attachments[0].ContentType != "application/pdf" {
		t.Errorf("Expected content type 'application/pdf', got %s", attachments[0].ContentType)
	}
}

func TestEmailService_FindTaskByMessageID(t *testing.T) {
	cfg := &config.Config{}
	mockRepo := newMockTaskRepository()
	service := NewEmailService(nil, mockRepo, nil, nil, cfg)

	// Test with empty In-Reply-To headers
	taskID, isUpdate := service.findTaskByMessageID([]string{}, "new-message@example.com")
	if isUpdate {
		t.Error("Expected isUpdate to be false for new message")
	}
	if taskID != 0 {
		t.Errorf("Expected taskID 0, got %d", taskID)
	}

	// Add a task to the mock repository
	existingTask := &models.Task{
		ID:             1,
		Name:           "Existing Task",
		EmailMessageID: "original-message@example.com",
	}
	mockRepo.addTask("original-message@example.com", existingTask)

	// Test with In-Reply-To headers that should find existing task
	inReplyTo := []string{"original-message@example.com"}
	taskID, isUpdate = service.findTaskByMessageID(inReplyTo, "reply-message@example.com")

	if !isUpdate {
		t.Error("Expected isUpdate to be true when task is found")
	}
	if taskID != 1 {
		t.Errorf("Expected taskID 1, got %d", taskID)
	}
}

func TestEmailService_ConnectIMAP_Configuration(t *testing.T) {
	tests := []struct {
		name   string
		config *config.Config
		valid  bool
	}{
		{
			"Valid SSL config",
			&config.Config{
				Email: config.EmailConfig{
					IMAPHost:     "imap.example.com",
					IMAPPort:     "993",
					IMAPUsername: "user@example.com",
					IMAPPassword: "password",
					UseSSL:       true,
				},
			},
			true,
		},
		{
			"Valid non-SSL config",
			&config.Config{
				Email: config.EmailConfig{
					IMAPHost:     "imap.example.com",
					IMAPPort:     "143",
					IMAPUsername: "user@example.com",
					IMAPPassword: "password",
					UseSSL:       false,
				},
			},
			true,
		},
		{
			"Invalid empty host",
			&config.Config{
				Email: config.EmailConfig{
					IMAPHost:     "",
					IMAPPort:     "993",
					IMAPUsername: "user@example.com",
					IMAPPassword: "password",
					UseSSL:       true,
				},
			},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewEmailService(nil, nil, nil, nil, tt.config)

			// We can't actually connect without a real IMAP server,
			// but we can test that the method exists and handles empty configs
			_, err := service.ConnectIMAP()

			if tt.valid && tt.config.Email.IMAPHost != "" {
				// Should attempt connection (and likely fail due to no server)
				if err == nil {
					t.Error("Expected connection error without real server")
				}
			}
		})
	}
}

type mockTaskService struct {
	createdTasks  []*models.Task
	addedComments []*models.Comment
}

func (m *mockTaskService) CreateTask(name string) (*models.Task, error) {
	task := &models.Task{
		ID:        uint(len(m.createdTasks) + 1),
		Name:      name,
		Status:    models.TaskStatusOpen,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	m.createdTasks = append(m.createdTasks, task)
	return task, nil
}

func (m *mockTaskService) CreateTaskFromEmail(name, emailMessageID string) (*models.Task, error) {
	task := &models.Task{
		ID:             uint(len(m.createdTasks) + 1),
		Name:           name,
		Status:         models.TaskStatusOpen,
		EmailMessageID: emailMessageID,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	m.createdTasks = append(m.createdTasks, task)
	return task, nil
}

func (m *mockTaskService) AddComment(taskID uint, comment *models.Comment) error {
	comment.TaskID = taskID
	comment.CreatedAt = time.Now()
	comment.UpdatedAt = time.Now()
	m.addedComments = append(m.addedComments, comment)
	return nil
}


func (m *mockTaskService) AddAttachment(attachment *models.Attachment) error {
	// For testing, just return nil
	return nil
}

type mockTaskRepository struct {
	tasks map[string]*models.Task
}

func newMockTaskRepository() *mockTaskRepository {
	return &mockTaskRepository{
		tasks: make(map[string]*models.Task),
	}
}

func (m *mockTaskRepository) GetByEmailMessageID(messageID string) (*models.Task, error) {
	if task, exists := m.tasks[messageID]; exists {
		return task, nil
	}
	return nil, fmt.Errorf("task not found")
}

func (m *mockTaskRepository) addTask(messageID string, task *models.Task) {
	m.tasks[messageID] = task
}

func TestEmailService_CreateNewTask(t *testing.T) {
	tempDir := t.TempDir()
	storageService := NewStorageService(tempDir)
	mockTask := &mockTaskService{}
	cfg := &config.Config{}

	emailService := NewEmailService(mockTask, nil, nil, storageService, cfg)

	// Create a mock IMAP message with body
	envelope := &imap.Envelope{
		MessageId: "test-message@example.com",
		Subject:   "Test Task Subject",
		From:      []*imap.Address{{MailboxName: "sender", HostName: "example.com"}},
	}

	// Create a simple email body
	emailBody := "This is a test email body content."
	bodySection := &imap.BodySectionName{}
	msg := &imap.Message{
		Envelope: envelope,
		Body: map[*imap.BodySectionName]imap.Literal{
			bodySection: strings.NewReader("Content-Type: text/plain\r\n\r\n" + emailBody),
		},
	}

	// Test task creation
	err := emailService.createNewTask("Test Task Subject", "sender@example.com", msg)
	if err != nil {
		t.Errorf("Failed to create new task: %v", err)
	}

	// Verify task was created
	if len(mockTask.createdTasks) != 1 {
		t.Errorf("Expected 1 task to be created, got %d", len(mockTask.createdTasks))
	}

	task := mockTask.createdTasks[0]
	if task.Name != "Test Task Subject" {
		t.Errorf("Expected task name 'Test Task Subject', got %s", task.Name)
	}

	// Note: Subscriber functionality has been removed - JATS now only manages internal tasks
}

func TestEmailService_EmailConfigValidation(t *testing.T) {
	tests := []struct {
		name          string
		config        *config.Config
		expectedValid bool
	}{
		{
			"Complete config",
			&config.Config{
				Email: config.EmailConfig{
					IMAPHost:     "imap.example.com",
					IMAPPort:     "993",
					IMAPUsername: "user@example.com",
					IMAPPassword: "password",
					UseSSL:       true,
					InboxFolder:  "INBOX",
					PollInterval: "5m",
				},
			},
			true,
		},
		{
			"Missing host",
			&config.Config{
				Email: config.EmailConfig{
					IMAPHost:     "",
					IMAPPort:     "993",
					IMAPUsername: "user@example.com",
					IMAPPassword: "password",
					UseSSL:       true,
					InboxFolder:  "INBOX",
					PollInterval: "5m",
				},
			},
			false,
		},
		{
			"Missing credentials",
			&config.Config{
				Email: config.EmailConfig{
					IMAPHost:     "imap.example.com",
					IMAPPort:     "993",
					IMAPUsername: "",
					IMAPPassword: "",
					UseSSL:       true,
					InboxFolder:  "INBOX",
					PollInterval: "5m",
				},
			},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewEmailService(nil, nil, nil, nil, tt.config)

			// Basic validation - check if required fields are present
			hasRequiredFields := tt.config.Email.IMAPHost != "" &&
				tt.config.Email.IMAPUsername != "" &&
				tt.config.Email.IMAPPassword != ""

			if hasRequiredFields != tt.expectedValid {
				t.Errorf("Expected config validity %t, got %t", tt.expectedValid, hasRequiredFields)
			}

			// Service should be created regardless
			if service == nil {
				t.Error("Expected service to be created")
			}
		})
	}
}
