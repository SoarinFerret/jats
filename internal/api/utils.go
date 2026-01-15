package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/soarinferret/jats/internal/models"
)

// ParseTaskFilters parses query parameters for task filtering
type TaskFilters struct {
	Status   []models.TaskStatus   `json:"status"`
	Priority []models.TaskPriority `json:"priority"`
	Tags     []string              `json:"tags"`
	Search   string                `json:"search"`
	Limit    int                   `json:"limit"`
	Offset   int                   `json:"offset"`
	Sort     string                `json:"sort"`
	Order    string                `json:"order"`
}

// ParseTaskFilters extracts task filters from query parameters
func ParseTaskFilters(values url.Values) TaskFilters {
	filters := TaskFilters{
		Limit:  20,  // default
		Offset: 0,   // default
		Sort:   "created_at", // default
		Order:  "desc", // default
	}

	// Parse status filter
	if statusStr := values.Get("status"); statusStr != "" {
		statusParts := strings.Split(statusStr, ",")
		for _, status := range statusParts {
			filters.Status = append(filters.Status, models.TaskStatus(strings.TrimSpace(status)))
		}
	}

	// Parse priority filter
	if priorityStr := values.Get("priority"); priorityStr != "" {
		priorityParts := strings.Split(priorityStr, ",")
		for _, priority := range priorityParts {
			filters.Priority = append(filters.Priority, models.TaskPriority(strings.TrimSpace(priority)))
		}
	}

	// Parse tags filter
	if tagsStr := values.Get("tags"); tagsStr != "" {
		tagParts := strings.Split(tagsStr, ",")
		for _, tag := range tagParts {
			filters.Tags = append(filters.Tags, strings.TrimSpace(tag))
		}
	}

	// Parse search
	filters.Search = values.Get("search")

	// Parse pagination
	if limitStr := values.Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			filters.Limit = limit
		}
	}
	if offsetStr := values.Get("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
			filters.Offset = offset
		}
	}

	// Parse sorting
	if sort := values.Get("sort"); sort != "" {
		filters.Sort = sort
	}
	if order := values.Get("order"); order != "" {
		filters.Order = order
	}

	return filters
}

// ParseJSON parses JSON request body into the provided interface
func ParseJSON(r *http.Request, v interface{}) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(v)
}

// GetIDFromPath extracts ID parameter from URL path
func GetIDFromPath(r *http.Request) (uint, error) {
	// Since we're using gin.WrapF, we need to manually parse the URL path
	// Expected paths: /api/v1/tasks/{id}, /api/v1/saved-queries/{id}, etc.
	path := r.URL.Path
	parts := strings.Split(path, "/")
	
	// Find the numeric ID part - it should be after "/tasks" or "/saved-queries"
	for i, part := range parts {
		if (part == "tasks" || part == "saved-queries") && i+1 < len(parts) {
			idStr := parts[i+1]
			// Check if this part is actually an ID and not another path segment
			if id, err := strconv.ParseUint(idStr, 10, 32); err == nil {
				return uint(id), nil
			}
		}
	}
	
	return 0, nil
}

// GetSubtaskIDFromPath extracts subtask ID from URL path like /api/v1/tasks/{id}/subtasks/{subtaskId}
func GetSubtaskIDFromPath(r *http.Request) (uint, error) {
	path := r.URL.Path
	parts := strings.Split(path, "/")

	// Find "subtasks" and get the ID after it
	for i, part := range parts {
		if part == "subtasks" && i+1 < len(parts) {
			idStr := parts[i+1]
			// Check if this part is actually an ID and not "toggle" or other path segment
			if id, err := strconv.ParseUint(idStr, 10, 32); err == nil {
				return uint(id), nil
			}
		}
	}

	return 0, fmt.Errorf("subtask ID not found in path")
}

// ValidateTaskRequest validates task creation/update request
type TaskRequest struct {
	Name        string                `json:"name"`
	Description string                `json:"description,omitempty"`
	Status      models.TaskStatus     `json:"status,omitempty"`
	Priority    models.TaskPriority   `json:"priority,omitempty"`
	Tags        []string              `json:"tags,omitempty"`
	Date        string                `json:"date,omitempty"`
}

func (tr *TaskRequest) Validate() []string {
	var errors []string
	
	if strings.TrimSpace(tr.Name) == "" {
		errors = append(errors, "name is required")
	}
	
	if tr.Status != "" {
		validStatuses := map[models.TaskStatus]bool{
			models.TaskStatusOpen:       true,
			models.TaskStatusInProgress: true,
			models.TaskStatusResolved:   true,
			models.TaskStatusClosed:     true,
		}
		if !validStatuses[tr.Status] {
			errors = append(errors, "invalid status")
		}
	}
	
	if tr.Priority != "" {
		validPriorities := map[models.TaskPriority]bool{
			models.TaskPriorityLow:    true,
			models.TaskPriorityMedium: true,
			models.TaskPriorityHigh:   true,
		}
		if !validPriorities[tr.Priority] {
			errors = append(errors, "invalid priority")
		}
	}
	
	return errors
}

// TimeEntryRequest represents a time entry creation/update request
type TimeEntryRequest struct {
	Description string `json:"description,omitempty"`
	Duration    int    `json:"duration"`
	Date        string `json:"date,omitempty"`
}

func (ter *TimeEntryRequest) Validate() []string {
	var errors []string
	
	if ter.Duration <= 0 {
		errors = append(errors, "duration must be greater than 0")
	}
	
	return errors
}

// CommentRequest represents a comment creation/update request
type CommentRequest struct {
	Content   string `json:"content"`
	IsPrivate bool   `json:"is_private,omitempty"`
	FromEmail string `json:"from_email,omitempty"`
}

func (cr *CommentRequest) Validate() []string {
	var errors []string
	
	if strings.TrimSpace(cr.Content) == "" {
		errors = append(errors, "content is required")
	}
	
	return errors
}

// SubtaskRequest represents a subtask creation/update request
type SubtaskRequest struct {
	Name      string `json:"name"`
	Completed bool   `json:"completed,omitempty"`
}

func (sr *SubtaskRequest) Validate() []string {
	var errors []string
	
	if strings.TrimSpace(sr.Name) == "" {
		errors = append(errors, "name is required")
	}
	
	return errors
}