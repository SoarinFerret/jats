package frontend

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/soarinferret/jats/internal/models"
)

// TaskDetailHandler serves the task detail panel
func (h *TaskHandler) TaskDetailHandler(c *gin.Context) {
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

	// Get time entries for the task
	timeEntries := task.TimeEntries
	if timeEntries == nil {
		timeEntries = []models.TimeEntry{}
	}

	// Get comments for the task
	comments := task.Comments
	if comments == nil {
		comments = []models.Comment{}
	}

	// Get attachments for the task
	attachments := task.Attachments
	if attachments == nil {
		attachments = []models.Attachment{}
	}

	// Calculate total time
	var totalMinutes int
	for _, entry := range timeEntries {
		totalMinutes += entry.Duration
	}
	hours := totalMinutes / 60
	minutes := totalMinutes % 60

	// Check if task is part of email thread
	isEmailThread := task.EmailMessageID != ""

	// Check if task allows modifications (open or in-progress)
	allowModifications := task.Status == models.TaskStatusOpen || task.Status == models.TaskStatusInProgress

	// Generate task detail HTML
	detailHTML := fmt.Sprintf(`
	<div class="flex flex-col h-full">
		<!-- Task Header -->
		<div class="p-6 border-b border-gray-200 flex-shrink-0">
			<div class="flex items-start justify-between">
				<div class="flex-1">
					<h3 class="text-lg font-medium text-gray-900">%s</h3>
					<div class="mt-2 flex items-center space-x-4 text-sm text-gray-500">
						<span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-blue-100 text-blue-800">
							%s
						</span>
						<span>Total Time: %dh %dm</span>
						%s
					</div>`,
		task.Name,
		string(task.Status),
		hours,
		minutes,
		func() string {
			if isEmailThread {
				return `<span class="inline-flex items-center px-2 py-1 rounded-full text-xs bg-green-100 text-green-800">
					<svg class="w-3 h-3 mr-1" fill="none" viewBox="0 0 24 24" stroke="currentColor">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 8l7.89 4.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
					</svg>
					Email Thread
				</span>`
			}
			return ""
		}())

	if task.Description != "" {
		detailHTML += fmt.Sprintf(`<p class="mt-3 text-sm text-gray-600">%s</p>`, task.Description)
	}

	detailHTML += `
				</div>
				<button onclick="hideDetailPanels()" class="text-gray-400 hover:text-gray-600 p-1">
					<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
					</svg>
				</button>
			</div>
		</div>

		<!-- Timeline Content -->
		<div class="flex-1 overflow-auto p-6">
			<h4 class="text-md font-medium text-gray-900 mb-4">Timeline</h4>
			<div class="space-y-4">`

	// Combine and sort timeline items
	type TimelineItem struct {
		Type      string
		Time      time.Time
		Content   string
		IsPrivate bool
	}

	var timeline []TimelineItem

	// Add time entries
	for _, entry := range timeEntries {
		content := fmt.Sprintf(`
		<div class="flex items-start space-x-3">
			<div class="flex-shrink-0">
				<div class="w-8 h-8 bg-blue-100 rounded-full flex items-center justify-center">
					<svg class="w-4 h-4 text-blue-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
					</svg>
				</div>
			</div>
			<div class="flex-1 min-w-0">
				<p class="text-sm font-medium text-gray-900">Time logged: %d minutes</p>
				%s
				<p class="text-xs text-gray-500 mt-1">%s</p>
			</div>
		</div>`, entry.Duration,
			func() string {
				if entry.Description != "" {
					return fmt.Sprintf(`<p class="text-sm text-gray-600 mt-1">%s</p>`, entry.Description)
				}
				return ""
			}(),
			entry.CreatedAt.Format("Jan 2, 2006 at 3:04 PM"))

		timeline = append(timeline, TimelineItem{
			Type:    "time",
			Time:    entry.CreatedAt,
			Content: content,
		})
	}

	// Add comments
	for _, comment := range comments {
		iconClass := "bg-gray-100"
		iconSvg := `<svg class="w-4 h-4 text-gray-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
			<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-3.582 8-8 8a8.955 8.955 0 01-4.126-.98L3 21l1.98-5.874A8.955 8.955 0 013 12a8 8 0 018-8 8 8 0 018 8z" />
		</svg>`

		privateLabel := ""
		if comment.IsPrivate {
			iconClass = "bg-orange-100"
			iconSvg = `<svg class="w-4 h-4 text-orange-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
			</svg>`
			privateLabel = `<span class="inline-flex items-center px-2 py-0.5 rounded-full text-xs bg-orange-100 text-orange-800 ml-2">Private</span>`
		}

		fromLabel := ""
		if comment.FromEmail != "" {
			fromLabel = fmt.Sprintf(" from %s", comment.FromEmail)
		}

		attachmentHTML := ""
		if len(comment.Attachments) > 0 {
			attachmentHTML = `<div class="mt-2 space-y-2">`
			for _, attachment := range comment.Attachments {
				fileIcon := "📎"
				if strings.HasPrefix(attachment.ContentType, "image/") {
					fileIcon = "🖼️"
				} else if strings.Contains(attachment.ContentType, "pdf") {
					fileIcon = "📄"
				}
				attachmentHTML += fmt.Sprintf(`
					<div class="inline-flex items-center px-3 py-1 rounded-md bg-gray-100 text-sm">
						<span class="mr-1">%s</span>
						<a href="/app/attachments/%d" target="_blank" class="text-blue-600 hover:text-blue-800">%s</a>
					</div>`, fileIcon, attachment.ID, attachment.OriginalName)
			}
			attachmentHTML += `</div>`
		}

		content := fmt.Sprintf(`
		<div class="flex items-start space-x-3">
			<div class="flex-shrink-0">
				<div class="w-8 h-8 %s rounded-full flex items-center justify-center">
					%s
				</div>
			</div>
			<div class="flex-1 min-w-0">
				<div class="flex items-center">
					<p class="text-sm font-medium text-gray-900">Comment%s</p>
					%s
				</div>
				<div class="mt-1 text-sm text-gray-700 whitespace-pre-wrap">%s</div>
				%s
				<p class="text-xs text-gray-500 mt-2">%s</p>
			</div>
		</div>`, iconClass, iconSvg, fromLabel, privateLabel, comment.Content, attachmentHTML, comment.CreatedAt.Format("Jan 2, 2006 at 3:04 PM"))

		timeline = append(timeline, TimelineItem{
			Type:      "comment",
			Time:      comment.CreatedAt,
			Content:   content,
			IsPrivate: comment.IsPrivate,
		})
	}

	// Add task attachments (not linked to comments)
	for _, attachment := range attachments {
		fileIcon := "📎"
		if strings.HasPrefix(attachment.ContentType, "image/") {
			fileIcon = "🖼️"
		} else if strings.Contains(attachment.ContentType, "pdf") {
			fileIcon = "📄"
		}

		content := fmt.Sprintf(`
		<div class="flex items-start space-x-3">
			<div class="flex-shrink-0">
				<div class="w-8 h-8 bg-green-100 rounded-full flex items-center justify-center">
					<svg class="w-4 h-4 text-green-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15.172 7l-6.586 6.586a2 2 0 102.828 2.828l6.414-6.586a4 4 0 00-5.656-5.656l-6.415 6.585a6 6 0 108.486 8.486L20.5 13" />
					</svg>
				</div>
			</div>
			<div class="flex-1 min-w-0">
				<p class="text-sm font-medium text-gray-900">Attachment added</p>
				<div class="mt-1">
					<div class="inline-flex items-center px-3 py-1 rounded-md bg-gray-100 text-sm">
						<span class="mr-1">%s</span>
						<a href="/app/attachments/%d" target="_blank" class="text-blue-600 hover:text-blue-800">%s</a>
					</div>
				</div>
				<p class="text-xs text-gray-500 mt-2">%s</p>
			</div>
		</div>`, fileIcon, attachment.ID, attachment.OriginalName, attachment.CreatedAt.Format("Jan 2, 2006 at 3:04 PM"))

		timeline = append(timeline, TimelineItem{
			Type:    "attachment",
			Time:    attachment.CreatedAt,
			Content: content,
		})
	}

	// Sort timeline by time (newest first)
	for i := 0; i < len(timeline); i++ {
		for j := i + 1; j < len(timeline); j++ {
			if timeline[j].Time.After(timeline[i].Time) {
				timeline[i], timeline[j] = timeline[j], timeline[i]
			}
		}
	}

	// Add timeline items to HTML
	if len(timeline) == 0 {
		detailHTML += `<p class="text-sm text-gray-500">No activity yet.</p>`
	} else {
		for _, item := range timeline {
			detailHTML += `<div class="border-l-2 border-gray-200 pl-4 pb-4">` + item.Content + `</div>`
		}
	}

	detailHTML += `
			</div>
		</div>`

	// Add comment form only if task allows modifications
	if allowModifications {
		detailHTML += `
		<!-- Comment Form -->
		<div class="border-t border-gray-200 p-6 flex-shrink-0">
			<form hx-post="/app/tasks/` + taskIDStr + `/comments"
				  hx-target="#task-detail"
				  hx-swap="outerHTML">
				<div class="space-y-3">
					<div>
						<label for="comment-content" class="sr-only">Add a comment</label>
						<textarea name="content" id="comment-content" rows="3" required
								  placeholder="Add a comment..."
								  class="w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500"></textarea>
					</div>`

		// Add private/public toggle for email threads
		if isEmailThread {
			detailHTML += `
						<div class="flex items-center space-x-4">
							<div class="flex items-center">
								<input type="radio" id="comment-public" name="is_private" value="false" checked
									   class="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300">
								<label for="comment-public" class="ml-2 text-sm text-gray-700">
									<span class="font-medium">Public</span> - Visible to email participants
								</label>
							</div>
							<div class="flex items-center">
								<input type="radio" id="comment-private" name="is_private" value="true"
									   class="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300">
								<label for="comment-private" class="ml-2 text-sm text-gray-700">
									<span class="font-medium">Private</span> - Internal only
								</label>
							</div>
						</div>`
		} else {
			detailHTML += `<input type="hidden" name="is_private" value="false">`
		}

		detailHTML += `
						<div class="flex justify-end">
							<button type="submit"
									class="px-4 py-2 bg-blue-600 text-white text-sm font-medium rounded-md hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500">
								Add Comment
							</button>
						</div>
					</div>
				</form>
			</div>`
	} else {
		// Show message for closed/resolved tasks
		detailHTML += `
		<!-- Disabled Comment Form -->
		<div class="border-t border-gray-200 p-6 flex-shrink-0 bg-gray-50">
			<div class="text-center">
				<svg class="mx-auto h-8 w-8 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
				</svg>
				<p class="mt-2 text-sm font-medium text-gray-900">Comments disabled</p>
				<p class="text-sm text-gray-500">Cannot add comments to ` + string(task.Status) + ` tasks</p>
			</div>
		</div>`
	}

	detailHTML += `
	</div>`

	c.Header("Content-Type", "text/html")
	c.String(http.StatusOK, detailHTML)
}

// AddTaskCommentHandler handles adding comments to tasks
func (h *TaskHandler) AddTaskCommentHandler(c *gin.Context) {
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot add comments to " + string(task.Status) + " tasks"})
		return
	}

	// Get form data
	content := strings.TrimSpace(c.PostForm("content"))
	isPrivateStr := c.PostForm("is_private")

	if content == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Comment content is required"})
		return
	}

	isPrivate := isPrivateStr == "true"

	// Create comment
	comment := &models.Comment{
		TaskID:    uint(taskID),
		Content:   content,
		IsPrivate: isPrivate,
	}

	err = h.taskService.AddComment(uint(taskID), comment)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add comment"})
		return
	}

	// Reload the task detail panel to show the new comment
	h.TaskDetailHandler(c)
}