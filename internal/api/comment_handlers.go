package api

import (
	"net/http"
	"time"

	"github.com/soarinferret/jats/internal/models"
	"github.com/soarinferret/jats/internal/services"
)

type CommentHandlers struct {
	taskService *services.TaskService
}

func NewCommentHandlers(taskService *services.TaskService) *CommentHandlers {
	return &CommentHandlers{
		taskService: taskService,
	}
}

// GetComments handles GET /api/v1/tasks/{id}/comments
func (h *CommentHandlers) GetComments(w http.ResponseWriter, r *http.Request) {
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
	
	// This would need repository method to get comments
	// For now, return empty array as placeholder
	SendSuccess(w, []models.Comment{}, "Comments retrieved successfully")
}

// CreateComment handles POST /api/v1/tasks/{id}/comments
func (h *CommentHandlers) CreateComment(w http.ResponseWriter, r *http.Request) {
	taskID, err := GetIDFromPath(r)
	if err != nil {
		SendBadRequest(w, "Invalid task ID", nil)
		return
	}
	
	var req CommentRequest
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
	
	// Create comment - all comments are now private (internal notes only)
	comment := &models.Comment{
		TaskID:    taskID,
		Content:   req.Content,
		IsPrivate: true, // Force all comments to be private
		FromEmail: req.FromEmail,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	
	if err := h.taskService.AddComment(taskID, comment); err != nil {
		SendInternalError(w, "Failed to create comment")
		return
	}
	
	SendCreated(w, comment, "Comment created successfully")
}

// UpdateComment handles PUT /api/v1/tasks/{taskId}/comments/{id}
func (h *CommentHandlers) UpdateComment(w http.ResponseWriter, r *http.Request) {
	taskID, err := GetIDFromPath(r)
	if err != nil {
		SendBadRequest(w, "Invalid task ID", nil)
		return
	}
	
	var req CommentRequest
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
	
	// This would need repository method to update comment
	// For now, return updated comment - all comments are now private (internal notes only)
	comment := &models.Comment{
		ID:        1, // placeholder
		TaskID:    taskID,
		Content:   req.Content,
		IsPrivate: true, // Force all comments to be private
		FromEmail: req.FromEmail,
		UpdatedAt: time.Now(),
	}
	
	SendSuccess(w, comment, "Comment updated successfully")
}

// DeleteComment handles DELETE /api/v1/tasks/{taskId}/comments/{id}
func (h *CommentHandlers) DeleteComment(w http.ResponseWriter, r *http.Request) {
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
	
	// This would need repository method to delete comment
	SendNoContent(w)
}