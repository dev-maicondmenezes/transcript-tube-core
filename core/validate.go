package core

import (
	"fmt"
	"regexp"
)

var videoIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{11}$`)

// ValidateVideoID reports whether videoID is a syntactically valid YouTube
// video identifier: exactly 11 characters from [A-Za-z0-9_-].
func ValidateVideoID(videoID string) error {
	if !videoIDPattern.MatchString(videoID) {
		return fmt.Errorf("validate: video id %q: %w", videoID, ErrInvalidVideoID)
	}
	return nil
}
