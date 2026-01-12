package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/soarinferret/jats/internal/cli/client"
)

var queriesCmd = &cobra.Command{
	Use:   "queries",
	Short: "List saved queries",
	Long:  `List all saved queries with their IDs and tag filters.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c := client.New()
		queries, err := c.GetSavedQueries()
		if err != nil {
			return fmt.Errorf("failed to get saved queries: %w", err)
		}

		if len(queries) == 0 {
			fmt.Println("No saved queries found")
			return nil
		}

		fmt.Printf("\nSaved Queries:\n")
		fmt.Printf("%-5s | %-30s | %-40s | %-40s\n", "ID", "Name", "Included Tags", "Excluded Tags")
		fmt.Printf("%s\n", strings.Repeat("-", 120))

		for _, q := range queries {
			includedTags := strings.Join(q.IncludedTags, ", ")
			if includedTags == "" {
				includedTags = "(none)"
			}
			excludedTags := strings.Join(q.ExcludedTags, ", ")
			if excludedTags == "" {
				excludedTags = "(none)"
			}

			fmt.Printf("%-5d | %-30s | %-40s | %-40s\n", q.ID, q.Name, includedTags, excludedTags)
		}
		fmt.Printf("\n")

		return nil
	},
}

func init() {
	rootCmd.AddCommand(queriesCmd)
}
