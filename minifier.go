package air

import (
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
	minifier *minify.M
}

// newMinifier returns a new instance of the `minifier` with the a.
func newMinifier(a *Air) *minifier {
	m := &minifier{
		a:        a,
		minifier: minify.New(),
	}

	m.minifier.Add("text/html", html.DefaultMinifier)
	m.minifier.Add("text/css", css.DefaultMinifier)
	m.minifier.Add("application/javascript", js.DefaultMinifier)
	m.minifier.Add("application/json", json.DefaultMinifier)
	m.minifier.Add("application/xml", xml.DefaultMinifier)
	m.minifier.Add("image/svg+xml", svg.DefaultMinifier)

	return m
}

// minify minifies the b based on the mimeType.
func (m *minifier) minify(mimeType string, b []byte) ([]byte, error) {
	b, err := m.minifier.Bytes(mimeType, b)
	if err != nil {
		if err == minify.ErrNotExist {
			return b, nil
		}

		return nil, err
	}

	return b, nil
}
