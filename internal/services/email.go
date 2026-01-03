package services

import (
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"strings"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/soarinferret/jats/internal/config"
	"github.com/soarinferret/jats/internal/models"
)

// TaskServiceInterface defines the interface for task operations needed by EmailService
type TaskServiceInterface interface {
	CreateTask(name string) (*models.Task, error)
	CreateTaskFromEmail(name, emailMessageID string) (*models.Task, error)
	AddComment(taskID uint, comment *models.Comment) error
	AddAttachment(attachment *models.Attachment) error
}

// TaskRepositoryInterface defines the interface for direct repository access needed by EmailService
type TaskRepositoryInterface interface {
	GetByEmailMessageID(messageID string) (*models.Task, error)
}

// AuthRepositoryInterface defines the interface for user validation
type AuthRepositoryInterface interface {
	GetUserByEmail(email string) (*models.User, error)
}

type EmailService struct {
	taskService    TaskServiceInterface
	taskRepository TaskRepositoryInterface
	authRepository AuthRepositoryInterface
	storageService *StorageService
	config         *config.Config
}

func NewEmailService(taskService TaskServiceInterface, taskRepository TaskRepositoryInterface, authRepository AuthRepositoryInterface, storageService *StorageService, cfg *config.Config) *EmailService {
	return &EmailService{
		taskService:    taskService,
		taskRepository: taskRepository,
		authRepository: authRepository,
		storageService: storageService,
		config:         cfg,
	}
}

func (s *EmailService) ConnectIMAP() (*client.Client, error) {
	address := fmt.Sprintf("%s:%s", s.config.Email.IMAPHost, s.config.Email.IMAPPort)

	var c *client.Client
	var err error

	if s.config.Email.UseSSL {
		tlsConfig := &tls.Config{
			ServerName:         s.config.Email.IMAPHost,
			InsecureSkipVerify: s.config.Email.IMAPInsecure,
		}
		c, err = client.DialTLS(address, tlsConfig)
	} else {
		c, err = client.Dial(address)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to IMAP server: %w", err)
	}

	if err := c.Login(s.config.Email.IMAPUsername, s.config.Email.IMAPPassword); err != nil {
		c.Logout()
		return nil, fmt.Errorf("failed to login: %w", err)
	}

	return c, nil
}

func (s *EmailService) ProcessInbox() error {
	c, err := s.ConnectIMAP()
	if err != nil {
		return err
	}
	defer c.Logout()

	mbox, err := c.Select(s.config.Email.InboxFolder, false)
	if err != nil {
		return fmt.Errorf("failed to select inbox: %w", err)
	}

	if mbox.Messages == 0 {
		return nil
	}

	// Search for unread messages only
	criteria := imap.NewSearchCriteria()
	criteria.WithoutFlags = []string{imap.SeenFlag}
	
	uids, err := c.Search(criteria)
	if err != nil {
		return fmt.Errorf("failed to search unread messages: %w", err)
	}

	if len(uids) == 0 {
		return nil // No unread messages
	}

	// Convert UIDs to sequence set
	seqset := new(imap.SeqSet)
	seqset.AddNum(uids...)

	messages := make(chan *imap.Message, 10)
	done := make(chan error, 1)

	go func() {
		done <- c.Fetch(seqset, []imap.FetchItem{imap.FetchEnvelope, imap.FetchBodyStructure, imap.FetchFlags, "BODY[]"}, messages)
	}()

	var processedUIDs []uint32
	for msg := range messages {
		if err := s.processMessage(msg); err != nil {
			// Log error but continue processing other messages
			fmt.Printf("Error processing message: %v\n", err)
		} else {
			// Track successfully processed message UIDs
			processedUIDs = append(processedUIDs, msg.Uid)
		}
	}

	if err := <-done; err != nil {
		return fmt.Errorf("failed to fetch messages: %w", err)
	}

	// Mark processed messages as read
	if len(processedUIDs) > 0 {
		if err := s.markMessagesAsRead(c, processedUIDs); err != nil {
			fmt.Printf("Warning: Failed to mark messages as read: %v\n", err)
		}
	}

	return nil
}

