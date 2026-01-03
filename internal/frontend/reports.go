package frontend

import (
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/soarinferret/jats/internal/models"
	"github.com/soarinferret/jats/internal/services"
)

// ReportHandler handles report-related frontend requests
type ReportHandler struct {
	taskService *services.TaskService
	templates   map[string]*template.Template
}

// NewReportHandler creates a new report handler
func NewReportHandler(taskService *services.TaskService, templates map[string]*template.Template) *ReportHandler {
	return &ReportHandler{
		taskService: taskService,
		templates:   templates,
	}
}

// ReportData holds the data for the report page
type ReportData struct {
	SavedQueries      []*models.SavedQuery
	SelectedQuery     *models.SavedQuery
	OpenTasks         int
	CompletedTasks    int
	TotalTimeSpent    float64 // hours
	TimeSpentChart    template.HTML
	Last7Days         []string
}

// ReportPageHandler renders the main report page
func (h *ReportHandler) ReportPageHandler(c *gin.Context) {
	_, exists := c.Get("auth")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Get saved queries for the navigation
	savedQueries, err := h.taskService.GetSavedQueries()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get saved queries"})
		return
	}

	// Check if a specific query is selected
	var selectedQuery *models.SavedQuery
	queryIDStr := c.Query("query")
	if queryIDStr != "" {
		queryID, err := strconv.ParseUint(queryIDStr, 10, 32)
		if err == nil {
			selectedQuery, _ = h.taskService.GetSavedQueryByID(uint(queryID))
		}
	}

	// Generate report data
	reportData, err := h.generateReportData(selectedQuery)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate report data"})
		return
	}

	reportData.SavedQueries = savedQueries
	reportData.SelectedQuery = selectedQuery

	// Always render just the content area for main app integration
	h.renderReportContentForApp(c, reportData)
}

