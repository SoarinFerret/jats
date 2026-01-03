package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/soarinferret/jats/internal/cli/client"
)

var (
	listStatus   string
	listTag      string
	listPriority string
	listLimit    int
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List tasks",
	Long: `List tasks with optional filtering.

Examples:
  jats list                    # List all tasks
  jats list --status open      # List open tasks
  jats list --tag urgent       # List tasks with 'urgent' tag
  jats list --priority high    # List high priority tasks
  jats list --limit 10         # Limit to 10 tasks`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c := client.New()
		
		filters := &client.TaskFilters{}
		if listStatus != "" {
			filters.Status = []string{listStatus}
		}
		if listTag != "" {
			filters.Tags = []string{listTag}
		}
		if listPriority != "" {
			filters.Priority = []string{listPriority}
		}
		if listLimit > 0 {
			filters.Limit = listLimit
		}

		tasks, err := c.GetTasks(filters)
		if err != nil {
			return fmt.Errorf("failed to get tasks: %w", err)
		}

		if len(tasks) == 0 {
			fmt.Println("No tasks found")
			return nil
		}

		// Print header
		fmt.Printf("%-4s %-10s %-8s %-15s %s\n", "ID", "STATUS", "PRIORITY", "TAGS", "NAME")
		fmt.Println(strings.Repeat("-", 80))

		for _, task := range tasks {
			statusStr := string(task.Status)
			if statusStr == "" {
				statusStr = "open"
			}
			
			priorityStr := string(task.Priority)
			if priorityStr == "" {
				priorityStr = "-"
			}

			tagsStr := "-"
			if len(task.Tags) > 0 {
				tagsStr = strings.Join(task.Tags, ",")
				if len(tagsStr) > 15 {
					tagsStr = tagsStr[:12] + "..."
				}
			}

			nameStr := task.Name
			if len(nameStr) > 40 {
				nameStr = nameStr[:37] + "..."
			}

			fmt.Printf("%-4d %-10s %-8s %-15s %s\n", 
				task.ID, statusStr, priorityStr, tagsStr, nameStr)
		}

		fmt.Printf("\nTotal: %d tasks\n", len(tasks))

		return nil
	},
}

var showCmd = &cobra.Command{
	Use:   "show <task-id>",
	Short: "Show detailed information about a task",
	Long: `Show detailed information about a specific task including comments and time entries.

Examples:
  jats show 123`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c := client.New()
		
		var taskID uint
		if _, err := fmt.Sscanf(args[0], "%d", &taskID); err != nil {
			return fmt.Errorf("invalid task ID: %s", args[0])
		}

		task, err := c.GetTask(taskID)
		if err != nil {
			return fmt.Errorf("failed to get task: %w", err)
		}

		// Print task details
		fmt.Printf("Task #%d: %s\n", task.ID, task.Name)
		fmt.Println(strings.Repeat("=", len(fmt.Sprintf("Task #%d: %s", task.ID, task.Name))))
		
		fmt.Printf("Status:      %s\n", getStatus(string(task.Status)))
		fmt.Printf("Priority:    %s\n", getPriority(string(task.Priority)))
		fmt.Printf("Created:     %s\n", task.CreatedAt.Format("2006-01-02 15:04"))
		fmt.Printf("Updated:     %s\n", task.UpdatedAt.Format("2006-01-02 15:04"))
		
		if len(task.Tags) > 0 {
			fmt.Printf("Tags:        %s\n", strings.Join(task.Tags, ", "))
		}

		// Calculate total time
		var totalMinutes int
		for _, entry := range task.TimeEntries {
			totalMinutes += entry.Duration
		}
		totalDuration := time.Duration(totalMinutes) * time.Minute
		fmt.Printf("Time logged: %s\n", formatDuration(totalDuration))

		// Show subtasks if any
		if len(task.Subtasks) > 0 {
			fmt.Printf("\nSubtasks:\n")
			for _, subtask := range task.Subtasks {
				status := "☐"
				if subtask.Completed {
					status = "☑"
				}
				fmt.Printf("  %s %s\n", status, subtask.Name)
			}
		}

		// Show time entries if any
		if len(task.TimeEntries) > 0 {
			fmt.Printf("\nTime entries:\n")
			for _, entry := range task.TimeEntries {
				duration := time.Duration(entry.Duration) * time.Minute
				fmt.Printf("  %s - %s", entry.CreatedAt.Format("2006-01-02 15:04"), formatDuration(duration))
				if entry.Description != "" {
					fmt.Printf(" (%s)", entry.Description)
				}
				fmt.Println()
			}
		}

		// Show comments if any
		if len(task.Comments) > 0 {
			fmt.Printf("\nComments:\n")
			for _, comment := range task.Comments {
				fmt.Printf("  %s: %s\n", comment.CreatedAt.Format("2006-01-02 15:04"), comment.Content)
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(showCmd)
	
	listCmd.Flags().StringVarP(&listStatus, "status", "s", "", "Filter by status (open, in-progress, resolved, closed)")
	listCmd.Flags().StringVarP(&listTag, "tag", "t", "", "Filter by tag")
	listCmd.Flags().StringVarP(&listPriority, "priority", "p", "", "Filter by priority (low, medium, high)")
	listCmd.Flags().IntVarP(&listLimit, "limit", "l", 0, "Limit number of results")
}

func getStatus(status string) string {
	if status == "" {
		return "open"
	}
	return status
}

func getPriority(priority string) string {
	if priority == "" {
		return "none"
	}
	return priority
}

func formatDuration(d time.Duration) string {
	if d == 0 {
		return "none"
	}
	
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	
	if hours > 0 {
		if minutes > 0 {
			return fmt.Sprintf("%dh %dm", hours, minutes)
		}
		return fmt.Sprintf("%dh", hours)
	}
	return fmt.Sprintf("%dm", minutes)
}