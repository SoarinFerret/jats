package frontend

import (
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/soarinferret/jats/internal/models"
	"github.com/soarinferret/jats/internal/services"
)

// SavedQueryHandler handles saved query-related frontend requests
type SavedQueryHandler struct {
	taskService *services.TaskService
	templates   map[string]*template.Template
}

// NewSavedQueryHandler creates a new saved query handler
func NewSavedQueryHandler(taskService *services.TaskService, templates map[string]*template.Template) *SavedQueryHandler {
	return &SavedQueryHandler{
		taskService: taskService,
		templates:   templates,
	}
}

// SavedQueriesListHandler renders the saved queries list
func (h *SavedQueryHandler) SavedQueriesListHandler(c *gin.Context) {
	queries, err := h.taskService.GetSavedQueries()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get saved queries"})
		return
	}

	// Check context to determine link targets
	context := c.PostForm("context")
	if context == "" {
		context = c.Query("context")
	}
	if context == "" {
		context = "tasks" // default context
	}

	queriesHTML := ""
	for _, query := range queries {
		var linkTarget, linkURL string
		var iconPath string
		var onclickAction string

		if context == "reports" {
			linkTarget = "#main-content"
			linkURL = fmt.Sprintf("/app/reports?query=%d", query.ID)
			iconPath = "M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z"
			onclickAction = ""
		} else {
			linkTarget = "#main-content"
			linkURL = fmt.Sprintf("/app/saved-queries/%d/tasks", query.ID)
			iconPath = "M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
			onclickAction = fmt.Sprintf("setActiveTaskView(this, 'query-%d')", query.ID)
		}

		queriesHTML += fmt.Sprintf(`
		<a href="#"
		   hx-get="%s"
		   hx-target="%s"
		   hx-trigger="click"
		   onclick="%s"
		   class="task-view-item flex items-center px-3 py-2 text-xs font-medium rounded-md text-gray-700 hover:bg-gray-100 hover:text-gray-900 group">
			<svg class="mr-2 h-3 w-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="%s" />
			</svg>
			<span class="flex-1 truncate">%s</span>
			<button hx-delete="/api/v1/saved-queries/%d"
					hx-target="closest .task-view-item"
					hx-swap="outerHTML"
					hx-confirm="Are you sure you want to delete this saved query?"
					onclick="event.stopPropagation()"
					class="opacity-0 group-hover:opacity-100 text-gray-400 hover:text-red-600 p-1 rounded">
				<svg class="h-2 w-2" fill="none" viewBox="0 0 24 24" stroke="currentColor">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
				</svg>
			</button>
		</a>`, linkURL, linkTarget, onclickAction, iconPath, query.Name, query.ID)
	}

	if len(queries) == 0 {
		queriesHTML = `<p class="text-xs text-gray-500 px-3 py-2">No saved queries</p>`
	}

	c.Header("Content-Type", "text/html")
	c.String(http.StatusOK, queriesHTML)
}

// NewSavedQueryFormHandler shows the new saved query form modal
func (h *SavedQueryHandler) NewSavedQueryFormHandler(c *gin.Context) {
	formHTML := `
	<div class="flex items-center justify-center min-h-screen">
		<div class="bg-white rounded-lg p-6 w-full max-w-md mx-4 shadow-xl">
			<div class="flex items-center justify-between mb-4">
				<h3 class="text-lg font-medium text-gray-900">New Saved Query</h3>
				<button onclick="hideModal('saved-query-modal')" class="text-gray-400 hover:text-gray-600">
					<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
					</svg>
				</button>
			</div>

			<form hx-post="/app/saved-queries"
				  hx-target="#saved-queries-list"
				  hx-on="htmx:afterRequest: hideModal('saved-query-modal')">
				<div class="space-y-4">
					<div>
						<label for="name" class="block text-sm font-medium text-gray-700">Name</label>
						<input type="text" name="name" id="name" required
							   class="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500"
							   placeholder="e.g., High Priority Client Tasks">
					</div>

					<div>
						<label for="included_tags" class="block text-sm font-medium text-gray-700">Include Tags</label>
						<input type="text" name="included_tags" id="included_tags"
							   class="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500"
							   placeholder="e.g., client1, urgent (comma-separated)">
						<p class="mt-1 text-xs text-gray-500">Tasks must have at least one of these tags</p>
					</div>

					<div>
						<label for="excluded_tags" class="block text-sm font-medium text-gray-700">Exclude Tags</label>
						<input type="text" name="excluded_tags" id="excluded_tags"
							   class="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500"
							   placeholder="e.g., archived, completed (comma-separated)">
						<p class="mt-1 text-xs text-gray-500">Tasks with any of these tags will be filtered out</p>
					</div>
				</div>

				<div class="mt-6 flex items-center justify-end space-x-3">
					<button type="button" onclick="hideModal('saved-query-modal')"
							class="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50">
						Cancel
					</button>
					<button type="submit"
							class="px-4 py-2 text-sm font-medium text-white bg-blue-600 border border-transparent rounded-md hover:bg-blue-700">
						Create Query
					</button>
				</div>
			</form>
		</div>
	</div>`

	c.Header("Content-Type", "text/html")
	c.String(http.StatusOK, formHTML)
}

// CreateSavedQueryHandler handles saved query creation from the form
func (h *SavedQueryHandler) CreateSavedQueryHandler(c *gin.Context) {
	_, exists := c.Get("auth")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Get form data
	name := strings.TrimSpace(c.PostForm("name"))
	includedTagsStr := strings.TrimSpace(c.PostForm("included_tags"))
	excludedTagsStr := strings.TrimSpace(c.PostForm("excluded_tags"))

	// Validate required fields
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Query name is required"})
		return
	}

	// Parse tags
	var includedTags []string
	if includedTagsStr != "" {
		for _, tag := range strings.Split(includedTagsStr, ",") {
			tag = strings.TrimSpace(tag)
			if tag != "" {
				includedTags = append(includedTags, tag)
			}
		}
	}

	var excludedTags []string
	if excludedTagsStr != "" {
		for _, tag := range strings.Split(excludedTagsStr, ",") {
			tag = strings.TrimSpace(tag)
			if tag != "" {
				excludedTags = append(excludedTags, tag)
			}
		}
	}

	// Create the saved query
	query := &models.SavedQuery{
		Name:         name,
		IncludedTags: includedTags,
		ExcludedTags: excludedTags,
	}

	_, err := h.taskService.CreateSavedQuery(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create saved query"})
		return
	}

	// Return updated saved queries list
	h.SavedQueriesListHandler(c)
}

// SavedQueryTasksHandler shows tasks for a specific saved query
func (h *SavedQueryHandler) SavedQueryTasksHandler(c *gin.Context, taskHandler *TaskHandler) {
	_, exists := c.Get("auth")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Get query ID from URL
	queryIDStr := c.Param("id")
	queryID, err := strconv.ParseUint(queryIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid query ID"})
		return
	}

	// Get the saved query
	query, err := h.taskService.GetSavedQueryByID(uint(queryID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Saved query not found"})
		return
	}

	// Get tasks filtered by saved query
	savedQueryTasks, err := h.taskService.GetTasksBySavedQuery(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get tasks"})
		return
	}

	// Add saved query info and filtered tasks to context
	c.Set("savedQuery", query)
	c.Set("savedQueryTasks", savedQueryTasks)

	// Reuse the existing TaskListHandler logic but with saved query applied
	taskHandler.TaskListHandler(c)
}