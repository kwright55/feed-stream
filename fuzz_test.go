package feedstream

import (
	"strings"
	"testing"
)

// FuzzDecoder feeds arbitrary byte sequences through the decoder. Feeds
// pulled off the network are never well-formed often enough to trust;
// the only contract that matters here is that garbage input produces an
// error (or a partial item) instead of a panic or a hang.
func FuzzDecoder(f *testing.F) {
	f.Add([]byte(sampleRSS))
	f.Add([]byte(sampleRDF))
	f.Add([]byte(sampleAtom))
	f.Add([]byte(``))
	f.Add([]byte(`<rss>`))
	f.Add([]byte(`<rss><channel><item></item></channel>`))
	f.Add([]byte(`<rss><channel><item><title>&</title></item></channel></rss>`))
	f.Add([]byte(`<feed><entry><link href="x" rel="alternate"></entry></feed>`))
	f.Add([]byte(`not xml at all`))
	f.Add([]byte(`<rss><channel><item><pubDate>` + strings.Repeat("9", 4096) + `</pubDate></item></channel></rss>`))

	f.Fuzz(func(t *testing.T, data []byte) {
		dec := NewDecoder(strings.NewReader(string(data)))

		// A malformed document could in principle describe an unbounded
		// run of items; cap the loop so a single corpus entry can't turn
		// into a hang, without treating a long-but-valid item list as a
		// failure.
		for i := 0; i < 10000; i++ {
			item, err := dec.Next()
			if err != nil {
				// Any error, io.EOF or otherwise, is an expected outcome
				// on malformed input; what matters is that we got here
				// without panicking.
				return
			}
			if item == nil {
				t.Fatal("Next returned nil item with nil error")
			}
		}

		dec.Feed()
	})
}
