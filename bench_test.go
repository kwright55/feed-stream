package feedstream

import (
	"fmt"
	"io"
	"strings"
	"testing"
)

// synthFeed generates an RSS 2.0 document with n items on the fly
// instead of building the whole thing as one string up front, so a
// benchmark against a "large feed" doesn't just move the memory cost
// from the decoder into the test itself.
type synthFeed struct {
	n      int
	i      int
	buf    *strings.Reader
	closed bool
}

func newSynthFeed(n int) *synthFeed {
	return &synthFeed{
		n: n,
		buf: strings.NewReader(`<?xml version="1.0"?>
<rss version="2.0" xmlns:content="http://purl.org/rss/1.0/modules/content/">
  <channel>
    <title>Bench Feed</title>
    <link>https://example.com</link>
    <description>Synthetic feed for benchmarking</description>
`),
	}
}

const synthItemTemplate = `    <item>
      <title>Item %d</title>
      <link>https://example.com/%d</link>
      <description>Synthetic description for item %d, with enough text to
      be representative of a real-world summary field rather than a bare
      placeholder.</description>
      <guid>https://example.com/%d</guid>
      <pubDate>Mon, 02 Jan 2006 15:04:05 GMT</pubDate>
      <content:encoded>&lt;p&gt;Full body text for item %d&lt;/p&gt;</content:encoded>
    </item>
`

func (f *synthFeed) Read(p []byte) (int, error) {
	for f.buf.Len() == 0 {
		if f.closed {
			return 0, io.EOF
		}
		if f.i >= f.n {
			f.buf = strings.NewReader("  </channel>\n</rss>")
			f.closed = true
			continue
		}
		f.i++
		f.buf = strings.NewReader(fmt.Sprintf(synthItemTemplate, f.i, f.i, f.i, f.i, f.i))
	}
	return f.buf.Read(p)
}

// BenchmarkDecoder decodes synthetic feeds of increasing item counts.
// Run with -benchmem and compare B/op and allocs/op across the
// sub-benchmarks: they should stay roughly flat per item rather than
// growing with the feed size, since Next only ever holds the current
// item in memory regardless of how many more follow it.
func BenchmarkDecoder(b *testing.B) {
	for _, n := range []int{1_000, 10_000, 100_000} {
		b.Run(fmt.Sprintf("items=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				dec := NewDecoder(newSynthFeed(n))
				for {
					_, err := dec.Next()
					if err == io.EOF {
						break
					}
					if err != nil {
						b.Fatal(err)
					}
				}
			}
		})
	}
}
