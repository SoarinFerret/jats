package cmd

import (
	"encoding/csv"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/soarinferret/jats/internal/cli/client"
)

var reportCmd = &cobra.Command{
	Use:   "report",
	Short: "Generate reports",
	Long:  `Commands for generating various reports from JATS data.`,
}

var timeBreakdownCmd = &cobra.Command{
	Use:   "time-breakdown",
	Short: "Generate time breakdown report by saved queries",
	Long: `Generate a time breakdown report showing time spent on tasks grouped by saved queries.

The report shows daily time entries broken down by saved query, with totals and percentages.

Examples:
  # Report for last 7 days with saved queries 1 and 2
  jats report time-breakdown --start 2024-01-01 --end 2024-01-07 --queries 1,2

  # Report excluding specific tags
  jats report time-breakdown --start 2024-01-01 --end 2024-01-07 --queries 1,2,3 --exclude personal,internal

  # Export report to CSV file
  jats report time-breakdown --start 2024-01-01 --end 2024-01-07 --queries 1,2,3 --csv report.csv`,
	RunE: runTimeBreakdownReport,
}

var (
	startDate    string
	endDate      string
	queryIDs     string
	excludeTags  string
	csvOutput    string
)

func runTimeBreakdownReport(cmd *cobra.Command, args []string) error {
	// Validate required flags
	if startDate == "" || endDate == "" || queryIDs == "" {
		return fmt.Errorf("--start, --end, and --queries are required")
	}

	// Parse dates
	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return fmt.Errorf("invalid start date format (expected YYYY-MM-DD): %w", err)
	}

	end, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		return fmt.Errorf("invalid end date format (expected YYYY-MM-DD): %w", err)
	}

	// Fetch report from API
	c := client.New()
	report, err := c.GetTimeBreakdownReport(startDate, endDate, queryIDs, excludeTags)
	if err != nil {
		return fmt.Errorf("failed to fetch report: %w", err)
	}

	// Export to CSV if requested
	if csvOutput != "" {
		if err := exportReportToCSV(report, csvOutput); err != nil {
			return fmt.Errorf("failed to export CSV: %w", err)
		}
		fmt.Printf("Report exported to: %s\n", csvOutput)
		return nil
	}

	// Display report to terminal
	displayTimeBreakdownReport(report, start, end)
	return nil
}

func displayTimeBreakdownReport(report *client.TimeBreakdownReport, start, end time.Time) {
	fmt.Printf("\n")
	fmt.Printf("Time Breakdown Report: %s to %s\n", start.Format("2006-01-02"), end.Format("2006-01-02"))
	fmt.Printf("================================================================================\n\n")

	// Display header row
	fmt.Printf("%-10s | %-12s", "Date", "Total Time")
	for _, queryName := range report.QueryNames {
		// Truncate long query names
		displayName := queryName
		if len(displayName) > 20 {
			displayName = displayName[:17] + "..."
		}
		fmt.Printf(" | %-20s", displayName)
	}
	fmt.Printf(" | %-20s", "Other")
	fmt.Printf("\n")

	// Display separator
	totalWidth := 10 + 3 + 12 + (len(report.QueryNames) * (3 + 20)) + 3 + 20
	fmt.Printf("%s\n", strings.Repeat("-", totalWidth))

	// Display daily data
	for _, daily := range report.DailyData {
		fmt.Printf("%-10s | %-12s", daily.Date, formatMinutes(daily.TotalTime))

		for _, queryTime := range daily.QueryTimes {
			timeStr := formatMinutes(queryTime.Time)

			// Add tags if present
			if len(queryTime.Tags) > 0 {
				tagsStr := strings.Join(queryTime.Tags, ", ")
				if len(tagsStr) > 15 {
					tagsStr = tagsStr[:12] + "..."
				}
				timeStr = fmt.Sprintf("%s (%s)", formatMinutes(queryTime.Time), tagsStr)
			}

			// Truncate if too long
			if len(timeStr) > 20 {
				timeStr = timeStr[:17] + "..."
			}

			fmt.Printf(" | %-20s", timeStr)
		}

		// Display Other column
		otherStr := formatMinutes(daily.OtherTime)
		if len(daily.OtherTags) > 0 {
			tagsStr := strings.Join(daily.OtherTags, ", ")
			if len(tagsStr) > 15 {
				tagsStr = tagsStr[:12] + "..."
			}
			otherStr = fmt.Sprintf("%s (%s)", formatMinutes(daily.OtherTime), tagsStr)
		}
		if len(otherStr) > 20 {
			otherStr = otherStr[:17] + "..."
		}
		fmt.Printf(" | %-20s", otherStr)

		fmt.Printf("\n")
	}

	// Display separator
	fmt.Printf("%s\n", strings.Repeat("-", totalWidth))

	// Display totals row
	fmt.Printf("%-10s | %-12s", "Total", formatMinutes(report.Totals.TotalTime))
	for _, queryTotal := range report.Totals.QueryTotals {
		fmt.Printf(" | %-20s", formatMinutes(queryTotal.TotalTime))
	}
	fmt.Printf(" | %-20s", formatMinutes(report.Totals.OtherTotal.TotalTime))
	fmt.Printf("\n")

	// Display percentages row
	fmt.Printf("%-10s | %-12s", "Percent", "100%")
	for _, queryTotal := range report.Totals.QueryTotals {
		fmt.Printf(" | %-20s", fmt.Sprintf("%.1f%%", queryTotal.Percentage))
	}
	fmt.Printf(" | %-20s", fmt.Sprintf("%.1f%%", report.Totals.OtherTotal.Percentage))
	fmt.Printf("\n\n")
}

