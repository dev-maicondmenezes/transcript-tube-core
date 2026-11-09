// Package transcript fetches YouTube transcripts using a cascade of
// strategies: InnerTube first, then the public timedtext endpoint.
package transcript

import (
	"context"
	"errors"
)

// Segment is a single transcript line. OffsetMs is the distance from the start
// of the video in milliseconds.
type Segment struct {
	OffsetMs int64
	Text     string
}

// Fetcher retrieves transcript segments for a specific language.
type Fetcher interface {
	Fetch(ctx context.Context, videoID, lang string) ([]Segment, error)
}

// Sentinel errors of the transcript adapter. The orchestration layer maps them
// to core.ErrTranscriptUnavailable.
var (
	ErrUnavailable = errors.New("transcrição indisponível")
	ErrNoSegments  = errors.New("nenhum segmento de transcrição encontrado")
)
