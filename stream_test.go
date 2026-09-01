package feedstream

import (
	"io"
	"strings"
	"testing"
	"time"
)

const sampleRSS = `<?xml version="1.0"?>
<rss version="2.0"
    xmlns:content="http://purl.org/rss/1.0/modules/content/"
    xmlns:media="http://search.yahoo.com/mrss/">
  <channel>
    <title>Example Log</title>
    <link>https://example.com</link>
    <description>Updates from example.com</description>
    <item>
      <title>First post</title>
      <link>https://example.com/1</link>
      <description>Hello &amp; welcome</description>
      <guid>https://example.com/1</guid>
      <pubDate>Mon, 02 Jan 2006 15:04:05 GMT</pubDate>
      <author>jane@example.com</author>
      <content:encoded>&lt;p&gt;Full HTML body&lt;/p&gt;</content:encoded>
      <media:content url="https://example.com/1.mp3" type="audio/mpeg"/>
    </item>
    <item>
      <title>Second post</title>
      <link>https://example.com/2</link>
      <description>Non-breaking&nbsp;space test</description>
    </item>
  </channel>
</rss>`

const sampleRDF = `<?xml version="1.0"?>
<rdf:RDF
    xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#"
    xmlns="http://purl.org/rss/1.0/"
    xmlns:dc="http://purl.org/dc/elements/1.1/">
  <channel rdf:about="https://example.com/rdf">
    <title>Example RDF Log</title>
    <link>https://example.com</link>
    <description>Updates from example.com</description>
  </channel>
  <item rdf:about="https://example.com/rdf/1">
    <title>RDF post</title>
    <link>https://example.com/rdf/1</link>
    <description>An RDF item</description>
    <dc:date>2006-01-02T15:04:05Z</dc:date>
    <dc:creator>Jane</dc:creator>
  </item>
</rdf:RDF>`

const sampleAtom = `<?xml version="1.0"?>
<feed xmlns="http://www.w3.org/2005/Atom" xmlns:media="http://search.yahoo.com/mrss/">
  <title>Example Atom Log</title>
  <link href="https://example.com" rel="alternate"/>
  <subtitle>Atom updates</subtitle>
  <entry>
    <title>Atom post</title>
    <id>urn:uuid:1</id>
    <updated>2006-01-02T15:04:05Z</updated>
    <summary>An atom entry</summary>
    <content>Full entry body</content>
    <author><name>Jane</name></author>
    <link href="https://example.com/atom/1" rel="alternate"/>
    <media:content url="https://example.com/atom/1.mp3" type="audio/mpeg"/>
  </entry>
</feed>`

func TestDecoderRSS(t *testing.T) {
	dec := NewDecoder(strings.NewReader(sampleRSS))

	first, err := dec.Next()
	if err != nil {
		t.Fatalf("first item: %v", err)
	}
	if first.Title != "First post" || first.Link != "https://example.com/1" {
		t.Fatalf("unexpected first item: %+v", first)
	}
	if first.Description != "Hello & welcome" {
		t.Fatalf("unexpected description: %q", first.Description)
	}
	wantPublished := time.Date(2006, time.January, 2, 15, 4, 5, 0, time.UTC)
	if !first.Published.Equal(wantPublished) {
		t.Fatalf("unexpected published time: %v", first.Published)
	}
	if first.Content != "<p>Full HTML body</p>" {
		t.Fatalf("unexpected content:encoded: %q", first.Content)
	}
	if first.MediaURL != "https://example.com/1.mp3" || first.MediaType != "audio/mpeg" {
		t.Fatalf("unexpected media:content: url=%q type=%q", first.MediaURL, first.MediaType)
	}

	second, err := dec.Next()
	if err != nil {
		t.Fatalf("second item: %v", err)
	}
	if second.Title != "Second post" {
		t.Fatalf("unexpected second item: %+v", second)
	}
	if !strings.Contains(second.Description, "Non-breaking") {
		t.Fatalf("unnamed entity handling failed: %q", second.Description)
	}
	if !second.Published.IsZero() {
		t.Fatalf("expected zero time for missing pubDate, got %v", second.Published)
	}

	if _, err := dec.Next(); err != io.EOF {
		t.Fatalf("expected io.EOF, got %v", err)
	}

	feed := dec.Feed()
	if feed.Title != "Example Log" || feed.Link != "https://example.com" {
		t.Fatalf("unexpected feed metadata: %+v", feed)
	}
}

func TestDecoderRDF(t *testing.T) {
	dec := NewDecoder(strings.NewReader(sampleRDF))

	item, err := dec.Next()
	if err != nil {
		t.Fatalf("item: %v", err)
	}
	if item.Title != "RDF post" || item.Link != "https://example.com/rdf/1" {
		t.Fatalf("unexpected item: %+v", item)
	}
	if item.GUID != "https://example.com/rdf/1" {
		t.Fatalf("unexpected guid (want rdf:about fallback): %q", item.GUID)
	}
	if item.Author != "Jane" {
		t.Fatalf("unexpected author (want dc:creator fallback): %q", item.Author)
	}
	wantPublished := time.Date(2006, time.January, 2, 15, 4, 5, 0, time.UTC)
	if !item.Published.Equal(wantPublished) {
		t.Fatalf("unexpected published time (want dc:date fallback): %v", item.Published)
	}

	if _, err := dec.Next(); err != io.EOF {
		t.Fatalf("expected io.EOF, got %v", err)
	}

	feed := dec.Feed()
	if feed.Title != "Example RDF Log" || feed.Link != "https://example.com" {
		t.Fatalf("unexpected feed metadata: %+v", feed)
	}
}

func TestDecoderAtom(t *testing.T) {
	dec := NewDecoder(strings.NewReader(sampleAtom))

	entry, err := dec.Next()
	if err != nil {
		t.Fatalf("entry: %v", err)
	}
	if entry.Title != "Atom post" || entry.Link != "https://example.com/atom/1" {
		t.Fatalf("unexpected entry: %+v", entry)
	}
	if entry.Author != "Jane" {
		t.Fatalf("unexpected author: %q", entry.Author)
	}
	wantPublished := time.Date(2006, time.January, 2, 15, 4, 5, 0, time.UTC)
	if !entry.Published.Equal(wantPublished) {
		t.Fatalf("unexpected published time: %v", entry.Published)
	}
	if entry.Content != "Full entry body" {
		t.Fatalf("unexpected content: %q", entry.Content)
	}
	if entry.MediaURL != "https://example.com/atom/1.mp3" || entry.MediaType != "audio/mpeg" {
		t.Fatalf("unexpected media:content: url=%q type=%q", entry.MediaURL, entry.MediaType)
	}

	if _, err := dec.Next(); err != io.EOF {
		t.Fatalf("expected io.EOF, got %v", err)
	}

	feed := dec.Feed()
	if feed.Title != "Example Atom Log" || feed.Description != "Atom updates" {
		t.Fatalf("unexpected feed metadata: %+v", feed)
	}
}
