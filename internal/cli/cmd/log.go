package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/soarinferret/jats/internal/cli/client"
)

var (
	note string
)

var logCmd = &cobra.Command{
	Use:   "log <task-id> <duration>",
	Short: "Log time spent on a task",
	Long: `Log time spent working on a task.

Duration formats:
  30m, 1h, 2h30m, 1.5h, or just 30 (assumes minutes)

Examples:
  jats log 123 30m                    # Log 30 minutes
  jats log 123 1h --note "debugging"  # Log 1 hour with note
  jats log 123 2.5h                   # Log 2.5 hours`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		c := client.New()
		
		var taskID uint
		if _, err := fmt.Sscanf(args[0], "%d", &taskID); err != nil {
			return fmt.Errorf("invalid task ID: %s", args[0])
		}

		durationMinutes, err := client.ParseDuration(args[1])
		if err != nil {
			return err
		}

		req := &client.LogTimeRequest{
			Duration:    durationMinutes,
			Description: note,
		}

		err = c.LogTime(taskID, req)
		if err != nil {
			return fmt.Errorf("failed to log time: %w", err)
		}

		// Format duration for display
		duration := time.Duration(durationMinutes) * time.Minute
		fmt.Printf("✓ Logged %s to task #%d\n", formatDurationDisplay(duration), taskID)
		if note != "" {
			fmt.Printf("  Note: %s\n", note)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(logCmd)
	logCmd.Flags().StringVarP(&note, "note", "n", "", "Note describing the work done")
}

func formatDurationDisplay(d time.Duration) string {
	if d == 0 {
		return "0m"
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