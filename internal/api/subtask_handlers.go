package api

import (
	"net/http"

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

	// Create subtask
	subtask := &models.Subtask{
		Name:      req.Name,
		Completed: req.Completed,
	}

	err = h.taskService.AddSubtask(taskID, subtask)
	if err != nil {
		SendInternalError(w, "Failed to create subtask")
		return
	}

	SendCreated(w, subtask, "Subtask created successfully")
}

// UpdateSubtask handles PUT /api/v1/tasks/{taskId}/subtasks/{id}
func (h *SubtaskHandlers) UpdateSubtask(w http.ResponseWriter, r *http.Request) {
	taskID, err := GetIDFromPath(r)
	if err != nil {
		SendBadRequest(w, "Invalid task ID", nil)
		return
	}

	subtaskID, err := GetSubtaskIDFromPath(r)
	if err != nil {
		SendBadRequest(w, "Invalid subtask ID", nil)
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

	// Get existing subtask
	subtask, err := h.taskService.GetSubtask(subtaskID)
	if err != nil {
		SendNotFound(w, "Subtask not found")
		return
	}

	// Update fields
	subtask.Name = req.Name

	err = h.taskService.UpdateSubtask(taskID, subtask)
	if err != nil {
		SendInternalError(w, "Failed to update subtask")
		return
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

	subtaskID, err := GetSubtaskIDFromPath(r)
	if err != nil {
		SendBadRequest(w, "Invalid subtask ID", nil)
		return
	}

	// Toggle subtask
	err = h.taskService.ToggleSubtask(taskID, subtaskID)
	if err != nil {
		SendInternalError(w, "Failed to toggle subtask")
		return
	}

	// Get updated subtask to return
	subtask, err := h.taskService.GetSubtask(subtaskID)
	if err != nil {
		SendInternalError(w, "Failed to retrieve updated subtask")
		return
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

	subtaskID, err := GetSubtaskIDFromPath(r)
	if err != nil {
		SendBadRequest(w, "Invalid subtask ID", nil)
		return
	}

	// Delete subtask
	err = h.taskService.DeleteSubtask(taskID, subtaskID)
	if err != nil {
		SendInternalError(w, "Failed to delete subtask")
		return
	}

	SendNoContent(w)
}