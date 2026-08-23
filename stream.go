package feedstream

import (
	"encoding/xml"
	"io"
)

// Decoder reads a feed from an io.Reader and produces one Item per call
// to Next. It only ever holds the current item and a bit of channel
// metadata in memory, no matter how many items the feed contains, which
// matters for feeds fetched over HTTP that can run to tens of megabytes.
type Decoder struct {
	xd   *xml.Decoder
	feed Feed
}

// NewDecoder wraps r for streaming decode. r is read incrementally as
// Next is called; the caller is responsible for closing it (e.g. an
// http.Response.Body) once done.
func NewDecoder(r io.Reader) *Decoder {
	xd := xml.NewDecoder(r)
	// Real-world feeds routinely leak unescaped HTML entities like
	// &nbsp; into description text. The strict XML entity set doesn't
	// know these, so fall back to the HTML table instead of failing
	// the whole decode over a stray entity.
	xd.Entity = xml.HTMLEntity
	return &Decoder{xd: xd}
}

// Next returns the next item in the feed. It returns io.EOF once the
// document is exhausted, matching the convention of io.Reader.
func (d *Decoder) Next() (*Item, error) {
	for {
		tok, err := d.xd.Token()
		if err != nil {
			return nil, err
		}

		se, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}

		switch se.Name.Local {
		case "item", "entry":
			return d.decodeItem(se)
		case "title", "link", "description", "subtitle":
			// DecodeElement below consumes item/entry subtrees whole,
			// so any of these seen here at the top level belong to
			// the channel/feed itself, never to an item.
			if err := d.captureFeedField(se); err != nil {
				return nil, err
			}
		}
	}
}

// Feed returns the channel/feed metadata gathered from whatever has
// been decoded so far. Since that metadata normally precedes the item
// list in a well-formed feed, calling this after the first Next() call
// is usually enough, but it's safe to call at any point.
func (d *Decoder) Feed() Feed {
	return d.feed
}

func (d *Decoder) captureFeedField(se xml.StartElement) error {
	switch se.Name.Local {
	case "title":
		var s string
		if err := d.xd.DecodeElement(&s, &se); err != nil {
			return err
		}
		d.feed.Title = s
	case "description", "subtitle":
		var s string
		if err := d.xd.DecodeElement(&s, &se); err != nil {
			return err
		}
		d.feed.Description = s
	case "link":
		for _, a := range se.Attr {
			if a.Name.Local == "href" {
				d.feed.Link = a.Value
				return d.xd.Skip()
			}
		}
		var s string
		if err := d.xd.DecodeElement(&s, &se); err != nil {
			return err
		}
		d.feed.Link = s
	}
	return nil
}

// rssItem mirrors the RSS 2.0 <item> element. It doubles as the RSS 1.0
// (RDF) <item> element, which uses the same title/link/description
// elements but carries its date and author in the Dublin Core namespace
// instead of pubDate/author, and has no guid element at all (rdf:about
// on the item itself is the closest equivalent). encoding/xml matches
// tags by local name when no namespace is given, so DCDate and DCCreator
// pick up dc:date and dc:creator without needing a separate struct.
type rssItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	GUID        string `xml:"guid"`
	PubDate     string `xml:"pubDate"`
	Author      string `xml:"author"`
	DCDate      string `xml:"date"`
	DCCreator   string `xml:"creator"`
}

// atomEntry mirrors the Atom <entry> element. Atom allows multiple
// <link> elements distinguished by rel; the alternate (or unlabeled)
// one is what most callers mean by "the link".
type atomEntry struct {
	Title   string `xml:"title"`
	ID      string `xml:"id"`
	Updated string `xml:"updated"`
	Summary string `xml:"summary"`
	Author  struct {
		Name string `xml:"name"`
	} `xml:"author"`
	Links []struct {
		Href string `xml:"href,attr"`
		Rel  string `xml:"rel,attr"`
	} `xml:"link"`
}

func (d *Decoder) decodeItem(se xml.StartElement) (*Item, error) {
	if se.Name.Local == "entry" {
		var e atomEntry
		if err := d.xd.DecodeElement(&e, &se); err != nil {
			return nil, err
		}
		var link string
		for _, l := range e.Links {
			if l.Rel == "" || l.Rel == "alternate" {
				link = l.Href
				break
			}
		}
		return &Item{
			Title:       e.Title,
			Link:        link,
			Description: e.Summary,
			GUID:        e.ID,
			PubDate:     e.Updated,
			Published:   parseDate(e.Updated),
			Author:      e.Author.Name,
		}, nil
	}

	var it rssItem
	if err := d.xd.DecodeElement(&it, &se); err != nil {
		return nil, err
	}

	guid := it.GUID
	if guid == "" {
		// RSS 1.0 items have no guid element; rdf:about on the item
		// itself is the nearest thing to a stable identifier.
		for _, a := range se.Attr {
			if a.Name.Local == "about" {
				guid = a.Value
				break
			}
		}
	}
	pubDate := it.PubDate
	if pubDate == "" {
		pubDate = it.DCDate
	}
	author := it.Author
	if author == "" {
		author = it.DCCreator
	}

	return &Item{
		Title:       it.Title,
		Link:        it.Link,
		Description: it.Description,
		GUID:        guid,
		PubDate:     pubDate,
		Published:   parseDate(pubDate),
		Author:      author,
	}, nil
}
