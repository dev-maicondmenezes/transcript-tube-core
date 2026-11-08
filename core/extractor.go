package core

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/dev-maicondmenezes/transcript-tube-core/core/chapters"
	"github.com/dev-maicondmenezes/transcript-tube-core/core/metadata"
	"github.com/dev-maicondmenezes/transcript-tube-core/core/transcript"
)

// MetadataProvider fetches video metadata. It is implemented by
// *metadata.Client and can be replaced by fakes in tests.
type MetadataProvider interface {
	Fetch(ctx context.Context, videoID string) (*metadata.Video, error)
}

// TranscriptProvider fetches a transcript. It is implemented by the internal
// adapter around *transcript.Client and can be replaced by fakes in tests.
type TranscriptProvider interface {
	Fetch(ctx context.Context, videoID, lang string) ([]TranscriptSegment, error)
}

// ChapterProvider fetches chapter markers. It is optional: when it is nil or
// fails, the extractor falls back to chapters parsed from the description.
type ChapterProvider interface {
	Fetch(ctx context.Context, videoID string) ([]Chapter, error)
}

// Extractor orchestrates metadata, transcript and optional chapter retrieval.
// Metadata and transcript providers are required.
type Extractor struct {
	Metadata   MetadataProvider
	Transcript TranscriptProvider
	Chapters   ChapterProvider
}

// NewExtractor builds an Extractor from explicit providers. It is the seam
// used by tests and by adapters that want to customise the clients.
func NewExtractor(metadataProvider MetadataProvider, transcriptProvider TranscriptProvider) *Extractor {
	return &Extractor{Metadata: metadataProvider, Transcript: transcriptProvider}
}

// NewExtractorWithChapters builds an Extractor with an explicit chapter
// provider. Chapters from the provider take precedence over chapters parsed
// from the description; failures fall back to the description.
func NewExtractorWithChapters(metadataProvider MetadataProvider, transcriptProvider TranscriptProvider, chapterProvider ChapterProvider) *Extractor {
	extractor := NewExtractor(metadataProvider, transcriptProvider)
	extractor.Chapters = chapterProvider
	return extractor
}

// Extract fetches metadata and the transcript for videoID using the default
// clients. See ExtractWithOptions for the full contract.
func Extract(ctx context.Context, videoID string) (*VideoData, error) {
	return ExtractWithOptions(ctx, videoID, Options{})
}

// ExtractWithOptions validates videoID, fetches metadata and transcript in
// parallel and returns the assembled VideoData.
//
// Metadata is mandatory. The transcript is mandatory unless
// opts.AllowPartialTranscript is true; in that case a transcript failure
// returns partially filled VideoData together with an error wrapping
// ErrTranscriptUnavailable.
func ExtractWithOptions(ctx context.Context, videoID string, opts Options) (*VideoData, error) {
	if err := ValidateVideoID(videoID); err != nil {
		return nil, err
	}

	extractor, err := newDefaultExtractor()
	if err != nil {
		return nil, err
	}
	return extractor.Extract(ctx, videoID, opts)
}

// Extract is the injectable orchestration entry point.
func (e *Extractor) Extract(ctx context.Context, videoID string, opts Options) (*VideoData, error) {
	if err := ValidateVideoID(videoID); err != nil {
		return nil, err
	}
	if e == nil || e.Metadata == nil || e.Transcript == nil {
		return nil, errors.New("extração: provedores de metadados e transcrição são obrigatórios")
	}

	var (
		wg               sync.WaitGroup
		video            *metadata.Video
		metadataErr      error
		segments         []TranscriptSegment
		transcriptErr    error
		providerChapters []Chapter
	)

	// The calls run without errgroup.WithContext on purpose: a transcript or
	// chapter failure must not cancel a metadata request that may still
	// succeed when partial results are allowed. Only the caller's context
	// cancels work.
	wg.Add(2)
	go func() {
		defer wg.Done()
		video, metadataErr = e.Metadata.Fetch(ctx, videoID)
	}()
	go func() {
		defer wg.Done()
		segments, transcriptErr = e.Transcript.Fetch(ctx, videoID, opts.Lang)
	}()
	if e.Chapters != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			providerChapters, _ = e.Chapters.Fetch(ctx, videoID)
		}()
	}
	wg.Wait()

	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("extração: %w", err)
	}

	if transcriptErr == nil && len(segments) == 0 {
		transcriptErr = transcript.ErrNoSegments
	}

	if metadataErr != nil {
		metadataErr = wrapMetadataError(videoID, metadataErr)
		if transcriptErr != nil {
			return nil, fmt.Errorf("extração: %w", errors.Join(metadataErr, wrapTranscriptError(transcriptErr)))
		}
		return nil, metadataErr
	}

	data := videoDataFromMetadata(videoID, video, segments)

	// YouTube's own chapter list (including automatic chapters) takes
	// precedence over timestamps parsed from the description. A provider error
	// or an empty list keeps the description fallback already in data.
	if normalized := normalizeChapters(providerChapters); len(normalized) > 0 {
		data.Chapters = normalized
	}

	if transcriptErr != nil {
		transcriptErr = wrapTranscriptError(transcriptErr)
		if opts.AllowPartialTranscript {
			data.Transcript = nil
			return data, transcriptErr
		}
		return nil, transcriptErr
	}

	return data, nil
}