// renderReportHTML renders the report page as static HTML
func (h *ReportHandler) renderReportHTML(c *gin.Context, data *ReportData) {
	// Generate navigation items for saved queries (for reports context)
	navItems := ""
	for _, query := range data.SavedQueries {
		selected := ""
		if data.SelectedQuery != nil && data.SelectedQuery.ID == query.ID {
			selected = "bg-blue-50 border-blue-500 text-blue-700"
		}
		navItems += fmt.Sprintf(`
			<a href="/app/reports?query=%d" 
			   hx-get="/app/reports?query=%d"
			   hx-target="#report-content"
			   hx-push-url="true"
			   class="group border-l-4 %s py-2 px-3 flex items-center text-sm font-medium hover:text-gray-900 hover:bg-gray-50">
				<svg class="text-gray-400 mr-3 h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
				</svg>
				<span class="flex-1 truncate">%s</span>
			</a>`, query.ID, query.ID, 
			func() string { 
				if selected != "" { 
					return "border-blue-500 text-blue-700 bg-blue-50" 
				}
				return "border-transparent text-gray-600" 
			}(), query.Name)
	}

	if len(data.SavedQueries) == 0 {
		navItems = `<p class="text-sm text-gray-500 px-3 py-2">No saved queries</p>`
	}

	html := fmt.Sprintf(`
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Reports - JATS</title>
    <script src="https://unpkg.com/htmx.org@1.9.3"></script>
    <script src="https://cdn.tailwindcss.com"></script>
    <style>
        .custom-scrollbar {
            scrollbar-width: thin;
            scrollbar-color: #94a3b8 #f1f5f9;
        }
        .custom-scrollbar::-webkit-scrollbar {
            width: 6px;
        }
        .custom-scrollbar::-webkit-scrollbar-track {
            background: #f1f5f9;
            border-radius: 3px;
        }
        .custom-scrollbar::-webkit-scrollbar-thumb {
            background: #94a3b8;
            border-radius: 3px;
        }
        .custom-scrollbar::-webkit-scrollbar-thumb:hover {
            background: #64748b;
        }
    </style>
</head>
<body class="bg-gray-50">
    <div class="flex h-screen">
        <!-- Sidebar -->
        <div class="hidden md:flex md:w-64 md:flex-col md:fixed md:inset-y-0">
            <div class="flex-1 flex flex-col min-h-0 bg-white border-r border-gray-200">
                <!-- Header -->
                <div class="p-6 border-b border-gray-200">
                    <div class="flex items-center justify-between">
                        <div>
                            <h1 class="text-2xl font-bold text-gray-900">JATS</h1>
                            <p class="text-sm text-gray-600 mt-1">Reports Dashboard</p>
                        </div>
                    </div>
                </div>
                
                <!-- Primary Navigation -->
                <nav class="p-4 space-y-2">
                    <a href="/" 
                       class="flex items-center px-4 py-2 text-sm font-medium rounded-md text-gray-700 hover:bg-gray-100 hover:text-gray-900">
                        <svg class="h-5 w-5 mr-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-3 7h3m-3 4h3m-6-4h.01M9 16h.01" />
                        </svg>
                        Tasks
                    </a>
                    
                    <a href="/app/reports"
                       class="flex items-center px-4 py-2 text-sm font-medium rounded-md bg-blue-100 text-blue-700">
                        <svg class="h-5 w-5 mr-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
                        </svg>
                        Reports
                    </a>
                </nav>
                
                <!-- Reports Filter Navigation -->
                <div class="flex-1 flex flex-col overflow-y-auto">
                    <div class="px-4 py-2 border-t border-gray-200">
                        <h3 class="text-xs font-semibold text-gray-500 uppercase tracking-wider">
                            Report Filters
                        </h3>
                    </div>
                    
                    <!-- Filter Navigation -->
                    <nav class="mt-4 flex-1 px-2 space-y-1">
                        <!-- All Tasks -->
                        <a href="/app/reports" 
                           hx-get="/app/reports"
                           hx-target="#report-content"
                           hx-push-url="true"
                           class="group border-l-4 %s py-2 px-3 flex items-center text-sm font-medium hover:text-gray-900 hover:bg-gray-50">
                            <svg class="text-gray-400 mr-3 h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
                            </svg>
                            All Tasks
                        </a>
                        
                        <!-- Divider -->
                        <div class="border-t border-gray-200 my-2"></div>
                        
                        <!-- Saved Queries -->
                        <div class="px-3 py-2">
                            <p class="text-xs font-semibold text-gray-500 uppercase tracking-wider">
                                Saved Queries
                            </p>
                        </div>
                        %s
                    </nav>
                </div>
                
                <!-- Footer -->
                <div class="flex-shrink-0 p-4 border-t border-gray-200">
                    <div class="flex items-center">
                        <div class="flex-1 min-w-0">
                            <p class="text-sm font-medium text-gray-900 truncate">User</p>
                            <p class="text-xs text-gray-500 truncate">user@example.com</p>
                        </div>
                        <form method="POST" action="/logout" class="ml-3">
                            <button type="submit" class="text-gray-400 hover:text-gray-600 p-1 rounded" title="Logout">
                                <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1" />
                                </svg>
                            </button>
                        </form>
                    </div>
                </div>
            </div>
        </div>

        <!-- Main content -->
        <div class="md:pl-64 flex flex-col flex-1">
            <div class="sticky top-0 z-10 md:hidden pl-1 pt-1 sm:pl-3 sm:pt-3 bg-gray-200">
                <button type="button" class="-ml-0.5 -mt-0.5 h-12 w-12 inline-flex items-center justify-center rounded-md text-gray-500 hover:text-gray-900 focus:outline-none focus:ring-2 focus:ring-inset focus:ring-indigo-500">
                    <span class="sr-only">Open sidebar</span>
                    <svg class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16M4 18h16" />
                    </svg>
                </button>
            </div>
            
            <main class="flex-1" id="report-content">
                %s
            </main>
        </div>
    </div>
</body>
</html>`, 
		func() string { 
			if data.SelectedQuery == nil { 
				return "border-blue-500 text-blue-700 bg-blue-50" 
			}
			return "border-transparent text-gray-600" 
		}(),
		navItems,
		h.renderReportContentHTML(data))

	c.Header("Content-Type", "text/html")
	c.String(http.StatusOK, html)
}

