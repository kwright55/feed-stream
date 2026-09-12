// Package feedstream reads RSS 2.0, RSS 1.0 (RDF), and Atom feeds one
// item at a time instead of unmarshalling the whole document into a
// slice up front.
package feedstream

import "time"

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
	// GUIDIsPermaLink reflects the RSS <guid isPermaLink="..."> attribute.
	// Per the RSS 2.0 spec it defaults to true when the guid element is
	// present but the attribute is omitted, so this is only meaningful
	// when GUID is non-empty; it's false for the RSS 1.0 rdf:about
	// fallback and for Atom, since neither carries the concept.
	GUIDIsPermaLink bool
	// PubDate is the raw value of RSS <pubDate> or Atom <updated>,
	// kept as-is since callers may want the original text.
	PubDate string
	// Published is PubDate parsed against the date layouts feeds
	// actually use in practice. It's the zero Time if PubDate was
	// empty or didn't match any of them, so check IsZero before
	// relying on it.
	Published time.Time
	Author    string
	// Content is the full item body from the RSS content:encoded
	// extension or an Atom <content> element, as opposed to the
	// (often truncated) Description/Summary. Empty if the feed
	// doesn't provide one.
	Content string
	// MediaURL and MediaType come from a Media RSS <media:content>
	// element, commonly used for podcast audio/video enclosures.
	// Both are empty if the item has no media:content.
	MediaURL  string
	MediaType string
}
