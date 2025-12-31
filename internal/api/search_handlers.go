package api

import (
	"net/http"
	"strings"

	"github.com/soarinferret/jats/internal/models"
	"github.com/soarinferret/jats/internal/services"
)

type SearchHandlers struct {
	taskService *services.TaskService
}

func NewSearchHandlers(taskService *services.TaskService) *SearchHandlers {
	return &SearchHandlers{
		taskService: taskService,
	}
}

// SearchResult represents a search result item
type SearchResult struct {
	ID        uint    `json:"id"`
	Type      string  `json:"type"` // "task", "comment"
	Name      string  `json:"name,omitempty"`
	Content   string  `json:"content,omitempty"`
	TaskID    *uint   `json:"task_id,omitempty"`
	MatchType string  `json:"match_type"` // "name", "description", "content"
	Score     float64 `json:"score"`
}

// SearchResponse represents the complete search response
type SearchResponse struct {
	Query   string                   `json:"query"`
	Results map[string][]SearchResult `json:"results"`
	Total   int                      `json:"total"`
}

// Search handles GET /api/v1/search
func (h *SearchHandlers) Search(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		SendBadRequest(w, "Query parameter 'q' is required", nil)
		return
	}
	
	searchType := r.URL.Query().Get("type")
	
	// Get all tasks
	tasks, err := h.taskService.GetTasks()
	if err != nil {
		SendInternalError(w, "Failed to retrieve tasks")
		return
	}
	
	// Apply additional filters if provided
	filters := ParseTaskFilters(r.URL.Query())
	filteredTasks := h.applyFilters(tasks, filters)
	
	// Perform search
	response := SearchResponse{
		Query:   query,
		Results: make(map[string][]SearchResult),
		Total:   0,
	}
	
	queryLower := strings.ToLower(query)
	
	// Search in tasks (if type not specified or is "task")
	if searchType == "" || searchType == "task" {
		var taskResults []SearchResult
		
		for _, task := range filteredTasks {
			// Search in task name
			if strings.Contains(strings.ToLower(task.Name), queryLower) {
				result := SearchResult{
					ID:        task.ID,
					Type:      "task",
					Name:      task.Name,
					MatchType: "name",
					Score:     calculateScore(task.Name, query),
				}
				taskResults = append(taskResults, result)
			} else if strings.Contains(strings.ToLower(task.Description), queryLower) {
				// Search in description if not found in name
				result := SearchResult{
					ID:        task.ID,
					Type:      "task", 
					Name:      task.Name,
					Content:   task.Description,
					MatchType: "description",
					Score:     calculateScore(task.Description, query),
				}
				taskResults = append(taskResults, result)
			}
		}
		
		response.Results["tasks"] = taskResults
		response.Total += len(taskResults)
	}
	
	// Search in comments (if type not specified or is "comment")
	if searchType == "" || searchType == "comment" {
		// This would need to be implemented when comment retrieval is available
		// For now, return empty results
		response.Results["comments"] = []SearchResult{}
	}
	
	SendSuccess(w, response, "Search completed successfully")
}

// KanbanResponse represents a kanban board view
type KanbanResponse struct {
	Project    string                              `json:"project,omitempty"`
	Columns    map[string][]*models.Task          `json:"columns"`
	Statistics map[string]int                     `json:"statistics"`
}

// GetKanban handles GET /api/v1/kanban
func (h *SearchHandlers) GetKanban(w http.ResponseWriter, r *http.Request) {
	// Get all tasks
	tasks, err := h.taskService.GetTasks()
	if err != nil {
		SendInternalError(w, "Failed to retrieve tasks")
		return
	}
	
	// Apply filters
	filters := ParseTaskFilters(r.URL.Query())
	filteredTasks := h.applyFilters(tasks, filters)
	
	// Organize into kanban columns
	columns := map[string][]*models.Task{
		"open":        {},
		"in-progress": {},
		"resolved":    {},
		"closed":      {},
	}
	
	statistics := map[string]int{
		"total":       0,
		"open":        0,
		"in-progress": 0,
		"resolved":    0,
		"closed":      0,
	}
	
	for _, task := range filteredTasks {
		statusStr := string(task.Status)
		columns[statusStr] = append(columns[statusStr], task)
		statistics[statusStr]++
		statistics["total"]++
	}
	
	response := KanbanResponse{
		Columns:    columns,
		Statistics: statistics,
	}
	
	SendSuccess(w, response, "Kanban board retrieved successfully")
}

