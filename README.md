# feed-stream

A Go library for reading RSS 2.0 and Atom feeds one item at a time,
without decoding the whole document into memory first.

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

`Decoder` handles both RSS `<item>` and Atom `<entry>` elements and
normalizes them to the same `Item` struct, so callers don't need to know
which format a given feed uses.

`Item.PubDate` keeps the raw `pubDate`/`updated` text as-is; `Item.Published`
holds it parsed into a `time.Time` against the layouts real feeds actually
use, and is the zero `time.Time` if parsing failed, so check `IsZero()`
before relying on it.

## Status

Early. Core streaming decode for RSS 2.0 and Atom works and is covered
by tests; see the issues for what's still missing (RDF/RSS 1.0 support,
namespaced extensions like `content:encoded`).

## Install

```
go get github.com/kwright55/feed-stream
```

## License

MIT, see LICENSE.
