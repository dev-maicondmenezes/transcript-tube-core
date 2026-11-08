package core

import (
	"time"

	"github.com/dev-maicondmenezes/transcript-tube-core/core/internal/durfmt"
)

// parseISODuration parses an ISO 8601 duration (for example "PT12M34S").
func parseISODuration(s string) (time.Duration, error) {
	return durfmt.ParseISO(s)
}

// formatDuration formats a duration as "mm:ss" or "hh:mm:ss".
func formatDuration(d time.Duration) string {
	return durfmt.Format(d)
}

// formatTimestamp formats a millisecond offset as "mm:ss" or "hh:mm:ss".
func formatTimestamp(ms int64) string {
	return durfmt.Timestamp(ms)
}
