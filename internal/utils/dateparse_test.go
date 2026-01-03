package utils

import (
	"testing"
	"time"
)

func TestParseDate(t *testing.T) {
	now := time.Now()
	
	tests := []struct {
		name     string
		input    string
		wantErr  bool
		checkDay bool
		dayDiff  int
	}{
		{
			name:     "empty string returns current time",
			input:    "",
			wantErr:  false,
			checkDay: true,
			dayDiff:  0,
		},
		{
			name:     "absolute date",
			input:    "2025-12-01",
			wantErr:  false,
			checkDay: false,
		},
		{
			name:     "yesterday",
			input:    "-1d",
			wantErr:  false,
			checkDay: true,
			dayDiff:  -1,
		},
		{
			name:     "one day ago with explicit minus",
			input:    "-1d",
			wantErr:  false,
			checkDay: true,
			dayDiff:  -1,
		},
		{
			name:     "tomorrow",
			input:    "+1d",
			wantErr:  false,
			checkDay: true,
			dayDiff:  1,
		},
		{
			name:     "one week ago",
			input:    "-1w",
			wantErr:  false,
			checkDay: true,
			dayDiff:  -7,
		},
		{
			name:     "invalid format",
			input:    "invalid",
			wantErr:  true,
		},
		{
			name:     "invalid absolute date",
			input:    "2025-13-01",
			wantErr:  true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseDate(tt.input)
			
			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseDate(%q) expected error, got nil", tt.input)
				}
				return
			}
			
			if err != nil {
				t.Errorf("ParseDate(%q) unexpected error: %v", tt.input, err)
				return
			}
			
			if tt.checkDay {
				expected := now.AddDate(0, 0, tt.dayDiff)
				if result.Year() != expected.Year() || result.Month() != expected.Month() || result.Day() != expected.Day() {
					t.Errorf("ParseDate(%q) = %v, want date %v", tt.input, result.Format("2006-01-02"), expected.Format("2006-01-02"))
				}
				
				// Check that time components are close to current time (within 1 second)
				if abs(result.Hour()-now.Hour()) > 1 {
					t.Errorf("ParseDate(%q) time should preserve current hour, got %d, want ~%d", tt.input, result.Hour(), now.Hour())
				}
			}
			
			if tt.input == "2025-12-01" {
				if result.Year() != 2025 || result.Month() != 12 || result.Day() != 1 {
					t.Errorf("ParseDate(%q) = %v, want 2025-12-01", tt.input, result.Format("2006-01-02"))
				}
				// Check that time components match current time
				if abs(result.Hour()-now.Hour()) > 1 {
					t.Errorf("ParseDate(%q) should preserve current time, got %d, want ~%d", tt.input, result.Hour(), now.Hour())
				}
			}
		})
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}