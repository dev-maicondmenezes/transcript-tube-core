package transcript

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// Client implements the transcript cascade. It tries InnerTube for every
// preferred language first and then falls back to timedtext.
type Client struct {
	innerTube  Fetcher
	timedText  Fetcher
	httpClient *http.Client
}

// Option customises a Client. Options exist mainly for tests.
type Option func(*Client)

// WithInnerTubeFetcher replaces the InnerTube strategy.
func WithInnerTubeFetcher(fetcher Fetcher) Option {
	return func(c *Client) {
		c.innerTube = fetcher
	}
}

// WithTimedTextFetcher replaces the timedtext strategy.
func WithTimedTextFetcher(fetcher Fetcher) Option {
	return func(c *Client) {
		c.timedText = fetcher
	}
}

// WithHTTPClient sets the HTTP client used by the default timedtext strategy.
func WithHTTPClient(client *http.Client) Option {
	return func(c *Client) {
		c.httpClient = client
	}
}

// New creates a transcript client with the default cascade.
func New(opts ...Option) *Client {
	client := &Client{}
	for _, opt := range opts {
		opt(client)
	}

	if client.innerTube == nil {
		client.innerTube = newInnerTubeFetcher()
	}
	if client.timedText == nil {
		httpClient := client.httpClient
		if httpClient == nil {
			httpClient = http.DefaultClient
		}
		client.timedText = newTimedTextFetcher(httpClient)
	}

	return client
}

// Fetch returns transcript segments for videoID. When lang is empty it tries
// pt, pt-BR and en, then any language offered by InnerTube. If every strategy
// fails, the returned error wraps ErrUnavailable.
func (c *Client) Fetch(ctx context.Context, videoID, lang string) ([]Segment, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	candidates := languageCandidates(lang)
	innerTubeCandidates := append(append([]string{}, candidates...), "") // "" = first available
	// timedtext requires a concrete language code.
	var errs []error

	for _, candidate := range innerTubeCandidates {
		segments, err := c.innerTube.Fetch(ctx, videoID, candidate)
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}
		if err == nil && len(segments) > 0 {
			return segments, nil
		}
		errs = append(errs, fetchError("innertube", candidate, err))
	}

	for _, candidate := range candidates {
		segments, err := c.timedText.Fetch(ctx, videoID, candidate)
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}
		if err == nil && len(segments) > 0 {
			return segments, nil
		}
		errs = append(errs, fetchError("timedtext", candidate, err))
	}

	return nil, fmt.Errorf("%w: %v", ErrUnavailable, errors.Join(errs...))
}

func languageCandidates(lang string) []string {
	ordered := make([]string, 0, 4)
	add := func(code string) {
		code = strings.TrimSpace(code)
		if code == "" {
			return
		}
		for _, existing := range ordered {
			if existing == code {
				return
			}
		}
		ordered = append(ordered, code)
	}

	add(lang)
	add("pt")
	add("pt-BR")
	add("en")
	return ordered
}

func fetchError(strategy, lang string, err error) error {
	language := lang
	if language == "" {
		language = "any"
	}
	if err == nil {
		err = ErrNoSegments
	}
	return fmt.Errorf("%s[%s]: %w", strategy, language, err)
}

// normalizeText collapses every whitespace run into a single space and trims
// the result. Transcript providers frequently pad lines and emit newlines.
func normalizeText(text string) string {
	return strings.Join(strings.Fields(text), " ")
}
