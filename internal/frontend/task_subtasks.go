package frontend

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/soarinferret/jats/internal/models"
)

// TaskSubtasksHandler serves the subtasks panel
func (h *TaskHandler) TaskSubtasksHandler(c *gin.Context) {
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

	// Check if task allows modifications (open or in-progress)
	allowModifications := task.Status == models.TaskStatusOpen || task.Status == models.TaskStatusInProgress

	subtasksHTML := fmt.Sprintf(`
	<div class="flex flex-col h-full">
		<!-- Header -->
		<div class="p-6 border-b border-gray-200 flex-shrink-0">
			<div class="flex items-center justify-between">
				<h3 class="text-lg font-medium text-gray-900">Subtasks</h3>
				<span class="text-sm text-gray-500" id="subtask-counter">%d of %d completed</span>
			</div>
		</div>`,
		h.getCompletedSubtaskCount(task.Subtasks),
		len(task.Subtasks))

	// Add input section only if task allows modifications
	if allowModifications {
		subtasksHTML += fmt.Sprintf(`
		<!-- Add Subtask Input -->
		<div class="p-4 border-b border-gray-100 flex-shrink-0">
			<form hx-post="/app/tasks/%s/subtasks"
				  hx-target="#subtasks-content"
				  hx-swap="outerHTML"
				  hx-on="htmx:afterRequest: updateSubtaskCounter(event)">
				<div class="flex items-center space-x-2">
					<svg class="w-4 h-4 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
					</svg>
					<input type="text"
						   name="name"
						   placeholder="Add a subtask and press Enter..."
						   required
						   class="flex-1 border-0 focus:ring-0 text-sm placeholder-gray-400 bg-transparent"
						   onkeydown="handleSubtaskEnter(event, this.form)">
				</div>
			</form>
		</div>`, taskIDStr)
	} else {
		// Show disabled message for closed/resolved tasks
		subtasksHTML += `
		<!-- Disabled Add Subtask -->
		<div class="p-4 border-b border-gray-100 flex-shrink-0 bg-gray-50">
			<div class="flex items-center space-x-2 text-gray-400">
				<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
				</svg>
				<span class="flex-1 text-sm">Subtask editing disabled for ` + string(task.Status) + ` tasks</span>
			</div>
		</div>`
	}

	// Add the subtasks content section
	subtasksHTML += h.getSubtasksContentHTML(task, taskIDStr, allowModifications)

	subtasksHTML += `
		</div>
	</div>`

	c.Header("Content-Type", "text/html")
	c.String(http.StatusOK, subtasksHTML)
}

// Helper function to count completed subtasks
func (h *TaskHandler) getCompletedSubtaskCount(subtasks []models.Subtask) int {
	count := 0
	for _, subtask := range subtasks {
		if subtask.Completed {
			count++
		}
	}
	return count
}

// Helper function to generate just the subtasks content (for HTMX updates)
func (h *TaskHandler) getSubtasksContentHTML(task *models.Task, taskIDStr string, allowModifications bool) string {
	var contentHTML string

	if len(task.Subtasks) == 0 {
		if allowModifications {
			contentHTML = `
			<div class="text-center py-8">
				<svg class="mx-auto h-12 w-12 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-6 9l2 2 4-4" />
				</svg>
				<p class="mt-2 text-sm font-medium text-gray-900">No subtasks yet</p>
				<p class="text-sm text-gray-500">Add your first subtask above</p>
			</div>`
		} else {
			contentHTML = `
			<div class="text-center py-8">
				<svg class="mx-auto h-12 w-12 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-6 9l2 2 4-4" />
				</svg>
				<p class="mt-2 text-sm font-medium text-gray-900">No subtasks</p>
				<p class="text-sm text-gray-500">Task is ` + string(task.Status) + ` - no subtask modifications allowed</p>
			</div>`
		}
	} else {
		contentHTML = `<div class="space-y-3">`
		for _, subtask := range task.Subtasks {
			checkedAttr := ""
			completedClass := ""
			checkboxClass := "h-4 w-4 text-blue-600 border-gray-300 rounded focus:ring-blue-500"
			disabledAttr := ""
			actionButtons := ""

			if subtask.Completed {
				checkedAttr = "checked"
				completedClass = "line-through text-gray-500"
			}

			// Add disabled styling and remove interactions for closed/resolved tasks
			if !allowModifications {
				checkboxClass = "h-4 w-4 text-gray-400 border-gray-300 rounded cursor-not-allowed"
				disabledAttr = "disabled"
				completedClass += " opacity-60"
			} else {
				// Add action buttons only for modifiable tasks
				actionButtons = fmt.Sprintf(`
				<button hx-delete="/app/tasks/%s/subtasks/%d"
						hx-target="#subtasks-content"
						hx-swap="outerHTML"
						hx-confirm="Delete this subtask?"
						class="opacity-0 group-hover:opacity-100 text-gray-400 hover:text-red-600 p-1 rounded">
					<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
					</svg>
				</button>`, taskIDStr, subtask.ID)
			}

			// Generate checkbox with conditional HTMX attributes
			var checkboxHTML string
			if allowModifications {
				checkboxHTML = fmt.Sprintf(`<input type="checkbox" %s
					   hx-post="/app/tasks/%s/subtasks/%d/toggle"
					   hx-target="#subtasks-content"
					   hx-swap="outerHTML"
					   class="%s">`, checkedAttr, taskIDStr, subtask.ID, checkboxClass)
			} else {
				checkboxHTML = fmt.Sprintf(`<input type="checkbox" %s %s class="%s">`,
					checkedAttr, disabledAttr, checkboxClass)
			}

			contentHTML += fmt.Sprintf(`
			<div class="flex items-center justify-between group">
				<div class="flex items-center space-x-3 flex-1">
					%s
					<span class="text-sm %s flex-1">%s</span>
				</div>
				%s
			</div>`, checkboxHTML, completedClass, subtask.Name, actionButtons)
		}
		contentHTML += `</div>`
	}

	return fmt.Sprintf(`<div id="subtasks-content" class="flex-1 overflow-auto p-4">%s</div>`, contentHTML)
}