// GetKanbanByTag handles GET /api/v1/kanban/{tag}
func (h *SearchHandlers) GetKanbanByTag(w http.ResponseWriter, r *http.Request) {
	tag := r.PathValue("tag")
	if tag == "" {
		SendBadRequest(w, "Tag is required", nil)
		return
	}
	
	// Get all tasks
	tasks, err := h.taskService.GetTasks()
	if err != nil {
		SendInternalError(w, "Failed to retrieve tasks")
		return
	}
	
	// Filter by tag
	var taggedTasks []*models.Task
	for _, task := range tasks {
		for _, taskTag := range task.Tags {
			if taskTag == tag {
				taggedTasks = append(taggedTasks, task)
				break
			}
		}
	}
	
	// Apply additional filters
	filters := ParseTaskFilters(r.URL.Query())
	filteredTasks := h.applyFiltersForKanban(taggedTasks, filters)
	
	// Organize into kanban columns
	columns := map[string][]*models.Task{
		"open":        {},
		"in-progress": {},
		"resolved":    {},
		"closed":      {},
	}
	
	statistics := map[string]int{
		"total":       0,
		"open":        0,
		"in-progress": 0,
		"resolved":    0,
		"closed":      0,
	}
	
	for _, task := range filteredTasks {
		statusStr := string(task.Status)
		columns[statusStr] = append(columns[statusStr], task)
		statistics[statusStr]++
		statistics["total"]++
	}
	
	response := KanbanResponse{
		Project:    tag,
		Columns:    columns,
		Statistics: statistics,
	}
	
	SendSuccess(w, response, "Kanban board retrieved successfully")
}

// Helper function to calculate search relevance score
func calculateScore(text, query string) float64 {
	textLower := strings.ToLower(text)
	queryLower := strings.ToLower(query)
	
	// Simple scoring based on match position and frequency
	score := 0.0
	
	// Exact match gets highest score
	if textLower == queryLower {
		score = 1.0
	} else if strings.HasPrefix(textLower, queryLower) {
		// Prefix match gets high score
		score = 0.9
	} else if strings.Contains(textLower, queryLower) {
		// Contains match gets medium score
		score = 0.7
		
		// Boost score based on position (earlier = better)
		index := strings.Index(textLower, queryLower)
		if index == 0 {
			score = 0.9
		} else if index < len(textLower)/4 {
			score = 0.8
		}
	}
	
	return score
}

// Helper method to apply filters (reuse from task handlers)
func (h *SearchHandlers) applyFilters(tasks []*models.Task, filters TaskFilters) []*models.Task {
	var filtered []*models.Task
	
	for _, task := range tasks {
		// Status filter
		if len(filters.Status) > 0 {
			found := false
			for _, status := range filters.Status {
				if task.Status == status {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		
		// Priority filter
		if len(filters.Priority) > 0 {
			found := false
			for _, priority := range filters.Priority {
				if task.Priority == priority {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		
		// Tags filter (task must have at least one matching tag)
		if len(filters.Tags) > 0 {
			found := false
			for _, filterTag := range filters.Tags {
				for _, taskTag := range task.Tags {
					if taskTag == filterTag {
						found = true
						break
					}
				}
				if found {
					break
				}
			}
			if !found {
				continue
			}
		}
		
		filtered = append(filtered, task)
	}
	
	return filtered
}

// Helper method for kanban filtering (similar to applyFilters but without search)
func (h *SearchHandlers) applyFiltersForKanban(tasks []*models.Task, filters TaskFilters) []*models.Task {
	var filtered []*models.Task
	
	for _, task := range tasks {
		// Priority filter
		if len(filters.Priority) > 0 {
			found := false
			for _, priority := range filters.Priority {
				if task.Priority == priority {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		
		filtered = append(filtered, task)
	}
	
	return filtered
}