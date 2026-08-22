// Package feedstream reads RSS 2.0 and Atom feeds one item at a time
// instead of unmarshalling the whole document into a slice up front.
package feedstream

// Feed holds the channel/feed-level metadata that sits alongside the
// item list in both RSS and Atom documents.
type Feed struct {
	Title       string
	Link        string
	Description string
}

// Item is a single entry from a feed, normalized across RSS <item> and
// Atom <entry> elements. Fields that a given feed doesn't populate are
// left as the empty string rather than causing an error.
type Item struct {
	Title       string
	Link        string
	Description string
	GUID        string
	PubDate     string
	Author      string
}