// AddSubtaskHandler handles adding new subtasks
func (h *TaskHandler) AddSubtaskHandler(c *gin.Context) {
	_, exists := c.Get("auth")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	taskIDStr := c.Param("id")
	taskID, err := strconv.ParseUint(taskIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID"})
		return
	}

	// Check task status before allowing modifications
	task, err := h.taskService.GetTask(uint(taskID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		return
	}

	// Prevent modifications on resolved/closed tasks
	if task.Status != models.TaskStatusOpen && task.Status != models.TaskStatusInProgress {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot modify subtasks on " + string(task.Status) + " tasks"})
		return
	}

	// Get form data
	name := strings.TrimSpace(c.PostForm("name"))
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Subtask name is required"})
		return
	}

	// Create subtask using the task service
	subtask := &models.Subtask{
		TaskID:    uint(taskID),
		Name:      name,
		Completed: false,
	}

	err = h.taskService.AddSubtask(uint(taskID), subtask)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add subtask"})
		return
	}

	// Get the updated task to show the new subtask
	updatedTask, err := h.taskService.GetTask(uint(taskID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get updated task"})
		return
	}

	// Return just the subtasks content with allowModifications=true since we already validated the status
	c.Header("Content-Type", "text/html")
	c.String(http.StatusOK, h.getSubtasksContentHTML(updatedTask, taskIDStr, true))
}

// ToggleSubtaskHandler handles toggling subtask completion
func (h *TaskHandler) ToggleSubtaskHandler(c *gin.Context) {
	_, exists := c.Get("auth")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	taskIDStr := c.Param("id")
	subtaskIDStr := c.Param("subtaskId")

	taskID, err := strconv.ParseUint(taskIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID"})
		return
	}

	subtaskID, err := strconv.ParseUint(subtaskIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid subtask ID"})
		return
	}

	// Check task status before allowing modifications
	task, err := h.taskService.GetTask(uint(taskID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		return
	}

	// Prevent modifications on resolved/closed tasks
	if task.Status != models.TaskStatusOpen && task.Status != models.TaskStatusInProgress {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot modify subtasks on " + string(task.Status) + " tasks"})
		return
	}

	err = h.taskService.ToggleSubtask(uint(taskID), uint(subtaskID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to toggle subtask"})
		return
	}

	// Get the updated task to show the changes
	updatedTask, err := h.taskService.GetTask(uint(taskID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get updated task"})
		return
	}

	// Return just the subtasks content with allowModifications=true since we already validated the status
	c.Header("Content-Type", "text/html")
	c.String(http.StatusOK, h.getSubtasksContentHTML(updatedTask, taskIDStr, true))
}

// DeleteSubtaskHandler handles deleting subtasks
func (h *TaskHandler) DeleteSubtaskHandler(c *gin.Context) {
	_, exists := c.Get("auth")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	taskIDStr := c.Param("id")
	subtaskIDStr := c.Param("subtaskId")

	taskID, err := strconv.ParseUint(taskIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID"})
		return
	}

	subtaskID, err := strconv.ParseUint(subtaskIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid subtask ID"})
		return
	}

	// Check task status before allowing modifications
	task, err := h.taskService.GetTask(uint(taskID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		return
	}

	// Prevent modifications on resolved/closed tasks
	if task.Status != models.TaskStatusOpen && task.Status != models.TaskStatusInProgress {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot modify subtasks on " + string(task.Status) + " tasks"})
		return
	}

	err = h.taskService.DeleteSubtask(uint(taskID), uint(subtaskID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete subtask"})
		return
	}

	// Get the updated task to show the changes
	updatedTask, err := h.taskService.GetTask(uint(taskID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get updated task"})
		return
	}

	// Return just the subtasks content with allowModifications=true since we already validated the status
	c.Header("Content-Type", "text/html")
	c.String(http.StatusOK, h.getSubtasksContentHTML(updatedTask, taskIDStr, true))
}