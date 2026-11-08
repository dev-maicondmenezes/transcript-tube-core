package core

import (
	"reflect"
	"testing"
)

func TestParseChapters(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		description string
		want        []Chapter
	}{
		{
			name: "basic chapters",
			description: `Welcome to the video.

0:00 Intro
1:23 - Middle
1:02:03 End`,
			want: []Chapter{
				{OffsetMs: 0, Title: "Intro"},
				{OffsetMs: 83000, Title: "Middle"},
				{OffsetMs: 3723000, Title: "End"},
			},
		},
		{
			name: "bullets brackets and colons",
			description: `Chapters:
- [0:00] Intro
* 00:05: First topic
• 00:10 | Second topic`,
			want: []Chapter{
				{OffsetMs: 0, Title: "Intro"},
				{OffsetMs: 5000, Title: "First topic"},
				{OffsetMs: 10000, Title: "Second topic"},
			},
		},
		{
			name: "unsorted and duplicated offsets",
			description: `0:00 Intro
2:00 End
1:00 Middle
1:00 Duplicated`,
			want: []Chapter{
				{OffsetMs: 0, Title: "Intro"},
				{OffsetMs: 60000, Title: "Middle"},
				{OffsetMs: 120000, Title: "End"},
			},
		},
		{
			name:        "fewer than two chapters",
			description: "0:00 Intro",
			want:        nil,
		},
		{
			name:        "first chapter is not at zero",
			description: "0:30 Intro\n1:30 End",
			want:        nil,
		},
		{
			name:        "invalid seconds are ignored",
			description: "0:00 Intro\n0:99 Invalid\n1:30 End",
			want: []Chapter{
				{OffsetMs: 0, Title: "Intro"},
				{OffsetMs: 90000, Title: "End"},
			},
		},
		{
			name:        "no timestamps",
			description: "Just a plain description without chapters.",
			want:        nil,
		},
		{
			name:        "timestamps without titles",
			description: "0:00\n1:00",
			want:        nil,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := parseChapters(test.description)
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("parseChapters() = %#v, want %#v", got, test.want)
			}
		})
	}
}

func TestParseChapterTimestamp(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  int64
		ok    bool
	}{
		{input: "0:00", want: 0, ok: true},
		{input: "1:23", want: 83000, ok: true},
		{input: "59:59", want: 3599000, ok: true},
		{input: "1:02:03", want: 3723000, ok: true},
		{input: "0:60", ok: false},
		{input: "1:60:00", ok: false},
		{input: "abc", ok: false},
	}

	for _, test := range tests {
		got, ok := parseChapterTimestamp(test.input)
		if ok != test.ok {
			t.Fatalf("parseChapterTimestamp(%q) ok = %v, want %v", test.input, ok, test.ok)
		}
		if ok && got != test.want {
			t.Fatalf("parseChapterTimestamp(%q) = %d, want %d", test.input, got, test.want)
		}
	}
}

func TestNormalizeChapters(t *testing.T) {
	t.Parallel()

	got := normalizeChapters([]Chapter{
		{OffsetMs: 10000, Title: "  Second   topic "},
		{OffsetMs: -1, Title: "Invalid"},
		{OffsetMs: 0, Title: ""},
		{OffsetMs: 0, Title: "First"},
		{OffsetMs: 10000, Title: "Duplicate"},
	})

	want := []Chapter{
		{OffsetMs: 0, Title: "First"},
		{OffsetMs: 10000, Title: "Second topic"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normalizeChapters() = %#v, want %#v", got, want)
	}
}
