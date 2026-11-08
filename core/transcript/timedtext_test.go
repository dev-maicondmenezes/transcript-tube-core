package transcript

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync/atomic"
	"testing"
)

func TestTimedTextFetcherPrefersJSON3(t *testing.T) {
	t.Parallel()

	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if got := r.URL.Query().Get("fmt"); got != "json3" {
			t.Errorf("fmt = %q, want json3", got)
		}
		if got := r.URL.Query().Get("v"); got != "dQw4w9WgXcQ" {
			t.Errorf("v = %q", got)
		}
		if got := r.URL.Query().Get("lang"); got != "pt" {
			t.Errorf("lang = %q", got)
		}
		if got := r.Header.Get("User-Agent"); got == "" {
			t.Error("User-Agent is empty")
		}
		fmt.Fprint(w, `{"events":[{"tStartMs":0,"segs":[{"utf8":"Olá"}]}]}`)
	}))
	defer server.Close()

	fetcher := newTimedTextFetcher(server.Client())
	fetcher.baseURL = server.URL

	segments, err := fetcher.Fetch(context.Background(), "dQw4w9WgXcQ", "pt")
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	want := []Segment{{OffsetMs: 0, Text: "Olá"}}
	if !reflect.DeepEqual(segments, want) {
		t.Fatalf("Fetch() = %#v, want %#v", segments, want)
	}
	if requests.Load() != 1 {
		t.Fatalf("requests = %d, want 1", requests.Load())
	}
}

func TestTimedTextFetcherFallsBackToSRV3(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Query().Get("fmt") {
		case "json3":
			fmt.Fprint(w, `{"events":[]}`)
		case "srv3":
			fmt.Fprint(w, `<transcript><text start="1.5">Fallback</text></transcript>`)
		default:
			t.Errorf("unexpected fmt %q", r.URL.Query().Get("fmt"))
		}
	}))
	defer server.Close()

	fetcher := newTimedTextFetcher(server.Client())
	fetcher.baseURL = server.URL

	segments, err := fetcher.Fetch(context.Background(), "dQw4w9WgXcQ", "pt")
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	want := []Segment{{OffsetMs: 1500, Text: "Fallback"}}
	if !reflect.DeepEqual(segments, want) {
		t.Fatalf("Fetch() = %#v, want %#v", segments, want)
	}
}

func TestTimedTextFetcherHTTPError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	fetcher := newTimedTextFetcher(server.Client())
	fetcher.baseURL = server.URL

	if _, err := fetcher.Fetch(context.Background(), "dQw4w9WgXcQ", "pt"); err == nil {
		t.Fatal("Fetch() error = nil, want error")
	}
}

func TestTimedTextFetcherEmptyLanguage(t *testing.T) {
	t.Parallel()

	fetcher := newTimedTextFetcher(http.DefaultClient)
	if _, err := fetcher.Fetch(context.Background(), "dQw4w9WgXcQ", ""); err == nil {
		t.Fatal("Fetch() error = nil, want error")
	}
}
