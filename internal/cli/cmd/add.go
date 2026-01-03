package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/soarinferret/jats/internal/cli/client"
)

var (
	priority  string
	timeSpent string
	completed bool
)

var addCmd = &cobra.Command{
	Use:   "add <task name>",
	Short: "Create a new task",
	Long: `Create a new task with optional tags, priority, time logging, and completion.

Task names can include inline tags:
  +tag    - Adds 'tag' to the task and removes '+tag' from the name
  @tag    - Adds 'tag' to the task and removes '@tag' from the name

Workflow flags:
  -t      - Log time immediately (30m, 1h, 2h30m, etc.)
  -c      - Mark task as resolved after creation

Examples:
  jats add Fix authentication bug
  jats add Update documentation +docs +urgent --priority high
  jats add @client1 restart +docker container -t 45m -c
  jats add testing new +framework -t 30m -c
  jats add "Fix bug with spaces" -t 1h (quotes optional but supported)`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Join all arguments to form the full task name
		fullTaskName := strings.Join(args, " ")
		
		// Parse inline tags from the task name
		taskName, tags := parseTaskNameAndTags(fullTaskName)

		c := client.New()
		
		req := &client.CreateTaskRequest{
			Name:     taskName,
			Priority: priority,
			Tags:     tags,
		}

		task, err := c.CreateTask(req)
		if err != nil {
			return fmt.Errorf("failed to create task: %w", err)
		}

		fmt.Printf("✓ Created task #%d: %s\n", task.ID, task.Name)
		if len(task.Tags) > 0 {
			fmt.Printf("  Tags: %s\n", strings.Join(task.Tags, ", "))
		}
		if task.Priority != "" {
			fmt.Printf("  Priority: %s\n", task.Priority)
		}

		// Log time if specified
		if timeSpent != "" {
			durationMinutes, err := client.ParseDuration(timeSpent)
			if err != nil {
				return fmt.Errorf("invalid duration format: %w", err)
			}

			timeReq := &client.LogTimeRequest{
				Duration:    durationMinutes,
				Description: "Time logged during task creation",
			}

			err = c.LogTime(task.ID, timeReq)
			if err != nil {
				return fmt.Errorf("failed to log time: %w", err)
			}

			fmt.Printf("  ⏱️  Logged %s\n", timeSpent)
		}

		// Mark as completed if specified
		if completed {
			updatedTask, err := c.UpdateTaskStatus(task.ID, "resolved")
			if err != nil {
				return fmt.Errorf("failed to mark task as completed: %w", err)
			}
			fmt.Printf("  ✅ Marked as resolved\n")
			_ = updatedTask // Use the variable to avoid unused warning
		}

		return nil
	},
}

// parseTaskNameAndTags extracts tags from task name and returns cleaned name and tags
func parseTaskNameAndTags(input string) (string, []string) {
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

func init() {
	rootCmd.AddCommand(addCmd)
	addCmd.Flags().StringVarP(&priority, "priority", "p", "", "Priority level (low, medium, high)")
	addCmd.Flags().StringVarP(&timeSpent, "time", "t", "", "Log time immediately (30m, 1h, 2h30m, etc.)")
	addCmd.Flags().BoolVarP(&completed, "complete", "c", false, "Mark task as resolved after creation")
}