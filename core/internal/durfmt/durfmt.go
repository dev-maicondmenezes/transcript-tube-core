// Package durfmt contains the shared duration parsing and formatting helpers
// used by core and by its metadata adapter.
package durfmt

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// isoDurationPattern accepts the subset of ISO 8601 durations returned by the
// YouTube Data API, for example PT12M34S, PT1H2M3S, PT45S and P1DT2H.
var isoDurationPattern = regexp.MustCompile(`^P(?:(\d+)D)?(?:T(?:(\d+)H)?(?:(\d+)M)?(?:(\d+(?:\.\d+)?)S)?)?$`)

// ParseISO parses an ISO 8601 duration such as "PT12M34S" into a
// time.Duration. An empty or malformed value returns an error.
func ParseISO(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	m := isoDurationPattern.FindStringSubmatch(s)
	if m == nil {
		return 0, fmt.Errorf("durfmt: invalid ISO 8601 duration %q", s)
	}

	hasAny := false
	var d time.Duration

	if m[1] != "" {
		days, err := strconv.ParseInt(m[1], 10, 64)
		if err != nil {
			return 0, fmt.Errorf("durfmt: invalid days in %q: %w", s, err)
		}
		d += time.Duration(days) * 24 * time.Hour
		hasAny = true
	}
	if m[2] != "" {
		hours, err := strconv.ParseInt(m[2], 10, 64)
		if err != nil {
			return 0, fmt.Errorf("durfmt: invalid hours in %q: %w", s, err)
		}
		d += time.Duration(hours) * time.Hour
		hasAny = true
	}
	if m[3] != "" {
		minutes, err := strconv.ParseInt(m[3], 10, 64)
		if err != nil {
			return 0, fmt.Errorf("durfmt: invalid minutes in %q: %w", s, err)
		}
		d += time.Duration(minutes) * time.Minute
		hasAny = true
	}
	if m[4] != "" {
		seconds, err := strconv.ParseFloat(m[4], 64)
		if err != nil {
			return 0, fmt.Errorf("durfmt: invalid seconds in %q: %w", s, err)
		}
		d += time.Duration(seconds * float64(time.Second))
		hasAny = true
	}

	if !hasAny {
		return 0, fmt.Errorf("durfmt: invalid ISO 8601 duration %q", s)
	}
	return d, nil
}

// Format formats a duration as "mm:ss", or "hh:mm:ss" when it is at least one
// hour long. Negative durations are normalised to "00:00".
func Format(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	totalSeconds := int64(d / time.Second)
	hours := totalSeconds / 3600
	minutes := (totalSeconds % 3600) / 60
	seconds := totalSeconds % 60
	if hours > 0 {
		return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
	}
	return fmt.Sprintf("%02d:%02d", minutes, seconds)
}

// Timestamp formats a millisecond offset as "mm:ss" or "hh:mm:ss". Negative
// offsets are normalised to "00:00".
func Timestamp(ms int64) string {
	if ms < 0 {
		ms = 0
	}
	return Format(time.Duration(ms) * time.Millisecond)
}
