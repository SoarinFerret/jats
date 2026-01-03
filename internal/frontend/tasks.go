package frontend

import (
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/soarinferret/jats/internal/models"
	"github.com/soarinferret/jats/internal/services"
	"sort"
	"time"
)

// TaskHandler handles task-related frontend requests
type TaskHandler struct {
	taskService *services.TaskService
	templates   map[string]*template.Template
}

// NewTaskHandler creates a new task handler
func NewTaskHandler(taskService *services.TaskService, templates map[string]*template.Template) *TaskHandler {
	return &TaskHandler{
		taskService: taskService,
		templates:   templates,
	}
}

// TaskListHandler serves the task list view
func (h *TaskHandler) TaskListHandler(c *gin.Context) {
	authContext, exists := c.Get("auth")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	auth := authContext.(*models.AuthContext)

	// Get filters from query parameters
	status := c.Query("status")
	priority := c.Query("priority")
	search := c.Query("search")
	tags := c.QueryArray("tags")

	// Default to "open" status if no status filter is specified
	// Exception: if user explicitly selected "All" (empty value), respect that choice
	originalStatus := c.Query("status")
	if status == "" && !c.Request.URL.Query().Has("status") {
		status = "open"
	}

	// Parse pagination parameters
	page := 1
	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	limit := 20
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	// Get tasks - either from saved query or all tasks
	var allTasks []*models.Task

	// Check if this is a saved query request with pre-filtered tasks
	if savedQueryTasks, exists := c.Get("savedQueryTasks"); exists {
		allTasks = savedQueryTasks.([]*models.Task)
	} else {
		var err error
		allTasks, err = h.taskService.GetTasks()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get tasks"})
			return
		}
	}

	// Convert to slice of models.Task (not pointers)
	tasks := make([]models.Task, len(allTasks))
	for i, task := range allTasks {
		tasks[i] = *task
	}

	// Check if this is a saved query request
	var savedQuery *models.SavedQuery
	if sq, exists := c.Get("savedQuery"); exists {
		savedQuery = sq.(*models.SavedQuery)
	}

	// Apply user filters to the tasks (saved query filtering already applied)
	filteredTasks := make([]models.Task, 0)
	for _, task := range tasks {
		// Filter by status
		if status != "" && string(task.Status) != status {
			continue
		}
		// Filter by priority
		if priority != "" && string(task.Priority) != priority {
			continue
		}
		// Simple search in name and description
		if search != "" {
			searchLower := strings.ToLower(search)
			if !strings.Contains(strings.ToLower(task.Name), searchLower) &&
				!strings.Contains(strings.ToLower(task.Description), searchLower) {
				continue
			}
		}
		// Filter by tags (basic implementation)
		if len(tags) > 0 {
			hasTag := false
			for _, tag := range tags {
				for _, taskTag := range task.Tags {
					if taskTag == tag {
						hasTag = true
						break
					}
				}
				if hasTag {
					break
				}
			}
			if !hasTag {
				continue
			}
		}
		filteredTasks = append(filteredTasks, task)
	}

	// Sort by most recent activity (considering comments, time entries, and task updates)
	sort.Slice(filteredTasks, func(i, j int) bool {
		return h.getLastActivityTime(filteredTasks[i]).After(h.getLastActivityTime(filteredTasks[j]))
	})

	// Simple pagination
	startIndex := (page - 1) * limit
	endIndex := startIndex + limit
	if startIndex > len(filteredTasks) {
		startIndex = len(filteredTasks)
	}
	if endIndex > len(filteredTasks) {
		endIndex = len(filteredTasks)
	}

	paginatedTasks := filteredTasks[startIndex:endIndex]

	pagination := map[string]interface{}{
		"page":       page,
		"limit":      limit,
		"total":      len(filteredTasks),
		"totalPages": (len(filteredTasks) + limit - 1) / limit,
	}

	// Create filters struct for template
	filters := map[string]interface{}{
		"Status":   originalStatus, // Use original status for template dropdown selection
		"Priority": priority,
		"Search":   search,
		"Tags":     tags,
	}

	// Check if this is an HTMX request targeting the task list container
	// This includes initial load from template and filter changes
	hxTarget := c.GetHeader("HX-Target")
	if c.GetHeader("HX-Request") == "true" && (hxTarget == "tasks-list" || hxTarget == "this") {
		// For HTMX requests targeting the task list, return just the task list content
		h.renderFilteredTaskList(c, filteredTasks)
		return
	}

	// For full page requests, return the complete template
	data := gin.H{
		"Tasks":      paginatedTasks,
		"Pagination": pagination,
		"Filters":    filters,
		"User":       auth.User,
		"SavedQuery": savedQuery, // Add saved query for template header
	}

	c.Header("Content-Type", "text/html")
	if err := h.templates["tasks"].Execute(c.Writer, data); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
}

// TaskToggleCompleteHandler handles task completion toggle
func (h *TaskHandler) TaskToggleCompleteHandler(c *gin.Context) {
	taskIDStr := c.Param("id")
	taskID, err := strconv.ParseUint(taskIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID"})
		return
	}

	task, err := h.taskService.GetTask(uint(taskID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		return
	}

	// Toggle status
	if task.Status == models.TaskStatusResolved {
		task.Status = models.TaskStatusOpen
	} else {
		task.Status = models.TaskStatusResolved
	}

	err = h.taskService.UpdateTask(task)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update task"})
		return
	}

	// Return updated entire task card HTML
	h.renderSingleTask(c, *task)
}

// renderSingleTask renders a complete task card HTML for HTMX updates
func (h *TaskHandler) renderSingleTask(c *gin.Context, task models.Task) {
	c.Header("Content-Type", "text/html")
	c.String(http.StatusOK, h.generateTaskCardHTML(task))
}

// getLastActivityTime calculates the most recent activity time for a task
// considering task updates and subtask changes
func (h *TaskHandler) getLastActivityTime(task models.Task) time.Time {
	lastActivity := task.UpdatedAt
	
	// Check subtasks for more recent activity
	for _, subtask := range task.Subtasks {
		if subtask.UpdatedAt.After(lastActivity) {
			lastActivity = subtask.UpdatedAt
		}
	}
	
	return lastActivity
}