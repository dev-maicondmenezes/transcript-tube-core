package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func basicVideoData() *VideoData {
	return &VideoData{
		ID:          "dQw4w9WgXcQ",
		Title:       "Never Gonna Give You Up",
		Channel:     "Rick Astley",
		Description: "The official video for \"Never Gonna Give You Up\".\n\nDirected by Simon West.",
		PublishedAt: time.Date(2009, time.October, 25, 7, 0, 0, 0, time.UTC),
		Duration:    "03:33",
		ViewCount:   1000000,
		LikeCount:   12345,
		Transcript: []TranscriptSegment{
			{OffsetMs: 0, Text: "We're no strangers to love"},
			{OffsetMs: 59000, Text: "You know the rules and so do I"},
		},
	}
}

func TestToMarkdownGoldenBasic(t *testing.T) {
	t.Parallel()

	got := ToMarkdown(basicVideoData())
	want := readGolden(t, "expected_basic.md")
	if got != want {
		t.Fatalf("ToMarkdown output differs from golden file.\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestToMarkdownGoldenChapters(t *testing.T) {
	t.Parallel()

	data := &VideoData{
		ID:          "abcdefghijk",
		Title:       "Chapters Video",
		Channel:     "Channel",
		Description: "Chapters:\n0:00 Intro\n1:23 - Go\n2:05 Testes",
		PublishedAt: time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC),
		Duration:    "02:30",
		ViewCount:   10,
		LikeCount:   1,
		Chapters: []Chapter{
			{OffsetMs: 0, Title: "Introdução"},
			{OffsetMs: 83000, Title: "Go"},
			{OffsetMs: 125000, Title: "Testes"},
		},
		Transcript: []TranscriptSegment{
			{OffsetMs: 0, Text: "Bem-vindos"},
			{OffsetMs: 5000, Text: "ao canal"},
			{OffsetMs: 83000, Text: "Agora falamos de Go"},
			{OffsetMs: 125000, Text: "e de testes"},
		},
	}

	got := ToMarkdown(data)
	want := readGolden(t, "expected_chapters.md")
	if got != want {
		t.Fatalf("ToMarkdown output differs from golden file.\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestToMarkdownAnchorsChapterToNextSegment(t *testing.T) {
	t.Parallel()

	data := &VideoData{
		ID: "abcdefghijk",
		Chapters: []Chapter{
			{OffsetMs: 10000, Title: "Middle"},
		},
		Transcript: []TranscriptSegment{
			{OffsetMs: 5000, Text: "before"},
			{OffsetMs: 15000, Text: "after"},
		},
	}

	got := ToMarkdown(data)
	heading := strings.Index(got, "### [00:10] Middle")
	before := strings.Index(got, "[00:05] before")
	after := strings.Index(got, "[00:15] after")
	if heading < 0 || before < 0 || after < 0 {
		t.Fatalf("missing expected content:\n%s", got)
	}
	if !(before < heading && heading < after) {
		t.Fatalf("chapter heading is not anchored to the next segment:\n%s", got)
	}
}

func TestToMarkdownIgnoresChaptersWithoutTranscript(t *testing.T) {
	t.Parallel()

	data := &VideoData{
		ID:       "abcdefghijk",
		Chapters: []Chapter{{OffsetMs: 0, Title: "Intro"}},
	}

	got := ToMarkdown(data)
	if strings.Contains(got, "### ") {
		t.Fatalf("chapter heading should not be emitted without a transcript:\n%s", got)
	}
	if !strings.Contains(got, "_Transcrição indisponível._") {
		t.Fatalf("missing unavailable marker:\n%s", got)
	}
}

func TestToMarkdownGoldenLong(t *testing.T) {
	t.Parallel()

	data := &VideoData{
		ID:          "abcdefghijk",
		Title:       "Long Video",
		Channel:     "Channel",
		Description: "A video longer than one hour.",
		PublishedAt: time.Date(2024, time.March, 1, 12, 30, 0, 0, time.UTC),
		Duration:    "01:02:03",
		ViewCount:   42,
		LikeCount:   7,
		Transcript: []TranscriptSegment{
			{OffsetMs: 0, Text: "Start"},
			{OffsetMs: 3600000, Text: "One hour in"},
			{OffsetMs: 3723000, Text: "End"},
		},
	}

	got := ToMarkdown(data)
	want := readGolden(t, "expected_long.md")
	if got != want {
		t.Fatalf("ToMarkdown output differs from golden file.\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestToMarkdownNil(t *testing.T) {
	t.Parallel()

	if got := ToMarkdown(nil); got != "" {
		t.Fatalf("ToMarkdown(nil) = %q, want empty string", got)
	}
}

func TestToMarkdownEmptyTranscriptIsPartial(t *testing.T) {
	t.Parallel()

	data := basicVideoData()
	data.Transcript = nil

	got := ToMarkdown(data)
	if want := "## Transcrição\n\n_Transcrição indisponível._\n"; !strings.Contains(got, want) {
		t.Fatalf("ToMarkdown() does not contain %q:\n%s", want, got)
	}
}

func TestToMarkdownUnknownValues(t *testing.T) {
	t.Parallel()

	data := &VideoData{ID: "dQw4w9WgXcQ", PublishedAt: time.Time{}, Duration: ""}

	got := ToMarkdown(data)
	for _, want := range []string{
		"**Publicado em:** desconhecida",
		"**Duração:** desconhecida",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("ToMarkdown() does not contain %q:\n%s", want, got)
		}
	}
}

func readGolden(t *testing.T, name string) string {
	t.Helper()

	content, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("reading golden file %s: %v", name, err)
	}
	return string(content)
}
