package services

import (
	"fmt"

	"github.com/soarinferret/jats/internal/models"
	"github.com/soarinferret/jats/internal/repository"
)

type NotificationService struct {
	taskRepo    *repository.TaskRepository
	authRepo    *repository.AuthRepository
	smtpService *SMTPService
}

func NewNotificationService(taskRepo *repository.TaskRepository, authRepo *repository.AuthRepository, smtpService *SMTPService) *NotificationService {
	return &NotificationService{
		taskRepo:    taskRepo,
		authRepo:    authRepo,
		smtpService: smtpService,
	}
}

func (n *NotificationService) NotifyTaskCreated(task *models.Task) error {
	// Get all JATS users for notifications
	users, err := n.authRepo.GetAllUsers()
	if err != nil {
		return fmt.Errorf("failed to get JATS users: %w", err)
	}

	if len(users) == 0 {
		return nil
	}

	subject := fmt.Sprintf("New Task: %s", task.Name)
	content := n.buildTaskCreatedContent(task)

	// Convert users to TaskSubscriber format for compatibility with SMTP service
	var subs []models.TaskSubscriber
	for _, user := range users {
		if user.IsActive { // Only notify active users
			subs = append(subs, models.TaskSubscriber{
				Email: user.Email,
			})
		}
	}

	if len(subs) == 0 {
		return nil
	}

	return n.smtpService.SendTaskNotification(task, subs, subject, content)
}

func (n *NotificationService) NotifyTaskUpdated(task *models.Task) error {
	// For internal task management, task updates are not sent via email
	// This functionality is reserved for task creation notifications only
	return nil
}

func (n *NotificationService) NotifyCommentAdded(task *models.Task, comment *models.Comment) error {
	// For internal task management, comments are internal notes only
	// No email notifications are sent for comment additions
	return nil
}

func (n *NotificationService) NotifyStatusChanged(task *models.Task, oldStatus, newStatus models.TaskStatus) error {
	// For internal task management, status changes are not sent via email
	// This functionality is reserved for task creation notifications only
	return nil
}

func (n *NotificationService) buildTaskCreatedContent(task *models.Task) string {
	content := fmt.Sprintf("A new task has been created:\n\n")
	content += fmt.Sprintf("Task: %s\n", task.Name)

	if task.Description != "" {
		content += fmt.Sprintf("Description: %s\n", task.Description)
	}

	content += fmt.Sprintf("Status: %s\n", task.Status)

	if task.Priority != "" {
		content += fmt.Sprintf("Priority: %s\n", task.Priority)
	}

	if len(task.Tags) > 0 {
		content += fmt.Sprintf("Tags: %s\n", fmt.Sprintf("%v", task.Tags))
	}

	return content
}

func (n *NotificationService) buildTaskUpdatedContent(task *models.Task) string {
	content := fmt.Sprintf("Task has been updated:\n\n")
	content += fmt.Sprintf("Task: %s\n", task.Name)

	if task.Description != "" {
		content += fmt.Sprintf("Description: %s\n", task.Description)
	}

	content += fmt.Sprintf("Status: %s\n", task.Status)

	if task.Priority != "" {
		content += fmt.Sprintf("Priority: %s\n", task.Priority)
	}

	if len(task.Tags) > 0 {
		content += fmt.Sprintf("Tags: %s\n", fmt.Sprintf("%v", task.Tags))
	}

	return content
}

func (n *NotificationService) buildStatusChangeContent(task *models.Task, oldStatus, newStatus models.TaskStatus) string {
	content := fmt.Sprintf("Task status has changed:\n\n")
	content += fmt.Sprintf("Task: %s\n", task.Name)
	content += fmt.Sprintf("Status: %s → %s\n", oldStatus, newStatus)

	if task.Priority != "" {
		content += fmt.Sprintf("Priority: %s\n", task.Priority)
	}

	return content
}
