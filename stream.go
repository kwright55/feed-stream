package feedstream

import (
	"context"
	"encoding/xml"
	"io"
)

// Decoder reads a feed from an io.Reader and produces one Item per call
// to Next. It only ever holds the current item and a bit of channel
// metadata in memory, no matter how many items the feed contains, which
// matters for feeds fetched over HTTP that can run to tens of megabytes.
type Decoder struct {
	xd   *xml.Decoder
	cr   *ctxReader
	feed Feed
}

// NewDecoder wraps r for streaming decode. r is read incrementally as
// Next is called; the caller is responsible for closing it (e.g. an
// http.Response.Body) once done.
func NewDecoder(r io.Reader) *Decoder {
	cr := &ctxReader{r: r, ctx: context.Background()}
	xd := xml.NewDecoder(cr)
	// Real-world feeds routinely leak unescaped HTML entities like
	// &nbsp; into description text. The strict XML entity set doesn't
	// know these, so fall back to the HTML table instead of failing
	// the whole decode over a stray entity.
	xd.Entity = xml.HTMLEntity
	return &Decoder{xd: xd, cr: cr}
}

// ctxReader lets a context passed to NextContext interrupt an
// in-progress decode. It checks the context both before and after the
// underlying Read, since Read itself may block for a while (a slow
// connection, a stalled proxy) without knowing anything about the
// context at all.
type ctxReader struct {
	r   io.Reader
	ctx context.Context
}

func (cr *ctxReader) Read(p []byte) (int, error) {
	if err := cr.ctx.Err(); err != nil {
		return 0, err
	}
	n, err := cr.r.Read(p)
	if err != nil {
		return n, err
	}
	if err := cr.ctx.Err(); err != nil {
		return n, err
	}
	return n, nil
}

// Next returns the next item in the feed. It returns io.EOF once the
// document is exhausted, matching the convention of io.Reader.
func (d *Decoder) Next() (*Item, error) {
	return d.NextContext(context.Background())
}

// NextContext is Next with a context that can cancel a decode still in
// progress. If ctx is already done, or becomes done while the
// underlying reader is blocked on a Read, Next returns ctx.Err() (or an
// error wrapping it, once the xml.Decoder has attached its own
// position info) instead of waiting for more data.
//
// This only works if the wrapped io.Reader eventually returns from a
// blocked Read once the peer goes away, since ctxReader can't interrupt
// a call already in flight; it can only check the context before
// starting one and right after one returns. Reading an
// *http.Response.Body from a request built with http.NewRequestWithContext
// satisfies this, since canceling that context closes the body itself.
func (d *Decoder) NextContext(ctx context.Context) (*Item, error) {
	d.cr.ctx = ctx
	defer func() { d.cr.ctx = context.Background() }()

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
	// GUID is a nested struct rather than a plain string because the
	// isPermaLink attribute lives alongside the text, not inside it.
	// encoding/xml matches both the element and its attribute by local
	// name, so this doesn't care what namespace prefix (if any) a feed
	// puts on isPermaLink.
	GUID struct {
		Value       string `xml:",chardata"`
		IsPermaLink string `xml:"isPermaLink,attr"`
	} `xml:"guid"`
	PubDate string `xml:"pubDate"`
	Author      string `xml:"author"`
	DCDate      string `xml:"date"`
	DCCreator   string `xml:"creator"`
	// ContentEncoded is content:encoded from the RSS content module,
	// picked up by local name the same way DCDate/DCCreator are.
	ContentEncoded string `xml:"encoded"`
	// Media is a media:content element (Media RSS), which carries its
	// data as attributes on a self-closing element rather than as
	// text, so it needs its own struct instead of a plain string.
	Media struct {
		URL  string `xml:"url,attr"`
		Type string `xml:"type,attr"`
	} `xml:"content"`
}

// atomEntry mirrors the Atom <entry> element. Atom allows multiple
// <link> elements distinguished by rel; the alternate (or unlabeled)
// one is what most callers mean by "the link".
type atomEntry struct {
	Title   string `xml:"title"`
	ID      string `xml:"id"`
	Updated string `xml:"updated"`
	Summary string `xml:"summary"`
	Content string `xml:"content"`
	Author  struct {
		Name string `xml:"name"`
	} `xml:"author"`
	// Media is a Media RSS media:content element, which some podcast
	// feeds attach to Atom entries alongside the native <content>
	// element. The namespace is given explicitly here (unlike the
	// RSS side) because Atom's own <content> shares the local name
	// "content" and would otherwise be ambiguous with it.
	Media struct {
		URL  string `xml:"url,attr"`
		Type string `xml:"type,attr"`
	} `xml:"http://search.yahoo.com/mrss/ content"`
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
			Content:     e.Content,
			MediaURL:    e.Media.URL,
			MediaType:   e.Media.Type,
		}, nil
	}

	var it rssItem
	if err := d.xd.DecodeElement(&it, &se); err != nil {
		return nil, err
	}

	guid := it.GUID.Value
	// isPermaLink defaults to true per the RSS 2.0 spec when the guid
	// element is present but the attribute itself is omitted.
	guidIsPermaLink := guid != "" && it.GUID.IsPermaLink != "false"
	if guid == "" {
		// RSS 1.0 items have no guid element; rdf:about on the item
		// itself is the nearest thing to a stable identifier, but it
		// carries no isPermaLink concept of its own.
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
		Title:           it.Title,
		Link:            it.Link,
		Description:     it.Description,
		GUID:            guid,
		GUIDIsPermaLink: guidIsPermaLink,
		PubDate:         pubDate,
		Published:       parseDate(pubDate),
		Author:          author,
		Content:         it.ContentEncoded,
		MediaURL:        it.Media.URL,
		MediaType:       it.Media.Type,
	}, nil
}
