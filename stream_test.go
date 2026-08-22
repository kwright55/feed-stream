package feedstream

import (
	"io"
	"strings"
	"testing"
)

const sampleRSS = `<?xml version="1.0"?>
<rss version="2.0">
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
    </item>
    <item>
      <title>Second post</title>
      <link>https://example.com/2</link>
      <description>Non-breaking&nbsp;space test</description>
    </item>
  </channel>
</rss>`

const sampleAtom = `<?xml version="1.0"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <title>Example Atom Log</title>
  <link href="https://example.com" rel="alternate"/>
  <subtitle>Atom updates</subtitle>
  <entry>
    <title>Atom post</title>
    <id>urn:uuid:1</id>
    <updated>2006-01-02T15:04:05Z</updated>
    <summary>An atom entry</summary>
    <author><name>Jane</name></author>
    <link href="https://example.com/atom/1" rel="alternate"/>
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

	if _, err := dec.Next(); err != io.EOF {
		t.Fatalf("expected io.EOF, got %v", err)
	}

	feed := dec.Feed()
	if feed.Title != "Example Log" || feed.Link != "https://example.com" {
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

	if _, err := dec.Next(); err != io.EOF {
		t.Fatalf("expected io.EOF, got %v", err)
	}

	feed := dec.Feed()
	if feed.Title != "Example Atom Log" || feed.Description != "Atom updates" {
		t.Fatalf("unexpected feed metadata: %+v", feed)
	}
}