// renderReportContent renders just the report content area for HTMX requests
func (h *ReportHandler) renderReportContent(c *gin.Context, data *ReportData) {
	content := h.renderReportContentHTML(data)
	c.Header("Content-Type", "text/html")
	c.String(http.StatusOK, content)
}

// renderReportContentForApp renders the report content for integration within main app
func (h *ReportHandler) renderReportContentForApp(c *gin.Context, data *ReportData) {
	queryName := "All Tasks"
	if data.SelectedQuery != nil {
		queryName = data.SelectedQuery.Name
	}

	content := fmt.Sprintf(`
<div class="p-6">
    <div class="mb-6">
        <h1 class="text-2xl font-semibold text-gray-900">%s Report</h1>
        <p class="mt-2 text-sm text-gray-600">Activity and time tracking for the last 7 days</p>
    </div>
    
    <!-- Metrics Grid -->
    <div class="grid grid-cols-1 gap-5 sm:grid-cols-3 mb-8">
        <!-- Open Tasks -->
        <div class="bg-white overflow-hidden shadow rounded-lg">
            <div class="p-5">
                <div class="flex items-center">
                    <div class="flex-shrink-0">
                        <svg class="h-6 w-6 text-yellow-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
                        </svg>
                    </div>
                    <div class="ml-5 w-0 flex-1">
                        <dl>
                            <dt class="text-sm font-medium text-gray-500 truncate">Open Tasks</dt>
                            <dd class="text-lg font-medium text-gray-900">%d</dd>
                        </dl>
                    </div>
                </div>
            </div>
        </div>

        <!-- Completed Tasks -->
        <div class="bg-white overflow-hidden shadow rounded-lg">
            <div class="p-5">
                <div class="flex items-center">
                    <div class="flex-shrink-0">
                        <svg class="h-6 w-6 text-green-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
                        </svg>
                    </div>
                    <div class="ml-5 w-0 flex-1">
                        <dl>
                            <dt class="text-sm font-medium text-gray-500 truncate">Completed (7d)</dt>
                            <dd class="text-lg font-medium text-gray-900">%d</dd>
                        </dl>
                    </div>
                </div>
            </div>
        </div>

        <!-- Total Time Spent -->
        <div class="bg-white overflow-hidden shadow rounded-lg">
            <div class="p-5">
                <div class="flex items-center">
                    <div class="flex-shrink-0">
                        <svg class="h-6 w-6 text-blue-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z" />
                        </svg>
                    </div>
                    <div class="ml-5 w-0 flex-1">
                        <dl>
                            <dt class="text-sm font-medium text-gray-500 truncate">Time Spent (7d)</dt>
                            <dd class="text-lg font-medium text-gray-900">%.1f hrs</dd>
                        </dl>
                    </div>
                </div>
            </div>
        </div>
    </div>

    <!-- Chart Section -->
    <div class="bg-white shadow rounded-lg">
        <div class="px-4 py-5 sm:p-6">
            <h3 class="text-lg leading-6 font-medium text-gray-900 mb-4">Daily Time Tracking</h3>
            <div class="mt-2">
                %s
            </div>
        </div>
    </div>
</div>`, queryName, data.OpenTasks, data.CompletedTasks, data.TotalTimeSpent, data.TimeSpentChart)

	c.Header("Content-Type", "text/html")
	c.String(http.StatusOK, content)
}

