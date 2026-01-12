package api

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/soarinferret/jats/internal/services"
)

type ReportHandlers struct {
	reportService *services.ReportService
}

func NewReportHandlers(reportService *services.ReportService) *ReportHandlers {
	return &ReportHandlers{
		reportService: reportService,
	}
}

// TimeBreakdownRequest represents the request for time breakdown report
type TimeBreakdownRequest struct {
	StartDate      string   `json:"start_date"`       // Format: YYYY-MM-DD
	EndDate        string   `json:"end_date"`         // Format: YYYY-MM-DD
	SavedQueryIDs  []uint   `json:"saved_query_ids"`  // List of saved query IDs
	ExcludedTags   []string `json:"excluded_tags"`    // Tags to exclude
}

// GetTimeBreakdownReport handles GET /api/v1/reports/time-breakdown
func (h *ReportHandlers) GetTimeBreakdownReport(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	startDateStr := r.URL.Query().Get("start_date")
	endDateStr := r.URL.Query().Get("end_date")
	savedQueryIDsStr := r.URL.Query().Get("saved_query_ids")
	excludedTagsStr := r.URL.Query().Get("excluded_tags")

	// Validate required parameters
	if startDateStr == "" || endDateStr == "" {
		SendBadRequest(w, "start_date and end_date are required", nil)
		return
	}

	// Parse dates
	startDate, err := time.Parse("2006-01-02", startDateStr)
	if err != nil {
		SendBadRequest(w, "invalid start_date format, expected YYYY-MM-DD", nil)
		return
	}

	endDate, err := time.Parse("2006-01-02", endDateStr)
	if err != nil {
		SendBadRequest(w, "invalid end_date format, expected YYYY-MM-DD", nil)
		return
	}

	// Validate date range
	if endDate.Before(startDate) {
		SendBadRequest(w, "end_date must be after start_date", nil)
		return
	}

	// Parse saved query IDs
	var savedQueryIDs []uint
	if savedQueryIDsStr != "" {
		idStrs := strings.Split(savedQueryIDsStr, ",")
		for _, idStr := range idStrs {
			id, err := strconv.ParseUint(strings.TrimSpace(idStr), 10, 32)
			if err != nil {
				SendBadRequest(w, "invalid saved_query_ids format", nil)
				return
			}
			savedQueryIDs = append(savedQueryIDs, uint(id))
		}
	}

	if len(savedQueryIDs) == 0 {
		SendBadRequest(w, "at least one saved_query_id is required", nil)
		return
	}

	// Parse excluded tags
	var excludedTags []string
	if excludedTagsStr != "" {
		tags := strings.Split(excludedTagsStr, ",")
		for _, tag := range tags {
			trimmed := strings.TrimSpace(tag)
			if trimmed != "" {
				excludedTags = append(excludedTags, trimmed)
			}
		}
	}

	// Generate report
	report, err := h.reportService.GenerateTimeBreakdownReport(startDate, endDate, savedQueryIDs, excludedTags)
	if err != nil {
		SendInternalError(w, "Failed to generate report: "+err.Error())
		return
	}

	SendSuccess(w, report, "Time breakdown report generated successfully")
}
