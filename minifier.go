package air

import (
	"sync"

	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/css"
	"github.com/tdewolff/minify/v2/html"
	"github.com/tdewolff/minify/v2/js"
	"github.com/tdewolff/minify/v2/json"
	"github.com/tdewolff/minify/v2/svg"
	"github.com/tdewolff/minify/v2/xml"
)

// minifier is a minifier that minifies contents based on the MIME types.
type minifier struct {
	a        *Air
	loadOnce *sync.Once
	minifier *minify.M
}

// newMinifier returns a new instance of the `minifier` with the a.
func newMinifier(a *Air) *minifier {
	return &minifier{
		a:        a,
		loadOnce: &sync.Once{},
	}
}

// load loads the stuff of the m up.
func (m *minifier) load() {
	m.minifier = minify.New()
	m.minifier.Add("text/html", html.DefaultMinifier)
	m.minifier.Add("text/css", css.DefaultMinifier)
	m.minifier.Add("application/javascript", js.DefaultMinifier)
	m.minifier.Add("application/json", json.DefaultMinifier)
	m.minifier.Add("application/xml", xml.DefaultMinifier)
	m.minifier.Add("image/svg+xml", svg.DefaultMinifier)
}

// minify minifies the b based on the mimeType.
func (m *minifier) minify(mimeType string, b []byte) ([]byte, error) {
	m.loadOnce.Do(m.load)

	mb, err := m.minifier.Bytes(mimeType, b)
	if err == minify.ErrNotExist {
		mb = b
		err = nil
	}

	return mb, err
}
