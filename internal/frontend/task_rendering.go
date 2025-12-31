package frontend

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/soarinferret/jats/internal/models"
)

// generateTaskCardHTML generates HTML for a single task card
func (h *TaskHandler) generateTaskCardHTML(task models.Task) string {
	// Task completion checkbox
	checkboxClass := "flex-shrink-0 h-5 w-5 rounded-full border-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
	checkboxContent := ""
	taskNameClass := "text-lg font-medium text-gray-900"

	if task.Status == models.TaskStatusResolved {
		checkboxClass += " border-green-500 bg-green-500"
		taskNameClass += " line-through text-gray-500"
		checkboxContent = `<svg class="h-3 w-3 text-white m-auto" fill="currentColor" viewBox="0 0 20 20">
			<path fill-rule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clip-rule="evenodd"></path>
		</svg>`
	} else {
		checkboxClass += " border-gray-300 hover:border-gray-400"
	}

	// Priority badge
	priorityClass := "inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium"
	switch task.Priority {
	case "urgent":
		priorityClass += " bg-red-100 text-red-800"
	case "high":
		priorityClass += " bg-orange-100 text-orange-800"
	case "medium":
		priorityClass += " bg-yellow-100 text-yellow-800"
	default:
		priorityClass += " bg-gray-100 text-gray-800"
	}

	// Build the task card HTML
	taskHTML := fmt.Sprintf(`
	<div class="bg-white rounded-lg border border-gray-200 p-4 hover:shadow-md transition-shadow cursor-pointer"
		 data-task-id="%d"
		 onclick="showTaskDetail(%d)">
		<div class="flex items-start justify-between">
			<div class="flex-1">
				<div class="flex items-center space-x-3">
					<button hx-post="/app/tasks/%d/toggle-complete"
							hx-target="closest .bg-white"
							hx-swap="outerHTML"
							onclick="event.stopPropagation()"
							class="%s">
						%s
					</button>
					<h3 class="%s">%s</h3>
					<span class="%s">%s</span>
				</div>`,
		task.ID, task.ID, task.ID,
		checkboxClass, checkboxContent,
		taskNameClass, task.Name,
		priorityClass, task.Priority)

	// Add description if present
	if task.Description != "" {
		taskHTML += fmt.Sprintf(`
				<p class="mt-2 text-gray-600 text-sm">%s</p>`, task.Description)
	}

	// Add metadata section
	taskHTML += `
				<div class="mt-3 flex items-center space-x-4 text-sm text-gray-500">
					<span class="inline-flex items-center">
						<svg class="mr-1 h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 4V2a1 1 0 011-1h8a1 1 0 011 1v2h4a1 1 0 110 2h-1v14a2 2 0 01-2 2H6a2 2 0 01-2-2V6H3a1 1 0 110-2h4z"/>
						</svg>` + string(task.Status) + `</span>`

	// Add tags if present
	if len(task.Tags) > 0 {
		taskHTML += `
					<div class="flex flex-wrap gap-1">`
		for _, tag := range task.Tags {
			taskHTML += fmt.Sprintf(`
						<span class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-blue-100 text-blue-800">%s</span>`, tag)
		}
		taskHTML += `
					</div>`
	}

	// Add creation date
	taskHTML += fmt.Sprintf(`
					<span class="inline-flex items-center">
						<svg class="mr-1 h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
						</svg>
						%s
					</span>
				</div>
			</div>
		</div>
	</div>`, task.CreatedAt.Format("Jan 2, 2006"))

	return taskHTML
}

// renderFilteredTaskList renders just the task list HTML for HTMX filter updates
func (h *TaskHandler) renderFilteredTaskList(c *gin.Context, tasks []models.Task) {
	// Generate task list HTML
	tasksHTML := ""
	if len(tasks) == 0 {
		tasksHTML = `<div class="text-center py-12">
			<svg class="mx-auto h-12 w-12 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-6 9l2 2 4-4" />
			</svg>
			<h3 class="mt-2 text-sm font-medium text-gray-900">No tasks</h3>
			<p class="mt-1 text-sm text-gray-500">No tasks match the current filters.</p>
		</div>`
	} else {
		for _, task := range tasks {
			tasksHTML += h.generateTaskCardHTML(task)
		}
	}

	c.Header("Content-Type", "text/html")
	c.String(http.StatusOK, tasksHTML)
}

// renderTaskList renders just the task list HTML for HTMX updates
func (h *TaskHandler) renderTaskList(c *gin.Context, auth *models.AuthContext) {
	// Get all tasks
	allTasks, err := h.taskService.GetTasks()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get tasks"})
		return
	}

	// Convert to slice of models.Task (not pointers)
	tasks := make([]models.Task, len(allTasks))
	for i, task := range allTasks {
		tasks[i] = *task
	}

	// Generate task list HTML using the shared function
	tasksHTML := ""
	if len(tasks) == 0 {
		tasksHTML = `<div class="text-center py-12">
			<svg class="mx-auto h-12 w-12 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-6 9l2 2 4-4" />
			</svg>
			<h3 class="mt-2 text-sm font-medium text-gray-900">No tasks</h3>
			<p class="mt-1 text-sm text-gray-500">Get started by creating a new task.</p>
		</div>`
	} else {
		for _, task := range tasks {
			tasksHTML += h.generateTaskCardHTML(task)
		}
	}

	c.Header("Content-Type", "text/html")
	c.String(http.StatusOK, tasksHTML)
}