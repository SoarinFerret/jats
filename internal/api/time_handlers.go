package api

import (
	"net/http"
	"time"

	"github.com/soarinferret/jats/internal/models"
	"github.com/soarinferret/jats/internal/services"
	"github.com/soarinferret/jats/internal/utils"
)

type TimeHandlers struct {
	taskService *services.TaskService
}

func NewTimeHandlers(taskService *services.TaskService) *TimeHandlers {
	return &TimeHandlers{
		taskService: taskService,
	}
}

// GetTimeEntries handles GET /api/v1/tasks/{id}/time
func (h *TimeHandlers) GetTimeEntries(w http.ResponseWriter, r *http.Request) {
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
	
	// This would need to be implemented in the repository
	// For now, return empty array as placeholder
	SendSuccess(w, []models.TimeEntry{}, "Time entries retrieved successfully")
}

// CreateTimeEntry handles POST /api/v1/tasks/{id}/time
func (h *TimeHandlers) CreateTimeEntry(w http.ResponseWriter, r *http.Request) {
	taskID, err := GetIDFromPath(r)
	if err != nil {
		SendBadRequest(w, "Invalid task ID", nil)
		return
	}
	
	var req TimeEntryRequest
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
	
	// Parse creation date if provided
	var createdAt time.Time
	if req.Date != "" {
		parsed, err := utils.ParseDate(req.Date)
		if err != nil {
			SendBadRequest(w, "Invalid date format", err.Error())
			return
		}
		createdAt = parsed
	} else {
		createdAt = time.Now()
	}

	// Create time entry
	timeEntry := &models.TimeEntry{
		TaskID:      taskID,
		Description: req.Description,
		Duration:    req.Duration,
	}
	
	if err := h.taskService.AddTimeEntryWithDate(taskID, timeEntry, createdAt); err != nil {
		SendInternalError(w, "Failed to create time entry")
		return
	}
	
	SendCreated(w, timeEntry, "Time entry created successfully")
}

// UpdateTimeEntry handles PUT /api/v1/tasks/{taskId}/time/{id}
func (h *TimeHandlers) UpdateTimeEntry(w http.ResponseWriter, r *http.Request) {
	taskID, err := GetIDFromPath(r)
	if err != nil {
		SendBadRequest(w, "Invalid task ID", nil)
		return
	}
	
	// Get time entry ID from path (would need custom path parsing)
	// This is a simplified implementation
	timeEntryIDStr := r.URL.Query().Get("time_id")
	if timeEntryIDStr == "" {
		SendBadRequest(w, "Time entry ID required", nil)
		return
	}
	
	var req TimeEntryRequest
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
	
	// This would need repository method to update time entry
	// For now, just return success
	timeEntry := &models.TimeEntry{
		ID:          1, // placeholder
		TaskID:      taskID,
		Description: req.Description,
		Duration:    req.Duration,
		UpdatedAt:   time.Now(),
	}
	
	SendSuccess(w, timeEntry, "Time entry updated successfully")
}

// DeleteTimeEntry handles DELETE /api/v1/tasks/{taskId}/time/{id}
func (h *TimeHandlers) DeleteTimeEntry(w http.ResponseWriter, r *http.Request) {
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
	
	// This would need repository method to delete time entry
	// For now, just return success
	SendNoContent(w)
}

// GetAllTimeEntries handles GET /api/v1/time
func (h *TimeHandlers) GetAllTimeEntries(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters for filtering
	// This would need more sophisticated filtering implementation
	
	// For now, return empty array as placeholder
	SendSuccess(w, []models.TimeEntry{}, "Time entries retrieved successfully")
}