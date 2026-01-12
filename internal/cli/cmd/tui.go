package cmd

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/spf13/cobra"
	"github.com/soarinferret/jats/internal/cli/client"
)

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Start interactive TUI interface",
	Long:  `Start an interactive terminal user interface for JATS task management.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		tui := NewTUI()
		return tui.Run()
	},
}

// TUI represents the terminal user interface
type TUI struct {
	app         *tview.Application
	client      *client.Client
	root        tview.Primitive
	header      *tview.TextView
	sidebar     *tview.List
	tasksTable  *tview.Table
	statusBar   *tview.TextView
	timeDialog  *tview.Modal
	tasks       []client.Task
	savedQueries []client.SavedQuery
	selectedQuery string
	globalInputHandler func(event *tcell.EventKey) *tcell.EventKey
	
	// Pagination fields
	currentPage int
	pageSize    int
	totalTasks  int
	
	// Search fields
	searchQuery string
	searchActive bool
	
	// Layout fields
	showSidebar bool
	mainFlex    *tview.Flex
}

// NewTUI creates a new TUI instance
func NewTUI() *TUI {
	return &TUI{
		app:         tview.NewApplication(),
		client:      client.New(),
		currentPage: 0,
		pageSize:    20,
		searchQuery: "",
		searchActive: false,
		showSidebar: true,  // Default to true, will be updated in Run()
	}
}

// Run starts the TUI application
func (t *TUI) Run() error {
	// Check terminal size to determine default sidebar visibility
	screen, err := tcell.NewScreen()
	if err == nil {
		if initErr := screen.Init(); initErr == nil {
			width, _ := screen.Size()
			t.showSidebar = width >= 120  // Hide sidebar on terminals smaller than 120 columns
			screen.Fini()
		}
	}

	// Initialize components
	t.setupHeader()
	t.setupSidebar()
	t.setupTasksTable()
	t.setupStatusBar()
	t.setupLayout()
	t.setupKeyBindings()

	// Load initial data
	if err := t.refreshData(); err != nil {
		return fmt.Errorf("failed to load initial data: %w", err)
	}

	// Set initial focus and status
	t.app.SetFocus(t.tasksTable)
	t.updateStatusForPane("tasks")

	// Start the application
	return t.app.Run()
}

// setupHeader creates the header showing task counts
func (t *TUI) setupHeader() {
	t.header = tview.NewTextView().
		SetText("Loading...").
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true)
	t.header.SetBorder(true).SetTitle("JATS - Task Summary")
}

// setupSidebar creates the left sidebar with saved queries
func (t *TUI) setupSidebar() {
	t.sidebar = tview.NewList()
	t.sidebar.SetBorder(true).SetTitle("Saved Queries")

	// Set default selection (sidebar will be populated by loadSavedQueries)
	t.selectedQuery = "active"
}

// setupTasksTable creates the main tasks table
func (t *TUI) setupTasksTable() {
	t.tasksTable = tview.NewTable()
	t.tasksTable.SetBorder(true).SetTitle("Tasks")
	t.tasksTable.SetSelectable(true, false)
	
	// Set headers
	headers := []string{"✓", "Name", "Tags", "Time", "Priority", "Status"}
	for i, header := range headers {
		cell := tview.NewTableCell(header).
			SetTextColor(tcell.ColorYellow).
			SetAlign(tview.AlignCenter).
			SetSelectable(false)
		t.tasksTable.SetCell(0, i, cell)
	}
}

// setupStatusBar creates the bottom status bar
func (t *TUI) setupStatusBar() {
	t.statusBar = tview.NewTextView().
		SetText("[yellow]r[white]: Resolve | [yellow]c[white]: Comment | [yellow]t[white]: Add Time | [yellow]Enter[white]: Details | [yellow]q[white]: Quit").
		SetDynamicColors(true)
	t.statusBar.SetBorder(false)
}

// setupLayout creates the main layout
func (t *TUI) setupLayout() {
	// Main content area
	t.mainFlex = tview.NewFlex().SetDirection(tview.FlexColumn)
	t.updateMainLayout()

	// Overall layout
	layout := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(t.header, 3, 0, false).
		AddItem(t.mainFlex, 0, 1, true).
		AddItem(t.statusBar, 1, 0, false)

	t.root = layout
	t.app.SetRoot(layout, true)
}

// updateMainLayout updates the main flex layout based on sidebar visibility
func (t *TUI) updateMainLayout() {
	t.mainFlex.Clear()
	
	if t.showSidebar {
		t.mainFlex.AddItem(t.sidebar, 0, 1, false)
		t.mainFlex.AddItem(t.tasksTable, 0, 3, true)
	} else {
		t.mainFlex.AddItem(t.tasksTable, 0, 1, true)
	}
}

// toggleSidebar toggles the visibility of the saved queries sidebar
func (t *TUI) toggleSidebar() {
	t.showSidebar = !t.showSidebar
	t.updateMainLayout()
	
	// Update focus appropriately
	if t.showSidebar {
		t.setStatus("Sidebar shown. Press Q to toggle, Tab to switch focus")
	} else {
		t.setStatus("Sidebar hidden. Press Q to show")
		// If sidebar was focused, move focus to tasks table
		if t.app.GetFocus() == t.sidebar {
			t.app.SetFocus(t.tasksTable)
		}
	}
}

// setupKeyBindings sets up context-aware key bindings
func (t *TUI) setupKeyBindings() {
	t.globalInputHandler = func(event *tcell.EventKey) *tcell.EventKey {
		focused := t.app.GetFocus()
		
		// Global shortcuts that work anywhere
		switch event.Rune() {
		case 'q':
			t.app.Stop()
			return nil
		case 'Q':
			t.toggleSidebar()
			return nil
		case 'A':
			t.showCreateTaskModal()
			return nil
		}
		
		switch event.Key() {
		case tcell.KeyF5:
			t.refreshData()
			return nil
		case tcell.KeyTab:
			// Switch focus between sidebar and table (only if sidebar is visible)
			if t.showSidebar {
				if focused == t.sidebar {
					t.app.SetFocus(t.tasksTable)
					t.updateStatusForPane("tasks")
				} else {
					t.app.SetFocus(t.sidebar)
					t.updateStatusForPane("queries")
				}
			}
			return nil
		}
		
		// Context-specific shortcuts
		if focused == t.tasksTable {
			return t.handleTaskPaneKeys(event)
		} else if focused == t.sidebar {
			return t.handleQueryPaneKeys(event)
		}
		
		return event
	}
	t.app.SetInputCapture(t.globalInputHandler)
}

// handleTaskPaneKeys handles keyboard shortcuts when tasks table has focus
func (t *TUI) handleTaskPaneKeys(event *tcell.EventKey) *tcell.EventKey {
	switch event.Rune() {
	case 'r':
		t.toggleTaskStatus()
		return nil
	case 'c':
		t.showCommentDialog()
		return nil
	case 't':
		t.showTimeDialog()
		return nil
	case 'e':
		t.showEditDialog()
		return nil
	case '/':
		t.showSearchDialog()
		return nil
	case 'n':
		t.nextPage()
		return nil
	case 'p':
		t.prevPage()
		return nil
	case 'x':
		t.clearSearch()
		return nil
	}
	
	switch event.Key() {
	case tcell.KeyEnter:
		t.viewTaskDetails()
		return nil
	case tcell.KeyCtrlN:
		t.nextPage()
		return nil
	case tcell.KeyCtrlP:
		t.prevPage()
		return nil
	}
	
	return event
}

// handleQueryPaneKeys handles keyboard shortcuts when queries sidebar has focus
func (t *TUI) handleQueryPaneKeys(event *tcell.EventKey) *tcell.EventKey {
	switch event.Rune() {
	case 'n':
		t.showNewQueryDialog()
		return nil
	}
	
	switch event.Key() {
	case tcell.KeyEnter:
		t.selectCurrentQuery()
		return nil
	}
	
	return event
}

// selectCurrentQuery selects the currently highlighted query
func (t *TUI) selectCurrentQuery() {
	currentItem := t.sidebar.GetCurrentItem()
	if currentItem < 0 {
		return
	}

	// Trigger the selected function for the current item
	t.sidebar.SetCurrentItem(currentItem)

	// The list item's selected function will be called automatically
	// but we need to manually trigger it since Enter doesn't do that by default
	_, selected := t.sidebar.GetItemText(currentItem)

	// Map the current item to our query selection logic
	switch currentItem {
	case 0: // All Active Tasks
		t.selectedQuery = "active"
	case 1: // Resolved
		t.selectedQuery = "resolved"
	default:
		// Saved query (currentItem - 2 since we have 2 default queries)
		savedIndex := currentItem - 2
		if savedIndex >= 0 && savedIndex < len(t.savedQueries) {
			t.selectedQuery = fmt.Sprintf("saved:%d", t.savedQueries[savedIndex].ID)
		}
	}

	// Reset pagination when changing query
	t.currentPage = 0

	t.refreshTasksOnly()
	if err := t.updateHeader(); err != nil {
		t.setStatus(fmt.Sprintf("Error updating header: %v", err))
	} else {
		t.setStatus(fmt.Sprintf("Selected query: %s", selected))
	}

	// Only switch focus to tasks table if there are tasks to display
	// Otherwise keep focus on sidebar to avoid freeze
	if len(t.tasks) > 0 {
		t.app.SetFocus(t.tasksTable)
		t.updateStatusForPane("tasks")
	} else {
		// Keep focus on sidebar when no tasks found
		t.setStatus(fmt.Sprintf("No tasks found for query: %s", selected))
		t.updateStatusForPane("queries")
	}
}

// updateStatusForPane updates the status bar based on which pane has focus
func (t *TUI) updateStatusForPane(pane string) {
	tabText := ""
	if t.showSidebar {
		tabText = " | [yellow]Tab[white]: Switch Panes"
	}
	
	if pane == "tasks" {
		t.statusBar.SetText("[yellow]A[white]: Add Task | [yellow]r[white]: Resolve/Reopen | [yellow]e[white]: Edit | [yellow]c[white]: Comment | [yellow]t[white]: Add Time | [yellow]/[white]: Search | [yellow]n/p[white]: Next/Prev Page | [yellow]x[white]: Clear Search | [yellow]Enter[white]: Details" + tabText + " | [yellow]Q[white]: Toggle Sidebar | [yellow]q[white]: Quit")
	} else if pane == "queries" {
		t.statusBar.SetText("[yellow]A[white]: Add Task | [yellow]n[white]: New Query | [yellow]Enter[white]: Select Query" + tabText + " | [yellow]Q[white]: Toggle Sidebar | [yellow]q[white]: Quit")
	}
}

// disableGlobalKeys disables global key bindings (for forms)
func (t *TUI) disableGlobalKeys() {
	t.app.SetInputCapture(nil)
}

// enableGlobalKeys re-enables global key bindings
func (t *TUI) enableGlobalKeys() {
	t.app.SetInputCapture(t.globalInputHandler)
}

// refreshData refreshes all data
func (t *TUI) refreshData() error {
	// Load saved queries and update sidebar
	if err := t.loadSavedQueries(); err != nil {
		t.setStatus(fmt.Sprintf("Error loading saved queries: %v", err))
	}
	
	// Update header with task counts
	if err := t.updateHeader(); err != nil {
		t.setStatus(fmt.Sprintf("Error updating header: %v", err))
	}
	
	// Refresh tasks
	return t.refreshTasks()
}

// loadSavedQueries loads saved queries from the API and adds them to the sidebar
func (t *TUI) loadSavedQueries() error {
	savedQueries, err := t.client.GetSavedQueries()
	if err != nil {
		return err
	}
	
	t.savedQueries = savedQueries
	
	// Clear existing sidebar and rebuild it
	t.sidebar.Clear()

	// Re-add default queries first
	t.sidebar.AddItem("All Active Tasks", "Show open and in-progress tasks", 'a', func() {
		t.selectedQuery = "active"
		t.refreshTasksOnly()
	})

	t.sidebar.AddItem("Resolved", "Show resolved tasks", 'r', func() {
		t.selectedQuery = "resolved"
		t.refreshTasksOnly()
	})
	
	// Add saved queries to sidebar after default ones
	for i, sq := range savedQueries {
		query := sq // capture for closure

		// Assign shortcuts 1-9 for the first 9 saved queries
		var shortcut rune
		if i < 9 {
			shortcut = rune('1' + i)
		} else {
			shortcut = 0 // No shortcut for 10th and beyond
		}

		t.sidebar.AddItem(query.Name, fmt.Sprintf("Tags: %s", strings.Join(query.IncludedTags, ", ")), shortcut, func() {
			t.selectedQuery = fmt.Sprintf("saved:%d", query.ID)
			t.refreshTasksOnly()
		})
	}
	
	// Restore selection to the currently active query
	t.restoreSidebarSelection()
	
	return nil
}

// restoreSidebarSelection restores the sidebar selection based on current selectedQuery
func (t *TUI) restoreSidebarSelection() {
	switch t.selectedQuery {
	case "active":
		t.sidebar.SetCurrentItem(0)
	case "resolved":
		t.sidebar.SetCurrentItem(1)
	default:
		// Check if it's a saved query
		if strings.HasPrefix(t.selectedQuery, "saved:") {
			idStr := strings.TrimPrefix(t.selectedQuery, "saved:")
			if id, err := strconv.ParseUint(idStr, 10, 32); err == nil {
				// Find the saved query index
				for i, sq := range t.savedQueries {
					if sq.ID == uint(id) {
						t.sidebar.SetCurrentItem(2 + i) // 2 default queries + saved query index
						return
					}
				}
			}
		}
		// Default to first item if not found
		t.sidebar.SetCurrentItem(0)
	}
}

// updateHeader updates the header with current task counts using the summary API
func (t *TUI) updateHeader() error {
	// Determine saved query ID for filtering
	var savedQueryID *uint = nil
	
	// Check if current selection is a saved query
	if strings.HasPrefix(t.selectedQuery, "saved:") {
		idStr := strings.TrimPrefix(t.selectedQuery, "saved:")
		if id, err := strconv.ParseUint(idStr, 10, 32); err == nil {
			queryID := uint(id)
			savedQueryID = &queryID
		}
	}
	
	// Get summary from API endpoint
	summary, err := t.client.GetTaskSummary(savedQueryID)
	if err != nil {
		return err
	}
	
	// Build header text with current filter context
	filterText := ""
	if savedQueryID != nil {
		// Find saved query name for display
		for _, sq := range t.savedQueries {
			if sq.ID == *savedQueryID {
				filterText = fmt.Sprintf(" (Filtered: %s)", sq.Name)
				break
			}
		}
	}
	
	headerText := fmt.Sprintf(
		"[green]Open: %d[white] | [yellow]In Progress: %d[white] | [cyan]Added (7d): %d[white] | [blue]Resolved (7d): %d[white]%s",
		summary.OpenTasks,
		summary.InProgressTasks,
		summary.RecentlyAddedTasks,
		summary.RecentlyResolvedTasks,
		filterText,
	)
	
	t.header.SetText(headerText)
	return nil
}

// refreshTasks refreshes the tasks table based on selected query
func (t *TUI) refreshTasks() error {
	return t.refreshTasksOnly()
}

// refreshTasksOnly refreshes only the tasks without reloading saved queries (prevents infinite loops)
func (t *TUI) refreshTasksOnly() error {
	filters := &client.TaskFilters{
		Limit:  t.pageSize,
		Offset: t.currentPage * t.pageSize,
	}
	
	// Add search filter if active
	if t.searchActive && t.searchQuery != "" {
		filters.Search = t.searchQuery
	}
	
	// Set filters based on selected query
	if strings.HasPrefix(t.selectedQuery, "saved:") {
		// Handle saved query
		idStr := strings.TrimPrefix(t.selectedQuery, "saved:")
		if id, err := strconv.ParseUint(idStr, 10, 32); err == nil {
			// Find the saved query
			for _, sq := range t.savedQueries {
				if sq.ID == uint(id) {
					if len(sq.IncludedTags) > 0 {
						filters.Tags = sq.IncludedTags
					}
					// Note: ExcludedTags not currently supported in TaskFilters
					break
				}
			}
		}
		// Saved queries default to showing only active tasks (open + in-progress)
		filters.Status = []string{"open", "in-progress"}
	} else {
		// Handle default queries
		switch t.selectedQuery {
		case "active":
			filters.Status = []string{"open", "in-progress"}
		case "resolved":
			filters.Status = []string{"resolved"}
		default:
			filters.Status = []string{"open", "in-progress"}
		}
	}
	
	tasks, err := t.client.GetTasks(filters)
	if err != nil {
		t.setStatus(fmt.Sprintf("Error loading tasks: %v", err))
		return err
	}
	
	t.tasks = tasks
	t.populateTasksTable()
	
	// Update status with pagination info
	searchInfo := ""
	if t.searchActive && t.searchQuery != "" {
		searchInfo = fmt.Sprintf(" (search: %s)", t.searchQuery)
	}
	
	pageInfo := fmt.Sprintf("Page %d%s", t.currentPage+1, searchInfo)
	t.setStatus(fmt.Sprintf("Loaded %d tasks - %s", len(tasks), pageInfo))
	
	return nil
}

// populateTasksTable populates the tasks table with current tasks
func (t *TUI) populateTasksTable() {
	// Clear existing rows (keep header)
	for row := t.tasksTable.GetRowCount() - 1; row > 0; row-- {
		t.tasksTable.RemoveRow(row)
	}

	// If no tasks, clear selection
	if len(t.tasks) == 0 {
		t.tasksTable.SetSelectable(false, false)
		return
	}

	// Enable selection when tasks are present
	t.tasksTable.SetSelectable(true, false)

	for i, task := range t.tasks {
		row := i + 1

		// Complete status
		completeSymbol := " "
		if task.Status == "resolved" || task.Status == "closed" {
			completeSymbol = "✓"
		}

		// Calculate total time
		totalTime := 0
		for _, entry := range task.TimeEntries {
			totalTime += entry.Duration
		}
		timeStr := tuiFormatDuration(totalTime)

		// Tags
		tagsStr := strings.Join(task.Tags, ", ")
		if tagsStr == "" {
			tagsStr = "-"
		}

		// Priority color
		priorityColor := ""
		switch task.Priority {
		case "high":
			priorityColor = "[red]"
		case "medium":
			priorityColor = "[yellow]"
		case "low":
			priorityColor = "[green]"
		default:
			priorityColor = "[white]"
		}
		
		// Status color
		statusColor := ""
		switch task.Status {
		case "open":
			statusColor = "[white]"
		case "in-progress":
			statusColor = "[yellow]"
		case "resolved":
			statusColor = "[green]"
		case "closed":
			statusColor = "[gray]"
		}
		
		cells := []struct {
			text  string
			align int
		}{
			{completeSymbol, tview.AlignCenter},
			{task.Name, tview.AlignLeft},
			{tagsStr, tview.AlignLeft},
			{timeStr, tview.AlignRight},
			{priorityColor + string(task.Priority), tview.AlignCenter},
			{statusColor + string(task.Status), tview.AlignCenter},
		}
		
		for col, cell := range cells {
			tableCell := tview.NewTableCell(cell.text).
				SetAlign(cell.align).
				SetReference(task)
			t.tasksTable.SetCell(row, col, tableCell)
		}
	}
	
	// Select first task if available
	if len(t.tasks) > 0 {
		t.tasksTable.Select(1, 0)
	}
}

// getSelectedTask returns the currently selected task
func (t *TUI) getSelectedTask() *client.Task {
	row, _ := t.tasksTable.GetSelection()
	if row == 0 || row > len(t.tasks) {
		return nil
	}
	return &t.tasks[row-1]
}

// toggleTaskStatus toggles between open/resolved states
func (t *TUI) toggleTaskStatus() {
	task := t.getSelectedTask()
	if task == nil {
		t.setStatus("No task selected")
		return
	}
	
	var newStatus string
	var actionMsg string
	
	if task.Status == "resolved" || task.Status == "closed" {
		// Reopen the task
		newStatus = "open"
		actionMsg = "Reopened"
	} else {
		// Resolve the task
		newStatus = "resolved"
		actionMsg = "Resolved"
	}
	
	_, err := t.client.UpdateTaskStatus(task.ID, newStatus)
	if err != nil {
		t.setStatus(fmt.Sprintf("Error updating task status: %v", err))
		return
	}
	
	t.setStatus(fmt.Sprintf("%s task: %s", actionMsg, task.Name))
	t.refreshData()
}

// showTimeDialog shows the time entry dialog
func (t *TUI) showTimeDialog() {
	task := t.getSelectedTask()
	if task == nil {
		t.setStatus("No task selected")
		return
	}
	
	form := tview.NewForm()
	form.SetBorder(true).SetTitle(fmt.Sprintf("Add Time - %s", task.Name))
	
	var duration, description, date string
	
	form.AddInputField("Duration", "", 20, nil, func(text string) {
		duration = text
	})
	form.AddInputField("Description", "Time logged via TUI", 50, nil, func(text string) {
		description = text
	})
	form.AddInputField("Date (optional)", "", 30, nil, func(text string) {
		date = text
	})
	
	originalRoot := t.root
	
	form.AddButton("Add Time", func() {
		if duration == "" {
			t.setStatus("Duration is required")
			return
		}
		
		durationMinutes, err := client.ParseDuration(duration)
		if err != nil {
			t.setStatus(fmt.Sprintf("Invalid duration format: %v", err))
			return
		}
		
		if description == "" {
			description = "Time logged via TUI"
		}
		
		timeReq := &client.LogTimeRequest{
			Duration:    durationMinutes,
			Description: description,
			Date:        date,
		}
		
		err = t.client.LogTime(task.ID, timeReq)
		if err != nil {
			t.setStatus(fmt.Sprintf("Error logging time: %v", err))
		} else {
			t.setStatus(fmt.Sprintf("Logged %s on task: %s", duration, task.Name))
			t.refreshData()
		}
		
		t.enableGlobalKeys()
		t.app.SetRoot(originalRoot, true)
	})
	
	form.AddButton("Cancel", func() {
		t.enableGlobalKeys()
		t.app.SetRoot(originalRoot, true)
	})
	
	// Disable global keys while form is active
	t.disableGlobalKeys()
	t.app.SetRoot(form, true)
}

// showCommentDialog shows the comment entry dialog
func (t *TUI) showCommentDialog() {
	task := t.getSelectedTask()
	if task == nil {
		t.setStatus("No task selected")
		return
	}
	
	form := tview.NewForm()
	form.SetBorder(true).SetTitle(fmt.Sprintf("Add Comment - %s", task.Name))
	
	var content string
	
	form.AddTextArea("Comment", "", 60, 5, 500, func(text string) {
		content = text
	})
	
	originalRoot := t.root
	
	form.AddButton("Add Comment", func() {
		if strings.TrimSpace(content) == "" {
			t.setStatus("Comment content is required")
			return
		}
		
		commentReq := &client.AddCommentRequest{
			Content:   content,
			IsPrivate: true, // Comments are always private
		}
		
		err := t.client.AddComment(task.ID, commentReq)
		if err != nil {
			t.setStatus(fmt.Sprintf("Error adding comment: %v", err))
		} else {
			t.setStatus(fmt.Sprintf("Added comment to task: %s", task.Name))
			t.refreshData()
		}
		
		t.enableGlobalKeys()
		t.app.SetRoot(originalRoot, true)
	})
	
	form.AddButton("Cancel", func() {
		t.enableGlobalKeys()
		t.app.SetRoot(originalRoot, true)
	})
	
	// Disable global keys while form is active
	t.disableGlobalKeys()
	t.app.SetRoot(form, true)
}

// showEditDialog shows the task edit dialog
func (t *TUI) showEditDialog() {
	task := t.getSelectedTask()
	if task == nil {
		t.setStatus("No task selected")
		return
	}
	
	form := tview.NewForm()
	form.SetBorder(true).SetTitle(fmt.Sprintf("Edit Task #%d", task.ID))
	
	var name, description, priority, tagsStr string
	
	// Initialize with current values
	name = task.Name
	description = task.Description
	priority = string(task.Priority)
	if priority == "" {
		priority = "none"
	}
	
	// Convert tags array to comma-separated string
	tagsStr = strings.Join(task.Tags, ", ")
	
	form.AddInputField("Name", name, 60, nil, func(text string) {
		name = text
	})
	
	form.AddTextArea("Description", description, 60, 3, 500, func(text string) {
		description = text
	})
	
	form.AddInputField("Tags", tagsStr, 60, nil, func(text string) {
		tagsStr = text
	})
	
	// Priority dropdown options
	priorityOptions := []string{"none", "low", "medium", "high"}
	currentPriorityIndex := 0
	for i, p := range priorityOptions {
		if p == priority {
			currentPriorityIndex = i
			break
		}
	}
	
	form.AddDropDown("Priority", priorityOptions, currentPriorityIndex, func(option string, optionIndex int) {
		priority = option
	})
	
	originalRoot := t.root
	
	form.AddButton("Save", func() {
		if strings.TrimSpace(name) == "" {
			t.setStatus("Task name is required")
			return
		}
		
		// Parse tags from comma-separated string
		var tags []string
		if strings.TrimSpace(tagsStr) != "" {
			tagParts := strings.Split(tagsStr, ",")
			for _, tag := range tagParts {
				trimmedTag := strings.TrimSpace(tag)
				if trimmedTag != "" {
					tags = append(tags, trimmedTag)
				}
			}
		}
		
		updateReq := &client.UpdateTaskRequest{
			Name:        name,
			Description: description,
			Tags:        tags,
		}
		
		// Only set priority if it's not "none"
		if priority != "none" {
			updateReq.Priority = priority
		}
		
		_, err := t.client.UpdateTask(task.ID, updateReq)
		if err != nil {
			t.setStatus(fmt.Sprintf("Error updating task: %v", err))
		} else {
			t.setStatus(fmt.Sprintf("Updated task: %s", name))
			t.refreshData()
		}
		
		t.enableGlobalKeys()
		t.app.SetRoot(originalRoot, true)
	})
	
	form.AddButton("Cancel", func() {
		t.enableGlobalKeys()
		t.app.SetRoot(originalRoot, true)
	})
	
	// Disable global keys while form is active
	t.disableGlobalKeys()
	t.app.SetRoot(form, true)
}

// showNewQueryDialog shows the new saved query dialog
func (t *TUI) showNewQueryDialog() {
	form := tview.NewForm()
	form.SetBorder(true).SetTitle("New Saved Query")
	
	var name, includedTags, excludedTags string
	
	form.AddInputField("Name", "", 60, nil, func(text string) {
		name = text
	})
	
	form.AddInputField("Included Tags", "", 60, nil, func(text string) {
		includedTags = text
	})
	
	form.AddInputField("Excluded Tags", "", 60, nil, func(text string) {
		excludedTags = text
	})
	
	// Add help text
	form.AddTextView("Help", "Enter comma-separated tags. Example: urgent, backend", 60, 2, true, false)
	
	originalRoot := t.root
	
	form.AddButton("Create", func() {
		if strings.TrimSpace(name) == "" {
			t.setStatus("Query name is required")
			return
		}
		
		// Parse included tags
		var includedTagsList []string
		if strings.TrimSpace(includedTags) != "" {
			tagParts := strings.Split(includedTags, ",")
			for _, tag := range tagParts {
				trimmedTag := strings.TrimSpace(tag)
				if trimmedTag != "" {
					includedTagsList = append(includedTagsList, trimmedTag)
				}
			}
		}
		
		// Parse excluded tags
		var excludedTagsList []string
		if strings.TrimSpace(excludedTags) != "" {
			tagParts := strings.Split(excludedTags, ",")
			for _, tag := range tagParts {
				trimmedTag := strings.TrimSpace(tag)
				if trimmedTag != "" {
					excludedTagsList = append(excludedTagsList, trimmedTag)
				}
			}
		}
		
		createReq := &client.CreateSavedQueryRequest{
			Name:         name,
			IncludedTags: includedTagsList,
			ExcludedTags: excludedTagsList,
		}
		
		savedQuery, err := t.client.CreateSavedQuery(createReq)
		if err != nil {
			t.setStatus(fmt.Sprintf("Error creating saved query: %v", err))
		} else {
			t.setStatus(fmt.Sprintf("Created saved query: %s", savedQuery.Name))
			// Select the new query after creation and refresh
			t.selectedQuery = fmt.Sprintf("saved:%d", savedQuery.ID)
			// Only reload saved queries and refresh tasks, don't reload header to avoid potential loops
			if err := t.loadSavedQueries(); err != nil {
				t.setStatus(fmt.Sprintf("Error loading saved queries: %v", err))
			} else {
				t.refreshTasksOnly()
			}
		}
		
		t.enableGlobalKeys()
		t.app.SetRoot(originalRoot, true)
	})
	
	form.AddButton("Cancel", func() {
		t.enableGlobalKeys()
		t.app.SetRoot(originalRoot, true)
	})
	
	// Disable global keys while form is active
	t.disableGlobalKeys()
	t.app.SetRoot(form, true)
}

// showCreateTaskModal shows a modal for creating a new task with inline tag parsing
func (t *TUI) showCreateTaskModal() {
	// Create an input field for the task input
	inputField := tview.NewInputField().
		SetLabel("Task: ").
		SetFieldWidth(60)
	
	inputField.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			input := inputField.GetText()
			if strings.TrimSpace(input) != "" {
				t.createTaskFromInput(input)
			}
			t.enableGlobalKeys()
			t.app.SetRoot(t.root, true)
		} else if key == tcell.KeyEscape {
			t.enableGlobalKeys()
			t.app.SetRoot(t.root, true)
		}
	})

	// Create a flex container for the input
	flex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(tview.NewTextView().SetText("Create new task (supports +tag, @tag, -c, -t 15m, -d -1d):").SetTextAlign(tview.AlignCenter), 1, 0, false).
		AddItem(tview.NewTextView(), 1, 0, false). // Spacer
		AddItem(inputField, 1, 0, true).
		AddItem(tview.NewTextView(), 1, 0, false). // Spacer
		AddItem(tview.NewTextView().SetText("Enter: Create | Escape: Cancel").SetTextAlign(tview.AlignCenter), 1, 0, false)

	// Set border and title
	flex.SetBorder(true).SetTitle("New Task")

	// Create a centered modal
	modalContainer := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(flex, 7, 0, true).
			AddItem(nil, 0, 1, false), 80, 0, true).
		AddItem(nil, 0, 1, false)

	// Disable global keys while modal is active
	t.disableGlobalKeys()
	t.app.SetRoot(modalContainer, true)
	t.app.SetFocus(inputField)
}

// createTaskFromInput parses the input and creates a task with inline tags and flags
func (t *TUI) createTaskFromInput(input string) {
	// Parse flags first
	var shouldComplete bool
	var timeSpent string
	var priority string
	var date string

	// Split input into words and process flags
	words := strings.Fields(input)
	var filteredWords []string
	
	for i := 0; i < len(words); i++ {
		word := words[i]
		if word == "-c" {
			shouldComplete = true
		} else if word == "-t" && i+1 < len(words) {
			timeSpent = words[i+1]
			i++ // Skip next word as it's the time value
		} else if word == "-p" && i+1 < len(words) {
			priority = words[i+1]
			i++ // Skip next word as it's the priority value
		} else if word == "-d" && i+1 < len(words) {
			date = words[i+1]
			i++ // Skip next word as it's the date value
		} else if strings.HasPrefix(word, "-t") && len(word) > 2 {
			// Handle -t15m format
			timeSpent = word[2:]
		} else if strings.HasPrefix(word, "-p") && len(word) > 2 {
			// Handle -phigh format
			priority = word[2:]
		} else if strings.HasPrefix(word, "-d") && len(word) > 2 {
			// Handle -d-1d format
			date = word[2:]
		} else if !strings.HasPrefix(word, "-") {
			filteredWords = append(filteredWords, word)
		}
	}

	// Rejoin the filtered input for tag parsing
	filteredInput := strings.Join(filteredWords, " ")
	
	// Parse task name and tags using the same logic as CLI add command
	taskName, tags := t.parseTaskNameAndTags(filteredInput)
	
	if strings.TrimSpace(taskName) == "" {
		t.setStatus("Task name cannot be empty")
		return
	}

	// Create the task
	req := &client.CreateTaskRequest{
		Name:     taskName,
		Priority: priority,
		Tags:     tags,
		Date:     date,
	}

	task, err := t.client.CreateTask(req)
	if err != nil {
		t.setStatus(fmt.Sprintf("Error creating task: %v", err))
		return
	}

	t.setStatus(fmt.Sprintf("✓ Created task #%d: %s", task.ID, task.Name))

	// Log time if specified
	if timeSpent != "" {
		durationMinutes, err := client.ParseDuration(timeSpent)
		if err != nil {
			t.setStatus(fmt.Sprintf("Invalid duration format: %v", err))
			return
		}

		timeReq := &client.LogTimeRequest{
			Duration:    durationMinutes,
			Description: "Time logged during task creation",
			Date:        date,
		}

		err = t.client.LogTime(task.ID, timeReq)
		if err != nil {
			t.setStatus(fmt.Sprintf("Error logging time: %v", err))
		} else {
			t.setStatus(fmt.Sprintf("✓ Created task #%d: %s, logged %s", task.ID, task.Name, timeSpent))
		}
	}

	// Mark as completed if specified
	if shouldComplete {
		_, err := t.client.UpdateTaskStatus(task.ID, "resolved")
		if err != nil {
			t.setStatus(fmt.Sprintf("Error completing task: %v", err))
		} else {
			status := fmt.Sprintf("✓ Created and completed task #%d: %s", task.ID, task.Name)
			if timeSpent != "" {
				status += fmt.Sprintf(", logged %s", timeSpent)
			}
			t.setStatus(status)
		}
	}

	t.refreshData()
}

// parseTaskNameAndTags extracts tags from task name and returns cleaned name and tags (copied from add.go)
func (t *TUI) parseTaskNameAndTags(input string) (string, []string) {
	words := strings.Fields(input)
	var cleanWords []string
	var tags []string
	
	for _, word := range words {
		if strings.HasPrefix(word, "+") {
			// +tag format: add tag and remove + from name
			tag := strings.TrimPrefix(word, "+")
			if tag != "" {
				tags = append(tags, tag)
				cleanWords = append(cleanWords, tag) // Add the tag word without + to the name
			}
		} else if strings.HasPrefix(word, "@") {
			// @tag format: add tag but remove entire @tag from name
			tag := strings.TrimPrefix(word, "@")
			if tag != "" {
				tags = append(tags, tag)
			}
			// Don't add this word to cleanWords (it gets removed from name)
		} else {
			// Regular word, keep in name
			cleanWords = append(cleanWords, word)
		}
	}
	
	cleanName := strings.Join(cleanWords, " ")
	return strings.TrimSpace(cleanName), tags
}

// viewTaskDetails shows detailed view of the selected task
func (t *TUI) viewTaskDetails() {
	selectedTask := t.getSelectedTask()
	if selectedTask == nil {
		t.setStatus("No task selected")
		return
	}
	
	// Fetch full task details including comments from API
	fullTask, err := t.client.GetTask(selectedTask.ID)
	if err != nil {
		t.setStatus(fmt.Sprintf("Error fetching task details: %v", err))
		return
	}
	
	// Convert models.Task to client.Task for display
	task := &client.Task{
		ID:          fullTask.ID,
		Name:        fullTask.Name,
		Description: fullTask.Description,
		Status:      fullTask.Status,
		Priority:    fullTask.Priority,
		Tags:        fullTask.Tags,
		CreatedAt:   fullTask.CreatedAt,
		UpdatedAt:   fullTask.UpdatedAt,
	}
	
	// Convert TimeEntries
	for _, te := range fullTask.TimeEntries {
		task.TimeEntries = append(task.TimeEntries, client.TimeEntry{
			ID:          te.ID,
			Description: te.Description,
			Duration:    te.Duration,
			CreatedAt:   te.CreatedAt,
		})
	}
	
	// Convert Comments
	for _, c := range fullTask.Comments {
		task.Comments = append(task.Comments, client.Comment{
			ID:        c.ID,
			Content:   c.Content,
			IsPrivate: c.IsPrivate,
			CreatedAt: c.CreatedAt,
		})
	}
	
	// Create detailed view
	detailsText := fmt.Sprintf(`[yellow]Task #%d[white]
