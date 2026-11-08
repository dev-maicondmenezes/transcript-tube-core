package transcript

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"
)

type fakeFetcher struct {
	segments []Segment
	err      error
	byLang   map[string][]Segment
	mu       sync.Mutex
	calls    []string
}

func (f *fakeFetcher) Fetch(_ context.Context, _ string, lang string) ([]Segment, error) {
	f.mu.Lock()
	f.calls = append(f.calls, lang)
	f.mu.Unlock()

	if f.byLang != nil {
		if segments, ok := f.byLang[lang]; ok {
			return segments, nil
		}
	}
	return f.segments, f.err
}

func TestClientCascadeInnerTubeThenTimedText(t *testing.T) {
	t.Parallel()

	innerTube := &fakeFetcher{err: errors.New("innertube down")}
	expected := []Segment{{OffsetMs: 0, Text: "from timedtext"}}
	timedText := &fakeFetcher{byLang: map[string][]Segment{"pt": expected}}

	client := New(
		WithInnerTubeFetcher(innerTube),
		WithTimedTextFetcher(timedText),
	)

	segments, err := client.Fetch(context.Background(), "dQw4w9WgXcQ", "pt")
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if !reflect.DeepEqual(segments, expected) {
		t.Fatalf("Fetch() = %#v, want %#v", segments, expected)
	}
	if len(innerTube.calls) == 0 {
		t.Fatal("InnerTube fetcher was not called")
	}
	if len(timedText.calls) == 0 {
		t.Fatal("timedtext fetcher was not called")
	}
}

func TestClientCascadeAllFail(t *testing.T) {
	t.Parallel()

	innerTube := &fakeFetcher{err: errors.New("innertube down")}
	timedText := &fakeFetcher{err: errors.New("timedtext down")}

	client := New(
		WithInnerTubeFetcher(innerTube),
		WithTimedTextFetcher(timedText),
	)

	_, err := client.Fetch(context.Background(), "dQw4w9WgXcQ", "pt")
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("Fetch() error = %v, want ErrUnavailable", err)
	}
}

func TestClientContextCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	client := New(
		WithInnerTubeFetcher(&fakeFetcher{segments: []Segment{{Text: "never"}}}),
		WithTimedTextFetcher(&fakeFetcher{segments: []Segment{{Text: "never"}}}),
	)

	_, err := client.Fetch(ctx, "dQw4w9WgXcQ", "pt")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Fetch() error = %v, want context.Canceled", err)
	}
}

func TestLanguageCandidates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		lang string
		want []string
	}{
		{lang: "", want: []string{"pt", "pt-BR", "en"}},
		{lang: "es", want: []string{"es", "pt", "pt-BR", "en"}},
		{lang: "pt", want: []string{"pt", "pt-BR", "en"}},
	}

	for _, test := range tests {
		if got := languageCandidates(test.lang); !reflect.DeepEqual(got, test.want) {
			t.Fatalf("languageCandidates(%q) = %v, want %v", test.lang, got, test.want)
		}
	}
}

func TestParseJSON3(t *testing.T) {
	t.Parallel()

	body := []byte(`{
		"events": [
			{"tStartMs": 0, "segs": [{"utf8": "Hello "}, {"utf8": "world"}]},
			{"tStartMs": 1500, "segs": [{"utf8": "  spaced \n text  "}]},
			{"tStartMs": 3000, "segs": [{"utf8": "   "}]},
			{"segs": [{"utf8": "no start"}]}
		]
	}`)

	segments, err := parseJSON3(body)
	if err != nil {
		t.Fatalf("parseJSON3() error = %v", err)
	}

	want := []Segment{
		{OffsetMs: 0, Text: "Hello world"},
		{OffsetMs: 1500, Text: "spaced text"},
	}
	if !reflect.DeepEqual(segments, want) {
		t.Fatalf("parseJSON3() = %#v, want %#v", segments, want)
	}
}

func TestParseSRV3(t *testing.T) {
	t.Parallel()

	body := []byte(`<transcript>
		<text start="0" dur="2">Hello &amp; welcome</text>
		<text start="2.5" dur="2">  Second line  </text>
	</transcript>`)

	segments, err := parseSRV3(body)
	if err != nil {
		t.Fatalf("parseSRV3() error = %v", err)
	}

	want := []Segment{
		{OffsetMs: 0, Text: "Hello & welcome"},
		{OffsetMs: 2500, Text: "Second line"},
	}
	if !reflect.DeepEqual(segments, want) {
		t.Fatalf("parseSRV3() = %#v, want %#v", segments, want)
	}
}
