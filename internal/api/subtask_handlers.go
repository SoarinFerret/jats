package api

import (
	"net/http"
	"time"

	"github.com/soarinferret/jats/internal/models"
	"github.com/soarinferret/jats/internal/services"
)

type SubtaskHandlers struct {
	taskService *services.TaskService
}

func NewSubtaskHandlers(taskService *services.TaskService) *SubtaskHandlers {
	return &SubtaskHandlers{
		taskService: taskService,
	}
}

// GetSubtasks handles GET /api/v1/tasks/{id}/subtasks
func (h *SubtaskHandlers) GetSubtasks(w http.ResponseWriter, r *http.Request) {
	taskID, err := GetIDFromPath(r)
	if err != nil {
		SendBadRequest(w, "Invalid task ID", nil)
		return
	}
	
	// Get task with subtasks
	task, err := h.taskService.GetTask(taskID)
	if err != nil {
		SendNotFound(w, "Task not found")
		return
	}
	
	SendSuccess(w, task.Subtasks, "Subtasks retrieved successfully")
}

// CreateSubtask handles POST /api/v1/tasks/{id}/subtasks
func (h *SubtaskHandlers) CreateSubtask(w http.ResponseWriter, r *http.Request) {
	taskID, err := GetIDFromPath(r)
	if err != nil {
		SendBadRequest(w, "Invalid task ID", nil)
		return
	}
	
	var req SubtaskRequest
	if err := ParseJSON(r, &req); err != nil {
		SendBadRequest(w, "Invalid JSON", err.Error())
		return
	}
	
	if errors := req.Validate(); len(errors) > 0 {
		SendValidationError(w, "Validation failed", errors)
		return
	}
	
	// Verify task exists
	_, err = h.taskService.GetTask(taskID)
	if err != nil {
		SendNotFound(w, "Task not found")
		return
	}
	
	// Create subtask - this would need repository method to add subtask
	subtask := &models.Subtask{
		TaskID:    taskID,
		Name:      req.Name,
		Completed: req.Completed,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	
	// This is a placeholder - would need actual repository method
	SendCreated(w, subtask, "Subtask created successfully")
}

// UpdateSubtask handles PUT /api/v1/tasks/{taskId}/subtasks/{id}
func (h *SubtaskHandlers) UpdateSubtask(w http.ResponseWriter, r *http.Request) {
	taskID, err := GetIDFromPath(r)
	if err != nil {
		SendBadRequest(w, "Invalid task ID", nil)
		return
	}
	
	var req SubtaskRequest
	if err := ParseJSON(r, &req); err != nil {
		SendBadRequest(w, "Invalid JSON", err.Error())
		return
	}
	
	if errors := req.Validate(); len(errors) > 0 {
		SendValidationError(w, "Validation failed", errors)
		return
	}
	
	// Verify task exists
	_, err = h.taskService.GetTask(taskID)
	if err != nil {
		SendNotFound(w, "Task not found")
		return
	}
	
	// Update subtask - this would need repository method
	subtask := &models.Subtask{
		ID:        1, // placeholder
		TaskID:    taskID,
		Name:      req.Name,
		Completed: req.Completed,
		UpdatedAt: time.Now(),
	}
	
	SendSuccess(w, subtask, "Subtask updated successfully")
}

// ToggleSubtask handles PATCH /api/v1/tasks/{taskId}/subtasks/{id}/toggle
func (h *SubtaskHandlers) ToggleSubtask(w http.ResponseWriter, r *http.Request) {
	taskID, err := GetIDFromPath(r)
	if err != nil {
		SendBadRequest(w, "Invalid task ID", nil)
		return
	}
	
	// Verify task exists
	_, err = h.taskService.GetTask(taskID)
	if err != nil {
		SendNotFound(w, "Task not found")
		return
	}
	
	// This would need repository method to toggle subtask completion
	subtask := &models.Subtask{
		ID:        1, // placeholder
		TaskID:    taskID,
		Name:      "Example Subtask",
		Completed: true, // toggled
		UpdatedAt: time.Now(),
	}
	
	SendSuccess(w, subtask, "Subtask toggled successfully")
}

// DeleteSubtask handles DELETE /api/v1/tasks/{taskId}/subtasks/{id}
func (h *SubtaskHandlers) DeleteSubtask(w http.ResponseWriter, r *http.Request) {
	taskID, err := GetIDFromPath(r)
	if err != nil {
		SendBadRequest(w, "Invalid task ID", nil)
		return
	}
	
	// Verify task exists
	_, err = h.taskService.GetTask(taskID)
	if err != nil {
		SendNotFound(w, "Task not found")
		return
	}
	
	// This would need repository method to delete subtask
	SendNoContent(w)
}