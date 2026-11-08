// Command core-demo is a minimal consumer of the core package. It accepts a
// YouTube URL or a bare video ID and prints the generated Markdown to stdout.
package main

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/dev-maicondmenezes/transcript-tube-core/core"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "uso: core-demo <URL-ou-ID>")
		os.Exit(1)
	}

	videoID, err := parseVideoID(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "erro: %v\n", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	data, err := core.ExtractWithOptions(ctx, videoID, core.Options{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "erro: %v\n", err)
		os.Exit(2)
	}

	fmt.Print(core.ToMarkdown(data))
}

func parseVideoID(raw string) (string, error) {
	if core.ValidateVideoID(raw) == nil {
		return raw, nil
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("URL inválida: %w", err)
	}

	var videoID string
	switch strings.ToLower(parsed.Hostname()) {
	case "youtu.be":
		videoID = strings.TrimPrefix(parsed.Path, "/")
	case "youtube.com", "www.youtube.com", "m.youtube.com", "music.youtube.com":
		videoID = parsed.Query().Get("v")
	default:
		return "", fmt.Errorf("URL não reconhecida: %s", raw)
	}

	if err := core.ValidateVideoID(videoID); err != nil {
		return "", err
	}
	return videoID, nil
}
