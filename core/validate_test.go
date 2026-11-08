package core

import (
	"errors"
	"testing"
)

func TestValidateVideoID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		videoID string
		wantErr bool
	}{
		{name: "valid alphanumeric", videoID: "dQw4w9WgXcQ"},
		{name: "valid underscores and hyphens", videoID: "_-abcDEF123"},
		{name: "empty", videoID: "", wantErr: true},
		{name: "too short", videoID: "dQw4w9WgXc", wantErr: true},
		{name: "too long", videoID: "dQw4w9WgXcQQ", wantErr: true},
		{name: "invalid character dot", videoID: "dQw4w9WgXc.", wantErr: true},
		{name: "invalid character space", videoID: "dQw4w9Wg cQ", wantErr: true},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateVideoID(test.videoID)
			if test.wantErr {
				if err == nil {
					t.Fatalf("ValidateVideoID(%q) = nil, want error", test.videoID)
				}
				if !errors.Is(err, ErrInvalidVideoID) {
					t.Fatalf("ValidateVideoID(%q) error = %v, want ErrInvalidVideoID", test.videoID, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("ValidateVideoID(%q) = %v, want nil", test.videoID, err)
			}
		})
	}
}
