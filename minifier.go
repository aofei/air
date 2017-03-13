package air

import (
	"io"

	"github.com/tdewolff/minify"
	"github.com/tdewolff/minify/css"
	"github.com/tdewolff/minify/html"
	"github.com/tdewolff/minify/js"
	"github.com/tdewolff/minify/json"
	"github.com/tdewolff/minify/svg"
	"github.com/tdewolff/minify/xml"
)

type (
	// Minifier is used to provide a `Minify()` method for an `Air` instance for minifies a
	// content by a MIME type.
	Minifier interface {
		// Init initializes the `Minifier`. It will be called in the `Air#Serve()`.
		Init() error

		// Minify minifies the r into the w by the MIMEType.
		Minify(MIMEType string, w io.Writer, r io.Reader) error
	}

	// minifier implements the `Minifier` by using the "github.com/tdewolff/minify".
	minifier struct {
		*minify.M
	}
)

// newMinifier returns a pointer of a new instance of the `minifier`.
func newMinifier() *minifier {
	return &minifier{
		M: minify.New(),
	}
}

// Init implements the `Minifier#Init()` by using the "github.com/tdewolff/minify".
func (m *minifier) Init() error {
	m.Add(MIMETextHTML, &html.Minifier{
		KeepWhitespace: true,
	})

	m.Add(MIMETextCSS, &css.Minifier{
		Decimals: -1,
	})

	m.Add(MIMETextJavaScript, &js.Minifier{})

	m.Add(MIMEApplicationJSON, &json.Minifier{})

	m.Add(MIMETextXML, &xml.Minifier{
		KeepWhitespace: true,
	})

	m.Add(MIMEImageSVGXML, &svg.Minifier{
		Decimals: -1,
	})

	return nil
}
