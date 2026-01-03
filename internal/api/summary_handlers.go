package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/soarinferret/jats/internal/models"
	"github.com/soarinferret/jats/internal/services"
)

type SummaryHandlers struct {
	taskService *services.TaskService
}

type TaskSummaryResponse struct {
	OpenTasks           int `json:"open_tasks"`
	InProgressTasks     int `json:"in_progress_tasks"`
	RecentlyAddedTasks  int `json:"recently_added_tasks"`  // Last 7 days
	RecentlyResolvedTasks int `json:"recently_resolved_tasks"` // Last 7 days
}

func NewSummaryHandlers(taskService *services.TaskService) *SummaryHandlers {
	return &SummaryHandlers{
		taskService: taskService,
	}
}

// GetTaskSummary handles GET /api/v1/summary/tasks
func (h *SummaryHandlers) GetTaskSummary(w http.ResponseWriter, r *http.Request) {
	// Get query parameters for filtering
	savedQueryID := r.URL.Query().Get("saved_query_id")
	
	// Get all tasks (we'll need this for filtering)
	allTasks, err := h.taskService.GetTasks()
	if err != nil {
		SendInternalError(w, "Failed to retrieve tasks")
		return
	}

	// Apply saved query filter if provided
	var filteredTasks []*models.Task
	if savedQueryID != "" {
		queryID, err := strconv.ParseUint(savedQueryID, 10, 32)
		if err != nil {
			SendBadRequest(w, "Invalid saved query ID", nil)
			return
		}

		// Get saved query to apply its filters
		savedQuery, err := h.taskService.GetSavedQueryByID(uint(queryID))
		if err != nil {
			SendNotFound(w, "Saved query not found")
			return
		}

		// Apply saved query filters to tasks
		filteredTasks = h.applyQueryFilters(allTasks, savedQuery)
	} else {
		filteredTasks = allTasks
	}

	// Calculate summary statistics
	summary := h.calculateSummary(filteredTasks)

	SendSuccess(w, summary, "Task summary retrieved successfully")
}

// applyQueryFilters applies saved query filters to tasks
func (h *SummaryHandlers) applyQueryFilters(tasks []*models.Task, query *models.SavedQuery) []*models.Task {
	var filtered []*models.Task

	for _, task := range tasks {
		if h.taskMatchesSavedQuery(task, query) {
			filtered = append(filtered, task)
		}
	}

	return filtered
}

// taskMatchesSavedQuery checks if a task matches a saved query's criteria
func (h *SummaryHandlers) taskMatchesSavedQuery(task *models.Task, query *models.SavedQuery) bool {
	// Check included tags
	if len(query.IncludedTags) > 0 {
		hasIncludedTag := false
		for _, includedTag := range query.IncludedTags {
			for _, taskTag := range task.Tags {
				if taskTag == includedTag {
					hasIncludedTag = true
					break
				}
			}
			if hasIncludedTag {
				break
			}
		}
		if !hasIncludedTag {
			return false
		}
	}

	// Check excluded tags
	for _, excludedTag := range query.ExcludedTags {
		for _, taskTag := range task.Tags {
			if taskTag == excludedTag {
				return false
			}
		}
	}

	return true
}

// calculateSummary calculates summary statistics from filtered tasks
func (h *SummaryHandlers) calculateSummary(tasks []*models.Task) *TaskSummaryResponse {
	var openTasks, inProgressTasks, recentlyAdded, recentlyResolved int
	
	// Calculate time boundaries
	sevenDaysAgo := time.Now().AddDate(0, 0, -7)

	for _, task := range tasks {
		// Count open tasks
		if task.Status == models.TaskStatusOpen {
			openTasks++
		}

		// Count in-progress tasks
		if task.Status == models.TaskStatusInProgress {
			inProgressTasks++
		}

		// Count recently added tasks (created in last 7 days)
		if task.CreatedAt.After(sevenDaysAgo) {
			recentlyAdded++
		}

		// Count recently resolved tasks (resolved in last 7 days)
		if task.Status == models.TaskStatusResolved && task.ResolvedAt != nil && task.ResolvedAt.After(sevenDaysAgo) {
			recentlyResolved++
		}
	}

	return &TaskSummaryResponse{
		OpenTasks:             openTasks,
		InProgressTasks:       inProgressTasks,
		RecentlyAddedTasks:    recentlyAdded,
		RecentlyResolvedTasks: recentlyResolved,
	}
}