package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/soarinferret/jats/internal/models"
	"github.com/soarinferret/jats/internal/services"
)

type TaskHandlers struct {
	taskService *services.TaskService
}

func NewTaskHandlers(taskService *services.TaskService) *TaskHandlers {
	return &TaskHandlers{
		taskService: taskService,
	}
}

// GetTasks handles GET /api/v1/tasks
func (h *TaskHandlers) GetTasks(w http.ResponseWriter, r *http.Request) {
	filters := ParseTaskFilters(r.URL.Query())
	
	// For now, implement basic filtering - can be enhanced later
	tasks, err := h.taskService.GetTasks()
	if err != nil {
		SendInternalError(w, "Failed to retrieve tasks")
		return
	}

	// Apply filters (basic implementation)
	filteredTasks := h.applyFilters(tasks, filters)
	
	// Apply pagination
	total := len(filteredTasks)
	start := filters.Offset
	end := start + filters.Limit
	
	if start >= total {
		filteredTasks = []*models.Task{}
	} else {
		if end > total {
			end = total
		}
		filteredTasks = filteredTasks[start:end]
	}
	
	// Calculate pagination meta
	pages := (total + filters.Limit - 1) / filters.Limit
	pagination := &PaginationMeta{
		Total:  total,
		Limit:  filters.Limit,
		Offset: filters.Offset,
		Pages:  pages,
	}
	
	SendPaginatedSuccess(w, filteredTasks, pagination, "Tasks retrieved successfully")
}

// GetTask handles GET /api/v1/tasks/{id}
func (h *TaskHandlers) GetTask(w http.ResponseWriter, r *http.Request) {
	id, err := GetIDFromPath(r)
	if err != nil {
		SendBadRequest(w, "Invalid task ID", nil)
		return
	}
	
	task, err := h.taskService.GetTask(id)
	if err != nil {
		SendNotFound(w, "Task not found")
		return
	}
	
	SendSuccess(w, task, "Task retrieved successfully")
}

// CreateTask handles POST /api/v1/tasks
func (h *TaskHandlers) CreateTask(w http.ResponseWriter, r *http.Request) {
	var req TaskRequest
	if err := ParseJSON(r, &req); err != nil {
		SendBadRequest(w, "Invalid JSON", err.Error())
		return
	}
	
	if errors := req.Validate(); len(errors) > 0 {
		SendValidationError(w, "Validation failed", errors)
		return
	}
	
	// Create task using service
	task, err := h.taskService.CreateTask(req.Name)
	if err != nil {
		SendInternalError(w, "Failed to create task")
		return
	}
	
	// Update additional fields if provided
	if req.Description != "" || req.Status != "" || req.Priority != "" || len(req.Tags) > 0 {
		if req.Description != "" {
			task.Description = req.Description
		}
		if req.Status != "" {
			task.Status = req.Status
		}
		if req.Priority != "" {
			task.Priority = req.Priority
		}
		if len(req.Tags) > 0 {
			task.Tags = req.Tags
		}
		
		task.UpdatedAt = time.Now()
		
		if err := h.taskService.UpdateTask(task); err != nil {
			SendInternalError(w, "Failed to update task details")
			return
		}
	}
	
	SendCreated(w, task, "Task created successfully")
}

// UpdateTask handles PUT /api/v1/tasks/{id}
func (h *TaskHandlers) UpdateTask(w http.ResponseWriter, r *http.Request) {
	id, err := GetIDFromPath(r)
	if err != nil {
		SendBadRequest(w, "Invalid task ID", nil)
		return
	}
	
	var req TaskRequest
	if err := ParseJSON(r, &req); err != nil {
		SendBadRequest(w, "Invalid JSON", err.Error())
		return
	}
	
	if errors := req.Validate(); len(errors) > 0 {
		SendValidationError(w, "Validation failed", errors)
		return
	}
	
	// Get existing task
	task, err := h.taskService.GetTask(id)
	if err != nil {
		SendNotFound(w, "Task not found")
		return
	}
	
	// Update fields
	task.Name = req.Name
	task.Description = req.Description
	if req.Status != "" {
		task.Status = req.Status
	}
	if req.Priority != "" {
		task.Priority = req.Priority
	}
	if len(req.Tags) > 0 {
		task.Tags = req.Tags
	}
	
	task.UpdatedAt = time.Now()
	
	if err := h.taskService.UpdateTask(task); err != nil {
		SendInternalError(w, "Failed to update task")
		return
	}
	
	SendSuccess(w, task, "Task updated successfully")
}

