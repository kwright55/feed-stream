package feedstream

import (
	"strings"
	"time"
)

// dateLayouts covers the date formats RSS pubDate and Atom updated
// elements show up in across real feeds. RFC 822 (as used by RSS) and
// RFC 3339 (as used by Atom) come first since they're the spec-correct
// forms; the rest are fallbacks for feeds that don't quite follow spec.
var dateLayouts = []string{
	time.RFC1123Z,
	time.RFC1123,
	time.RFC3339,
	time.RFC3339Nano,
	"2006-01-02T15:04:05-0700",
	"2006-01-02T15:04:05",
	"2006-01-02 15:04:05",
	"02 Jan 2006 15:04:05 -0700",
	"02 Jan 2006 15:04:05 MST",
}

// parseDate tries each known layout in turn and returns the zero Time
// if none of them match. A malformed date in one item shouldn't fail
// the whole decode, so the caller is expected to check IsZero rather
// than treat an error as fatal.
func parseDate(s string) time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}
	}
	for _, layout := range dateLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Time{}
}
