package core

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/dev-maicondmenezes/transcript-tube-core/core/metadata"
)

type fakeMetadataProvider struct {
	video   *metadata.Video
	err     error
	started chan<- string
	release <-chan struct{}
}

func (f fakeMetadataProvider) Fetch(ctx context.Context, _ string) (*metadata.Video, error) {
	if f.started != nil {
		f.started <- "metadata"
	}
	if f.release != nil {
		select {
		case <-f.release:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return f.video, f.err
}

type fakeTranscriptProvider struct {
	segments []TranscriptSegment
	err      error
	started  chan<- string
	release  <-chan struct{}
}

func (f fakeTranscriptProvider) Fetch(ctx context.Context, _, _ string) ([]TranscriptSegment, error) {
	if f.started != nil {
		f.started <- "transcript"
	}
	if f.release != nil {
		select {
		case <-f.release:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return f.segments, f.err
}

type fakeChapterProvider struct {
	chapters []Chapter
	err      error
}

func (f fakeChapterProvider) Fetch(ctx context.Context, _ string) ([]Chapter, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return f.chapters, f.err
}

func testMetadata() *metadata.Video {
	return &metadata.Video{
		ID:        "dQw4w9WgXcQ",
		Title:     "Test Video",
		Channel:   "Test Channel",
		Duration:  "03:33",
		ViewCount: 10,
		LikeCount: 2,
	}
}

func testSegments() []TranscriptSegment {
	return []TranscriptSegment{
		{OffsetMs: 0, Text: "Hello"},
		{OffsetMs: 1500, Text: "world"},
	}
}

func TestExtractorRunsProvidersInParallel(t *testing.T) {
	t.Parallel()

	started := make(chan string, 2)
	release := make(chan struct{})
	var releaseOnce sync.Once
	releaseAll := func() { releaseOnce.Do(func() { close(release) }) }
	defer releaseAll()

	extractor := NewExtractor(
		fakeMetadataProvider{video: testMetadata(), started: started, release: release},
		fakeTranscriptProvider{segments: testSegments(), started: started, release: release},
	)

	type outcome struct {
		data *VideoData
		err  error
	}
	results := make(chan outcome, 1)
	go func() {
		data, err := extractor.Extract(context.Background(), "dQw4w9WgXcQ", Options{})
		results <- outcome{data: data, err: err}
	}()

	seen := make(map[string]bool, 2)
	for len(seen) < 2 {
		select {
		case provider := <-started:
			seen[provider] = true
		case <-time.After(2 * time.Second):
			t.Fatalf("providers did not start concurrently: seen %v", seen)
		}
	}
	releaseAll()

	select {
	case result := <-results:
		if result.err != nil {
			t.Fatalf("Extract() error = %v", result.err)
		}
		if result.data == nil {
			t.Fatal("Extract() = nil data")
		}
		if result.data.ID != "dQw4w9WgXcQ" {
			t.Fatalf("ID = %q, want dQw4w9WgXcQ", result.data.ID)
		}
		if result.data.Title != "Test Video" {
			t.Fatalf("Title = %q, want Test Video", result.data.Title)
		}
		if len(result.data.Transcript) != 2 {
			t.Fatalf("Transcript length = %d, want 2", len(result.data.Transcript))
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Extract() did not return after providers were released")
	}
}

func TestExtractorParsesChaptersFromDescription(t *testing.T) {
	t.Parallel()

	video := testMetadata()
	video.Description = "0:00 Intro\n1:00 Middle\n2:00 End"

	extractor := NewExtractor(
		fakeMetadataProvider{video: video},
		fakeTranscriptProvider{segments: testSegments()},
	)

	data, err := extractor.Extract(context.Background(), "dQw4w9WgXcQ", Options{})
	if err != nil {
		t.Fatalf("Extract() error = %v", err)
	}
	if len(data.Chapters) != 3 {
		t.Fatalf("Chapters length = %d, want 3: %#v", len(data.Chapters), data.Chapters)
	}
	want := []Chapter{
		{OffsetMs: 0, Title: "Intro"},
		{OffsetMs: 60000, Title: "Middle"},
		{OffsetMs: 120000, Title: "End"},
	}
	if !reflect.DeepEqual(data.Chapters, want) {
		t.Fatalf("Chapters = %#v, want %#v", data.Chapters, want)
	}
}

func TestExtractorPrefersProviderChapters(t *testing.T) {
	t.Parallel()

	video := testMetadata()
	video.Description = "0:00 Description Intro\n1:00 Description Middle"

	extractor := NewExtractorWithChapters(
		fakeMetadataProvider{video: video},
		fakeTranscriptProvider{segments: testSegments()},
		fakeChapterProvider{chapters: []Chapter{
			{OffsetMs: 0, Title: "Provider Intro"},
			{OffsetMs: 30000, Title: "Provider Topic"},
		}},
	)

	data, err := extractor.Extract(context.Background(), "dQw4w9WgXcQ", Options{})
	if err != nil {
		t.Fatalf("Extract() error = %v", err)
	}

	want := []Chapter{
		{OffsetMs: 0, Title: "Provider Intro"},
		{OffsetMs: 30000, Title: "Provider Topic"},
	}
	if !reflect.DeepEqual(data.Chapters, want) {
		t.Fatalf("Chapters = %#v, want %#v", data.Chapters, want)
	}
}

func TestExtractorFallsBackToDescriptionWhenProviderFails(t *testing.T) {
	t.Parallel()

	video := testMetadata()
	video.Description = "0:00 Description Intro\n1:00 Description Middle"

	extractor := NewExtractorWithChapters(
		fakeMetadataProvider{video: video},
		fakeTranscriptProvider{segments: testSegments()},
		fakeChapterProvider{err: errors.New("innertube down")},
	)

	data, err := extractor.Extract(context.Background(), "dQw4w9WgXcQ", Options{})
	if err != nil {
		t.Fatalf("Extract() error = %v, want nil (chapter failure must not fail extraction)", err)
	}

	want := []Chapter{
		{OffsetMs: 0, Title: "Description Intro"},
		{OffsetMs: 60000, Title: "Description Middle"},
	}
	if !reflect.DeepEqual(data.Chapters, want) {
		t.Fatalf("Chapters = %#v, want %#v", data.Chapters, want)
	}
}

func TestExtractorFallsBackToDescriptionWhenProviderReturnsNothing(t *testing.T) {
	t.Parallel()

	video := testMetadata()
	video.Description = "0:00 Description Intro\n1:00 Description Middle"

	extractor := NewExtractorWithChapters(
		fakeMetadataProvider{video: video},
		fakeTranscriptProvider{segments: testSegments()},
		fakeChapterProvider{},
	)

	data, err := extractor.Extract(context.Background(), "dQw4w9WgXcQ", Options{})
	if err != nil {
		t.Fatalf("Extract() error = %v", err)
	}

	want := []Chapter{
		{OffsetMs: 0, Title: "Description Intro"},
		{OffsetMs: 60000, Title: "Description Middle"},
	}
	if !reflect.DeepEqual(data.Chapters, want) {
		t.Fatalf("Chapters = %#v, want %#v", data.Chapters, want)
	}
}

func TestExtractorNormalizesProviderChapters(t *testing.T) {
	t.Parallel()

	extractor := NewExtractorWithChapters(
		fakeMetadataProvider{video: testMetadata()},
		fakeTranscriptProvider{segments: testSegments()},
		fakeChapterProvider{chapters: []Chapter{
			{OffsetMs: 60000, Title: "  Second   topic "},
			{OffsetMs: 60000, Title: "Duplicate"},
			{OffsetMs: -1, Title: "Invalid"},
			{OffsetMs: 0, Title: "First"},
		}},
	)

	data, err := extractor.Extract(context.Background(), "dQw4w9WgXcQ", Options{})
	if err != nil {
		t.Fatalf("Extract() error = %v", err)
	}

	want := []Chapter{
		{OffsetMs: 0, Title: "First"},
		{OffsetMs: 60000, Title: "Second topic"},
	}
	if !reflect.DeepEqual(data.Chapters, want) {
		t.Fatalf("Chapters = %#v, want %#v", data.Chapters, want)
	}
}

func TestExtractorMetadataFailure(t *testing.T) {
	t.Parallel()

	extractor := NewExtractor(
		fakeMetadataProvider{err: errors.New("network down")},
		fakeTranscriptProvider{segments: testSegments()},
	)

	data, err := extractor.Extract(context.Background(), "dQw4w9WgXcQ", Options{})
	if err == nil {
		t.Fatal("Extract() error = nil, want error")
	}
	if data != nil {
		t.Fatalf("Extract() data = %#v, want nil", data)
	}
	if !errors.Is(err, ErrMetadataUnavailable) {
		t.Fatalf("Extract() error = %v, want ErrMetadataUnavailable", err)
	}
}

func TestExtractorVideoNotFound(t *testing.T) {
	t.Parallel()

	extractor := NewExtractor(
		fakeMetadataProvider{err: metadata.ErrNotFound},
		fakeTranscriptProvider{segments: testSegments()},
	)

	_, err := extractor.Extract(context.Background(), "dQw4w9WgXcQ", Options{})
	if !errors.Is(err, ErrVideoNotFound) {
		t.Fatalf("Extract() error = %v, want ErrVideoNotFound", err)
	}
}

func TestExtractorTranscriptFailure(t *testing.T) {
	t.Parallel()

	extractor := NewExtractor(
		fakeMetadataProvider{video: testMetadata()},
		fakeTranscriptProvider{err: errors.New("no captions")},
	)

	data, err := extractor.Extract(context.Background(), "dQw4w9WgXcQ", Options{})
	if err == nil {
		t.Fatal("Extract() error = nil, want error")
	}
	if data != nil {
		t.Fatalf("Extract() data = %#v, want nil", data)
	}
	if !errors.Is(err, ErrTranscriptUnavailable) {
		t.Fatalf("Extract() error = %v, want ErrTranscriptUnavailable", err)
	}
}

func TestExtractorAllowPartialTranscript(t *testing.T) {
	t.Parallel()

	extractor := NewExtractor(
		fakeMetadataProvider{video: testMetadata()},
		fakeTranscriptProvider{err: errors.New("no captions")},
	)

	data, err := extractor.Extract(context.Background(), "dQw4w9WgXcQ", Options{AllowPartialTranscript: true})
	if !errors.Is(err, ErrTranscriptUnavailable) {
		t.Fatalf("Extract() error = %v, want ErrTranscriptUnavailable", err)
	}
	if data == nil {
		t.Fatal("Extract() data = nil, want partial data")
	}
	if data.Title != "Test Video" {
		t.Fatalf("Title = %q, want Test Video", data.Title)
	}
	if len(data.Transcript) != 0 {
		t.Fatalf("Transcript length = %d, want 0", len(data.Transcript))
	}

	markdown := ToMarkdown(data)
	if !strings.Contains(markdown, "_Transcrição indisponível._") {
		t.Fatalf("partial Markdown does not mark the transcript as unavailable:\n%s", markdown)
	}
}

func TestExtractorBothProvidersFail(t *testing.T) {
	t.Parallel()

	extractor := NewExtractor(
		fakeMetadataProvider{err: errors.New("metadata down")},
		fakeTranscriptProvider{err: errors.New("transcript down")},
	)

	_, err := extractor.Extract(context.Background(), "dQw4w9WgXcQ", Options{})
	if !errors.Is(err, ErrMetadataUnavailable) {
		t.Fatalf("Extract() error = %v, want ErrMetadataUnavailable", err)
	}
	if !errors.Is(err, ErrTranscriptUnavailable) {
		t.Fatalf("Extract() error = %v, want ErrTranscriptUnavailable", err)
	}
}

func TestExtractorEmptyTranscriptIsUnavailable(t *testing.T) {
	t.Parallel()

	extractor := NewExtractor(
		fakeMetadataProvider{video: testMetadata()},
		fakeTranscriptProvider{},
	)

	_, err := extractor.Extract(context.Background(), "dQw4w9WgXcQ", Options{})
	if !errors.Is(err, ErrTranscriptUnavailable) {
		t.Fatalf("Extract() error = %v, want ErrTranscriptUnavailable", err)
	}
}

func TestExtractorInvalidVideoID(t *testing.T) {
	t.Parallel()

	extractor := NewExtractor(fakeMetadataProvider{}, fakeTranscriptProvider{})
	_, err := extractor.Extract(context.Background(), "short", Options{})
	if !errors.Is(err, ErrInvalidVideoID) {
		t.Fatalf("Extract() error = %v, want ErrInvalidVideoID", err)
	}
}

func TestExtractWithOptionsMissingAPIKey(t *testing.T) {
	t.Setenv("YOUTUBE_API_KEY", "")

	_, err := ExtractWithOptions(context.Background(), "dQw4w9WgXcQ", Options{})
	if !errors.Is(err, ErrMissingAPIKey) {
		t.Fatalf("ExtractWithOptions() error = %v, want ErrMissingAPIKey", err)
	}
}
