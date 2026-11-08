package transcript

import (
	"context"

	yt_transcript "github.com/horiagug/youtube-transcript-api-go/pkg/yt_transcript"
	"github.com/horiagug/youtube-transcript-api-go/pkg/yt_transcript_models"
)

const innerTubeTimeoutSeconds = 30

// innerTubeFetcher isolates the third-party library behind the Fetcher
// interface so that library changes never leak into the core package.
type innerTubeFetcher struct {
	client *yt_transcript.YtTranscriptClient
}

func newInnerTubeFetcher() *innerTubeFetcher {
	return &innerTubeFetcher{
		client: yt_transcript.NewClient(yt_transcript.WithTimeout(innerTubeTimeoutSeconds)),
	}
}

func (f *innerTubeFetcher) Fetch(ctx context.Context, videoID, lang string) ([]Segment, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// An empty language slice tells the library to return every available
	// track, which implements the "first available language" fallback.
	var languages []string
	if lang != "" {
		languages = []string{lang}
	}

	type result struct {
		transcripts []yt_transcript_models.Transcript
		err         error
	}

	results := make(chan result, 1)
	go func() {
		transcripts, err := f.client.GetTranscripts(videoID, languages)
		results <- result{transcripts: transcripts, err: err}
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case res := <-results:
		if res.err != nil {
			return nil, res.err
		}
		for _, transcript := range res.transcripts {
			segments := segmentsFromLines(transcript.Lines)
			if len(segments) > 0 {
				return segments, nil
			}
		}
		return nil, ErrNoSegments
	}
}

func segmentsFromLines(lines []yt_transcript_models.TranscriptLine) []Segment {
	segments := make([]Segment, 0, len(lines))
	for _, line := range lines {
		text := normalizeText(line.Text)
		if text == "" {
			continue
		}
		segments = append(segments, Segment{
			OffsetMs: int64(line.Start * 1000),
			Text:     text,
		})
	}
	return segments
}
