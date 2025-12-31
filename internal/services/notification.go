package services

import (
	"fmt"

	"github.com/soarinferret/jats/internal/models"
	"github.com/soarinferret/jats/internal/repository"
)

type NotificationService struct {
	taskRepo    *repository.TaskRepository
	smtpService *SMTPService
}

func NewNotificationService(taskRepo *repository.TaskRepository, smtpService *SMTPService) *NotificationService {
	return &NotificationService{
		taskRepo:    taskRepo,
		smtpService: smtpService,
	}
}

func (n *NotificationService) NotifyTaskCreated(task *models.Task) error {
	subscribers, err := n.taskRepo.GetSubscribers(task.ID)
	if err != nil {
		return fmt.Errorf("failed to get subscribers: %w", err)
	}

	if len(subscribers) == 0 {
		return nil
	}

	subject := fmt.Sprintf("New Task: %s", task.Name)
	content := n.buildTaskCreatedContent(task)

	var subs []models.TaskSubscriber
	for _, sub := range subscribers {
		subs = append(subs, *sub)
	}

	return n.smtpService.SendTaskNotification(task, subs, subject, content)
}

func (n *NotificationService) NotifyTaskUpdated(task *models.Task) error {
	subscribers, err := n.taskRepo.GetSubscribers(task.ID)
	if err != nil {
		return fmt.Errorf("failed to get subscribers: %w", err)
	}

	if len(subscribers) == 0 {
		return nil
	}

	subject := fmt.Sprintf("Task Updated: %s", task.Name)
	content := n.buildTaskUpdatedContent(task)

	var subs []models.TaskSubscriber
	for _, sub := range subscribers {
		subs = append(subs, *sub)
	}

	return n.smtpService.SendTaskNotification(task, subs, subject, content)
}

func (n *NotificationService) NotifyCommentAdded(task *models.Task, comment *models.Comment) error {
	subscribers, err := n.taskRepo.GetSubscribers(task.ID)
	if err != nil {
		return fmt.Errorf("failed to get subscribers: %w", err)
	}

	if len(subscribers) == 0 {
		return nil
	}

	// Don't notify the person who added the comment
	var filteredSubs []models.TaskSubscriber
	for _, sub := range subscribers {
		if comment.FromEmail != "" && sub.Email != comment.FromEmail {
			filteredSubs = append(filteredSubs, *sub)
		} else if comment.FromEmail == "" {
			// Internal comment, notify all subscribers
			filteredSubs = append(filteredSubs, *sub)
		}
	}

	if len(filteredSubs) == 0 {
		return nil
	}

	return n.smtpService.SendTaskUpdate(task, filteredSubs, comment)
}

func (n *NotificationService) NotifyStatusChanged(task *models.Task, oldStatus, newStatus models.TaskStatus) error {
	subscribers, err := n.taskRepo.GetSubscribers(task.ID)
	if err != nil {
		return fmt.Errorf("failed to get subscribers: %w", err)
	}

	if len(subscribers) == 0 {
		return nil
	}

	subject := fmt.Sprintf("Task Status Changed: %s", task.Name)
	content := n.buildStatusChangeContent(task, oldStatus, newStatus)

	var subs []models.TaskSubscriber
	for _, sub := range subscribers {
		subs = append(subs, *sub)
	}

	return n.smtpService.SendTaskNotification(task, subs, subject, content)
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
