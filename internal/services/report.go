package services

import (
	"fmt"
	"time"

	"github.com/soarinferret/jats/internal/models"
	"github.com/soarinferret/jats/internal/repository"
)

type ReportService struct {
	taskRepo *repository.TaskRepository
}

func NewReportService(taskRepo *repository.TaskRepository) *ReportService {
	return &ReportService{
		taskRepo: taskRepo,
	}
}

// TimeBreakdownReport represents a time breakdown report by date and saved queries
type TimeBreakdownReport struct {
	StartDate   time.Time                   `json:"start_date"`
	EndDate     time.Time                   `json:"end_date"`
	QueryNames  []string                    `json:"query_names"`
	DailyData   []DailyTimeBreakdown        `json:"daily_data"`
	Totals      QueryTotals                 `json:"totals"`
}

// DailyTimeBreakdown represents time breakdown for a single day
type DailyTimeBreakdown struct {
	Date       string                 `json:"date"`
	TotalTime  int                    `json:"total_time"` // minutes
	QueryTimes []QueryTimeBreakdown   `json:"query_times"`
	OtherTime  int                    `json:"other_time"`  // minutes - time not matching any query
	OtherTags  []string               `json:"other_tags"`  // tags from tasks not matching any query
}

// QueryTimeBreakdown represents time for a specific saved query on a day
type QueryTimeBreakdown struct {
	QueryID   uint     `json:"query_id"`
	QueryName string   `json:"query_name"`
	Time      int      `json:"time"` // minutes
	Tags      []string `json:"tags"` // unique tags from time entries
}

// QueryTotals represents total time and percentages for each query
type QueryTotals struct {
	TotalTime   int                 `json:"total_time"` // minutes
	QueryTotals []QueryTotal        `json:"query_totals"`
	OtherTotal  OtherTotal          `json:"other_total"` // time not matching any query
}

// OtherTotal represents total for unmatched time
type OtherTotal struct {
	TotalTime  int      `json:"total_time"`
	Percentage float64  `json:"percentage"`
	Tags       []string `json:"tags"` // all tags from unmatched tasks
}

// QueryTotal represents total time and percentage for a specific query
type QueryTotal struct {
	QueryID    uint    `json:"query_id"`
	QueryName  string  `json:"query_name"`
	TotalTime  int     `json:"total_time"` // minutes
	Percentage float64 `json:"percentage"`
}

// GenerateTimeBreakdownReport generates a time breakdown report
func (s *ReportService) GenerateTimeBreakdownReport(startDate, endDate time.Time, savedQueryIDs []uint, excludedTags []string) (*TimeBreakdownReport, error) {
	// Get all saved queries
	queries, err := s.getSavedQueriesByIDs(savedQueryIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get saved queries: %w", err)
	}

	// Get all tasks
	allTasks, err := s.taskRepo.GetAll()
	if err != nil {
		return nil, fmt.Errorf("failed to get tasks: %w", err)
	}

	// Build query names list
	queryNames := make([]string, len(queries))
	for i, q := range queries {
		queryNames[i] = q.Name
	}

	// Generate daily breakdown
	dailyData := s.generateDailyBreakdown(startDate, endDate, allTasks, queries, excludedTags)

	// Calculate totals
	totals := s.calculateTotals(dailyData, queries)

	return &TimeBreakdownReport{
		StartDate:  startDate,
		EndDate:    endDate,
		QueryNames: queryNames,
		DailyData:  dailyData,
		Totals:     totals,
	}, nil
}

// getSavedQueriesByIDs retrieves saved queries by their IDs
func (s *ReportService) getSavedQueriesByIDs(ids []uint) ([]*models.SavedQuery, error) {
	queries := make([]*models.SavedQuery, 0, len(ids))
	for _, id := range ids {
		query, err := s.taskRepo.GetSavedQueryByID(id)
		if err != nil {
			return nil, fmt.Errorf("failed to get saved query %d: %w", id, err)
		}
		queries = append(queries, query)
	}
	return queries, nil
}

