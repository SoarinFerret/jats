package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/soarinferret/jats/internal/cli/client"
)

var closeCmd = &cobra.Command{
	Use:   "close <task-id>",
	Short: "Mark a task as resolved",
	Long: `Mark a task as resolved (completed).

Examples:
  jats close 123`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return updateTaskStatus(args[0], "resolved")
	},
}

var reopenCmd = &cobra.Command{
	Use:   "reopen <task-id>",
	Short: "Reopen a closed task",
	Long: `Reopen a previously resolved or closed task.

Examples:
  jats reopen 123`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return updateTaskStatus(args[0], "open")
	},
}

var startCmd = &cobra.Command{
	Use:   "start <task-id>",
	Short: "Start working on a task",
	Long: `Mark a task as in-progress.

Examples:
  jats start 123`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return updateTaskStatus(args[0], "in-progress")
	},
}

func updateTaskStatus(taskIDStr, status string) error {
	c := client.New()
	
	var taskID uint
	if _, err := fmt.Sscanf(taskIDStr, "%d", &taskID); err != nil {
		return fmt.Errorf("invalid task ID: %s", taskIDStr)
	}

	task, err := c.UpdateTaskStatus(taskID, status)
	if err != nil {
		return fmt.Errorf("failed to update task status: %w", err)
	}

	statusVerbs := map[string]string{
		"open":        "reopened",
		"in-progress": "started",
		"resolved":    "closed",
		"closed":      "closed",
	}

	verb := statusVerbs[status]
	if verb == "" {
		verb = fmt.Sprintf("marked as %s", status)
	}

	fmt.Printf("✓ Task #%d %s: %s\n", task.ID, verb, task.Name)
	return nil
}

func init() {
	rootCmd.AddCommand(closeCmd)
	rootCmd.AddCommand(reopenCmd)
	rootCmd.AddCommand(startCmd)
}