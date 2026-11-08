package transcript

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
)

const (
	defaultTimedTextURL = "https://www.youtube.com/api/timedtext"
	timedTextUserAgent  = "transcript-tube-core/0.2.0 (+https://github.com/dev-maicondmenezes/transcript-tube-core)"
	maxTimedTextBody    = 8 << 20 // 8 MiB defensive limit
)

// timedTextFetcher implements the public timedtext fallback. It first asks for
// JSON3 and then for srv3 XML.
type timedTextFetcher struct {
	httpClient *http.Client
	baseURL    string
}

func newTimedTextFetcher(httpClient *http.Client) *timedTextFetcher {
	return &timedTextFetcher{
		httpClient: httpClient,
		baseURL:    defaultTimedTextURL,
	}
}

func (f *timedTextFetcher) Fetch(ctx context.Context, videoID, lang string) ([]Segment, error) {
	if lang == "" {
		return nil, ErrNoSegments
	}

	jsonSegments, jsonErr := f.fetchFormat(ctx, videoID, lang, "json3")
	if jsonErr == nil && len(jsonSegments) > 0 {
		return jsonSegments, nil
	}

	xmlSegments, xmlErr := f.fetchFormat(ctx, videoID, lang, "srv3")
	if xmlErr == nil && len(xmlSegments) > 0 {
		return xmlSegments, nil
	}

	return nil, fmt.Errorf("timedtext: json3: %v; srv3: %v", jsonErr, xmlErr)
}

func (f *timedTextFetcher) fetchFormat(ctx context.Context, videoID, lang, format string) ([]Segment, error) {
	endpoint, err := url.Parse(f.baseURL)
	if err != nil {
		return nil, fmt.Errorf("timedtext: invalid endpoint: %w", err)
	}

	query := endpoint.Query()
	query.Set("v", videoID)
	query.Set("lang", lang)
	query.Set("fmt", format)
	endpoint.RawQuery = query.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("timedtext: creating request: %w", err)
	}
	request.Header.Set("User-Agent", timedTextUserAgent)

	response, err := f.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("timedtext: request failed: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("timedtext: unexpected status %d", response.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(response.Body, maxTimedTextBody))
	if err != nil {
		return nil, fmt.Errorf("timedtext: reading body: %w", err)
	}
	if len(body) == 0 {
		return nil, ErrNoSegments
	}

	if format == "json3" {
		return parseJSON3(body)
	}
	return parseSRV3(body)
}

type json3Document struct {
	Events []json3Event `json:"events"`
}

type json3Event struct {
	StartMs  *int64     `json:"tStartMs"`
	Segments []json3Seg `json:"segs"`
}

type json3Seg struct {
	UTF8 string `json:"utf8"`
}

func parseJSON3(body []byte) ([]Segment, error) {
	var document json3Document
	if err := json.Unmarshal(body, &document); err != nil {
		return nil, fmt.Errorf("timedtext: parsing json3: %w", err)
	}

	segments := make([]Segment, 0, len(document.Events))
	for _, event := range document.Events {
		if event.StartMs == nil {
			continue
		}
		text := normalizeText(joinJSON3Segments(event.Segments))
		if text == "" {
			continue
		}
		segments = append(segments, Segment{OffsetMs: *event.StartMs, Text: text})
	}
	if len(segments) == 0 {
		return nil, ErrNoSegments
	}
	return segments, nil
}

func joinJSON3Segments(segments []json3Seg) string {
	result := ""
	for _, segment := range segments {
		result += segment.UTF8
	}
	return result
}

type srv3Document struct {
	XMLName xml.Name   `xml:"transcript"`
	Texts   []srv3Text `xml:"text"`
}

type srv3Text struct {
	Start string `xml:"start,attr"`
	Text  string `xml:",chardata"`
}

func parseSRV3(body []byte) ([]Segment, error) {
	var document srv3Document
	if err := xml.Unmarshal(body, &document); err != nil {
		return nil, fmt.Errorf("timedtext: parsing srv3: %w", err)
	}

	segments := make([]Segment, 0, len(document.Texts))
	for _, text := range document.Texts {
		value := normalizeText(text.Text)
		if value == "" {
			continue
		}
		startSeconds, err := strconv.ParseFloat(text.Start, 64)
		if err != nil {
			continue
		}
		segments = append(segments, Segment{
			OffsetMs: int64(startSeconds * 1000),
			Text:     value,
		})
	}
	if len(segments) == 0 {
		return nil, ErrNoSegments
	}
	return segments, nil
}
