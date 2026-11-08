package core

import (
	"fmt"
	"strings"
	"time"
)

const unknownValue = "desconhecida"

// ToMarkdown formats data using the canonical Markdown template. A nil value
// produces an empty string.
func ToMarkdown(data *VideoData) string {
	if data == nil {
		return ""
	}

	var b strings.Builder

	fmt.Fprintf(&b, "# %s\n\n", data.Title)
	fmt.Fprintf(&b, "**Canal:** %s\n", data.Channel)
	fmt.Fprintf(&b, "**URL:** https://www.youtube.com/watch?v=%s\n", data.ID)
	fmt.Fprintf(&b, "**Publicado em:** %s\n", formatPublishedAt(data.PublishedAt))
	fmt.Fprintf(&b, "**Duração:** %s\n", valueOrUnknown(data.Duration))
	fmt.Fprintf(&b, "**Visualizações:** %d\n", data.ViewCount)
	fmt.Fprintf(&b, "**Likes:** %d\n", data.LikeCount)

	b.WriteString("\n---\n\n## Descrição\n\n")
	if description := strings.TrimSpace(data.Description); description != "" {
		b.WriteString(description)
		b.WriteString("\n\n")
	}

	b.WriteString("---\n\n## Transcrição\n\n")

	chapters := normalizeChapters(data.Chapters)
	chapterIndex := 0
	wroteSegment := false
	for _, segment := range data.Transcript {
		text := strings.TrimSpace(segment.Text)
		if text == "" {
			continue
		}

		// Emit every chapter that starts at or before this segment. Chapters
		// that do not align with a segment are anchored to the first segment
		// after their start time.
		for chapterIndex < len(chapters) && chapters[chapterIndex].OffsetMs <= segment.OffsetMs {
			if wroteSegment {
				b.WriteString("\n")
			}
			fmt.Fprintf(&b, "### [%s] %s\n\n", formatTimestamp(chapters[chapterIndex].OffsetMs), chapters[chapterIndex].Title)
			chapterIndex++
		}

		fmt.Fprintf(&b, "[%s] %s\n", formatTimestamp(segment.OffsetMs), text)
		wroteSegment = true
	}
	if !wroteSegment {
		b.WriteString("_Transcrição indisponível._\n")
	}

	return b.String()
}

func formatPublishedAt(t time.Time) string {
	if t.IsZero() {
		return unknownValue
	}
	return t.UTC().Format("02/01/2006 15:04") + " UTC"
}

func valueOrUnknown(value string) string {
	if strings.TrimSpace(value) == "" {
		return unknownValue
	}
	return value
}
