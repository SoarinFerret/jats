package common

import (
	"encoding/json"
	"net/http"
	"time"
)

// APIResponse represents a standard API response
type APIResponse struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Message   string      `json:"message,omitempty"`
	Error     *APIError   `json:"error,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

// APIError represents an API error
type APIError struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

// SendSuccessResponse sends a successful API response
func SendSuccessResponse(w http.ResponseWriter, status int, data interface{}, message string) {
	response := APIResponse{
		Success:   true,
		Data:      data,
		Message:   message,
		Timestamp: time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(response)
}

// SendErrorResponse sends an error API response
func SendErrorResponse(w http.ResponseWriter, status int, code, message string, details interface{}) {
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
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(response)
}

// Legacy aliases for existing API compatibility
var SendError = SendErrorResponse

// SendSuccess sends a 200 OK response
func SendSuccess(w http.ResponseWriter, data interface{}, message string) {
	SendSuccessResponse(w, http.StatusOK, data, message)
}

// SendCreated sends a 201 Created response
func SendCreated(w http.ResponseWriter, data interface{}, message string) {
	SendSuccessResponse(w, http.StatusCreated, data, message)
}