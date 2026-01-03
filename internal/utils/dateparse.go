package utils

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	relativeDatePattern = regexp.MustCompile(`^([+-]?)(\d+)([dwmy])$`)
	absoluteDatePattern = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
)

// ParseDate parses both relative dates (like "-1d", "+2w") and absolute dates (like "2025-12-01")
// Returns a time.Time with the current time but the specified date
func ParseDate(dateStr string) (time.Time, error) {
	now := time.Now()
	
	if dateStr == "" {
		return now, nil
	}
	
	dateStr = strings.TrimSpace(dateStr)
	
	// Check if it's an absolute date (YYYY-MM-DD format)
	if absoluteDatePattern.MatchString(dateStr) {
		parsedDate, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid date format: %s", dateStr)
		}
		
		// Combine the parsed date with current time
		return time.Date(
			parsedDate.Year(), parsedDate.Month(), parsedDate.Day(),
			now.Hour(), now.Minute(), now.Second(), now.Nanosecond(),
			now.Location(),
		), nil
	}
	
	// Check if it's a relative date
	matches := relativeDatePattern.FindStringSubmatch(dateStr)
	if len(matches) != 4 {
		return time.Time{}, fmt.Errorf("invalid date format: %s (expected formats: YYYY-MM-DD, ±Nd, ±Nw, ±Nm, ±Ny)", dateStr)
	}
	
	sign := matches[1]
	amountStr := matches[2]
	unit := matches[3]
	
	amount, err := strconv.Atoi(amountStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid amount in date: %s", dateStr)
	}
	
	// Handle sign (default to negative if no sign provided for backward compatibility)
	if sign == "+" {
		// amount stays positive
	} else if sign == "-" || sign == "" {
		amount = -amount
	}
	
	// Apply the relative offset
	var resultTime time.Time
	switch unit {
	case "d":
		resultTime = now.AddDate(0, 0, amount)
	case "w":
		resultTime = now.AddDate(0, 0, amount*7)
	case "m":
		resultTime = now.AddDate(0, amount, 0)
	case "y":
		resultTime = now.AddDate(amount, 0, 0)
	default:
		return time.Time{}, fmt.Errorf("invalid time unit: %s (expected d, w, m, y)", unit)
	}
	
	return resultTime, nil
}