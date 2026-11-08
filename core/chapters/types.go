// Package chapters fetches video chapter markers from YouTube's InnerTube API.
//
// The InnerTube "next" endpoint exposes the same chapter list rendered by the
// YouTube player, including chapters generated automatically by YouTube that
// are not present in the video description.
package chapters

import (
	"context"
	"errors"
)

// Chapter marks the beginning of a video section. OffsetMs is the distance from
// the beginning of the video in milliseconds.
type Chapter struct {
	OffsetMs int64
	Title    string
}

// Fetcher retrieves chapter markers for a video.
type Fetcher interface {
	Fetch(ctx context.Context, videoID string) ([]Chapter, error)
}

// Sentinel errors of the chapters adapter. The orchestration layer treats any
// error as "no chapters" and falls back to the video description.
var (
	ErrUnavailable = errors.New("chapters unavailable")
	ErrNoChapters  = errors.New("no chapter markers")
)