// PartialUpdateTask handles PATCH /api/v1/tasks/{id}
func (h *TaskHandlers) PartialUpdateTask(w http.ResponseWriter, r *http.Request) {
	id, err := GetIDFromPath(r)
	if err != nil {
		SendBadRequest(w, "Invalid task ID", nil)
		return
	}
	
	// Parse partial update as map
	var updates map[string]interface{}
	if err := ParseJSON(r, &updates); err != nil {
		SendBadRequest(w, "Invalid JSON", err.Error())
		return
	}
	
	// Get existing task
	task, err := h.taskService.GetTask(id)
	if err != nil {
		SendNotFound(w, "Task not found")
		return
	}
	
	// Apply partial updates
	if name, ok := updates["name"].(string); ok && name != "" {
		task.Name = name
	}
	if description, ok := updates["description"].(string); ok {
		task.Description = description
	}
	if status, ok := updates["status"].(string); ok && status != "" {
		task.Status = models.TaskStatus(status)
	}
	if priority, ok := updates["priority"].(string); ok && priority != "" {
		task.Priority = models.TaskPriority(priority)
	}
	if tags, ok := updates["tags"].([]interface{}); ok {
		task.Tags = make([]string, len(tags))
		for i, tag := range tags {
			if tagStr, ok := tag.(string); ok {
				task.Tags[i] = tagStr
			}
		}
	}
	
	task.UpdatedAt = time.Now()
	
	if err := h.taskService.UpdateTask(task); err != nil {
		SendInternalError(w, "Failed to update task")
		return
	}
	
	SendSuccess(w, task, "Task updated successfully")
}

// DeleteTask handles DELETE /api/v1/tasks/{id}
func (h *TaskHandlers) DeleteTask(w http.ResponseWriter, r *http.Request) {
	id, err := GetIDFromPath(r)
	if err != nil {
		SendBadRequest(w, "Invalid task ID", nil)
		return
	}
	
	if err := h.taskService.DeleteTask(id); err != nil {
		SendInternalError(w, "Failed to delete task")
		return
	}
	
	SendNoContent(w)
}

// Helper method to apply filters (basic implementation)
func (h *TaskHandlers) applyFilters(tasks []*models.Task, filters TaskFilters) []*models.Task {
	var filtered []*models.Task
	
	for _, task := range tasks {
		// Status filter
		if len(filters.Status) > 0 {
			found := false
			for _, status := range filters.Status {
				if task.Status == status {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		
		// Priority filter
		if len(filters.Priority) > 0 {
			found := false
			for _, priority := range filters.Priority {
				if task.Priority == priority {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		
		// Tags filter (task must have at least one matching tag)
		if len(filters.Tags) > 0 {
			found := false
			for _, filterTag := range filters.Tags {
				for _, taskTag := range task.Tags {
					if taskTag == filterTag {
						found = true
						break
					}
				}
				if found {
					break
				}
			}
			if !found {
				continue
			}
		}
		
		// Search filter (simple text search in name and description)
		if filters.Search != "" {
			searchLower := strings.ToLower(filters.Search)
			nameLower := strings.ToLower(task.Name)
			descLower := strings.ToLower(task.Description)
			if !strings.Contains(nameLower, searchLower) && !strings.Contains(descLower, searchLower) {
				continue
			}
		}
		
		filtered = append(filtered, task)
	}
	
	return filtered
}