// generateDailyBreakdown generates daily time breakdown
func (s *ReportService) generateDailyBreakdown(startDate, endDate time.Time, tasks []*models.Task, queries []*models.SavedQuery, excludedTags []string) []DailyTimeBreakdown {
	dailyMap := make(map[string]*DailyTimeBreakdown)

	// Normalize start and end dates to beginning and end of day to handle timezone issues
	startDate = time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, time.UTC)
	endDate = time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 23, 59, 59, 999999999, time.UTC)

	// Initialize daily entries
	for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
		dateStr := d.Format("1/2/2006")
		dailyMap[dateStr] = &DailyTimeBreakdown{
			Date:       dateStr,
			TotalTime:  0,
			QueryTimes: make([]QueryTimeBreakdown, len(queries)),
			OtherTime:  0,
			OtherTags:  []string{},
		}
		// Initialize query times
		for i, query := range queries {
			dailyMap[dateStr].QueryTimes[i] = QueryTimeBreakdown{
				QueryID:   query.ID,
				QueryName: query.Name,
				Time:      0,
				Tags:      []string{},
			}
		}
	}

	// Process each task
	for _, task := range tasks {
		// Skip if task has any excluded tags
		if s.hasAnyTag(task.Tags, excludedTags) {
			continue
		}

		// Process each time entry
		for _, timeEntry := range task.TimeEntries {
			// Normalize entry date for comparison (convert to UTC for consistency)
			entryDateUTC := timeEntry.CreatedAt.UTC()

			// Check if time entry is within date range (inclusive)
			if entryDateUTC.Before(startDate) || entryDateUTC.After(endDate) {
				continue
			}

			// Use date with year for lookup
			dateStr := timeEntry.CreatedAt.Format("1/2/2006")
			daily, exists := dailyMap[dateStr]
			if !exists {
				continue
			}

			// Add to total time
			daily.TotalTime += timeEntry.Duration

			// Check if task matches any saved query
			matchedAnyQuery := false
			for i, query := range queries {
				if s.taskMatchesSavedQuery(task, query) {
					matchedAnyQuery = true
					daily.QueryTimes[i].Time += timeEntry.Duration
					// Add task tags to query tags, excluding the query's filter tags
					filteredTags := s.excludeQueryFilterTags(task.Tags, query)
					daily.QueryTimes[i].Tags = append(daily.QueryTimes[i].Tags, filteredTags...)
				}
			}

			// If didn't match any query, add to "Other"
			if !matchedAnyQuery {
				daily.OtherTime += timeEntry.Duration
				daily.OtherTags = append(daily.OtherTags, task.Tags...)
			}
		}
	}

	// Deduplicate tags and sort daily data, skipping days with no time
	var result []DailyTimeBreakdown
	for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
		dateStr := d.Format("1/2/2006")
		if daily, exists := dailyMap[dateStr]; exists {
			// Skip days with no time entries
			if daily.TotalTime == 0 {
				continue
			}

			// Deduplicate tags for each query
			for i := range daily.QueryTimes {
				daily.QueryTimes[i].Tags = s.deduplicateTags(daily.QueryTimes[i].Tags)
			}
			// Deduplicate tags for Other
			daily.OtherTags = s.deduplicateTags(daily.OtherTags)

			result = append(result, *daily)
		}
	}

	return result
}

// calculateTotals calculates total time and percentages
func (s *ReportService) calculateTotals(dailyData []DailyTimeBreakdown, queries []*models.SavedQuery) QueryTotals {
	totals := QueryTotals{
		TotalTime:   0,
		QueryTotals: make([]QueryTotal, len(queries)),
		OtherTotal: OtherTotal{
			TotalTime:  0,
			Percentage: 0,
			Tags:       []string{},
		},
	}

	// Initialize query totals
	for i, query := range queries {
		totals.QueryTotals[i] = QueryTotal{
			QueryID:   query.ID,
			QueryName: query.Name,
			TotalTime: 0,
		}
	}

	// Sum up daily data
	for _, daily := range dailyData {
		totals.TotalTime += daily.TotalTime
		for i, queryTime := range daily.QueryTimes {
			totals.QueryTotals[i].TotalTime += queryTime.Time
		}
		// Add Other time and tags
		totals.OtherTotal.TotalTime += daily.OtherTime
		totals.OtherTotal.Tags = append(totals.OtherTotal.Tags, daily.OtherTags...)
	}

	// Deduplicate Other tags
	totals.OtherTotal.Tags = s.deduplicateTags(totals.OtherTotal.Tags)

	// Calculate percentages
	for i := range totals.QueryTotals {
		if totals.TotalTime > 0 {
			totals.QueryTotals[i].Percentage = float64(totals.QueryTotals[i].TotalTime) / float64(totals.TotalTime) * 100
		}
	}

	// Calculate Other percentage
	if totals.TotalTime > 0 {
		totals.OtherTotal.Percentage = float64(totals.OtherTotal.TotalTime) / float64(totals.TotalTime) * 100
	}

	return totals
}

// taskMatchesSavedQuery checks if a task matches a saved query's criteria
func (s *ReportService) taskMatchesSavedQuery(task *models.Task, query *models.SavedQuery) bool {
	// Check included tags
	if len(query.IncludedTags) > 0 {
		hasIncludedTag := false
		for _, includedTag := range query.IncludedTags {
			for _, taskTag := range task.Tags {
				if taskTag == includedTag {
					hasIncludedTag = true
					break
				}
			}
			if hasIncludedTag {
				break
			}
		}
		if !hasIncludedTag {
			return false
		}
	}

	// Check excluded tags
	for _, excludedTag := range query.ExcludedTags {
		for _, taskTag := range task.Tags {
			if taskTag == excludedTag {
				return false
			}
		}
	}

	return true
}

// hasAnyTag checks if any tag from the task matches any excluded tag
func (s *ReportService) hasAnyTag(taskTags, excludedTags []string) bool {
	for _, taskTag := range taskTags {
		for _, excludedTag := range excludedTags {
			if taskTag == excludedTag {
				return true
			}
		}
	}
	return false
}

// deduplicateTags removes duplicate tags from a slice
func (s *ReportService) deduplicateTags(tags []string) []string {
	seen := make(map[string]bool)
	result := []string{}
	for _, tag := range tags {
		if !seen[tag] {
			seen[tag] = true
			result = append(result, tag)
		}
	}
	return result
}

// excludeQueryFilterTags removes tags that are used as filters in the saved query
func (s *ReportService) excludeQueryFilterTags(taskTags []string, query *models.SavedQuery) []string {
	var result []string

	// Build a set of tags to exclude (query's included and excluded tags)
	excludeSet := make(map[string]bool)
	for _, tag := range query.IncludedTags {
		excludeSet[tag] = true
	}
	for _, tag := range query.ExcludedTags {
		excludeSet[tag] = true
	}

	// Only include tags that are not in the exclude set
	for _, tag := range taskTags {
		if !excludeSet[tag] {
			result = append(result, tag)
		}
	}

	return result
}
