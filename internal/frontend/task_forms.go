package frontend

import (
	"fmt"
	"net/http"
	"strconv"
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

// EditTaskFormHandler serves the edit task form modal
func (h *TaskHandler) EditTaskFormHandler(c *gin.Context) {
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

	// Convert tags array to comma-separated string
	tagsStr := strings.Join(task.Tags, ", ")

	formHTML := fmt.Sprintf(`
	<div class="relative bg-white rounded-lg shadow-xl max-w-md w-full mx-auto mt-20 p-6">
		<div class="flex justify-between items-center mb-4">
			<h3 class="text-lg font-medium text-gray-900">Edit Task</h3>
			<button onclick="hideModal('task-edit-modal')" class="text-gray-400 hover:text-gray-600">
				<svg class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
				</svg>
			</button>
		</div>

		<form hx-put="/app/tasks/%s"
			  hx-target="#task-detail"
			  hx-swap="innerHTML"
			  hx-on::after-request="if(event.detail.successful) hideModal('task-edit-modal')"
			  class="space-y-4">

			<div>
				<label for="edit-task-name" class="block text-sm font-medium text-gray-700">Task Name</label>
				<input type="text"
					   id="edit-task-name"
					   name="name"
					   value="%s"
					   required
					   class="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm">
			</div>

			<div>
				<label for="edit-task-description" class="block text-sm font-medium text-gray-700">Description</label>
				<textarea id="edit-task-description"
						  name="description"
						  rows="3"
						  class="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm">%s</textarea>
			</div>

			<div>
				<label for="edit-task-status" class="block text-sm font-medium text-gray-700">Status</label>
				<select id="edit-task-status"
						name="status"
						class="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm">
					<option value="open" %s>Open</option>
					<option value="in-progress" %s>In Progress</option>
					<option value="resolved" %s>Resolved</option>
					<option value="closed" %s>Closed</option>
				</select>
			</div>

			<div>
				<label for="edit-task-priority" class="block text-sm font-medium text-gray-700">Priority</label>
				<select id="edit-task-priority"
						name="priority"
						class="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm">
					<option value="low" %s>Low</option>
					<option value="medium" %s>Medium</option>
					<option value="high" %s>High</option>
				</select>
			</div>

			<div>
				<label for="edit-task-tags" class="block text-sm font-medium text-gray-700">Tags</label>
				<input type="text"
					   id="edit-task-tags"
					   name="tags"
					   value="%s"
					   class="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm"
					   placeholder="project, urgent, client-name (comma separated)">
			</div>

			<div class="flex justify-end space-x-3 pt-4">
				<button type="button"
						onclick="hideModal('task-edit-modal')"
						class="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500">
					Cancel
				</button>
				<button type="submit"
						class="px-4 py-2 text-sm font-medium text-white bg-blue-600 border border-transparent rounded-md hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500">
					Update Task
				</button>
			</div>
		</form>
	</div>`,
		taskIDStr,
		task.Name,
		task.Description,
		func() string { if task.Status == models.TaskStatusOpen { return "selected" }; return "" }(),
		func() string { if task.Status == models.TaskStatusInProgress { return "selected" }; return "" }(),
		func() string { if task.Status == models.TaskStatusResolved { return "selected" }; return "" }(),
		func() string { if task.Status == models.TaskStatusClosed { return "selected" }; return "" }(),
		func() string { if task.Priority == models.TaskPriorityLow { return "selected" }; return "" }(),
		func() string { if task.Priority == models.TaskPriorityMedium { return "selected" }; return "" }(),
		func() string { if task.Priority == models.TaskPriorityHigh { return "selected" }; return "" }(),
		tagsStr)

	c.Header("Content-Type", "text/html")
	c.String(http.StatusOK, formHTML)
}

// UpdateTaskHandler handles task updates from the edit form
func (h *TaskHandler) UpdateTaskHandler(c *gin.Context) {
	taskIDStr := c.Param("id")
	taskID, err := strconv.ParseUint(taskIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID"})
		return
	}

	// Get the existing task
	task, err := h.taskService.GetTask(uint(taskID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		return
	}

	// Get form data
	name := strings.TrimSpace(c.PostForm("name"))
	description := strings.TrimSpace(c.PostForm("description"))
	status := c.PostForm("status")
	priority := c.PostForm("priority")
	tagsStr := strings.TrimSpace(c.PostForm("tags"))

	// Validate required fields
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Task name is required"})
		return
	}

	// Update task fields
	task.Name = name
	task.Description = description
	if status != "" {
		task.Status = models.TaskStatus(status)
	}
	if priority != "" {
		task.Priority = models.TaskPriority(priority)
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
	task.Tags = tags

	// Save the updated task
	if err := h.taskService.UpdateTask(task); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update task"})
		return
	}

	// Return updated task detail view
	h.TaskDetailHandler(c)
}