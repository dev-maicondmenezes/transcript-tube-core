// Package metadata fetches video metadata from the YouTube Data API v3.
package metadata

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/dev-maicondmenezes/transcript-tube-core/core/internal/durfmt"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"
)

// ErrNotFound indicates that the requested video does not exist.
var ErrNotFound = errors.New("metadata: video not found")

// Client is a YouTube Data API v3 metadata client.
type Client struct {
	service    *youtube.Service
	httpClient *http.Client
	baseURL    string
}

// Option customises a Client. Options exist mainly for tests.
type Option func(*Client)

// WithHTTPClient injects an HTTP client, typically one backed by httptest.
func WithHTTPClient(client *http.Client) Option {
	return func(c *Client) {
		c.httpClient = client
	}
}

// WithBaseURL overrides the YouTube API endpoint, typically with an
// httptest.Server URL.
func WithBaseURL(baseURL string) Option {
	return func(c *Client) {
		c.baseURL = baseURL
	}
}

// New creates a metadata client authenticated with apiKey.
func New(apiKey string, opts ...Option) (*Client, error) {
	if strings.TrimSpace(apiKey) == "" {
		return nil, errors.New("metadata: API key is required")
	}

	client := &Client{}
	for _, opt := range opts {
		opt(client)
	}

	clientOptions := []option.ClientOption{option.WithAPIKey(apiKey)}
	if client.httpClient != nil {
		clientOptions = append(clientOptions, option.WithHTTPClient(client.httpClient))
	}

	service, err := youtube.NewService(context.Background(), clientOptions...)
	if err != nil {
		return nil, fmt.Errorf("metadata: creating YouTube client: %w", err)
	}
	if client.baseURL != "" {
		// The generated service resolves methods relative to BasePath, so an
		// httptest server receives requests at /youtube/v3/videos.
		service.BasePath = strings.TrimRight(client.baseURL, "/") + "/"
	}
	client.service = service

	return client, nil
}

// Fetch retrieves metadata for videoID with part=snippet,statistics,contentDetails.
func (c *Client) Fetch(ctx context.Context, videoID string) (*Video, error) {
	response, err := c.service.Videos.
		List([]string{"snippet", "statistics", "contentDetails"}).
		Id(videoID).
		Context(ctx).
		Do()
	if err != nil {
		return nil, mapAPIError(videoID, err)
	}
	if len(response.Items) == 0 {
		return nil, fmt.Errorf("metadata: video %q: %w", videoID, ErrNotFound)
	}

	item := response.Items[0]
	video := &Video{ID: videoID}

	if snippet := item.Snippet; snippet != nil {
		video.Title = snippet.Title
		video.Channel = snippet.ChannelTitle
		video.Description = snippet.Description
		if publishedAt, err := time.Parse(time.RFC3339, snippet.PublishedAt); err == nil {
			video.PublishedAt = publishedAt
		}
	}

	if statistics := item.Statistics; statistics != nil {
		video.ViewCount = statistics.ViewCount
		video.LikeCount = statistics.LikeCount
	}

	if details := item.ContentDetails; details != nil {
		if duration, err := durfmt.ParseISO(details.Duration); err == nil {
			video.Duration = durfmt.Format(duration)
		}
	}

	return video, nil
}

func mapAPIError(videoID string, err error) error {
	var apiError *googleapi.Error
	if errors.As(err, &apiError) && apiError.Code == http.StatusNotFound {
		return fmt.Errorf("metadata: video %q: %w: %v", videoID, ErrNotFound, err)
	}
	return fmt.Errorf("metadata: videos.list %q: %w", videoID, err)
}
