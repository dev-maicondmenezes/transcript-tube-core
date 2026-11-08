package core

import (
	"testing"
	"time"
)

func TestFormatTimestamp(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		ms   int64
		want string
	}{
		{name: "zero", ms: 0, want: "00:00"},
		{name: "fifty nine seconds", ms: 59000, want: "00:59"},
		{name: "one minute", ms: 60000, want: "01:00"},
		{name: "fifty nine fifty nine", ms: 3599000, want: "59:59"},
		{name: "one hour", ms: 3600000, want: "01:00:00"},
		{name: "more than one hour", ms: 3661000, want: "01:01:01"},
		{name: "negative", ms: -1500, want: "00:00"},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if got := formatTimestamp(test.ms); got != test.want {
				t.Fatalf("formatTimestamp(%d) = %q, want %q", test.ms, got, test.want)
			}
		})
	}
}

func TestParseISODuration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    time.Duration
		wantErr bool
	}{
		{name: "minutes and seconds", input: "PT12M34S", want: 12*time.Minute + 34*time.Second},
		{name: "hours minutes seconds", input: "PT1H2M3S", want: time.Hour + 2*time.Minute + 3*time.Second},
		{name: "seconds only", input: "PT45S", want: 45 * time.Second},
		{name: "days", input: "P1DT2H", want: 26 * time.Hour},
		{name: "empty", input: "", wantErr: true},
		{name: "invalid", input: "12:34", wantErr: true},
		{name: "empty time part", input: "PT", wantErr: true},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseISODuration(test.input)
			if test.wantErr {
				if err == nil {
					t.Fatalf("parseISODuration(%q) = %v, want error", test.input, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseISODuration(%q) error = %v", test.input, err)
			}
			if got != test.want {
				t.Fatalf("parseISODuration(%q) = %v, want %v", test.input, got, test.want)
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		duration time.Duration
		want     string
	}{
		{duration: 0, want: "00:00"},
		{duration: 12*time.Minute + 34*time.Second, want: "12:34"},
		{duration: time.Hour + 2*time.Minute + 3*time.Second, want: "01:02:03"},
		{duration: -time.Second, want: "00:00"},
	}

	for _, test := range tests {
		if got := formatDuration(test.duration); got != test.want {
			t.Fatalf("formatDuration(%v) = %q, want %q", test.duration, got, test.want)
		}
	}
}
