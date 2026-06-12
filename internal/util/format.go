package util

import (
	"fmt"
	"math"
)

// FormatBytes converts bytes to human-readable string (e.g., "1.5 KB").
func FormatBytes(bytes int64) string {
	if bytes == 0 {
		return "0 B"
	}
	units := []string{"B", "KB", "MB", "GB"}
	k := float64(1024)
	i := int(math.Floor(math.Log(float64(bytes)) / math.Log(k)))
	if i >= len(units) {
		i = len(units) - 1
	}
	value := float64(bytes) / math.Pow(k, float64(i))
	if i > 0 {
		return fmt.Sprintf("%.1f %s", value, units[i])
	}
	return fmt.Sprintf("%.0f %s", value, units[i])
}

// Truncate shortens text to length runes, appending "..." when trimmed.
func Truncate(text string, length int) string {
	if text == "" {
		return ""
	}
	if len(text) <= length {
		return text
	}
	if length <= 3 {
		return text[:length]
	}
	return text[:length-3] + "..."
}