func (s *EmailService) markMessagesAsRead(c *client.Client, uids []uint32) error {
	seqset := new(imap.SeqSet)
	seqset.AddNum(uids...)
	
	item := imap.FormatFlagsOp(imap.AddFlags, true)
	flags := []interface{}{imap.SeenFlag}
	
	return c.UidStore(seqset, item, flags, nil)
}

func (s *EmailService) processMessage(msg *imap.Message) error {
	if msg.Envelope == nil {
		return nil
	}

	subject := msg.Envelope.Subject
	from := ""
	if len(msg.Envelope.From) > 0 {
		from = msg.Envelope.From[0].Address()
	}

	// Validate that sender is a JATS user
	user, err := s.authRepository.GetUserByEmail(from)
	if err != nil || user == nil {
		fmt.Printf("Ignoring email from non-JATS user: %s\n", from)
		return nil // Silently ignore emails from non-users
	}

	// Check if this is a reply to an existing task using In-Reply-To or References headers
	taskID, isUpdate := s.findTaskByMessageID([]string{msg.Envelope.InReplyTo}, msg.Envelope.MessageId)

	if isUpdate && taskID > 0 {
		return s.updateExistingTask(taskID, subject, from, msg)
	} else {
		return s.createNewTask(subject, from, msg)
	}
}

func (s *EmailService) findTaskByMessageID(inReplyTo []string, messageID string) (uint, bool) {
	// Look up task by original message ID stored in task metadata
	// First check In-Reply-To headers for the original message ID
	for _, replyToID := range inReplyTo {
		if replyToID != "" {
			if taskID := s.getTaskByMessageID(replyToID); taskID > 0 {
				return taskID, true
			}
		}
	}

	return 0, false
}

func (s *EmailService) getTaskByMessageID(messageID string) uint {
	task, err := s.taskRepository.GetByEmailMessageID(messageID)
	if err != nil {
		return 0
	}
	return task.ID
}

func (s *EmailService) createNewTask(subject, from string, msg *imap.Message) error {
	// Extract task name from subject (remove "Re:", "Fwd:", etc.)
	taskName := s.cleanSubject(subject)

	// Extract body content and attachments
	body, attachments, err := s.parseEmailContent(msg)
	if err != nil {
		return fmt.Errorf("failed to parse email content: %w", err)
	}

	// Create task with email message ID for future reply linking
	createdTask, err := s.taskService.CreateTaskFromEmail(taskName, msg.Envelope.MessageId)
	if err != nil {
		return fmt.Errorf("failed to create task: %w", err)
	}

	// Add initial comment with body content (now internal notes only)
	var commentID *uint
	if body != "" {
		comment := &models.Comment{
			Content:   body,
			FromEmail: from,
			IsPrivate: true, // Comments are now internal notes only
		}
		if err := s.taskService.AddComment(createdTask.ID, comment); err != nil {
			return fmt.Errorf("failed to add initial comment: %w", err)
		}
		commentID = &comment.ID
	}

	// Save attachments - link to comment if one was created, otherwise to task
	for _, attachment := range attachments {
		if commentID != nil {
			attachment.CommentID = commentID
		} else {
			attachment.TaskID = &createdTask.ID
		}
		if err := s.taskService.AddAttachment(attachment); err != nil {
			fmt.Printf("Warning: Failed to save attachment %s: %v\n", attachment.OriginalName, err)
		}
	}

	return nil
}

func (s *EmailService) updateExistingTask(taskID uint, subject, from string, msg *imap.Message) error {
	// Extract body content and attachments
	body, attachments, err := s.parseEmailContent(msg)
	if err != nil {
		return fmt.Errorf("failed to parse email content: %w", err)
	}

	// Add comment to existing task from email body (internal notes only)
	var commentID *uint
	if body != "" {
		comment := &models.Comment{
			Content:   body,
			FromEmail: from,
			IsPrivate: true, // Comments are now internal notes only
		}
		if err := s.taskService.AddComment(taskID, comment); err != nil {
			return fmt.Errorf("failed to add comment to task %d: %w", taskID, err)
		}
		commentID = &comment.ID
	}

	// Save attachments - link to comment if one was created, otherwise to task
	for _, attachment := range attachments {
		if commentID != nil {
			attachment.CommentID = commentID
		} else {
			attachment.TaskID = &taskID
		}
		if err := s.taskService.AddAttachment(attachment); err != nil {
			fmt.Printf("Warning: Failed to save attachment %s: %v\n", attachment.OriginalName, err)
		}
	}

	return nil
}

