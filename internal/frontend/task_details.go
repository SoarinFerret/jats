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
				<div class="flex items-center space-x-2">
					<button hx-get="/app/tasks/` + taskIDStr + `/edit" 
							hx-target="#task-edit-modal" 
							hx-trigger="click"
							onclick="showModal('task-edit-modal')"
							class="text-blue-600 hover:text-blue-800 p-2 rounded-md hover:bg-blue-50"
							title="Edit Task">
						<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
						</svg>
					</button>
					<button onclick="hideDetailPanels()" class="text-gray-400 hover:text-gray-600 p-1">
						<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
						</svg>
					</button>
				</div>
			</div>
		</div>

		<!-- Timeline Content -->
		<div class="flex-1 overflow-auto p-6">
			<h4 class="text-md font-medium text-gray-900 mb-4">Timeline</h4>
			<div id="timeline-content-` + taskIDStr + `" class="space-y-4">`

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

		// All comments are now internal notes - use consistent styling
		iconClass = "bg-blue-100"
		iconSvg = `<svg class="w-4 h-4 text-blue-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
			<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-3.582 8-8 8a8.955 8.955 0 01-4.126-.98L3 21l1.98-5.874A8.955 8.955 0 013 12a8 8 0 018-8 8 8 0 018 8z" />
		</svg>`
		privateLabel := ""

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
					<p class="text-sm font-medium text-gray-900">Note%s</p>
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
				  hx-target="#timeline-content-` + taskIDStr + `"
				  hx-swap="innerHTML"
				  hx-on::after-request="if(event.detail.xhr.status === 200) { this.reset(); this.querySelector('textarea').focus(); }"
				  hx-indicator="#submit-indicator-` + taskIDStr + `">
				<div class="space-y-3">
					<div>
						<label for="comment-content-` + taskIDStr + `" class="sr-only">Add an internal note</label>
						<textarea name="content" id="comment-content-` + taskIDStr + `" rows="3" required
								  placeholder="Add an internal note... (Ctrl+Enter to submit)"
								  onkeydown="handleCommentKeydown(event, this.form)"
								  class="w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500"></textarea>
					</div>
					<input type="hidden" name="is_private" value="true">
					<div class="flex justify-between items-center">
						<div id="submit-indicator-` + taskIDStr + `" class="htmx-indicator text-sm text-gray-600">
							<svg class="animate-spin -ml-1 mr-2 h-4 w-4 text-gray-600 inline" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
								<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
								<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
							</svg>
							Adding note...
						</div>
						<div class="flex gap-3">
							<button type="button"
									onclick="showTimeEntryModal('` + taskIDStr + `')"
									class="px-4 py-2 bg-green-600 text-white text-sm font-medium rounded-md hover:bg-green-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-green-500">
								<svg class="inline mr-1.5 h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
								</svg>
								Add Time
							</button>
							<button type="submit"
									class="px-4 py-2 bg-blue-600 text-white text-sm font-medium rounded-md hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50">
								Add Note
							</button>
						</div>
					</div>
				</div>
			</form>
		</div>`
	} else {
		// Show message for closed/resolved tasks
		detailHTML += `
		<!-- Disabled Note Form -->
		<div class="border-t border-gray-200 p-6 flex-shrink-0 bg-gray-50">
			<div class="text-center">
				<svg class="mx-auto h-8 w-8 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
				</svg>
				<p class="mt-2 text-sm font-medium text-gray-900">Notes disabled</p>
				<p class="text-sm text-gray-500">Cannot add notes to ` + string(task.Status) + ` tasks</p>
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot add notes to " + string(task.Status) + " tasks"})
		return
	}

	// Get form data
	content := strings.TrimSpace(c.PostForm("content"))
	isPrivateStr := c.PostForm("is_private")

	if content == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Note content is required"})
		return
	}

	// All comments are now private (internal notes only)
	isPrivate := true
	if isPrivateStr != "" {
		isPrivate = isPrivateStr == "true"
	}

	// Create comment
	comment := &models.Comment{
		TaskID:    uint(taskID),
		Content:   content,
		IsPrivate: isPrivate,
	}

	err = h.taskService.AddComment(uint(taskID), comment)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add note"})
		return
	}

	// Return just the timeline content to update the specific section
	h.TaskTimelineHandler(c)
}

// TaskTimelineHandler returns just the timeline content for HTMX updates
func (h *TaskHandler) TaskTimelineHandler(c *gin.Context) {
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

	// Generate timeline HTML (reuse the same logic from TaskDetailHandler)
	timelineHTML := h.generateTimelineHTML(timeEntries, comments, attachments)

	c.Header("Content-Type", "text/html")
	c.String(http.StatusOK, timelineHTML)
}

// generateTimelineHTML extracts the timeline generation logic for reuse
func (h *TaskHandler) generateTimelineHTML(timeEntries []models.TimeEntry, comments []models.Comment, attachments []models.Attachment) string {
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
		// All comments are now internal notes - use consistent styling
		iconClass := "bg-blue-100"
		iconSvg := `<svg class="w-4 h-4 text-blue-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
			<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-3.582 8-8 8a8.955 8.955 0 01-4.126-.98L3 21l1.98-5.874A8.955 8.955 0 013 12a8 8 0 018-8 8 8 0 018 8z" />
		</svg>`
		privateLabel := ""

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
					<p class="text-sm font-medium text-gray-900">Note%s</p>
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

	// Generate timeline HTML
	var timelineHTML string
	if len(timeline) == 0 {
		timelineHTML = `<p class="text-sm text-gray-500">No activity yet.</p>`
	} else {
		for _, item := range timeline {
			timelineHTML += `<div class="border-l-2 border-gray-200 pl-4 pb-4">` + item.Content + `</div>`
		}
	}

	return timelineHTML
}

// AddTimeEntryHandler handles adding time entries to tasks
func (h *TaskHandler) AddTimeEntryHandler(c *gin.Context) {
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot add time entries to " + string(task.Status) + " tasks"})
		return
	}

	// Get form data
	durationStr := strings.TrimSpace(c.PostForm("duration"))
	description := strings.TrimSpace(c.PostForm("description"))

	if durationStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Duration is required"})
		return
	}

	duration, err := strconv.Atoi(durationStr)
	if err != nil || duration < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Duration must be a positive number"})
		return
	}

	// Create time entry
	timeEntry := &models.TimeEntry{
		TaskID:      uint(taskID),
		Duration:    duration,
		Description: description,
	}

	err = h.taskService.AddTimeEntry(uint(taskID), timeEntry)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add time entry"})
		return
	}

	// Return success
	c.JSON(http.StatusOK, gin.H{"success": true})
}