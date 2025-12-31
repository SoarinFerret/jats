package api

import (
	"net/http"
	"strings"

	"github.com/soarinferret/jats/internal/models"
	"github.com/soarinferret/jats/internal/services"
)

type TagHandlers struct {
	taskService *services.TaskService
}

func NewTagHandlers(taskService *services.TaskService) *TagHandlers {
	return &TagHandlers{
		taskService: taskService,
	}
}

// TagInfo represents tag information with usage statistics
type TagInfo struct {
	Name     string `json:"name"`
	Count    int    `json:"count"`
	LastUsed string `json:"last_used"`
}

// GetTags handles GET /api/v1/tags
func (h *TagHandlers) GetTags(w http.ResponseWriter, r *http.Request) {
	// Get all tasks to analyze tags
	tasks, err := h.taskService.GetTasks()
	if err != nil {
		SendInternalError(w, "Failed to retrieve tasks")
		return
	}
	
	// Count tag usage
	tagCounts := make(map[string]int)
	tagLastUsed := make(map[string]string)
	
	for _, task := range tasks {
		for _, tag := range task.Tags {
			tagCounts[tag]++
			// Use task updated time as last used time
			if tagLastUsed[tag] == "" || task.UpdatedAt.Format("2006-01-02T15:04:05Z") > tagLastUsed[tag] {
				tagLastUsed[tag] = task.UpdatedAt.Format("2006-01-02T15:04:05Z")
			}
		}
	}
	
	// Convert to response format
	var tagInfos []TagInfo
	for tag, count := range tagCounts {
		tagInfos = append(tagInfos, TagInfo{
			Name:     tag,
			Count:    count,
			LastUsed: tagLastUsed[tag],
		})
	}
	
	response := map[string]interface{}{
		"tags": tagInfos,
	}
	
	SendSuccess(w, response, "Tags retrieved successfully")
}

// GetTasksByTag handles GET /api/v1/tags/{tag}/tasks
func (h *TagHandlers) GetTasksByTag(w http.ResponseWriter, r *http.Request) {
	tag := r.PathValue("tag")
	if tag == "" {
		SendBadRequest(w, "Tag is required", nil)
		return
	}
	
	// Get all tasks and filter by tag
	tasks, err := h.taskService.GetTasks()
	if err != nil {
		SendInternalError(w, "Failed to retrieve tasks")
		return
	}
	
	var filteredTasks []*models.Task
	for _, task := range tasks {
		for _, taskTag := range task.Tags {
			if taskTag == tag {
				filteredTasks = append(filteredTasks, task)
				break
			}
		}
	}
	
	SendSuccess(w, filteredTasks, "Tasks retrieved successfully")
}

// AddTaskTags handles POST /api/v1/tasks/{id}/tags
func (h *TagHandlers) AddTaskTags(w http.ResponseWriter, r *http.Request) {
	taskID, err := GetIDFromPath(r)
	if err != nil {
		SendBadRequest(w, "Invalid task ID", nil)
		return
	}
	
	var req struct {
		Tags []string `json:"tags"`
	}
	if err := ParseJSON(r, &req); err != nil {
		SendBadRequest(w, "Invalid JSON", err.Error())
		return
	}
	
	if len(req.Tags) == 0 {
		SendBadRequest(w, "At least one tag is required", nil)
		return
	}
	
	// Get existing task
	task, err := h.taskService.GetTask(taskID)
	if err != nil {
		SendNotFound(w, "Task not found")
		return
	}
	
	// Add new tags (avoiding duplicates)
	tagMap := make(map[string]bool)
	for _, tag := range task.Tags {
		tagMap[tag] = true
	}
	
	for _, tag := range req.Tags {
		tag = strings.TrimSpace(tag)
		if tag != "" && !tagMap[tag] {
			task.Tags = append(task.Tags, tag)
			tagMap[tag] = true
		}
	}
	
	// Update task
	if err := h.taskService.UpdateTask(task); err != nil {
		SendInternalError(w, "Failed to update task tags")
		return
	}
	
	SendSuccess(w, task, "Tags added successfully")
}

// RemoveTaskTag handles DELETE /api/v1/tasks/{id}/tags/{tag}
func (h *TagHandlers) RemoveTaskTag(w http.ResponseWriter, r *http.Request) {
	taskID, err := GetIDFromPath(r)
	if err != nil {
		SendBadRequest(w, "Invalid task ID", nil)
		return
	}
	
	tagToRemove := r.PathValue("tag")
	if tagToRemove == "" {
		SendBadRequest(w, "Tag is required", nil)
		return
	}
	
	// Get existing task
	task, err := h.taskService.GetTask(taskID)
	if err != nil {
		SendNotFound(w, "Task not found")
		return
	}
	
	// Remove tag
	var newTags []string
	found := false
	for _, tag := range task.Tags {
		if tag != tagToRemove {
			newTags = append(newTags, tag)
		} else {
			found = true
		}
	}
	
	if !found {
		SendNotFound(w, "Tag not found on task")
		return
	}
	
	task.Tags = newTags
	
	// Update task
	if err := h.taskService.UpdateTask(task); err != nil {
		SendInternalError(w, "Failed to update task tags")
		return
	}
	
	SendSuccess(w, task, "Tag removed successfully")
}