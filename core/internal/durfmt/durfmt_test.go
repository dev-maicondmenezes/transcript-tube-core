package durfmt

import (
	"testing"
	"time"
)

func TestParseISO(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input   string
		want    time.Duration
		wantErr bool
	}{
		{input: "PT12M34S", want: 12*time.Minute + 34*time.Second},
		{input: "PT1H2M3S", want: time.Hour + 2*time.Minute + 3*time.Second},
		{input: "PT45S", want: 45 * time.Second},
		{input: "PT1.5S", want: 1500 * time.Millisecond},
		{input: "P1DT2H", want: 26 * time.Hour},
		{input: "", wantErr: true},
		{input: "PT", wantErr: true},
		{input: "P", wantErr: true},
		{input: "12:34", wantErr: true},
		{input: "PT1H2M3", wantErr: true},
	}

	for _, test := range tests {
		test := test
		t.Run(test.input, func(t *testing.T) {
			t.Parallel()

			got, err := ParseISO(test.input)
			if test.wantErr {
				if err == nil {
					t.Fatalf("ParseISO(%q) = %v, want error", test.input, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseISO(%q) error = %v", test.input, err)
			}
			if got != test.want {
				t.Fatalf("ParseISO(%q) = %v, want %v", test.input, got, test.want)
			}
		})
	}
}

func TestFormatAndTimestamp(t *testing.T) {
	t.Parallel()

	tests := []struct {
		duration  time.Duration
		formatted string
	}{
		{duration: -time.Second, formatted: "00:00"},
		{duration: 0, formatted: "00:00"},
		{duration: 59 * time.Second, formatted: "00:59"},
		{duration: time.Minute, formatted: "01:00"},
		{duration: 59*time.Minute + 59*time.Second, formatted: "59:59"},
		{duration: time.Hour, formatted: "01:00:00"},
		{duration: time.Hour + time.Minute + time.Second, formatted: "01:01:01"},
	}

	for _, test := range tests {
		if got := Format(test.duration); got != test.formatted {
			t.Fatalf("Format(%v) = %q, want %q", test.duration, got, test.formatted)
		}
	}

	if got := Timestamp(-1500); got != "00:00" {
		t.Fatalf("Timestamp(-1500) = %q, want 00:00", got)
	}
	if got := Timestamp(3661000); got != "01:01:01" {
		t.Fatalf("Timestamp(3661000) = %q, want 01:01:01", got)
	}
}
