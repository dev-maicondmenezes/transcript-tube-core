package chapters

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

const nextResponseFixture = `{
  "playerOverlays": {
    "playerOverlayRenderer": {
      "decoratedPlayerBarRenderer": {
        "decoratedPlayerBarRenderer": {
          "playerBar": {
            "multiMarkersPlayerBarRenderer": {
              "markersMap": [
                {
                  "key": "AUTO_CHAPTERS",
                  "value": {
                    "chapters": [
                      {"chapterRenderer": {"title": {"simpleText": "Segundo"}, "timeRangeStartMillis": 90000}},
                      {"chapterRenderer": {"title": {"runs": [{"text": "Pri"}, {"text": "meiro"}]}, "timeRangeStartMillis": 0}},
                      {"chapterRenderer": {"title": {"simpleText": "  Duplicado  "}, "timeRangeStartMillis": 90000}},
                      {"chapterRenderer": {"title": {"simpleText": "Sem offset"}}}
                    ]
                  }
                }
              ]
            }
          }
        }
      }
    }
  }
}`

func TestClientFetchParsesChapters(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/youtubei/v1/next" {
			t.Errorf("path = %s, want /youtubei/v1/next", r.URL.Path)
		}
		if r.URL.Query().Get("prettyPrint") != "false" {
			t.Errorf("prettyPrint = %q, want false", r.URL.Query().Get("prettyPrint"))
		}
		if contentType := r.Header.Get("Content-Type"); contentType != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", contentType)
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decoding request body: %v", err)
			return
		}
		if body["videoId"] != "_MkQLDMak-4" {
			t.Errorf("videoId = %v, want _MkQLDMak-4", body["videoId"])
		}
		contextValue, _ := body["context"].(map[string]any)
		clientValue, _ := contextValue["client"].(map[string]any)
		if clientValue["clientName"] != "WEB" {
			t.Errorf("clientName = %v, want WEB", clientValue["clientName"])
		}
		if clientValue["clientVersion"] != defaultClientVersion {
			t.Errorf("clientVersion = %v, want %s", clientValue["clientVersion"], defaultClientVersion)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(nextResponseFixture))
	}))
	defer server.Close()

	client := New(WithBaseURL(server.URL))
	got, err := client.Fetch(context.Background(), "_MkQLDMak-4")
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	want := []Chapter{
		{OffsetMs: 0, Title: "Primeiro"},
		{OffsetMs: 90000, Title: "Segundo"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Fetch() = %#v, want %#v", got, want)
	}
}

func TestClientFetchCustomClientVersion(t *testing.T) {
	t.Parallel()

	var received string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decoding request body: %v", err)
			return
		}
		contextValue, _ := body["context"].(map[string]any)
		clientValue, _ := contextValue["client"].(map[string]any)
		received, _ = clientValue["clientVersion"].(string)
		_, _ = w.Write([]byte(nextResponseFixture))
	}))
	defer server.Close()

	client := New(WithBaseURL(server.URL), WithClientVersion("9.9.9"))
	if _, err := client.Fetch(context.Background(), "_MkQLDMak-4"); err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if received != "9.9.9" {
		t.Fatalf("clientVersion = %q, want 9.9.9", received)
	}
}

func TestClientFetchNoChapters(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"playerOverlays":{"playerOverlayRenderer":{}}}`))
	}))
	defer server.Close()

	client := New(WithBaseURL(server.URL))
	_, err := client.Fetch(context.Background(), "_MkQLDMak-4")
	if !errors.Is(err, ErrNoChapters) {
		t.Fatalf("Fetch() error = %v, want ErrNoChapters", err)
	}
}

func TestClientFetchHTTPError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := New(WithBaseURL(server.URL))
	_, err := client.Fetch(context.Background(), "_MkQLDMak-4")
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("Fetch() error = %v, want ErrUnavailable", err)
	}
}

func TestClientFetchMalformedJSON(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"playerOverlays":`))
	}))
	defer server.Close()

	client := New(WithBaseURL(server.URL))
	_, err := client.Fetch(context.Background(), "_MkQLDMak-4")
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("Fetch() error = %v, want ErrUnavailable", err)
	}
}

func TestClientFetchCanceledContext(t *testing.T) {
	t.Parallel()

	client := New(WithBaseURL("http://127.0.0.1:1"))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := client.Fetch(ctx, "_MkQLDMak-4")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Fetch() error = %v, want context.Canceled", err)
	}
}

func TestClientFetchEmptyVideoID(t *testing.T) {
	t.Parallel()

	client := New(WithBaseURL("http://127.0.0.1:1"))
	_, err := client.Fetch(context.Background(), "   ")
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("Fetch() error = %v, want ErrUnavailable", err)
	}
}

func TestParseNextResponseIgnoresNonChapterNodes(t *testing.T) {
	t.Parallel()

	body := []byte(`{
		"chapterRenderer": {"title": {"simpleText": "Direct"}, "timeRangeStartMillis": 0},
		"other": {"title": {"simpleText": "Not a chapter"}}
	}`)
	got, err := parseNextResponse(body)
	if err != nil {
		t.Fatalf("parseNextResponse() error = %v", err)
	}
	want := []Chapter{{OffsetMs: 0, Title: "Direct"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parseNextResponse() = %#v, want %#v", got, want)
	}
}
