package metadata

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestClientFetch(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/youtube/v3/videos" {
			t.Errorf("path = %q, want /youtube/v3/videos", r.URL.Path)
		}
		if got := r.URL.Query().Get("id"); got != "dQw4w9WgXcQ" {
			t.Errorf("id = %q, want dQw4w9WgXcQ", got)
		}
		if got := strings.Join(r.URL.Query()["part"], ","); got != "snippet,statistics,contentDetails" {
			t.Errorf("part = %q, want snippet,statistics,contentDetails", got)
		}
		if got := r.URL.Query().Get("key"); got != "test-key" {
			t.Errorf("key = %q, want test-key", got)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"items": [{
				"snippet": {
					"title": "Test Video",
					"channelTitle": "Test Channel",
					"description": "A description",
					"publishedAt": "2020-01-02T03:04:05Z"
				},
				"statistics": {"viewCount": "123456", "likeCount": "789"},
				"contentDetails": {"duration": "PT1H2M3S"}
			}]
		}`)
	}))
	defer server.Close()

	client, err := New("test-key", WithBaseURL(server.URL))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	video, err := client.Fetch(context.Background(), "dQw4w9WgXcQ")
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	if video.ID != "dQw4w9WgXcQ" {
		t.Errorf("ID = %q", video.ID)
	}
	if video.Title != "Test Video" {
		t.Errorf("Title = %q", video.Title)
	}
	if video.Channel != "Test Channel" {
		t.Errorf("Channel = %q", video.Channel)
	}
	if video.Description != "A description" {
		t.Errorf("Description = %q", video.Description)
	}
	if want := time.Date(2020, time.January, 2, 3, 4, 5, 0, time.UTC); !video.PublishedAt.Equal(want) {
		t.Errorf("PublishedAt = %v, want %v", video.PublishedAt, want)
	}
	if video.Duration != "01:02:03" {
		t.Errorf("Duration = %q, want 01:02:03", video.Duration)
	}
	if video.ViewCount != 123456 {
		t.Errorf("ViewCount = %d, want 123456", video.ViewCount)
	}
	if video.LikeCount != 789 {
		t.Errorf("LikeCount = %d, want 789", video.LikeCount)
	}
}

func TestClientFetchMissingOptionalFields(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"items": [{"snippet": {"title": "Only Title"}}]}`)
	}))
	defer server.Close()

	client, err := New("test-key", WithBaseURL(server.URL))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	video, err := client.Fetch(context.Background(), "dQw4w9WgXcQ")
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if video.ViewCount != 0 || video.LikeCount != 0 {
		t.Fatalf("counts = (%d, %d), want zeroes", video.ViewCount, video.LikeCount)
	}
	if video.Duration != "" {
		t.Fatalf("Duration = %q, want empty", video.Duration)
	}
}

func TestClientFetchEmptyItems(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"items": []}`)
	}))
	defer server.Close()

	client, err := New("test-key", WithBaseURL(server.URL))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	_, err = client.Fetch(context.Background(), "dQw4w9WgXcQ")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Fetch() error = %v, want ErrNotFound", err)
	}
}

func TestClientFetchHTTP404(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"error": {"code": 404, "message": "video not found"}}`)
	}))
	defer server.Close()

	client, err := New("test-key", WithBaseURL(server.URL))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	_, err = client.Fetch(context.Background(), "dQw4w9WgXcQ")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Fetch() error = %v, want ErrNotFound", err)
	}
}

func TestNewRequiresAPIKey(t *testing.T) {
	t.Parallel()

	if _, err := New(""); err == nil {
		t.Fatal("New(\"\") error = nil, want error")
	}
}
