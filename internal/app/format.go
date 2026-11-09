package app

import (
	"fmt"
	"strings"
	"time"
	"unicode"
)

// formatClock writes a position as a player does: 4:05, or 1:04:05.
func formatClock(value time.Duration) string {
	if value < 0 {
		value = 0
	}
	total := int(value.Round(time.Second) / time.Second)
	hours := total / 3600
	minutes := (total % 3600) / 60
	seconds := total % 60
	if hours > 0 {
		return fmt.Sprintf("%d:%02d:%02d", hours, minutes, seconds)
	}
	return fmt.Sprintf("%d:%02d", minutes, seconds)
}

// formatDuration writes a length as people say it: 6h 12m, 45m, 30s.
func formatDuration(value time.Duration) string {
	if value <= 0 {
		return "0m"
	}
	minutes := int(value / time.Minute)
	if minutes < 1 {
		return fmt.Sprintf("%ds", int(value.Round(time.Second)/time.Second))
	}
	hours := minutes / 60
	minutes %= 60
	if hours > 0 && minutes > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh", hours)
	}
	return fmt.Sprintf("%dm", minutes)
}

// initials makes a cover placeholder from a title.
func initials(title string) string {
	var result []rune
	for _, word := range strings.Fields(title) {
		for _, r := range word {
			if unicode.IsLetter(r) || unicode.IsDigit(r) {
				result = append(result, unicode.ToUpper(r))
				break
			}
		}
		if len(result) == 2 {
			break
		}
	}
	if len(result) == 0 {
		return "?"
	}
	return string(result)
}
