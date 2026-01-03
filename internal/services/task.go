package services

import (
	"time"

	"github.com/soarinferret/jats/internal/models"
	"github.com/soarinferret/jats/internal/repository"
)

type TaskService struct {
	repo         *repository.TaskRepository
	notification *NotificationService
}

func NewTaskService(repo *repository.TaskRepository, notification *NotificationService) *TaskService {
	return &TaskService{
		repo:         repo,
		notification: notification,
	}
}

func (s *TaskService) CreateTask(name string) (*models.Task, error) {
	return s.CreateTaskWithDate(name, time.Now())
}

func (s *TaskService) CreateTaskWithDate(name string, createdAt time.Time) (*models.Task, error) {
	task := &models.Task{
		Name:      name,
		Status:    models.TaskStatusOpen,
		CreatedAt: createdAt,
		UpdatedAt: time.Now(),
	}

	err := s.repo.Create(task)
	if err != nil {
		return nil, err
	}

	// Notify subscribers (if any exist from email creation)
	if s.notification != nil {
		go s.notification.NotifyTaskCreated(task)
	}

	return task, nil
}

func (s *TaskService) CreateTaskFromEmail(name, emailMessageID string) (*models.Task, error) {
	task := &models.Task{
		Name:           name,
		Status:         models.TaskStatusOpen,
		EmailMessageID: emailMessageID,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	err := s.repo.Create(task)
	if err != nil {
		return nil, err
	}

	// Notify subscribers (if any exist from email creation)
	if s.notification != nil {
		go s.notification.NotifyTaskCreated(task)
	}

	return task, nil
}

func (s *TaskService) GetTask(id uint) (*models.Task, error) {
	return s.repo.GetByID(id)
}

func (s *TaskService) GetTasks() ([]*models.Task, error) {
	return s.repo.GetAll()
}

func (s *TaskService) UpdateTask(task *models.Task) error {
	// Get current task for status comparison
	currentTask, err := s.repo.GetByID(task.ID)
	if err != nil {
		return err
	}

	oldStatus := currentTask.Status
	task.UpdatedAt = time.Now()

	// Update resolved timestamp if status changed to resolved
	if task.Status == models.TaskStatusResolved && oldStatus != models.TaskStatusResolved {
		now := time.Now()
		task.ResolvedAt = &now
	}

	err = s.repo.Update(task)
	if err != nil {
		return err
	}

	// Send notifications
	if s.notification != nil {
		if oldStatus != task.Status {
			go s.notification.NotifyStatusChanged(task, oldStatus, task.Status)
		} else {
			go s.notification.NotifyTaskUpdated(task)
		}
	}

	return nil
}

func (s *TaskService) DeleteTask(id uint) error {
	return s.repo.Delete(id)
}

func (s *TaskService) AddTimeEntry(taskID uint, entry *models.TimeEntry) error {
	return s.AddTimeEntryWithDate(taskID, entry, time.Now())
}

func (s *TaskService) AddTimeEntryWithDate(taskID uint, entry *models.TimeEntry, createdAt time.Time) error {
	entry.TaskID = taskID
	entry.CreatedAt = createdAt
	entry.UpdatedAt = time.Now()
	
	err := s.repo.AddTimeEntry(entry)
	if err != nil {
		return err
	}
	
	// Get task to check current status and update
	task, err := s.repo.GetByID(taskID)
	if err != nil {
		return err
	}
	
	// If task status is open, change it to in-progress
	// Do not change status if it's already resolved or closed
	oldStatus := task.Status
	if task.Status == models.TaskStatusOpen {
		task.Status = models.TaskStatusInProgress
	}
	
	// Update task's updated_at timestamp for proper sorting
	task.UpdatedAt = time.Now()
	
	err = s.repo.Update(task)
	if err != nil {
		return err
	}
	
	// Send status change notification if status changed
	if s.notification != nil && oldStatus != task.Status {
		go s.notification.NotifyStatusChanged(task, oldStatus, task.Status)
	}
	
	return nil
}

func (s *TaskService) AddComment(taskID uint, comment *models.Comment) error {
	comment.TaskID = taskID
	comment.CreatedAt = time.Now()
	comment.UpdatedAt = time.Now()

	err := s.repo.AddComment(comment)
	if err != nil {
		return err
	}

	// Get task for notifications and update its timestamp
	task, err := s.repo.GetByID(taskID)
	if err != nil {
		return err
	}
	
	// Update task's updated_at timestamp for proper sorting
	task.UpdatedAt = time.Now()
	err = s.repo.Update(task)
	if err != nil {
		return err
	}

	// Send notifications
	if s.notification != nil {
		go s.notification.NotifyCommentAdded(task, comment)
	}

	return nil
}


func (s *TaskService) CreateSavedQuery(query *models.SavedQuery) (*models.SavedQuery, error) {
	query.CreatedAt = time.Now()
	query.UpdatedAt = time.Now()
	
	err := s.repo.CreateSavedQuery(query)
	if err != nil {
		return nil, err
	}
	
	return query, nil
}

func (s *TaskService) GetSavedQueries() ([]*models.SavedQuery, error) {
	return s.repo.GetSavedQueries()
}

func (s *TaskService) GetSavedQueryByID(id uint) (*models.SavedQuery, error) {
	return s.repo.GetSavedQueryByID(id)
}

func (s *TaskService) UpdateSavedQuery(query *models.SavedQuery) (*models.SavedQuery, error) {
	query.UpdatedAt = time.Now()
	
	err := s.repo.UpdateSavedQuery(query)
	if err != nil {
		return nil, err
	}
	
	return query, nil
}

func (s *TaskService) DeleteSavedQuery(id uint) error {
	return s.repo.DeleteSavedQuery(id)
}

func (s *TaskService) GetTasksBySavedQuery(query *models.SavedQuery) ([]*models.Task, error) {
	tasks, err := s.repo.GetAll()
	if err != nil {
		return nil, err
	}
	
	var filteredTasks []*models.Task
	for _, task := range tasks {
		if s.matchesSavedQuery(task, query) {
			filteredTasks = append(filteredTasks, task)
		}
	}
	
	return filteredTasks, nil
}

func (s *TaskService) matchesSavedQuery(task *models.Task, query *models.SavedQuery) bool {
	if len(query.IncludedTags) > 0 {
		hasIncluded := false
		for _, includedTag := range query.IncludedTags {
			for _, taskTag := range task.Tags {
				if taskTag == includedTag {
					hasIncluded = true
					break
				}
			}
			if hasIncluded {
				break
			}
		}
		if !hasIncluded {
			return false
		}
	}
	
	if len(query.ExcludedTags) > 0 {
		for _, excludedTag := range query.ExcludedTags {
			for _, taskTag := range task.Tags {
				if taskTag == excludedTag {
					return false
				}
			}
		}
	}
	
	return true
}

func (s *TaskService) AddSubtask(taskID uint, subtask *models.Subtask) error {
	subtask.TaskID = taskID
	subtask.CreatedAt = time.Now()
	subtask.UpdatedAt = time.Now()
	
	err := s.repo.AddSubtask(subtask)
	if err != nil {
		return err
	}
	
	// Update task's updated_at timestamp for proper sorting
	task, err := s.repo.GetByID(taskID)
	if err != nil {
		return err
	}
	task.UpdatedAt = time.Now()
	return s.repo.Update(task)
}

func (s *TaskService) ToggleSubtask(taskID uint, subtaskID uint) error {
	err := s.repo.ToggleSubtask(subtaskID)
	if err != nil {
		return err
	}
	
	// Update task's updated_at timestamp for proper sorting
	task, err := s.repo.GetByID(taskID)
	if err != nil {
		return err
	}
	task.UpdatedAt = time.Now()
	return s.repo.Update(task)
}

func (s *TaskService) DeleteSubtask(taskID uint, subtaskID uint) error {
	return s.repo.DeleteSubtask(subtaskID)
}

func (s *TaskService) GetAttachment(attachmentID uint) (*models.Attachment, error) {
	return s.repo.GetAttachment(attachmentID)
}

func (s *TaskService) AddAttachment(attachment *models.Attachment) error {
	attachment.CreatedAt = time.Now()
	attachment.UpdatedAt = time.Now()
	return s.repo.AddAttachment(attachment)
}