func formatMinutes(minutes int) string {
	if minutes == 0 {
		return "0h"
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

func exportReportToCSV(report *client.TimeBreakdownReport, filename string) error {
	// Create CSV file
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header row
	header := []string{"Date", "Total Time (hours)"}
	for _, queryName := range report.QueryNames {
		header = append(header, queryName+" (hours)")
		header = append(header, queryName+" Tags")
	}
	header = append(header, "Other (hours)")
	header = append(header, "Other Tags")

	if err := writer.Write(header); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	// Write daily data rows
	for _, daily := range report.DailyData {
		row := []string{
			daily.Date,
			minutesToHoursDecimal(daily.TotalTime),
		}

		for _, queryTime := range daily.QueryTimes {
			row = append(row, minutesToHoursDecimal(queryTime.Time))
			row = append(row, strings.Join(queryTime.Tags, ", "))
		}

		row = append(row, minutesToHoursDecimal(daily.OtherTime))
		row = append(row, strings.Join(daily.OtherTags, ", "))

		if err := writer.Write(row); err != nil {
			return fmt.Errorf("failed to write data row: %w", err)
		}
	}

	// Write totals row
	totalsRow := []string{"Total", minutesToHoursDecimal(report.Totals.TotalTime)}
	for _, queryTotal := range report.Totals.QueryTotals {
		totalsRow = append(totalsRow, minutesToHoursDecimal(queryTotal.TotalTime))
		totalsRow = append(totalsRow, "") // No tags in totals
	}
	totalsRow = append(totalsRow, minutesToHoursDecimal(report.Totals.OtherTotal.TotalTime))
	totalsRow = append(totalsRow, strings.Join(report.Totals.OtherTotal.Tags, ", "))

	if err := writer.Write(totalsRow); err != nil {
		return fmt.Errorf("failed to write totals row: %w", err)
	}

	// Write percentages row
	percentRow := []string{"Percent", "100"}
	for _, queryTotal := range report.Totals.QueryTotals {
		percentRow = append(percentRow, fmt.Sprintf("%.1f", queryTotal.Percentage))
		percentRow = append(percentRow, "") // No tags in percentages
	}
	percentRow = append(percentRow, fmt.Sprintf("%.1f", report.Totals.OtherTotal.Percentage))
	percentRow = append(percentRow, "")

	if err := writer.Write(percentRow); err != nil {
		return fmt.Errorf("failed to write percent row: %w", err)
	}

	return nil
}

func minutesToHoursDecimal(minutes int) string {
	hours := float64(minutes) / 60.0
	return fmt.Sprintf("%.2f", hours)
}

func init() {
	rootCmd.AddCommand(reportCmd)
	reportCmd.AddCommand(timeBreakdownCmd)

	// Flags for time-breakdown command
	timeBreakdownCmd.Flags().StringVar(&startDate, "start", "", "Start date (YYYY-MM-DD)")
	timeBreakdownCmd.Flags().StringVar(&endDate, "end", "", "End date (YYYY-MM-DD)")
	timeBreakdownCmd.Flags().StringVar(&queryIDs, "queries", "", "Comma-separated list of saved query IDs")
	timeBreakdownCmd.Flags().StringVar(&excludeTags, "exclude", "", "Comma-separated list of tags to exclude")
	timeBreakdownCmd.Flags().StringVar(&csvOutput, "csv", "", "Export report to CSV file (e.g., report.csv)")

	timeBreakdownCmd.MarkFlagRequired("start")
	timeBreakdownCmd.MarkFlagRequired("end")
	timeBreakdownCmd.MarkFlagRequired("queries")
}
