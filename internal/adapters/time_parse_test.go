package adapters

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestParseTimeFlexible(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected time.Time
	}{
		{
			name:     "RFC3339",
			input:    "2025-06-15T10:30:00Z",
			expected: time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC),
		},
		{
			name:     "RFC3339 with offset",
			input:    "2025-06-15T12:30:00+02:00",
			expected: time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC),
		},
		{
			name:     "RFC3339Nano",
			input:    "2025-06-15T10:30:00.123456789Z",
			expected: time.Date(2025, 6, 15, 10, 30, 0, 123456789, time.UTC),
		},
		{
			name:     "datetime without timezone",
			input:    "2025-06-15 10:30:00",
			expected: time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC),
		},
		{
			name:     "empty string",
			input:    "",
			expected: time.Time{},
		},
		{
			name:     "whitespace only",
			input:    "   ",
			expected: time.Time{},
		},
		{
			name:     "unparseable returns zero",
			input:    "not-a-date",
			expected: time.Time{},
		},
		{
			name:     "leading/trailing whitespace stripped",
			input:    "  2025-06-15T10:30:00Z  ",
			expected: time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseTimeFlexible(tt.input)
			assert.Equal(t, tt.expected, got)
		})
	}
}
