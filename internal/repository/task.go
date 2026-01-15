package repository

import (
	"github.com/soarinferret/jats/internal/models"
	"gorm.io/gorm"
)

type TaskRepository struct {
	db *gorm.DB
}

func NewTaskRepository(db *gorm.DB) *TaskRepository {
	return &TaskRepository{db: db}
}

func (r *TaskRepository) Create(task *models.Task) error {
	return r.db.Create(task).Error
}

func (r *TaskRepository) GetByID(id uint) (*models.Task, error) {
	var task models.Task
	err := r.db.Preload("Subtasks").Preload("TimeEntries").Preload("Comments.Attachments").Preload("Subscribers").Preload("Attachments").First(&task, id).Error
	if err != nil {
		return nil, err
	}
	return &task, nil
}

func (r *TaskRepository) GetAll() ([]*models.Task, error) {
	var tasks []*models.Task
	// Sort by most recent activity - basic sorting by task updated_at
	err := r.db.Preload("Subtasks").
		Preload("TimeEntries").
		Order("updated_at DESC").
		Find(&tasks).Error
	return tasks, err
}

func (r *TaskRepository) Update(task *models.Task) error {
	return r.db.Save(task).Error
}

func (r *TaskRepository) Delete(id uint) error {
	return r.db.Delete(&models.Task{}, id).Error
}

func (r *TaskRepository) GetByEmailMessageID(messageID string) (*models.Task, error) {
	var task models.Task
	err := r.db.Where("email_message_id = ?", messageID).First(&task).Error
	if err != nil {
		return nil, err
	}
	return &task, nil
}

func (r *TaskRepository) GetTimeEntries(taskID uint) ([]*models.TimeEntry, error) {
	var entries []*models.TimeEntry
	err := r.db.Where("task_id = ?", taskID).Find(&entries).Error
	return entries, err
}

func (r *TaskRepository) AddTimeEntry(entry *models.TimeEntry) error {
	return r.db.Create(entry).Error
}

func (r *TaskRepository) GetComments(taskID uint) ([]*models.Comment, error) {
	var comments []*models.Comment
	err := r.db.Where("task_id = ?", taskID).Order("created_at asc").Find(&comments).Error
	return comments, err
}

func (r *TaskRepository) AddComment(comment *models.Comment) error {
	return r.db.Create(comment).Error
}

func (r *TaskRepository) AddSubscriber(subscriber *models.TaskSubscriber) error {
	return r.db.FirstOrCreate(subscriber, "task_id = ? AND email = ?", subscriber.TaskID, subscriber.Email).Error
}

func (r *TaskRepository) GetSubscribers(taskID uint) ([]*models.TaskSubscriber, error) {
	var subscribers []*models.TaskSubscriber
	err := r.db.Where("task_id = ?", taskID).Find(&subscribers).Error
	return subscribers, err
}

func (r *TaskRepository) CreateSavedQuery(query *models.SavedQuery) error {
	return r.db.Create(query).Error
}

func (r *TaskRepository) GetSavedQueries() ([]*models.SavedQuery, error) {
	var queries []*models.SavedQuery
	err := r.db.Order("name").Find(&queries).Error
	return queries, err
}

func (r *TaskRepository) GetSavedQueryByID(id uint) (*models.SavedQuery, error) {
	var query models.SavedQuery
	err := r.db.First(&query, id).Error
	if err != nil {
		return nil, err
	}
	return &query, nil
}

func (r *TaskRepository) UpdateSavedQuery(query *models.SavedQuery) error {
	return r.db.Save(query).Error
}

func (r *TaskRepository) DeleteSavedQuery(id uint) error {
	return r.db.Delete(&models.SavedQuery{}, id).Error
}

func (r *TaskRepository) AddSubtask(subtask *models.Subtask) error {
	return r.db.Create(subtask).Error
}

func (r *TaskRepository) GetSubtask(subtaskID uint) (*models.Subtask, error) {
	var subtask models.Subtask
	err := r.db.First(&subtask, subtaskID).Error
	if err != nil {
		return nil, err
	}
	return &subtask, nil
}

func (r *TaskRepository) UpdateSubtask(subtask *models.Subtask) error {
	return r.db.Save(subtask).Error
}

func (r *TaskRepository) ToggleSubtask(subtaskID uint) error {
	var subtask models.Subtask
	if err := r.db.First(&subtask, subtaskID).Error; err != nil {
		return err
	}
	
	subtask.Completed = !subtask.Completed
	return r.db.Save(&subtask).Error
}

func (r *TaskRepository) DeleteSubtask(subtaskID uint) error {
	return r.db.Delete(&models.Subtask{}, subtaskID).Error
}

func (r *TaskRepository) GetAttachment(attachmentID uint) (*models.Attachment, error) {
	var attachment models.Attachment
	err := r.db.First(&attachment, attachmentID).Error
	if err != nil {
		return nil, err
	}
	return &attachment, nil
}

func (r *TaskRepository) AddAttachment(attachment *models.Attachment) error {
	return r.db.Create(attachment).Error
}
