# feed-stream

A Go library for reading RSS 2.0, RSS 1.0 (RDF), and Atom feeds one item
at a time, without decoding the whole document into memory first.

## Why

Most feed-parsing libraries hand you `[]Item` at the end: they read the
entire response body, unmarshal it into a slice, and only then let you
look at anything. That's fine for a single small feed, but it stops
working well the moment you're pulling from a lot of feeds concurrently,
or a feed happens to be a few hundred megabytes because someone never
bothered to cap history. You end up holding the full XML document, plus
its parsed slice, for a result you're going to iterate over once anyway.

feed-stream decodes directly off the `io.Reader` using `encoding/xml`'s
token API. Each call to `Next()` reads just far enough to produce one
`Item` and returns it; nothing before or after it is buffered.

## Usage

```go
package main

import (
	"fmt"
	"io"
	"log"
	"net/http"

	feedstream "github.com/kwright55/feed-stream"
)

func main() {
	resp, err := http.Get("https://example.com/feed.xml")
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	dec := feedstream.NewDecoder(resp.Body)
	for {
		item, err := dec.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(item.Title, "-", item.Link)
	}

	fmt.Println("feed:", dec.Feed().Title)
}
```

`Decoder` handles RSS `<item>` (2.0 and 1.0/RDF) and Atom `<entry>`
elements and normalizes them to the same `Item` struct, so callers
don't need to know which format a given feed uses. For RSS 1.0, which
has no `<guid>` element and puts date/author in the Dublin Core
namespace, `Item.GUID` falls back to the item's `rdf:about` attribute
and `PubDate`/`Author` fall back to `dc:date`/`dc:creator`.

`Item.PubDate` keeps the raw `pubDate`/`updated` text as-is; `Item.Published`
holds it parsed into a `time.Time` against the layouts real feeds actually
use, and is the zero `time.Time` if parsing failed, so check `IsZero()`
before relying on it.

`NextContext(ctx)` is `Next` with a context, for callers fetching a lot of
feeds concurrently who need to give up on a slow one:

```go
item, err := dec.NextContext(ctx)
if errors.Is(err, context.DeadlineExceeded) {
	// this feed is taking too long; move on
}
```

It can only cut a call short between reads or right after one returns; if
the underlying reader blocks on a single Read indefinitely (a connection
that never sends and never times out), NextContext can't interrupt it.
Building the request with `http.NewRequestWithContext` using the same
context covers that case, since canceling it closes the response body and
unblocks the read.

## Status

Early. Core streaming decode for RSS 2.0, RSS 1.0 (RDF), and Atom works
and is covered by tests; see the issues for what's still missing
(namespaced extensions like `content:encoded`, fuzz testing).

## Install

```
go get github.com/kwright55/feed-stream
```

## License

MIT, see LICENSE.
