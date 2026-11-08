package chapters

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
)

const (
	defaultBaseURL       = "https://www.youtube.com"
	defaultClientVersion = "2.20260101.00.00"
	defaultUserAgent     = "transcript-tube-core/0.2.0 (+https://github.com/dev-maicondmenezes/transcript-tube-core)"
	maxResponseBody      = 8 << 20 // 8 MiB defensive limit
)

// Client fetches chapter markers from the InnerTube "next" endpoint.
type Client struct {
	httpClient    *http.Client
	baseURL       string
	clientVersion string
}

// Option customises a Client. Options exist mainly for tests.
type Option func(*Client)

// WithHTTPClient injects an HTTP client, typically one backed by httptest.
func WithHTTPClient(client *http.Client) Option {
	return func(c *Client) {
		c.httpClient = client
	}
}

// WithBaseURL overrides the YouTube endpoint, typically with an
// httptest.Server URL.
func WithBaseURL(baseURL string) Option {
	return func(c *Client) {
		c.baseURL = baseURL
	}
}

// WithClientVersion overrides the InnerTube WEB client version.
func WithClientVersion(version string) Option {
	return func(c *Client) {
		c.clientVersion = version
	}
}

// New creates a chapters client using the WEB InnerTube client. The endpoint
// works without an API key.
func New(opts ...Option) *Client {
	client := &Client{
		httpClient:    http.DefaultClient,
		baseURL:       defaultBaseURL,
		clientVersion: defaultClientVersion,
	}
	for _, opt := range opts {
		opt(client)
	}

	if client.httpClient == nil {
		client.httpClient = http.DefaultClient
	}
	if strings.TrimSpace(client.baseURL) == "" {
		client.baseURL = defaultBaseURL
	}
	if strings.TrimSpace(client.clientVersion) == "" {
		client.clientVersion = defaultClientVersion
	}
	return client
}

// Fetch returns the chapter list for videoID. A response without chapters
// returns ErrNoChapters; transport, status and parse failures return
// ErrUnavailable.
func (c *Client) Fetch(ctx context.Context, videoID string) ([]Chapter, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if strings.TrimSpace(videoID) == "" {
		return nil, fmt.Errorf("chapters: empty video id: %w", ErrUnavailable)
	}

	payload, err := json.Marshal(nextRequest{
		Context: nextContext{
			Client: nextClient{
				ClientName:    "WEB",
				ClientVersion: c.clientVersion,
			},
		},
		VideoID: videoID,
	})
	if err != nil {
		return nil, fmt.Errorf("chapters: encoding request: %w", err)
	}

	endpoint := strings.TrimRight(c.baseURL, "/") + "/youtubei/v1/next?prettyPrint=false"
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("chapters: creating request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("User-Agent", defaultUserAgent)

	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("chapters: request failed: %w: %w", ErrUnavailable, err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("chapters: unexpected status %d: %w", response.StatusCode, ErrUnavailable)
	}

	body, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBody))
	if err != nil {
		return nil, fmt.Errorf("chapters: reading body: %w: %w", ErrUnavailable, err)
	}

	chapters, err := parseNextResponse(body)
	if err != nil {
		return nil, err
	}
	if len(chapters) == 0 {
		return nil, ErrNoChapters
	}
	return chapters, nil
}

type nextRequest struct {
	Context nextContext `json:"context"`
	VideoID string      `json:"videoId"`
}

type nextContext struct {
	Client nextClient `json:"client"`
}

type nextClient struct {
	ClientName    string `json:"clientName"`
	ClientVersion string `json:"clientVersion"`
}

// parseNextResponse walks the InnerTube response looking for chapterRenderer
// entries. Walking recursively keeps the parser resilient to the exact
// playerOverlays nesting, which changes between YouTube releases.
func parseNextResponse(body []byte) ([]Chapter, error) {
	var document any
	if err := json.Unmarshal(body, &document); err != nil {
		return nil, fmt.Errorf("chapters: parsing response: %w: %w", ErrUnavailable, err)
	}

	var collected []Chapter
	collectChapters(document, &collected)
	return normalize(collected), nil
}

func collectChapters(node any, collected *[]Chapter) {
	switch value := node.(type) {
	case map[string]any:
		if raw, ok := value["chapterRenderer"]; ok {
			if renderer, ok := raw.(map[string]any); ok {
				if chapter, ok := chapterFromRenderer(renderer); ok {
					*collected = append(*collected, chapter)
				}
			}
		}
		for _, child := range value {
			collectChapters(child, collected)
		}
	case []any:
		for _, child := range value {
			collectChapters(child, collected)
		}
	}
}

func chapterFromRenderer(renderer map[string]any) (Chapter, bool) {
	offset, ok := milliseconds(renderer["timeRangeStartMillis"])
	if !ok || offset < 0 {
		return Chapter{}, false
	}

	title := titleText(renderer["title"])
	if title == "" {
		return Chapter{}, false
	}
	return Chapter{OffsetMs: offset, Title: title}, true
}

// titleText supports both representations used by InnerTube: simpleText and
// runs (a list of text fragments).
func titleText(node any) string {
	switch value := node.(type) {
	case map[string]any:
		if simple, ok := value["simpleText"].(string); ok {
			return normalizeText(simple)
		}
		if runs, ok := value["runs"].([]any); ok {
			var builder strings.Builder
			for _, rawRun := range runs {
				run, ok := rawRun.(map[string]any)
				if !ok {
					continue
				}
				text, ok := run["text"].(string)
				if !ok {
					continue
				}
				builder.WriteString(text)
			}
			return normalizeText(builder.String())
		}
	case string:
		return normalizeText(value)
	}
	return ""
}

func milliseconds(node any) (int64, bool) {
	value, ok := node.(float64)
	if !ok {
		return 0, false
	}
	return int64(value), true
}

// normalize sorts chapters by offset, trims titles, discards invalid entries
// and removes duplicate offsets.
func normalize(items []Chapter) []Chapter {
	filtered := make([]Chapter, 0, len(items))
	for _, item := range items {
		title := normalizeText(item.Title)
		if title == "" || item.OffsetMs < 0 {
			continue
		}
		filtered = append(filtered, Chapter{OffsetMs: item.OffsetMs, Title: title})
	}

	sort.SliceStable(filtered, func(i, j int) bool {
		return filtered[i].OffsetMs < filtered[j].OffsetMs
	})

	deduped := filtered[:0]
	for _, item := range filtered {
		if len(deduped) > 0 && deduped[len(deduped)-1].OffsetMs == item.OffsetMs {
			continue
		}
		deduped = append(deduped, item)
	}
	return deduped
}

func normalizeText(text string) string {
	return strings.Join(strings.Fields(text), " ")
}
