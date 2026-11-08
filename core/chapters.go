package core

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// chapterLinePattern matches a timestamp at the beginning of a description
// line, optionally wrapped in brackets and preceded by a bullet, followed by
// an optional separator and the chapter title. Examples:
//
//	0:00 Intro
//	1:23 - Middle
//	[02:05] End
//	00:00: Intro
var chapterLinePattern = regexp.MustCompile(`^\s*(?:[-*•]\s*)?\[?(\d{1,2}(?::\d{2}){1,2})\]?\s*(?:[-–—:|]\s*)?(.*)$`)

// parseChapters extracts chapters from the timestamp lines of a YouTube
// description. YouTube builds its chapter list from exactly this kind of
// content, so the description is the canonical source available through the
// Data API.
//
// To avoid false positives, at least two timestamped lines are required and
// the first chapter must start at 00:00. The result is sorted by offset and
// duplicate offsets are discarded.
func parseChapters(description string) []Chapter {
	chapters := make([]Chapter, 0)
	for _, line := range strings.Split(description, "\n") {
		matches := chapterLinePattern.FindStringSubmatch(line)
		if matches == nil {
			continue
		}

		offsetMs, ok := parseChapterTimestamp(matches[1])
		if !ok {
			continue
		}

		title := strings.TrimSpace(matches[2])
		if title == "" {
			continue
		}

		chapters = append(chapters, Chapter{OffsetMs: offsetMs, Title: title})
	}

	chapters = normalizeChapters(chapters)
	if len(chapters) < 2 || chapters[0].OffsetMs != 0 {
		return nil
	}
	return chapters
}

// parseChapterTimestamp parses "mm:ss" or "hh:mm:ss" into milliseconds.
func parseChapterTimestamp(value string) (int64, bool) {
	parts := strings.Split(value, ":")
	switch len(parts) {
	case 2:
		minutes, errMinutes := strconv.Atoi(parts[0])
		seconds, errSeconds := strconv.Atoi(parts[1])
		if errMinutes != nil || errSeconds != nil {
			return 0, false
		}
		if minutes < 0 || seconds < 0 || seconds > 59 {
			return 0, false
		}
		return int64(minutes*60+seconds) * 1000, true
	case 3:
		hours, errHours := strconv.Atoi(parts[0])
		minutes, errMinutes := strconv.Atoi(parts[1])
		seconds, errSeconds := strconv.Atoi(parts[2])
		if errHours != nil || errMinutes != nil || errSeconds != nil {
			return 0, false
		}
		if hours < 0 || minutes < 0 || minutes > 59 || seconds < 0 || seconds > 59 {
			return 0, false
		}
		return int64(hours*3600+minutes*60+seconds) * 1000, true
	default:
		return 0, false
	}
}

// normalizeChapters sorts chapters by offset, trims titles, discards invalid
// entries and removes duplicate offsets. It is also used by ToMarkdown so that
// externally constructed VideoData behaves consistently.
func normalizeChapters(chapters []Chapter) []Chapter {
	filtered := make([]Chapter, 0, len(chapters))
	for _, chapter := range chapters {
		title := strings.Join(strings.Fields(chapter.Title), " ")
		if title == "" || chapter.OffsetMs < 0 {
			continue
		}
		filtered = append(filtered, Chapter{OffsetMs: chapter.OffsetMs, Title: title})
	}

	sort.SliceStable(filtered, func(i, j int) bool {
		return filtered[i].OffsetMs < filtered[j].OffsetMs
	})

	deduped := filtered[:0]
	for _, chapter := range filtered {
		if len(deduped) > 0 && deduped[len(deduped)-1].OffsetMs == chapter.OffsetMs {
			continue
		}
		deduped = append(deduped, chapter)
	}
	return deduped
}
