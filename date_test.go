package feedstream

import (
	"testing"
	"time"
)

func TestParseDate(t *testing.T) {
	want := time.Date(2006, time.January, 2, 15, 4, 5, 0, time.UTC)

	cases := []string{
		"Mon, 02 Jan 2006 15:04:05 GMT",
		"Mon, 02 Jan 2006 15:04:05 +0000",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05+00:00",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"02 Jan 2006 15:04:05 +0000",
		"02 Jan 2006 15:04:05 GMT",
	}
	for _, s := range cases {
		got := parseDate(s)
		if !got.Equal(want) {
			t.Errorf("parseDate(%q) = %v, want %v", s, got, want)
		}
	}
}

func TestParseDateInvalid(t *testing.T) {
	for _, s := range []string{"", "   ", "not a date", "next Tuesday"} {
		if got := parseDate(s); !got.IsZero() {
			t.Errorf("parseDate(%q) = %v, want zero time", s, got)
		}
	}
}
