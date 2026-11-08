package core

import "time"

// VideoData is the complete representation of a YouTube video extracted by the
// core package. It is the public contract consumed by the CLI, the Native
// Messaging host and any other adapter.
type VideoData struct {
	ID          string
	Title       string
	Channel     string
	Description string
	PublishedAt time.Time
	Duration    string // formatted as "mm:ss" or "hh:mm:ss"
	ViewCount   uint64
	LikeCount   uint64
	Transcript  []TranscriptSegment
	Chapters    []Chapter
}

// Chapter marks the beginning of a video section/chapter. OffsetMs is the
// distance from the beginning of the video in milliseconds.
type Chapter struct {
	OffsetMs int64
	Title    string
}

// TranscriptSegment is a single line of a video transcript. OffsetMs is the
// distance from the beginning of the video in milliseconds.
type TranscriptSegment struct {
	OffsetMs int64
	Text     string
}

// Options configures extraction.
//
// Lang is the preferred transcript language code (for example "pt" or "en").
// When empty, the core negotiates a preference order of pt, pt-BR and en and
// finally falls back to the first language made available by the provider.
//
// AllowPartialTranscript makes it possible to receive VideoData with metadata
// even when the transcript cannot be fetched. In that case the returned error
// still wraps ErrTranscriptUnavailable.
type Options struct {
	Lang                   string
	AllowPartialTranscript bool
}