func newDefaultExtractor() (*Extractor, error) {
	apiKey := strings.TrimSpace(os.Getenv("YOUTUBE_API_KEY"))
	if apiKey == "" {
		return nil, fmt.Errorf("configuração: %w", ErrMissingAPIKey)
	}

	metadataClient, err := metadata.New(apiKey)
	if err != nil {
		return nil, fmt.Errorf("metadados: %w", err)
	}

	return NewExtractorWithChapters(
		metadataClient,
		transcriptClientAdapter{client: transcript.New()},
		chapterClientAdapter{client: chapters.New()},
	), nil
}

// transcriptClientAdapter converts transcript.Segment into the core type so
// that core/transcript never needs to import the root package.
type transcriptClientAdapter struct {
	client *transcript.Client
}

func (a transcriptClientAdapter) Fetch(ctx context.Context, videoID, lang string) ([]TranscriptSegment, error) {
	segments, err := a.client.Fetch(ctx, videoID, lang)
	if err != nil {
		return nil, err
	}

	result := make([]TranscriptSegment, 0, len(segments))
	for _, segment := range segments {
		result = append(result, TranscriptSegment{
			OffsetMs: segment.OffsetMs,
			Text:     segment.Text,
		})
	}
	return result, nil
}

// chapterClientAdapter converts chapters.Chapter into the core type so that
// core/chapters never needs to import the root package.
type chapterClientAdapter struct {
	client *chapters.Client
}

func (a chapterClientAdapter) Fetch(ctx context.Context, videoID string) ([]Chapter, error) {
	fetched, err := a.client.Fetch(ctx, videoID)
	if err != nil {
		return nil, err
	}

	result := make([]Chapter, 0, len(fetched))
	for _, chapter := range fetched {
		result = append(result, Chapter{
			OffsetMs: chapter.OffsetMs,
			Title:    chapter.Title,
		})
	}
	return result, nil
}

func videoDataFromMetadata(videoID string, video *metadata.Video, segments []TranscriptSegment) *VideoData {
	data := &VideoData{
		ID:         videoID,
		Transcript: segments,
	}
	if video == nil {
		return data
	}

	data.Title = video.Title
	data.Channel = video.Channel
	data.Description = video.Description
	data.PublishedAt = video.PublishedAt
	data.Duration = video.Duration
	data.ViewCount = video.ViewCount
	data.LikeCount = video.LikeCount
	data.Chapters = parseChapters(video.Description)

	// The YouTube Data API occasionally omits contentDetails.duration. In that
	// case derive an approximate duration from the last transcript offset.
	if data.Duration == "" && len(segments) > 0 {
		data.Duration = formatTimestamp(segments[len(segments)-1].OffsetMs)
	}

	return data
}

func wrapMetadataError(videoID string, err error) error {
	if errors.Is(err, metadata.ErrNotFound) {
		return fmt.Errorf("metadados: vídeo %q: %w", videoID, errors.Join(ErrVideoNotFound, err))
	}
	return fmt.Errorf("metadados: vídeo %q: %w", videoID, errors.Join(ErrMetadataUnavailable, err))
}

func wrapTranscriptError(err error) error {
	return fmt.Errorf("transcrição: %w", errors.Join(ErrTranscriptUnavailable, err))
}