// renderReportContentHTML generates the HTML for the report content area
func (h *ReportHandler) renderReportContentHTML(data *ReportData) string {
	queryName := "All Tasks"
	if data.SelectedQuery != nil {
		queryName = data.SelectedQuery.Name
	}

	return fmt.Sprintf(`
<div class="py-6">
    <div class="max-w-7xl mx-auto px-4 sm:px-6 md:px-8">
        <h1 class="text-2xl font-semibold text-gray-900">%s Report</h1>
        <p class="mt-2 text-sm text-gray-600">Activity and time tracking for the last 7 days</p>
    </div>
    
    <div class="max-w-7xl mx-auto px-4 sm:px-6 md:px-8">
        <!-- Metrics Grid -->
        <div class="mt-8">
            <div class="grid grid-cols-1 gap-5 sm:grid-cols-3">
                <!-- Open Tasks -->
                <div class="bg-white overflow-hidden shadow rounded-lg">
                    <div class="p-5">
                        <div class="flex items-center">
                            <div class="flex-shrink-0">
                                <svg class="h-6 w-6 text-yellow-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
                                </svg>
                            </div>
                            <div class="ml-5 w-0 flex-1">
                                <dl>
                                    <dt class="text-sm font-medium text-gray-500 truncate">Open Tasks</dt>
                                    <dd class="text-lg font-medium text-gray-900">%d</dd>
                                </dl>
                            </div>
                        </div>
                    </div>
                </div>

                <!-- Completed Tasks -->
                <div class="bg-white overflow-hidden shadow rounded-lg">
                    <div class="p-5">
                        <div class="flex items-center">
                            <div class="flex-shrink-0">
                                <svg class="h-6 w-6 text-green-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
                                </svg>
                            </div>
                            <div class="ml-5 w-0 flex-1">
                                <dl>
                                    <dt class="text-sm font-medium text-gray-500 truncate">Completed (7d)</dt>
                                    <dd class="text-lg font-medium text-gray-900">%d</dd>
                                </dl>
                            </div>
                        </div>
                    </div>
                </div>

                <!-- Total Time Spent -->
                <div class="bg-white overflow-hidden shadow rounded-lg">
                    <div class="p-5">
                        <div class="flex items-center">
                            <div class="flex-shrink-0">
                                <svg class="h-6 w-6 text-blue-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z" />
                                </svg>
                            </div>
                            <div class="ml-5 w-0 flex-1">
                                <dl>
                                    <dt class="text-sm font-medium text-gray-500 truncate">Time Spent (7d)</dt>
                                    <dd class="text-lg font-medium text-gray-900">%.1f hrs</dd>
                                </dl>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </div>

        <!-- Chart Section -->
        <div class="mt-8">
            <div class="bg-white shadow rounded-lg">
                <div class="px-4 py-5 sm:p-6">
                    <h3 class="text-lg leading-6 font-medium text-gray-900 mb-4">Daily Time Tracking</h3>
                    <div class="mt-2">
                        %s
                    </div>
                </div>
            </div>
        </div>
    </div>
</div>`, queryName, data.OpenTasks, data.CompletedTasks, data.TotalTimeSpent, data.TimeSpentChart)
}

