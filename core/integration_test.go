//go:build integration

package core_test

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/dev-maicondmenezes/transcript-tube-core/core"
)

func TestIntegrationExtract(t *testing.T) {
	if os.Getenv("YOUTUBE_API_KEY") == "" {
		t.Skip("YOUTUBE_API_KEY is not set")
	}

	videoID := os.Getenv("TEST_VIDEO_ID")
	if videoID == "" {
		videoID = "dQw4w9WgXcQ"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	data, err := core.ExtractWithOptions(ctx, videoID, core.Options{AllowPartialTranscript: true})
	if err != nil && !errors.Is(err, core.ErrTranscriptUnavailable) {
		t.Fatalf("ExtractWithOptions() error = %v", err)
	}
	if data == nil {
		t.Fatal("ExtractWithOptions() = nil data")
	}
	if data.ID != videoID {
		t.Fatalf("ID = %q, want %q", data.ID, videoID)
	}
	if data.Title == "" {
		t.Fatal("Title is empty")
	}
	if err != nil {
		t.Logf("transcript unavailable, partial extraction succeeded: %v", err)
	}
	if markdown := core.ToMarkdown(data); !strings.HasPrefix(markdown, "# ") {
		t.Fatalf("unexpected Markdown prefix: %q", markdown)
	}
}
