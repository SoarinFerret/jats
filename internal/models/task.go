package models

import (
	"gorm.io/gorm"
	"time"
)

type TaskStatus string

const (
	TaskStatusOpen       TaskStatus = "open"
	TaskStatusInProgress TaskStatus = "in-progress"
	TaskStatusResolved   TaskStatus = "resolved"
	TaskStatusClosed     TaskStatus = "closed"
)

type TaskPriority string

const (
	TaskPriorityLow    TaskPriority = "low"
	TaskPriorityMedium TaskPriority = "medium"
	TaskPriorityHigh   TaskPriority = "high"
)

type Task struct {
	ID             uint             `json:"id" gorm:"primaryKey"`
	Name           string           `json:"name" gorm:"not null"`
	Description    string           `json:"description,omitempty"`
	Status         TaskStatus       `json:"status" gorm:"default:open"`
	Priority       TaskPriority     `json:"priority,omitempty"`
	Tags           []string         `json:"tags,omitempty" gorm:"serializer:json"`
	Subtasks       []Subtask        `json:"subtasks,omitempty" gorm:"foreignKey:TaskID"`
	EmailMessageID string           `json:"email_message_id,omitempty"`
	TimeEntries    []TimeEntry      `json:"time_entries,omitempty" gorm:"foreignKey:TaskID"`
	Comments       []Comment        `json:"comments,omitempty" gorm:"foreignKey:TaskID"`
	Subscribers    []TaskSubscriber `json:"subscribers,omitempty" gorm:"foreignKey:TaskID"`
	Attachments    []Attachment     `json:"attachments,omitempty" gorm:"foreignKey:TaskID"`
	CreatedAt      time.Time        `json:"created_at"`
	UpdatedAt      time.Time        `json:"updated_at"`
	DeletedAt      gorm.DeletedAt   `json:"deleted_at,omitempty" gorm:"index"`
	ResolvedAt     *time.Time       `json:"resolved_at,omitempty"`
}

type Subtask struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	TaskID    uint      `json:"task_id" gorm:"not null"`
	Name      string    `json:"name" gorm:"not null"`
	Completed bool      `json:"completed" gorm:"default:false"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type TimeEntry struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	TaskID      uint      `json:"task_id" gorm:"not null"`
	Description string    `json:"description,omitempty"`
	Duration    int       `json:"duration" gorm:"not null"` // minutes
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Comment struct {
	ID          uint         `json:"id" gorm:"primaryKey"`
	TaskID      uint         `json:"task_id" gorm:"not null"`
	Content     string       `json:"content" gorm:"type:text;not null"`
	IsPrivate   bool         `json:"is_private" gorm:"default:false"`
	FromEmail   string       `json:"from_email,omitempty"`
	Attachments []Attachment `json:"attachments,omitempty" gorm:"foreignKey:CommentID"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

type EmailMessage struct {
	ID          uint       `json:"id" gorm:"primaryKey"`
	MessageID   string     `json:"message_id" gorm:"uniqueIndex;not null"`
	TaskID      *uint      `json:"task_id,omitempty"`
	Subject     string     `json:"subject" gorm:"not null"`
	From        string     `json:"from" gorm:"not null"`
	To          []string   `json:"to,omitempty" gorm:"serializer:json"`
	CC          []string   `json:"cc,omitempty" gorm:"serializer:json"`
	Body        string     `json:"body" gorm:"type:text"`
	Processed   bool       `json:"processed" gorm:"default:false"`
	ProcessedAt *time.Time `json:"processed_at,omitempty"`
	ReceivedAt  time.Time  `json:"received_at" gorm:"not null"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type TaskSubscriber struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	TaskID    uint      `json:"task_id" gorm:"not null"`
	Email     string    `json:"email" gorm:"not null"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Attachment struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	TaskID       *uint     `json:"task_id,omitempty"`
	CommentID    *uint     `json:"comment_id,omitempty"`
	FileName     string    `json:"filename" gorm:"not null"`
	OriginalName string    `json:"original_name" gorm:"not null"`
	ContentType  string    `json:"content_type"`
	FilePath     string    `json:"file_path" gorm:"not null"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type SavedQuery struct {
	ID           uint     `json:"id" gorm:"primaryKey"`
	Name         string   `json:"name" gorm:"not null"`
	IncludedTags []string `json:"included_tags,omitempty" gorm:"serializer:json"`
	ExcludedTags []string `json:"excluded_tags,omitempty" gorm:"serializer:json"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
