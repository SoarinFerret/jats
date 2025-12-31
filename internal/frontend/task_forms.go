package frontend

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/soarinferret/jats/internal/models"
)

// NewTaskFormHandler serves the new task form modal
func (h *TaskHandler) NewTaskFormHandler(c *gin.Context) {
	formHTML := `
	<div class="relative bg-white rounded-lg shadow-xl max-w-md w-full mx-auto mt-20 p-6">
		<div class="flex justify-between items-center mb-4">
			<h3 class="text-lg font-medium text-gray-900">Create New Task</h3>
			<button onclick="hideModal('task-form-modal')" class="text-gray-400 hover:text-gray-600">
				<svg class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
				</svg>
			</button>
		</div>

		<form hx-post="/app/tasks"
			  hx-target="#tasks-list"
			  hx-swap="innerHTML"
			  hx-on::after-request="if(event.detail.successful) hideModal('task-form-modal')"
			  class="space-y-4">

			<div>
				<label for="task-name" class="block text-sm font-medium text-gray-700">Task Name</label>
				<input type="text"
					   id="task-name"
					   name="name"
					   required
					   class="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm"
					   placeholder="Enter task name...">
			</div>

			<div>
				<label for="task-description" class="block text-sm font-medium text-gray-700">Description (Optional)</label>
				<textarea id="task-description"
						  name="description"
						  rows="3"
						  class="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm"
						  placeholder="Enter task description..."></textarea>
			</div>

			<div>
				<label for="task-priority" class="block text-sm font-medium text-gray-700">Priority</label>
				<select id="task-priority"
						name="priority"
						class="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm">
					<option value="medium">Medium</option>
					<option value="low">Low</option>
					<option value="high">High</option>
					<option value="urgent">Urgent</option>
				</select>
			</div>

			<div>
				<label for="task-tags" class="block text-sm font-medium text-gray-700">Tags (Optional)</label>
				<input type="text"
					   id="task-tags"
					   name="tags"
					   class="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm"
					   placeholder="project, urgent, client-name (comma separated)">
			</div>

			<div class="flex justify-end space-x-3 pt-4">
				<button type="button"
						onclick="hideModal('task-form-modal')"
						class="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500">
					Cancel
				</button>
				<button type="submit"
						class="px-4 py-2 text-sm font-medium text-white bg-blue-600 border border-transparent rounded-md hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500">
					Create Task
				</button>
			</div>
		</form>
	</div>`

	c.Header("Content-Type", "text/html")
	c.String(http.StatusOK, formHTML)
}

// CreateTaskHandler handles task creation from the form
func (h *TaskHandler) CreateTaskHandler(c *gin.Context) {
	authContext, exists := c.Get("auth")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Get form data
	name := strings.TrimSpace(c.PostForm("name"))
	description := strings.TrimSpace(c.PostForm("description"))
	priority := c.PostForm("priority")
	tagsStr := strings.TrimSpace(c.PostForm("tags"))

	// Validate required fields
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Task name is required"})
		return
	}

	// Parse tags
	var tags []string
	if tagsStr != "" {
		for _, tag := range strings.Split(tagsStr, ",") {
			tag = strings.TrimSpace(tag)
			if tag != "" {
				tags = append(tags, tag)
			}
		}
	}

	// Create the task using the simple TaskService interface
	task, err := h.taskService.CreateTask(name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create task"})
		return
	}

	// Update the task with additional fields
	if description != "" {
		task.Description = description
	}
	if priority != "" {
		task.Priority = models.TaskPriority(priority)
	}
	if len(tags) > 0 {
		task.Tags = tags
	}

	// Save the updated task
	if err := h.taskService.UpdateTask(task); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update task details"})
		return
	}

	// Return updated task list
	h.renderTaskList(c, authContext.(*models.AuthContext))
}