// generateReportData calculates report metrics and generates charts
func (h *ReportHandler) generateReportData(savedQuery *models.SavedQuery) (*ReportData, error) {
	// Get all tasks or filtered tasks
	var tasks []*models.Task
	var err error

	if savedQuery != nil {
		tasks, err = h.taskService.GetTasksBySavedQuery(savedQuery)
	} else {
		tasks, err = h.taskService.GetTasks()
	}

	if err != nil {
		return nil, err
	}

	// Calculate date range for last 7 days
	now := time.Now()
	sevenDaysAgo := now.AddDate(0, 0, -7)

	// Calculate metrics
	openTasks := 0
	completedTasks := 0
	totalTimeSpent := 0.0

	// Group time entries by day
	dailyTime := make(map[string]float64)
	last7Days := make([]string, 7)
	
	// Initialize last 7 days
	for i := 0; i < 7; i++ {
		date := now.AddDate(0, 0, -6+i)
		dateStr := date.Format("2006-01-02")
		last7Days[i] = dateStr
		dailyTime[dateStr] = 0.0
	}

	for _, task := range tasks {
		// Count open tasks
		if task.Status == models.TaskStatusOpen || task.Status == models.TaskStatusInProgress {
			openTasks++
		}

		// Count completed tasks in last 7 days
		if (task.Status == models.TaskStatusResolved || task.Status == models.TaskStatusClosed) && 
		   task.ResolvedAt != nil && task.ResolvedAt.After(sevenDaysAgo) {
			completedTasks++
		}

		// Sum time entries for last 7 days
		for _, timeEntry := range task.TimeEntries {
			if timeEntry.CreatedAt.After(sevenDaysAgo) {
				hours := float64(timeEntry.Duration) / 60.0
				totalTimeSpent += hours
				
				// Add to daily breakdown
				dateStr := timeEntry.CreatedAt.Format("2006-01-02")
				dailyTime[dateStr] += hours
			}
		}
	}

	// Generate chart
	chartHTML := h.generateTimeSpentChart(last7Days, dailyTime)

	return &ReportData{
		OpenTasks:      openTasks,
		CompletedTasks: completedTasks,
		TotalTimeSpent: totalTimeSpent,
		TimeSpentChart: template.HTML(chartHTML),
		Last7Days:      last7Days,
	}, nil
}

// generateTimeSpentChart creates a bar chart for daily time spent
func (h *ReportHandler) generateTimeSpentChart(days []string, dailyTime map[string]float64) string {
	// Create bar chart
	bar := charts.NewBar()
	
	// Set global options
	bar.SetGlobalOptions(charts.WithTitleOpts(opts.Title{
		Title: "Daily Time Spent",
		Subtitle: "Hours tracked over the last 7 days",
	}))

	// Prepare data
	xAxis := make([]string, len(days))
	yData := make([]opts.BarData, len(days))
	
	for i, day := range days {
		// Format day for display (Mon 01/02)
		if date, err := time.Parse("2006-01-02", day); err == nil {
			xAxis[i] = date.Format("Mon 01/02")
		} else {
			xAxis[i] = day
		}
		
		hours := dailyTime[day]
		yData[i] = opts.BarData{Value: hours}
	}

	// Add data to chart
	bar.SetXAxis(xAxis).
		AddSeries("Hours", yData).
		SetSeriesOptions(charts.WithBarChartOpts(opts.BarChart{
			XAxisIndex: 0,
			YAxisIndex: 0,
		}))

	// Generate HTML
	bar.Validate()
	
	// Since we can't easily get the HTML output, we'll create a simple HTML chart
	// This is a fallback implementation
	maxHours := 0.0
	for _, hours := range dailyTime {
		if hours > maxHours {
			maxHours = hours
		}
	}
	
	chartBars := ""
	for _, day := range days {
		// Format day for display
		displayDay := day
		if date, err := time.Parse("2006-01-02", day); err == nil {
			displayDay = date.Format("Mon 01/02")
		}
		
		hours := dailyTime[day]
		heightPercent := 0.0
		if maxHours > 0 {
			heightPercent = (hours / maxHours) * 100
		}
		
		chartBars += fmt.Sprintf(`
			<div class="flex flex-col items-center">
				<div class="h-32 w-12 bg-gray-100 rounded flex items-end justify-center relative">
					<div class="bg-blue-500 w-10 rounded-sm" style="height: %.1f%%" title="%.1f hours"></div>
				</div>
				<div class="mt-2 text-xs text-gray-600">%s</div>
				<div class="text-xs font-medium text-gray-900">%.1fh</div>
			</div>`, heightPercent, hours, displayDay, hours)
	}
	
	return fmt.Sprintf(`
		<div class="flex justify-between items-end space-x-2 px-4">
			%s
		</div>`, chartBars)
}