[white]Name:[white] %s
[white]Status:[white] %s
[white]Priority:[white] %s
[white]Tags:[white] %s
[white]Created:[white] %s
[white]Updated:[white] %s

[yellow]Description:[white]
%s

[yellow]Time Entries:[white]
`, task.ID, task.Name, task.Status, task.Priority, 
	strings.Join(task.Tags, ", "), 
	task.CreatedAt.Format("2006-01-02 15:04:05"),
	task.UpdatedAt.Format("2006-01-02 15:04:05"),
	task.Description)
	
	totalTime := 0
	for _, entry := range task.TimeEntries {
		totalTime += entry.Duration
		detailsText += fmt.Sprintf("• %s - %s (%s)\n", 
			entry.CreatedAt.Format("2006-01-02 15:04"), 
			entry.Description,
			tuiFormatDuration(entry.Duration))
	}
	
	if len(task.TimeEntries) == 0 {
		detailsText += "No time entries\n"
	}
	
	detailsText += fmt.Sprintf("\n[yellow]Total Time:[white] %s", tuiFormatDuration(totalTime))
	
	if len(task.Comments) > 0 {
		detailsText += "\n\n[yellow]Comments:[white]\n"
		for _, comment := range task.Comments {
			detailsText += fmt.Sprintf("• [%s] %s\n", 
				comment.CreatedAt.Format("2006-01-02 15:04"), 
				comment.Content)
		}
	}
	
	textView := tview.NewTextView().
		SetText(detailsText).
		SetDynamicColors(true).
		SetScrollable(true)
	textView.SetBorder(true).SetTitle("Task Details (ESC to close)")
	
	originalRoot := t.root
	
	textView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			t.enableGlobalKeys()
			t.app.SetRoot(originalRoot, true)
			return nil
		}
		return event
	})
	
	// Disable global keys while viewing details
	t.disableGlobalKeys()
	t.app.SetRoot(textView, true)
}

// setStatus updates the status bar with a temporary message, then restores context-aware status
func (t *TUI) setStatus(message string) {
	// Show the message temporarily
	t.statusBar.SetText(fmt.Sprintf("[white]%s", message))
	
	// After a short delay, restore the context-appropriate status bar
	go func() {
		time.Sleep(3 * time.Second)
		focused := t.app.GetFocus()
		if focused == t.tasksTable {
			t.updateStatusForPane("tasks")
		} else if focused == t.sidebar {
			t.updateStatusForPane("queries")
		}
	}()
}

// tuiFormatDuration formats duration in minutes to human readable format
func tuiFormatDuration(minutes int) string {
	if minutes == 0 {
		return "0m"
	}
	
	hours := minutes / 60
	mins := minutes % 60
	
	if hours == 0 {
		return fmt.Sprintf("%dm", mins)
	} else if mins == 0 {
		return fmt.Sprintf("%dh", hours)
	} else {
		return fmt.Sprintf("%dh%dm", hours, mins)
	}
}

// nextPage moves to the next page of tasks
func (t *TUI) nextPage() {
	t.currentPage++
	if err := t.refreshTasksOnly(); err != nil {
		t.setStatus(fmt.Sprintf("Error loading next page: %v", err))
		t.currentPage-- // Revert on error
	}
}

// prevPage moves to the previous page of tasks
func (t *TUI) prevPage() {
	if t.currentPage > 0 {
		t.currentPage--
		if err := t.refreshTasksOnly(); err != nil {
			t.setStatus(fmt.Sprintf("Error loading previous page: %v", err))
			t.currentPage++ // Revert on error
		}
	} else {
		t.setStatus("Already on first page")
	}
}

// showSearchDialog shows the search input dialog
func (t *TUI) showSearchDialog() {
	inputField := tview.NewInputField().
		SetLabel("Search tasks: ").
		SetText(t.searchQuery).
		SetFieldWidth(50)
		
	inputField.SetBorder(true).SetTitle("Search (ESC to cancel)")
	
	originalRoot := t.root
	
	inputField.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter:
			searchText := inputField.GetText()
			t.searchQuery = searchText
			t.searchActive = searchText != ""
			t.currentPage = 0 // Reset to first page on new search
			
			t.enableGlobalKeys()
			t.app.SetRoot(originalRoot, true)
			
			if err := t.refreshTasksOnly(); err != nil {
				t.setStatus(fmt.Sprintf("Search error: %v", err))
			} else if searchText == "" {
				t.setStatus("Search cleared")
			} else {
				t.setStatus(fmt.Sprintf("Searching for: %s", searchText))
			}
			
		case tcell.KeyEscape:
			t.enableGlobalKeys()
			t.app.SetRoot(originalRoot, true)
			t.setStatus("Search cancelled")
		}
	})
	
	// Disable global keys while searching
	t.disableGlobalKeys()
	t.app.SetRoot(inputField, true)
	t.app.SetFocus(inputField)
}

// clearSearch clears the current search and resets to page 1
func (t *TUI) clearSearch() {
	if t.searchActive {
		t.searchQuery = ""
		t.searchActive = false
		t.currentPage = 0
		
		if err := t.refreshTasksOnly(); err != nil {
			t.setStatus(fmt.Sprintf("Error clearing search: %v", err))
		} else {
			t.setStatus("Search cleared")
		}
	} else {
		t.setStatus("No active search to clear")
	}
}

func init() {
	rootCmd.AddCommand(tuiCmd)
}