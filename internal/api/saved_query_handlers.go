package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/soarinferret/jats/internal/models"
	"github.com/soarinferret/jats/internal/services"
)

type SavedQueryHandlers struct {
	taskService *services.TaskService
}

func NewSavedQueryHandlers(taskService *services.TaskService) *SavedQueryHandlers {
	return &SavedQueryHandlers{
		taskService: taskService,
	}
}

func (h *SavedQueryHandlers) GetSavedQueries(w http.ResponseWriter, r *http.Request) {
	queries, err := h.taskService.GetSavedQueries()
	if err != nil {
		SendInternalError(w, "Failed to retrieve saved queries")
		return
	}

	SendSuccess(w, queries, "Saved queries retrieved successfully")
}

func (h *SavedQueryHandlers) GetSavedQuery(w http.ResponseWriter, r *http.Request) {
	c := r.Context().Value("gin").(*gin.Context)
	idParam := c.Param("id")
	
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		SendBadRequest(w, "Invalid query ID", nil)
		return
	}
	
	query, err := h.taskService.GetSavedQueryByID(uint(id))
	if err != nil {
		SendNotFound(w, "Saved query not found")
		return
	}

	SendSuccess(w, query, "Saved query retrieved successfully")
}

func (h *SavedQueryHandlers) CreateSavedQuery(w http.ResponseWriter, r *http.Request) {
	var query models.SavedQuery
	
	if err := json.NewDecoder(r.Body).Decode(&query); err != nil {
		SendBadRequest(w, "Invalid JSON", nil)
		return
	}

	if query.Name == "" {
		SendBadRequest(w, "Query name is required", nil)
		return
	}

	createdQuery, err := h.taskService.CreateSavedQuery(&query)
	if err != nil {
		SendInternalError(w, "Failed to create saved query")
		return
	}

	SendCreated(w, createdQuery, "Saved query created successfully")
}

func (h *SavedQueryHandlers) UpdateSavedQuery(w http.ResponseWriter, r *http.Request) {
	c := r.Context().Value("gin").(*gin.Context)
	idParam := c.Param("id")
	
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		SendBadRequest(w, "Invalid query ID", nil)
		return
	}

	existing, err := h.taskService.GetSavedQueryByID(uint(id))
	if err != nil {
		SendNotFound(w, "Saved query not found")
		return
	}

	var updates models.SavedQuery
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		SendBadRequest(w, "Invalid JSON", nil)
		return
	}

	if updates.Name != "" {
		existing.Name = updates.Name
	}
	if updates.IncludedTags != nil {
		existing.IncludedTags = updates.IncludedTags
	}
	if updates.ExcludedTags != nil {
		existing.ExcludedTags = updates.ExcludedTags
	}

	updatedQuery, err := h.taskService.UpdateSavedQuery(existing)
	if err != nil {
		SendInternalError(w, "Failed to update saved query")
		return
	}

	SendSuccess(w, updatedQuery, "Saved query updated successfully")
}

func (h *SavedQueryHandlers) DeleteSavedQuery(w http.ResponseWriter, r *http.Request) {
	c := r.Context().Value("gin").(*gin.Context)
	idParam := c.Param("id")
	
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		SendBadRequest(w, "Invalid query ID", nil)
		return
	}

	err = h.taskService.DeleteSavedQuery(uint(id))
	if err != nil {
		SendInternalError(w, "Failed to delete saved query")
		return
	}

	SendNoContent(w)
}

func (h *SavedQueryHandlers) GetTasksBySavedQuery(w http.ResponseWriter, r *http.Request) {
	c := r.Context().Value("gin").(*gin.Context)
	idParam := c.Param("id")
	
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		SendBadRequest(w, "Invalid query ID", nil)
		return
	}

	query, err := h.taskService.GetSavedQueryByID(uint(id))
	if err != nil {
		SendNotFound(w, "Saved query not found")
		return
	}

	tasks, err := h.taskService.GetTasksBySavedQuery(query)
	if err != nil {
		SendInternalError(w, "Failed to retrieve tasks")
		return
	}

	SendSuccess(w, tasks, "Tasks retrieved successfully")
}