func (s *EmailService) cleanSubject(subject string) string {
	// Remove common email prefixes
	prefixes := []string{"Re:", "RE:", "Fwd:", "FWD:", "Fw:"}
	cleaned := subject

	for _, prefix := range prefixes {
		cleaned = strings.TrimPrefix(strings.TrimSpace(cleaned), prefix)
		cleaned = strings.TrimSpace(cleaned)
	}

	return cleaned
}

func (s *EmailService) parseEmailContent(msg *imap.Message) (body string, attachments []*models.Attachment, err error) {
	// Get the full message body
	var bodyReader io.Reader
	for _, literal := range msg.Body {
		bodyReader = literal
		break
	}

	if bodyReader == nil {
		return "", nil, fmt.Errorf("no body found in message")
	}

	// Parse the email message
	mailMsg, err := mail.ReadMessage(bodyReader)
	if err != nil {
		return "", nil, fmt.Errorf("failed to parse email: %w", err)
	}

	// Get content type and boundary
	contentType := mailMsg.Header.Get("Content-Type")
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		// Default to plain text if we can't parse
		body, err = s.readPlainBody(mailMsg.Body)
		return body, nil, err
	}

	if strings.HasPrefix(mediaType, "multipart/") {
		return s.parseMultipartMessage(mailMsg.Body, params["boundary"])
	} else {
		// Single part message
		body, err = s.readPlainBody(mailMsg.Body)
		return body, nil, err
	}
}

func (s *EmailService) parseMultipartMessage(body io.Reader, boundary string) (string, []*models.Attachment, error) {
	var textBody string
	var attachments []*models.Attachment

	mr := multipart.NewReader(body, boundary)
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", nil, err
		}

		disposition := part.Header.Get("Content-Disposition")
		contentType := part.Header.Get("Content-Type")

		if strings.HasPrefix(disposition, "attachment") || strings.Contains(disposition, "filename") {
			// This is an attachment
			attachment, err := s.processAttachment(part)
			if err != nil {
				fmt.Printf("Error processing attachment: %v\n", err)
				continue
			}
			attachments = append(attachments, attachment)
		} else if strings.HasPrefix(contentType, "text/plain") || strings.HasPrefix(contentType, "text/html") {
			// This is the message body - handle encoding
			encoding := strings.ToLower(part.Header.Get("Content-Transfer-Encoding"))
			
			var reader io.Reader = part
			switch encoding {
			case "base64":
				reader = base64.NewDecoder(base64.StdEncoding, part)
			case "quoted-printable":
				reader = quotedprintable.NewReader(part)
			}

			bodyBytes, err := io.ReadAll(reader)
			if err != nil {
				continue
			}
			textBody = string(bodyBytes)
		}
	}

	return textBody, attachments, nil
}

func (s *EmailService) processAttachment(part *multipart.Part) (*models.Attachment, error) {
	// Get content transfer encoding
	encoding := strings.ToLower(part.Header.Get("Content-Transfer-Encoding"))
	
	// Create the appropriate reader based on encoding
	var reader io.Reader = part
	switch encoding {
	case "base64":
		reader = base64.NewDecoder(base64.StdEncoding, part)
	case "quoted-printable":
		reader = quotedprintable.NewReader(part)
	}

	// Read and decode attachment data
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read attachment data: %w", err)
	}

	// Get filename from Content-Disposition header
	filename := part.FileName()
	if filename == "" {
		filename = "attachment"
	}

	// Get content type
	contentType := part.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// Save attachment using storage service
	return s.storageService.SaveAttachment(filename, contentType, data)
}

func (s *EmailService) readPlainBody(body io.Reader) (string, error) {
	bodyBytes, err := io.ReadAll(body)
	if err != nil {
		return "", err
	}
	return string(bodyBytes), nil
}

func (s *EmailService) StartPolling() error {
	ticker := time.NewTicker(s.config.GetPollInterval())
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := s.ProcessInbox(); err != nil {
				fmt.Printf("Error processing inbox: %v\n", err)
			}
		}
	}
}
