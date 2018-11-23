package air

import (
	"bytes"
	"mime"
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
	minifier *minify.M
	once     *sync.Once
}

// newMinifier returns a new instance of the `minifier` with the a.
func newMinifier(a *Air) *minifier {
	return &minifier{
		a:        a,
		minifier: minify.New(),
		once:     &sync.Once{},
	}
}

// bind binds the r into the v.
// minify minifies the b based on the mimeType.
func (m *minifier) minify(mimeType string, b []byte) ([]byte, error) {
	if !m.a.MinifierEnabled {
		return b, nil
	}

	m.once.Do(func() {
		m.minifier.Add("text/html", html.DefaultMinifier)
		m.minifier.Add("text/css", css.DefaultMinifier)
		m.minifier.Add("application/javascript", js.DefaultMinifier)
		m.minifier.Add("application/json", json.DefaultMinifier)
		m.minifier.Add("application/xml", xml.DefaultMinifier)
		m.minifier.Add("image/svg+xml", svg.DefaultMinifier)
	})

	mimeType, _, err := mime.ParseMediaType(mimeType)
	if err != nil {
		return nil, err
	}

	buf := bytes.Buffer{}
	if err := m.minifier.Minify(
		mimeType,
		&buf,
		bytes.NewReader(b),
	); err == minify.ErrNotExist {
		return b, nil
	} else if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
