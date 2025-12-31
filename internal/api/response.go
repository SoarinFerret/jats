package api

import (
	"encoding/json"
	"net/http"
	"time"
)

// APIResponse represents the standard API response format
type APIResponse struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Message   string      `json:"message,omitempty"`
	Error     *APIError   `json:"error,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

// APIError represents error details in API responses
type APIError struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

// PaginationMeta represents pagination information
type PaginationMeta struct {
	Total  int `json:"total"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
	Pages  int `json:"pages"`
}

// PaginatedResponse wraps data with pagination info
type PaginatedResponse struct {
	Items      interface{}     `json:"items"`
	Pagination *PaginationMeta `json:"pagination"`
}

// SendSuccess sends a successful API response
func SendSuccess(w http.ResponseWriter, data interface{}, message string) {
	response := APIResponse{
		Success:   true,
		Data:      data,
		Message:   message,
		Timestamp: time.Now(),
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// SendCreated sends a 201 Created response
func SendCreated(w http.ResponseWriter, data interface{}, message string) {
	response := APIResponse{
		Success:   true,
		Data:      data,
		Message:   message,
		Timestamp: time.Now(),
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// SendNoContent sends a 204 No Content response
func SendNoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// SendError sends an error response
func SendError(w http.ResponseWriter, statusCode int, code, message string, details interface{}) {
	response := APIResponse{
		Success: false,
		Error: &APIError{
			Code:    code,
			Message: message,
			Details: details,
		},
		Timestamp: time.Now(),
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

// SendBadRequest sends a 400 Bad Request response
func SendBadRequest(w http.ResponseWriter, message string, details interface{}) {
	SendError(w, http.StatusBadRequest, "BAD_REQUEST", message, details)
}

// SendNotFound sends a 404 Not Found response
func SendNotFound(w http.ResponseWriter, message string) {
	SendError(w, http.StatusNotFound, "NOT_FOUND", message, nil)
}

// SendInternalError sends a 500 Internal Server Error response
func SendInternalError(w http.ResponseWriter, message string) {
	SendError(w, http.StatusInternalServerError, "INTERNAL_ERROR", message, nil)
}

// SendValidationError sends a 422 Unprocessable Entity response
func SendValidationError(w http.ResponseWriter, message string, details interface{}) {
	SendError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", message, details)
}

// SendPaginatedSuccess sends a paginated successful response
func SendPaginatedSuccess(w http.ResponseWriter, items interface{}, pagination *PaginationMeta, message string) {
	data := PaginatedResponse{
		Items:      items,
		Pagination: pagination,
	}
	
	response := APIResponse{
		Success:   true,
		Data:      data,
		Message:   message,
		Timestamp: time.Now(),
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}